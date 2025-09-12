package router

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-http-utils/headers"
	"github.com/platform-mesh/golang-commons/directive"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/ast"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	gqlgen "github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"

	"github.com/platform-mesh/iam-service/internal/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/resolver"
	"github.com/platform-mesh/iam-service/pkg/service"
)

type Options func(*graph.Config)

func WithAuthorizedDirective(
	dir func(ctx context.Context, obj any, next gqlgen.Resolver, relation string, entityType *string, entityTypeParamName *string, entityParamName string) (res any, err error),
) Options {
	return func(cfg *graph.Config) {
		cfg.Directives.Authorized = dir
	}
}

func CreateRouter(
	appConfig config.Config,
	svc *service.Service,
	log *logger.Logger,
	opts ...Options,
) *chi.Mux {
	router := chi.NewRouter()

	// On local the iam responds to CORS requests, on the cluster this is handled by istio
	if appConfig.IsLocal {
		router.Use(cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
			AllowedHeaders:   []string{headers.Accept, headers.Authorization, headers.ContentType, headers.XCSRFToken},
			Debug:            false,
		}).Handler)
	}

	logResolver := logger.StdLogger.ComponentLogger("resolver")
	gql := graph.Config{
		Resolvers: resolver.New(svc, logResolver),
		Directives: graph.DirectiveRoot{
			PeersOnly: func(ctx context.Context, obj interface{}, next gqlgen.Resolver) (res interface{}, err error) {
				return next(ctx)
			},
		},
	}

	for _, opt := range opts {
		opt(&gql)
	}

	if gql.Directives.Authorized == nil {
		conn, err := grpc.NewClient(appConfig.Openfga.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatal().Err(err).Msg("unable to establish openfga connection")
		}

		openfgaClient := openfgav1.NewOpenFGAServiceClient(conn)
		gql.Directives.Authorized = directive.Authorized(openfgaClient, log)
	}

	gqHandler := handler.New(graph.NewExecutableSchema(gql))
	gqHandler.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	gqHandler.AddTransport(transport.Options{})
	gqHandler.AddTransport(transport.GET{})
	gqHandler.AddTransport(transport.POST{})
	gqHandler.AddTransport(transport.MultipartForm{})
	gqHandler.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	gqHandler.Use(extension.Introspection{})
	gqHandler.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	if appConfig.IsLocal {
		router.Handle("/", playground.Handler("GraphQL playground", "/query"))
	}
	router.Handle("/query", gqHandler)
	return router
}
