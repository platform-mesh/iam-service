package resolver

import (
	"context"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	pmcontext "github.com/platform-mesh/golang-commons/context"

	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/resolver/api"
	"github.com/platform-mesh/iam-service/pkg/resolver/errors"
	"github.com/platform-mesh/iam-service/pkg/service/keycloak"
)

var _ api.ResolverService = (*Service)(nil)

type Service struct {
	fgaClient       openfgav1.OpenFGAServiceClient
	keycloakService *keycloak.Service
}

func (s *Service) Me(ctx context.Context) (*graph.User, error) {
	// Get Current User
	webToken, err := pmcontext.GetWebTokenFromContext(ctx)
	if err != nil {
		return nil, errors.InternalError
	}
	return s.keycloakService.UserByMail(ctx, webToken.Mail)
}

func (s *Service) User(ctx context.Context, userID string) (*graph.User, error) {
	return s.keycloakService.UserByMail(ctx, userID)
}

func NewResolverService(fgaClient openfgav1.OpenFGAServiceClient, service *keycloak.Service) *Service {
	return &Service{
		fgaClient:       fgaClient,
		keycloakService: service,
	}
}
