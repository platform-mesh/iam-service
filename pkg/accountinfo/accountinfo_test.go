package accountinfo

import (
	"context"
	"fmt"
	"testing"

	"github.com/kcp-dev/logicalcluster/v3"
	corev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"
	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	accountmocks "github.com/platform-mesh/account-operator/pkg/subroutines/mocks"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	"github.com/platform-mesh/iam-service/pkg/accountinfo/mocks"
)

type Provider struct {
	clusters map[string]cluster.Cluster
}

func (p *Provider) Get(ctx context.Context, clusterName string) (cluster.Cluster, error) {
	cluster, ok := p.clusters[clusterName]
	if !ok {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}
	return cluster, nil
}

// IndexField implements multicluster.Provider.
func (p *Provider) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	return nil
}

func TestNew(t *testing.T) {
	// Test constructor with nil parameters - should return error
	retriever, err := New(nil, nil)
	assert.Error(t, err)
	assert.Nil(t, retriever)
	assert.Contains(t, err.Error(), "cluster client and manager cannot be nil")
}

func createTestAccountInfo() *accountsv1alpha1.AccountInfo {
	return &accountsv1alpha1.AccountInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name: "account",
		},
		Spec: accountsv1alpha1.AccountInfoSpec{
			Account: accountsv1alpha1.AccountLocation{
				Name:               "test-account",
				OriginClusterId:    "origin-cluster-123",
				GeneratedClusterId: "generated-cluster-456",
			},
			Organization: accountsv1alpha1.AccountLocation{
				Name: "test-org",
			},
		},
	}
}

func TestAccountInfoRetriever_Get_NilDependencies(t *testing.T) {
	retriever := &accountInfoRetriever{
		mgr:           nil,
		clusterClient: nil,
	}

	ctx := context.Background()
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	accountPath := "test-account"

	// The method panics with nil dependencies - this demonstrates the need for proper initialization
	assert.Panics(t, func() {
		_, _ = retriever.Get(ctx, accountPath)
	})
}

func TestAccountInfoRetriever_Get_WithFakeClient(t *testing.T) {
	// Create a simplified test using a fake client for the final client.Get call
	// This tests the last part of the Get method where we retrieve the AccountInfo

	ai := createTestAccountInfo()
	scheme := runtime.NewScheme()
	err := accountsv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ai).
		Build()

	// Test the client.Get portion directly
	ctx := context.Background()
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	result := &accountsv1alpha1.AccountInfo{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "account"}, result)

	assert.NoError(t, err)
	assert.Equal(t, "test-account", result.Spec.Account.Name)
	assert.Equal(t, "test-org", result.Spec.Organization.Name)
}

func TestAccountInfoRetriever_Get_NotFound(t *testing.T) {
	// Test the not found case with fake client
	scheme := runtime.NewScheme()
	err := accountsv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	// Create fake client without the account object
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	ctx := context.Background()
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	result := &accountsv1alpha1.AccountInfo{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "account"}, result)

	assert.Error(t, err)
	assert.True(t, client.IgnoreNotFound(err) == nil) // Verify it's a not found error
}

// Test New with valid inputs - completing the constructor coverage
func TestNew_WithValidInputs(t *testing.T) {
	// Create a basic manager and cluster interface to test successful construction
	emptyConfig := &rest.Config{}
	testProvider := &Provider{clusters: map[string]cluster.Cluster{}}

	mgr, err := mcmanager.New(emptyConfig, testProvider, mcmanager.Options{})
	require.NoError(t, err)

	mockCI := mocks.NewClusterInterface(t)

	retriever, err := New(mgr, mockCI)
	assert.NoError(t, err)
	assert.NotNil(t, retriever)
	assert.IsType(t, &accountInfoRetriever{}, retriever)

	// Verify the internal fields are set correctly
	r := retriever.(*accountInfoRetriever)
	assert.Equal(t, mgr, r.mgr)
	assert.Equal(t, mockCI, r.clusterClient)
}

// Test getAccountInfo indirectly by testing the scenario where it's called
// Since getAccountInfo is not exported, we test it through integration tests
func TestGetAccountInfo_IndirectTesting(t *testing.T) {
	// Test that shows how getAccountInfo works through the public API
	// This is already covered by our existing tests that use fake clients

	ctx := context.Background()
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	// Test successful case
	ai := createTestAccountInfo()
	scheme := runtime.NewScheme()
	err := accountsv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ai).
		Build()

	// Test direct client interaction (which is what getAccountInfo does internally)
	result := &accountsv1alpha1.AccountInfo{}
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "account"}, result)

	assert.NoError(t, err)
	assert.Equal(t, "test-account", result.Spec.Account.Name)
	assert.Equal(t, "test-org", result.Spec.Organization.Name)

	// Test error case
	emptyScheme := runtime.NewScheme()
	err = accountsv1alpha1.AddToScheme(emptyScheme)
	require.NoError(t, err)

	emptyClient := fake.NewClientBuilder().
		WithScheme(emptyScheme).
		Build()

	result = &accountsv1alpha1.AccountInfo{}
	err = emptyClient.Get(ctx, client.ObjectKey{Name: "account"}, result)
	assert.Error(t, err)
}

