package fga

import (
	"context"
	"errors"
	"testing"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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

	// Verify the results
	expected := UserIDToRoles{
		"user1": []string{"owner"},
		"user2": []string{"owner", "member"},
		"user3": []string{"member"},
	}
	assert.Equal(t, expected, result)
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

	expected := UserIDToRoles{
		"user1": []string{"owner"},
		"user2": []string{"member"},
	}
	assert.Equal(t, expected, result)
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

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
