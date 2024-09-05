package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	openmfpCtx "github.com/openmfp/golang-commons/context"

	"github.com/openmfp/iam-service/pkg/db"
	"github.com/openmfp/iam-service/pkg/db/mocks"
	"github.com/openmfp/iam-service/pkg/graph"
	"github.com/openmfp/iam-service/pkg/service"
)

func setupService(t *testing.T) (*service.Service, *mocks.DatabaseService) {
	t.Helper()
	mockDb := mocks.NewDatabaseService(t)
	svc := &service.Service{
		Db: mockDb,
	}
	return svc, mockDb
}

func Test_InviteUser_Success(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	email := "user@foo.com"
	entityType := "project"
	entityId := "projectID123"
	roles := []string{
		"admin",
	}
	tenantID := "tenantID123"
	notifyByEmail := true

	invite := graph.Invite{
		Email: email,
		Entity: &graph.EntityInput{
			EntityType: entityType,
			EntityID:   entityId,
		},
		Roles: roles,
	}

	// mock, expect
	userInput := graph.UserInput{
		Email: email,
	}
	mockDb.EXPECT().GetOrCreateUser(ctx, tenantID, userInput).Return(nil, nil).Once()
	mockDb.EXPECT().InviteUser(ctx, tenantID, invite, notifyByEmail).Return(nil).Once()

	success, err := service.InviteUser(ctx, tenantID, invite, notifyByEmail)

	// asserts
	assert.NoError(t, err)
	assert.True(t, success)
}

func Test_InviteUser_Error(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	email := "user@foo.com"
	entityType := "project"
	entityId := "projectID123"
	roles := []string{
		"admin",
	}
	tenantID := "tenantID123"
	notifyByEmail := true

	invite := graph.Invite{
		Email: email,
		Entity: &graph.EntityInput{
			EntityType: entityType,
			EntityID:   entityId,
		},
		Roles: roles,
	}

	// ERROR case: GetOrCreateUser return error
	userInput := graph.UserInput{
		Email: email,
	}
	mockDb.EXPECT().GetOrCreateUser(ctx, tenantID, userInput).Return(nil, errors.New("mock error")).Once()
	// mockDb.EXPECT().InviteUser(ctx, tenantID, invite, notifyByEmail).Return(nil).Times(0)

	success, err := service.InviteUser(ctx, tenantID, invite, notifyByEmail)

	// asserts
	assert.Error(t, err)
	assert.False(t, success)

	// ERROR case: InviteUser return error
	mockDb.EXPECT().GetOrCreateUser(ctx, tenantID, userInput).Return(nil, nil).Once()
	mockDb.EXPECT().InviteUser(ctx, tenantID, invite, notifyByEmail).Return(errors.New("mock error")).Once()

	success, err = service.InviteUser(ctx, tenantID, invite, notifyByEmail)

	// asserts
	assert.Error(t, err)
	assert.False(t, success)
}

func Test_DeleteInvite_Success(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	email := "user@foo.com"
	entityType := "project"
	entityId := "projectID123"
	roles := []string{
		"admin",
	}
	tenantID := "tenantID123"

	invite := graph.Invite{
		Email: email,
		Entity: &graph.EntityInput{
			EntityType: entityType,
			EntityID:   entityId,
		},
		Roles: roles,
	}

	// mock
	byEmailAndEntity := db.Invite{Email: invite.Email, EntityType: invite.Entity.EntityType, EntityID: invite.Entity.EntityID}
	mockDb.EXPECT().DeleteInvite(ctx, byEmailAndEntity).Return(nil).Once()

	// Act
	success, err := service.DeleteInvite(ctx, tenantID, invite)

	assert.NoError(t, err)
	assert.True(t, success)
}

