package context

import (
	"context"
	"testing"

	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKCPContext(t *testing.T) {
	ctx := context.Background()

	// Test setting and getting KCP context
	kcpCtx := KCPContext{
		IDMTenant:        "test-tenant",
		OrganizationName: "test-org",
	}

	// Set KCP context
	ctxWithKCP := SetKCPContext(ctx, kcpCtx)

	// Get KCP context
	retrievedKCP, err := GetKCPContext(ctxWithKCP)
	require.NoError(t, err)
	assert.Equal(t, kcpCtx.IDMTenant, retrievedKCP.IDMTenant)
	assert.Equal(t, kcpCtx.OrganizationName, retrievedKCP.OrganizationName)

	// Test getting KCP context from empty context
	_, err = GetKCPContext(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kcp user context not found in context")
}

func TestAccountInfo(t *testing.T) {
	ctx := context.Background()

	// Test setting and getting account info (use a minimal structure)
	accountInfo := &accountsv1alpha1.AccountInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-account",
		},
	}

	// Set account info
	ctxWithAccount := SetAccountInfo(ctx, accountInfo)

	// Get account info
	retrievedAccount, err := GetAccountInfo(ctxWithAccount)
	require.NoError(t, err)
	assert.Equal(t, accountInfo.Name, retrievedAccount.Name)

	// Test getting account info from empty context
	_, err = GetAccountInfo(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account info not found in context")
}

func TestContextChaining(t *testing.T) {
	ctx := context.Background()

	// Test setting both contexts
	kcpCtx := KCPContext{
		IDMTenant:        "test-tenant",
		OrganizationName: "test-org",
	}

	accountInfo := &accountsv1alpha1.AccountInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-account",
		},
	}

	// Chain context operations
	ctxWithKCP := SetKCPContext(ctx, kcpCtx)
	ctxWithBoth := SetAccountInfo(ctxWithKCP, accountInfo)

	// Verify both contexts are accessible
	retrievedKCP, err := GetKCPContext(ctxWithBoth)
	require.NoError(t, err)
	assert.Equal(t, kcpCtx.OrganizationName, retrievedKCP.OrganizationName)

	retrievedAccount, err := GetAccountInfo(ctxWithBoth)
	require.NoError(t, err)
	assert.Equal(t, accountInfo.Name, retrievedAccount.Name)
}
