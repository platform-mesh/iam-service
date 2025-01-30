package contract_tests

import (
	"net/http"
	"testing"

	"github.com/openmfp/iam-service/contract-tests/gqlAssertions"
	"github.com/steinfletcher/apitest"
	jsonpath "github.com/steinfletcher/apitest-jsonpath"
	"github.com/stretchr/testify/suite"
)

type QueriesTestSuite struct {
	CommonTestSuite
}

func TestContractTestSuite(t *testing.T) {
	suite.Run(t, new(QueriesTestSuite))
}

func (suite *QueriesTestSuite) TestQuery_AvailableRolesForEntityType() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(availableRolesForEntityTypeQuery()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.availableRolesForEntityType", []interface{}{})).
		End()

}

func (suite *QueriesTestSuite) TestQuery_AvailableRolesForEntity() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(availableRolesForEntityQuery()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal(
			"$.data.availableRolesForEntity",
			[]interface{}{
				map[string]interface{}{
					"displayName":   "Owner",
					"technicalName": "owner",
					"permissions":   nil,
				},
				map[string]interface{}{
					"displayName":   "Member",
					"technicalName": "member",
					"permissions":   nil,
				},
				map[string]interface{}{
					"displayName":   "Vault Maintainer",
					"technicalName": "vault_maintainer",
					"permissions":   nil,
				},
			},
		)).
		End()
}

func (suite *QueriesTestSuite) TestQuery_User() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(userQuery()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.user.userId", "OOD8JOOM2Z")).
		Assert(jsonpath.Equal("$.data.user.email", "OOD8JOOM2Z@mycorp.com")).
		Assert(jsonpath.Equal("$.data.user.firstName", nil)).
		Assert(jsonpath.Equal("$.data.user.lastName", nil)).
		Assert(jsonpath.Equal("$.data.user.invitationOutstanding", false)).
		End()
}

// Query: userByEmail

func userByEmailQuery() apitest.GraphQLRequestBody {
	const apiTestQuery = `query userByEmail ($tenantId:String!, $email:String!)
	{
		userByEmail(tenantId:$tenantId, email:$email)
		{
			userId
			email
			firstName
			lastName
			invitationOutstanding
		}
	}`

	return apitest.GraphQLRequestBody{
		Query: apiTestQuery,
		Variables: map[string]interface{}{
			"tenantId": "eCh0yae7ooWaek2iejo8geiqua",
			"email":    "OOD8JOOM2Z@mycorp.com",
		},
	}
}

func (suite *QueriesTestSuite) TestQuery_UserByEmail() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(userByEmailQuery()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.userByEmail.userId", "OOD8JOOM2Z")).
		Assert(jsonpath.Equal("$.data.userByEmail.email", "OOD8JOOM2Z@mycorp.com")).
		Assert(jsonpath.Equal("$.data.userByEmail.firstName", nil)).
		Assert(jsonpath.Equal("$.data.userByEmail.lastName", nil)).
		Assert(jsonpath.Equal("$.data.userByEmail.invitationOutstanding", false)).
		End()
}

func (suite *QueriesTestSuite) TestQuery_InviteUser() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(inviteUserQuery()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.inviteUser", true)).
		End()
}

// Test TenantInfo
func (suite *QueriesTestSuite) TestQuery_TenantInfo() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(tenantInfoQuery()).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.tenantInfo.tenantId", "eCh0yae7ooWaek2iejo8geiqua")).
		End()
}

func (suite *QueriesTestSuite) TestQuery_RolesForUserOfEntity() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(rolesForUserOfEntityQuery(tenantId, "test", "project", iamAdminName)).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Equal("$.data.rolesForUserOfEntity[0].displayName", "Owner")).
		Assert(jsonpath.Equal("$.data.rolesForUserOfEntity[0].technicalName", "owner")).
		Assert(jsonpath.Equal("$.data.rolesForUserOfEntity[1].displayName", "Vault Maintainer")).
		Assert(jsonpath.Equal("$.data.rolesForUserOfEntity[1].technicalName", "vault_maintainer")).
		End()
}

// Test usersOfEntity
func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filterSearchtermAndRoles() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntity_filterSearchtermAndRoles_Query(tenantId)).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 2)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.email", "ALICE@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.email", "ALICE2@mycorp.com")).
		Assert(jsonpath.Len("$.data.usersOfEntity.users[0].roles", 1)).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filterSearchtermAndRoles2() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntity_filterSearchtermAndRoles2_Query(tenantId)).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 1)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.email", "BOB@mycorp.com")).
		Assert(jsonpath.Len("$.data.usersOfEntity.users[0].roles", 1)).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filterRoles() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntity_filterRoles_Query(tenantId)).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 5)).
		End()
}
