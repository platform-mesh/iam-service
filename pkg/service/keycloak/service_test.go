package keycloak

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/platform-mesh/iam-service/pkg/graph"
	keycloakClient "github.com/platform-mesh/iam-service/pkg/keycloak/client"
	"github.com/platform-mesh/iam-service/pkg/middleware/kcp"
	"github.com/platform-mesh/iam-service/pkg/service/keycloak/mocks"
)

func TestUserByMail_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	// Create expected user response
	userID := "test-user-id"
	userEmail := "test@example.com"
	users := []keycloakClient.UserRepresentation{
		{
			Id:    &userID,
			Email: &userEmail,
		},
	}

	response := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200:      &users,
	}

	// Setup mock expectations
	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.MatchedBy(func(params *keycloakClient.GetUsersParams) bool {
			return params != nil &&
				params.Email != nil && *params.Email == userEmail &&
				params.Max != nil && *params.Max == int32(1) &&
				params.BriefRepresentation != nil && *params.BriefRepresentation == true &&
				params.Exact != nil && *params.Exact == true
		}),
		mock.Anything,
	).Return(response, nil)

	// Execute
	result, err := service.UserByMail(ctx, userEmail)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, userEmail, result.Email)
}

func TestUserByMail_UserNotFound(t *testing.T) {
	// Setup
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	// Create empty user response
	users := []keycloakClient.UserRepresentation{}
	response := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200:      &users,
	}

	// Setup mock expectations
	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return(response, nil)

	// Execute
	result, err := service.UserByMail(ctx, "nonexistent@example.com")

	// Assert
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestUserByMail_MultipleUsersFound(t *testing.T) {
	// Setup
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	// Create response with multiple users
	userID1 := "test-user-id-1"
	userID2 := "test-user-id-2"
	userEmail := "test@example.com"
	users := []keycloakClient.UserRepresentation{
		{
			Id:    &userID1,
			Email: &userEmail,
		},
		{
			Id:    &userID2,
			Email: &userEmail,
		},
	}

	response := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200:      &users,
	}

	// Setup mock expectations
	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return(response, nil)

	// Execute
	result, err := service.UserByMail(ctx, userEmail)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "expected 1 user, got 2")
}

func TestUserByMail_APIError(t *testing.T) {
	// Setup
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	// Setup mock expectations - API call fails
	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return((*keycloakClient.GetUsersResponse)(nil), errors.New("API error"))

	// Execute
	result, err := service.UserByMail(ctx, "test@example.com")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "API error")
}

func TestUserByMail_Non200Response(t *testing.T) {
	// Setup
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	// Create response with non-200 status
	response := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 500},
		JSON200:      nil,
	}

	// Setup mock expectations
	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return(response, nil)

	// Execute
	result, err := service.UserByMail(ctx, "test@example.com")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "keycloak API returned status 500")
}

func TestUserByMail_NoKcpContext(t *testing.T) {
	// Setup
	ctx := context.Background()

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	// Execute
	result, err := service.UserByMail(ctx, "test@example.com")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "kcp user context")
}

func TestUserByMail_NullJSON200Response(t *testing.T) {
	// Setup
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	// Create response with 200 but null JSON200
	response := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200:      nil,
	}

	// Setup mock expectations
	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return(response, nil)

	// Execute
	result, err := service.UserByMail(ctx, "test@example.com")

	// Assert
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestEnrichUserRoles_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	// Create test user roles with partial data (only emails from FGA)
	userRoles := []*graph.UserRoles{
		{
			User: &graph.User{
				Email: "user1@example.com",
			},
		},
		{
			User: &graph.User{
				Email: "user2@example.com",
			},
		},
	}

	// Create expected Keycloak users
	userID1 := "keycloak-user-1"
	userID2 := "keycloak-user-2"
	firstName1 := "John"
	firstName2 := "Jane"
	lastName1 := "Doe"
	lastName2 := "Smith"

	users := []keycloakClient.UserRepresentation{
		{
			Id:        &userID1,
			Email:     stringPtr("user1@example.com"),
			FirstName: &firstName1,
			LastName:  &lastName1,
		},
		{
			Id:        &userID2,
			Email:     stringPtr("user2@example.com"),
			FirstName: &firstName2,
			LastName:  &lastName2,
		},
	}

	// Setup mock expectations for individual user calls (parallel processing)
	// EnrichUserRoles calls GetUsersByEmails which makes individual calls for each user
	// We need to handle both calls - they can happen in any order due to parallel processing
	// Use a more flexible approach that matches specific email parameters
	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.MatchedBy(func(params *keycloakClient.GetUsersParams) bool {
			return params != nil && params.Email != nil && *params.Email == "user1@example.com"
		}),
		mock.Anything,
	).Return(&keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200:      &[]keycloakClient.UserRepresentation{users[0]},
	}, nil)

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.MatchedBy(func(params *keycloakClient.GetUsersParams) bool {
			return params != nil && params.Email != nil && *params.Email == "user2@example.com"
		}),
		mock.Anything,
	).Return(&keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200:      &[]keycloakClient.UserRepresentation{users[1]},
	}, nil)

	// Execute
	err := service.EnrichUserRoles(ctx, userRoles)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, userID1, userRoles[0].User.UserID)
	assert.Equal(t, "user1@example.com", userRoles[0].User.Email)
	assert.Equal(t, firstName1, *userRoles[0].User.FirstName)
	assert.Equal(t, lastName1, *userRoles[0].User.LastName)

	assert.Equal(t, userID2, userRoles[1].User.UserID)
	assert.Equal(t, "user2@example.com", userRoles[1].User.Email)
	assert.Equal(t, firstName2, *userRoles[1].User.FirstName)
	assert.Equal(t, lastName2, *userRoles[1].User.LastName)
}

func TestEnrichUserRoles_EmptySlice(t *testing.T) {
	// Setup
	service := &Service{}

	// Execute with empty slice
	err := service.EnrichUserRoles(context.Background(), []*graph.UserRoles{})

	// Assert
	assert.NoError(t, err)

	// Execute with nil slice
	err = service.EnrichUserRoles(context.Background(), nil)

	// Assert
	assert.NoError(t, err)
}

func TestEnrichUserRoles_NoValidEmails(t *testing.T) {
	// Setup
	service := &Service{}

	userRoles := []*graph.UserRoles{
		{
			User: &graph.User{
				Email: "", // Empty email
			},
		},
		{
			User: nil, // Nil user
		},
		{
			User: &graph.User{
				// No email field
			},
		},
	}

	// Execute
	err := service.EnrichUserRoles(context.Background(), userRoles)

	// Assert
	assert.NoError(t, err)
}

func TestEnrichUserRoles_GetUsersByEmailsError(t *testing.T) {
	// Setup
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	userRoles := []*graph.UserRoles{
		{
			User: &graph.User{
				Email: "test@example.com",
			},
		},
	}

	// Setup mock to return an error
	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return((*keycloakClient.GetUsersResponse)(nil), errors.New("Keycloak API error"))

	// Execute
	err := service.EnrichUserRoles(ctx, userRoles)

	// Assert - EnrichUserRoles doesn't fail when individual fetches fail,
	// it logs warnings but continues processing
	assert.NoError(t, err)

	// User roles should remain unchanged since the API call failed
	assert.Equal(t, "", userRoles[0].User.UserID)
	assert.Nil(t, userRoles[0].User.FirstName)
	assert.Nil(t, userRoles[0].User.LastName)
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
