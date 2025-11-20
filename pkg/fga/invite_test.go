package fga

import (
	"context"
	"testing"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	"github.com/platform-mesh/golang-commons/logger"
	securityv1alpha1 "github.com/platform-mesh/security-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appcontext "github.com/platform-mesh/iam-service/pkg/context"
	fgamocks "github.com/platform-mesh/iam-service/pkg/fga/mocks"
	"github.com/platform-mesh/iam-service/pkg/graph"
)

func TestService_AssignRolesToUsers_WithInvites_UserExists(t *testing.T) {
	service, client := createTestService(t)

	// Create mocks for workspace client factory and IDM user checker
	mockWsFactory := fgamocks.NewClientFactory(t)
	mockIDMChecker := fgamocks.NewIDMUserChecker(t)

	// Set the mocks on the service
	service.wsClientFactory = mockWsFactory
	service.idmChecker = mockIDMChecker

	ctx := context.Background()
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
		IDMTenant:        "test-tenant",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		Group: "core.platform-mesh.io",
		Kind:  "Account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: ptr.To("default"),
		},
		AccountPath: "root:org:test-account",
	}

	ai := &accountsv1alpha1.AccountInfo{
		ObjectMeta: metav1.ObjectMeta{Name: "account"},
		Spec: accountsv1alpha1.AccountInfoSpec{
			Account: accountsv1alpha1.AccountLocation{
				GeneratedClusterId: "cluster-123",
			},
		},
	}

	invites := []*graph.InviteInput{
		{
			Email: "newuser@example.com",
			Roles: []string{"member"},
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

	// Mock IDM check - user exists, so no invite should be created
	mockIDMChecker.EXPECT().UserByMail(mock.Anything, "newuser@example.com").Return(&graph.User{
		UserID: "newuser@example.com",
		Email:  "newuser@example.com",
	}, nil).Once()

	// Mock Write calls for role assignment (2 writes per role: assignee + role tuple)
	client.EXPECT().Write(mock.Anything, mock.MatchedBy(func(req *openfgav1.WriteRequest) bool {
		return req.StoreId == storeID &&
			len(req.Writes.TupleKeys) == 1
	})).Return(&openfgav1.WriteResponse{}, nil).Times(2)

	// Set cluster ID in context
	ctx = appcontext.SetClusterId(ctx, ai.Spec.Account.GeneratedClusterId)

	result, err := service.AssignRolesToUsers(ctx, rCtx, nil, invites)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 2, result.AssignedCount)
	assert.Empty(t, result.Errors)
}

func TestService_AssignRolesToUsers_WithInvites_UserDoesNotExist(t *testing.T) {
	service, client := createTestService(t)

	// Create mocks for workspace client factory and IDM user checker
	mockWsFactory := fgamocks.NewClientFactory(t)
	mockIDMChecker := fgamocks.NewIDMUserChecker(t)

	// Create scheme with security operator API
	scheme := runtime.NewScheme()
	err := securityv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	mockWsClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	// Set the mocks on the service
	service.wsClientFactory = mockWsFactory
	service.idmChecker = mockIDMChecker

	ctx := context.Background()
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
		IDMTenant:        "test-tenant",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		Group: "core.platform-mesh.io",
		Kind:  "Account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: ptr.To("default"),
		},
		AccountPath: "root:org:test-account",
	}

	ai := &accountsv1alpha1.AccountInfo{
		ObjectMeta: metav1.ObjectMeta{Name: "account"},
		Spec: accountsv1alpha1.AccountInfoSpec{
			Account: accountsv1alpha1.AccountLocation{
				GeneratedClusterId: "cluster-123",
			},
		},
	}

	invites := []*graph.InviteInput{
		{
			Email: "newuser@example.com",
			Roles: []string{"member"},
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

	// Mock IDM check - user doesn't exist
	mockIDMChecker.EXPECT().UserByMail(mock.Anything, "newuser@example.com").Return(
		nil, assert.AnError).Once()

	// Mock workspace client creation
	mockWsFactory.EXPECT().New(mock.Anything, "root:org:test-account").Return(mockWsClient, nil).Once()

	// Mock Write calls for role assignment (2 writes per role: assignee + role tuple)
	client.EXPECT().Write(mock.Anything, mock.MatchedBy(func(req *openfgav1.WriteRequest) bool {
		return req.StoreId == storeID &&
			len(req.Writes.TupleKeys) == 1
	})).Return(&openfgav1.WriteResponse{}, nil).Times(2)

	// Set cluster ID in context
	ctx = appcontext.SetClusterId(ctx, ai.Spec.Account.GeneratedClusterId)

	result, err := service.AssignRolesToUsers(ctx, rCtx, nil, invites)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 2, result.AssignedCount)
	assert.Empty(t, result.Errors)
}

