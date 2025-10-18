package fga

import (
	"context"
	"fmt"
	"sync"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/platform-mesh/golang-commons/errors"
	"github.com/platform-mesh/golang-commons/logger"
	"go.opentelemetry.io/otel"

	"github.com/platform-mesh/iam-service/pkg/config"
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

func NewWithConfig(client openfgav1.OpenFGAServiceClient, cfg *config.ServiceConfig) *Service {
	return &Service{
		client: client,
		helper: NewStoreHelperWithTTL(cfg.Keycloak.Cache.TTL),
	}
}

func (s *Service) ListUsers(ctx context.Context, rCtx graph.ResourceContext, roleFilters []string) ([]*graph.UserRoles, error) {
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

	// If no roles to process, return empty result
	if len(appliedRoles) == 0 {
		return []*graph.UserRoles{}, nil
	}

	// Use parallel processing for multiple roles
	return s.listUsersParallel(ctx, rCtx, kctx, storeID, appliedRoles)
}

// listUsersParallel performs parallel ListUsers calls for multiple roles
func (s *Service) listUsersParallel(ctx context.Context, rCtx graph.ResourceContext, kctx kcp.KCPContext, storeID string, roles []string) ([]*graph.UserRoles, error) {
	// Result structures
	type roleResult struct {
		role  string
		users *openfgav1.ListUsersResponse
		err   error
	}

	// Create channels for goroutine communication
	resultChan := make(chan roleResult, len(roles))

	// Launch goroutines for each role
	for _, role := range roles {
		go func(role string) {
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
			resultChan <- roleResult{
				role:  role,
				users: users,
				err:   err,
			}
		}(role)
	}

	// Collect results from all goroutines
	allUserIDToRoles := UserIDToRoles{}
	var mu sync.Mutex

	for i := 0; i < len(roles); i++ {
		result := <-resultChan

		// Handle any errors
		if result.err != nil {
			return nil, errors.Wrap(result.err, "failed to list users for resource %s with role %s", rCtx.Resource.Name, result.role)
		}

		// Process users for this role with thread safety
		mu.Lock()
		for _, tuple := range result.users.Users {
			user := tuple.User.(*openfgav1.User_Object)
			allUserIDToRoles[user.Object.Id] = append(allUserIDToRoles[user.Object.Id], result.role)
		}
		mu.Unlock()
	}

	// Convert UserIDToRoles to []*graph.UserRoles
	return s.convertToGraphUserRoles(allUserIDToRoles), nil
}

// convertToGraphUserRoles converts UserIDToRoles map to []*graph.UserRoles
func (s *Service) convertToGraphUserRoles(userIDToRoles UserIDToRoles) []*graph.UserRoles {
	var result []*graph.UserRoles

	for userID, roleNames := range userIDToRoles {
		// Create User with available information (only userID from OpenFGA)
		user := &graph.User{
			UserID: "",
			Email:  userID, // Not available from OpenFGA ListUsers response
		}

		// Convert role names to Role objects
		var roles []*graph.Role
		for _, roleName := range roleNames {
			role := &graph.Role{
				TechnicalName: roleName,
				DisplayName:   roleName, // Using technical name as display name for now
			}
			roles = append(roles, role)
		}

		// Create UserRoles entry
		userRoles := &graph.UserRoles{
			User:  user,
			Roles: roles,
		}

		result = append(result, userRoles)
	}

	return result
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

// AssignRolesToUsers creates tuples in FGA for the given users and roles
func (s *Service) AssignRolesToUsers(ctx context.Context, rCtx graph.ResourceContext, changes []*graph.UserRoleChange) (*graph.RoleAssignmentResult, error) {
	log := logger.LoadLoggerFromContext(ctx)
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "fga.AssignRolesToUsers")
	defer span.End()

	kctx, err := kcp.GetKcpUserContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kcp user context")
	}

	storeID, err := s.helper.GetStoreID(ctx, s.client, kctx.OrganizationName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get store ID for organization %s", kctx.OrganizationName)
	}

	var allErrors []string
	var totalAssigned int

	// Process each user role change
	for _, change := range changes {
		log.Debug().Str("userId", change.UserID).Interface("roles", change.Roles).Msg("Processing role assignment")

		// Validate that only default roles are being assigned
		for _, role := range change.Roles {
			if !containsString(defaultRoles, role) {
				errMsg := fmt.Sprintf("role '%s' is not allowed for user '%s'. Only roles %v are permitted", role, change.UserID, defaultRoles)
				allErrors = append(allErrors, errMsg)
				log.Warn().Str("role", role).Str("userId", change.UserID).Msg("Invalid role assignment attempted")
				continue
			}

			// Create the tuple for this user-role combination
			tuple := &openfgav1.TupleKey{
				User:     fmt.Sprintf("user:%s", change.UserID),
				Relation: "assignee",
				Object: fmt.Sprintf("role:%s/%s/%s/%s",
					rCtx.GroupResource,
					kctx.ClusterId,
					rCtx.Resource.Name,
					role),
			}

			// Write the tuple to FGA
			writeReq := &openfgav1.WriteRequest{
				StoreId: storeID,
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{tuple},
				},
			}

			_, err := s.client.Write(ctx, writeReq)
			if err != nil {
				// Check if this is a duplicate write error (tuple already exists)
				if s.helper.IsDuplicateWriteError(err) {
					// Treat duplicate writes as successful - the role is already assigned
					totalAssigned++
					log.Info().Str("role", role).Str("userId", change.UserID).Msg("Role already assigned to user - skipping duplicate")
				} else {
					// This is a real error
					errMsg := fmt.Sprintf("failed to assign role '%s' to user '%s': %v", role, change.UserID, err)
					allErrors = append(allErrors, errMsg)
					log.Error().Err(err).Str("role", role).Str("userId", change.UserID).Msg("Failed to write tuple to FGA")
				}
			} else {
				totalAssigned++
				log.Info().Str("role", role).Str("userId", change.UserID).Msg("Successfully assigned role to user")
			}
		}
	}

	// Determine overall success
	success := len(allErrors) == 0

	return &graph.RoleAssignmentResult{
		Success:       success,
		Errors:        allErrors,
		AssignedCount: totalAssigned,
	}, nil
}

var containsString = func(arr []string, s string) bool {
	for _, a := range arr {
		if a == s {
			return true
		}
	}
	return false
}
