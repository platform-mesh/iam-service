package fga

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"testing"
	"time"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/platform-mesh/iam-service/pkg/config"
	fgamocks "github.com/platform-mesh/iam-service/pkg/fga/mocks"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/middleware/kcp"
	"github.com/platform-mesh/iam-service/pkg/roles"
)

// createTestConfig creates a test configuration
func createTestConfig(t *testing.T) *config.ServiceConfig {
	testRolesFile := filepath.Join("testdata", "roles.yaml")
	return &config.ServiceConfig{
		Roles: struct {
			FilePath string `mapstructure:"roles-file-path" default:"input/roles.yaml"`
		}{
			FilePath: testRolesFile,
		},
		Keycloak: struct {
			BaseURL      string `mapstructure:"keycloak-base-url" default:"https://portal.dev.local:8443/keycloak"`
			ClientID     string `mapstructure:"keycloak-client-id" default:"admin-cli"`
			User         string `mapstructure:"keycloak-user" default:"keycloak-admin"`
			PasswordFile string `mapstructure:"keycloak-password-file" default:".secret/keycloak/password"`
			Cache        struct {
				TTL     time.Duration `mapstructure:"keycloak-cache-ttl" default:"5m"`
				Enabled bool          `mapstructure:"keycloak-cache-enabled" default:"true"`
			} `mapstructure:",squash"`
		}{
			Cache: struct {
				TTL     time.Duration `mapstructure:"keycloak-cache-ttl" default:"5m"`
				Enabled bool          `mapstructure:"keycloak-cache-enabled" default:"true"`
			}{
				TTL: 5 * time.Minute,
			},
		},
	}
}

// createTestService creates a test service with a real roles retriever
func createTestService(t *testing.T) (*Service, *fgamocks.OpenFGAServiceClient) {
	client := fgamocks.NewOpenFGAServiceClient(t)

	// Use real roles retriever with test data
	testRolesFile := filepath.Join("testdata", "roles.yaml")
	rolesRetriever, err := roles.NewFileBasedRolesRetriever(testRolesFile)
	if err != nil {
		t.Fatalf("Failed to create roles retriever: %v", err)
	}

	service := NewWithRolesRetriever(client, nil, rolesRetriever)
	return service, client
}

func TestNew(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)

	// Create config with testdata roles file
	cfg := createTestConfig(t)
	service, err := New(client, cfg)

	// Should succeed with test config
	assert.NoError(t, err)
	assert.NotNil(t, service)
}

func TestService_ListUsers_Success(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	roleFilters := []string{"owner", "member"}
	storeID := "store-123"

	// Mock ListStores call for StoreHelper
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: "test-org",
			},
		},
	}
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(listStoresResponse, nil)

	// Mock ListUsers calls for each role
	ownerUsersResponse := &openfgav1.ListUsersResponse{
		Users: []*openfgav1.User{
			{
				User: &openfgav1.User_Object{
					Object: &openfgav1.Object{
						Type: "user",
						Id:   "user1",
					},
				},
			},
			{
				User: &openfgav1.User_Object{
					Object: &openfgav1.Object{
						Type: "user",
						Id:   "user2",
					},
				},
			},
		},
	}

	memberUsersResponse := &openfgav1.ListUsersResponse{
		Users: []*openfgav1.User{
			{
				User: &openfgav1.User_Object{
					Object: &openfgav1.Object{
						Type: "user",
						Id:   "user2",
					},
				},
			},
			{
				User: &openfgav1.User_Object{
					Object: &openfgav1.Object{
						Type: "user",
						Id:   "user3",
					},
				},
			},
		},
	}

	// Expect calls for owner role
	client.EXPECT().ListUsers(mock.Anything, mock.MatchedBy(func(req *openfgav1.ListUsersRequest) bool {
		return req.StoreId == storeID &&
			req.Object.Type == "role" &&
			req.Object.Id == "core_platform-mesh_io_account/cluster-123/test-account/owner" &&
			req.Relation == "assignee"
	})).Return(ownerUsersResponse, nil)

	// Expect calls for member role
	client.EXPECT().ListUsers(mock.Anything, mock.MatchedBy(func(req *openfgav1.ListUsersRequest) bool {
		return req.StoreId == storeID &&
			req.Object.Type == "role" &&
			req.Object.Id == "core_platform-mesh_io_account/cluster-123/test-account/member" &&
			req.Relation == "assignee"
	})).Return(memberUsersResponse, nil)

	result, err := service.ListUsers(ctx, rCtx, roleFilters)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify the results - convert to map for easier testing
	resultMap := make(map[string][]string)
	for _, userRoles := range result {
		var roleNames []string
		for _, role := range userRoles.Roles {
			roleNames = append(roleNames, role.ID)
		}
		sort.Strings(roleNames) // Sort for deterministic comparison
		resultMap[userRoles.User.Email] = roleNames
	}

	expected := map[string][]string{
		"user1": []string{"owner"},
		"user2": []string{"member", "owner"}, // sorted alphabetically
		"user3": []string{"member"},
	}

	assert.Equal(t, expected, resultMap)
}