func TestService_AssignRolesToUsers_WithInvites_InvalidRole(t *testing.T) {
	service, client := createTestService(t)

	// Create mocks for workspace client factory and IDM user checker
	mockWsFactory := fgamocks.NewClientFactory(t)
	mockIDMChecker := fgamocks.NewIDMUserChecker(t)

	// Set the mocks on the service
	service.wsClientFactory = mockWsFactory
	service.idmChecker = mockIDMChecker

	ctx := context.Background()
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
		IDMTenant:        "test-tenant",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		Group: "core.platform-mesh.io",
		Kind:  "Account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: ptr.To("default"),
		},
		AccountPath: "root:org:test-account",
	}

	ai := &accountsv1alpha1.AccountInfo{
		ObjectMeta: metav1.ObjectMeta{Name: "account"},
		Spec: accountsv1alpha1.AccountInfoSpec{
			Account: accountsv1alpha1.AccountLocation{
				GeneratedClusterId: "cluster-123",
			},
		},
	}

	invites := []*graph.InviteInput{
		{
			Email: "newuser@example.com",
			Roles: []string{"admin"}, // Invalid role for Account
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

	// Mock IDM check - user exists
	mockIDMChecker.EXPECT().UserByMail(mock.Anything, "newuser@example.com").Return(&graph.User{
		UserID: "newuser@example.com",
		Email:  "newuser@example.com",
	}, nil).Once()

	// Set cluster ID in context
	ctx = appcontext.SetClusterId(ctx, ai.Spec.Account.GeneratedClusterId)

	result, err := service.AssignRolesToUsers(ctx, rCtx, nil, invites)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, 0, result.AssignedCount)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0], "admin")
	assert.Contains(t, result.Errors[0], "not allowed")
}

func TestService_AssignRolesToUsers_WithBothChangesAndInvites(t *testing.T) {
	service, client := createTestService(t)

	// Create mocks for workspace client factory and IDM user checker
	mockWsFactory := fgamocks.NewClientFactory(t)
	mockIDMChecker := fgamocks.NewIDMUserChecker(t)

	// Set the mocks on the service
	service.wsClientFactory = mockWsFactory
	service.idmChecker = mockIDMChecker

	ctx := context.Background()
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
		IDMTenant:        "test-tenant",
		OrganizationName: "test-org",
	})
	log, _ := logger.New(logger.DefaultConfig())
	ctx = logger.SetLoggerInContext(ctx, log)

	rCtx := graph.ResourceContext{
		Group: "core.platform-mesh.io",
		Kind:  "Account",
		Resource: &graph.Resource{
			Name:      "test-account",
			Namespace: ptr.To("default"),
		},
		AccountPath: "root:org:test-account",
	}

	ai := &accountsv1alpha1.AccountInfo{
		ObjectMeta: metav1.ObjectMeta{Name: "account"},
		Spec: accountsv1alpha1.AccountInfoSpec{
			Account: accountsv1alpha1.AccountLocation{
				GeneratedClusterId: "cluster-123",
			},
		},
	}

	changes := []*graph.UserRoleChange{
		{
			UserID: "existinguser@example.com",
			Roles:  []string{"owner"},
		},
	}

	invites := []*graph.InviteInput{
		{
			Email: "newuser@example.com",
			Roles: []string{"member"},
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

	// Mock IDM check for invite
	mockIDMChecker.EXPECT().UserByMail(mock.Anything, "newuser@example.com").Return(&graph.User{
		UserID: "newuser@example.com",
		Email:  "newuser@example.com",
	}, nil).Once()

	// Mock Write calls for role assignments
	// 2 writes for owner role (existing user) + 2 writes for member role (invited user) = 4 total
	client.EXPECT().Write(mock.Anything, mock.MatchedBy(func(req *openfgav1.WriteRequest) bool {
		return req.StoreId == storeID &&
			len(req.Writes.TupleKeys) == 1
	})).Return(&openfgav1.WriteResponse{}, nil).Times(4)

	// Set cluster ID in context
	ctx = appcontext.SetClusterId(ctx, ai.Spec.Account.GeneratedClusterId)

	result, err := service.AssignRolesToUsers(ctx, rCtx, changes, invites)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 4, result.AssignedCount) // 2 for existing user + 2 for invited user
	assert.Empty(t, result.Errors)
}
