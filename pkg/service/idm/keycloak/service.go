package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/coreos/go-oidc"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"

	"github.com/platform-mesh/iam-service/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/middleware/kcp"
	"github.com/platform-mesh/iam-service/pkg/service/idm"
)

var _ idm.Service = (*Service)(nil)

type keycloakUser struct {
	ID              string   `json:"id,omitempty"`
	Email           string   `json:"email,omitempty"`
	RequiredActions []string `json:"requiredActions,omitempty"`
	Enabled         bool     `json:"enabled,omitempty"`
}

type Service struct {
	cfg    *config.ServiceConfig
	client *http.Client
}

func (s *Service) UserByMail(ctx context.Context, userID string) (*graph.User, error) {
	kctx, err := kcp.GetKcpUserContext(ctx)
	if err != nil {
		return nil, err
	}

	v := url.Values{
		"email":               {userID},
		"max":                 {"1"},
		"briefRepresentation": {"true"},
	}
	res, err := s.client.Get(fmt.Sprintf("%s/admin/realms/%s/users?%s", s.cfg.Keycloak.BaseURL, kctx.IDMTenant, v.Encode()))
	if err != nil { // coverage-ignore
		log.Err(err).Msg("Failed to query users")
		return nil, err
	}
	defer res.Body.Close() //nolint:errcheck
	if res.StatusCode != http.StatusOK {
		return nil, err
	}

	var users []keycloakUser
	if err = json.NewDecoder(res.Body).Decode(&users); err != nil { // coverage-ignore
		return nil, err
	}

	if len(users) == 0 {
		return nil, nil
	}
	if len(users) != 1 {
		log.Info().Str("email", userID).Msg("unexpected user count")
		return nil, err
	}

	return &graph.User{
		UserID: users[0].ID,
		Email:  users[0].Email,
	}, nil

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

	return &Service{
		cfg:    cfg,
		client: oauthC.Client(ctx, token),
	}, nil
}