func TestService_ListUsers_NoKCPContext(t *testing.T) {
	service, _ := createTestService(t)

	ctx := context.Background()

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	result, err := service.ListUsers(ctx, rCtx, []string{"owner"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "kcp user context")
}

func TestService_ListUsers_StoreHelperError(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	// Mock ListStores call to fail
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(nil, errors.New("store not found"))

	result, err := service.ListUsers(ctx, rCtx, []string{"owner"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get store ID")
}

func TestService_ListUsers_ListUsersError(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	storeID := "store-123"

	// Mock ListStores call for StoreHelper
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: "test-org",
			},
		},
	}
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(listStoresResponse, nil)

	// Mock ListUsers call to fail
	client.EXPECT().ListUsers(mock.Anything, mock.Anything).Return(nil, errors.New("list users failed"))

	result, err := service.ListUsers(ctx, rCtx, []string{"owner"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list users")
}

func TestService_ListUsers_EmptyRoleFilters(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	storeID := "store-123"

	// Mock ListStores call for StoreHelper
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: "test-org",
			},
		},
	}
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(listStoresResponse, nil)

	// Mock ListUsers calls for default roles (owner and member)
	ownerUsersResponse := &openfgav1.ListUsersResponse{
		Users: []*openfgav1.User{
			{
				User: &openfgav1.User_Object{
					Object: &openfgav1.Object{
						Type: "user",
						Id:   "user1",
					},
				},
			},
		},
	}

	memberUsersResponse := &openfgav1.ListUsersResponse{
		Users: []*openfgav1.User{
			{
				User: &openfgav1.User_Object{
					Object: &openfgav1.Object{
						Type: "user",
						Id:   "user2",
					},
				},
			},
		},
	}

	client.EXPECT().ListUsers(mock.Anything, mock.MatchedBy(func(req *openfgav1.ListUsersRequest) bool {
		return req.Object.Id == "core_platform-mesh_io_account/cluster-123/test-account/owner"
	})).Return(ownerUsersResponse, nil)

	client.EXPECT().ListUsers(mock.Anything, mock.MatchedBy(func(req *openfgav1.ListUsersRequest) bool {
		return req.Object.Id == "core_platform-mesh_io_account/cluster-123/test-account/member"
	})).Return(memberUsersResponse, nil)

	result, err := service.ListUsers(ctx, rCtx, []string{}) // Empty role filters

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify the results - convert to map for easier testing
	resultMap := make(map[string][]string)
	for _, userRoles := range result {
		var roleNames []string
		for _, role := range userRoles.Roles {
			roleNames = append(roleNames, role.ID)
		}
		sort.Strings(roleNames) // Sort for deterministic comparison
		resultMap[userRoles.User.Email] = roleNames
	}

	expected := map[string][]string{
		"user1": []string{"owner"},
		"user2": []string{"member"},
	}

	assert.Equal(t, expected, resultMap)
}

