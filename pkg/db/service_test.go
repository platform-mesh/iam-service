package db_test

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/pkg/errors"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/platform-mesh/iam-service/pkg/db"
	"github.com/platform-mesh/iam-service/pkg/db/mocks"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	return db
}

func TestNew_WhenSuccessful_CreatesDatabaseInstance(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	// Act
	database, err := db.New(cfg, gormDB, log, false, false)

	// Assert
	assert.NoError(t, err, "New should not return an error")
	assert.NotNil(t, database, "database instance should not be nil")
}

func TestNew_WhenMigrateEnabled_AutoMigrateIsCalled(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	autoMigrateCalled := false

	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "AutoMigrate", func(_ *gorm.DB, dst ...interface{}) error {
		autoMigrateCalled = true
		return nil // Simulate successful migration
	})

	defer patch.Reset()

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	// Act
	database, err := db.New(cfg, gormDB, log, true, false)

	// Assert
	assert.NoError(t, err, "New should not return an error")
	assert.NotNil(t, database, "database instance should not be nil")
	assert.True(t, autoMigrateCalled, "AutoMigrate should have been called")
}

func TestNew_WhenDBIsCalled_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	dbCalled := false
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "DB", func(*gorm.DB) (*sql.DB, error) {
		dbCalled = true
		return nil, assert.AnError
	})

	defer patch.Reset()

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	// Act
	database, err := db.New(cfg, gormDB, log, false, false)

	// Assert
	assert.Error(t, err, "New should return an error")
	assert.Nil(t, database, "database instance should be nil")
	assert.True(t, dbCalled, "DB should have been called")
}

func TestNew_CfgDSNotEmpty_ReturnsDBandNil(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		DSN: "postgres://user:password@localhost:5432/dbname",
	}

	// Act
	database, err := db.New(cfg, gormDB, log, false, false)

	// Assert
	assert.NoError(t, err, "New should not return an error")
	assert.NotNil(t, database, "database instance should not be nil")
}

func TestNew_AutoMigrateReturnsError_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "AutoMigrate", func(_ *gorm.DB, dst ...interface{}) error {
		return assert.AnError
	})

	defer patch.Reset()
	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	// Act
	database, err := db.New(cfg, gormDB, log, true, false)

	// Assert
	assert.Error(t, err, "New should return an error")
	assert.Nil(t, database, "database instance should be nil")
}

func TestLoadTenantConfigData_WhenSuccessful_ReturnsTenantConfigData(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathTenantConfiguration: "input/tenantConfigurations.yaml",
		},
	}

	loadTenantConfigDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		loadTenantConfigDataCalls++
		return []byte(`configs:
- tenantId: example-tenant
  issuer: https://issuer.my.corp
  audience: a2b50a84-f380-4c88-84d4-424059236cb3
  zoneId: 9b38c8d2-ee84-45c2-9e16-4ebaf811ca58
`), nil
	})

	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadTenantConfigData(cfg.LocalData.DataPathTenantConfiguration)

	// Assert
	assert.NoError(t, err, "LoadTenantConfigData should not return an error")
	assert.Equal(t, 1, loadTenantConfigDataCalls, "LoadTenantConfigData should have been called once")
}

func TestLoadTenantConfigData_Error(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathTenantConfiguration: "input/tenantConfigurations.yaml",
		},
	}

	loadTenantConfigDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		loadTenantConfigDataCalls++
		return []byte(`configs:
- tenantId: sap-btp
  issuer: https://accounts.sap.com
  audience: e3284ced-3a27-476b-9ae6-d5ad1ba05266
  zoneId: 123123-123123
- tenantId: example-tenant
  issuer: https://issuer.my.corp
  audience: a2b50a84-f380-4c88-84d4-424059236cb3
  zoneId: 9b38c8d2-ee84-45c2-9e16-4ebaf811ca58
- tenantId: hyperspacedev
  issuer: https://hyperspacedev.accounts.ondemand.com
  audience: f2cf17ca-5599-46f9-866b-fee5e8af96e8
  zoneId: 9b38c8d2-ee84-45c2-9e16-4ebaf811ca58
- tenantId: 29y87kiy4iakrkbb/test
  issuer: https://hyperspacedev.accounts.ondemand.com
  audience: f2cf17ca-5599-46f9-866b-fee5e8af96e8
  zoneId: 9b38c8d2-ee84-45c2-9e16-4ebaf811ca58
- tenantId: 29y87kiy4iakrkbb/test
  issuer: https://hyperspacedev.accounts.ondemand.com
  audience: f2cf17ca-5599-46f9-866b-fee5e8af96e8
  zoneId: 9b38c8d2-ee84-45c2-9e16-4ebaf811ca58
`), nil
	})

	defer patch.Reset()

	// monkey patch the delete method to return an error
	patchCreate := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Create", func(db *gorm.DB, value interface{}) *gorm.DB {
		gormDB.Error = errors.New("create error")
		return gormDB
	})

	defer patchCreate.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadTenantConfigData(cfg.LocalData.DataPathTenantConfiguration)

	// Assert
	assert.Error(t, err, "LoadTenantConfigData should return an error")
	assert.Equal(t, 1, loadTenantConfigDataCalls, "LoadTenantConfigData should have been called once")

}

