package core

import (
	"context"

	"github.com/openmfp/golang-commons/logger"

	"github.com/openmfp/iam-service/pkg/layers/core/models"
)

func (s *Service) CreateUser(ctx context.Context, input models.User) (*models.User, error) {
	log := logger.LoadLoggerFromContext(ctx).
		ComponentLogger("core").
		With().
		Str("Email", input.Email).
		Str("tenantID", input.TenantID).Logger()

	err := s.verifier.VerifyCreateUserInput(ctx, input)
	if err != nil {
		log.Error().Err(err).
			Msg("ValidateCreateUserInput failed")

		return nil, err
	}

	createdUser, err := s.db.CreateUser(ctx, input)
	if err != nil {
		log.Error().Err(err).
			Msg("CreateUser failed")

		return nil, err
	}

	err = s.hooks.UserCreated(ctx, *createdUser)
	if err != nil {
		log.Error().Err(err).
			Msg("UserCreated failed")
	}

	return createdUser, nil
}
