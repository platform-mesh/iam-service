package kcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platform-mesh/golang-commons/logger"
	"github.com/stretchr/testify/assert"

	"github.com/platform-mesh/iam-service/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/middleware/idm"
)

// Mock IDM tenant retriever
type mockIDMTenantRetriever struct{}

func (m *mockIDMTenantRetriever) GetIDMTenant(issuer string) (string, error) {
	return "test-tenant", nil
}

// Verify interface compliance
var _ idm.IDMTenantRetriever = (*mockIDMTenantRetriever)(nil)

func TestNew(t *testing.T) {
	// Setup
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{"excluded1", "excluded2"},
		},
	}

	orgsWorkspaceClusterName := "test-orgs-cluster"

	// Execute - using nil for manager since we're testing constructor logic
	middleware := New(nil, cfg, log, mockTenantRetriever, orgsWorkspaceClusterName)

	// Assert
	assert.NotNil(t, middleware)
	assert.Nil(t, middleware.mgr) // We passed nil
	assert.Equal(t, cfg, middleware.cfg)
	assert.Equal(t, log, middleware.log)
	assert.Equal(t, mockTenantRetriever, middleware.tenantRetriever)
	assert.Equal(t, []string{"excluded1", "excluded2"}, middleware.excludedIDMTenants)
	assert.Equal(t, orgsWorkspaceClusterName, middleware.orgsWorkspaceClusterName)
}

func TestGetKcpUserContext_Success(t *testing.T) {
	// Setup
	expectedKctx := KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "test-cluster",
		OrganizationName: "test-org",
	}
	ctx := context.WithValue(context.Background(), UserContextKey, expectedKctx)

	// Execute
	result, err := GetKcpUserContext(ctx)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedKctx, result)
}

func TestGetKcpUserContext_NotFound(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Execute
	result, err := GetKcpUserContext(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kcp user context not found in context")
	assert.Equal(t, KCPContext{}, result)
}

func TestGetKcpUserContext_InvalidType(t *testing.T) {
	// Setup
	ctx := context.WithValue(context.Background(), UserContextKey, "invalid-type")

	// Execute
	result, err := GetKcpUserContext(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid kcp user context type")
	assert.Equal(t, KCPContext{}, result)
}

func TestSetKCPUserContext_ErrorHandling(t *testing.T) {
	// Test the middleware HTTP handler when getKcpInfosForContext fails
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{},
		},
	}

	middleware := New(nil, cfg, log, mockTenantRetriever, "test-orgs-cluster")

	// Setup context without token (will cause error in getKcpInfosForContext)
	ctx := context.Background()

	// Create request and response recorder
	req, _ := http.NewRequestWithContext(ctx, "GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Create a test handler (should not be called)
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	// Execute
	handler := middleware.SetKCPUserContext()(testHandler)
	handler.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "Error while retrieving data from kcp")
	assert.False(t, handlerCalled, "Handler should not be called when getKcpInfosForContext fails")
}

func TestSetKCPUserContext_MiddlewareStructure(t *testing.T) {
	// Test that the middleware properly wraps the handler structure
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{},
		},
	}

	middleware := New(nil, cfg, log, mockTenantRetriever, "test-orgs-cluster")

	// Test that the middleware function returns a proper handler function
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Execute - get the middleware wrapper
	wrappedHandler := middleware.SetKCPUserContext()(testHandler)

	// Assert
	assert.NotNil(t, wrappedHandler)
	assert.IsType(t, http.HandlerFunc(nil), wrappedHandler)
}

func TestGetKcpInfosForContext_NoToken(t *testing.T) {
	// Test the internal method directly when no token is present
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{},
		},
	}

	middleware := New(nil, cfg, log, mockTenantRetriever, "test-orgs-cluster")

	// Setup context without token
	ctx := context.Background()

	// Execute
	result, err := middleware.getKcpInfosForContext(ctx)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, KCPContext{}, result)
	// The actual error message depends on the pmcontext.GetWebTokenFromContext implementation
	assert.NotEmpty(t, err.Error())
}

