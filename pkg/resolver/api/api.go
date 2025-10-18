package api

import (
	"context"

	"github.com/platform-mesh/iam-service/pkg/graph"
)

type ResolverService interface {
	Me(ctx context.Context) (*graph.User, error)
	User(ctx context.Context, userID string) (*graph.User, error)
	Users(ctx context.Context, context graph.ResourceContext, roleFilters []string, sortBy *graph.SortByInput, page *graph.PageInput) (*graph.UserConnection, error)
	Roles(ctx context.Context, context graph.ResourceContext) ([]*graph.Role, error)
	AssignRolesToUsers(ctx context.Context, context graph.ResourceContext, changes []*graph.UserRoleChange) (*graph.RoleAssignmentResult, error)
	RemoveRole(ctx context.Context, context graph.ResourceContext, input graph.RemoveRoleInput) (*graph.RoleRemovalResult, error)
}
