package verifier

import (
	"context"

	"github.com/openmfp/iam-service/pkg/layers/core/models"
)

type Provider interface {
	VerifyCreateUserInput(ctx context.Context, params models.User) error
}
