package service

import (
	"context"
	"errors"

	"slices"
	"strings"

	"fmt"

	fgastore "github.com/platform-mesh/golang-commons/fga/store"

	"gorm.io/gorm"

	pmctx "github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/sentry"

	"github.com/platform-mesh/iam-service/pkg/db"
	"github.com/platform-mesh/iam-service/pkg/fga"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.opentelemetry.io/otel"
)

type ServiceInterface interface { // nolint: interfacebloat
	AssignRoleBindings(ctx context.Context, tenantID string, entityType string, entityID string, input []*graph.Change) (bool, error)
	UsersOfEntity(ctx context.Context, tenantID string, entity graph.EntityInput, limit *int,
		page *int, showInvitees *bool, searchTerm *string, rolesFilter []*graph.RoleInput, sortBy *graph.SortByInput,
	) (*graph.GrantedUserConnection, error)
	RemoveFromEntity(ctx context.Context, tenantID string, entityType string, userID string, entityID string) (bool, error)
	LeaveEntity(ctx context.Context, tenantID string, entityType string, entityID string) (bool, error)
	RolesForUserOfEntity(ctx context.Context, tenantID string, entity graph.EntityInput, userID string) ([]*graph.Role, error)
	AvailableRolesForEntity(ctx context.Context, tenantID string, entity graph.EntityInput) ([]*graph.Role, error)
	AvailableRolesForEntityType(ctx context.Context, tenantID string, entityType string) ([]*graph.Role, error)
	InviteUser(ctx context.Context, tenantID string, invite graph.Invite, notifyByEmail bool) (bool, error)
	DeleteInvite(ctx context.Context, tenantID string, invite graph.Invite) (bool, error)
	CreateUser(ctx context.Context, tenantID string, input graph.UserInput) (*graph.User, error)
	RemoveUser(ctx context.Context, tenantID string, rawUserID *string, rawEmail *string) (*bool, error)
	User(ctx context.Context, tenantID string, userID string) (*graph.User, error)
	UserByEmail(ctx context.Context, tenantID string, email string) (*graph.User, error)
	UsersConnection(ctx context.Context, tenantID string, limit *int, page *int) (*graph.UserConnection, error)
	GetZone(ctx context.Context) (*graph.Zone, error)
	CreateAccount(ctx context.Context, tenantID string, entityType string, entityID string, owner string) (bool, error)
	RemoveAccount(ctx context.Context, tenantID string, entityType string, entityID string) (bool, error)
	TenantInfo(ctx context.Context, tenantIdInput *string) (*graph.TenantInfo, error)
	SearchUsers(ctx context.Context, query string) ([]*graph.User, error)
	UsersByIds(ctx context.Context, tenantID string, userIds []string) ([]*graph.User, error)
}

type Service struct {
	Db  db.Service
	Fga fga.Service
}

var (
	minusOne = -1 // nolint: gochecknoglobals
	one      = 1  // nolint: gochecknoglobals
)

func New(db db.Service, fga fga.Service) *Service {
	return &Service{
		Db:  db,
		Fga: fga,
	}
}

func (s *Service) InviteUser(ctx context.Context, tenantID string, invite graph.Invite, notifyByEmail bool) (bool, error) {
	logger := setupLogger(ctx)
	_, err := s.Db.GetOrCreateUser(ctx, tenantID, graph.UserInput{Email: invite.Email})
	if err != nil {
		logger.Error().Err(err).Str("Email", invite.Email).Str("EntityID", invite.Entity.EntityID).Msg("EnsureUserExist failed")
		return false, sentry.SentryError(err)
	}

	err = s.Db.InviteUser(ctx, tenantID, invite, notifyByEmail)
	if err != nil {
		logger.Error().Err(err).Str("Email", invite.Email).Str("EntityID", invite.Entity.EntityID).Msg("InviteUser failed")
		return false, sentry.SentryError(err)
	}

	return true, nil
}

func (s *Service) DeleteInvite(ctx context.Context, tenantID string, invite graph.Invite) (bool, error) {
	logger := setupLogger(ctx)
	byEmailAndEntity := db.Invite{Email: invite.Email, EntityType: invite.Entity.EntityType, EntityID: invite.Entity.EntityID}
	err := s.Db.DeleteInvite(ctx, byEmailAndEntity)
	if err != nil {
		logger.Error().Err(err).Str("Email", invite.Email).Str("EntityID", invite.Entity.EntityID).Msg("Delete invite failed")
		return false, sentry.SentryError(err)
	}

	return true, nil
}

