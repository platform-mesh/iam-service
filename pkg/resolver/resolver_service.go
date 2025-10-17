package resolver

import (
	"context"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	pmcontext "github.com/platform-mesh/golang-commons/context"

	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/resolver/api"
	"github.com/platform-mesh/iam-service/pkg/resolver/errors"
	"github.com/platform-mesh/iam-service/pkg/service/fga"
	"github.com/platform-mesh/iam-service/pkg/service/keycloak"
)

var _ api.ResolverService = (*Service)(nil)

type Service struct {
	fgaService      *fga.Service
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

func (s *Service) Users(ctx context.Context, context graph.ResourceContext, roleFilters []string, sortBy *graph.SortByInput) (*graph.UserConnection, error) {
	// Retrieve users with roles from fga
	s.fgaService.UsersForResource(ctx, context, roleFilters)
	return nil, nil

	// Fill users from keycloak with metadata
}
func NewResolverService(fgaClient openfgav1.OpenFGAServiceClient, service *keycloak.Service) *Service {
	return &Service{
		fgaService:      fga.New(fgaClient),
		keycloakService: service,
	}
}