func TestApplyRoleFilter_WithFilters(t *testing.T) {
	// Create a logger for testing
	log, _ := logger.New(logger.DefaultConfig())

	service, _ := createTestService(t)
	roleFilters := []string{"owner"}
	result, err := service.applyRoleFilter("core_platform-mesh_io_account", roleFilters, log)

	assert.NoError(t, err)
	expected := []string{"owner"}
	assert.Equal(t, expected, result)
}

func TestApplyRoleFilter_WithMultipleFilters(t *testing.T) {
	// Create a logger for testing
	log, _ := logger.New(logger.DefaultConfig())

	service, _ := createTestService(t)
	roleFilters := []string{"owner", "member", "invalid-role"}
	result, err := service.applyRoleFilter("core_platform-mesh_io_account", roleFilters, log)

	assert.NoError(t, err)
	expected := []string{"owner", "member"}
	assert.Equal(t, expected, result)
}

func TestApplyRoleFilter_EmptyFilters(t *testing.T) {
	// Create a logger for testing
	log, _ := logger.New(logger.DefaultConfig())

	service, _ := createTestService(t)
	roleFilters := []string{}
	result, err := service.applyRoleFilter("core_platform-mesh_io_account", roleFilters, log)

	assert.NoError(t, err)
	expected := []string{"owner", "member"}
	assert.Equal(t, expected, result)
}

func TestApplyRoleFilter_NilFilters(t *testing.T) {
	// Create a logger for testing
	log, _ := logger.New(logger.DefaultConfig())

	service, _ := createTestService(t)
	result, err := service.applyRoleFilter("core_platform-mesh_io_account", nil, log)

	assert.NoError(t, err)
	expected := []string{"owner", "member"}
	assert.Equal(t, expected, result)
}

func TestContainsString_Found(t *testing.T) {
	arr := []string{"owner", "member", "viewer"}
	result := containsString(arr, "member")
	assert.True(t, result)
}

func TestContainsString_NotFound(t *testing.T) {
	arr := []string{"owner", "member", "viewer"}
	result := containsString(arr, "admin")
	assert.False(t, result)
}

func TestContainsString_EmptyArray(t *testing.T) {
	arr := []string{}
	result := containsString(arr, "owner")
	assert.False(t, result)
}

func TestService_GetRoles_Success(t *testing.T) {
	service, _ := createTestService(t)

	ctx := context.Background()
	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	result, err := service.GetRoles(ctx, rCtx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)

	// Check the roles are properly mapped
	roleMap := make(map[string]*graph.Role)
	for _, role := range result {
		roleMap[role.ID] = role
	}

	ownerRole, exists := roleMap["owner"]
	assert.True(t, exists)
	assert.Equal(t, "owner", ownerRole.ID)
	assert.Equal(t, "Owner", ownerRole.DisplayName)
	assert.Equal(t, "Full access to all resources within the account.", ownerRole.Description)

	memberRole, exists := roleMap["member"]
	assert.True(t, exists)
	assert.Equal(t, "member", memberRole.ID)
	assert.Equal(t, "Member", memberRole.DisplayName)
	assert.Equal(t, "Limited access to resources within the account. Can view and interact with resources but cannot administrate them.", memberRole.Description)
}

func TestService_GetRoles_EmptyGroupResource(t *testing.T) {
	service, _ := createTestService(t)

	ctx := context.Background()
	rCtx := graph.ResourceContext{
		GroupResource: "nonexistent_resource", // Not defined in testdata/roles.yaml
		Resource: &graph.Resource{
			Name:      "test-resource",
			Namespace: stringPtr("default"),
		},
	}

	result, err := service.GetRoles(ctx, rCtx)

	assert.NoError(t, err)
	assert.Empty(t, result) // Should return empty for non-existent group resource
}

func TestService_GetRoles_RolesRetrieverError(t *testing.T) {
	// Create a service with an invalid roles file to trigger error
	invalidRolesFile := filepath.Join("testdata", "nonexistent.yaml")
	rolesRetriever, err := roles.NewFileBasedRolesRetriever(invalidRolesFile)
	assert.Error(t, err) // This should fail
	assert.Nil(t, rolesRetriever)

	// We can't easily test a roles retriever error with real implementation
	// since errors happen at construction time, not at GetRoleDefinitions time.
	// This test validates that invalid files are caught during setup.
}