func TestLoadTenantConfigData_FirstRowsAffected(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathTenantConfiguration: "input/tenantConfigurations.yaml",
		},
	}

	loadTenantConfigDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		loadTenantConfigDataCalls++
		return []byte(`configs:
- tenantId: sap-btp
  issuer: https://accounts.sap.com
  audience: e3284ced-3a27-476b-9ae6-d5ad1ba05266
  zoneId: 123123-123123
- tenantId: example-tenant
  issuer: https://issuer.my.corp
  audience: a2b50a84-f380-4c88-84d4-424059236cb3
  zoneId: 9b38c8d2-ee84-45c2-9e16-4ebaf811ca58
- tenantId: hyperspacedev
  issuer: https://hyperspacedev.accounts.ondemand.com
  audience: f2cf17ca-5599-46f9-866b-fee5e8af96e8
  zoneId: 9b38c8d2-ee84-45c2-9e16-4ebaf811ca58
- tenantId: 29y87kiy4iakrkbb/test
  issuer: https://hyperspacedev.accounts.ondemand.com
  audience: f2cf17ca-5599-46f9-866b-fee5e8af96e8
  zoneId: 9b38c8d2-ee84-45c2-9e16-4ebaf811ca58
`), nil
	})

	defer patch.Reset()

	// monkey patch the delete method to return an error
	patchCreate := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "First", func(db *gorm.DB, dest interface{}, conds ...interface{}) *gorm.DB {
		gormDB.RowsAffected = 1
		return gormDB
	})

	defer patchCreate.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadTenantConfigData(cfg.LocalData.DataPathTenantConfiguration)

	// Assert
	assert.NoError(t, err, "LoadTenantConfigData should not return an error")
	assert.Equal(t, 1, loadTenantConfigDataCalls, "LoadTenantConfigData should have been called once")

}

func TestLoadTenantConfigData_ReadFileFails_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		LocalData: db.DatabaseLocalData{
			DataPathTenantConfiguration: "input/tenantConfigurations.yaml",
		},
	}

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		return nil, assert.AnError
	})

	defer patch.Reset()
	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadTenantConfigData(cfg.LocalData.DataPathTenantConfiguration)

	// Assert
	assert.Error(t, err, "LoadTenantConfigData should return an error")
}

func TestLoadTenantConfigData_YamlUnmarshalFails_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		LocalData: db.DatabaseLocalData{
			DataPathTenantConfiguration: "input/tenantConfigurations.yaml",
		},
	}

	loadTenantConfigDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		loadTenantConfigDataCalls++
		return []byte(`invalid`), nil
	})

	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadTenantConfigData(cfg.LocalData.DataPathTenantConfiguration)

	// Assert
	assert.Error(t, err, "LoadTenantConfigData should return an error")
	assert.Equal(t, 1, loadTenantConfigDataCalls, "LoadTenantConfigData should have been called once")
}

