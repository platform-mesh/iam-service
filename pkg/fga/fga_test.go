package fga

import (
	"context"
	"errors"
	"sort"
	"testing"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fgamocks "github.com/platform-mesh/iam-service/pkg/fga/mocks"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/middleware/kcp"
)

func TestNew(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	service := New(client)

	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
	assert.NotNil(t, service.helper)
}

func TestService_ListUsers_Success(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	service := New(client)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})

	rCtx := graph.ResourceContext{
		GroupResource: "apps.v1/deployments",
		Resource: &graph.Resource{
			Name:      "test-deployment",
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
			req.Object.Id == "apps.v1/deployments/cluster-123/test-deployment/owner" &&
			req.Relation == "assignee"
	})).Return(ownerUsersResponse, nil)

	// Expect calls for member role
	client.EXPECT().ListUsers(mock.Anything, mock.MatchedBy(func(req *openfgav1.ListUsersRequest) bool {
		return req.StoreId == storeID &&
			req.Object.Type == "role" &&
			req.Object.Id == "apps.v1/deployments/cluster-123/test-deployment/member" &&
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
			roleNames = append(roleNames, role.TechnicalName)
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
	client := fgamocks.NewOpenFGAServiceClient(t)
	service := New(client)

	ctx := context.Background()

	rCtx := graph.ResourceContext{
		GroupResource: "apps.v1/deployments",
		Resource: &graph.Resource{
			Name:      "test-deployment",
			Namespace: stringPtr("default"),
		},
	}

	result, err := service.ListUsers(ctx, rCtx, []string{"owner"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "kcp user context")
}

func TestService_ListUsers_StoreHelperError(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	service := New(client)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})

	rCtx := graph.ResourceContext{
		GroupResource: "apps.v1/deployments",
		Resource: &graph.Resource{
			Name:      "test-deployment",
			Namespace: stringPtr("default"),
		},
	}

	// Mock ListStores call to fail
	client.EXPECT().ListStores(mock.Anything, mock.Anything).Return(nil, errors.New("store not found"))

	result, err := service.ListUsers(ctx, rCtx, []string{"owner"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get store ID")
}

func TestService_ListUsers_ListUsersError(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	service := New(client)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})

	rCtx := graph.ResourceContext{
		GroupResource: "apps.v1/deployments",
		Resource: &graph.Resource{
			Name:      "test-deployment",
			Namespace: stringPtr("default"),
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

	// Mock ListUsers call to fail
	client.EXPECT().ListUsers(mock.Anything, mock.Anything).Return(nil, errors.New("list users failed"))

	result, err := service.ListUsers(ctx, rCtx, []string{"owner"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list users")
}

func TestService_ListUsers_EmptyRoleFilters(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	service := New(client)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})

	rCtx := graph.ResourceContext{
		GroupResource: "apps.v1/deployments",
		Resource: &graph.Resource{
			Name:      "test-deployment",
			Namespace: stringPtr("default"),
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
		return req.Object.Id == "apps.v1/deployments/cluster-123/test-deployment/owner"
	})).Return(ownerUsersResponse, nil)

	client.EXPECT().ListUsers(mock.Anything, mock.MatchedBy(func(req *openfgav1.ListUsersRequest) bool {
		return req.Object.Id == "apps.v1/deployments/cluster-123/test-deployment/member"
	})).Return(memberUsersResponse, nil)

	result, err := service.ListUsers(ctx, rCtx, []string{}) // Empty role filters

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify the results - convert to map for easier testing
	resultMap := make(map[string][]string)
	for _, userRoles := range result {
		var roleNames []string
		for _, role := range userRoles.Roles {
			roleNames = append(roleNames, role.TechnicalName)
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

	roleFilters := []string{"owner"}
	result := applyRoleFilter(roleFilters, log)

	expected := []string{"owner"}
	assert.Equal(t, expected, result)
}

func TestApplyRoleFilter_WithMultipleFilters(t *testing.T) {
	// Create a logger for testing
	log, _ := logger.New(logger.DefaultConfig())

	roleFilters := []string{"owner", "member", "invalid-role"}
	result := applyRoleFilter(roleFilters, log)

	expected := []string{"owner", "member"}
	assert.Equal(t, expected, result)
}

func TestApplyRoleFilter_EmptyFilters(t *testing.T) {
	// Create a logger for testing
	log, _ := logger.New(logger.DefaultConfig())

	roleFilters := []string{}
	result := applyRoleFilter(roleFilters, log)

	expected := []string{"owner", "member"}
	assert.Equal(t, expected, result)
}

func TestApplyRoleFilter_NilFilters(t *testing.T) {
	// Create a logger for testing
	log, _ := logger.New(logger.DefaultConfig())

	result := applyRoleFilter(nil, log)

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

func TestService_AssignRolesToUsers_Success(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	service := New(client)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		GroupResource: "apps.v1/deployments",
		Resource: &graph.Resource{
			Name:      "test-deployment",
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
		"role:apps.v1/deployments/cluster-123/test-deployment/owner",
		"user:user1@example.com",
		"role:apps.v1/deployments/cluster-123/test-deployment/member",
		"user:user2@example.com",
		"role:apps.v1/deployments/cluster-123/test-deployment/member",
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
	client := fgamocks.NewOpenFGAServiceClient(t)
	service := New(client)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		GroupResource: "apps.v1/deployments",
		Resource: &graph.Resource{
			Name:      "test-deployment",
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
			req.Writes.TupleKeys[0].Object == "role:apps.v1/deployments/cluster-123/test-deployment/owner" &&
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
	client := fgamocks.NewOpenFGAServiceClient(t)
	service := New(client)

	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant:        "test-tenant",
		ClusterId:        "cluster-123",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		GroupResource: "apps.v1/deployments",
		Resource: &graph.Resource{
			Name:      "test-deployment",
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
		"cannot write a tuple which already exists: user: 'user:user1@example.com', relation: 'assignee', object: 'role:apps.v1/deployments/cluster-123/test-deployment/owner': tuple to be written already existed")

	// Mock Write call for owner role - returns duplicate error
	client.EXPECT().Write(mock.Anything, mock.MatchedBy(func(req *openfgav1.WriteRequest) bool {
		return req.StoreId == storeID &&
			len(req.Writes.TupleKeys) == 1 &&
			req.Writes.TupleKeys[0].User == "user:user1@example.com" &&
			req.Writes.TupleKeys[0].Object == "role:apps.v1/deployments/cluster-123/test-deployment/owner" &&
			req.Writes.TupleKeys[0].Relation == "assignee"
	})).Return(nil, duplicateError).Once()

	// Mock Write call for member role - succeeds
	client.EXPECT().Write(mock.Anything, mock.MatchedBy(func(req *openfgav1.WriteRequest) bool {
		return req.StoreId == storeID &&
			len(req.Writes.TupleKeys) == 1 &&
			req.Writes.TupleKeys[0].User == "user:user1@example.com" &&
			req.Writes.TupleKeys[0].Object == "role:apps.v1/deployments/cluster-123/test-deployment/member" &&
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
	client := fgamocks.NewOpenFGAServiceClient(t)
	service := New(client)

	ctx := context.Background()

	rCtx := graph.ResourceContext{
		GroupResource: "apps.v1/deployments",
		Resource: &graph.Resource{
			Name:      "test-deployment",
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

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
