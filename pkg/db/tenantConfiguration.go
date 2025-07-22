package db

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	commonsCtx "github.com/platform-mesh/golang-commons/context"
)

type TenantConfiguration struct {
	TenantID string `gorm:"primary_key"`
	Issuer   string `gorm:"primary_key"`
	Audience string `gorm:"primary_key"`
	ZoneId   string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (d *Database) GetTenantConfigurationForContext(ctx context.Context) (*TenantConfiguration, error) {
	// retrieve jwt from context
	tokenInfo, err := commonsCtx.GetWebTokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return d.GetTenantConfigurationByIssuerAndAudience(ctx, tokenInfo.Issuer, tokenInfo.Audiences)
}

func (d *Database) GetTenantConfigurationByIssuerAndAudience(
	ctx context.Context, issuer string, audiences []string,
) (*TenantConfiguration, error) {
	var item TenantConfiguration
	res := d.db.
		Where("issuer = ?", issuer).
		Where("audience IN ?", audiences).
		First(&item)

	if res.Error != nil && !errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, res.Error
	}

	if res.RowsAffected == 1 {
		return &item, nil
	}

	// Audience did not match
	return nil, nil

}
