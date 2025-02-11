package db_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

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

func getDbCfg() db.ConfigDatabase {
	return db.ConfigDatabase{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxConnLifetime: "1h",
	}
}

func TestUser_GetUserByID(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	result, err := database.GetUsersByUserIDs(ctx, tenantID, userIDs, 10, -2, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, len(userIDs), len(result))
}

func TestUser_GetUserByEmail(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

func Test_SearchUsers(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	seedUsers := []graph.User{
		// users that must appear in the result
		{TenantID: "tenant1", UserID: "JOHN"},          // capital letters should match
		{TenantID: "tenant1", Email: "john@gmail.com"}, // lower letters should match
		{TenantID: "tenant1", UserID: "userID1", FirstName: ptr.To("JOhn")},
		{TenantID: "tenant1", UserID: "userID2", LastName: ptr.To("jonson")},
		// users that must not appear in the result
		{TenantID: "tenant1", UserID: "Jann"},                                 // mistake in name should not match
		{TenantID: "tenant2", UserID: "John2"},                                // another tenantID should not match
		{TenantID: "tenant2", Email: "John2@gmail.com"},                       // another tenantID should not match
		{TenantID: "tenant2", UserID: "userID10", FirstName: ptr.To("John")},  // another tenantID should not match
		{TenantID: "tenant2", UserID: "userID20", LastName: ptr.To("Jonson")}, // another tenantID should not match
		{UserID: "John3"}, // if no tenantID, should not match
	}

	// Insert test data
	for _, user := range seedUsers {
		err = gormDB.Create(&user).Error
		require.NoError(t, err)
	}

	tests := []struct {
		name                 string
		limit                int
		expectedResultLength int
	}{
		{name: "check_all_possible_results", limit: 100, expectedResultLength: 4},
		{name: "check_if_limit_works", limit: 1, expectedResultLength: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database.SetConfig(db.ConfigDatabase{MaxSearchUsersLimit: tt.limit, MaxSearchUsersTimeout: 5000})
			result, err := database.SearchUsers(context.TODO(), "tenant1", "Jo")
			assert.NoError(t, err)
			require.Len(t, result, tt.expectedResultLength) // 4 users must be found
			for _, user := range result {                   // check that all users are from tenant1
				require.Equal(t, "tenant1", user.TenantID)
			}
		})
	}
}

func Test_GetUsersByUserIDs2(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	searchTerm := "test"
	result, err := database.GetUsersByUserIDs(ctx, tenantID, userIDs, 10, -2, &searchTerm, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, len(userIDs), len(result))
}

func Test_GetUsersByUserIDs3(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	searchTerm := "test"
	result, err := database.GetUsersByUserIDs(ctx, tenantID, userIDs, 10, -2, &searchTerm, &graph.SortByInput{Field: "user", Direction: "Asc"})
	assert.Error(t, err)
	assert.Nil(t, result)
}