func TestService_AssignRolesToUsers_Success(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	changes := []*graph.UserRoleChange{
		{
			UserID: "user1@example.com",
			Roles:  []string{"owner", "member"},
		},
		{
			UserID: "user2@example.com",
			Roles:  []string{"member"},
		},
	}

	storeID := "store-123"

	// Mock ListStores call for StoreHelper
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: "test-org",
			},
		},
	}
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(listStoresResponse, nil)

	// Mock Write calls for each role assignment
	expectedWrites := []string{
		"user:user1@example.com",
		"role:core_platform-mesh_io_account/cluster-123/test-account/owner",
		"user:user1@example.com",
		"role:core_platform-mesh_io_account/cluster-123/test-account/member",
		"user:user2@example.com",
		"role:core_platform-mesh_io_account/cluster-123/test-account/member",
	}

	writeCallCount := 0
	client.EXPECT().Write(mock.Anything, mock.MatchedBy(func(req *openfgav1.WriteRequest) bool {
		if req.StoreId != storeID {
			return false
		}
		if len(req.Writes.TupleKeys) != 1 {
			return false
		}
		tuple := req.Writes.TupleKeys[0]
		if tuple.Relation != "assignee" {
			return false
		}

		// Check user and object match expected values
		expectedUser := expectedWrites[writeCallCount*2]
		expectedObject := expectedWrites[writeCallCount*2+1]
		writeCallCount++

		return tuple.User == expectedUser && tuple.Object == expectedObject
	})).Return(&openfgav1.WriteResponse{}, nil).Times(3)

	result, err := service.AssignRolesToUsers(ctx, rCtx, changes)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 3, result.AssignedCount)
	assert.Empty(t, result.Errors)
}

func TestService_AssignRolesToUsers_InvalidRole(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	changes := []*graph.UserRoleChange{
		{
			UserID: "user1@example.com",
			Roles:  []string{"owner", "admin"}, // admin is not in defaultRoles
		},
	}

	storeID := "store-123"

	// Mock ListStores call for StoreHelper
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: "test-org",
			},
		},
	}
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(listStoresResponse, nil)

	// Mock Write call for owner role only (admin should be rejected)
	client.EXPECT().Write(mock.Anything, mock.MatchedBy(func(req *openfgav1.WriteRequest) bool {
		return req.StoreId == storeID &&
			len(req.Writes.TupleKeys) == 1 &&
			req.Writes.TupleKeys[0].User == "user:user1@example.com" &&
			req.Writes.TupleKeys[0].Object == "role:core_platform-mesh_io_account/cluster-123/test-account/owner" &&
			req.Writes.TupleKeys[0].Relation == "assignee"
	})).Return(&openfgav1.WriteResponse{}, nil).Once()

	result, err := service.AssignRolesToUsers(ctx, rCtx, changes)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, 1, result.AssignedCount)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "role 'admin' is not allowed")
}

