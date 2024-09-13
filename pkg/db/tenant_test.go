package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/google/uuid"
	commonsCtx "github.com/openmfp/golang-commons/context"
	"github.com/openmfp/golang-commons/jwt"
	"github.com/openmfp/golang-commons/logger"
	"github.com/openmfp/iam-service/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestDatabase_GetTenantConfigurationForContext(t *testing.T) {
	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	gormDB := setupSQLiteDB(t)
	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.Background()

	tokenInfo := &jwt.WebToken{
		IssuerAttributes: jwt.IssuerAttributes{
			Issuer:  "https://accounts.sap.com",
			Subject: "123123-123123",
		},
		UserAttributes: jwt.UserAttributes{
			FirstName: "John",
			LastName:  "Doe",
		},
		ParsedAttributes: jwt.ParsedAttributes{
			Audiences: []string{"e3284ced-3a27-476b-9ae6-d5ad1ba05266"},
			Mail:      "mail@mail.test",
		},
	}

	// patch commonsCtx.GetWebTokenFromContext(ctx) to return tokenInfo
	patch := gomonkey.ApplyFunc(commonsCtx.GetWebTokenFromContext, func(ctx context.Context) (jwt.WebToken, error) {
		return *tokenInfo, nil
	})

	defer patch.Reset()

	// Insert test data
	tenantConfig := db.TenantConfiguration{
		TenantID:  uuid.New().String(),
		Issuer:    tokenInfo.Issuer,
		Audience:  tokenInfo.Audiences[0],
		ZoneId:    "123123-123123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	gormDB.Create(&tenantConfig)

	retrievedConfig, err := database.GetTenantConfigurationForContext(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedConfig)
	assert.Equal(t, tenantConfig.TenantID, retrievedConfig.TenantID)
}

func TestDatabase_GetTenantConfigurationForContextGetWebTokenFromContextReturnsError_ReturnsError(t *testing.T) {
	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	gormDB := setupSQLiteDB(t)
	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.Background()

	// patch commonsCtx.GetWebTokenFromContext(ctx) to return error
	patch := gomonkey.ApplyFunc(commonsCtx.GetWebTokenFromContext, func(ctx context.Context) (jwt.WebToken, error) {
		return jwt.WebToken{}, assert.AnError
	})

	defer patch.Reset()

	retrievedConfig, err := database.GetTenantConfigurationForContext(ctx)
	assert.Error(t, err)
	assert.Nil(t, retrievedConfig)
}

func TestDatabase_GetTenantConfigurationByIssuerAndAudience(t *testing.T) {
	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	gormDB := setupSQLiteDB(t)
	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.TODO()
	issuer := "https://accounts.sap.com"
	audiences := []string{"e3284ced-3a27-476b-9ae6-d5ad1ba05266"}

	// Insert test data
	tenantConfig := db.TenantConfiguration{
		TenantID:  uuid.New().String(),
		Issuer:    issuer,
		Audience:  audiences[0],
		ZoneId:    "123123-123123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	gormDB.Create(&tenantConfig)

	retrievedConfig, err := database.GetTenantConfigurationByIssuerAndAudience(ctx, issuer, audiences)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedConfig)
	assert.Equal(t, tenantConfig.TenantID, retrievedConfig.TenantID)
}

func TestDatabase_GetTenantConfigurationByIssuerAndAudienceDBFirstReturnsError_ReturnsError(t *testing.T) {
	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	gormDB := setupSQLiteDB(t)
	database, err := db.New(cfg, gormDB, log, false, false)
	assert.NoError(t, err)

	ctx := context.TODO()
	issuer := "https://accounts.sap.com"
	audiences := []string{"e3284ced-3a27-476b-9ae6-d5ad1ba05266"}

	retrievedConfig, err := database.GetTenantConfigurationByIssuerAndAudience(ctx, issuer, audiences)
	assert.Error(t, err)
	assert.Nil(t, retrievedConfig)
}

func TestDatabase_GetTenantConfigurationByIssuerAndAudienceDBFirstReturnsNoRows_ReturnsNil(t *testing.T) {
	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	gormDB := setupSQLiteDB(t)
	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.TODO()
	issuer := "https://accounts.sap.com"
	audiences := []string{"e3284ced-3a27-476b-9ae6-d5ad1ba05266"}

	// Insert test data
	tenantConfig := db.TenantConfiguration{
		TenantID:  uuid.New().String(),
		Issuer:    issuer,
		Audience:  "wrong_audience",
		ZoneId:    "123123-123123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	gormDB.Create(&tenantConfig)

	retrievedConfig, err := database.GetTenantConfigurationByIssuerAndAudience(ctx, issuer, audiences)
	assert.Nil(t, retrievedConfig)
	assert.NoError(t, err)
}
