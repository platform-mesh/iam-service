package resolver

import (
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/platform-mesh/iam-service/pkg/service"
)

//go:generate go run github.com/99designs/gqlgen@v0.17.72 generate

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	service service.ServiceInterface
	logger  *logger.Logger
}

func New(service service.ServiceInterface, logger *logger.Logger) *Resolver {
	return &Resolver{
		service: service,
		logger:  logger,
	}
}