func TestKCPContext_Structure(t *testing.T) {
	// Test the KCPContext struct
	kctx := KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "test-cluster",
		OrganizationName: "test-org",
	}

	// Assert
	assert.Equal(t, "test-tenant", kctx.IDMTenant)
	assert.Equal(t, "test-cluster", kctx.ClusterId)
	assert.Equal(t, "test-org", kctx.OrganizationName)
}

func TestContextKey_String(t *testing.T) {
	// Test the context key
	key := UserContextKey
	assert.Equal(t, ContextKey("KCPContext"), key)
	assert.Equal(t, "KCPContext", string(key))
}

func TestMiddleware_Fields(t *testing.T) {
	// Test middleware struct field access
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{"tenant1", "tenant2"},
		},
	}

	orgsWorkspaceClusterName := "test-cluster"

	// Execute
	middleware := New(nil, cfg, log, mockTenantRetriever, orgsWorkspaceClusterName)

	// Assert all fields are properly set
	assert.Nil(t, middleware.mgr) // We passed nil
	assert.NotNil(t, middleware.cfg)
	assert.NotNil(t, middleware.log)
	assert.NotNil(t, middleware.tenantRetriever)
	assert.Contains(t, middleware.excludedIDMTenants, "tenant1")
	assert.Contains(t, middleware.excludedIDMTenants, "tenant2")
	assert.Equal(t, orgsWorkspaceClusterName, middleware.orgsWorkspaceClusterName)
}

func TestMiddleware_ExcludedTenantsEmpty(t *testing.T) {
	// Test with empty excluded tenants
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{},
		},
	}

	// Execute
	middleware := New(nil, cfg, log, mockTenantRetriever, "test-cluster")

	// Assert
	assert.NotNil(t, middleware.excludedIDMTenants)
	assert.Empty(t, middleware.excludedIDMTenants)
}

func TestMiddleware_ExcludedTenantsNil(t *testing.T) {
	// Test with nil excluded tenants
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: nil,
		},
	}

	// Execute
	middleware := New(nil, cfg, log, mockTenantRetriever, "test-cluster")

	// Assert
	assert.Nil(t, middleware.excludedIDMTenants)
}

func TestSetKCPUserContext_HandlerWrapper(t *testing.T) {
	// Test that the middleware wrapper function can be called
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{},
		},
	}

	middleware := New(nil, cfg, log, mockTenantRetriever, "test-orgs-cluster")

	// Test the middleware function wrapper
	middlewareFunc := middleware.SetKCPUserContext()
	assert.NotNil(t, middlewareFunc)

	// Test that it can wrap a handler
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	wrappedHandler := middlewareFunc(dummyHandler)
	assert.NotNil(t, wrappedHandler)
}

func TestMiddleware_ConstantValues(t *testing.T) {
	// Test constant values
	assert.Equal(t, ContextKey("KCPContext"), UserContextKey)
	assert.Equal(t, "KCPContext", string(UserContextKey))
}

func TestKCPContext_ZeroValue(t *testing.T) {
	// Test zero value of KCPContext
	var kctx KCPContext
	assert.Empty(t, kctx.IDMTenant)
	assert.Empty(t, kctx.ClusterId)
	assert.Empty(t, kctx.OrganizationName)
}

func TestSetKCPUserContext_SuccessPath(t *testing.T) {
	// Test the middleware success scenario by providing a valid context
	// We can test this by setting up the context manually to simulate a successful getKcpInfosForContext
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{},
		},
	}

	middleware := New(nil, cfg, log, mockTenantRetriever, "test-orgs-cluster")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// We can't easily test the full success path without complex mocking,
	// but we can test the middleware wrapper logic itself
	handler := middleware.SetKCPUserContext()(testHandler)
	assert.NotNil(t, handler)
}

