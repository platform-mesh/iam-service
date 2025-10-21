package keycloak

import (
	"context"
	"fmt"
	"os"

	"github.com/coreos/go-oidc"
	"github.com/platform-mesh/golang-commons/errors"
	"github.com/platform-mesh/golang-commons/logger"
	"golang.org/x/oauth2"

	"github.com/platform-mesh/iam-service/pkg/cache"
	"github.com/platform-mesh/iam-service/pkg/config"
	appcontext "github.com/platform-mesh/iam-service/pkg/context"
	"github.com/platform-mesh/iam-service/pkg/graph"
	keycloakClient "github.com/platform-mesh/iam-service/pkg/keycloak/client"
)

type Service struct {
	cfg            *config.ServiceConfig
	keycloakClient KeycloakClientInterface
	userCache      *cache.UserCache
}

func (s *Service) UserByMail(ctx context.Context, userID string) (*graph.User, error) {
	kctx, err := appcontext.GetKCPContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get KCP user context")
	}

	realm := kctx.IDMTenant

	// Try cache first if enabled
	if s.userCache != nil {
		if cachedUser := s.userCache.Get(realm, userID); cachedUser != nil {
			return cachedUser, nil
		}
	}

	// Cache miss - fetch from Keycloak
	user, err := s.fetchUserFromKeycloak(ctx, realm, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch user from Keycloak for email %s", userID)
	}

	// Store in cache if user found and cache enabled
	if user != nil && s.userCache != nil {
		s.userCache.Set(realm, userID, user)
	}

	return user, nil
}

// fetchUserFromKeycloak fetches a single user from Keycloak by email
func (s *Service) fetchUserFromKeycloak(ctx context.Context, realm, email string) (*graph.User, error) {
	log := logger.LoadLoggerFromContext(ctx)
	// Configure search parameters
	briefRepresentation := true
	maxResults := int32(1)
	exact := true
	params := &keycloakClient.GetUsersParams{
		Email:               &email,
		Max:                 &maxResults,
		BriefRepresentation: &briefRepresentation,
		Exact:               &exact,
	}

	// Query users using the generated client
	resp, err := s.keycloakClient.GetUsersWithResponse(ctx, realm, params)
	if err != nil { // coverage-ignore
		log.Err(err).Str("email", email).Msg("Failed to query user")
		return nil, errors.Wrap(err, "failed to query Keycloak API for user %s in realm %s", email, realm)
	}

	if resp.StatusCode() != 200 {
		log.Error().Int("status_code", resp.StatusCode()).Str("email", email).Msg("Non-200 response from Keycloak")
		return nil, errors.New("keycloak API returned status %d for user %s", resp.StatusCode(), email)
	}

	if resp.JSON200 == nil {
		return nil, nil
	}

	users := *resp.JSON200
	if len(users) == 0 {
		return nil, nil
	}
	if len(users) != 1 {
		log.Info().Str("email", email).Int("count", len(users)).Msg("unexpected user count")
		return nil, errors.New("expected 1 user, got %d for email %s", len(users), email)
	}

	user := users[0]
	result := &graph.User{
		UserID:    *user.Id,
		Email:     *user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}

	return result, nil
}

func (s *Service) GetUsersByEmails(ctx context.Context, emails []string) (map[string]*graph.User, error) {
	log := logger.LoadLoggerFromContext(ctx)
	if len(emails) == 0 {
		return map[string]*graph.User{}, nil
	}

	kctx, err := appcontext.GetKCPContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get KCP user context")
	}

	realm := kctx.IDMTenant
	result := make(map[string]*graph.User)

	var missingEmails []string

	// Check cache first if enabled
	if s.userCache != nil {
		cached, missing := s.userCache.GetMany(realm, emails)

		// Add cached users to result
		for email, user := range cached {
			result[email] = user
		}

		missingEmails = missing

		log.Debug().
			Int("requested", len(emails)).
			Int("cached_hits", len(cached)).
			Int("cache_misses", len(missing)).
			Msg("Cache lookup completed")
	} else {
		// No cache - need to fetch all
		missingEmails = emails
	}

	// Fetch missing users in parallel
	if len(missingEmails) > 0 {
		fetchedUsers, err := s.fetchUsersInParallel(ctx, realm, missingEmails)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch users in parallel for realm %s", realm)
		}

		// Add fetched users to result and cache
		for email, user := range fetchedUsers {
			result[email] = user

			// Store in cache if enabled
			if s.userCache != nil {
				s.userCache.Set(realm, email, user)
			}
		}
	}

	log.Info().
		Int("requested_emails", len(emails)).
		Int("returned_users", len(result)).
		Int("api_calls", len(missingEmails)).
		Msg("Completed user lookup with cache")

	return result, nil
}