func (s *Service) CreateUser(ctx context.Context, tenantID string, input graph.UserInput) (*graph.User, error) {
	logger := setupLogger(ctx)
	res, err := s.Db.GetOrCreateUser(ctx, tenantID, input)
	if err != nil {
		logger.Error().Err(err).Msg("GetOrCreateUser failed")
		return nil, sentry.SentryError(err)
	}

	return res, nil
}

func (s *Service) RemoveUser(ctx context.Context, tenantID string, rawUserID *string, rawEmail *string) (*bool, error) {
	logger := setupLogger(ctx)
	var userID, email string
	if rawUserID != nil {
		userID = *rawUserID
	}
	if rawEmail != nil {
		email = *rawEmail
	}
	res, err := s.Db.RemoveUser(ctx, tenantID, userID, email)
	if err != nil {
		logger.Error().Err(err).Msg("RemoveUser failed")
		return nil, sentry.SentryError(err)
	}

	return &res, nil
}

func (s *Service) User(ctx context.Context, tenantID string, userID string) (*graph.User, error) {
	logger := setupLogger(ctx)
	user, err := s.Db.GetUserByID(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		logger.Error().Err(err).Msg("Failed to fetch user")
		return nil, sentry.SentryError(err)
	}

	return user, nil
}

func (s *Service) UserByEmail(ctx context.Context, tenantID string, email string) (*graph.User, error) {
	logger := setupLogger(ctx)

	user, err := s.Db.GetUserByEmail(ctx, tenantID, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error().Err(err).Msg("UserByEmail failed")
		return nil, sentry.SentryError(err)
	}

	return user, nil
}

func (s *Service) UsersConnection(ctx context.Context, tenantID string, limit *int, page *int) (*graph.UserConnection, error) {
	if err := VerifyLimitsWithOverride(limit, page); err != nil {
		return nil, err
	}

	logger := setupLogger(ctx)
	userConnection, err := s.Db.GetUsers(ctx, tenantID, *limit, *page)
	if err != nil {
		logger.Error().Err(err).Msg("UsersConnection failed")
		return nil, sentry.SentryError(err)
	}

	return userConnection, nil
}

func (s *Service) GetZone(ctx context.Context) (*graph.Zone, error) {
	// retrieve jwt from context
	tc, err := s.Db.GetTenantConfigurationForContext(ctx)
	if err != nil {
		return nil, err
	}

	if tc != nil {
		return &graph.Zone{
			ZoneID:   tc.ZoneId,
			TenantID: tc.TenantID,
		}, nil
	}
	return nil, nil
}

func (s *Service) AssignRoleBindings(ctx context.Context, tenantID string, entityType string,
	entityID string, input []*graph.Change) (bool, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "service.AssignRoleBindings")
	defer span.End()

	for _, in := range input {
		if len(in.Roles) == 0 {
			return false, gqlerror.Errorf("invalid input combination - please make sure to provide at least one element for input[*].roles")
		}
	}

	err := s.Fga.AssignRoleBindings(ctx, tenantID, entityType, entityID, input)

	return err == nil, err
}

