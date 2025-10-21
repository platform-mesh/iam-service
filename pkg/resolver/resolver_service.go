package resolver

import (
	"context"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	pmcontext "github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/errors"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	"github.com/platform-mesh/iam-service/pkg/config"
	appcontext "github.com/platform-mesh/iam-service/pkg/context"
	"github.com/platform-mesh/iam-service/pkg/fga"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/keycloak"
	"github.com/platform-mesh/iam-service/pkg/pager"
	"github.com/platform-mesh/iam-service/pkg/resolver/api"
	serrors "github.com/platform-mesh/iam-service/pkg/resolver/errors"
	"github.com/platform-mesh/iam-service/pkg/sorter"
)

var _ api.ResolverService = (*Service)(nil)

type Service struct {
	fgaService      *fga.Service
	keycloakService *keycloak.Service
	userSorter      sorter.UserSorter
	pager           pager.Pager
	mgr             mcmanager.Manager
}

func (s *Service) Me(ctx context.Context) (*graph.User, error) {
	webToken, err := pmcontext.GetWebTokenFromContext(ctx)
	if err != nil {
		return nil, serrors.ErrInternal
	}
	return s.keycloakService.UserByMail(ctx, webToken.Mail)
}

func (s *Service) User(ctx context.Context, userID string) (*graph.User, error) {
	return s.keycloakService.UserByMail(ctx, userID)
}

func (s *Service) Users(ctx context.Context, rctx graph.ResourceContext, roleFilters []string, sortBy *graph.SortByInput, page *graph.PageInput) (*graph.UserConnection, error) {

	ai, err := appcontext.GetAccountInfo(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster ID from account path")
	}

	allUserRoles, err := s.fgaService.ListUsers(ctx, rctx, roleFilters, ai)
	if err != nil {
		return nil, err
	}

	err = s.keycloakService.EnrichUserRoles(ctx, allUserRoles)
	if err != nil {
		return nil, err
	}

	s.userSorter.SortUserRoles(allUserRoles, sortBy)

	totalCount := len(allUserRoles)
	paginatedUserRoles, pageInfo := s.pager.PaginateUserRoles(allUserRoles, page, totalCount)

	return &graph.UserConnection{Users: paginatedUserRoles, PageInfo: pageInfo}, nil
}

func (s *Service) AssignRolesToUsers(ctx context.Context, rCtx graph.ResourceContext, changes []*graph.UserRoleChange) (*graph.RoleAssignmentResult, error) {
	ai, err := appcontext.GetAccountInfo(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster ID from account path")
	}
	return s.fgaService.AssignRolesToUsers(ctx, rCtx, ai, changes)
}

func (s *Service) RemoveRole(ctx context.Context, rCtx graph.ResourceContext, input graph.RemoveRoleInput) (*graph.RoleRemovalResult, error) {
	ai, err := appcontext.GetAccountInfo(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster ID from account path")
	}
	return s.fgaService.RemoveRole(ctx, rCtx, input, ai)
}

func (s *Service) Roles(ctx context.Context, context graph.ResourceContext) ([]*graph.Role, error) {
	return s.fgaService.GetRoles(ctx, context)
}

func NewResolverService(fgaClient openfgav1.OpenFGAServiceClient, service *keycloak.Service, cfg *config.ServiceConfig, mgr mcmanager.Manager) (*Service, error) {
	fgaService, err := fga.New(fgaClient, cfg)
	if err != nil {
		return nil, err
	}

	return &Service{
		fgaService:      fgaService,
		keycloakService: service,
		userSorter:      sorter.NewUserSorterWithConfig(cfg),
		pager:           pager.NewPager(cfg),
		mgr:             mgr,
	}, nil
}
