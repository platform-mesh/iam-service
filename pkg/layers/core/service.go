package core

import (
	"errors"

	"github.com/openmfp/iam-service/pkg/layers/db"
	"github.com/openmfp/iam-service/pkg/layers/hooks"
	"github.com/openmfp/iam-service/pkg/layers/verifier"
	noopHooks "github.com/openmfp/iam-service/pkg/noop/hooks"
	noopVerifier "github.com/openmfp/iam-service/pkg/noop/verifier"
)

import (
	"context"

	"github.com/openmfp/iam-service/pkg/layers/core/models"
)

// Provider will take care of all business logic
type Provider interface {
	CreateUser(ctx context.Context, input models.User) (*models.User, error)
}

type Service struct {
	db       db.Provider
	hooks    hooks.Provider
	verifier verifier.Provider
}

// New returns a new initialized Service
// Required fields: db,
// Optional fields: hooks, verifier
func New(
	db db.Provider,
	hooks hooks.Provider,
	verifier verifier.Provider,
) (*Service, error) {
	if db == nil {
		return nil, errors.New("db not set")
	}

	if hooks == nil {
		hooks = noopHooks.NewNoop()
	}

	if verifier == nil {
		verifier = noopVerifier.NewNoop()
	}

	return &Service{
		db:       db,
		hooks:    hooks,
		verifier: verifier,
	}, nil
}