func (s *Service) UsersOfEntity( // nolint: funlen, cyclop, gocognit
	ctx context.Context,
	tenantID string,
	entity graph.EntityInput,
	limit *int,
	page *int,
	showInvitees *bool,
	searchTerm *string,
	rolesfilter []*graph.RoleInput,
	sortBy *graph.SortByInput,
) (*graph.GrantedUserConnection, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "service.UsersOfEntity")
	defer span.End()

	if err := VerifyLimitsWithOverride(limit, page); err != nil {
		return nil, err
	}
	logger := setupLogger(ctx)

	var userIDToRoles map[string][]string
	var err error
	if len(rolesfilter) > 0 {
		userIDToRoles, err = s.Fga.UsersForEntityRolefilter(ctx, tenantID, entity.EntityID, entity.EntityType, rolesfilter)
	} else {
		userIDToRoles, err = s.Fga.UsersForEntity(ctx, tenantID, entity.EntityID, entity.EntityType)
	}
	if err != nil {
		return nil, err
	}

	if limit == nil {
		limit = &minusOne
	}

	if page == nil {
		page = &one
	}

	userIDs := GetUserIDsFromUserIDRoles(userIDToRoles)

	// count users for all pages
	allUsers, _ := s.Db.GetUsersByUserIDs(ctx, tenantID, userIDs, 0, *page, searchTerm, sortBy)
	ownerCount := 0
	for _, u := range allUsers {
		for _, role := range userIDToRoles[u.UserID] {
			if role == "owner" {
				ownerCount++
			}
		}
	}
	users, err := s.Db.GetUsersByUserIDs(ctx, tenantID, userIDs, *limit, *page, searchTerm, sortBy)

	if err != nil {
		logger.Error().Err(err).Msg("unable to get users by id")
		return nil, err
	}

	out := graph.GrantedUserConnection{
		Users: make([]*graph.GrantedUser, 0, len(users)),
		PageInfo: &graph.PageInfo{
			OwnerCount: ownerCount,
			TotalCount: len(allUsers),
		},
	}

	showUsers := true
	if *limit != minusOne {
		showUsers = out.PageInfo.TotalCount > ((*page - 1) * (*limit))
	}

	if showUsers {
		for _, user := range users {
			roles := userIDToRoles[user.UserID]

			resolvedRoles, err := s.getResolvedRoles(ctx, entity.EntityType, roles)
			if err != nil {
				return nil, err
			}

			out.Users = append(out.Users, &graph.GrantedUser{
				User:  user,
				Roles: resolvedRoles,
			})
		}
	}

	showInvitations := false
	if showInvitees != nil {
		showInvitations = *showInvitees
	}

	if showInvitations && *limit != MAX_INT {
		showInvitations = out.PageInfo.TotalCount < ((*page) * (*limit))
	}

	invites, err := s.Db.GetInvitesForEntity(ctx, tenantID, entity.EntityType, entity.EntityID)
	if err != nil {
		logger.Error().Err(err).Str("EntityType", entity.EntityType).
			Str("EntityID", entity.EntityID).
			Msg("unable to get invitations users for scope")
		return nil, err
	}
	invites, invitedOwners := FilterInvites(invites, searchTerm, rolesfilter)
	invitesLength := len(invites)

	if showInvitations {
		if *limit != MAX_INT {
			sliceStart, sliceEnd := GeneratePaginationLimits(*limit, len(allUsers), *page, len(invites))
			invites = invites[sliceStart:sliceEnd]
		}

		out.Users = slices.Grow(out.Users, len(invites))

		for _, invite := range invites {
			roles := strings.Split(invite.Roles, ",")
			resolvedRoles, err := s.getResolvedRoles(ctx, entity.EntityType, roles)
			if err != nil {
				return nil, err
			}

			out.Users = append(out.Users, &graph.GrantedUser{
				User: &graph.User{
					Email: invite.Email,
				},
				Roles: resolvedRoles,
			})
		}
	}
	out.PageInfo.OwnerCount += invitedOwners
	out.PageInfo.TotalCount += invitesLength

	return &out, nil
}

func (s *Service) getResolvedRoles(ctx context.Context, entityType string, roles []string) ([]*graph.Role, error) {
	dbRoles, err := s.Db.GetRolesByTechnicalNames(ctx, entityType, roles)
	if err != nil {
		return nil, err
	}

	resolvedRoles := make([]*graph.Role, 0, len(dbRoles))
	for _, role := range dbRoles {
		resolvedRoles = append(resolvedRoles, &graph.Role{
			DisplayName:   role.DisplayName,
			TechnicalName: role.TechnicalName,
		})
	}

	return resolvedRoles, nil
}

func (s *Service) RemoveFromEntity(ctx context.Context, tenantID string, entityType string, userID string, entityID string) (bool, error) {
	err := s.Fga.RemoveFromEntity(ctx, tenantID, entityType, entityID, userID)

	return err == nil, err
}

func (s *Service) LeaveEntity(ctx context.Context, tenantID string, entityType string, entityID string) (bool, error) {

	token, err := pmctx.GetWebTokenFromContext(ctx)
	if err != nil {
		return false, err
	}

	err = s.Fga.RemoveFromEntity(ctx, tenantID, entityType, entityID, token.Subject)

	return err == nil, err
}

