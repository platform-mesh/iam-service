package db_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/openmfp/golang-commons/logger"
	"github.com/openmfp/iam-service/pkg/db"
	"github.com/openmfp/iam-service/pkg/db/mocks"
	"github.com/openmfp/iam-service/pkg/graph"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestGetInvitesForEmail_WhenInvitesExist_ReturnsInvites(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(cfg, gormDB, log, true, false)

	assert.NoError(t, err)

	ctx := context.Background()

	// Act
	invites, err := database.GetInvitesForEmail(ctx, "tenantID", "email")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, invites)
}

func TestGetInvitesForEntity_WhenInvitesExist_ReturnsInvites(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(cfg, gormDB, log, true, false)

	assert.NoError(t, err)

	ctx := context.Background()

	// Act
	invites, err := database.GetInvitesForEntity(ctx, "tenantID", "entityType", "entityID")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, invites)
}

func TestInviteUser_WhenSuccessful_CreatesInvite(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(cfg, gormDB, log, true, false)

	assert.NoError(t, err)

	ctx := context.Background()

	invite := graph.Invite{
		Email: "email@email.test",
		Entity: &graph.EntityInput{
			EntityType: "entityType",
			EntityID:   "entityID",
		},
		Roles: []string{"role"},
	}

	// Act
	errInvite := database.InviteUser(ctx, "tenantID", invite, false)
	inviteForEmail, err := database.GetInvitesForEmail(ctx, "tenantID", "email@email.test")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, inviteForEmail)
	assert.Equal(t, invite.Email, inviteForEmail[0].Email)
	assert.Equal(t, invite.Entity.EntityType, inviteForEmail[0].EntityType)
	assert.Equal(t, invite.Entity.EntityID, inviteForEmail[0].EntityID)
	assert.Equal(t, invite.Roles[0], inviteForEmail[0].Roles)
	assert.NoError(t, errInvite)
}

func TestRemoveRoleFromInvite_WhenRoleExists_ReturnNil(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(cfg, gormDB, log, true, false)

	assert.NoError(t, err)

	ctx := context.Background()

	errRemoveRole := database.RemoveRoleFromInvite(ctx, db.Invite{
		TenantID: "tenantID", Email: "email@email.test", EntityType: "entityType", EntityID: "entityID",
	}, "role")

	assert.NoError(t, errRemoveRole)
	assert.Nil(t, errRemoveRole)
}

func TestRemoveRoleFromInvite_WhenDBThrowsError_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(cfg, gormDB, log, true, false)

	assert.NoError(t, err)

	ctx := context.Background()

	invite := graph.Invite{
		Email: "test@email.com",
		Entity: &graph.EntityInput{
			EntityType: "entityType",
			EntityID:   "entityID",
		},
		Roles: []string{"role"},
	}

	errInvite := database.InviteUser(ctx, "tenantID", invite, false)
	inviteForEmail, err := database.GetInvitesForEmail(ctx, "tenantID", "test@email.com")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(inviteForEmail))

	errRemoveRole := database.RemoveRoleFromInvite(ctx, db.Invite{
		TenantID: "tenantID", Email: "test@email.com", EntityType: "entityType", EntityID: "entityID",
	}, "role")

	gormDB.Error = errors.New("error")

	// Act
	err = database.RemoveRoleFromInvite(ctx, db.Invite{
		TenantID: "tenantID", Email: "test@email.com", EntityType: "entityType", EntityID: "entityID",
	}, "role")

	// Assert
	assert.Error(t, err)
	assert.NoError(t, errInvite)
	assert.NoError(t, errRemoveRole)
}

func TestRemoveRoleFromInvite_WhenRoleDoesNotExist_ReturnsNil(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(cfg, gormDB, log, true, false)

	assert.NoError(t, err)

	ctx := context.Background()

	invite := graph.Invite{
		Email: "email@email.test",
		Entity: &graph.EntityInput{
			EntityType: "entityType",
			EntityID:   "entityID",
		},
		Roles: []string{"role"},
	}

	errInvite := database.InviteUser(ctx, "tenantID", invite, false)
	inviteForEmail, err := database.GetInvitesForEmail(ctx, "tenantID", "email@email.test")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(inviteForEmail))

	errRemoveRole := database.RemoveRoleFromInvite(ctx, db.Invite{
		TenantID: "tenantID", Email: "email@email.test", EntityType: "entityType", EntityID: "entityID",
	}, "role2")
	// Act
	err = database.RemoveRoleFromInvite(ctx, db.Invite{
		TenantID: "tenantID", Email: "email@email.test", EntityType: "entityType", EntityID: "entityID",
	}, "role2")
	// Assert
	assert.NoError(t, err)
	assert.NoError(t, errInvite)
	assert.NoError(t, errRemoveRole)
}

