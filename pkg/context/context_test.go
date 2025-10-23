package context

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestClusterId(t *testing.T) {
	ctx := context.Background()

	// Test setting and getting cluster ID
	clusterId := "test-cluster-123"

	// Set cluster ID
	ctxWithCluster := SetClusterId(ctx, clusterId)

	// Get cluster ID
	retrievedClusterId, err := GetClusterId(ctxWithCluster)
	require.NoError(t, err)
	assert.Equal(t, clusterId, retrievedClusterId)

	// Test getting cluster ID from empty context
	_, err = GetClusterId(ctx)
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

	clusterId := "test-cluster-456"

	// Chain context operations
	ctxWithKCP := SetKCPContext(ctx, kcpCtx)
	ctxWithBoth := SetClusterId(ctxWithKCP, clusterId)

	// Verify both contexts are accessible
	retrievedKCP, err := GetKCPContext(ctxWithBoth)
	require.NoError(t, err)
	assert.Equal(t, kcpCtx.OrganizationName, retrievedKCP.OrganizationName)

	retrievedClusterId, err := GetClusterId(ctxWithBoth)
	require.NoError(t, err)
	assert.Equal(t, clusterId, retrievedClusterId)
}
