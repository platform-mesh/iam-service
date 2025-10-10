package middleware

import (
	"context"
	"net/http"
	"strings"

	pmcontext "github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/logger"

	"github.com/platform-mesh/golang-commons/policy_services"
)

type middlewareProvider struct {
	retriever policy_services.TenantRetriever
}

const OrganizationAccountName = pmcontext.ContextKey("OrganizationAccountName")
const ParentClusterId = pmcontext.ContextKey("ParentClusterId")

func (tp *middlewareProvider) storeTenantIdCtxValue() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := logger.LoadLoggerFromContext(ctx)

			tenantId, err := tp.retriever.RetrieveTenant(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Error while retrieving the tenant from the iam service")
				http.Error(w, "invalid tenant", http.StatusForbidden)
				return
			}

			triple := strings.Split(tenantId, "/")
			if res := triple; len(res) != 3 {
				http.Error(w, "invalid tenant", http.StatusInternalServerError)
			}

			parentClusterId, accountName, clusterId := triple[0], triple[1], triple[2]
			ctx = pmcontext.AddTenantToContext(ctx, clusterId)
			ctx = context.WithValue(ctx, OrganizationAccountName, accountName)
			ctx = context.WithValue(ctx, ParentClusterId, parentClusterId)

			log.Trace().Str("tenantId", tenantId).Msg("TenantId was added to the context")

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func StoreTenantIdCtxValue(r policy_services.TenantRetriever) func(http.Handler) http.Handler {
	return createMiddleware(r).storeTenantIdCtxValue()
}

func createMiddleware(r policy_services.TenantRetriever) *middlewareProvider {
	return &middlewareProvider{retriever: r}
}