func TestLoadTeamData_WhenSuccessful_ReturnsTeamData(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathTeam: "input/team.yaml",
			DataPathUser: "input/user.yaml",
		},
	}

	loadUserDataCalls := 0
	loadTeamDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		switch filename {
		case "input/team.yaml":
			loadTeamDataCalls++
			return []byte(`team:
  - tenantId: 29y87kiy4iakrkbb/test
    name: exampleTeam1`), nil
		case "input/user.yaml":
			loadUserDataCalls++
			return []byte(`user:
  - tenant_id: abc123456
    user_id: OOS6VEIL5I
    email: OOS6VEIL5I@sap.com
    first_name: zNameStartingWithZ
    groupsAssignments:
      - group: projectAdmins
        scope: exampleProject
        entity: project`), nil
		}

		return nil, nil
	})
	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	users, errUsers := database.LoadUserData(cfg.LocalData.DataPathUser)
	err = database.LoadTeamData(cfg.LocalData.DataPathTeam, users)

	// Assert
	assert.NoError(t, err, "LoadTeamData should not return an error")
	assert.NoError(t, errUsers, "LoadUserData should not return an error")
	assert.Equal(t, 1, loadUserDataCalls, "LoadUserData should have been called once")
	assert.Equal(t, 1, loadTeamDataCalls, "LoadTeamData should have been called once")

	// rows affected > 0
	patchCreate := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "First", func(db *gorm.DB, dest interface{}, conds ...interface{}) *gorm.DB {
		gormDB.RowsAffected = 1
		return gormDB
	})
	defer patchCreate.Reset()

	// Act
	err = database.LoadTeamData(cfg.LocalData.DataPathTeam, users)

	// Assert
	assert.NoError(t, err, "LoadTeamData should not return an error")
	assert.NoError(t, errUsers, "LoadUserData should not return an error")
	assert.Equal(t, 2, loadTeamDataCalls, "LoadTeamData should have been called once")

}

func TestLoadTeamData_ZeroUser_Error(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathTeam: "input/team.yaml",
			DataPathUser: "input/user.yaml",
		},
	}

	loadUserDataCalls := 0
	loadTeamDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		switch filename {
		case "input/user.yaml":
			loadUserDataCalls++
			return []byte(`user:`), nil
		case "input/team.yaml":
			loadTeamDataCalls++
			return []byte(`team:
  - tenantId: 29y87kiy4iakrkbb/test
    name: exampleTeam1`), nil
		}

		return nil, nil
	})
	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	users, errUsers := database.LoadUserData(cfg.LocalData.DataPathUser)
	err = database.LoadTeamData(cfg.LocalData.DataPathTeam, users)

	// Assert
	assert.NoError(t, err, "LoadTeamData should not return an error")
	assert.Error(t, errUsers)
	assert.Equal(t, 1, loadUserDataCalls, "LoadUserData should have been called once")
	assert.Equal(t, 1, loadTeamDataCalls, "LoadTeamData should have been called once")

}

func TestLoadTeamData_ZeroTeams(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathTeam: "input/team.yaml",
			DataPathUser: "input/user.yaml",
		},
	}

	loadUserDataCalls := 0
	loadTeamDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		switch filename {
		case "input/team.yaml":
			loadTeamDataCalls++
			return []byte(`team:`), nil
		case "input/user.yaml":
			loadUserDataCalls++
			return []byte(`user:
  - tenant_id: abc123456
    user_id: OOS6VEIL5I
    email: OOS6VEIL5I@sap.com
    first_name: zNameStartingWithZ
    groupsAssignments:
      - group: projectAdmins
        scope: exampleProject
        entity: project`), nil
		}

		return nil, nil
	})
	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	users, errUsers := database.LoadUserData(cfg.LocalData.DataPathUser)
	err = database.LoadTeamData(cfg.LocalData.DataPathTeam, users)

	// Assert
	assert.Error(t, err)
	assert.NoError(t, errUsers, "LoadUserData should not return an error")
	assert.Equal(t, 1, loadUserDataCalls, "LoadUserData should have been called once")
	assert.Equal(t, 1, loadTeamDataCalls, "LoadTeamData should have been called once")
}

