package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-jose/go-jose/v4"
	mfpcontext "github.com/platform-mesh/golang-commons/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/platform-mesh/iam-service/pkg/db"
	"github.com/platform-mesh/iam-service/pkg/db/mocks"
	fgamock "github.com/platform-mesh/iam-service/pkg/fga/mocks"
	"github.com/platform-mesh/iam-service/pkg/fga/types"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/service"
	"google.golang.org/grpc/status"
)

var (
	validToken = "eyJqa3UiOiJodHRwczovL2FlMmdjbGNlbC5hY2NvdW50czQwMC5vbmRlbWFuZC5jb20vb2F1dGgyL2NlcnRzIiwia2lkIjoick9VdUo0aWxjYWEyLU9xemMtWWVxY2h1aDRRIiwiYWxnIjoiUlMyNTYifQ.eyJzdWIiOiJjOGMzMjBjZC1lYzA3LTQ4MzUtOGFhMi04NzgwNTkwNjQ4MTQiLCJhdWQiOiJjOGMzMjBjZC1lYzA3LTQ4MzUtOGFhMi04NzgwNTkwNjQ4MTQiLCJhcHBfdGlkIjoiOGVmNzQ5ZmItZjM1OC00ZDYyLWI5OWQtMTgwZDg1NzkwNzM0IiwiaXNzIjoiaHR0cHM6Ly9hZTJnY2xjZWwuYWNjb3VudHM0MDAub25kZW1hbmQuY29tIiwiem9uZV91dWlkIjoiOGVmNzQ5ZmItZjM1OC00ZDYyLWI5OWQtMTgwZDg1NzkwNzM0IiwiY25mIjp7Ing1dCNTMjU2IjoidjF3SXl5cEJEaldNQ3phd3owazRxeEhhekVJSXdJLW5pZlRHSUw4c21GdyJ9LCJleHAiOjE2ODcxNzkxNTEsImlhdCI6MTY4NzE3NTU1MSwianRpIjoiZGRmMThhYjItZDM3MS00ZGU5LWJjZjMtZTc4ZDJiY2I2NzEyIn0.ZGirtxmcJU6RgB4GqfOWqCinya3t4-GD5jbFeGN75gvakSWXh57UedyChWpnxMqKlvb_QnQYCX-FGoOTgesx4FRSE3-GwB9uuNftjVvCNcLer5v23hCq84_RgDfN_7zo5Tm71YQxX250nA1HZ3DzPn984FiOuyDAu-iHAvQwBEE4pOqWbx40zbI63l6ttAIwclLoYvUXk5eOaD6jn5FQlwyvjB3nvmD5KDmuJQ-NsKVMdLnVyeXx9jRo_guMloCiUGZZ19h7MR-xhx-YnW408S3ELHpD90VA1E8fcyhhKUWd45RGvjTt-yqSk6ER1qRGaAyWoDC5iR0XkvxytpnD5A" // nolint: gosec, gochecknoglobals, lll
)

func setupService(t *testing.T) (*service.Service, *mocks.DatabaseService, *fgamock.Service) {
	t.Helper()
	mockDb := mocks.NewDatabaseService(t)
	mockFga := fgamock.NewService(t)
	svc := service.New(mockDb, mockFga)
	return svc, mockDb, mockFga
}

func Test_InviteUser_Success(t *testing.T) {
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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
	service, mockDb, _ := setupService(t)
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

	// Act
	limit = 100000
	userConnection, err = service.UsersConnection(ctx, tenantID, &limit, &page)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, userConnection)

}
func Test_GetZone_Success(t *testing.T) {
	service, mockDb, _ := setupService(t)
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
	mockDb.EXPECT().GetTenantConfigurationForContext(mock.Anything).Return(&tcReturn, nil).Once()

	// Act
	tc, err := service.GetZone(ctx)

	// asserts
	assert.NoError(t, err)
	assert.Equal(t, tc.TenantID, tcReturn.TenantID)
	assert.Equal(t, tc.ZoneID, tcReturn.ZoneId)
}

func Test_GetZone_NoTenantConfiguration(t *testing.T) {
	service, mockDb, _ := setupService(t)
	ctx := context.Background()

	// mock
	mockDb.EXPECT().GetTenantConfigurationForContext(mock.Anything).Return(nil, nil).Once()

	// Act
	zone, err := service.GetZone(ctx)

	// asserts
	assert.NoError(t, err)
	assert.Nil(t, zone)
}

func Test_GetZone_Error(t *testing.T) {
	service, mockDb, _ := setupService(t)
	ctx := context.Background()

	// mock
	mockDb.EXPECT().GetTenantConfigurationForContext(mock.Anything).Return(nil, errors.New("mock error")).Once()

	// Act
	zone, err := service.GetZone(ctx)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, zone)
}

func TestNew(t *testing.T) {
	db := &mocks.DatabaseService{}
	svc := service.New(db, nil)
	assert.NotNil(t, svc)
	assert.Equal(t, db, svc.Db)
}

func Test_AssignRoleBindings(t *testing.T) {
	service, _, mockFga := setupService(t)
	ctx := context.Background()

	// set parameters
	tenantID := "tenantID123"
	entityType := "project"
	entityId := "entityId"
	input := []*graph.Change{
		{
			UserID: "userID123",
			Roles:  []string{"admin"},
		},
	}

	// mock
	mockFga.EXPECT().AssignRoleBindings(mock.Anything, tenantID, entityType, entityId, input).Return(nil).Once()

	// Act
	success, err := service.AssignRoleBindings(ctx, tenantID, entityType, entityId, input)
	assert.NoError(t, err)
	assert.True(t, success)
}

