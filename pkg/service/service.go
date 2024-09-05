package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	openmfpCtx "github.com/openmfp/golang-commons/context"
	"github.com/openmfp/golang-commons/sentry"

	"github.com/openmfp/iam-service/pkg/db"
	"github.com/openmfp/iam-service/pkg/graph"
)

const (
	MaxSearchUsersResults = 5
)

type ServiceInterface interface {
	InviteUser(ctx context.Context, tenantID string, invite graph.Invite, notifyByEmail bool) (bool, error)
	DeleteInvite(ctx context.Context, tenantID string, invite graph.Invite) (bool, error)
	CreateUser(ctx context.Context, tenantID string, input graph.UserInput) (*graph.User, error)
	RemoveUser(ctx context.Context, tenantID string, rawUserID *string, rawEmail *string) (*bool, error)
	User(ctx context.Context, tenantID string, userID string) (*graph.User, error)
	UserByEmail(ctx context.Context, tenantID string, email string) (*graph.User, error)
	UsersConnection(ctx context.Context, tenantID string, limit *int, page *int) (*graph.UserConnection, error)
	GetZone(ctx context.Context) (*graph.Zone, error)
	SearchUsers(ctx context.Context, query string) ([]*graph.User, error)
}

type Service struct {
	Db db.Service
}

func New(db db.Service) *Service {
	return &Service{Db: db}
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

func (s *Service) SearchUsers(ctx context.Context, query string) ([]*graph.User, error) {
	logger := setupLogger(ctx)

	tenantID, err := openmfpCtx.GetTenantFromContext(ctx)
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

	users, err := s.Db.SearchUsers(ctx, tenantID, query, MaxSearchUsersResults)
	if err != nil {
		logger.Error().Err(err).Msg("SearchUsers failed")
		return nil, sentry.SentryError(err)
	}

	return users, nil
}
