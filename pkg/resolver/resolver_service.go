package resolver

import (
	"context"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	pmcontext "github.com/platform-mesh/golang-commons/context"

	"github.com/platform-mesh/iam-service/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/fga"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/keycloak"
	"github.com/platform-mesh/iam-service/pkg/pager"
	"github.com/platform-mesh/iam-service/pkg/resolver/api"
	"github.com/platform-mesh/iam-service/pkg/resolver/errors"
	"github.com/platform-mesh/iam-service/pkg/sorter"
)

var _ api.ResolverService = (*Service)(nil)

type Service struct {
	fgaService      *fga.Service
	keycloakService *keycloak.Service
	userSorter      sorter.UserSorter
	pager           pager.Pager
}

func (s *Service) Me(ctx context.Context) (*graph.User, error) {
	// Get Current User
	webToken, err := pmcontext.GetWebTokenFromContext(ctx)
	if err != nil {
		return nil, errors.ErrInternal
	}
	return s.keycloakService.UserByMail(ctx, webToken.Mail)
}

func (s *Service) User(ctx context.Context, userID string) (*graph.User, error) {
	return s.keycloakService.UserByMail(ctx, userID)
}

func (s *Service) Users(ctx context.Context, context graph.ResourceContext, roleFilters []string, sortBy *graph.SortByInput, page *graph.PageInput) (*graph.UserConnection, error) {
	// Retrieve users with roles from fga
	allUserRoles, err := s.fgaService.ListUsers(ctx, context, roleFilters)
	if err != nil {
		return nil, err
	}

	// Fill users from openfga with metadata from keycloak
	err = s.keycloakService.EnrichUserRoles(ctx, allUserRoles)
	if err != nil {
		return nil, err
	}

	// Apply sorting
	s.userSorter.SortUserRoles(allUserRoles, sortBy)

	totalCount := len(allUserRoles)

	// Apply pagination
	paginatedUserRoles, pageInfo := s.pager.PaginateUserRoles(allUserRoles, page, totalCount)

	return &graph.UserConnection{Users: paginatedUserRoles, PageInfo: pageInfo}, nil
}

func (s *Service) AssignRolesToUsers(ctx context.Context, context graph.ResourceContext, changes []*graph.UserRoleChange) (*graph.RoleAssignmentResult, error) {
	return s.fgaService.AssignRolesToUsers(ctx, context, changes)
}

func (s *Service) RemoveRole(ctx context.Context, context graph.ResourceContext, input graph.RemoveRoleInput) (*graph.RoleRemovalResult, error) {
	return s.fgaService.RemoveRole(ctx, context, input)
}

func (s *Service) Roles(ctx context.Context, context graph.ResourceContext) ([]*graph.Role, error) {
	return s.fgaService.GetRoles(ctx, context)
}

func NewResolverService(fgaClient openfgav1.OpenFGAServiceClient, service *keycloak.Service, cfg *config.ServiceConfig) *Service {
	return &Service{
		fgaService:      fga.NewWithConfig(fgaClient, cfg),
		keycloakService: service,
		userSorter:      sorter.NewUserSorterWithConfig(cfg),
		pager:           pager.NewPager(cfg),
	}
}
