package keycloak

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/coreos/go-oidc"
	"github.com/platform-mesh/iam-service/internal/config"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/service/idm"
	"golang.org/x/oauth2"
)

var _ idm.Service = (*Service)(nil)

type Service struct {
	cfg config.ServiceConfig
}

func (s *Service) UserById(ctx context.Context, userID string) (*graph.User, error) {
	// Get Realm from context

}
func New(ctx context.Context, cfg config.ServiceConfig) (*Service, error) {
	//issuer := fmt.Sprintf("%s/realms/master", cfg.Keycloak.BaseURL)
	//provider, err := oidc.NewProvider(ctx, issuer)
	//if err != nil {
	//	return nil, err
	//}
	//
	//oauthC := oauth2.Config{ClientID: cfg.Keycloak.ClientID, Endpoint: provider.Endpoint()}
	//pwd, err := os.ReadFile(cfg.Keycloak.PasswordFile)
	//if err != nil {
	//	return nil, err
	//}
	//
	//token, err := oauthC.PasswordCredentialsToken(ctx, cfg.Keycloak.User, string(pwd))
	//if err != nil {
	//	return nil, err
	//}

	return &Service{
		cfg: cfg,
	}, nil
}
