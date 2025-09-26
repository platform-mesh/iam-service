package tenant

import (
	"context"
	"fmt"
	"regexp"

	commonsCtx "github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/errors"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/platform-mesh/golang-commons/policy_services"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/platform-mesh/iam-service/internal/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/db"
)

var accountsGVR = schema.GroupVersionResource{
	Group:    "core.platform-mesh.io",
	Version:  "v1alpha1",
	Resource: "accounts",
}

type TenantReader struct {
	log       *logger.Logger
	dynClient dynamic.Interface
}

func NewTenantReader(logger *logger.Logger, database db.Service, appConfig config.Config) (policy_services.TenantIdReader, error) { // nolint: ireturn
	cfg, err := clientcmd.BuildConfigFromFlags("", appConfig.KCP.Kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build kcp rest config")
	}

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dynamic client")
	}
	return &TenantReader{
		log:       logger,
		dynClient: dynClient,
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

	unstructuredAccount, err := s.dynClient.Resource(accountsGVR).Get(ctx, realm, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get account from kcp")
	}

	val, found, err := unstructured.NestedString(unstructuredAccount.UnstructuredContent(), "spec", "type")
	if !found || err != nil {
		return "", errors.New("account type not found in kcp")
	}

	if val != "org" {
		return "", errors.New("invalid account type, expected 'org'")
	}
	clusterId, found, err := unstructured.NestedString(unstructuredAccount.UnstructuredContent(), "metadata", "annotations", "kcp.io/cluster")
	if !found || err != nil {
		return "", errors.New("clusterid not found")
	}
	return fmt.Sprintf("%s/%s", clusterId, realm), nil
}
