package hooks

import (
	"context"

	"github.com/openmfp/iam-service/pkg/layers/core/models"
)

// Provider will take care of specific hooks
type Provider interface {
	User
}

type User interface {
	UserCreated(ctx context.Context, params models.User) error
}