func TestLoadTeamData_CreateError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathTeam: "input/team.yaml",
			DataPathUser: "input/user.yaml",
		},
	}

	loadUserDataCalls := 0
	loadTeamDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		switch filename {
		case "input/team.yaml":
			loadTeamDataCalls++
			return []byte(`team:
  - tenantId: 29y87kiy4iakrkbb/test
    name: exampleTeam1`), nil
		case "input/user.yaml":
			loadUserDataCalls++
			return []byte(`user:
  - tenant_id: abc123456
    user_id: OOS6VEIL5I
    email: OOS6VEIL5I@sap.com
    first_name: zNameStartingWithZ
    groupsAssignments:
      - group: projectAdmins
        scope: exampleProject
        entity: project`), nil
		}

		return nil, nil
	})
	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	users, errUsers := database.LoadUserData(cfg.LocalData.DataPathUser)

	// db.Create() error
	patchCreate := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Create", func(db *gorm.DB, val interface{}) *gorm.DB {
		gormDB.Error = assert.AnError
		return gormDB
	})
	defer patchCreate.Reset()

	// Act
	err = database.LoadTeamData(cfg.LocalData.DataPathTeam, users)

	// Assert
	assert.Error(t, err)
	assert.NoError(t, errUsers, "LoadUserData should not return an error")
	assert.Equal(t, 1, loadTeamDataCalls, "LoadTeamData should have been called once")

}

func TestLoadTeamData_ReadFileFails_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		LocalData: db.DatabaseLocalData{
			DataPathTeam: "input/team.yaml",
		},
	}

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		return nil, assert.AnError
	})

	defer patch.Reset()
	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadTeamData(cfg.LocalData.DataPathTeam, nil)

	// Assert
	assert.Error(t, err, "LoadTeamData should return an error")
}

func TestLoadTeamData_YamlUnmarshalFails_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		LocalData: db.DatabaseLocalData{
			DataPathTeam: "input/team.yaml",
		},
	}

	loadTeamDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		loadTeamDataCalls++
		return []byte(`invalid`), nil
	})

	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadTeamData(cfg.LocalData.DataPathTeam, nil)

	// Assert
	assert.Error(t, err, "LoadTeamData should return an error")
	assert.Equal(t, 1, loadTeamDataCalls, "LoadTeamData should have been called once")
}

func TestLoadInvitationData_WhenSuccessful_ReturnsInvitationData(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathInvitations: "input/invitations.yaml",
		},
	}

	loadInvitationDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		loadInvitationDataCalls++
		return []byte(`invitations:
  - tenantId: 29y87kiy4iakrkbb/test
    email: invited-admin-member@it.corp
    roles: owner,member
    entityType: project
    entityId: test`), nil
	})

	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadInvitationData(cfg.LocalData.DataPathInvitations)

	// Assert
	assert.NoError(t, err, "LoadInvitationData should not return an error")
	assert.Equal(t, 1, loadInvitationDataCalls, "LoadInvitationData should have been called once")
}

func TestLoadInvitationData_ReadFileFails_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		LocalData: db.DatabaseLocalData{
			DataPathInvitations: "input/invitations.yaml",
		},
	}

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		return nil, assert.AnError
	})

	defer patch.Reset()
	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadInvitationData(cfg.LocalData.DataPathInvitations)

	// Assert
	assert.Error(t, err, "LoadInvitationData should return an error")
}

func TestLoadInvitationData_YamlUnmarshalFails_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		LocalData: db.DatabaseLocalData{
			DataPathInvitations: "input/invitations.yaml",
		},
	}

	loadInvitationDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		loadInvitationDataCalls++
		return []byte(`invalid`), nil
	})

	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadInvitationData(cfg.LocalData.DataPathInvitations)

	// Assert
	assert.Error(t, err, "LoadInvitationData should return an error")
	assert.Equal(t, 1, loadInvitationDataCalls, "LoadInvitationData should have been called once")
}

func TestLoadRoleData_WhenSuccessful_ReturnsRoleData(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathRoles: "input/roles.yaml",
		},
	}

	loadRoleDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		loadRoleDataCalls++
		return []byte(`
- displayName: Owner
  technicalName: owner
  entityType: project`), nil
	})

	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadRoleData(cfg.LocalData.DataPathRoles)

	// Assert
	assert.NoError(t, err, "LoadRoleData should not return an error")
	assert.Equal(t, 1, loadRoleDataCalls, "LoadRoleData should have been called once")
}

func TestLoadRoleData_ReadFileFails_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		LocalData: db.DatabaseLocalData{
			DataPathRoles: "input/roles.yaml",
		},
	}

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		return nil, assert.AnError
	})

	defer patch.Reset()
	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadRoleData(cfg.LocalData.DataPathRoles)

	// Assert
	assert.Error(t, err, "LoadRoleData should return an error")
}

