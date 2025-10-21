package kcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platform-mesh/golang-commons/logger"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"

	"github.com/platform-mesh/iam-service/pkg/config"
	appcontext "github.com/platform-mesh/iam-service/pkg/context"
	"github.com/platform-mesh/iam-service/pkg/middleware/idm"
)

// Mock IDM tenant retriever
type mockIDMTenantRetriever struct {
	tenant string
	err    error
}

func (m *mockIDMTenantRetriever) GetIDMTenant(issuer string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.tenant != "" {
		return m.tenant, nil
	}
	return "test-tenant", nil
}

var _ idm.IDMTenantRetriever = (*mockIDMTenantRetriever)(nil)

func TestNew(t *testing.T) {
	mockTenantRetriever := &mockIDMTenantRetriever{}
	log, _ := logger.New(logger.Config{Level: "debug"})
	cfg := &config.ServiceConfig{
		IDM: struct {
			ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
		}{
			ExcludedTenants: []string{"excluded1", "excluded2"},
		},
	}

	middleware := New(nil, cfg, log, mockTenantRetriever, "test-orgs-cluster")

	assert.NotNil(t, middleware)
	assert.Equal(t, cfg, middleware.cfg)
	assert.Equal(t, log, middleware.log)
	assert.Equal(t, mockTenantRetriever, middleware.tenantRetriever)
	assert.Equal(t, []string{"excluded1", "excluded2"}, middleware.excludedIDMTenants)
	assert.Equal(t, "test-orgs-cluster", middleware.orgsWorkspaceClusterName)
}

func TestGetKCPContext(t *testing.T) {
	tests := []struct {
		name         string
		contextValue interface{}
		expectError  bool
		expectedErr  string
	}{
		{
			name: "success",
			contextValue: appcontext.KCPContext{
				IDMTenant:        "test-tenant",
				OrganizationName: "test-org",
			},
			expectError: false,
		},
		{
			name:         "not found",
			contextValue: nil,
			expectError:  true,
			expectedErr:  "kcp user context not found in context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.contextValue != nil {
				if kcpCtx, ok := tt.contextValue.(appcontext.KCPContext); ok {
					ctx = appcontext.SetKCPContext(context.Background(), kcpCtx)
				}
			} else {
				ctx = context.Background()
			}

			result, err := appcontext.GetKCPContext(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Equal(t, appcontext.KCPContext{}, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.contextValue, result)
			}
		})
	}
}

func TestSetKCPUserContext(t *testing.T) {
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

	// Test middleware wrapper creation
	middlewareFunc := middleware.SetKCPUserContext()
	assert.NotNil(t, middlewareFunc)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrappedHandler := middlewareFunc(testHandler)
	assert.NotNil(t, wrappedHandler)
	assert.IsType(t, http.HandlerFunc(nil), wrappedHandler)

	// Test error handling (nil manager causes cluster retrieval to fail)
	req, _ := http.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	func() {
		defer func() {
			if r := recover(); r != nil {
				assert.Contains(t, fmt.Sprintf("%v", r), "nil pointer dereference")
			}
		}()
		wrappedHandler.ServeHTTP(rr, req)
	}()
}

// TestGetKCPInfosForContext is removed as the method was refactored away
// The new implementation uses checkToken function and direct subdomain extraction

// Note: Testing the full SetKCPUserContext middleware requires complex setup
// of web tokens, auth headers, and proper manager mocking. For coverage improvement,
// we focus on testing the checkToken function and middleware construction.

// Tests for checkToken function
func TestCheckToken_InvalidURL(t *testing.T) {
	ctx := context.Background()
	ctx = logger.SetLoggerInContext(ctx, logger.StdLogger)

	cfg := &rest.Config{
		Host: "://invalid-url",
	}

	result, err := checkToken(ctx, "Bearer token", "test-org", cfg)

	assert.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "invalid KCP host URL")
}

func TestCheckToken_ValidURL(t *testing.T) {
	// Create a mock HTTP server to simulate KCP API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request path and authorization
		expectedPath := "/clusters/root:orgs:test-org/version"
		if r.URL.Path != expectedPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	ctx = logger.SetLoggerInContext(ctx, logger.StdLogger)

	cfg := &rest.Config{
		Host: server.URL,
	}

	result, err := checkToken(ctx, "Bearer test-token", "test-org", cfg)

	assert.NoError(t, err)
	assert.True(t, result)
}

func TestCheckToken_Unauthorized(t *testing.T) {
	// Create a mock HTTP server that returns unauthorized
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	ctx := context.Background()
	ctx = logger.SetLoggerInContext(ctx, logger.StdLogger)

	cfg := &rest.Config{
		Host: server.URL,
	}

	result, err := checkToken(ctx, "Bearer invalid-token", "test-org", cfg)

	assert.NoError(t, err)
	assert.False(t, result)
}

func TestCheckToken_Forbidden(t *testing.T) {
	// Create a mock HTTP server that returns forbidden (which is considered valid)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	ctx := context.Background()
	ctx = logger.SetLoggerInContext(ctx, logger.StdLogger)

	cfg := &rest.Config{
		Host: server.URL,
	}

	result, err := checkToken(ctx, "Bearer test-token", "test-org", cfg)

	assert.NoError(t, err)
	assert.True(t, result) // Forbidden is considered a valid response
}
