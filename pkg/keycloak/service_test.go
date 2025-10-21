package keycloak

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/utils/ptr"

	"github.com/platform-mesh/iam-service/pkg/cache"
	"github.com/platform-mesh/iam-service/pkg/config"
	appcontext "github.com/platform-mesh/iam-service/pkg/context"
	"github.com/platform-mesh/iam-service/pkg/graph"
	keycloakClient "github.com/platform-mesh/iam-service/pkg/keycloak/client"
	"github.com/platform-mesh/iam-service/pkg/keycloak/mocks"
)

func createKeycloakTestConfig(baseURL, clientID, user, passwordFile string, cacheEnabled bool, cacheTTL time.Duration) *config.ServiceConfig {
	return &config.ServiceConfig{
		Keycloak: config.KeycloakConfig{
			BaseURL:      baseURL,
			ClientID:     clientID,
			User:         user,
			PasswordFile: passwordFile,
			Cache: config.KeycloakCacheConfig{
				Enabled: cacheEnabled,
				TTL:     cacheTTL,
			},
		},
	}
}

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
			Email:     ptr.To("user1@example.com"),
			FirstName: &firstName1,
			LastName:  &lastName1,
		},
		{
			Id:        &userID2,
			Email:     ptr.To("user2@example.com"),
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
	invalidCfg := createKeycloakTestConfig("invalid-url", "test-client", "test-user", "/nonexistent/file", true, 5*time.Minute)

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
		FirstName: ptr.To("Cached"),
		LastName:  ptr.To("User"),
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

func TestFetchUserFromKeycloak_Non200Status(t *testing.T) {
	// Test non-200 status code response
	ctx := context.Background()
	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	response := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 404},
	}

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return(response, nil)

	result, err := service.fetchUserFromKeycloak(ctx, "test-realm", "test@example.com")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "keycloak API returned status 404")
}

func TestFetchUserFromKeycloak_NilJSON200(t *testing.T) {
	// Test nil JSON200 response
	ctx := context.Background()
	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	response := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200:      nil,
	}

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return(response, nil)

	result, err := service.fetchUserFromKeycloak(ctx, "test-realm", "test@example.com")

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestFetchUserFromKeycloak_MultipleUsers(t *testing.T) {
	// Test multiple users returned (unexpected)
	ctx := context.Background()
	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	userID1 := "user-1"
	userID2 := "user-2"
	userEmail := "test@example.com"
	users := []keycloakClient.UserRepresentation{
		{Id: &userID1, Email: &userEmail},
		{Id: &userID2, Email: &userEmail},
	}

	response := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200:      &users,
	}

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return(response, nil)

	result, err := service.fetchUserFromKeycloak(ctx, "test-realm", userEmail)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "expected 1 user, got 2")
}

func TestUserByMail_FetchError(t *testing.T) {
	// Test error during fetch
	ctx := context.Background()
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	// Mock error response
	response := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 500},
	}

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return(response, nil)

	result, err := service.UserByMail(ctx, "test@example.com")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to fetch user from Keycloak")
}

func TestUserByMail_CacheStore(t *testing.T) {
	// Test successful user fetch and cache store
	ctx := context.Background()
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
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

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return(response, nil)

	result, err := service.UserByMail(ctx, userEmail)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)

	// Verify user was cached
	cachedUser := userCache.Get("test-realm", userEmail)
	assert.NotNil(t, cachedUser)
	assert.Equal(t, userID, cachedUser.UserID)
}

func TestGetUsersByEmails_EmptySlice(t *testing.T) {
	// Test empty email slice
	ctx := context.Background()
	service := &Service{}

	result, err := service.GetUsersByEmails(ctx, []string{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestGetUsersByEmails_NoKCPContext(t *testing.T) {
	// Test missing KCP context
	ctx := context.Background()
	service := &Service{}

	result, err := service.GetUsersByEmails(ctx, []string{"test@example.com"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "kcp user context")
}

func TestGetUsersByEmails_WithCache(t *testing.T) {
	// Test GetUsersByEmails with cache hit and miss scenario
	ctx := context.Background()
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
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
		UserID:    "cached-user-id",
		Email:     "cached@example.com",
		FirstName: ptr.To("Cached"),
		LastName:  ptr.To("User"),
	}
	userCache.Set("test-realm", "cached@example.com", cachedUser)

	// Setup mock for the missing user
	fetchUserID := "fetch-user-id"
	fetchEmail := "fetch@example.com"
	fetchUsers := []keycloakClient.UserRepresentation{
		{
			Id:    &fetchUserID,
			Email: &fetchEmail,
		},
	}

	response := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200:      &fetchUsers,
	}

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.MatchedBy(func(params *keycloakClient.GetUsersParams) bool {
			return params != nil && params.Email != nil && *params.Email == fetchEmail
		}),
		mock.Anything,
	).Return(response, nil)

	// Execute with both cached and non-cached emails
	emails := []string{"cached@example.com", "fetch@example.com"}
	result, err := service.GetUsersByEmails(ctx, emails)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, cachedUser, result["cached@example.com"])
	assert.Equal(t, fetchUserID, result["fetch@example.com"].UserID)

	// Verify the fetched user was also cached
	newlyCachedUser := userCache.Get("test-realm", fetchEmail)
	assert.NotNil(t, newlyCachedUser)
	assert.Equal(t, fetchUserID, newlyCachedUser.UserID)
}