func Test_AssignRoleBindings_EmptyRolesError(t *testing.T) {
	service, _, _ := setupService(t)
	ctx := context.Background()

	// set parameters
	tenantID := "tenantID123"
	entityType := "project"
	entityId := "entityId"
	input := []*graph.Change{
		{
			UserID: "userID123",
			Roles:  []string{},
		},
	}

	// Act
	success, err := service.AssignRoleBindings(ctx, tenantID, entityType, entityId, input)
	assert.Error(t, err)
	assert.False(t, success)

}

func Test_UsersOfEntity(t *testing.T) {
	service, mockDb, mockFga := setupService(t)
	ctx := context.Background()

	// set parameters
	tenantID := "tenantID123"
	entity := graph.EntityInput{
		EntityType: "project",
		EntityID:   "entityId",
	}
	var page = 1
	var limit = 10
	var showInvitees = true

	userIDToRoles := types.UserIDToRoles{
		"userID123": []string{"owner"},
		"userID345": []string{"member"},
		"userID678": []string{"owner"},
	}

	mockResponseUsers := []*graph.User{
		{
			TenantID: tenantID,
			UserID:   "userID123",
			Email:    "user123@sap.com",
		},
		{
			TenantID: tenantID,
			UserID:   "userID345",
			Email:    "user345@sap.com",
		},
		{
			TenantID: tenantID,
			UserID:   "userID678",
			Email:    "userID678@sap.com",
		},
	}

	invites := []db.Invite{
		{
			TenantID:   tenantID,
			Email:      "someuser",
			EntityType: "project",
			EntityID:   "entityId",
		},
	}

	// mock
	mockFga.EXPECT().UsersForEntity(mock.Anything, tenantID, entity.EntityID, entity.EntityType).Return(userIDToRoles, nil).Once()
	mockDb.EXPECT().GetUsersByUserIDs(mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockResponseUsers, nil).Twice()
	mockDb.EXPECT().GetRolesByTechnicalNames(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(mockFunc_GetRolesByTechnicalNames)
	mockDb.EXPECT().GetInvitesForEntity(mock.Anything, tenantID, entity.EntityType, entity.EntityID).Return(invites, nil).Once()

	// Act
	uc, err := service.UsersOfEntity(ctx, tenantID, entity, &limit, &page, &showInvitees, nil, []*graph.RoleInput{}, nil)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, uc)

	assert.Equal(t, 4, uc.PageInfo.TotalCount)
	assert.Equal(t, 2, uc.PageInfo.OwnerCount)

	assert.Equal(t, 4, len(uc.Users))
}

func Test_UsersOfEntity_Filter(t *testing.T) {
	service, mockDb, mockFga := setupService(t)
	ctx := context.Background()

	// set parameters
	tenantID := "tenantID123"
	entity := graph.EntityInput{
		EntityType: "project",
		EntityID:   "entityId",
	}
	var page = 1
	var limit = 10
	var showInvitees = true

	mockres_UserForEntityRolefilter := types.UserIDToRoles{
		"userID123": []string{"owner"},
	}

	mockres_dbUsers := []*graph.User{
		{TenantID: tenantID, UserID: "userID123", Email: "user123@sap.com"},
	}

	invites := []db.Invite{
		{TenantID: tenantID, Email: "user123456", EntityType: "project", EntityID: "entityId", Roles: "owner"},
	}
	searchTerm := "user123"
	rolesFilter := []*graph.RoleInput{
		{DisplayName: "Owner", TechnicalName: "owner"},
	}

	// mock
	mockFga.EXPECT().UsersForEntityRolefilter(mock.Anything, tenantID, entity.EntityID, entity.EntityType, rolesFilter).
		Return(mockres_UserForEntityRolefilter, nil).Once()
	mockDb.EXPECT().GetUsersByUserIDs(mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockres_dbUsers, nil).Once()
	mockDb.EXPECT().GetUsersByUserIDs(mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockres_dbUsers, nil).Once()
	mockDb.EXPECT().GetRolesByTechnicalNames(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(mockFunc_GetRolesByTechnicalNames)
	mockDb.EXPECT().GetInvitesForEntity(mock.Anything, tenantID, entity.EntityType, entity.EntityID).Return(invites, nil).Once()

	// Act
	uc, err := service.UsersOfEntity(ctx, tenantID, entity, &limit, &page, &showInvitees, &searchTerm, rolesFilter, nil)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, uc)

	assert.Equal(t, 2, len(uc.Users))
	assert.Equal(t, 2, uc.PageInfo.TotalCount)
	assert.Equal(t, 2, uc.PageInfo.OwnerCount)
}

