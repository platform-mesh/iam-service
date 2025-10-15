package router

import (
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-http-utils/headers"
	pmconfig "github.com/platform-mesh/golang-commons/config"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/rs/cors"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/platform-mesh/iam-service/internal/config"
	"github.com/platform-mesh/iam-service/pkg/graph"
)

type Options func(*graph.Config)

//func WithAuthorizedDirective(
//	dir func(ctx context.Context, obj any, next gqlgen.Resolver, relation string, entityType *string, entityTypeParamName *string, entityParamName string) (res any, err error),
//) Options {
//	return func(cfg *graph.Config) {
//		cfg.Directives.Authorized = dir
//	}
//}

func CreateRouter(
	commonCfg *pmconfig.CommonServiceConfig,
	serviceConfig *config.ServiceConfig,
	res graph.ResolverRoot,
	log *logger.Logger,
	mws []func(http.Handler) http.Handler,
	opts ...Options,
) *chi.Mux {
	router := chi.NewRouter()

	// On local the iam responds to CORS requests, on the cluster this is handled by istio
	if commonCfg.IsLocal {
		router.Use(cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
			AllowedHeaders:   []string{headers.Accept, headers.Authorization, headers.ContentType, headers.XCSRFToken},
			Debug:            false,
		}).Handler)
	}

	gql := graph.Config{
		Resolvers: res,
	}

	for _, opt := range opts {
		opt(&gql)
	}

	//if gql.Directives.Authorized == nil {
	//	conn, err := grpc.NewClient(serviceConfig.OpenFGA.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	//	if err != nil {
	//		log.Fatal().Err(err).Msg("unable to establish openfga connection")
	//	}
	//
	//	openfgaClient := openfgav1.NewOpenFGAServiceClient(conn)
	//	gql.Directives.Authorized = directive.Authorized(openfgaClient, log)
	//}

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

	if commonCfg.IsLocal {
		router.Handle("/", playground.Handler("GraphQL playground", "/graphql"))
	}

	router.With(mws...).Handle("/graphql", gqHandler)
	return router
}
