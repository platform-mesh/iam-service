package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/joho/godotenv/autoload"
	"github.com/kcp-dev/multicluster-provider/apiexport"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	pmmws "github.com/platform-mesh/golang-commons/middleware"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/client-go/rest"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	"github.com/platform-mesh/golang-commons/logger"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/platform-mesh/iam-service/pkg/config"
	kcpmiddleware "github.com/platform-mesh/iam-service/pkg/middleware/kcp"
	keycloakmw "github.com/platform-mesh/iam-service/pkg/middleware/keycloak"
	"github.com/platform-mesh/iam-service/pkg/resolver"
	"github.com/platform-mesh/iam-service/pkg/service/keycloak"

	pmcontext "github.com/platform-mesh/golang-commons/context"

	ctrl "sigs.k8s.io/controller-runtime"

	iamRouter "github.com/platform-mesh/iam-service/pkg/router"
)

var serverCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start serving",
	Long:  `Start the IAM Service as a Webservice`,
	Run: func(cmd *cobra.Command, args []string) {
		serveFunc()
	},
}

func serveFunc() {
	ctx, _, shutdown := pmcontext.StartContext(log, serviceCfg, defaultCfg.ShutdownTimeout)
	defer shutdown()

	mgr := setupManagerAsync(ctx, log)
	router := setupRouter(ctx, mgr, setupFGAClient())
	start(serviceCfg, router, ctx, log, defaultCfg.IsLocal)
}

func setupRouter(ctx context.Context, mgr mcmanager.Manager, fgaClient openfgav1.OpenFGAServiceClient) *chi.Mux {
	kcpmw := kcpmiddleware.New(mgr, serviceCfg, log, &keycloakmw.KeycloakIDMRetriever{})
	mws := pmmws.CreateMiddleware(log, true)
	mws = append(mws, kcpmw.SetKCPUserContext())
	//tr := tenant.NewTenantReader(log, database)
	//ctr := policy_services.NewCustomTenantRetriever(tr)
	//mws = append(mws, middleware.StoreTenantIdCtxValue(ctr))

	// Prepare Directives
	//directives := iamRouter.WithAuthorizedDirective(ad.Authorized)

	// create Resolver Service
	idmClient, err := keycloak.New(ctx, serviceCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create keycloak client")
	}
	svc := resolver.NewResolverService(fgaClient, idmClient)
	//.New(database, compatService, log)
	//ad := directives.NewAuthorizedDirective(fgaStoreHelper, openfgaClient)

	res := resolver.New(svc, log.ComponentLogger("resolver"))
	//router := iamRouter.CreateRouter(serviceCfg, res, log, mws, directives)
	router := iamRouter.CreateRouter(defaultCfg, serviceCfg, res, log, mws)
	return router
}

func setupFGAClient() openfgav1.OpenFGAServiceClient {
	fgaConn, err := grpc.NewClient(serviceCfg.OpenFGA.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start grpc server")
	}

	fgaClient := openfgav1.NewOpenFGAServiceClient(fgaConn)
	return fgaClient
}

func setupManagerAsync(ctx context.Context, log *logger.Logger) mcmanager.Manager {
	restCfg := ctrl.GetConfigOrDie()
	restCfg.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(rt)
	})
	providerCfg := rest.CopyConfig(restCfg)
	provider, err := apiexport.New(providerCfg, apiexport.Options{Scheme: scheme})
	if err != nil {
		log.Fatal().Err(err).Msg("unable to construct APIExport provider")
	}
	var tlsOpts []func(*tls.Config)
	disableHTTP2 := func(c *tls.Config) {
		log.Info().Msg("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}
	if !defaultCfg.EnableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}
	mgr, err := mcmanager.New(providerCfg, provider, mcmanager.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   defaultCfg.Metrics.BindAddress,
			SecureServing: defaultCfg.Metrics.Secure,
			TLSOpts:       tlsOpts,
		},

		BaseContext:            func() context.Context { return ctx },
		HealthProbeBindAddress: defaultCfg.HealthProbeBindAddress,
		LeaderElection:         false,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("unable to start manager")
	}
	log.Info().Msg("starting APIExport provider")
	go func() {
		if err := provider.Run(ctx, mgr); err != nil {
			log.Fatal().Err(err).Msg("problem running APIExport provider")
		}
	}()

	log.Info().Msg("starting manager")
	go func() {
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			log.Fatal().Err(err).Msg("problem running manager")
		}
	}()
	return mgr
}

func start(serviceCfg *config.ServiceConfig, router *chi.Mux, ctx context.Context, log *logger.Logger, isLocal bool) {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", serviceCfg.Port),
		Handler:      router,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
		BaseContext:  func(listener net.Listener) context.Context { return ctx },
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("failed to start http server")
		}
	}()

	log.Info().Msgf("service started on port: %d", serviceCfg.Port)
	if isLocal {
		log.Info().Msgf("connect to http://localhost:%d/ for graphQL playground", serviceCfg.Port)
	}
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		log.Panic().Err(err).Msg("Graceful shutdown failed")
	}
}
