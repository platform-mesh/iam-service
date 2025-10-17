package keycloak

import (
	"context"
	"fmt"
	"os"

	"github.com/coreos/go-oidc"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"

	"github.com/platform-mesh/iam-service/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/graph"
	keycloakClient "github.com/platform-mesh/iam-service/pkg/keycloak/client"
	"github.com/platform-mesh/iam-service/pkg/middleware/kcp"
)

type Service struct {
	cfg            *config.ServiceConfig
	keycloakClient KeycloakClientInterface
}

func (s *Service) UserByMail(ctx context.Context, userID string) (*graph.User, error) {
	kctx, err := kcp.GetKcpUserContext(ctx)
	if err != nil {
		return nil, err
	}

	// Configure search parameters
	briefRepresentation := true
	maxResults := int32(1)
	exact := true
	params := &keycloakClient.GetUsersParams{
		Email:               &userID,
		Max:                 &maxResults,
		BriefRepresentation: &briefRepresentation,
		Exact:               &exact,
	}

	// Query users using the generated client
	resp, err := s.keycloakClient.GetUsersWithResponse(ctx, kctx.IDMTenant, params)
	if err != nil { // coverage-ignore
		log.Err(err).Msg("Failed to query users")
		return nil, err
	}

	if resp.StatusCode() != 200 {
		log.Error().Int("status_code", resp.StatusCode()).Msg("Non-200 response from Keycloak")
		return nil, fmt.Errorf("keycloak API returned status %d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, nil
	}

	users := *resp.JSON200
	if len(users) == 0 {
		return nil, nil
	}
	if len(users) != 1 {
		log.Info().Str("email", userID).Int("count", len(users)).Msg("unexpected user count")
		return nil, fmt.Errorf("expected 1 user, got %d", len(users))
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

func New(ctx context.Context, cfg *config.ServiceConfig) (*Service, error) {
	issuer := fmt.Sprintf("%s/realms/master", cfg.Keycloak.BaseURL)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	oauthC := oauth2.Config{ClientID: cfg.Keycloak.ClientID, Endpoint: provider.Endpoint()}
	pwd, err := os.ReadFile(cfg.Keycloak.PasswordFile)
	if err != nil {
		return nil, err
	}

	token, err := oauthC.PasswordCredentialsToken(ctx, cfg.Keycloak.User, string(pwd))
	if err != nil {
		return nil, err
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

	return &Service{
		cfg:            cfg,
		keycloakClient: kcClient,
	}, nil
}
