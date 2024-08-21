package db_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/openmfp/golang-commons/logger"
	"github.com/openmfp/iam-service/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestRole_BeforeCreate(t *testing.T) {
	t.Run("BeforeCreate", func(t *testing.T) {
		r := &db.Role{}
		err := r.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, r.ID)
	})
}

func TestRole_GetRolesForEntity_Success(t *testing.T) {
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
	entityType := "example"
	entityID := "exampleID"

	// Insert test data
	role := db.Role{
		ID:            uuid.New().String(),
		DisplayName:   "Test Role",
		TechnicalName: "test_role",
		EntityType:    entityType,
		EntityID:      entityID,
	}
	gormDB.Create(&role)

	roles, err := database.GetRolesForEntity(ctx, entityType, entityID)
	assert.NoError(t, err)
	assert.NotNil(t, roles)
	assert.Equal(t, 1, len(roles))
	assert.Equal(t, role.ID, roles[0].ID)
}

func TestRole_GetRolesByTechnicalNames_Success(t *testing.T) {
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
	entityType := "example"
	technicalNames := []string{"name1", "name2"}

	// Insert test data
	role1 := db.Role{
		ID:            uuid.New().String(),
		DisplayName:   "Role 1",
		TechnicalName: "name1",
		EntityType:    entityType,
	}
	role2 := db.Role{
		ID:            uuid.New().String(),
		DisplayName:   "Role 2",
		TechnicalName: "name2",
		EntityType:    entityType,
	}
	gormDB.Create(&role1)
	gormDB.Create(&role2)

	roles, err := database.GetRolesByTechnicalNames(ctx, entityType, technicalNames)
	assert.NoError(t, err)
	assert.NotNil(t, roles)
	assert.Equal(t, 2, len(roles))
}
