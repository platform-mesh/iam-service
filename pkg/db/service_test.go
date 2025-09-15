package db_test

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/platform-mesh/golang-commons/logger"

	"github.com/platform-mesh/iam-service/pkg/db"
	"github.com/platform-mesh/iam-service/pkg/graph"
)

// Mock implementation of UserHooks for testing
type MockUserHooks struct{}

func (m *MockUserHooks) UserInvited(ctx context.Context, user *graph.User, tenantID string, scope string, userInvited bool) {
}
func (m *MockUserHooks) UserCreated(ctx context.Context, user *graph.User, tenantID string) {}
func (m *MockUserHooks) UserRemoved(ctx context.Context, user *graph.User, tenantID string) {}
func (m *MockUserHooks) UserLogin(ctx context.Context, user *graph.User, tenantID string)   {}

func TestService_LoadTenantConfigData_FileNotFound(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Test with non-existent file
	err = database.LoadTenantConfigData("non-existent-file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error occurred when reading data file")
}

func TestService_LoadTenantConfigData_InvalidYAML(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with invalid YAML
	tmpFile, err := os.CreateTemp("", "invalid-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = tmpFile.WriteString("invalid yaml content [")
	require.NoError(t, err)
	_ = tmpFile.Close()

	err = database.LoadTenantConfigData(tmpFile.Name())
	assert.Error(t, err)
}

func TestService_LoadTenantConfigData_CreateError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with valid YAML
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `configs:
  - tenant_id: "test-tenant"
    issuer: "test-issuer"
    audience: "test-audience"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Monkey patch the Create method to return error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Create", func(tx *gorm.DB, value interface{}) *gorm.DB {
		gormDB.Error = errors.New("create error")
		return gormDB
	})
	defer patch.Reset()

	err = database.LoadTenantConfigData(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create error")
}

func TestService_LoadTeamData_FileNotFound(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	err = database.LoadTeamData("non-existent-file.yaml", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error occurred when reading data file")
}

func TestService_LoadTeamData_InvalidYAML(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with invalid YAML
	tmpFile, err := os.CreateTemp("", "invalid-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = tmpFile.WriteString("invalid yaml content [")
	require.NoError(t, err)
	_ = tmpFile.Close()

	err = database.LoadTeamData(tmpFile.Name(), nil)
	assert.Error(t, err)
}

func TestService_LoadTeamData_CreateError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with valid YAML
	tmpFile, err := os.CreateTemp("", "teams-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `team:
  - id: "team1"
    tenantId: "tenant1" 
    name: "Test Team"
    displayName: "Test Team"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Monkey patch the Create method to return error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Create", func(tx *gorm.DB, value interface{}) *gorm.DB {
		gormDB.Error = errors.New("create error")
		return gormDB
	})
	defer patch.Reset()

	err = database.LoadTeamData(tmpFile.Name(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create error")
}

func TestService_LoadTeamData_NoTeamsLoaded(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with empty teams
	tmpFile, err := os.CreateTemp("", "teams-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `team: []`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	err = database.LoadTeamData(tmpFile.Name(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no teams where loaded into the DB")
}

func TestService_LoadUserData_FileNotFound(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	users, err := database.LoadUserData("non-existent-file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error occurred when reading data file")
	assert.Nil(t, users)
}

func TestService_LoadUserData_InvalidYAML(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with invalid YAML
	tmpFile, err := os.CreateTemp("", "invalid-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = tmpFile.WriteString("invalid yaml content [")
	require.NoError(t, err)
	_ = tmpFile.Close()

	users, err := database.LoadUserData(tmpFile.Name())
	assert.Error(t, err)
	assert.Nil(t, users)
}

func TestService_LoadUserData_CreateError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with valid YAML
	tmpFile, err := os.CreateTemp("", "users-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `user:
  - id: "user1"
    tenant_id: "tenant1"
    user_id: "test-user"
    email: "test@example.com"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Monkey patch the Create method to return error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Create", func(tx *gorm.DB, value interface{}) *gorm.DB {
		gormDB.Error = errors.New("create error")
		return gormDB
	})
	defer patch.Reset()

	users, err := database.LoadUserData(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create error")
	assert.Nil(t, users)
}

func TestService_LoadUserData_NoUsersLoaded(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with empty users
	tmpFile, err := os.CreateTemp("", "users-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `user: []`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	users, err := database.LoadUserData(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no users where loaded into the DB")
	assert.Nil(t, users)
}

func TestService_LoadInvitationData_FileNotFound(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	err = database.LoadInvitationData("non-existent-file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error occurred when reading data file")
}

func TestService_LoadInvitationData_InvalidYAML(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with invalid YAML
	tmpFile, err := os.CreateTemp("", "invalid-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = tmpFile.WriteString("invalid yaml content [")
	require.NoError(t, err)
	_ = tmpFile.Close()

	err = database.LoadInvitationData(tmpFile.Name())
	assert.Error(t, err)
}

func TestService_LoadInvitationData_CreateError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with valid YAML
	tmpFile, err := os.CreateTemp("", "invites-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `invitations:
  - tenant_id: "tenant1"
    email: "test@example.com"
    entity_type: "team"
    entity_id: "team1"
    roles: "admin"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Monkey patch the Create method to return error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Create", func(tx *gorm.DB, value interface{}) *gorm.DB {
		gormDB.Error = errors.New("create error")
		return gormDB
	})
	defer patch.Reset()

	err = database.LoadInvitationData(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create error")
}

func TestService_LoadInvitationData_NoInvitationsLoaded(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with empty invitations
	tmpFile, err := os.CreateTemp("", "invites-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `invitations: []`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	err = database.LoadInvitationData(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no invitations where loaded into the DB")
}

func TestService_LoadRoleData_EmptyFilePath(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Test with empty file path - should return nil
	err = database.LoadRoleData("")
	assert.NoError(t, err)
}

func TestService_LoadRoleData_FileNotFound(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	err = database.LoadRoleData("non-existent-file.yaml")
	assert.Error(t, err)
}

func TestService_LoadRoleData_InvalidYAML(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with invalid YAML
	tmpFile, err := os.CreateTemp("", "invalid-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = tmpFile.WriteString("invalid yaml content [")
	require.NoError(t, err)
	_ = tmpFile.Close()

	err = database.LoadRoleData(tmpFile.Name())
	assert.Error(t, err)
}

func TestService_LoadRoleData_FirstError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with valid YAML
	tmpFile, err := os.CreateTemp("", "roles-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `- technical_name: "admin"
  display_name: "Administrator"
  entity_type: "team"
  entity_id: "team1"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Monkey patch the First method to return a non-ErrRecordNotFound error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "First", func(tx *gorm.DB, dest interface{}, conds ...interface{}) *gorm.DB {
		gormDB.Error = errors.New("database error")
		return gormDB
	})
	defer patch.Reset()

	err = database.LoadRoleData(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestService_LoadRoleData_SaveError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create existing role
	existingRole := db.Role{
		TechnicalName: "admin",
		EntityType:    "team",
		DisplayName:   "Old Admin",
	}
	gormDB.Create(&existingRole)

	// Create temp file with valid YAML
	tmpFile, err := os.CreateTemp("", "roles-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `- technical_name: "admin"
  display_name: "New Administrator"
  entity_type: "team"
  entity_id: "team1"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Monkey patch the Save method to return error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Save", func(tx *gorm.DB, value interface{}) *gorm.DB {
		gormDB.Error = errors.New("save error")
		return gormDB
	})
	defer patch.Reset()

	err = database.LoadRoleData(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save error")
}

func TestService_LoadRoleData_CreateError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with valid YAML
	tmpFile, err := os.CreateTemp("", "roles-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `- technical_name: "admin"
  display_name: "Administrator"
  entity_type: "team"
  entity_id: "team1"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Monkey patch the Create method to return error after First returns ErrRecordNotFound
	var callCount int
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "First", func(tx *gorm.DB, dest interface{}, conds ...interface{}) *gorm.DB {
		gormDB.Error = gorm.ErrRecordNotFound
		gormDB.RowsAffected = 0
		return gormDB
	})
	defer patch.Reset()

	patch2 := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Create", func(tx *gorm.DB, value interface{}) *gorm.DB {
		callCount++
		if callCount > 0 {
			gormDB.Error = errors.New("create error")
		}
		return gormDB
	})
	defer patch2.Reset()

	err = database.LoadRoleData(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create error")
}

func TestService_GetTenantConfigurationForContext(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.TODO()

	// Test with context that doesn't have tenant config - should return error
	config, err := database.GetTenantConfigurationForContext(ctx)
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestService_Close(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	err = database.Close()
	assert.NoError(t, err)
}

func TestService_Close_DBError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Monkey patch the DB method to return error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "DB", func(db *gorm.DB) (*sql.DB, error) {
		return nil, errors.New("db connection error")
	})
	defer patch.Reset()

	err = database.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db connection error")
}

func TestService_New_DBConnectionError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	// Monkey patch the DB method to return error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "DB", func(db *gorm.DB) (*sql.DB, error) {
		return nil, errors.New("connection error")
	})
	defer patch.Reset()

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Error connecting to database")
	assert.Nil(t, database)
}

