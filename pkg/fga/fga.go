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
	"github.com/platform-mesh/iam-service/pkg/roles"
)

var (
	userFilter = []*openfgav1.UserTypeFilter{{Type: "user"}}
)

type UserIDToRoles map[string][]string

type Service struct {
	client         openfgav1.OpenFGAServiceClient
	helper         *StoreHelper
	rolesRetriever roles.RolesRetriever
}

func New(client openfgav1.OpenFGAServiceClient) *Service {
	// For backward compatibility, use default roles retriever
	var rolesRetriever roles.RolesRetriever
	defaultRetriever, err := roles.NewDefaultRolesRetriever()
	if err != nil {
		// Fallback to a basic implementation if file not found
		rolesRetriever = NewStaticRolesRetriever()
	} else {
		rolesRetriever = defaultRetriever
	}

	return &Service{
		client:         client,
		helper:         NewStoreHelper(),
		rolesRetriever: rolesRetriever,
	}
}

func NewWithConfig(client openfgav1.OpenFGAServiceClient, cfg *config.ServiceConfig) *Service {
	// For backward compatibility, use default roles retriever
	var rolesRetriever roles.RolesRetriever
	defaultRetriever, err := roles.NewDefaultRolesRetriever()
	if err != nil {
		// Fallback to a basic implementation if file not found
		rolesRetriever = NewStaticRolesRetriever()
	} else {
		rolesRetriever = defaultRetriever
	}

	return &Service{
		client:         client,
		helper:         NewStoreHelperWithTTL(cfg.Keycloak.Cache.TTL),
		rolesRetriever: rolesRetriever,
	}
}

// NewWithRolesRetriever creates a new FGA service with a custom roles retriever
func NewWithRolesRetriever(client openfgav1.OpenFGAServiceClient, cfg *config.ServiceConfig, rolesRetriever roles.RolesRetriever) *Service {
	helper := NewStoreHelper()
	if cfg != nil {
		helper = NewStoreHelperWithTTL(cfg.Keycloak.Cache.TTL)
	}

	return &Service{
		client:         client,
		helper:         helper,
		rolesRetriever: rolesRetriever,
	}
}

// StaticRolesRetriever provides backward compatibility with hardcoded roles
type StaticRolesRetriever struct{}

// NewStaticRolesRetriever creates a static roles retriever with hardcoded roles
func NewStaticRolesRetriever() *StaticRolesRetriever {
	return &StaticRolesRetriever{}
}

// GetAvailableRoles returns the static list of roles (backward compatibility)
func (r *StaticRolesRetriever) GetAvailableRoles(groupResource string) ([]string, error) {
	// Return the old default roles for backward compatibility
	return []string{"owner", "member"}, nil
}

// GetRoleDefinitions returns static role definitions
func (r *StaticRolesRetriever) GetRoleDefinitions(groupResource string) ([]roles.RoleDefinition, error) {
	return []roles.RoleDefinition{
		{ID: "owner", DisplayName: "Owner", Description: "Full access to all resources"},
		{ID: "member", DisplayName: "Member", Description: "Limited access to resources"},
	}, nil
}

// Reload is a no-op for static retriever
func (r *StaticRolesRetriever) Reload() error {
	return nil
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

	appliedRoles, err := s.applyRoleFilter(rCtx.GroupResource, roleFilters, log)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available roles for group resource %s", rCtx.GroupResource)
	}

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
	return s.convertToGraphUserRoles(rCtx.GroupResource, allUserIDToRoles), nil
}

