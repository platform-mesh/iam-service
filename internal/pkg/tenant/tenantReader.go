package tenant

import (
	"context"
	"fmt"
	"regexp"

	kcptenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	commonsCtx "github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/errors"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/platform-mesh/golang-commons/policy_services"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

var accountsGVR = schema.GroupVersionResource{
	Group:    "core.platform-mesh.io",
	Version:  "v1alpha1",
	Resource: "accounts",
}
var workspaceGVR = schema.GroupVersionResource{
	Group:    "tenancy.kcp.io",
	Version:  "v1alpha1",
	Resource: "workspaces",
}

type TenantReader struct {
	log             *logger.Logger
	mgr             mcmanager.Manager
	orgsClusterName string
}

func NewTenantReader(logger *logger.Logger, mgr mcmanager.Manager, orgsClusterName string) (policy_services.TenantIdReader, error) { // nolint: ireturn
	return &TenantReader{
		log:             logger,
		mgr:             mgr,
		orgsClusterName: orgsClusterName,
	}, nil
}

func (t *TenantReader) Read(parentCtx context.Context) (string, error) {
	tenant, err := t.GetTenant(parentCtx)
	if err != nil {
		return "", err
	}
	return tenant, nil
}

func (s *TenantReader) GetTenant(ctx context.Context) (string, error) {
	tenantId, err := s.GetTenantForContext(ctx)
	if err != nil {
		return "", err
	}
	if tenantId == "" {
		return "", errors.New("tenant configuration not found")
	}
	return tenantId, nil
}

func (s *TenantReader) GetTenantForContext(ctx context.Context) (string, error) {
	tokenInfo, err := commonsCtx.GetWebTokenFromContext(ctx)
	if err != nil {
		return "", err
	}

	cluster, err := s.mgr.GetCluster(ctx, s.orgsClusterName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get orgs cluster from multicluster manager")
	}

	// Parse realm from issuer
	regex := regexp.MustCompile(`^.*\/realms\/(.*?)\/?$`)
	if !regex.MatchString(tokenInfo.Issuer) {
		return "", errors.New("token issuer is not valid")
	}

	realm := regex.FindStringSubmatch(tokenInfo.Issuer)[1]
	s.log.Debug().Str("realm", realm).Msg("Parsed realm from issuer")

	if realm == "welcome" {
		return "", errors.New("invalid tenant")
	}

	acc := &accountsv1alpha1.Account{}
	err = cluster.GetClient().Get(ctx, client.ObjectKey{Name: realm}, acc)
	if err != nil {
		return "", errors.Wrap(err, "failed to get account from kcp")
	}

	if acc.Spec.Type != "org" {
		return "", errors.New("invalid account type, expected 'org'")
	}

	ws := &kcptenancyv1alpha1.Workspace{}
	err = cluster.GetClient().Get(ctx, client.ObjectKey{Name: acc.Name}, ws)
	if err != nil {
		return "", errors.Wrap(err, "failed to get workspace from kcp")
	}

	parentClusterId := ws.Annotations["kcp.io/cluster"]
	if parentClusterId == "" {
		return "", errors.New("parent cluster not found")
	}
	return fmt.Sprintf("%s/%s/%s", parentClusterId, realm, ws.Spec.Cluster), nil
}