func TestService_New_MigrationError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	// Monkey patch the AutoMigrate method to return error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "AutoMigrate", func(db *gorm.DB, dst ...interface{}) error {
		return errors.New("migration error")
	})
	defer patch.Reset()

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to migrate model")
	assert.Nil(t, database)
}

func TestService_New_WithConnectionPoolSettings(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := getDbCfg()
	cfg.MaxOpenConns = 10
	cfg.MaxIdleConns = 5
	cfg.MaxConnLifetime = "1h"

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)
	assert.NotNil(t, database)

	err = database.Close()
	assert.NoError(t, err)
}

func TestService_New_InvalidConnLifetime(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := getDbCfg()
	cfg.MaxConnLifetime = "invalid-duration"

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)
	assert.NotNil(t, database)

	err = database.Close()
	assert.NoError(t, err)
}

func TestService_New_LocalMode_LoadDataErrors(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := getDbCfg()
	cfg.LocalData = db.DatabaseLocalData{
		DataPathUser:                "non-existent-users.yaml",
		DataPathInvitations:         "non-existent-invitations.yaml",
		DataPathTeam:                "non-existent-teams.yaml",
		DataPathTenantConfiguration: "non-existent-config.yaml",
		DataPathRoles:               "non-existent-roles.yaml",
	}

	// Should still create database successfully even if local data loading fails
	database, err := db.New(cfg, gormDB, log, true, true)
	assert.NoError(t, err)
	assert.NotNil(t, database)

	err = database.Close()
	assert.NoError(t, err)
}

