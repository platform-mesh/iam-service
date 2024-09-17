package cmd

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/99designs/gqlgen/graphql/playground"
	_ "github.com/joho/godotenv/autoload"
	"github.com/openmfp/iam-service/pkg/db"
	"github.com/openmfp/iam-service/pkg/fga"
	myresolver "github.com/openmfp/iam-service/pkg/resolver"
	"github.com/spf13/cobra"

	"github.com/openmfp/golang-commons/logger"

	"github.com/99designs/gqlgen/graphql/handler"
	openmfpcontext "github.com/openmfp/golang-commons/context"
	gormlogger "github.com/openmfp/iam-service/internal/pkg/logger"
	iamRouter "github.com/openmfp/iam-service/internal/pkg/router"
	"github.com/openmfp/iam-service/internal/pkg/tenant"
	"github.com/openmfp/iam-service/pkg/graph"
	openmfpservice "github.com/openmfp/iam-service/pkg/service"
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
	ctx, _, shutdown := openmfpcontext.StartContext(log, nil, appConfig.ShutdownTimeout)
	defer shutdown()

	// open db connection
	dbConn, err := getGormConn(log, appConfig.Database)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to open db connection")
	}

	database, err := db.New(appConfig.Database, dbConn, log, appConfig.IsLocal, appConfig.IsLocal)
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

	fgaServer, compatService, err := fga.NewFGAServer(appConfig.Openfga.GRPCAddr, database, nil, tr, appConfig.IsLocal)
	if err != nil {
		log.Panic().Err(err).Msg("failed to init service")
	}

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

	// create openmfp Resolver
	svc := openmfpservice.New(database, compatService)
	router := iamRouter.CreateRouter(appConfig, svc, log)

	server := &http.Server{
		Addr:         ":" + appConfig.Port,
		Handler:      router,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
	}
	log.Info().Msg("Resolver created")
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &myresolver.Resolver{}}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	go func() {
		if appConfig.LocalSsl {
			server.ListenAndServeTLS("../ssl/server.crt", "../ssl/server.key") // nolint: errcheck
		} else {
			server.ListenAndServe() // nolint: errcheck
		}
	}()

	log.Info().Msg("Service started")

	if appConfig.IsLocal {
		log.Info().Msgf("connect to http://localhost:%s/ for GraphQL playground", appConfig.Port)
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
