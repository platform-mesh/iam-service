package fga

import (
	"context"
	"fmt"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/platform-mesh/golang-commons/errors"
	"github.com/platform-mesh/golang-commons/logger"
	"go.opentelemetry.io/otel"

	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/middleware/kcp"
)

var (
	defaultRoles = []string{"owner", "member"}
	userFilter   = []*openfgav1.UserTypeFilter{{Type: "user"}}
)

type UserIDToRoles map[string][]string

type Service struct {
	client openfgav1.OpenFGAServiceClient
	helper *StoreHelper
}

func New(client openfgav1.OpenFGAServiceClient) *Service {
	return &Service{
		client: client,
		helper: NewStoreHelper(),
	}
}

func (s *Service) ListUsers(ctx context.Context, rCtx graph.ResourceContext, roleFilters []string) (UserIDToRoles, error) {
	log := logger.LoadLoggerFromContext(ctx)
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "fga.ListUsers")
	defer span.End()

	kctx, err := kcp.GetKcpUserContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kcp user context")
	}

	storeID, err := s.helper.GetStoreID(ctx, s.client, kctx.OrganizationName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get store ID for organization %s", kctx.OrganizationName)
	}

	appliedRoles := applyRoleFilter(roleFilters, log)

	allUserIDToRoles := UserIDToRoles{}
	for _, role := range appliedRoles {

		req := &openfgav1.ListUsersRequest{
			StoreId: storeID,
			Object: &openfgav1.Object{
				Type: "role",
				Id: fmt.Sprintf("%s/%s/%s/%s",
					rCtx.GroupResource,
					kctx.ClusterId,
					rCtx.Resource.Name,
					role),
			},
			Relation:    "assignee",
			UserFilters: userFilter,
		}
		users, err := s.client.ListUsers(ctx, req)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list users for resource %s with role %s", rCtx.Resource.Name, role)
		}
		for _, tuple := range users.Users {
			user := tuple.User.(*openfgav1.User_Object)
			allUserIDToRoles[user.Object.Id] = append(allUserIDToRoles[user.Object.Id], role)
		}
	}
	return allUserIDToRoles, nil
}

func applyRoleFilter(roleFilters []string, log *logger.Logger) []string {
	var appliedRoles []string
	if len(roleFilters) > 0 {
		log.Debug().Interface("roleFilters", roleFilters).Msg("Applying role filters")
		for _, role := range defaultRoles {
			if contains := containsString(roleFilters, role); contains {
				appliedRoles = append(appliedRoles, role)
			}
		}
	} else {
		appliedRoles = defaultRoles
	}
	return appliedRoles
}

var containsString = func(arr []string, s string) bool {
	for _, a := range arr {
		if a == s {
			return true
		}
	}
	return false
}
