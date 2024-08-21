package db_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/openmfp/golang-commons/logger"
	"github.com/openmfp/iam-service/pkg/db"
	"github.com/openmfp/iam-service/pkg/db/mocks"
	"github.com/openmfp/iam-service/pkg/graph"
)

func TestUser_GetUserByID(t *testing.T) {
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

	ctx := context.TODO()
	tenantID := "tenant1"
	userID := uuid.New().String()

	// Insert test data
	user := graph.User{
		UserID:   userID,
		TenantID: tenantID,
		Email:    "test@example.com",
	}
	gormDB.Create(&user)

	result, err := database.GetUserByID(ctx, tenantID, userID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)
}

func TestUser_GetUsersByUserIDs(t *testing.T) {
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

	ctx := context.TODO()
	tenantID := "tenant1"
	userIDs := []string{uuid.New().String(), uuid.New().String()}

	// Insert test data
	for _, userID := range userIDs {
		user := graph.User{
			UserID:   userID,
			TenantID: tenantID,
			Email:    "test" + userID + "@example.com",
		}
		gormDB.Create(&user)
	}

	result, err := database.GetUsersByUserIDs(ctx, tenantID, userIDs, 10, -2)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, len(userIDs), len(result))
}

func TestUser_GetUserByEmail(t *testing.T) {
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

	ctx := context.TODO()
	tenantID := "tenant1"
	email := "test@example.com"

	// Insert test data
	user := graph.User{
		UserID:   uuid.New().String(),
		TenantID: tenantID,
		Email:    email,
	}
	gormDB.Create(&user)

	result, err := database.GetUserByEmail(ctx, tenantID, email)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, email, result.Email)
}

func TestUser_GetOrCreateUser(t *testing.T) {
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

	firstName := "Test"
	lastName := "User"

	ctx := context.TODO()
	tenantID := "tenant1"
	input := graph.UserInput{
		UserID:    uuid.New().String(),
		Email:     "test@example.com",
		FirstName: &firstName,
		LastName:  &lastName,
	}

	// Test creating a new user
	result, err := database.GetOrCreateUser(ctx, tenantID, input)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, input.Email, result.Email)

	// Test getting an existing user
	existingResult, err := database.GetOrCreateUser(ctx, tenantID, input)
	assert.NoError(t, err)
	assert.NotNil(t, existingResult)
	assert.Equal(t, result.UserID, existingResult.UserID)
}

func TestUser_GetOrCreateUser_CreateError(t *testing.T) {
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

	firstName := "Test"
	lastName := "User"

	ctx := context.TODO()
	tenantID := "tenant1"
	input := graph.UserInput{
		UserID:    uuid.New().String(),
		Email:     "test@example.com",
		FirstName: &firstName,
		LastName:  &lastName,
	}

	// monkey patch the delete method to return an error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Create", func(value interface{}) (tx *gorm.DB) {
		gormDB.Error = errors.New("delete error")
		return gormDB
	})
	defer patch.Reset()

	// Test creating a new user
	result, err := database.GetOrCreateUser(ctx, tenantID, input)
	assert.Error(t, err)
	assert.Nil(t, result)

}

func TestUser_GetOrCreateUser_Userhooks_Nil_Error(t *testing.T) {
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

	firstName := "Test"
	lastName := "User"

	ctx := context.TODO()
	tenantID := "tenant1"
	input := graph.UserInput{
		UserID:    uuid.New().String(),
		Email:     "test@example.com",
		FirstName: &firstName,
		LastName:  &lastName,
	}

	userHook := mocks.NewUserHooks(t)
	userHook.On("UserCreated", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	database.SetUserHooks(userHook)

	// Test creating a new user
	result, err := database.GetOrCreateUser(ctx, tenantID, input)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, input.Email, result.Email)
}

func TestUser_RemoveUser(t *testing.T) {
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

	userHook := mocks.NewUserHooks(t)

	ctx := context.Background()

	userHook.On("UserRemoved", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Act
	database.SetUserHooks(userHook)

	tenantID := "tenant1"
	userID := uuid.New().String()
	email := "test@example.com"

	// Insert test data
	user := graph.User{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
	}
	gormDB.Create(&user)

	// Test removing a user
	removed, err := database.RemoveUser(ctx, tenantID, userID, "")
	assert.NoError(t, err)
	assert.True(t, removed)
}

func TestUser_RemoveUserEmptyUserIDAndEmail(t *testing.T) {
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

	ctx := context.TODO()
	tenantID := "tenant1"

	// Test removing a user with no userID or email
	removed, err := database.RemoveUser(ctx, tenantID, "", "")
	assert.Error(t, err)
	assert.False(t, removed)
}

func TestUser_RemoveUserDBDeleteError(t *testing.T) {
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

	ctx := context.TODO()
	tenantID := "tenant1"
	userID := uuid.New().String()
	email := "test@example.com"

	// Insert test data
	user := graph.User{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
	}
	gormDB.Create(&user)

	// monkey patch the delete method to return an error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Delete", func(db *gorm.DB, value interface{}, conds ...interface{}) *gorm.DB {
		gormDB.Error = errors.New("delete error")
		return gormDB
	})

	defer patch.Reset()

	removed, err := database.RemoveUser(ctx, tenantID, userID, email)
	assert.Error(t, err)
	assert.False(t, removed)
}

func TestUser_RemoveUser_getUserByIDOrEmail_ErrRecordNotFound(t *testing.T) {
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

	ctx := context.TODO()
	tenantID := "tenant1"
	userID := uuid.New().String()
	email := "test@example.com"

	removed, err := database.RemoveUser(ctx, tenantID, userID, email)
	assert.Error(t, err)
	assert.True(t, removed)
}

func TestUser_GetUsers(t *testing.T) {
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

	ctx := context.TODO()
	tenantID := "tenant1"

	// Insert test data
	for i := 0; i < 5; i++ {
		user := graph.User{
			UserID:   uuid.New().String(),
			TenantID: tenantID,
			Email:    "test" + uuid.New().String() + "@example.com",
		}
		gormDB.Create(&user)
	}

	result, err := database.GetUsers(ctx, tenantID, 10, 1)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5, len(result.User))
}

func TestUser_GetUserByIdDBReturnsError(t *testing.T) {
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

	ctx := context.TODO()
	tenantID := "tenant1"
	userID := uuid.New().String()

	// Insert test data
	user := graph.User{
		UserID:   userID,
		TenantID: tenantID,
		Email:    "test@example.com",
	}
	gormDB.Create(&user)

	// Test getting a user that does not exist
	result, err := database.GetUserByID(ctx, tenantID, "non-existing-user")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUser_GetOrCreateUserEmptyUserIdAndEmail(t *testing.T) {
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

	firstName := "Test"
	lastName := "User"

	ctx := context.TODO()
	tenantID := "tenant1"
	input := graph.UserInput{
		Email:     "",
		FirstName: &firstName,
		LastName:  &lastName,
	}

	// Test creating a new user
	result, err := database.GetOrCreateUser(ctx, tenantID, input)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUser_GetUserByEmailEmptyEmail(t *testing.T) {
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

	ctx := context.TODO()
	tenantID := "tenant1"
	email := ""

	// Test getting a user that does not exist
	result, err := database.GetUserByEmail(ctx, tenantID, email)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUser_Save(t *testing.T) {
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

	firstName, lastName := "John", "Doe"
	err = database.Save(&graph.User{
		ID:        "1",
		UserID:    "test",
		TenantID:  "test",
		Email:     "test@nomail.com",
		FirstName: &firstName,
		LastName:  &lastName,
	})
	assert.NoError(t, err)
}
