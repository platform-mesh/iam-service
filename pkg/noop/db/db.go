package db

import (
	"context"

	"github.com/openmfp/iam-service/pkg/layers/core/models"
)

type Noop struct{}

func NewNoop() *Noop {
	return &Noop{}
}

func (*Noop) CreateUser(ctx context.Context, input models.User) (*models.User, error) {
	return &input, nil
}