func Test_UsersOfEntity_FilterAdvanced(t *testing.T) {
	service, mockDb, mockFga := setupService(t)
	ctx := context.Background()

	// set parameters
	tenantID := "01emp2m3v3batersxj73qhm5zq"
	entity := graph.EntityInput{
		EntityType: "project",
		EntityID:   "dolomiten-sync",
	}
	var page = 1
	var limit = 10
	var showInvitees = true

	mockResponseFgaUserIDRoles := types.UserIDToRoles{
		"usermiID123":  []string{"owner"},
		"userID345":    []string{"member"},
		"usermiID678":  []string{"owner"},
		"userID111":    []string{"owner"},
		"userID222":    []string{"member"},
		"userID333":    []string{"owner"},
		"usermiID444":  []string{"member"},
		"userID555":    []string{"owner"},
		"userID666":    []string{"member"},
		"userID777":    []string{"owner"},
		"usermiID888":  []string{"member", "owner"},
		"userID999":    []string{"admin"},
		"userID000":    []string{"admin"},
		"userID1111":   []string{"owner"},
		"userID2222":   []string{"member"},
		"userID3333":   []string{"owner"},
		"usermiID4444": []string{"admin", "owner"},
		"userID5555":   []string{"admin"},
		"userID6666":   []string{"member"},
		"userID7777":   []string{"admin", "owner"},
		"userID8888":   []string{"member"},
		"userID9999":   []string{"admin"},
		"userID0000":   []string{"member"},
	}

	mockres_DbUsers := []*graph.User{ // 5, owners: 4
		{UserID: "usermiID123", Email: "something@corp.co", TenantID: "t1"},
		{UserID: "usermiID678", Email: "usermiID678@corp.co", TenantID: "t1"},
		{UserID: "usermiID444", Email: "usermiID444@corp.co", TenantID: "t1"},
		{UserID: "usermiID888", Email: "usermiID888@corp.co", TenantID: "t1"},
		{UserID: "usermiID4444", Email: "usermiID4444@corp.co", TenantID: "t1"},
	}

	mockResponseInvites := []db.Invite{ // searchTerm: 7, owners: 3
		{TenantID: tenantID, Email: "mi@sap.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
		{TenantID: tenantID, Email: "user_001@example.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
		{TenantID: tenantID, Email: "user_002@example.com", EntityType: "project", EntityID: "entityId", Roles: "admin"},
		{TenantID: tenantID, Email: "mi2@sap.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
		{TenantID: tenantID, Email: "mi3@sap.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
		{TenantID: tenantID, Email: "mi4@sap.com", EntityType: "project", EntityID: "entityId", Roles: "owner"},
		{TenantID: tenantID, Email: "mi5@sap.com", EntityType: "project", EntityID: "entityId", Roles: "owner,admin"},
		{TenantID: tenantID, Email: "mi6@sap.com", EntityType: "project", EntityID: "entityId", Roles: "owner"},
		{TenantID: tenantID, Email: "mi7@sap.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
	}
	searchTerm := "mi"
	emptyRolesFilter := []*graph.RoleInput{}

	// mock
	mockFga.EXPECT().UsersForEntity(mock.Anything, tenantID, entity.EntityID, entity.EntityType).Return(mockResponseFgaUserIDRoles, nil).Once()
	mockDb.EXPECT().GetUsersByUserIDs(mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockres_DbUsers, nil).Once()
	mockDb.EXPECT().GetUsersByUserIDs(mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockres_DbUsers, nil).Once()
	mockDb.EXPECT().GetRolesByTechnicalNames(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(mockFunc_GetRolesByTechnicalNames)
	mockDb.EXPECT().GetInvitesForEntity(mock.Anything, tenantID, entity.EntityType, entity.EntityID).Return(mockResponseInvites, nil).Once()

	// Act
	guc, err := service.UsersOfEntity(ctx, tenantID, entity, &limit, &page, &showInvitees, &searchTerm, emptyRolesFilter, nil)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, guc)

	assert.Equal(t, 10, len(guc.Users))
	assert.Equal(t, 12, guc.PageInfo.TotalCount)
	assert.Equal(t, 7, guc.PageInfo.OwnerCount)

}

func Test_UsersOfEntity_FilterAdvanced_Groupfilter(t *testing.T) {
	service, mockSvc, mockFga := setupService(t)
	ctx := context.Background()

	// set parameters
	tenantID := "01emp2m3v3batersxj73qhm5zq"
	entity := graph.EntityInput{
		EntityType: "project",
		EntityID:   "dolomiten-sync",
	}
	var page = 1
	var limit = 10
	var showInvitees = true

	mockres_fgaUserRoles_rolesfiltered := types.UserIDToRoles{ // 11
		"userID123":  []string{"owner"},
		"miID678":    []string{"owner"},
		"userID111":  []string{"owner"},
		"userID333":  []string{"owner"},
		"userID555":  []string{"owner"},
		"userID777":  []string{"owner"},
		"miID888":    []string{"member", "owner"},
		"userID1111": []string{"owner"},
		"userID3333": []string{"owner"},
		"userID7777": []string{"admin", "owner"},
	}

	mockres_dbUsers_termfiltered := []*graph.User{ // 2
		{TenantID: tenantID, UserID: "miID678", Email: "miID678@corp.co"},
		{TenantID: tenantID, UserID: "miID888", Email: "miID888@corp.co"},
	}

	mockres_dbInvites := []db.Invite{ // invitees: 3, owners+searchFilter: 1
		{TenantID: tenantID, Email: "newemail1@example.com", EntityType: "project", EntityID: "entityId", Roles: "owner"},
		{TenantID: tenantID, Email: "newemail2@example.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
		{TenantID: tenantID, Email: "mi3@example.com", EntityType: "project", EntityID: "entityId", Roles: "owner"},
	}
	searchTerm := "mi"
	dnOwner := "Owner"
	tnOwner := "owner"
	rolesFilter := []*graph.RoleInput{
		{
			DisplayName:   dnOwner,
			TechnicalName: tnOwner,
		},
	}

	// mock
	mockFga.EXPECT().UsersForEntityRolefilter(mock.Anything, tenantID, entity.EntityID, entity.EntityType, rolesFilter).
		Return(mockres_fgaUserRoles_rolesfiltered, nil).Once()
	mockSvc.EXPECT().GetUsersByUserIDs(mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockres_dbUsers_termfiltered, nil).Once()
	mockSvc.EXPECT().GetUsersByUserIDs(mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockres_dbUsers_termfiltered, nil).Once()
	mockSvc.EXPECT().GetRolesByTechnicalNames(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(mockFunc_GetRolesByTechnicalNames)
	mockSvc.EXPECT().GetInvitesForEntity(mock.Anything, tenantID, entity.EntityType, entity.EntityID).Return(mockres_dbInvites, nil).Once()

	// Act
	guc, err := service.UsersOfEntity(ctx, tenantID, entity, &limit, &page, &showInvitees, &searchTerm, rolesFilter, nil)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, guc)

	assert.Equal(t, 3, len(guc.Users))
	assert.Equal(t, 3, guc.PageInfo.TotalCount)
	assert.Equal(t, 3, guc.PageInfo.OwnerCount)

}

func Test_UsersOfEntity_With_Invitations(t *testing.T) {
	service, mockDb, mockFga := setupService(t)
	ctx := context.Background()

	// set parameters
	tenantID := "tenantID123"
	entity := graph.EntityInput{
		EntityType: "project",
		EntityID:   "entityId",
	}
	var page = 0
	var limit = -1
	var showInvitees = true

	userIDToRoles := types.UserIDToRoles{
		"userID123": []string{"admin"},
	}
	users := []*graph.User{
		{
			TenantID: tenantID,
			UserID:   "userID123",
			Email:    "user@sap.com",
		},
	}
	invites := []db.Invite{
		{
			TenantID:   tenantID,
			Email:      "invited-user@sap.com",
			EntityType: "project",
			EntityID:   "entityId",
		},
	}

	// mock
	mockFga.EXPECT().UsersForEntity(mock.Anything, tenantID, entity.EntityID, entity.EntityType).Return(userIDToRoles, nil).Once()
	mockDb.EXPECT().GetUsersByUserIDs(
		mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(users, nil).Twice()
	mockDb.EXPECT().GetRolesByTechnicalNames(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(mockFunc_GetRolesByTechnicalNames)
	mockDb.EXPECT().GetInvitesForEntity(mock.Anything, tenantID, entity.EntityType, entity.EntityID).Return(invites, nil).Once()

	// Act
	uc, err := service.UsersOfEntity(ctx, tenantID, entity, &limit, &page, &showInvitees, nil, []*graph.RoleInput{}, nil)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, uc)
	assert.Equal(t, uc.Users[1].User.Email, "invited-user@sap.com")
}

func Test_UsersOfEntity_Errors(t *testing.T) {
	service, mockDb, mockFga := setupService(t)
	ctx := context.Background()

	// set parameters
	tenantID := "tenantID123"
	entity := graph.EntityInput{
		EntityType: "project",
		EntityID:   "entityId",
	}
	var page = 1
	var limit = 100000
	var showInvitees = true

	userIDToRoles := types.UserIDToRoles{
		"userID123": []string{"admin"},
	}

	// Act
	uc, err := service.UsersOfEntity(ctx, tenantID, entity, &limit, &page, &showInvitees, nil, []*graph.RoleInput{}, nil)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, uc)

	// mock
	limit = 100
	mockFga.EXPECT().UsersForEntity(mock.Anything, tenantID, entity.EntityID, entity.EntityType).Return(nil, errors.New("")).Once()

	// Act
	uc, err = service.UsersOfEntity(ctx, tenantID, entity, &limit, &page, &showInvitees, nil, []*graph.RoleInput{}, nil)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, uc)

	// mock
	mockFga.EXPECT().UsersForEntity(
		mock.Anything, tenantID, entity.EntityID, entity.EntityType,
	).Return(userIDToRoles, nil).Once()
	mockDb.EXPECT().GetUsersByUserIDs(
		mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil, errors.New("")).Twice()

	// Act
	uc, err = service.UsersOfEntity(ctx, tenantID, entity, &limit, &page, &showInvitees, nil, []*graph.RoleInput{}, nil)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, uc)
}

func Test_RemoveFromEntity(t *testing.T) {
	service, _, mockFga := setupService(t)
	ctx := context.Background()

	// set parameters
	tenantID := "tenantID123"
	entityType := "project"
	entityId := "entityId"
	userId := "userID123"

	// mock
	mockFga.EXPECT().RemoveFromEntity(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	// Act
	success, err := service.RemoveFromEntity(ctx, tenantID, entityType, userId, entityId)
	assert.NoError(t, err)
	assert.True(t, success)
}

func Test_LeaveEntity_Success(t *testing.T) {
	service, _, mockFga := setupService(t)
	ctx := context.Background()

	ctx = mfpcontext.AddWebTokenToContext(ctx, validToken, []jose.SignatureAlgorithm{jose.RS256})
	ctx = mfpcontext.AddAuthHeaderToContext(ctx, "Bearer token")

	tenantID := "tenantID123"
	entityType := "project"
	entityID := "entityId"

	// mock
	mockFga.EXPECT().
		RemoveFromEntity(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	// Act
	success, err := service.LeaveEntity(ctx, tenantID, entityType, entityID)

	// asserts
	assert.NoError(t, err)
	assert.True(t, success)
}

func Test_LeaveEntity_Error(t *testing.T) {
	service, _, _ := setupService(t)
	ctx := context.Background()

	invalidToken := "invalid"
	ctx = mfpcontext.AddWebTokenToContext(ctx, invalidToken, []jose.SignatureAlgorithm{jose.RS256})
	ctx = mfpcontext.AddAuthHeaderToContext(ctx, "Bearer token")

	tenantID := "tenantID123"
	entityType := "project"
	entityID := "entityId"

	// Act
	success, err := service.LeaveEntity(ctx, tenantID, entityType, entityID)

	// asserts
	assert.Error(t, err)
	assert.False(t, success)
}
func Test_RolesForUserOfEntity_Success(t *testing.T) {
	service, mockDb, mockFga := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	entityType := "project"
	entityID := "entityID"
	userID := "userID123"

	// mock
	userIDToRoles := types.UserIDToRoles{
		userID: []string{"admin"},
	}
	mockFga.EXPECT().UsersForEntity(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(userIDToRoles, nil).Once()
	role := &db.Role{
		DisplayName:   "admin",
		TechnicalName: "admin",
	}
	mockDb.EXPECT().GetRolesByTechnicalNames(mock.Anything, mock.Anything, mock.Anything).Return([]*db.Role{role}, nil).Once()

	// Act
	roles, err := service.RolesForUserOfEntity(ctx, tenantID, graph.EntityInput{
		EntityType: entityType,
		EntityID:   entityID,
	}, userID)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, roles)
	assert.Len(t, roles, 1)
	assert.Equal(t, "admin", roles[0].DisplayName)
	assert.Equal(t, "admin", roles[0].TechnicalName)
}

func Test_RolesForUserOfEntity_UserNotFound(t *testing.T) {
	service, _, mockFga := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	entityType := "project"
	entityID := "entityID"
	userID := "userID123"

	// mock
	userIDToRoles := types.UserIDToRoles{}
	mockFga.EXPECT().UsersForEntity(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(userIDToRoles, nil).Once()

	// Act
	roles, err := service.RolesForUserOfEntity(ctx, tenantID, graph.EntityInput{
		EntityType: entityType,
		EntityID:   entityID,
	}, userID)

	// asserts
	assert.NoError(t, err)
	assert.Nil(t, roles)
}

func Test_RolesForUserOfEntity_Error(t *testing.T) {
	service, _, mockFga := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	entityType := "project"
	entityID := "entityID"
	userID := "userID123"

	// mock
	mockFga.EXPECT().UsersForEntity(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("mock error")).Once()

	// Act
	roles, err := service.RolesForUserOfEntity(ctx, tenantID, graph.EntityInput{
		EntityType: entityType,
		EntityID:   entityID,
	}, userID)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, roles)
}

func Test_RolesForUserOfEntity_UnableToGetTechnicalRoleNames(t *testing.T) {
	service, mockDb, mockFga := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	entityType := "project"
	entityID := "entityID"
	userID := "userID123"

	// mock
	userIDToRoles := types.UserIDToRoles{
		"userID123": []string{"admin"},
	}

	mockFga.EXPECT().UsersForEntity(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(userIDToRoles, nil).Once()
	mockDb.EXPECT().GetRolesByTechnicalNames(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("mock error")).Once()

	// Act
	roles, err := service.RolesForUserOfEntity(ctx, tenantID, graph.EntityInput{
		EntityType: entityType,
		EntityID:   entityID,
	}, userID)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, roles)
}

func Test_AvailableRolesForEntity_Success(t *testing.T) {
	service, mockDb, _ := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	entity := graph.EntityInput{
		EntityType: "project",
		EntityID:   "entityId",
	}

	// mock
	mockRoles := []db.Role{
		{
			DisplayName:   "admin",
			TechnicalName: "admin",
		},
		{
			DisplayName:   "user",
			TechnicalName: "user",
		},
	}
	mockDb.EXPECT().GetRolesForEntity(mock.Anything, mock.Anything, mock.Anything).Return(mockRoles, nil).Once()

	// Act
	roles, err := service.AvailableRolesForEntity(ctx, tenantID, entity)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, roles)
}

func Test_AvailableRolesForEntity_Error(t *testing.T) {
	service, mockDb, _ := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	entity := graph.EntityInput{
		EntityType: "project",
		EntityID:   "entityId",
	}

	// mock
	mockDb.EXPECT().GetRolesForEntity(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("mock error")).Once()

	// Act
	roles, err := service.AvailableRolesForEntity(ctx, tenantID, entity)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, roles)
}
func Test_Service_AvailableRolesForEntityType(t *testing.T) {
	service, mockDb, _ := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	entityType := "project"

	// mock
	mockRoles := []db.Role{
		{
			DisplayName:   "admin",
			TechnicalName: "admin",
		},
		{
			DisplayName:   "user",
			TechnicalName: "user",
		},
	}
	mockDb.EXPECT().GetRolesForEntity(mock.Anything, mock.Anything, mock.Anything).Return(mockRoles, nil).Once()

	// Act
	roles, err := service.AvailableRolesForEntityType(ctx, tenantID, entityType)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, roles)
	assert.Equal(t, len(mockRoles), len(roles))
	for i, role := range roles {
		assert.Equal(t, mockRoles[i].DisplayName, role.DisplayName)
		assert.Equal(t, mockRoles[i].TechnicalName, role.TechnicalName)
	}
}
func Test_CreateAccount_Success(t *testing.T) {
	service, _, mockFga := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	entityType := "project"
	entityID := "entityID123"
	owner := "ownerID123"

	// mock
	mockFga.EXPECT().CreateAccount(mock.Anything, tenantID, entityType, entityID, owner).Return(nil).Once()

	// Act
	success, err := service.CreateAccount(ctx, tenantID, entityType, entityID, owner)

	// asserts
	assert.NoError(t, err)
	assert.True(t, success)
}

func Test_CreateAccount_DuplicateWriteError(t *testing.T) {
	service, _, mockFga := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	entityType := "project"
	entityID := "entityID123"
	owner := "ownerID123"
	errDuplicate := status.Error(2017, "Duplicate error")

	// mock
	mockFga.EXPECT().CreateAccount(mock.Anything, tenantID, entityType, entityID, owner).Return(errDuplicate).Once()

	// Act
	success, err := service.CreateAccount(ctx, tenantID, entityType, entityID, owner)

	// asserts
	assert.NoError(t, err)
	assert.True(t, success)
}

func Test_RemoveAccount_Success(t *testing.T) {
	service, _, mockFga := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	entityType := "project"
	entityID := "entityId"

	// mock
	mockFga.EXPECT().RemoveAccount(mock.Anything, tenantID, entityType, entityID).Return(nil).Once()

	// Act
	success, err := service.RemoveAccount(ctx, tenantID, entityType, entityID)

	// asserts
	assert.NoError(t, err)
	assert.True(t, success)
}

func Test_RemoveAccount_DuplicateWriteError(t *testing.T) {
	service, _, mockFga := setupService(t)
	ctx := context.Background()

	tenantID := "tenantID123"
	entityType := "project"
	entityID := "entityId"
	errDuplicate := status.Error(2017, "Duplicate error")

	// mock
	mockFga.EXPECT().RemoveAccount(mock.Anything, tenantID, entityType, entityID).Return(errDuplicate).Once()

	// Act
	success, err := service.RemoveAccount(ctx, tenantID, entityType, entityID)

	// asserts
	assert.NoError(t, err)
	assert.True(t, success)
}
func Test_TenantInfo_Success(t *testing.T) {
	service, mockDb, _ := setupService(t)
	ctx := context.Background()

	// mock
	mockDb.EXPECT().GetTenantConfigurationForContext(mock.Anything).Return(&db.TenantConfiguration{
		TenantID: "tenantID123",
	}, nil).Once()

	// Act
	tenantInfo, err := service.TenantInfo(ctx, nil)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, tenantInfo)
	assert.Equal(t, "tenantID123", tenantInfo.TenantID)
}

func Test_TenantInfo_NoTenantConfiguration(t *testing.T) {
	service, mockDb, _ := setupService(t)
	ctx := context.Background()

	// mock
	mockDb.EXPECT().GetTenantConfigurationForContext(mock.Anything).Return(nil, nil).Once()

	// Act
	tenantInfo, err := service.TenantInfo(ctx, nil)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, tenantInfo)
}

func Test_TenantInfo_Error(t *testing.T) {
	service, mockDb, _ := setupService(t)
	ctx := context.Background()

	// mock
	mockDb.EXPECT().GetTenantConfigurationForContext(mock.Anything).Return(nil, errors.New("mock error")).Once()

	// Act
	tenantInfo, err := service.TenantInfo(ctx, nil)

	// asserts
	assert.Error(t, err)
	assert.Nil(t, tenantInfo)
}

func TestSearchUsers(t *testing.T) {
	svc, mockDb, _ := setupService(t)

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
			ctx:   mfpcontext.AddTenantToContext(context.TODO(), "tenant1"),
			query: "jo",
			setupMocks: func(ctx context.Context, db *mocks.DatabaseService) {
				mockDb.EXPECT().SearchUsers(ctx, "tenant1", "jo").Return(mockUsers, nil).Once()
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
			ctx:       mfpcontext.AddTenantToContext(context.TODO(), ""),
			result:    nil,
			errString: "tenantID must not be empty",
		},
		{
			name:      "EmptyQueryError",
			ctx:       mfpcontext.AddTenantToContext(context.TODO(), "tenant1"),
			result:    nil,
			errString: "query must not be empty",
		},
		{
			name:  "DbError",
			ctx:   mfpcontext.AddTenantToContext(context.TODO(), "tenant1"),
			query: "jo",
			setupMocks: func(ctx context.Context, db *mocks.DatabaseService) {
				mockDb.EXPECT().SearchUsers(ctx, "tenant1", "jo").
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

func mockFunc_GetRolesByTechnicalNames(ctx context.Context, tenantID string, technicalNames []string) ([]*db.Role, error) {
	resolvedRoles := []*db.Role{}
	for _, role := range technicalNames {
		switch role {
		case "owner":
			resolvedRoles = append(resolvedRoles, &db.Role{TechnicalName: "owner", DisplayName: "Owner"})
		case "member":
			resolvedRoles = append(resolvedRoles, &db.Role{TechnicalName: "member", DisplayName: "Member"})
		case "admin":
			resolvedRoles = append(resolvedRoles, &db.Role{TechnicalName: "admin", DisplayName: "Admin"})
		}
	}
	return resolvedRoles, nil
}

func Test_UsersOfEntity_pagination(t *testing.T) {
	service, mockDb, mockFga := setupService(t)
	ctx := context.Background()

	// set parameters
	tenantID := "tenantid"
	entity := graph.EntityInput{
		EntityType: "project",
		EntityID:   "dolomiten-sync",
	}
	var page = 3
	var limit = 10
	var showInvitees = true

	mockResponse_UsersForEntity := types.UserIDToRoles{
		"userID123":    []string{"owner"},
		"userID345":    []string{"member"},
		"usermiID678":  []string{"owner"},
		"userID111":    []string{"owner"},
		"userID222":    []string{"member"},
		"userID333":    []string{"owner"},
		"usermiID444":  []string{"member"},
		"userID555":    []string{"owner"},
		"userID666":    []string{"member"},
		"userID777":    []string{"owner"}, // end first 10, 6 owners
		"usermiID888":  []string{"member", "owner"},
		"userID999":    []string{"admin"},
		"userID000":    []string{"admin"},
		"userID1111":   []string{"owner"},
		"userID2222":   []string{"member"},
		"userID3333":   []string{"owner"},
		"usermiID4444": []string{"admin", "owner"},
		"userID5555":   []string{"admin"},
		"userID6666":   []string{"member"},
		"userID7777":   []string{"admin", "owner"}, // end second 10, 5 owners
		"userID8888":   []string{"member"},
		"userID9999":   []string{"admin"},
		"userID0000":   []string{"member"},
	}

	mockResponseDbUsers1 := []*graph.User{
		{TenantID: tenantID, UserID: "userID123", Email: "userID123@corp.co"},
		{TenantID: tenantID, UserID: "userID345", Email: "userID345@corp.co"},
		{TenantID: tenantID, UserID: "usermiID678", Email: "usermiID678@corp.co"},
		{TenantID: tenantID, UserID: "userID111", Email: "userID111@corp.co"},
		{TenantID: tenantID, UserID: "userID222", Email: "userID222@corp.co"},
		{TenantID: tenantID, UserID: "userID333", Email: "userID333@corp.co"},
		{TenantID: tenantID, UserID: "userID444", Email: "userID444@corp.co"},
		{TenantID: tenantID, UserID: "userID555", Email: "userID555@corp.co"},
		{TenantID: tenantID, UserID: "userID666", Email: "userID666@corp.co"},
		{TenantID: tenantID, UserID: "userID777", Email: "userID777@corp.co"},
		{TenantID: tenantID, UserID: "usermiID888", Email: "usermiID888@corp.co"},
		{TenantID: tenantID, UserID: "userID999", Email: "userID999@corp.co"},
		{TenantID: tenantID, UserID: "userID000", Email: "userID000@corp.co"},
		{TenantID: tenantID, UserID: "userID1111", Email: "userID1111@corp.co"},
		{TenantID: tenantID, UserID: "userID2222", Email: "userID2222@corp.co"},
		{TenantID: tenantID, UserID: "userID3333", Email: "userID3333@corp.co"},
		{TenantID: tenantID, UserID: "usermiID4444", Email: "usermiID4444@corp.co"},
		{TenantID: tenantID, UserID: "userID5555", Email: "userID5555@corp.co"},
		{TenantID: tenantID, UserID: "userID6666", Email: "userID6666@corp.co"},
		{TenantID: tenantID, UserID: "userID7777", Email: "userID7777@corp.co"},
		{TenantID: tenantID, UserID: "userID8888", Email: "userID8888@corp.co"},
		{TenantID: tenantID, UserID: "userID9999", Email: "userID9999@corp.co"},
		{TenantID: tenantID, UserID: "userID0000", Email: "userID0000@corp.co"},
	}
	mockResponseDbUsers2 := []*graph.User{
		{TenantID: tenantID, UserID: "userID8888", Email: "userID8888@corp.co"},
		{TenantID: tenantID, UserID: "userID9999", Email: "userID9999@corp.co"},
		{TenantID: tenantID, UserID: "userID0000", Email: "userID0000@corp.co"},
	}

	mockResponseInvites := []db.Invite{
		{TenantID: tenantID, Email: "mi@sap.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
		{TenantID: tenantID, Email: "user_001@example.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
		{TenantID: tenantID, Email: "user_002@example.com", EntityType: "project", EntityID: "entityId", Roles: "admin"},
		{TenantID: tenantID, Email: "mi2@sap.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
	}
	// mock
	mockFga.EXPECT().UsersForEntity(mock.Anything, tenantID, entity.EntityID, entity.EntityType).
		Return(mockResponse_UsersForEntity, nil).Once()
	mockDb.EXPECT().GetUsersByUserIDs(mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockResponseDbUsers1, nil).Once()
	mockDb.EXPECT().GetUsersByUserIDs(mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockResponseDbUsers2, nil).Once()
	mockDb.EXPECT().GetRolesByTechnicalNames(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(mockFunc_GetRolesByTechnicalNames)
	mockDb.EXPECT().GetInvitesForEntity(mock.Anything, tenantID, entity.EntityType, entity.EntityID).Return(mockResponseInvites, nil).Once()

	// Act
	guc, err := service.UsersOfEntity(ctx, tenantID, entity, &limit, &page, &showInvitees, nil, []*graph.RoleInput{}, nil)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, guc)

	assert.Equal(t, 7, len(guc.Users))
	assert.Equal(t, 27, guc.PageInfo.TotalCount)
	assert.Equal(t, 11, guc.PageInfo.OwnerCount)

}

func Test_UsersOfEntity_pagination_nils(t *testing.T) {
	service, mockDb, mockFga := setupService(t)
	ctx := context.Background()

	// set parameters
	tenantID := "tenantid"
	entity := graph.EntityInput{
		EntityType: "project",
		EntityID:   "dolomiten-sync",
	}
	var showInvitees = true

	mockResponse_UsersForEntity := types.UserIDToRoles{
		"userID123":    []string{"owner"},
		"userID345":    []string{"member"},
		"usermiID678":  []string{"owner"},
		"userID111":    []string{"owner"},
		"userID222":    []string{"member"},
		"userID333":    []string{"owner"},
		"usermiID444":  []string{"member"},
		"userID555":    []string{"owner"},
		"userID666":    []string{"member"},
		"userID777":    []string{"owner"}, // end first 10, 6 owners
		"usermiID888":  []string{"member", "owner"},
		"userID999":    []string{"admin"},
		"userID000":    []string{"admin"},
		"userID1111":   []string{"owner"},
		"userID2222":   []string{"member"},
		"userID3333":   []string{"owner"},
		"usermiID4444": []string{"admin", "owner"},
		"userID5555":   []string{"admin"},
		"userID6666":   []string{"member"},
		"userID7777":   []string{"admin", "owner"}, // end second 10, 5 owners
		"userID8888":   []string{"member"},
		"userID9999":   []string{"admin"},
		"userID0000":   []string{"member"},
	}

	mockResponseDbUsers := []*graph.User{
		{TenantID: tenantID, UserID: "userID123", Email: "userID123@corp.co"},
		{TenantID: tenantID, UserID: "userID345", Email: "userID345@corp.co"},
		{TenantID: tenantID, UserID: "usermiID678", Email: "usermiID678@corp.co"},
		{TenantID: tenantID, UserID: "userID111", Email: "userID111@corp.co"},
		{TenantID: tenantID, UserID: "userID222", Email: "userID222@corp.co"},
		{TenantID: tenantID, UserID: "userID333", Email: "userID333@corp.co"},
		{TenantID: tenantID, UserID: "userID444", Email: "userID444@corp.co"},
		{TenantID: tenantID, UserID: "userID555", Email: "userID555@corp.co"},
		{TenantID: tenantID, UserID: "userID666", Email: "userID666@corp.co"},
		{TenantID: tenantID, UserID: "userID777", Email: "userID777@corp.co"},
		{TenantID: tenantID, UserID: "usermiID888", Email: "usermiID888@corp.co"},
		{TenantID: tenantID, UserID: "userID999", Email: "userID999@corp.co"},
		{TenantID: tenantID, UserID: "userID000", Email: "userID000@corp.co"},
		{TenantID: tenantID, UserID: "userID1111", Email: "userID1111@corp.co"},
		{TenantID: tenantID, UserID: "userID2222", Email: "userID2222@corp.co"},
		{TenantID: tenantID, UserID: "userID3333", Email: "userID3333@corp.co"},
		{TenantID: tenantID, UserID: "usermiID4444", Email: "usermiID4444@corp.co"},
		{TenantID: tenantID, UserID: "userID5555", Email: "userID5555@corp.co"},
		{TenantID: tenantID, UserID: "userID6666", Email: "userID6666@corp.co"},
		{TenantID: tenantID, UserID: "userID7777", Email: "userID7777@corp.co"},
		{TenantID: tenantID, UserID: "userID8888", Email: "userID8888@corp.co"},
		{TenantID: tenantID, UserID: "userID9999", Email: "userID9999@corp.co"},
		{TenantID: tenantID, UserID: "userID0000", Email: "userID0000@corp.co"},
	}

	mockResponseInvites := []db.Invite{
		{TenantID: tenantID, Email: "mi@sap.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
		{TenantID: tenantID, Email: "user_001@example.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
		{TenantID: tenantID, Email: "user_002@example.com", EntityType: "project", EntityID: "entityId", Roles: "admin"},
		{TenantID: tenantID, Email: "mi2@sap.com", EntityType: "project", EntityID: "entityId", Roles: "member"},
	}
	// mock
	mockFga.EXPECT().UsersForEntity(mock.Anything, tenantID, entity.EntityID, entity.EntityType).
		Return(mockResponse_UsersForEntity, nil).Once()
	mockDb.EXPECT().GetUsersByUserIDs(mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockResponseDbUsers, nil).Twice()
	mockDb.EXPECT().GetRolesByTechnicalNames(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(mockFunc_GetRolesByTechnicalNames)
	mockDb.EXPECT().GetInvitesForEntity(mock.Anything, tenantID, entity.EntityType, entity.EntityID).Return(mockResponseInvites, nil).Once()

	// Act
	guc, err := service.UsersOfEntity(ctx, tenantID, entity, nil, nil, &showInvitees, nil, []*graph.RoleInput{}, nil)

	// asserts
	assert.NoError(t, err)
	assert.NotNil(t, guc)

	assert.Equal(t, 23, len(guc.Users))
	assert.Equal(t, 27, guc.PageInfo.TotalCount)
	assert.Equal(t, 11, guc.PageInfo.OwnerCount)

}
