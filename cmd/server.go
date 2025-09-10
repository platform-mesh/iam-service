package cmd

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "github.com/joho/godotenv/autoload"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlparser/v2/ast"

	internalfga "github.com/platform-mesh/iam-service/internal/pkg/fga"
	"github.com/platform-mesh/iam-service/pkg/db"
	"github.com/platform-mesh/iam-service/pkg/fga"
	myresolver "github.com/platform-mesh/iam-service/pkg/resolver"

	"github.com/platform-mesh/golang-commons/logger"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	pmcontext "github.com/platform-mesh/golang-commons/context"

	"github.com/platform-mesh/iam-service/internal/pkg/directives"
	gormlogger "github.com/platform-mesh/iam-service/internal/pkg/logger"
	iamRouter "github.com/platform-mesh/iam-service/internal/pkg/router"
	"github.com/platform-mesh/iam-service/internal/pkg/tenant"
	"github.com/platform-mesh/iam-service/pkg/graph"
	iamservice "github.com/platform-mesh/iam-service/pkg/service"
)

func getServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start serving",
		Long:  `Start the IAM Service as a Webservice`,
		Run: func(cmd *cobra.Command, args []string) {
			serveFunc()
		},
	}
}

func InitServeCmd(rootCmd *cobra.Command) {
	rootCmd.AddCommand(getServeCmd())
}

func getGormConn(log *logger.Logger, cfg db.ConfigDatabase) (*gorm.DB, error) {
	var dbDialect gorm.Dialector
	if cfg.InMemory { // local sqlite db
		dsn := "file::memory:?cache=shared"
		log.Debug().Msg(dsn)
		dbDialect = sqlite.Open(dsn)
	} else {
		dbDialect = postgres.Open(cfg.DSN)
	}

	return gorm.Open(dbDialect, &gorm.Config{
		Logger: gormlogger.NewFromLogger(log.ComponentLogger("gorm")),
	})
}

func serveFunc() { // nolint: funlen,cyclop,gocognit
	appConfig, log := initApp()
	ctx, _, shutdown := pmcontext.StartContext(log, nil, appConfig.ShutdownTimeout)
	defer shutdown()

	database, err := initDB(appConfig, log)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to init database")
	}
	defer func(database *db.Database) {
		err := database.Close()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to close the db connection")
		}
	}(database)

	tr := tenant.NewTenantReader(log, database)

	resolver.SetDefaultScheme("passthrough")
	log.Info().Str("addr", appConfig.Openfga.ListenAddr).Msg("starting grpc server")
	lis, err := net.Listen("tcp", appConfig.Openfga.ListenAddr)
	if err != nil {
		log.Panic().Err(err).Msg("failed to listen on ListenAddr")
	}
	log.Info().Str("addr", appConfig.Openfga.ListenAddr).Msg("successfully started grpc listener")

	fgaStoreHelper := internalfga.NewOpenMFPStoreHelper()

	fgaServer, compatService, err := fga.NewFGAServer(appConfig.Openfga.GRPCAddr, database, nil, tr, appConfig.IsLocal)
	if err != nil {
		log.Panic().Err(err).Msg("failed to init service")
	}
	compatService = compatService.WithFGAStoreHelper(fgaStoreHelper)

	go func() {
		err := fgaServer.Serve(lis)
		if !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatal().Err(err).Msg("failed to start grpc server")
		}
		log.Info().Msg("serving grpc server without errors")
		if err != nil {
			log.Info().Msg("grpc server shut down..")
			fgaServer.Stop()
		}
	}()

	conn, err := grpc.NewClient(appConfig.Openfga.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)

	if err != nil {
		log.Fatal().Err(err).Msg("failed to start grpc server")
	}

	openfgaClient := openfgav1.NewOpenFGAServiceClient(conn)

	// create openmfp Resolver
	svc := iamservice.New(database, compatService)
	ad := directives.NewAuthorizedDirective(fgaStoreHelper, openfgaClient)
	router := iamRouter.CreateRouter(appConfig, svc, log, iamRouter.WithAuthorizedDirective(ad.Authorized))
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

	server := &http.Server{
		Addr:         ":" + appConfig.Port,
		Handler:      router,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
	}
	log.Info().Msg("Resolver created")
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &myresolver.Resolver{}}))
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	go func() {
		var err error
		if appConfig.LocalSsl {
			err = server.ListenAndServeTLS("../ssl/server.crt", "../ssl/server.key")
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("failed to start http server")
		}
	}()

	log.Info().Msgf("service started on port: %s", appConfig.Port)
	if appConfig.IsLocal {
		log.Info().Msgf("connect to http://localhost:%s/ for graphQL playground", appConfig.Port)
	}
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fgaServer.GracefulStop()
	err = server.Shutdown(shutdownCtx)
	if err != nil {
		log.Panic().Err(err).Msg("Graceful shutdown failed")
	}
}