func TestService_AssignRolesToUsers_DuplicateTuple(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	changes := []*graph.UserRoleChange{
		{
			UserID: "user1@example.com",
			Roles:  []string{"owner", "member"},
		},
	}

	storeID := "store-123"

	// Mock ListStores call for StoreHelper
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: "test-org",
			},
		},
	}
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(listStoresResponse, nil)

	// Create a duplicate write error similar to the one in the issue
	duplicateError := status.Error(codes.Code(openfgav1.ErrorCode_write_failed_due_to_invalid_input),
		"cannot write a tuple which already exists: user: 'user:user1@example.com', relation: 'assignee', object: 'role:core_platform-mesh_io_account/cluster-123/test-account/owner': tuple to be written already existed")

	// Mock Write call for owner role - returns duplicate error
	client.EXPECT().Write(mock.Anything, mock.MatchedBy(func(req *openfgav1.WriteRequest) bool {
		return req.StoreId == storeID &&
			len(req.Writes.TupleKeys) == 1 &&
			req.Writes.TupleKeys[0].User == "user:user1@example.com" &&
			req.Writes.TupleKeys[0].Object == "role:core_platform-mesh_io_account/cluster-123/test-account/owner" &&
			req.Writes.TupleKeys[0].Relation == "assignee"
	})).Return(nil, duplicateError).Once()

	// Mock Write call for member role - succeeds
	client.EXPECT().Write(mock.Anything, mock.MatchedBy(func(req *openfgav1.WriteRequest) bool {
		return req.StoreId == storeID &&
			len(req.Writes.TupleKeys) == 1 &&
			req.Writes.TupleKeys[0].User == "user:user1@example.com" &&
			req.Writes.TupleKeys[0].Object == "role:core_platform-mesh_io_account/cluster-123/test-account/member" &&
			req.Writes.TupleKeys[0].Relation == "assignee"
	})).Return(&openfgav1.WriteResponse{}, nil).Once()

	result, err := service.AssignRolesToUsers(ctx, rCtx, changes)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)           // Should still be successful
	assert.Equal(t, 2, result.AssignedCount) // Both roles should count as assigned
	assert.Empty(t, result.Errors)           // No errors should be reported
}

func TestService_AssignRolesToUsers_NoKCPContext(t *testing.T) {
	service, _ := createTestService(t)

	ctx := context.Background()

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	changes := []*graph.UserRoleChange{
		{
			UserID: "user1@example.com",
			Roles:  []string{"owner"},
		},
	}

	result, err := service.AssignRolesToUsers(ctx, rCtx, changes)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get kcp user context")
}

func TestService_RemoveRole_Success(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	input := graph.RemoveRoleInput{
		UserID: "user1@example.com",
		Role:   "owner",
	}

	storeID := "store-123"

	// Mock ListStores call for StoreHelper
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: "test-org",
			},
		},
	}
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(listStoresResponse, nil)

	// Mock Read call to check if tuple exists - returns tuple (role is assigned)
	readResponse := &openfgav1.ReadResponse{
		Tuples: []*openfgav1.Tuple{
			{
				Key: &openfgav1.TupleKey{
					User:     "user:user1@example.com",
					Relation: "assignee",
					Object:   "role:core_platform-mesh_io_account/cluster-123/test-account/owner",
				},
			},
		},
	}
	client.EXPECT().Read(mock.Anything, mock.MatchedBy(func(req *openfgav1.ReadRequest) bool {
		return req.StoreId == storeID &&
			req.TupleKey.User == "user:user1@example.com" &&
			req.TupleKey.Object == "role:core_platform-mesh_io_account/cluster-123/test-account/owner" &&
			req.TupleKey.Relation == "assignee"
	})).Return(readResponse, nil).Once()

	// Mock Write call for deletion
	client.EXPECT().Write(mock.Anything, mock.MatchedBy(func(req *openfgav1.WriteRequest) bool {
		return req.StoreId == storeID &&
			req.Deletes != nil &&
			len(req.Deletes.TupleKeys) == 1 &&
			req.Deletes.TupleKeys[0].User == "user:user1@example.com" &&
			req.Deletes.TupleKeys[0].Object == "role:core_platform-mesh_io_account/cluster-123/test-account/owner" &&
			req.Deletes.TupleKeys[0].Relation == "assignee"
	})).Return(&openfgav1.WriteResponse{}, nil).Once()

	result, err := service.RemoveRole(ctx, rCtx, input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.True(t, result.WasAssigned)
	assert.Nil(t, result.Error)
}