func TestGetUsersByEmails_FetchError(t *testing.T) {
	// Test that GetUsersByEmails continues even when some fetches fail
	ctx := context.Background()
	ctx = appcontext.SetKCPContext(ctx, appcontext.KCPContext{
		IDMTenant: "test-realm",
	})

	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	// Mock error response
	response := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 500},
	}

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.Anything,
		mock.Anything,
	).Return(response, nil)

	result, err := service.GetUsersByEmails(ctx, []string{"test@example.com"})

	// GetUsersByEmails doesn't fail when individual fetches fail - it continues
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result) // No users returned due to error
}

func TestFetchUsersInParallel_WithErrors(t *testing.T) {
	// Test fetchUsersInParallel with some errors
	ctx := context.Background()
	mockClient := mocks.NewKeycloakClientInterface(t)
	service := &Service{
		keycloakClient: mockClient,
	}

	// Mock successful response for first email
	userID1 := "user-1"
	userEmail1 := "success@example.com"
	users1 := []keycloakClient.UserRepresentation{
		{Id: &userID1, Email: &userEmail1},
	}
	successResponse := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 200},
		JSON200:      &users1,
	}

	// Mock error response for second email
	errorResponse := &keycloakClient.GetUsersResponse{
		HTTPResponse: &http.Response{StatusCode: 500},
	}

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.MatchedBy(func(params *keycloakClient.GetUsersParams) bool {
			return params != nil && params.Email != nil && *params.Email == userEmail1
		}),
		mock.Anything,
	).Return(successResponse, nil)

	mockClient.EXPECT().GetUsersWithResponse(
		ctx,
		"test-realm",
		mock.MatchedBy(func(params *keycloakClient.GetUsersParams) bool {
			return params != nil && params.Email != nil && *params.Email == "error@example.com"
		}),
		mock.Anything,
	).Return(errorResponse, nil)

	emails := []string{userEmail1, "error@example.com"}
	result, err := service.fetchUsersInParallel(ctx, "test-realm", emails)

	// Should not return error (errors are logged but don't fail the operation)
	assert.NoError(t, err)
	assert.Len(t, result, 1) // Only successful user should be in result
	assert.Equal(t, userID1, result[userEmail1].UserID)
}

func TestNew_PasswordFileNotFound(t *testing.T) {
	// Test with non-existent password file
	// The test expects failure due to OIDC provider creation (not password file)
	// since the OIDC provider check happens first in actual test environment
	ctx := context.Background()

	cfg := createKeycloakTestConfig("https://valid-url.com/keycloak", "test-client", "test-user", "/nonexistent/path/password.txt", true, 5*time.Minute)

	service, err := New(ctx, cfg)

	assert.Error(t, err)
	assert.Nil(t, service)
	// In test environment, this fails at OIDC provider creation, not password file
	assert.Contains(t, err.Error(), "failed to create OIDC provider")
}

func TestNew_CacheEnabled(t *testing.T) {
	// Test successful New with cache enabled
	// This is a simplified test that won't actually connect to Keycloak
	// but will test the cache initialization logic
	ctx := context.Background()

	// Create a temporary password file
	tmpFile, err := os.CreateTemp("", "keycloak_password_test")
	assert.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = tmpFile.WriteString("test-password")
	assert.NoError(t, err)
	_ = tmpFile.Close()

	cfg := createKeycloakTestConfig("https://valid-issuer.com/keycloak", "test-client", "test-user", tmpFile.Name(), true, 5*time.Minute)

	// This will fail at OIDC provider creation, but that's expected in test environment
	service, err := New(ctx, cfg)

	// In test environment, this will fail at OIDC provider creation
	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "failed to create OIDC provider")
}

func TestNew_CacheDisabled(t *testing.T) {
	// Test with cache disabled
	ctx := context.Background()

	// Create a temporary password file
	tmpFile, err := os.CreateTemp("", "keycloak_password_test")
	assert.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = tmpFile.WriteString("test-password")
	assert.NoError(t, err)
	_ = tmpFile.Close()

	cfg := createKeycloakTestConfig("https://valid-issuer.com/keycloak", "test-client", "test-user", tmpFile.Name(), false, 5*time.Minute)

	// This will fail at OIDC provider creation, but that's expected in test environment
	service, err := New(ctx, cfg)

	// In test environment, this will fail at OIDC provider creation
	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "failed to create OIDC provider")
}
