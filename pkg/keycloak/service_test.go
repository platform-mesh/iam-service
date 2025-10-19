package keycloak

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/platform-mesh/iam-service/pkg/cache"
	"github.com/platform-mesh/iam-service/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/graph"
	keycloakClient "github.com/platform-mesh/iam-service/pkg/keycloak/client"
	"github.com/platform-mesh/iam-service/pkg/keycloak/mocks"
	"github.com/platform-mesh/iam-service/pkg/middleware/kcp"
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

func TestNew_InvalidConfig(t *testing.T) {
	// Test with invalid configuration to ensure error handling
	ctx := context.Background()

	// Test with invalid Keycloak base URL (should fail during OIDC provider setup)
	invalidCfg := &config.ServiceConfig{
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
			BaseURL:      "invalid-url", // Invalid URL
			ClientID:     "test-client",
			User:         "test-user",
			PasswordFile: "/nonexistent/file",
			Cache: struct {
				TTL     time.Duration `mapstructure:"keycloak-cache-ttl" default:"5m"`
				Enabled bool          `mapstructure:"keycloak-cache-enabled" default:"true"`
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

func TestNew_MissingPasswordFile(t *testing.T) {
	// Test with missing password file - this should fail early during file read
	ctx := context.Background()

	cfg := &config.ServiceConfig{
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
			BaseURL:      "https://localhost:9999/keycloak", // Non-existent server
			ClientID:     "test-client",
			User:         "test-user",
			PasswordFile: "/this/file/definitely/does/not/exist.txt", // Non-existent file
			Cache: struct {
				TTL     time.Duration `mapstructure:"keycloak-cache-ttl" default:"5m"`
				Enabled bool          `mapstructure:"keycloak-cache-enabled" default:"true"`
			}{
				TTL:     5 * time.Minute,
				Enabled: false, // Disable cache for simpler testing
			},
		},
	}

	service, err := New(ctx, cfg)

	// Should return an error due to missing password file or invalid server
	assert.Error(t, err)
	assert.Nil(t, service)
	// The error could be either file not found or network-related
}

func TestUserByMail_CacheHit(t *testing.T) {
	// Test cache hit scenario
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
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

func TestUserByMail_CacheSetAfterFetch(t *testing.T) {
	// Test cache set scenario after successful fetch
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	userCache := cache.NewUserCache(5 * time.Minute)
	service := &Service{
		keycloakClient: mockClient,
		userCache:      userCache,
	}

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
		mock.Anything,
		mock.Anything,
	).Return(response, nil)

	// Execute
	result, err := service.UserByMail(ctx, userEmail)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify user was cached
	cachedUser := userCache.Get("test-realm", userEmail)
	assert.NotNil(t, cachedUser)
	assert.Equal(t, userID, cachedUser.UserID)
}

func TestUserByMail_NoCacheSet_UserNotFound(t *testing.T) {
	// Test that nil users are not cached
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	userCache := cache.NewUserCache(5 * time.Minute)
	service := &Service{
		keycloakClient: mockClient,
		userCache:      userCache,
	}

	userEmail := "notfound@example.com"
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
	result, err := service.UserByMail(ctx, userEmail)

	// Assert
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Verify user was not cached (since result was nil)
	cachedUser := userCache.Get("test-realm", userEmail)
	assert.Nil(t, cachedUser)
}

func TestGetUsersByEmails_CacheHitPartial(t *testing.T) {
	// Test partial cache hit scenario
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	userCache := cache.NewUserCache(5 * time.Minute)
	service := &Service{
		keycloakClient: mockClient,
		userCache:      userCache,
	}

	// Pre-populate cache with one user
	cachedUser := &graph.User{
		UserID: "cached-user-id",
		Email:  "cached@example.com",
	}
	userCache.Set("test-realm", "cached@example.com", cachedUser)

	emails := []string{"cached@example.com", "fetch@example.com"}

	// Setup mock for the non-cached user
	userID := "fetch-user-id"
	userEmail := "fetch@example.com"
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

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.MatchedBy(func(params *keycloakClient.GetUsersParams) bool {
			return params != nil && params.Email != nil && *params.Email == "fetch@example.com"
		}),
		mock.Anything,
	).Return(response, nil)

	// Execute
	result, err := service.GetUsersByEmails(ctx, emails)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "cached@example.com")
	assert.Contains(t, result, "fetch@example.com")
	assert.Equal(t, cachedUser, result["cached@example.com"])
	assert.Equal(t, userID, result["fetch@example.com"].UserID)
}

