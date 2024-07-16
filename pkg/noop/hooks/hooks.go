package hooks

import (
	"context"

	"github.com/openmfp/iam-service/pkg/layers/core/models"
	"github.com/openmfp/iam-service/pkg/layers/hooks"
)

type Noop struct{}

func NewNoop() hooks.Provider {
	return &Noop{}
}

func (*Noop) UserCreated(ctx context.Context, params models.User) error {
	return nil
}