func Test_DeleteInvite_Error(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	email := "user@foo.com"
	entityType := "project"
	entityId := "projectID123"
	roles := []string{
		"admin",
	}
	tenantID := "tenantID123"

	invite := graph.Invite{
		Email: email,
		Entity: &graph.EntityInput{
			EntityType: entityType,
			EntityID:   entityId,
		},
		Roles: roles,
	}

	// mock
	byEmailAndEntity := db.Invite{Email: invite.Email, EntityType: invite.Entity.EntityType, EntityID: invite.Entity.EntityID}
	mockDb.EXPECT().DeleteInvite(ctx, byEmailAndEntity).Return(errors.New("mock")).Once()

	// Act
	success, err := service.DeleteInvite(ctx, tenantID, invite)

	assert.Error(t, err)
	assert.False(t, success)
}
func Test_CreateUser_Success(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	email := "user@foo.com"
	userInput := graph.UserInput{
		Email: email,
	}

	// mock, expect
	mockDb.EXPECT().GetOrCreateUser(ctx, tenantID, userInput).Return(&graph.User{}, nil).Once()

	user, err := service.CreateUser(ctx, tenantID, userInput)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, user)
}

func Test_CreateUser_Error(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	email := "user@foo.com"
	userInput := graph.UserInput{
		Email: email,
	}

	// ERROR case: GetOrCreateUser return error
	mockDb.EXPECT().GetOrCreateUser(ctx, tenantID, userInput).Return(nil, errors.New("mock error")).Once()

	user, err := service.CreateUser(ctx, tenantID, userInput)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, user)
}
func Test_RemoveUser_Success(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	userID := "userID123"
	email := "email@foo.bar"

	// mock
	mockDb.EXPECT().RemoveUser(ctx, tenantID, userID, mock.Anything).Return(true, nil).Once()

	// Act
	res, err := service.RemoveUser(ctx, tenantID, &userID, nil)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.True(t, *res)

	// mock
	mockDb.EXPECT().RemoveUser(ctx, tenantID, userID, mock.Anything).Return(true, nil).Once()

	// Act
	res, err = service.RemoveUser(ctx, tenantID, &userID, &email)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.True(t, *res)

}

func Test_RemoveUser_Error(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	userID := "userID123"

	// mock
	success := false
	mockDb.EXPECT().RemoveUser(ctx, tenantID, userID, mock.Anything).Return(success, errors.New("mock")).Once()

	// Act
	_, err := service.RemoveUser(ctx, tenantID, &userID, nil)

	// asserts
	assert.Error(t, err)
}
func Test_User_Success(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	userID := "userID123"

	// mock
	mockUser := &graph.User{}
	mockDb.EXPECT().GetUserByID(ctx, tenantID, userID).Return(mockUser, nil).Once()

	// Act
	user, err := service.User(ctx, tenantID, userID)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, mockUser, user)
}

func Test_User_NotFound(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	userID := "userID123"

	// mock
	mockDb.EXPECT().GetUserByID(ctx, tenantID, userID).Return(nil, gorm.ErrRecordNotFound).Once()

	// Act
	user, err := service.User(ctx, tenantID, userID)

	// asserts
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func Test_User_Error(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	userID := "userID123"

	// mock
	mockDb.EXPECT().GetUserByID(ctx, tenantID, userID).Return(nil, errors.New("mock error")).Once()

	// Act
	user, err := service.User(ctx, tenantID, userID)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, user)
}
func Test_UserByEmail_Success(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	email := "user@foo.com"

	// mock
	mockUser := &graph.User{}
	mockDb.EXPECT().GetUserByEmail(ctx, tenantID, email).Return(mockUser, nil).Once()

	// Act
	user, err := service.UserByEmail(ctx, tenantID, email)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, mockUser, user)
}

func Test_UserByEmail_NotFound(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	email := "user@foo.com"

	// mock
	mockDb.EXPECT().GetUserByEmail(ctx, tenantID, email).Return(nil, gorm.ErrRecordNotFound).Once()

	// Act
	user, err := service.UserByEmail(ctx, tenantID, email)

	// asserts
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func Test_UserByEmail_Error(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	email := "user@foo.com"

	// mock
	mockDb.EXPECT().GetUserByEmail(ctx, tenantID, email).Return(nil, errors.New("mock error")).Once()

	// Act
	user, err := service.UserByEmail(ctx, tenantID, email)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, user)
}
func Test_UsersConnection_Success(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	limit := 10
	page := 1

	// mock
	mockUserConnection := &graph.UserConnection{}
	mockDb.EXPECT().GetUsers(ctx, tenantID, limit, page).Return(mockUserConnection, nil).Once()

	// Act
	userConnection, err := service.UsersConnection(ctx, tenantID, &limit, &page)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, userConnection)
	assert.Equal(t, mockUserConnection, userConnection)
}