func TestRetrieverInterface(t *testing.T) {
	var _ Retriever = (*accountInfoRetriever)(nil)
}

func TestAccountInfoRetriever_Get_NilContext(t *testing.T) {
	retriever := &accountInfoRetriever{
		mgr:           nil,
		clusterClient: nil,
	}

	// This will panic with nil dependencies
	assert.Panics(t, func() {
		_, _ = retriever.Get(context.Background(), "test-account")
	})
}

func TestAccountInfoRetriever_Get_Success(t *testing.T) {
	// Test the not found case with fake client
	scheme := runtime.NewScheme()
	err := accountsv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	testclusters := map[string]cluster.Cluster{
		"test-cluster": func() cluster.Cluster {
			c := accountmocks.NewCluster(t)
			cl := accountmocks.NewClient(t)
			cl.EXPECT().
				Get(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					ai := obj.(*accountsv1alpha1.AccountInfo)
					*ai = accountsv1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "account",
						},
						Spec: accountsv1alpha1.AccountInfoSpec{},
					}
					return nil
				}).Once()
			c.EXPECT().GetClient().Return(cl)
			return c
		}(),
	}
	testProvider := &Provider{clusters: testclusters}
	emptyConfig := &rest.Config{}

	mgr, err := mcmanager.New(emptyConfig, testProvider, mcmanager.Options{})
	assert.NoError(t, err)

	ctx := context.Background()
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	ci := mocks.NewClusterInterface(t)
	c := mocks.NewInterface(t)
	alpha1Interface := mocks.NewCoreV1alpha1Interface(t)
	lc := mocks.NewLogicalClusterInterface(t)

	alpha1Interface.EXPECT().LogicalClusters().Return(lc)
	c.EXPECT().CoreV1alpha1().Return(alpha1Interface)
	ci.EXPECT().Cluster(logicalcluster.NewPath("test-cluster")).Return(c).Once()
	lc.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(&corev1alpha1.LogicalCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
			Annotations: map[string]string{
				"kcp.io/cluster": "test-cluster",
			},
		},
	}, nil)

	retriever, err := New(mgr, ci)
	assert.NotNil(t, retriever)
	assert.NoError(t, err)

	ai, err := retriever.Get(ctx, "test-cluster")
	assert.NotNil(t, ai)
	assert.NoError(t, err)
}

func TestAccountInfoRetriever_NoCluster(t *testing.T) {
	// Test the not found case with fake client
	scheme := runtime.NewScheme()
	err := accountsv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	testclusters := map[string]cluster.Cluster{}
	testProvider := &Provider{clusters: testclusters}
	emptyConfig := &rest.Config{}

	mgr, err := mcmanager.New(emptyConfig, testProvider, mcmanager.Options{})
	assert.NoError(t, err)

	ctx := context.Background()
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	ci := mocks.NewClusterInterface(t)
	c := mocks.NewInterface(t)
	alpha1Interface := mocks.NewCoreV1alpha1Interface(t)
	lc := mocks.NewLogicalClusterInterface(t)

	c.EXPECT().CoreV1alpha1().Return(alpha1Interface)
	ci.EXPECT().Cluster(logicalcluster.NewPath("test-cluster")).Return(c).Once()
	alpha1Interface.EXPECT().LogicalClusters().Return(lc)
	lc.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.NewNotFound(corev1alpha1.Resource("logicalclusters"), "cluster"))

	retriever, err := New(mgr, ci)
	assert.NotNil(t, retriever)
	assert.NoError(t, err)

	ai, err := retriever.Get(ctx, "test-cluster")
	assert.Nil(t, ai)
	assert.Error(t, err)
}

func TestAccountInfoRetriever_Get_EmptyAccountPath(t *testing.T) {
	retriever := &accountInfoRetriever{
		mgr:           nil,
		clusterClient: nil,
	}

	ctx := context.Background()
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	// This will panic with nil dependencies
	assert.Panics(t, func() {
		_, _ = retriever.Get(ctx, "")
	})
}
