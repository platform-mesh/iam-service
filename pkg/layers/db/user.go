package db

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/openmfp/iam-service/pkg/layers/core/models"
	"github.com/openmfp/iam-service/pkg/layers/db/dbmodels"
)

func (s *Service) CreateUser(ctx context.Context, input models.User) (*models.User, error) {
	// Check if user already exists
	existingUser, err := s.GetUserByIDOrEmail(ctx, input)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// if user exists, return
	if existingUser.ID != "" {
		return existingUser, nil
	}

	// GetUserByID does not exist, go ahead and create it
	newUser := &dbmodels.User{
		UserID:    input.UserID,
		TenantID:  input.TenantID,
		Email:     input.Email,
		FirstName: input.FirstName,
		LastName:  input.LastName,
	}

	err = s.conn.Create(&newUser).Error
	if err != nil {
		return nil, errors.New("could not create item")
	}

	return &models.User{
		ID:        newUser.ID,
		UserID:    newUser.UserID,
		TenantID:  newUser.TenantID,
		Email:     newUser.Email,
		FirstName: newUser.FirstName,
		LastName:  newUser.LastName,
	}, nil
}

// GetUserByIDOrEmail returns a member by ID or email
func (s *Service) GetUserByIDOrEmail(ctx context.Context, input models.User) (*models.User, error) {
	if input.TenantID == "" {
		return nil, errors.New("tenantID is required")
	}

	if input.UserID == "" && input.Email == "" {
		return nil, errors.New("at least one of userId and email has to be provided")
	}

	var item models.User
	userSelector := s.conn
	if input.UserID != "" {
		userSelector = userSelector.Or("user_id = ?", input.UserID)
	}
	if input.Email != "" {
		userSelector = userSelector.Or("email = ?", input.Email)
	}

	result := s.conn.
		Where("tenant_id = ?", input.TenantID).
		Where(userSelector).
		First(&item)

	return &item, result.Error
}
