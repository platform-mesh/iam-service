package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	pmconfig "github.com/platform-mesh/golang-commons/config"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/platform-mesh/iam-service/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/resolver"
)

// testResolverService is a minimal service implementation for router HTTP tests
// Router tests focus on HTTP routing behavior, not GraphQL business logic,
// so a simple test implementation is appropriate here
type testResolverService struct{}

func (s *testResolverService) Me(ctx context.Context) (*graph.User, error) {
	return &graph.User{UserID: "test", Email: "test@example.com"}, nil
}

func (s *testResolverService) User(ctx context.Context, userID string) (*graph.User, error) {
	return &graph.User{UserID: userID, Email: userID + "@example.com"}, nil
}

func (s *testResolverService) Users(ctx context.Context, resourceContext graph.ResourceContext, roleFilters []string, sortBy *graph.SortByInput, page *graph.PageInput) (*graph.UserConnection, error) {
	return &graph.UserConnection{
		Users:    []*graph.UserRoles{},
		PageInfo: &graph.PageInfo{Count: 0, TotalCount: 0, HasNextPage: false, HasPreviousPage: false},
	}, nil
}

func (s *testResolverService) Roles(ctx context.Context, resourceContext graph.ResourceContext) ([]*graph.Role, error) {
	return []*graph.Role{}, nil
}

func (s *testResolverService) AssignRolesToUsers(ctx context.Context, resourceContext graph.ResourceContext, changes []*graph.UserRoleChange) (*graph.RoleAssignmentResult, error) {
	return &graph.RoleAssignmentResult{Success: true, AssignedCount: 0}, nil
}

func (s *testResolverService) RemoveRole(ctx context.Context, resourceContext graph.ResourceContext, input graph.RemoveRoleInput) (*graph.RoleRemovalResult, error) {
	return &graph.RoleRemovalResult{Success: true, WasAssigned: true}, nil
}

// createTestResolver creates a GraphQL resolver for HTTP routing tests
// Since router tests focus on HTTP behavior (CORS, middleware, endpoints) rather than
// GraphQL business logic, a simple test service implementation is appropriate
func createTestResolver(t *testing.T) graph.ResolverRoot {
	// Create minimal test service for HTTP routing tests
	resolverService := &testResolverService{}

	// Create logger
	log, err := logger.New(logger.Config{})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create GraphQL resolver
	return resolver.New(resolverService, log)
}

// createEmptyDirectiveRoot returns an empty DirectiveRoot for testing
func createEmptyDirectiveRoot() graph.DirectiveRoot {
	return graph.DirectiveRoot{}
}

func TestCreateRouter_BasicConfiguration(t *testing.T) {
	// Setup
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: false,
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	// Execute
	router := CreateRouter(commonCfg, serviceCfg, resolver, log, nil, createEmptyDirectiveRoot())

	// Assert
	assert.NotNil(t, router)
	assert.IsType(t, &chi.Mux{}, router)
}

func TestCreateRouter_LocalEnvironment_EnablesCORS(t *testing.T) {
	// Setup
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: true,
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	// Execute
	router := CreateRouter(commonCfg, serviceCfg, resolver, log, nil, createEmptyDirectiveRoot())

	// Test CORS with a simple actual request (not preflight)
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "{ __typename }"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// CORS should add origin header to actual requests
	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))

	// Also test preflight request
	preflightReq := httptest.NewRequest("OPTIONS", "/graphql", nil)
	preflightReq.Header.Set("Origin", "http://localhost:3000")
	preflightReq.Header.Set("Access-Control-Request-Method", "POST")
	preflightReq.Header.Set("Access-Control-Request-Headers", "Content-Type")

	preflightRR := httptest.NewRecorder()
	router.ServeHTTP(preflightRR, preflightReq)

	// Preflight should return 204 and have vary headers
	assert.Equal(t, http.StatusNoContent, preflightRR.Code)
	assert.Contains(t, preflightRR.Header().Get("Vary"), "Origin")
}

func TestCreateRouter_LocalEnvironment_EnablesPlayground(t *testing.T) {
	// Setup
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: true,
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	// Execute
	router := CreateRouter(commonCfg, serviceCfg, resolver, log, nil, createEmptyDirectiveRoot())

	// Assert - Test playground endpoint
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Should contain GraphQL playground content
	body := rr.Body.String()
	assert.Contains(t, body, "GraphQL")
}

