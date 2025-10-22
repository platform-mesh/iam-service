package router

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	pmconfig "github.com/platform-mesh/golang-commons/config"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/platform-mesh/iam-service/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/graph"
)

func CreateRouter(
	commonCfg *pmconfig.CommonServiceConfig,
	serviceConfig *config.ServiceConfig,
	res graph.ResolverRoot,
	log *logger.Logger,
	mws []func(http.Handler) http.Handler,
	ad graph.DirectiveRoot,
) *chi.Mux {
	router := chi.NewRouter()

	gql := graph.Config{
		Resolvers: res,
	}

	gql.Directives = ad
	gqHandler := handler.New(graph.NewExecutableSchema(gql))

	gqHandler.AddTransport(transport.Options{})
	gqHandler.AddTransport(transport.GET{})
	gqHandler.AddTransport(transport.POST{})

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
