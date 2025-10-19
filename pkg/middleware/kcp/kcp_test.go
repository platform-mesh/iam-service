package kcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	kcptenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	"github.com/platform-mesh/golang-commons/jwt"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/platform-mesh/iam-service/pkg/config"
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

func TestGetKcpUserContext(t *testing.T) {
	tests := []struct {
		name         string
		contextValue interface{}
		expectError  bool
		expectedErr  string
	}{
		{
			name: "success",
			contextValue: KCPContext{
				IDMTenant:        "test-tenant",
				ClusterId:        "test-cluster",
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
		{
			name:         "invalid type",
			contextValue: "invalid-type",
			expectError:  true,
			expectedErr:  "invalid kcp user context type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.contextValue != nil {
				ctx = context.WithValue(context.Background(), UserContextKey, tt.contextValue)
			} else {
				ctx = context.Background()
			}

			result, err := GetKcpUserContext(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Equal(t, KCPContext{}, result)
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

func TestGetKCPInfosForContext(t *testing.T) {
	tests := []struct {
		name            string
		tenantRetriever *mockIDMTenantRetriever
		excludedTenants []string
		setupClient     func() *fake.ClientBuilder
		expectError     bool
		expectedErr     string
		expectedResult  *KCPContext
	}{
		{
			name:            "success",
			tenantRetriever: &mockIDMTenantRetriever{tenant: "test-tenant"},
			excludedTenants: []string{},
			setupClient: func() *fake.ClientBuilder {
				scheme := runtime.NewScheme()
				_ = accountsv1alpha1.AddToScheme(scheme)
				_ = kcptenancyv1alpha1.AddToScheme(scheme)

				account := &accountsv1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{Name: "test-tenant"},
					Spec:       accountsv1alpha1.AccountSpec{Type: "org"},
				}
				workspace := &kcptenancyv1alpha1.Workspace{
					ObjectMeta: metav1.ObjectMeta{Name: "test-tenant"},
					Spec:       kcptenancyv1alpha1.WorkspaceSpec{Cluster: "test-cluster"},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(account, workspace)
			},
			expectError: false,
			expectedResult: &KCPContext{
				IDMTenant:        "test-tenant",
				ClusterId:        "test-cluster",
				OrganizationName: "test-tenant",
			},
		},
		{
			name:            "idm tenant retrieval error",
			tenantRetriever: &mockIDMTenantRetriever{err: fmt.Errorf("idm service unavailable")},
			excludedTenants: []string{},
			setupClient: func() *fake.ClientBuilder {
				scheme := runtime.NewScheme()
				return fake.NewClientBuilder().WithScheme(scheme)
			},
			expectError: true,
			expectedErr: "failed to get idm tenant from token issuer",
		},
		{
			name:            "excluded tenant",
			tenantRetriever: &mockIDMTenantRetriever{tenant: "excluded-tenant"},
			excludedTenants: []string{"excluded-tenant"},
			setupClient: func() *fake.ClientBuilder {
				scheme := runtime.NewScheme()
				return fake.NewClientBuilder().WithScheme(scheme)
			},
			expectError: true,
			expectedErr: "invalid tenant",
		},
		{
			name:            "account not found",
			tenantRetriever: &mockIDMTenantRetriever{tenant: "missing-tenant"},
			excludedTenants: []string{},
			setupClient: func() *fake.ClientBuilder {
				scheme := runtime.NewScheme()
				_ = accountsv1alpha1.AddToScheme(scheme)
				_ = kcptenancyv1alpha1.AddToScheme(scheme)
				return fake.NewClientBuilder().WithScheme(scheme)
			},
			expectError: true,
			expectedErr: "failed to get account from kcp",
		},
		{
			name:            "invalid account type",
			tenantRetriever: &mockIDMTenantRetriever{tenant: "user-account"},
			excludedTenants: []string{},
			setupClient: func() *fake.ClientBuilder {
				scheme := runtime.NewScheme()
				_ = accountsv1alpha1.AddToScheme(scheme)
				_ = kcptenancyv1alpha1.AddToScheme(scheme)

				account := &accountsv1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{Name: "user-account"},
					Spec:       accountsv1alpha1.AccountSpec{Type: "user"},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(account)
			},
			expectError: true,
			expectedErr: "invalid account type, expected 'org'",
		},
		{
			name:            "workspace not found",
			tenantRetriever: &mockIDMTenantRetriever{tenant: "no-workspace"},
			excludedTenants: []string{},
			setupClient: func() *fake.ClientBuilder {
				scheme := runtime.NewScheme()
				_ = accountsv1alpha1.AddToScheme(scheme)
				_ = kcptenancyv1alpha1.AddToScheme(scheme)

				account := &accountsv1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{Name: "no-workspace"},
					Spec:       accountsv1alpha1.AccountSpec{Type: "org"},
				}

				return fake.NewClientBuilder().WithScheme(scheme).WithObjects(account)
			},
			expectError: true,
			expectedErr: "failed to get workspace from kcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, _ := logger.New(logger.Config{Level: "debug"})
			cfg := &config.ServiceConfig{
				IDM: struct {
					ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
				}{
					ExcludedTenants: tt.excludedTenants,
				},
			}

			middleware := New(nil, cfg, log, tt.tenantRetriever, "test-orgs-cluster")
			fakeClient := tt.setupClient().Build()

			mockToken := jwt.WebToken{
				IssuerAttributes: jwt.IssuerAttributes{
					Issuer: "test-issuer",
				},
			}

			result, err := middleware.getKCPInfosForContext(context.Background(), mockToken, KCPContext{}, fakeClient)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Equal(t, KCPContext{}, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, *tt.expectedResult, result)
			}
		})
	}
}
