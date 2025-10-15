package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/joho/godotenv/autoload"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	pmmws "github.com/platform-mesh/golang-commons/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/platform-mesh/iam-service/internal/config"
	"github.com/platform-mesh/iam-service/pkg/resolver"

	"github.com/platform-mesh/golang-commons/logger"

	pmcontext "github.com/platform-mesh/golang-commons/context"

	iamRouter "github.com/platform-mesh/iam-service/internal/pkg/router"
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

	fgaConn, err := grpc.NewClient(serviceCfg.OpenFGA.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start grpc server")
	}

	fgaClient := openfgav1.NewOpenFGAServiceClient(fgaConn)

	// Prepare Middlewares
	mws := pmmws.CreateMiddleware(log, true)
	//tr := tenant.NewTenantReader(log, database)
	//ctr := policy_services.NewCustomTenantRetriever(tr)
	//mws = append(mws, middleware.StoreTenantIdCtxValue(ctr))

	// Prepare Directives
	//directives := iamRouter.WithAuthorizedDirective(ad.Authorized)

	// create Resolver Service
	idmClient := keycloak.New()
	svc := resolver.NewResolverService(fgaClient)
	//.New(database, compatService, log)
	//ad := directives.NewAuthorizedDirective(fgaStoreHelper, openfgaClient)

	res := resolver.New(svc, log.ComponentLogger("resolver"))
	//router := iamRouter.CreateRouter(serviceCfg, res, log, mws, directives)
	router := iamRouter.CreateRouter(defaultCfg, serviceCfg, res, log, mws)
	setupObsHandler(router, log)

	log.Info().Msg("Resolver created")
	start(serviceCfg, router, ctx, log, defaultCfg.IsLocal)
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

func setupObsHandler(router *chi.Mux, log *logger.Logger) {
	metricsHandler := promhttp.Handler()
	router.Handle("/metrics", metricsHandler)
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Error().Err(err).Msg("Failed to write response for health check")
		}
	})
	router.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Error().Err(err).Msg("Failed to write response for readiness check")
		}
	})
}
