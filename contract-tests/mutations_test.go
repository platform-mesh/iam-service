package contract_tests

import (
	"net/http"
	"testing"

	"github.com/openmfp/iam-service/contract-tests/gqlAssertions"
	dbMocks "github.com/openmfp/iam-service/pkg/db/mocks"
	"github.com/openmfp/iam-service/pkg/fga/mocks"
	graphql "github.com/openmfp/iam-service/pkg/graph"
	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MutationsTestSuite struct {
	CommonTestSuite
}

func TestLoginTestSuite(t *testing.T) {
	suite.Run(t, new(MutationsTestSuite))
}

func (suite *MutationsTestSuite) TestMutation_UsersConnection() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersConnectionMutation()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.usersConnection.pageInfo.totalCount", float64(144))).
		Assert(jsonpath.Equal("$.data.usersConnection.user[0].userId", "")).
		Assert(jsonpath.Equal("$.data.usersConnection.user[0].email", "invited-admin-member@it.corp")).
		Assert(jsonpath.Equal("$.data.usersConnection.user[0].firstName", nil)).
		Assert(jsonpath.Equal("$.data.usersConnection.user[0].lastName", nil)).
		Assert(jsonpath.Equal("$.data.usersConnection.user[0].invitationOutstanding", false)).
		End()
}

func (suite *MutationsTestSuite) TestMutation_DeleteInvite() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(deleteInviteMutation()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.deleteInvite", true)).
		End()
}

// Test RemoveUser
func (suite *MutationsTestSuite) TestMutation_RemoveUser() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)
	userMocks := dbMocks.NewUserHooks(suite.T())
	userMocks.EXPECT().UserRemoved(mock.Anything, mock.Anything, mock.Anything).Once()

	suite.GqlApiTest(&userInjection, userMocks, nil).
		GraphQLRequest(removeUserMutation()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.removeUser", true)).
		End()

	userMocks.AssertExpectations(suite.T())
}

func removeUserMutation() apitest.GraphQLRequestBody {
	const query = `mutation ($tenantId:String!, $email:String!) {
		removeUser(tenantId:$tenantId, email:$email)
	}`

	return apitest.GraphQLRequestBody{
		Query: query,
		Variables: map[string]interface{}{
			"tenantId": "eCh0yae7ooWaek2iejo8geiqua",
			"email":    "OOD8JOOM2Z@mycorp.com",
		},
	}
}

func (suite *MutationsTestSuite) TestMutation_CreateAccount() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(createAccountMutation(tenantId, "project", "test", iamAdminName)).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.createAccount", true)).
		End()
}

func (suite *MutationsTestSuite) TestMutation_RemoveAccount() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(removeAccountMutation(tenantId, "project", "test")).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.removeAccount", true)).
		End()
}

func (suite *MutationsTestSuite) TestMutation_AssignRoleBindings() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	currentUserRoles := []string{FGA_ROLE_PROJECT_OWNER, FGA_ROLE_PROJECT_VAULT_MAINTAINER}
	newUserRoles := []string{FGA_ROLE_PROJECT_OWNER}
	mockFgaEvents := mocks.NewFgaEvents(suite.T())
	mockFgaEvents.EXPECT().UserRoleChanged(
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, iamAdminName, currentUserRoles, newUserRoles,
	).Return(nil).Once()

	suite.GqlApiTest(&userInjection, nil, mockFgaEvents).
		GraphQLRequest(assignRoleBindingsMutation(tenantId, "test", "project", iamAdminName, "owner")).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.assignRoleBindings", true)).
		End()
}

func (suite *MutationsTestSuite) TestMutation_RemoveFromEntity() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(removeFromEntityMutation(tenantId, "test", "project", iamAdminName)).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.removeFromEntity", true)).
		End()
}

// TestMutation_CreateUser tests the createUser mutation
// 1. We create a user with full data
// 2. Then we create a user using the same email and userId with no other fields specified.
// Second time we should receive already existing user with all fields filled up.
// So the test checks if there is a firstName in the response of second user creation
// despite it absence in the input
func (suite *MutationsTestSuite) TestMutation_CreateUserTwoTimes() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	userID := "userId123"
	userEmail := "newUser@gmail.com"
	firstName := "John"
	lastName := "Doe"

	// create the user with full data
	userMocks := dbMocks.NewUserHooks(suite.T())
	userMocks.EXPECT().UserCreated(mock.Anything, mock.Anything, mock.Anything).Once()
	req := suite.GqlApiTest(&userInjection, userMocks, nil)

	req.
		GraphQLQuery(createUser, map[string]interface{}{
			"tenantId": tenantId,
			"input": graphql.UserInput{
				UserID:    userID,
				Email:     userEmail,
				FirstName: &firstName,
				LastName:  &lastName,
			},
		}).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.createUser.firstName", firstName)).
		Assert(jsonpath.Equal("$.data.createUser.lastName", lastName)).
		End()

	// this is a try of creating the user with the same email and userId, and it should existing user from db
	req.
		GraphQLQuery(createUser, map[string]interface{}{
			"tenantId": tenantId,
			"input": graphql.UserInput{
				UserID: userID,
				Email:  userEmail,
			},
		}).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.createUser.firstName", firstName)).
		Assert(jsonpath.Equal("$.data.createUser.lastName", lastName)).
		End()

	userMocks.AssertExpectations(suite.T())
}

