package contract_tests

import (
	"net/http"
	"testing"

	"github.com/openmfp/iam-service/contract-tests/gqlAssertions"
	graphql "github.com/openmfp/iam-service/pkg/graph"
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
				map[string]interface{}{
					"displayName":   "Admin",
					"technicalName": "admin",
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

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_BOB_and_Owner() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntity_filter_BOB_and_Owner_Query(tenantId)).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 1)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.email", "BOB@mycorp.com")).
		Assert(jsonpath.Len("$.data.usersOfEntity.users[0].roles", 1)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.ownerCount", float64(1))).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.totalCount", float64(1))).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_BIXIE() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFiltered(map[string]interface{}{
			"tenantId": tenantId,
			"page":     1,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"BIXIEProject",
			},
			"showInvitees": false,
			"searchTerm":   "BIXIE",
			"roles":        []*graphql.RoleInput{},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 10)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.email", "BIXIE1@mycorp.com")).
		Assert(jsonpath.Len("$.data.usersOfEntity.users[0].roles", 2)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.ownerCount", float64(7))).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.totalCount", float64(15))).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_BIXIE_p2() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFiltered(map[string]interface{}{
			"tenantId": tenantId,
			"page":     2,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"BIXIEProject",
			},
			"showInvitees": false,
			"searchTerm":   "BIXIE",
			"roles":        []*graphql.RoleInput{},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 4)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.ownerCount", float64(7))).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.totalCount", float64(15))).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_BIXIE_p2_invitees() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFiltered(map[string]interface{}{
			"tenantId": tenantId,
			"page":     2,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"BIXIEProject",
			},
			"showInvitees": true,
			"searchTerm":   "BIXIE",
			"roles":        []*graphql.RoleInput{},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 5)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.ownerCount", float64(7))).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.totalCount", float64(15))).
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

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_FOOBAR_p2_invitees() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFiltered(map[string]interface{}{
			"tenantId": tenantId,
			"page":     2,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": true,
			"searchTerm":   "FOOBAR",
			"roles":        []*graphql.RoleInput{},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 10)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.ownerCount", float64(16))).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.totalCount", float64(24))).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_FOOBAR_p3_invitees() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFiltered(map[string]interface{}{
			"tenantId": tenantId,
			"page":     3,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": true,
			"searchTerm":   "FOOBAR",
			"roles":        []*graphql.RoleInput{},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 4)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.ownerCount", float64(16))).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.totalCount", float64(24))).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_FOOBAR_p1() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFiltered(map[string]interface{}{
			"tenantId": tenantId,
			"page":     1,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": false,
			"searchTerm":   "FOOBAR",
			"roles":        []*graphql.RoleInput{},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 10)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.ownerCount", float64(16))).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.totalCount", float64(24))).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_FOOBAR_p1_owners() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFiltered(map[string]interface{}{
			"tenantId": tenantId,
			"page":     1,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": false,
			"searchTerm":   "FOOBAR",
			"roles": []*graphql.RoleInput{
				{
					DisplayName:   "Owner",
					TechnicalName: "owner",
				},
			},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 10)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.ownerCount", float64(16))).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.totalCount", float64(16))).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_FOOBAR_p2_owners() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFiltered(map[string]interface{}{
			"tenantId": tenantId,
			"page":     2,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": false,
			"searchTerm":   "FOOBAR",
			"roles": []*graphql.RoleInput{
				{
					DisplayName:   "Owner",
					TechnicalName: "owner",
				},
			},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 4)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.ownerCount", float64(16))).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.totalCount", float64(16))).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_FOOBAR_p2_owners_invitees() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFiltered(map[string]interface{}{
			"tenantId": tenantId,
			"page":     2,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": true,
			"searchTerm":   "FOOBAR",
			"roles": []*graphql.RoleInput{
				{
					DisplayName:   "Owner",
					TechnicalName: "owner",
				},
			},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 6)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.ownerCount", float64(16))).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.totalCount", float64(16))).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_FOOBAR_p1_owners_asc() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFilteredSortby(map[string]interface{}{
			"tenantId": tenantId,
			"page":     1,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": false,
			"searchTerm":   "FOOBAR",
			"sortBy": &graphql.SortBy{
				Field:     "user",
				Direction: "asc",
			},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 10)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.email", "FOOBAR2@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.firstName", "Alice")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.lastName", "Johnson")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.email", "FOOBAR9@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.firstName", "Angela")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.lastName", "Thomas")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.email", "FOOBAR8@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.firstName", "Christopher")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.lastName", "Anderson")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.email", "FOOBAR6@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.firstName", "David")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.lastName", "Rodriguez")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[4].user.email", "FOOBAR12@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[4].user.firstName", "James")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[4].user.lastName", "Harris")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[5].user.email", "FOOBAR7@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[5].user.firstName", "Jennifer")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[5].user.lastName", "Martinez")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[6].user.email", "FOOBAR1@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[6].user.firstName", "John")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[6].user.lastName", "Smith")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[7].user.email", "FOOBAR10@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[7].user.firstName", "Joseph")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[7].user.lastName", "Jackson")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[8].user.email", "FOOBAR14@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[8].user.firstName", "Kevin")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[8].user.lastName", "Thompson")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[9].user.email", "FOOBAR13@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[9].user.firstName", "Laura")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[9].user.lastName", "Martin")).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_FOOBAR_p2_owners_asc() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFilteredSortby(map[string]interface{}{
			"tenantId": tenantId,
			"page":     2,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": false,
			"searchTerm":   "FOOBAR",
			"sortBy": &graphql.SortBy{
				Field:     "user",
				Direction: "asc",
			},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 4)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.email", "FOOBAR11@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.firstName", "Lisa")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.lastName", "White")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.email", "FOOBAR5@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.firstName", "Maria")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.lastName", "Garcia")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.email", "FOOBAR4@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.firstName", "Michael")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.lastName", "Brown")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.email", "FOOBAR3@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.firstName", "Robert")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.lastName", "Williams")).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_FOOBAR_p2_owners_desc() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFilteredSortby(map[string]interface{}{
			"tenantId": tenantId,
			"page":     2,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": false,
			"searchTerm":   "FOOBAR",
			"sortBy": &graphql.SortBy{
				Field:     "user",
				Direction: "desc",
			},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 4)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.email", "FOOBAR6@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.firstName", "David")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.lastName", "Rodriguez")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.email", "FOOBAR8@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.firstName", "Christopher")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.lastName", "Anderson")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.email", "FOOBAR9@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.firstName", "Angela")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.lastName", "Thomas")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.email", "FOOBAR2@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.firstName", "Alice")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.lastName", "Johnson")).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_FOOBAR_p1_owners_desc() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFilteredSortby(map[string]interface{}{
			"tenantId": tenantId,
			"page":     1,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": false,
			"searchTerm":   "FOOBAR",
			"sortBy": &graphql.SortBy{
				Field:     "user",
				Direction: "desc",
			},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 10)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.email", "FOOBAR3@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.firstName", "Robert")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[0].user.lastName", "Williams")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.email", "FOOBAR4@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.firstName", "Michael")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[1].user.lastName", "Brown")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.email", "FOOBAR5@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.firstName", "Maria")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[2].user.lastName", "Garcia")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.email", "FOOBAR11@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.firstName", "Lisa")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[3].user.lastName", "White")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[4].user.email", "FOOBAR13@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[4].user.firstName", "Laura")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[4].user.lastName", "Martin")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[5].user.email", "FOOBAR14@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[5].user.firstName", "Kevin")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[5].user.lastName", "Thompson")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[6].user.email", "FOOBAR10@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[6].user.firstName", "Joseph")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[6].user.lastName", "Jackson")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[7].user.email", "FOOBAR1@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[7].user.firstName", "John")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[7].user.lastName", "Smith")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[8].user.email", "FOOBAR7@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[8].user.firstName", "Jennifer")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[8].user.lastName", "Martinez")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[9].user.email", "FOOBAR12@mycorp.com")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[9].user.firstName", "James")).
		Assert(jsonpath.Equal("$.data.usersOfEntity.users[9].user.lastName", "Harris")).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_FOOBAR_p1_owners_Desc() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFilteredSortby(map[string]interface{}{
			"tenantId": tenantId,
			"page":     1,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": false,
			"searchTerm":   "FOOBAR",
			"sortBy": &graphql.SortBy{
				Field:     "user",
				Direction: "Desc",
			},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.HasGQLErrors()).
		End()
}

func (suite *QueriesTestSuite) TestQuery_UsersOfEntity_filter_p2_owners_invitees() {
	userInjection := getUserInjection(iamAdminNameToken, defaultSpiffeeHeaderValue)

	suite.GqlApiTest(&userInjection, nil, nil).
		GraphQLRequest(usersOfEntityFiltered(map[string]interface{}{
			"tenantId": tenantId,
			"page":     2,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"FoobarProject",
			},
			"showInvitees": true,
			"searchTerm":   nil,
			"roles": []*graphql.RoleInput{
				{
					DisplayName:   "Owner",
					TechnicalName: "owner",
				},
			},
		})).
		Expect(suite.T()).
		Status(http.StatusOK).
		Assert(gqlAssertions.NoGQLErrors()).
		Assert(jsonpath.Len("$.data.usersOfEntity.users", 6)).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.ownerCount", float64(16))).
		Assert(jsonpath.Equal("$.data.usersOfEntity.pageInfo.totalCount", float64(16))).
		End()
}
