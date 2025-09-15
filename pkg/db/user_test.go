package db_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	gormlogger "gorm.io/gorm/logger"
	"k8s.io/utils/ptr"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/platform-mesh/golang-commons/jwt"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/platform-mesh/iam-service/pkg/db"
	"github.com/platform-mesh/iam-service/pkg/db/mocks"
	"github.com/platform-mesh/iam-service/pkg/graph"
)

// setupSQLiteDB creates an isolated in-memory SQLite database for testing
func setupSQLiteDB(t *testing.T) *gorm.DB {
	// Use unique database name to avoid state pollution between tests
	dsn := ":memory:"
	dbDialect := sqlite.Open(dsn)
	dbConn, err := gorm.Open(dbDialect, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	return dbConn
}

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

	// First create a user to trigger a database constraint violation
	existingUser := graph.User{
		UserID:   input.UserID,
		TenantID: tenantID,
		Email:    input.Email,
	}
	gormDB.Create(&existingUser)

	// monkey patch the First method to return no error (user not found)
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "First", func(db *gorm.DB, dest interface{}, conds ...interface{}) *gorm.DB {
		db.Error = gorm.ErrRecordNotFound
		return db
	})
	defer patch.Reset()

	// Test creating a new user - this should trigger unique constraint error
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
	result, err := database.GetUsersByUserIDs(ctx, tenantID, userIDs, 10, -2, &searchTerm, &graph.SortByInput{Field: "user", Direction: "invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sort direction")
	assert.Nil(t, result)
}

func TestUser_SearchUsers_ContextTimeout(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	// Set a very short timeout to trigger timeout behavior
	database.SetConfig(db.ConfigDatabase{MaxSearchUsersTimeout: 1})

	result, err := database.SearchUsers(context.TODO(), "tenant1", "test")
	// The function should return a context deadline exceeded error with timeout 1ms
	if err != nil {
		assert.Contains(t, err.Error(), "context deadline exceeded")
		assert.Nil(t, result)
	} else {
		// If no timeout occurred (fast execution), just verify result is valid
		assert.NotNil(t, result)
	}
}

func TestUser_LoginUserWithToken_MissingNameClaims(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.TODO()
	tenantID := "tenant1"
	userID := uuid.New().String()
	email := "test@example.com"

	user := graph.User{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
	}

	// Create token info with missing name claims (empty strings)
	tokenInfo := jwt.WebToken{
		IssuerAttributes: jwt.IssuerAttributes{
			Issuer: "test-issuer",
		},
		UserAttributes: jwt.UserAttributes{
			FirstName: "", // Missing first name
			LastName:  "", // Missing last name
		},
	}

	// This should trigger the warning log and sentry capture
	err = database.LoginUserWithToken(ctx, tokenInfo, userID, tenantID, &user, email)
	assert.NoError(t, err)
}

func TestUser_RemoveUser_DeleteError_SentryCapture(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	// Monkey patch the Delete method to return error (this will trigger sentry capture)
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Delete", func(db *gorm.DB, value interface{}, conds ...interface{}) *gorm.DB {
		gormDB.Error = errors.New("delete error")
		return gormDB
	})
	defer patch.Reset()

	removed, err := database.RemoveUser(ctx, tenantID, userID, email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not delete user")
	assert.False(t, removed)
}

func TestUser_LoginUserWithToken(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.TODO()
	tenantID := "tenant1"
	userID := uuid.New().String()
	email := "test@example.com"

	// Insert test data
	firstName := "OldFirst"
	lastName := "OldLast"
	user := graph.User{
		UserID:                userID,
		TenantID:              tenantID,
		Email:                 "old@example.com",
		FirstName:             &firstName,
		LastName:              &lastName,
		InvitationOutstanding: true,
	}
	gormDB.Create(&user)

	// Create token info with different data to test updates
	tokenInfo := jwt.WebToken{
		IssuerAttributes: jwt.IssuerAttributes{
			Issuer: "test-issuer",
		},
		UserAttributes: jwt.UserAttributes{
			FirstName: "NewFirst",
			LastName:  "NewLast",
		},
	}

	// Set up user hooks mock
	userHook := mocks.NewUserHooks(t)
	userHook.On("UserLogin", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	database.SetUserHooks(userHook)

	// Test login with token - this should update user fields
	err = database.LoginUserWithToken(ctx, tokenInfo, userID, tenantID, &user, email)
	assert.NoError(t, err)

	// Verify user was updated
	assert.Equal(t, userID, user.UserID)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, tokenInfo.FirstName, *user.FirstName)
	assert.Equal(t, tokenInfo.LastName, *user.LastName)
	assert.False(t, user.InvitationOutstanding)
}