func (s *Service) RolesForUserOfEntity(ctx context.Context, tenantID string,
	entity graph.EntityInput, userID string) ([]*graph.Role, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "service.RolesForUserOfEntity")
	defer span.End()

	logger := setupLogger(ctx)

	userIDToRoles, err := s.Fga.UsersForEntity(ctx, tenantID, entity.EntityID, entity.EntityType)
	if err != nil {
		return nil, err
	}

	for grantedUserID, grantedUserRoles := range userIDToRoles {
		if grantedUserID == userID {
			roles, err := s.Db.GetRolesByTechnicalNames(ctx, entity.EntityType, grantedUserRoles)
			if err != nil {
				logger.Error().Err(err).Msg("unable to get roles by technical names")
				return nil, err
			}

			var grantedRolesForUser []*graph.Role
			for _, role := range roles {
				grantedRolesForUser = append(grantedRolesForUser, &graph.Role{
					DisplayName:   role.DisplayName,
					TechnicalName: role.TechnicalName,
				})
			}

			return grantedRolesForUser, nil
		}
	}

	return nil, nil
}

func (s *Service) AvailableRolesForEntity(ctx context.Context, tenantID string, entity graph.EntityInput) ([]*graph.Role, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "service.AvailableRolesForEntity")
	defer span.End()
	return s.getRolesForEntity(ctx, entity.EntityType, entity.EntityID)
}

func (s *Service) AvailableRolesForEntityType(ctx context.Context, tenantID string, entityType string) ([]*graph.Role, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "service.AvailableRolesForEntityType")
	defer span.End()
	return s.getRolesForEntity(ctx, entityType, "")
}

func (s *Service) getRolesForEntity(ctx context.Context, entityType string, entityID string) ([]*graph.Role, error) {
	roles, err := s.Db.GetRolesForEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, err
	}

	returnRoles := make([]*graph.Role, len(roles))
	for i, role := range roles {
		returnRoles[i] = &graph.Role{
			DisplayName:   role.DisplayName,
			TechnicalName: role.TechnicalName,
		}
	}

	return returnRoles, nil
}

func (s *Service) CreateAccount(ctx context.Context, tenantID string, entityType string, entityID string, owner string) (bool, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "service.CreateAccount")
	defer span.End()

	err := s.Fga.CreateAccount(ctx, tenantID, entityType, entityID, owner)

	fgaStore := fgastore.New()

	if fgaStore.IsDuplicateWriteError(err) { // do not send out events if the account already exists
		return true, nil
	}

	return err == nil, err
}

func (s *Service) RemoveAccount(ctx context.Context, tenantID string, entityType string, entityID string) (bool, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "service.RemoveAccount")
	defer span.End()

	err := s.Fga.RemoveAccount(ctx, tenantID, entityType, entityID)
	fgaStore := fgastore.New()

	if fgaStore.IsDuplicateWriteError(err) { // this error happens when the user does not exists and should be ignored
		return true, nil
	}

	return err == nil, err
}

func (s *Service) TenantInfo(ctx context.Context, tenantIdInput *string) (*graph.TenantInfo, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "service.TenantInfo")
	defer span.End()

	var tenantID string
	if tenantIdInput == nil {
		tc, err := s.Db.GetTenantConfigurationForContext(ctx)
		if err != nil {
			return nil, sentry.SentryError(err)
		}
		if tc == nil {
			return nil, sentry.SentryError(errors.New("tenant not found from JWT"))
		}
		tenantID = tc.TenantID
	} else {
		tenantID = *tenantIdInput
	}

	return &graph.TenantInfo{
		TenantID: tenantID,
	}, nil
}

func (s *Service) SearchUsers(ctx context.Context, query string) ([]*graph.User, error) {
	logger := setupLogger(ctx)

	tenantID, err := pmctx.GetTenantFromContext(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("no tenantID found in context")
		return nil, sentry.SentryError(err)
	}

	if tenantID == "" {
		return nil, fmt.Errorf("tenantID must not be empty")
	}
	if query == "" {
		return nil, fmt.Errorf("query must not be empty")
	}

	users, err := s.Db.SearchUsers(ctx, tenantID, query)
	if err != nil {
		logger.Error().Err(err).Msg("SearchUsers failed")
		return nil, sentry.SentryError(err)
	}

	return users, nil
}

func (s *Service) UsersByIds(
	ctx context.Context,
	tenantID string,
	userIds []string,
) ([]*graph.User, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "service.UsersByIds")
	defer span.End()

	logger := setupLogger(ctx)

	if tenantID == "" {
		return nil, fmt.Errorf("tenantID must not be empty")
	}
	if len(userIds) == 0 {
		return nil, fmt.Errorf("userIds must not be empty")
	}

	users, err := s.Db.GetUsersByUserIDs(ctx, tenantID, userIds, 0, 0, nil, nil)
	if err != nil {
		logger.Error().Err(err).Msg("UsersByIds failed")
		return nil, sentry.SentryError(err)
	}

	return users, nil
}