func Test_UsersConnection_Error(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	limit := 10
	page := 1

	// mock
	mockDb.EXPECT().GetUsers(ctx, tenantID, limit, page).Return(nil, errors.New("mock error")).Once()

	// Act
	userConnection, err := service.UsersConnection(ctx, tenantID, &limit, &page)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, userConnection)
}
func Test_GetZone_Success(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	// mock
	zone := &graph.Zone{
		ZoneID:   "zoneID123",
		TenantID: "tenantID123",
	}
	tcReturn := db.TenantConfiguration{
		TenantID: zone.TenantID,
		ZoneId:   zone.ZoneID,
	}
	mockDb.EXPECT().GetTenantConfigurationForContext(ctx).Return(&tcReturn, nil).Once()

	// Act
	tc, err := service.GetZone(ctx)

	// asserts
	assert.NoError(t, err)
	assert.Equal(t, tc.TenantID, tcReturn.TenantID)
	assert.Equal(t, tc.ZoneID, tcReturn.ZoneId)
}

func Test_GetZone_NoTenantConfiguration(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	// mock
	mockDb.EXPECT().GetTenantConfigurationForContext(ctx).Return(nil, nil).Once()

	// Act
	zone, err := service.GetZone(ctx)

	// asserts
	assert.NoError(t, err)
	assert.Nil(t, zone)
}

func Test_GetZone_Error(t *testing.T) {
	service, mockDb := setupService(t)
	ctx := context.Background()

	// mock
	mockDb.EXPECT().GetTenantConfigurationForContext(ctx).Return(nil, errors.New("mock error")).Once()

	// Act
	zone, err := service.GetZone(ctx)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, zone)
}

func TestNew(t *testing.T) {
	db := &mocks.DatabaseService{}
	svc := service.New(db)
	assert.NotNil(t, svc)
	assert.Equal(t, db, svc.Db)
}

func TestSearchUsers(t *testing.T) {
	svc, mockDb := setupService(t)

	mockUsers := []*graph.User{{ID: "userID123"}}
	tests := []struct {
		name       string
		ctx        context.Context
		query      string
		setupMocks func(ctx context.Context, databaseService *mocks.DatabaseService)
		result     []*graph.User
		errString  string
	}{
		{
			name:  "Success",
			ctx:   openmfpCtx.AddTenantToContext(context.TODO(), "tenant1"),
			query: "jo",
			setupMocks: func(ctx context.Context, db *mocks.DatabaseService) {
				mockDb.EXPECT().SearchUsers(ctx, "tenant1", "jo", service.MaxSearchUsersResults).Return(mockUsers, nil).Once()
			},
			result:    mockUsers,
			errString: "",
		},
		{
			name:      "NoTenantIdInContextError",
			ctx:       context.TODO(),
			result:    nil,
			errString: "someone stored a wrong value in the [tenantId] key with type [<nil>], expected [string]",
		},
		{
			name:      "EmptyTenantIdError",
			ctx:       openmfpCtx.AddTenantToContext(context.TODO(), ""),
			result:    nil,
			errString: "tenantID must not be empty",
		},
		{
			name:      "EmptyQueryError",
			ctx:       openmfpCtx.AddTenantToContext(context.TODO(), "tenant1"),
			result:    nil,
			errString: "query must not be empty",
		},
		{
			name:  "DbError",
			ctx:   openmfpCtx.AddTenantToContext(context.TODO(), "tenant1"),
			query: "jo",
			setupMocks: func(ctx context.Context, db *mocks.DatabaseService) {
				mockDb.EXPECT().SearchUsers(ctx, "tenant1", "jo", service.MaxSearchUsersResults).
					Return(nil, assert.AnError).Once()
			},
			result:    nil,
			errString: assert.AnError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks(tt.ctx, mockDb)
			}
			users, err := svc.SearchUsers(tt.ctx, tt.query)
			assert.Equal(t, tt.result, users)
			if tt.errString != "" {
				assert.Equal(t, tt.errString, err.Error())
			}
		})
	}
}