// fetchUsersInParallel fetches multiple users from Keycloak in parallel
func (s *Service) fetchUsersInParallel(ctx context.Context, realm string, emails []string) (map[string]*graph.User, error) {
	log := logger.LoadLoggerFromContext(ctx)
	type userResult struct {
		email string
		user  *graph.User
		err   error
	}

	resultChan := make(chan userResult, len(emails))

	// Launch goroutines for each email
	for _, email := range emails {
		go func(email string) {
			user, err := s.fetchUserFromKeycloak(ctx, realm, email)
			resultChan <- userResult{
				email: email,
				user:  user,
				err:   err,
			}
		}(email)
	}

	// Collect results
	userMap := make(map[string]*graph.User)
	var errors []string

	for i := 0; i < len(emails); i++ {
		result := <-resultChan

		if result.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.email, result.err))
			continue
		}

		if result.user != nil {
			userMap[result.email] = result.user
		}
	}

	// Log any errors but don't fail the entire operation
	if len(errors) > 0 {
		log.Warn().Strs("errors", errors).Msg("Some user fetches failed")
	}

	return userMap, nil
}

// EnrichUserRoles enriches user roles with complete user information from Keycloak
// Updates the UserRoles slice in-place with FirstName, LastName, and UserID from Keycloak
func (s *Service) EnrichUserRoles(ctx context.Context, userRoles []*graph.UserRoles) error {
	if len(userRoles) == 0 {
		return nil
	}

	// Extract unique email addresses from user roles
	emailSet := make(map[string]bool)
	var emails []string

	for _, userRole := range userRoles {
		if userRole.User != nil && userRole.User.Email != "" {
			if !emailSet[userRole.User.Email] {
				emailSet[userRole.User.Email] = true
				emails = append(emails, userRole.User.Email)
			}
		}
	}

	if len(emails) == 0 {
		return nil
	}

	// Batch call to get all users at once
	userMap, err := s.GetUsersByEmails(ctx, emails)
	if err != nil {
		return errors.Wrap(err, "failed to get users by emails for enrichment")
	}

	// Update user roles with Keycloak data using the lookup map
	for _, userRole := range userRoles {
		if userRole.User != nil && userRole.User.Email != "" {
			if keycloakUser, exists := userMap[userRole.User.Email]; exists {
				// Update the user with complete information from Keycloak
				userRole.User.UserID = keycloakUser.UserID
				userRole.User.FirstName = keycloakUser.FirstName
				userRole.User.LastName = keycloakUser.LastName
				// Email is already set from OpenFGA
			}
		}
	}

	return nil
}

func New(ctx context.Context, cfg *config.ServiceConfig) (*Service, error) {
	log := logger.LoadLoggerFromContext(ctx)
	issuer := fmt.Sprintf("%s/realms/master", cfg.Keycloak.BaseURL)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create OIDC provider for issuer %s", issuer)
	}

	oauthC := oauth2.Config{ClientID: cfg.Keycloak.ClientID, Endpoint: provider.Endpoint()}
	pwd, err := os.ReadFile(cfg.Keycloak.PasswordFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read Keycloak password file %s", cfg.Keycloak.PasswordFile)
	}

	token, err := oauthC.PasswordCredentialsToken(ctx, cfg.Keycloak.User, string(pwd))
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain password credentials token for user %s", cfg.Keycloak.User)
	}

	// Create authenticated HTTP client
	httpClient := oauthC.Client(ctx, token)

	// Create Keycloak client with the authenticated HTTP client
	kcClient, err := keycloakClient.NewClientWithResponses(
		cfg.Keycloak.BaseURL,
		keycloakClient.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Keycloak client: %w", err)
	}

	// Initialize cache if enabled
	var userCache *cache.UserCache
	if cfg.Keycloak.Cache.Enabled {
		userCache = cache.NewUserCache(cfg.Keycloak.Cache.TTL)
		log.Info().Dur("ttl", cfg.Keycloak.Cache.TTL).Msg("Keycloak user cache enabled")
	} else {
		log.Info().Msg("Keycloak user cache disabled")
	}

	return &Service{
		cfg:            cfg,
		keycloakClient: kcClient,
		userCache:      userCache,
	}, nil
}