func TestService_SetUserHooks(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Test getting hooks when none are set
	hooks := database.GetUserHooks()
	assert.Nil(t, hooks)

	// Mock hooks for testing
	mockHooks := &MockUserHooks{}
	database.SetUserHooks(mockHooks)

	retrievedHooks := database.GetUserHooks()
	assert.Equal(t, mockHooks, retrievedHooks)
}

func TestService_SetConfig(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	newCfg := db.ConfigDatabase{
		MaxOpenConns:    20,
		MaxIdleConns:    10,
		MaxConnLifetime: "2h",
	}

	database.SetConfig(newCfg)

	// We can't directly test the config was set, but we can ensure no error occurred
	assert.NotNil(t, database)
}

func TestService_GetGormDB(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	retrievedDB := database.GetGormDB()
	assert.Equal(t, gormDB, retrievedDB)
}

func TestService_LoadTeamData_WithParentTeam(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with teams that have parent-child relationships
	tmpFile, err := os.CreateTemp("", "teams-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `team:
  - id: "parent-team"
    tenantId: "tenant1"
    name: "Parent Team"
    displayName: "Parent Team"
  - id: "child-team"
    tenantId: "tenant1"
    name: "Child Team"
    displayName: "Child Team"
    parentTeam: "Parent Team"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	err = database.LoadTeamData(tmpFile.Name(), nil)
	assert.NoError(t, err)
}

func TestService_LoadTeamData_ExistingTeam(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create existing team
	existingTeam := &graph.Team{
		Name:     "Existing Team",
		TenantID: "tenant1",
	}
	gormDB.Create(&existingTeam)

	// Create temp file with same team
	tmpFile, err := os.CreateTemp("", "teams-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `team:
  - id: "team1"
    tenantId: "tenant1"
    name: "Existing Team"
    displayName: "Existing Team"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	err = database.LoadTeamData(tmpFile.Name(), nil)
	assert.NoError(t, err)
}

func TestService_LoadUserData_ExistingUser(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create existing user
	existingUser := &db.User{
		UserID:   "existing-user",
		TenantID: "tenant1",
		Email:    "existing@example.com",
	}
	gormDB.Create(&existingUser)

	// Create temp file with same user
	tmpFile, err := os.CreateTemp("", "users-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `user:
  - id: "user1"
    tenant_id: "tenant1"
    user_id: "existing-user"
    email: "existing@example.com"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	users, err := database.LoadUserData(tmpFile.Name())
	assert.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestService_LoadInvitationData_ExistingInvitation(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create existing invitation
	existingInvite := &db.Invite{
		Email:      "existing@example.com",
		TenantID:   "tenant1",
		EntityType: "team",
		EntityID:   "team1",
		Roles:      "admin",
	}
	gormDB.Create(&existingInvite)

	// Create temp file with same invitation
	tmpFile, err := os.CreateTemp("", "invites-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `invitations:
  - tenant_id: "tenant1"
    email: "existing@example.com"
    entity_type: "team"
    entity_id: "team1"
    roles: "admin"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	err = database.LoadInvitationData(tmpFile.Name())
	assert.NoError(t, err)
}

func TestService_LoadRoleData_Success(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create temp file with valid YAML
	tmpFile, err := os.CreateTemp("", "roles-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `- technical_name: "admin"
  display_name: "Administrator"
  entity_type: "team"
  entity_id: "team1"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	err = database.LoadRoleData(tmpFile.Name())
	assert.NoError(t, err)
}

func TestService_LoadRoleData_UpdateExistingRole(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Create existing role
	existingRole := db.Role{
		TechnicalName: "admin",
		EntityType:    "team",
		DisplayName:   "Old Admin",
	}
	gormDB.Create(&existingRole)

	// Create temp file with updated role
	tmpFile, err := os.CreateTemp("", "roles-*.yaml")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	yamlContent := `- technical_name: "admin"
  display_name: "New Administrator"
  entity_type: "team"
  entity_id: "team1"`

	_, err = tmpFile.WriteString(yamlContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	err = database.LoadRoleData(tmpFile.Name())
	assert.NoError(t, err)
}
