package db

import (
	"context"
	"errors"
	"gorm.io/gorm"

	"github.com/openmfp/iam-service/pkg/layers/core/models"
)

// Provider will take care of all database operations
type Provider interface {
	CreateUser(ctx context.Context, input models.User) (*models.User, error)
}

type Service struct {
	conn *gorm.DB
}

func New(conn *gorm.DB) (*Service, error) {
	if conn == nil {
		return nil, errors.New("connection is nil")
	}

	return &Service{
		conn: conn,
	}, nil
}
