package keycloak

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/platform-mesh/iam-service/pkg/cache"
	"github.com/platform-mesh/iam-service/pkg/config"
	appcontext "github.com/platform-mesh/iam-service/pkg/context"
	"github.com/platform-mesh/iam-service/pkg/graph"
	keycloakClient "github.com/platform-mesh/iam-service/pkg/keycloak/client"
	"github.com/platform-mesh/iam-service/pkg/keycloak/mocks"
)

func TestUserByMail_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
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
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
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

func TestEnrichUserRoles_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
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

	// Setup mock expectations for individual user calls
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

func TestNew_InvalidConfig(t *testing.T) {
	// Test with invalid configuration to ensure error handling
	ctx := context.Background()

	// Test with invalid Keycloak base URL
	invalidCfg := &config.ServiceConfig{
		Keycloak: struct {
			BaseURL      string `mapstructure:"keycloak-base-url" default:"https://portal.dev.local:8443/keycloak"`
			ClientID     string `mapstructure:"keycloak-client-id" default:"admin-cli"`
			User         string `mapstructure:"keycloak-user" default:"keycloak-admin"`
			PasswordFile string `mapstructure:"keycloak-password-file" default:".secret/keycloak/password"`
			Cache        struct {
				Enabled bool          `mapstructure:"keycloak-cache-enabled" default:"true"`
				TTL     time.Duration `mapstructure:"keycloak-user-cache-ttl" default:"5m"`
			} `mapstructure:",squash"`
		}{
			BaseURL:      "invalid-url", // Invalid URL
			ClientID:     "test-client",
			User:         "test-user",
			PasswordFile: "/nonexistent/file",
			Cache: struct {
				Enabled bool          `mapstructure:"keycloak-cache-enabled" default:"true"`
				TTL     time.Duration `mapstructure:"keycloak-user-cache-ttl" default:"5m"`
			}{
				TTL:     5 * time.Minute,
				Enabled: true,
			},
		},
	}

	service, err := New(ctx, invalidCfg)

	// Should return an error due to invalid configuration
	assert.Error(t, err)
	assert.Nil(t, service)
}

func TestUserByMail_CacheHit(t *testing.T) {
	// Test cache hit scenario
	ctx := context.Background()
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
		IDMTenant: "test-realm",
	})

	// Create a service with cache enabled
	userCache := cache.NewUserCache(5 * time.Minute)
	service := &Service{
		userCache: userCache,
	}

	userEmail := "cached@example.com"
	expectedUser := &graph.User{
		UserID:    "cached-user-id",
		Email:     userEmail,
		FirstName: stringPtr("Cached"),
		LastName:  stringPtr("User"),
	}

	// Pre-populate cache
	userCache.Set("test-realm", userEmail, expectedUser)

	// Execute - should get from cache without calling keycloak client
	result, err := service.UserByMail(ctx, userEmail)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedUser, result)
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
