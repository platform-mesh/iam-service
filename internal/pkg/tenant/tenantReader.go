package tenant

import (
	"context"

	"github.com/openmfp/golang-commons/logger"
	"github.com/openmfp/golang-commons/policy_services"
	"github.com/openmfp/iam-service/pkg/db"
)

type TenantReader struct {
	logger *logger.Logger
	db     db.Service
}

func NewTenantReader(logger *logger.Logger, database db.Service) policy_services.TenantIdReader { // nolint: ireturn
	return &TenantReader{
		logger: logger,
		db:     database,
	}
}

func (t *TenantReader) Read(parentCtx context.Context) (string, error) {
	tenant, err := t.GetTenant(parentCtx)
	if err != nil {
		return "", err
	}
	return tenant, nil
}

func (s *TenantReader) GetTenant(ctx context.Context) (string, error) {
	tc, err := s.db.GetTenantConfigurationForContext(ctx)
	if err != nil {
		return "", err
	}
	return tc.TenantID, nil
}