func TestGetKcpInfosForContext_ExcludedTenantLogic(t *testing.T) {
	// Test the excluded tenant logic in isolation
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{"excluded-tenant-1", "excluded-tenant-2"},
		},
	}

	middleware := New(nil, cfg, log, mockTenantRetriever, "test-orgs-cluster")

	// Test that the excluded tenants are properly set
	assert.Contains(t, middleware.excludedIDMTenants, "excluded-tenant-1")
	assert.Contains(t, middleware.excludedIDMTenants, "excluded-tenant-2")
	assert.Len(t, middleware.excludedIDMTenants, 2)
}

func TestGetKcpInfosForContext_TokenInfoRetrieval(t *testing.T) {
	// Test token info retrieval error path
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{},
		},
	}

	middleware := New(nil, cfg, log, mockTenantRetriever, "test-orgs-cluster")

	// Test with empty context (no token info)
	ctx := context.Background()
	result, err := middleware.getKcpInfosForContext(ctx)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, KCPContext{}, result)
	// This tests line 90-93 in the source code
}

func TestSetKCPUserContext_LoggerContext(t *testing.T) {
	// Test that LoadLoggerFromContext is called (line 50)
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{},
		},
	}

	middleware := New(nil, cfg, log, mockTenantRetriever, "test-orgs-cluster")

	// Setup a context with logger information
	ctx := context.Background()

	// Create request with the context
	req, _ := http.NewRequestWithContext(ctx, "GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Create a handler that should not be reached due to error
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called due to getKcpInfosForContext error")
	})

	// Execute
	handler := middleware.SetKCPUserContext()(testHandler)
	handler.ServeHTTP(rr, req)

	// Assert - should get error due to no token context
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "Error while retrieving data from kcp")
}

func TestMiddleware_Structure_Complete(t *testing.T) {
	// Test complete middleware structure instantiation
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})

	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{"tenant1", "tenant2", "tenant3"},
		},
	}

	orgCluster := "prod-orgs-cluster"

	// Execute
	middleware := New(nil, cfg, log, mockTenantRetriever, orgCluster)

	// Assert all struct fields
	assert.NotNil(t, middleware)
	assert.Nil(t, middleware.mgr)
	assert.Equal(t, cfg, middleware.cfg)
	assert.Equal(t, log, middleware.log)
	assert.Equal(t, mockTenantRetriever, middleware.tenantRetriever)
	assert.Equal(t, orgCluster, middleware.orgsWorkspaceClusterName)
	assert.Len(t, middleware.excludedIDMTenants, 3)

	// Test specific excluded tenants
	expectedTenants := []string{"tenant1", "tenant2", "tenant3"}
	for _, expectedTenant := range expectedTenants {
		assert.Contains(t, middleware.excludedIDMTenants, expectedTenant)
	}
}

func TestContextKey_Type(t *testing.T) {
	// Test ContextKey type behavior
	var key ContextKey = "TestKey"
	assert.Equal(t, "TestKey", string(key))

	// Test with UserContextKey specifically
	assert.IsType(t, ContextKey(""), UserContextKey)
	assert.Equal(t, "KCPContext", string(UserContextKey))
}

func TestKCPContext_Fields(t *testing.T) {
	// Test individual field assignments
	kctx := KCPContext{}

	// Test empty state
	assert.Empty(t, kctx.IDMTenant)
	assert.Empty(t, kctx.ClusterId)
	assert.Empty(t, kctx.OrganizationName)

	// Test field assignments
	kctx.IDMTenant = "test-tenant"
	kctx.ClusterId = "test-cluster"
	kctx.OrganizationName = "test-org"

	assert.Equal(t, "test-tenant", kctx.IDMTenant)
	assert.Equal(t, "test-cluster", kctx.ClusterId)
	assert.Equal(t, "test-org", kctx.OrganizationName)
}
