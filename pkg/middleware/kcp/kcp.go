package kcp

import (
	"context"
	"net/http"

	kcptenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	pmcontext "github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/errors"
	"github.com/platform-mesh/golang-commons/logger"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	"github.com/platform-mesh/iam-service/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/middleware/idm"
)

type ContextKey string

const (
	UserContextKey ContextKey = "KCPContext"
)

type Middleware struct {
	mgr                      mcmanager.Manager
	cfg                      *config.ServiceConfig
	log                      *logger.Logger
	tenantRetriever          idm.IDMTenantRetriever
	excludedIDMTenants       []string
	orgsWorkspaceClusterName string
}

func New(mgr mcmanager.Manager, cfg *config.ServiceConfig, log *logger.Logger, tenantRetriever idm.IDMTenantRetriever, orgsWorkspaceClusterName string) *Middleware {
	excludedIDMTenants := cfg.IDM.ExcludedTenants
	return &Middleware{
		mgr:                      mgr,
		cfg:                      cfg,
		log:                      log,
		tenantRetriever:          tenantRetriever,
		excludedIDMTenants:       excludedIDMTenants,
		orgsWorkspaceClusterName: orgsWorkspaceClusterName,
	}
}

func (m *Middleware) SetKCPUserContext() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := logger.LoadLoggerFromContext(ctx)
			kctx, err := m.getKcpInfosForContext(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Error while retrieving data from kcp")
				http.Error(w, "Error while retrieving data from kcp", http.StatusInternalServerError)
				return
			}

			ctx = context.WithValue(ctx, UserContextKey, kctx)
			log.Trace().
				Str("IDMTenant", kctx.IDMTenant).
				Str("ClusterId", kctx.ClusterId).
				Msg("Added information to context was added to the context")

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
func GetKcpUserContext(ctx context.Context) (KCPContext, error) {
	val := ctx.Value(UserContextKey)
	if val == nil {
		return KCPContext{}, errors.New("kcp user context not found in context")
	}

	kctx, ok := val.(KCPContext)
	if !ok {
		return KCPContext{}, errors.New("invalid kcp user context type")
	}

	return kctx, nil
}

type KCPContext struct {
	IDMTenant string
	ClusterId string
}

func (s *Middleware) getKcpInfosForContext(ctx context.Context) (KCPContext, error) {
	kctx := KCPContext{}
	tokenInfo, err := pmcontext.GetWebTokenFromContext(ctx)
	if err != nil {
		return kctx, err
	}

	cluster, err := s.mgr.GetCluster(ctx, s.orgsWorkspaceClusterName)
	if err != nil {
		return kctx, errors.Wrap(err, "failed to get orgs cluster from multicluster manager")
	}

	idmTenant, err := s.tenantRetriever.GetIDMTenant(tokenInfo.Issuer)
	if err != nil {
		return kctx, errors.Wrap(err, "failed to get idm tenant from token issuer")
	}

	for _, excluded := range s.excludedIDMTenants {
		if idmTenant == excluded {
			return kctx, errors.New("invalid tenant")
		}
	}

	acc := &accountsv1alpha1.Account{}
	err = cluster.GetClient().Get(ctx, client.ObjectKey{Name: idmTenant}, acc)
	if err != nil {
		return kctx, errors.Wrap(err, "failed to get account from kcp")
	}

	if acc.Spec.Type != "org" {
		return kctx, errors.New("invalid account type, expected 'org'")
	}

	ws := &kcptenancyv1alpha1.Workspace{}
	err = cluster.GetClient().Get(ctx, client.ObjectKey{Name: acc.Name}, ws)
	if err != nil {
		return kctx, errors.Wrap(err, "failed to get workspace from kcp")
	}

	kctx.IDMTenant = idmTenant
	kctx.ClusterId = ws.Spec.Cluster
	return kctx, nil
}
