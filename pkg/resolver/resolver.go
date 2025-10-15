package resolver

import (
	"github.com/platform-mesh/golang-commons/logger"

	"github.com/platform-mesh/iam-service/pkg/resolver/api"
)

//go:generate go run github.com/99designs/gqlgen@v0.17.81 generate

type Resolver struct {
	svc    api.ResolverService
	logger *logger.Logger
}

func New(svc api.ResolverService, logger *logger.Logger) *Resolver {
	return &Resolver{
		svc:    svc,
		logger: logger,
	}
}