func TestInviteUser_AlreadyInvitedWithNewRoles_UpdateInviteRoles(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(cfg, gormDB, log, true, false)

	assert.NoError(t, err)

	ctx := context.Background()

	invite := graph.Invite{
		Email: "email@email.test",
		Entity: &graph.EntityInput{
			EntityType: "entityType",
			EntityID:   "entityID",
		},
		Roles: []string{"role"},
	}

	errInvite := database.InviteUser(ctx, "tenantID", invite, false)
	assert.NoError(t, errInvite)
	inviteForEmail, err := database.GetInvitesForEmail(ctx, "tenantID", "email@email.test")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(inviteForEmail))
	assert.Equal(t, "role", inviteForEmail[0].Roles)

	invite.Roles = []string{"role", "newRole"}

	errInvite = database.InviteUser(ctx, "tenantID", invite, false)
	inviteForEmail, err = database.GetInvitesForEmail(ctx, "tenantID", "email@email.test")

	assert.NoError(t, err)
	assert.NotNil(t, inviteForEmail)
	assert.Equal(t, 1, len(inviteForEmail))
	assert.Equal(t, "role,newRole", inviteForEmail[0].Roles)
	assert.NoError(t, errInvite)
}

func TestInviteUser_DbFindThrowsError_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(cfg, gormDB, log, true, false)

	assert.NoError(t, err)

	ctx := context.Background()

	invite := graph.Invite{
		Email: "email@email.test",
		Entity: &graph.EntityInput{
			EntityType: "entityType",
			EntityID:   "entityID",
		},
		Roles: []string{"role"},
	}

	errInvite := database.InviteUser(ctx, "tenantID", invite, false)
	assert.NoError(t, errInvite)

	gormDB.Error = errors.New("error")

	// Act
	err = database.InviteUser(ctx, "tenantID", invite, false)

	// Assert
	assert.Error(t, err)
	assert.NoError(t, errInvite)
}

func TestInviteUser_UpdateInviteRolesThrowsError_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	// monkey Patch gormDB.Error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Update", func(*gorm.DB, string, interface{}) *gorm.DB {
		gormDB.Error = errors.New("error")
		return gormDB
	})

	defer patch.Reset()

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(cfg, gormDB, log, true, false)

	assert.NoError(t, err)

	ctx := context.Background()

	invite := graph.Invite{
		Email: "email@email.test",
		Entity: &graph.EntityInput{
			EntityType: "entityType",
			EntityID:   "entityID",
		},
		Roles: []string{"role"},
	}

	errInvite := database.InviteUser(ctx, "tenantID", invite, false)
	assert.NoError(t, errInvite)
	invite.Roles = []string{"newRole"}

	// Act
	err = database.InviteUser(ctx, "tenantID", invite, false)

	// Assert
	assert.Error(t, err)
	assert.NoError(t, errInvite)
}

func TestInviteUser_CreateInviteThrowsError_ReturnsError(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	// monkey Patch gormDB.Error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Create", func(*gorm.DB, interface{}) *gorm.DB {
		gormDB.Error = errors.New("error")
		return gormDB
	})

	defer patch.Reset()

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(cfg, gormDB, log, true, false)

	assert.NoError(t, err)

	ctx := context.Background()

	invite := graph.Invite{
		Email: "email@email.test",
		Entity: &graph.EntityInput{
			EntityType: "entityType",
			EntityID:   "entityID",
		},
		Roles: []string{"role"},
	}

	// Act
	err = database.InviteUser(ctx, "tenantID", invite, false)

	// Assert
	assert.Error(t, err)
}

func TestInviteUser_UserHooksUserInvited_ShouldBeCalled(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	// monkey Patch gormDB.Error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Create", func(*gorm.DB, interface{}) *gorm.DB {
		return gormDB
	})

	defer patch.Reset()

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	mockUserHook := mocks.NewUserHooks(t)

	mockUserHook.On("UserInvited", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	database, err := db.New(cfg, gormDB, log, true, false)
	database.SetUserHooks(mockUserHook)

	assert.NoError(t, err)

	ctx := context.Background()

	invite := graph.Invite{
		Email: "email@email.test",
		Entity: &graph.EntityInput{
			EntityType: "entityType",
			EntityID:   "entityID",
		},
		Roles: []string{"role"},
	}

	// Act
	err = database.InviteUser(ctx, "tenantID", invite, false)

	// Assert
	assert.NoError(t, err)
	mockUserHook.AssertExpectations(t)
}

func TestDeleteInvitesForEmail_WhenInvitesExist_ReturnsNil(t *testing.T) {
	// Arrange
	gormDB := setupSQLiteDB(t)

	cfg := db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(cfg, gormDB, log, true, false)

	assert.NoError(t, err)

	ctx := context.Background()

	invite := graph.Invite{
		Email: "email@email.test",
		Entity: &graph.EntityInput{
			EntityType: "entityType",
			EntityID:   "entityID",
		},
		Roles: []string{"role"},
	}

	errInvite := database.InviteUser(ctx, "tenantID", invite, false)
	assert.NoError(t, errInvite)

	// Act
	err = database.DeleteInvitesForEmail(ctx, "tenantID", "email@email.test")

	// Assert
	assert.NoError(t, err)
}
