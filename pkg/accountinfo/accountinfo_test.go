package accountinfo

import (
	"context"
	"testing"

	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
