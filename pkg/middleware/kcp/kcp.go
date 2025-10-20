package kcp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	pmcontext "github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/errors"
	"github.com/platform-mesh/golang-commons/logger"
	"k8s.io/client-go/rest"
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

			tokenInfo, err := pmcontext.GetWebTokenFromContext(ctx)
			if err != nil {
				msg := "Error while retrieving tokenInfo"
				log.Error().Err(err).Msg(msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			idmTenant, err := m.tenantRetriever.GetIDMTenant(tokenInfo.Issuer)
			if err != nil {
				msg := "Error while retrieving realm info"
				log.Error().Err(err).Msg(msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			authHeader, err := pmcontext.GetAuthHeaderFromContext(ctx)
			if err != nil {
				msg := "Error while retrieving tokenInfo"
				log.Error().Err(err).Msg(msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			// retrieve subdomain from url
			subdomain := strings.Split(r.Host, ".")[0]
			log.Debug().Str("subdmain", subdomain).Msg("processing request")

			// Create API Request against root:orgs:subdomain
			allowed, err := checkToken(ctx, authHeader, subdomain, m.mgr.GetLocalManager().GetConfig())
			if err != nil {
				msg := "Error while checking auth"
				log.Error().Err(err).Msg(msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}
			if !allowed {
				msg := "access denied"
				log.Error().Err(err).Msg(msg)
				http.Error(w, msg, http.StatusForbidden)
				return
			}

			kctx := KCPContext{
				OrganizationName: subdomain,
				IDMTenant:        idmTenant,
			}
			ctx = context.WithValue(ctx, UserContextKey, kctx)
			log.Trace().
				Str("organization", kctx.OrganizationName).
				Msg("Added information to context was added to the context")

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func checkToken(ctx context.Context, authHeader string, subdomain string, mgrcfg *rest.Config) (bool, error) {
	cfg := rest.CopyConfig(mgrcfg)
	log := logger.LoadLoggerFromContext(ctx)
	clusterUrl, err := url.Parse(cfg.Host)
	if err != nil {
		log.Error().Err(errors.WithStack(err)).Msg("Error parsing KCP host URL")
	}

	if clusterUrl == nil {
		return false, errors.New("invalid KCP host URL")
	}

	clusterPath := fmt.Sprintf("root:orgs:%s", subdomain)
	requestURL := fmt.Sprintf("%s://%s/clusters/%s/version", clusterUrl.Scheme, clusterUrl.Host, clusterPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, http.NoBody)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", authHeader)

	wsClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return false, err
	}

	res, err := wsClient.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close() //nolint:errcheck

	switch res.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusForbidden:
		return true, nil
	}
	return false, nil
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
	IDMTenant        string
	OrganizationName string
}