func TestLoadRoleData_YamlUnmarshalFails_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		LocalData: db.DatabaseLocalData{
			DataPathRoles: "input/roles.yaml",
		},
	}

	loadRoleDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		loadRoleDataCalls++
		return []byte(`invalid`), nil
	})

	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	err = database.LoadRoleData(cfg.LocalData.DataPathRoles)

	// Assert
	assert.Error(t, err, "LoadRoleData should return an error")
	assert.Equal(t, 1, loadRoleDataCalls, "LoadRoleData should have been called once")
}

func TestLoadUserData_SuccessfullLoadIsLocalData_ReturnsData(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathUser:                "./input/user.yaml",
			DataPathInvitations:         "./input/invitations.yaml",
			DataPathTeam:                "./input/team.yaml",
			DataPathTenantConfiguration: "./input/tenantConfigurations.yaml",
			DataPathDomainConfiguration: "./input/domainConfigurations.yaml",
			DataPathRoles:               "./input/roles.yaml",
		},
	}

	ctx := context.Background()

	// Act
	database, err := db.New(cfg, gormDB, log, true, true)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, database)

	user, err := database.GetUserByEmail(ctx, "tnt1234567", "USR547890@company.com")
	assert.NoError(t, err)
	assert.Equal(t, user.Email, "USR547890@company.com")
}

func TestLoadUserData_LoadIsLocalData_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathUser:                "./input/user.yaml",
			DataPathInvitations:         "./input/invitations.yaml",
			DataPathTeam:                "./input/team.yaml",
			DataPathTenantConfiguration: "./input/tenantConfigurations.yaml",
			DataPathDomainConfiguration: "./input/domainConfigurations.yaml",
			DataPathRoles:               "./input/roles.yaml",
		},
	}

	loadUserDataCalls := 0
	// patch database.LoadUserData to return an error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(&db.Database{}), "LoadUserData", func(*db.Database, string) ([]*graph.User, error) {
		loadUserDataCalls++
		return nil, assert.AnError
	})

	defer patch.Reset()

	// Act
	database, err := db.New(cfg, gormDB, log, true, true)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, database)

}

func TestLoadUserData_LoadInvitationData_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathUser:                "./input/user.yaml",
			DataPathInvitations:         "./input/invitations.yaml",
			DataPathTeam:                "./input/team.yaml",
			DataPathTenantConfiguration: "./input/tenantConfigurations.yaml",
			DataPathDomainConfiguration: "./input/domainConfigurations.yaml",
			DataPathRoles:               "./input/roles.yaml",
		},
	}

	loadInvitationDataCalls := 0
	// patch database.LoadInvitationData to return an error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(&db.Database{}), "LoadInvitationData", func(*db.Database, string) error {
		loadInvitationDataCalls++
		return assert.AnError
	})

	defer patch.Reset()

	// Act
	database, err := db.New(cfg, gormDB, log, true, true)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, database)

}

func TestLoadUserData_LoadTeamData_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathUser:                "./input/user.yaml",
			DataPathInvitations:         "./input/invitations.yaml",
			DataPathTeam:                "./input/team.yaml",
			DataPathTenantConfiguration: "./input/tenantConfigurations.yaml",
			DataPathDomainConfiguration: "./input/domainConfigurations.yaml",
			DataPathRoles:               "./input/roles.yaml",
		},
	}

	loadTeamDataCalls := 0
	// patch database.LoadTeamData to return an error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(&db.Database{}), "LoadTeamData", func(*db.Database, string, []*graph.User) error {
		loadTeamDataCalls++
		return assert.AnError
	})

	defer patch.Reset()

	// Act
	database, err := db.New(cfg, gormDB, log, true, true)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, database)

}

func TestLoadUserData_LoadTenantConfigData_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathUser:                "./input/user.yaml",
			DataPathInvitations:         "./input/invitations.yaml",
			DataPathTeam:                "./input/team.yaml",
			DataPathTenantConfiguration: "./input/tenantConfigurations.yaml",
			DataPathDomainConfiguration: "./input/domainConfigurations.yaml",
			DataPathRoles:               "./input/roles.yaml",
		},
	}

	loadTenantConfigDataCalls := 0
	// patch database.LoadTenantConfigData to return an error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(&db.Database{}), "LoadTenantConfigData", func(*db.Database, string) error {
		loadTenantConfigDataCalls++
		return assert.AnError
	})

	defer patch.Reset()

	// Act
	database, err := db.New(cfg, gormDB, log, true, true)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, database)

}