func TestCreateRouter_ProductionEnvironment_DisablesPlayground(t *testing.T) {
	// Setup
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: false,
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	// Execute
	router := CreateRouter(commonCfg, serviceCfg, resolver, log, nil, createEmptyDirectiveRoot())

	// Assert - Test playground endpoint should return 404
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestCreateRouter_GraphQLEndpoint_Available(t *testing.T) {
	// Setup
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: false,
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	// Execute
	router := CreateRouter(commonCfg, serviceCfg, resolver, log, nil, createEmptyDirectiveRoot())

	// Assert - Test GraphQL endpoint responds
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "{ __typename }"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should not return 404 (GraphQL handler should handle the request)
	assert.NotEqual(t, http.StatusNotFound, rr.Code)
}

func TestCreateRouter_WithMiddleware(t *testing.T) {
	// Setup
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: false,
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	middlewareCalled := false
	testMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			w.Header().Set("X-Test-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	middlewares := []func(http.Handler) http.Handler{testMiddleware}

	// Execute
	router := CreateRouter(commonCfg, serviceCfg, resolver, log, middlewares, createEmptyDirectiveRoot())

	// Assert - Test middleware is applied to GraphQL endpoint
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "{ __typename }"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.True(t, middlewareCalled, "Middleware should have been called")
	assert.Equal(t, "applied", rr.Header().Get("X-Test-Middleware"))
}

func TestCreateRouter_GraphQLHandlerConfiguration(t *testing.T) {
	// Setup
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: false,
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	// Execute
	router := CreateRouter(commonCfg, serviceCfg, resolver, log, nil, createEmptyDirectiveRoot())

	// Assert - Test various GraphQL transports are configured
	testCases := []struct {
		name   string
		method string
		body   string
	}{
		{
			name:   "POST request",
			method: "POST",
			body:   `{"query": "{ __typename }"}`,
		},
		{
			name:   "GET request",
			method: "GET",
			body:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, "/graphql", strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, "/graphql?query={__typename}", nil)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			// Should not return 404 or 405 (method not allowed)
			assert.NotEqual(t, http.StatusNotFound, rr.Code)
			assert.NotEqual(t, http.StatusMethodNotAllowed, rr.Code)
		})
	}
}

func TestCreateRouter_CORSConfiguration(t *testing.T) {
	// Setup for local environment
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: true,
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	router := CreateRouter(commonCfg, serviceCfg, resolver, log, nil, createEmptyDirectiveRoot())

	// Test CORS headers are properly set with actual requests
	testCases := []struct {
		name   string
		origin string
	}{
		{
			name:   "Local development origin",
			origin: "http://localhost:3000",
		},
		{
			name:   "Different origin",
			origin: "https://example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with actual POST request to see CORS response headers
			req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "{ __typename }"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", tc.origin)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			// With AllowedOrigins: ["*"], CORS returns "*" as the origin
			assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
			// Check for credentials header
			assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
		})
	}
}

func TestCreateRouter_MiddlewareOrder(t *testing.T) {
	// Setup
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: false,
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	var callOrder []string

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "middleware1")
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "middleware2")
			next.ServeHTTP(w, r)
		})
	}

	middlewares := []func(http.Handler) http.Handler{middleware1, middleware2}

	// Execute
	router := CreateRouter(commonCfg, serviceCfg, resolver, log, middlewares, createEmptyDirectiveRoot())

	// Assert - Test middleware order
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "{ __typename }"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Len(t, callOrder, 2)
	assert.Equal(t, "middleware1", callOrder[0])
	assert.Equal(t, "middleware2", callOrder[1])
}

func TestCreateRouter_EmptyMiddlewareSlice(t *testing.T) {
	// Setup
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: false,
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	// Execute with empty middleware slice
	router := CreateRouter(commonCfg, serviceCfg, resolver, log, []func(http.Handler) http.Handler{}, createEmptyDirectiveRoot())

	// Assert
	assert.NotNil(t, router)

	// Should still work without middleware
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "{ __typename }"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.NotEqual(t, http.StatusNotFound, rr.Code)
}

func TestCreateRouter_NilMiddleware(t *testing.T) {
	// Setup
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: false,
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	// Execute with nil middleware
	router := CreateRouter(commonCfg, serviceCfg, resolver, log, nil, createEmptyDirectiveRoot())

	// Assert
	assert.NotNil(t, router)

	// Should still work without middleware
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query": "{ __typename }"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.NotEqual(t, http.StatusNotFound, rr.Code)
}

func TestCreateRouter_GraphQLIntrospection(t *testing.T) {
	// Setup
	commonCfg := &pmconfig.CommonServiceConfig{
		IsLocal: true, // Enable introspection in local mode
	}
	serviceCfg := &config.ServiceConfig{}
	resolver := createTestResolver(t)
	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)

	// Execute
	router := CreateRouter(commonCfg, serviceCfg, resolver, log, nil, createEmptyDirectiveRoot())

	// Assert - Test introspection query
	introspectionQuery := `{"query": "{ __schema { types { name } } }"}`
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(introspectionQuery))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should not return error (introspection should be enabled)
	assert.NotEqual(t, http.StatusNotFound, rr.Code)
	assert.NotEqual(t, http.StatusMethodNotAllowed, rr.Code)
}