// convertToGraphUserRoles converts UserIDToRoles map to []*graph.UserRoles
func (s *Service) convertToGraphUserRoles(groupResource string, userIDToRoles UserIDToRoles) []*graph.UserRoles {
	var result []*graph.UserRoles

	// Get role definitions for this group resource
	roleDefinitions, err := s.rolesRetriever.GetRoleDefinitions(groupResource)
	if err != nil {
		// Fallback to basic roles if we can't get definitions
		roleDefinitions = []roles.RoleDefinition{}
	}

	// Create a map for quick role definition lookup
	roleDefMap := make(map[string]roles.RoleDefinition)
	for _, roleDef := range roleDefinitions {
		roleDefMap[roleDef.ID] = roleDef
	}

	for userID, roleNames := range userIDToRoles {
		// Create User with available information (only userID from OpenFGA)
		user := &graph.User{
			UserID: "",
			Email:  userID, // Not available from OpenFGA ListUsers response
		}

		// Convert role names to Role objects
		var roles []*graph.Role
		for _, roleName := range roleNames {
			// Try to get the full role definition, fallback to basic info
			if roleDef, exists := roleDefMap[roleName]; exists {
				role := &graph.Role{
					ID:          roleDef.ID,
					DisplayName: roleDef.DisplayName,
					Description: roleDef.Description,
				}
				roles = append(roles, role)
			} else {
				// Fallback for roles not found in definitions
				role := &graph.Role{
					ID:          roleName,
					DisplayName: roleName,
					Description: "Role definition not available",
				}
				roles = append(roles, role)
			}
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

func (s *Service) GetRoles(ctx context.Context, rCtx graph.ResourceContext) ([]*graph.Role, error) {
	log := logger.LoadLoggerFromContext(ctx)
	_, span := otel.GetTracerProvider().Tracer("").Start(ctx, "fga.GetRoles")
	defer span.End()

	log.Debug().Str("groupResource", rCtx.GroupResource).Msg("Getting available roles")

	// Get role definitions from the roles retriever
	roleDefinitions, err := s.rolesRetriever.GetRoleDefinitions(rCtx.GroupResource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get role definitions for group resource %s", rCtx.GroupResource)
	}

	// Convert to graph.Role objects
	var roles []*graph.Role
	for _, roleDef := range roleDefinitions {
		role := &graph.Role{
			ID:          roleDef.ID,
			DisplayName: roleDef.DisplayName,
			Description: roleDef.Description,
		}
		roles = append(roles, role)
	}

	log.Debug().Int("roleCount", len(roles)).Msg("Successfully retrieved roles")
	return roles, nil
}

func (s *Service) applyRoleFilter(groupResource string, roleFilters []string, log *logger.Logger) ([]string, error) {
	availableRoles, err := s.rolesRetriever.GetAvailableRoles(groupResource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available roles for group resource %s", groupResource)
	}

	var appliedRoles []string
	if len(roleFilters) > 0 {
		log.Debug().Interface("roleFilters", roleFilters).Interface("availableRoles", availableRoles).Msg("Applying role filters")
		for _, role := range availableRoles {
			if contains := containsString(roleFilters, role); contains {
				appliedRoles = append(appliedRoles, role)
			}
		}
	} else {
		appliedRoles = availableRoles
	}
	return appliedRoles, nil
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

		// Validate that only available roles are being assigned
		availableRoles, err := s.rolesRetriever.GetAvailableRoles(rCtx.GroupResource)
		if err != nil {
			errMsg := fmt.Sprintf("failed to get available roles for group resource '%s': %v", rCtx.GroupResource, err)
			allErrors = append(allErrors, errMsg)
			log.Error().Err(err).Str("groupResource", rCtx.GroupResource).Msg("Failed to retrieve available roles")
			continue
		}

		for _, role := range change.Roles {
			if !containsString(availableRoles, role) {
				errMsg := fmt.Sprintf("role '%s' is not allowed for user '%s'. Only roles %v are permitted", role, change.UserID, availableRoles)
				allErrors = append(allErrors, errMsg)
				log.Warn().Str("role", role).Str("userId", change.UserID).Interface("availableRoles", availableRoles).Msg("Invalid role assignment attempted")
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

// RemoveRole removes a role from a user by deleting the tuple in FGA
func (s *Service) RemoveRole(ctx context.Context, rCtx graph.ResourceContext, input graph.RemoveRoleInput) (*graph.RoleRemovalResult, error) {
	log := logger.LoadLoggerFromContext(ctx)
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "fga.RemoveRole")
	defer span.End()

	kctx, err := kcp.GetKcpUserContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kcp user context")
	}

	storeID, err := s.helper.GetStoreID(ctx, s.client, kctx.OrganizationName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get store ID for organization %s", kctx.OrganizationName)
	}

	log.Debug().Str("userId", input.UserID).Str("role", input.Role).Msg("Processing role removal")

	// Validate that only available roles can be removed
	availableRoles, err := s.rolesRetriever.GetAvailableRoles(rCtx.GroupResource)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get available roles for group resource '%s': %v", rCtx.GroupResource, err)
		log.Error().Err(err).Str("groupResource", rCtx.GroupResource).Msg("Failed to retrieve available roles")
		return &graph.RoleRemovalResult{
			Success:     false,
			Error:       &errMsg,
			WasAssigned: false,
		}, nil
	}

	if !containsString(availableRoles, input.Role) {
		errMsg := fmt.Sprintf("role '%s' is not allowed. Only roles %v are permitted", input.Role, availableRoles)
		log.Warn().Str("role", input.Role).Str("userId", input.UserID).Interface("availableRoles", availableRoles).Msg("Invalid role removal attempted")
		return &graph.RoleRemovalResult{
			Success:     false,
			Error:       &errMsg,
			WasAssigned: false,
		}, nil
	}

	// First, check if the tuple exists by trying to read it
	readTuple := &openfgav1.ReadRequestTupleKey{
		User:     fmt.Sprintf("user:%s", input.UserID),
		Relation: "assignee",
		Object: fmt.Sprintf("role:%s/%s/%s/%s",
			rCtx.GroupResource,
			kctx.ClusterId,
			rCtx.Resource.Name,
			input.Role),
	}

	readReq := &openfgav1.ReadRequest{
		StoreId:  storeID,
		TupleKey: readTuple,
	}

	readResp, err := s.client.Read(ctx, readReq)
	if err != nil {
		log.Error().Err(err).Str("role", input.Role).Str("userId", input.UserID).Msg("Failed to check if tuple exists")
		errMsg := fmt.Sprintf("failed to check role assignment: %v", err)
		return &graph.RoleRemovalResult{
			Success:     false,
			Error:       &errMsg,
			WasAssigned: false,
		}, nil
	}

	// Check if the tuple was found
	wasAssigned := len(readResp.Tuples) > 0
	if !wasAssigned {
		log.Info().Str("role", input.Role).Str("userId", input.UserID).Msg("Role was not assigned to user - nothing to remove")
		return &graph.RoleRemovalResult{
			Success:     true,
			Error:       nil,
			WasAssigned: false,
		}, nil
	}

	// Delete the tuple from FGA
	deleteTuple := &openfgav1.TupleKeyWithoutCondition{
		User:     fmt.Sprintf("user:%s", input.UserID),
		Relation: "assignee",
		Object: fmt.Sprintf("role:%s/%s/%s/%s",
			rCtx.GroupResource,
			kctx.ClusterId,
			rCtx.Resource.Name,
			input.Role),
	}

	deleteReq := &openfgav1.WriteRequest{
		StoreId: storeID,
		Deletes: &openfgav1.WriteRequestDeletes{
			TupleKeys: []*openfgav1.TupleKeyWithoutCondition{deleteTuple},
		},
	}

	_, err = s.client.Write(ctx, deleteReq)
	if err != nil {
		log.Error().Err(err).Str("role", input.Role).Str("userId", input.UserID).Msg("Failed to delete tuple from FGA")
		errMsg := fmt.Sprintf("failed to remove role '%s' from user '%s': %v", input.Role, input.UserID, err)
		return &graph.RoleRemovalResult{
			Success:     false,
			Error:       &errMsg,
			WasAssigned: true,
		}, nil
	}

	log.Info().Str("role", input.Role).Str("userId", input.UserID).Msg("Successfully removed role from user")
	return &graph.RoleRemovalResult{
		Success:     true,
		Error:       nil,
		WasAssigned: true,
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