func TestLoadUserData_LoadRoleData_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathUser:                "./input/user.yaml",
			DataPathInvitations:         "./input/invitations.yaml",
			DataPathTeam:                "./input/team.yaml",
			DataPathTenantConfiguration: "./input/tenantConfigurations.yaml",
			DataPathDomainConfiguration: "./input/domainConfigurations.yaml",
			DataPathRoles:               "./input/roles.yaml",
		},
	}

	loadRoleDataCalls := 0
	// patch database.LoadRoleData to return an error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(&db.Database{}), "LoadRoleData", func(*db.Database, string) error {
		loadRoleDataCalls++
		return assert.AnError
	})

	defer patch.Reset()

	// Act
	database, err := db.New(cfg, gormDB, log, true, true)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, database)
}

func TestLoadUserData_EmptyFilePaths_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{}

	database, err := db.New(cfg, gormDB, log, true, true)
	assert.NoError(t, err)

	// Act
	user, err := database.LoadUserData("")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestLoadUserData_DBFirstReturnsNil_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
		LocalData: db.DatabaseLocalData{
			DataPathTeam: "input/team.yaml",
			DataPathUser: "input/user.yaml",
		},
	}

	loadUserDataCalls := 0
	loadTeamDataCalls := 0

	patch := gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		switch filename {
		case "input/team.yaml":
			loadTeamDataCalls++
			return []byte(`team:
  - tenantId: 29y87kiy4iakrkbb/test
    name: exampleTeam1`), nil
		case "input/user.yaml":
			loadUserDataCalls++
			return []byte(`user:
  - tenant_id: abc123456
    user_id: OOS6VEIL5I
    email: OOS6VEIL5I@sap.com
    first_name: zNameStartingWithZ
    groupsAssignments:
      - group: projectAdmins
        scope: exampleProject
        entity: project`), nil
		}

		return nil, nil
	})

	defer patch.Reset()

	database, err := db.New(cfg, gormDB, log, true, false)
	assert.NoError(t, err)

	// Act
	users, errUsers := database.LoadUserData(cfg.LocalData.DataPathUser)
	err = database.LoadTeamData(cfg.LocalData.DataPathTeam, users)

	// Assert
	assert.NoError(t, err, "LoadTeamData should not return an error")
	assert.NoError(t, errUsers, "LoadUserData should not return an error")
	assert.Equal(t, 1, loadUserDataCalls, "LoadUserData should have been called once")
	assert.Equal(t, 1, loadTeamDataCalls, "LoadTeamData should have been called once")
}

func TestSetUserHooks_WhenSuccessful_UpdateDatabaseHooks(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(db.ConfigDatabase{}, gormDB, log, false, false)
	assert.NoError(t, err)

	userHook := mocks.NewUserHooks(t)

	// Act
	database.SetUserHooks(userHook)

	// Assert
	assert.NotNil(t, database.GetUserHooks())
	assert.Equal(t, userHook, database.GetUserHooks())
}

func TestClose_WhenSuccessful_ClosesDatabaseConnection(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(db.ConfigDatabase{}, gormDB, log, false, false)
	assert.NoError(t, err)

	// Act
	err = database.Close()

	// Assert
	assert.NoError(t, err)
}

func TestClose_DB_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	// Act
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "DB", func(*gorm.DB) (*sql.DB, error) {
		return nil, assert.AnError
	})

	defer patch.Reset()

	database, err := db.New(db.ConfigDatabase{}, gormDB, log, false, false)

	// Assert
	assert.Nil(t, database)
	assert.Error(t, err)
}

func TestClose_DB_NilError(t *testing.T) {
	// test the Close method error handling by creating a database that will fail on Close()
	// we'll create a database, close the underlying connection, and then try to close again

	// Arrange
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(db.ConfigDatabase{}, gormDB, log, false, false)
	assert.NotNil(t, database)
	assert.NoError(t, err)

	sqlDB, err := gormDB.DB()
	assert.NoError(t, err)
	err = sqlDB.Close()
	assert.NoError(t, err)

	err = database.Close()

	assert.NoError(t, err)
}