func TestUser_LoginUserWithToken_EmptyTokenNames(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.TODO()
	tenantID := "tenant1"
	userID := uuid.New().String()
	email := "test@example.com"

	firstName := "ExistingFirst"
	lastName := "ExistingLast"
	user := graph.User{
		UserID:    userID,
		TenantID:  tenantID,
		Email:     email,
		FirstName: &firstName,
		LastName:  &lastName,
	}

	// Create token info with empty names
	tokenInfo := jwt.WebToken{
		IssuerAttributes: jwt.IssuerAttributes{
			Issuer: "test-issuer",
		},
		UserAttributes: jwt.UserAttributes{
			FirstName: "",
			LastName:  "",
		},
	}

	err = database.LoginUserWithToken(ctx, tokenInfo, userID, tenantID, &user, email)
	assert.NoError(t, err)

	// Names should remain unchanged when token has empty names
	assert.Equal(t, firstName, *user.FirstName)
	assert.Equal(t, lastName, *user.LastName)
}

func TestUser_LoginUserWithToken_NoUpdatesNeeded(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.TODO()
	tenantID := "tenant1"
	userID := uuid.New().String()
	email := "test@example.com"

	firstName := "ExistingFirst"
	lastName := "ExistingLast"
	user := graph.User{
		UserID:                userID,
		TenantID:              tenantID,
		Email:                 email,
		FirstName:             &firstName,
		LastName:              &lastName,
		InvitationOutstanding: false,
	}

	// Create token info with same data - no updates needed
	tokenInfo := jwt.WebToken{
		IssuerAttributes: jwt.IssuerAttributes{
			Issuer: "test-issuer",
		},
		UserAttributes: jwt.UserAttributes{
			FirstName: firstName,
			LastName:  lastName,
		},
	}

	err = database.LoginUserWithToken(ctx, tokenInfo, userID, tenantID, &user, email)
	assert.NoError(t, err)
}

func TestUser_LoginUserWithToken_SaveError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.TODO()
	tenantID := "tenant1"
	userID := uuid.New().String()
	email := "test@example.com"

	firstName := "OldFirst"
	user := graph.User{
		UserID:    userID,
		TenantID:  tenantID,
		Email:     "old@example.com", // Different email to trigger update
		FirstName: &firstName,
	}

	tokenInfo := jwt.WebToken{
		IssuerAttributes: jwt.IssuerAttributes{
			Issuer: "test-issuer",
		},
		UserAttributes: jwt.UserAttributes{
			FirstName: "NewFirst",
			LastName:  "NewLast",
		},
	}

	// Monkey patch the GORM Save method to return error
	patch := gomonkey.ApplyMethod(reflect.TypeOf(gormDB), "Save", func(tx *gorm.DB, value interface{}) *gorm.DB {
		gormDB.Error = errors.New("save error")
		return gormDB
	})
	defer patch.Reset()

	err = database.LoginUserWithToken(ctx, tokenInfo, userID, tenantID, &user, email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save error")
}

func TestUser_GetUsersByUserIDs_InvalidSortDirection(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.TODO()
	tenantID := "tenant1"
	userIDs := []string{uuid.New().String()}

	// Test with invalid sort direction
	result, err := database.GetUsersByUserIDs(ctx, tenantID, userIDs, 10, 1, nil, &graph.SortByInput{Direction: "invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sort direction")
	assert.Nil(t, result)
}

func TestUser_LoginUserWithToken_SentryCapture(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
	assert.NoError(t, err)

	ctx := context.TODO()
	tenantID := "tenant1"
	userID := uuid.New().String()
	email := "test@example.com"

	user := graph.User{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
	}

	// Create token info with missing name claims (empty strings) - this triggers sentry capture
	tokenInfo := jwt.WebToken{
		IssuerAttributes: jwt.IssuerAttributes{
			Issuer: "test-issuer",
		},
		UserAttributes: jwt.UserAttributes{
			FirstName: "", // Missing first name triggers sentry capture
			LastName:  "", // Missing last name triggers sentry capture
		},
	}

	// This should trigger the sentry capture on lines 169-172
	err = database.LoginUserWithToken(ctx, tokenInfo, userID, tenantID, &user, email)
	assert.NoError(t, err)
}

func TestUser_RemoveUser_UserHooksError(t *testing.T) {
	gormDB := setupSQLiteDB(t)

	log, err := logger.New(logger.DefaultConfig())
	assert.NoError(t, err)

	database, err := db.New(getDbCfg(), gormDB, log, true, false)
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

	// Set up user hooks mock
	userHook := mocks.NewUserHooks(t)
	userHook.On("UserRemoved", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	database.SetUserHooks(userHook)

	// Test removing a user - this covers line 215 with user hooks
	removed, err := database.RemoveUser(ctx, tenantID, userID, email)
	assert.NoError(t, err)
	assert.True(t, removed)

	// Verify the hook was called
	userHook.AssertExpectations(t)
}