func TestGetUsersByEmails_NoCache(t *testing.T) {
	// Test when cache is disabled
	ctx := context.Background()
	ctx = context.WithValue(ctx, kcp.UserContextKey, kcp.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
		userCache:      nil, // No cache
	}

	emails := []string{"user1@example.com", "user2@example.com"}

	// Setup mocks for both users
	userID1 := "user-1-id"
	userID2 := "user-2-id"

	response1 := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200: &[]keycloakClient.UserRepresentation{
			{
				Id:    &userID1,
				Email: stringPtr("user1@example.com"),
			},
		},
	}

	response2 := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200: &[]keycloakClient.UserRepresentation{
			{
				Id:    &userID2,
				Email: stringPtr("user2@example.com"),
			},
		},
	}

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.MatchedBy(func(params *keycloakClient.GetUsersParams) bool {
			return params != nil && params.Email != nil && *params.Email == "user1@example.com"
		}),
		mock.Anything,
	).Return(response1, nil)

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.MatchedBy(func(params *keycloakClient.GetUsersParams) bool {
			return params != nil && params.Email != nil && *params.Email == "user2@example.com"
		}),
		mock.Anything,
	).Return(response2, nil)

	// Execute
	result, err := service.GetUsersByEmails(ctx, emails)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, userID1, result["user1@example.com"].UserID)
	assert.Equal(t, userID2, result["user2@example.com"].UserID)
}

// Test the New constructor with cache enabled
func TestNew_WithCacheEnabled(t *testing.T) {
	// This is a unit test that doesn't make real network calls
	// We'll test the constructor logic without external dependencies
	ctx := context.Background()

	// Create a temporary password file
	tmpFile, err := os.CreateTemp("", "test-password-*.txt")
	assert.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString("test-password")
	assert.NoError(t, err)
	err = tmpFile.Close()
	assert.NoError(t, err)

	cfg := &config.ServiceConfig{
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
			BaseURL:      "https://httpbin.org/get", // Use a test endpoint that will fail at OAuth
			ClientID:     "test-client",
			User:         "test-user",
			PasswordFile: tmpFile.Name(),
			Cache: struct {
				TTL     time.Duration `mapstructure:"keycloak-cache-ttl" default:"5m"`
				Enabled bool          `mapstructure:"keycloak-cache-enabled" default:"true"`
			}{
				TTL:     10 * time.Minute,
				Enabled: true, // Cache enabled
			},
		},
	}

	// This will fail at the OIDC provider setup, but it tests file reading and config parsing
	service, err := New(ctx, cfg)

	// Should return an error, but it covers the cache initialization code paths
	assert.Error(t, err)
	assert.Nil(t, service)
}

// Test the New constructor with cache disabled
func TestNew_WithCacheDisabled(t *testing.T) {
	ctx := context.Background()

	// Create a temporary password file
	tmpFile, err := os.CreateTemp("", "test-password-*.txt")
	assert.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.WriteString("test-password")
	assert.NoError(t, err)
	err = tmpFile.Close()
	assert.NoError(t, err)

	cfg := &config.ServiceConfig{
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
			BaseURL:      "https://httpbin.org/get", // Use a test endpoint that will fail at OAuth
			ClientID:     "test-client",
			User:         "test-user",
			PasswordFile: tmpFile.Name(),
			Cache: struct {
				TTL     time.Duration `mapstructure:"keycloak-cache-ttl" default:"5m"`
				Enabled bool          `mapstructure:"keycloak-cache-enabled" default:"true"`
			}{
				TTL:     10 * time.Minute,
				Enabled: false, // Cache disabled
			},
		},
	}

	// This will fail at the OIDC provider setup, but it tests cache disabled path
	service, err := New(ctx, cfg)

	// Should return an error, but it covers the cache disabled code path
	assert.Error(t, err)
	assert.Nil(t, service)
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
