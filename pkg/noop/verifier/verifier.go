package verifier

import (
	"context"

	"github.com/openmfp/iam-service/pkg/layers/core/models"
)

type Noop struct{}

func NewNoop() *Noop {
	return &Noop{}
}

func (*Noop) VerifyCreateUserInput(ctx context.Context, params models.User) error {
	return nil
}