func TestService_RemoveRole_RoleNotAssigned(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	input := graph.RemoveRoleInput{
		UserID: "user1@example.com",
		Role:   "member",
	}

	storeID := "store-123"

	// Mock ListStores call for StoreHelper
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: "test-org",
			},
		},
	}
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(listStoresResponse, nil)

	// Mock Read call to check if tuple exists - returns empty (role is not assigned)
	readResponse := &openfgav1.ReadResponse{
		Tuples: []*openfgav1.Tuple{}, // Empty - no tuples found
	}
	client.EXPECT().Read(mock.Anything, mock.MatchedBy(func(req *openfgav1.ReadRequest) bool {
		return req.StoreId == storeID &&
			req.TupleKey.User == "user:user1@example.com" &&
			req.TupleKey.Object == "role:core_platform-mesh_io_account/cluster-123/test-account/member" &&
			req.TupleKey.Relation == "assignee"
	})).Return(readResponse, nil).Once()

	// No Write call should be made since the role wasn't assigned

	result, err := service.RemoveRole(ctx, rCtx, input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)      // Still successful since idempotent
	assert.False(t, result.WasAssigned) // But role wasn't assigned
	assert.Nil(t, result.Error)
}

func TestService_RemoveRole_InvalidRole(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	input := graph.RemoveRoleInput{
		UserID: "user1@example.com",
		Role:   "admin", // invalid role
	}

	storeID := "store-123"

	// Mock ListStores call for StoreHelper
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: "test-org",
			},
		},
	}
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(listStoresResponse, nil)

	// No Read or Write calls should be made for invalid roles

	result, err := service.RemoveRole(ctx, rCtx, input)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.False(t, result.WasAssigned)
	assert.NotNil(t, result.Error)
	assert.Contains(t, *result.Error, "role 'admin' is not allowed")
}

func TestService_RemoveRole_ReadError(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	input := graph.RemoveRoleInput{
		UserID: "user1@example.com",
		Role:   "owner",
	}

	storeID := "store-123"

	// Mock ListStores call for StoreHelper
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: "test-org",
			},
		},
	}
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(listStoresResponse, nil)

	// Mock Read call to return an error
	readError := errors.New("FGA read error")
	client.EXPECT().Read(mock.Anything, mock.Anything).Return(nil, readError).Once()

	result, err := service.RemoveRole(ctx, rCtx, input)

	assert.NoError(t, err) // Service method doesn't return error for business logic failures
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.False(t, result.WasAssigned)
	assert.NotNil(t, result.Error)
	assert.Contains(t, *result.Error, "failed to check role assignment")
}

func TestService_RemoveRole_WriteError(t *testing.T) {
	service, client := createTestService(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	input := graph.RemoveRoleInput{
		UserID: "user1@example.com",
		Role:   "owner",
	}

	storeID := "store-123"

	// Mock ListStores call for StoreHelper
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: "test-org",
			},
		},
	}
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(listStoresResponse, nil)

	// Mock Read call to check if tuple exists - returns tuple (role is assigned)
	readResponse := &openfgav1.ReadResponse{
		Tuples: []*openfgav1.Tuple{
			{
				Key: &openfgav1.TupleKey{
					User:     "user:user1@example.com",
					Relation: "assignee",
					Object:   "role:core_platform-mesh_io_account/cluster-123/test-account/owner",
				},
			},
		},
	}
	client.EXPECT().Read(mock.Anything, mock.Anything).Return(readResponse, nil).Once()

	// Mock Write call for deletion - returns error
	writeError := errors.New("FGA write error")
	client.EXPECT().Write(mock.Anything, mock.Anything).Return(nil, writeError).Once()

	result, err := service.RemoveRole(ctx, rCtx, input)

	assert.NoError(t, err) // Service method doesn't return error for business logic failures
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.True(t, result.WasAssigned) // Role was assigned but removal failed
	assert.NotNil(t, result.Error)
	assert.Contains(t, *result.Error, "failed to remove role")
}

func TestService_RemoveRole_NoKCPContext(t *testing.T) {
	service, _ := createTestService(t)

	ctx := context.Background()

	rCtx := graph.ResourceContext{
		GroupResource: "core_platform-mesh_io_account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: stringPtr("default"),
		},
		AccountPath: "test-account",
	}

	input := graph.RemoveRoleInput{
		UserID: "user1@example.com",
		Role:   "owner",
	}

	result, err := service.RemoveRole(ctx, rCtx, input)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get kcp user context")
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