func (suite *MutationsTestSuite) TestMutation_CreateUser_Error() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	userMocks := dbMocks.NewUserHooks(suite.T())
	suite.GqlApiTest(&userInjection, userMocks, nil).
		GraphQLQuery(createUser, map[string]interface{}{
			"tenantId": tenantId,
			"input":    graphql.UserInput{},
		}).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.HasGQLErrors()).
		End()
}

func (suite *MutationsTestSuite) TestMutation_CreateUser() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)
	userMocks := dbMocks.NewUserHooks(suite.T())
	userMocks.EXPECT().UserCreated(mock.Anything, mock.Anything, mock.Anything).Once()
	var createUserQuery_userID = generateID()

	suite.GqlApiTest(&userInjection, userMocks, nil).
		GraphQLRequest(createUserQuery(createUserQuery_userID)).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.createUser.userId", createUserQuery_userID)).
		Assert(jsonpath.Equal("$.data.createUser.email", "test@mycorp.com")).
		Assert(jsonpath.Equal("$.data.createUser.firstName", "testFirstName")).
		Assert(jsonpath.Equal("$.data.createUser.lastName", "testLastName")).
		Assert(jsonpath.Equal("$.data.createUser.invitationOutstanding", false)).
		End()

	userMocks.AssertExpectations(suite.T())
}

func (suite *MutationsTestSuite) TestMutation_AssignRoleBindings_RemoveUserRole() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	currentUserRoles := []string{FGA_ROLE_PROJECT_OWNER, FGA_ROLE_PROJECT_VAULT_MAINTAINER}
	newUserRoles := []string{FGA_ROLE_PROJECT_MEMBER}
	mockFgaEvents := mocks.NewFgaEvents(suite.T())
	mockFgaEvents.EXPECT().UserRoleChanged(
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, iamAdminName, currentUserRoles, newUserRoles,
	).Return(nil).Once()

	suite.GqlApiTest(&userInjection, nil, mockFgaEvents).
		GraphQLRequest(assignRoleBindingsMutation(tenantId, "test", "project", iamAdminName, "member")).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.assignRoleBindings", true)).
		End()
}

func (suite *MutationsTestSuite) TestMutation_AssignRoleBindings_UserRoleChanged() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	currentUserRoles := []string{FGA_ROLE_PROJECT_OWNER, FGA_ROLE_PROJECT_VAULT_MAINTAINER}
	newUserRoles := []string{FGA_ROLE_PROJECT_MEMBER}
	mockFgaEvents := mocks.NewFgaEvents(suite.T())
	mockFgaEvents.EXPECT().UserRoleChanged(
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, iamAdminName, currentUserRoles, newUserRoles,
	).Return(nil).Once()

	// role change for user ['vault_maintainer', 'owner'] -> ['member']
	request := suite.GqlApiTest(&userInjection, nil, mockFgaEvents)
	request.
		GraphQLRequest(assignRoleBindingsMutation(tenantId, "test", "project", iamAdminName, "member")).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.assignRoleBindings", true)).
		End()

	// role change for user ['member'] -> ['vault_maintainer']
	currentUserRoles = []string{FGA_ROLE_PROJECT_MEMBER}
	newUserRoles = []string{FGA_ROLE_PROJECT_VAULT_MAINTAINER}
	mockFgaEvents.EXPECT().UserRoleChanged(
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, iamAdminName, currentUserRoles, newUserRoles,
	).Return(nil).Once()

	request.
		GraphQLRequest(assignRoleBindingsMutation(tenantId, "test", "project", iamAdminName, "vault_maintainer")).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.assignRoleBindings", true)).
		End()

}

func (suite *MutationsTestSuite) TestMutation_AssignRoleBindings_UserRoleAdded() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	newUserId := "ID111111"
	currentUserRoles := []string{}
	newUserRoles := []string{FGA_ROLE_PROJECT_MEMBER}
	mockFgaEvents := mocks.NewFgaEvents(suite.T())
	mockFgaEvents.EXPECT().UserRoleChanged(
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, newUserId, currentUserRoles, newUserRoles,
	).Return(nil).Once()

	suite.GqlApiTest(&userInjection, nil, mockFgaEvents).
		GraphQLRequest(assignRoleBindingsMutation(tenantId, "test", "project", newUserId, "member")).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.assignRoleBindings", true)).
		End()
}

func (suite *MutationsTestSuite) TestMutation_AssignRoleBindings_No_UserRoleChanged() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	newUserId := "ID111111"
	mockFgaEvents := mocks.NewFgaEvents(suite.T())
	mockFgaEvents.AssertNotCalled(suite.T(), "UserRoleChanged")

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(assignRoleBindingsMutation_EmptyRoles(tenantId, "test", "project", newUserId)).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.HasGQLErrors()).
		End()

}
