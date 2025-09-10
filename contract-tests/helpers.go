package contract_tests

import (
	"crypto/rand"
	"encoding/base64"

	graphql "github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/steinfletcher/apitest"
)

const (
	iamAdminName      = "AHCAE4EIVI"
	iamAdminNameToken = "eyJraWQiOiJJdE8xWTZuT0U2NlpWOEtOclBXbG5FOTNmSnMiLCJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJBSENBRTRFSVZJIiwibWFpbCI6InRlc3QudXNlckBzYXAuY29tIiwiaHR0cDovL3NjaGVtYXMubWljcm9zb2Z0LmNvbS9pZGVudGl0eS9jbGFpbXMvaWRlbnRpdHlwcm92aWRlciI6Imh0dHBzOi8vc3RzLndpbmRvd3MubmV0LzQyZjc2NzZjLWY0NTUtNDIzYy04MmY2LWRjMmQ5OTc5MWFmNy8iLCJpc3MiOiJodHRwczovL2EtaWFzLXRlbmFudC5hY2NvdW50cy5vbmRlbWFuZC5jb20iLCJsYXN0X25hbWUiOiJVc2VyIiwiaHR0cDovL3NjaGVtYXMubWljcm9zb2Z0LmNvbS9jbGFpbXMvYXV0aG5tZXRob2RzcmVmZXJlbmNlcyI6WyJodHRwOi8vc2NoZW1hcy5taWNyb3NvZnQuY29tL3dzLzIwMDgvMDYvaWRlbnRpdHkvYXV0aGVudGljYXRpb25tZXRob2QveDUwOSIsImh0dHA6Ly9zY2hlbWFzLm1pY3Jvc29mdC5jb20vY2xhaW1zL211bHRpcGxlYXV0aG4iXSwic2lkIjoiUy1TUC1iZWI0MThiZS05MWFlLTRjNGUtOWM5Yi0yZGIxNTBjMjFhMzQiLCJhdWQiOiIzZjc2OTM1MS1iMTFmLTRhZmYtYjQ4YS0wYTg4ZjNmMjA3ZjIiLCJodHRwOi8vc2NoZW1hcy5taWNyb3NvZnQuY29tL2lkZW50aXR5L2NsYWltcy90ZW5hbnRpZCI6IjQyZjc2NzZjLWY0NTUtNDIzYy04MmY2LWRjMmQ5OTc5MWFmNyIsImh0dHA6Ly9zY2hlbWFzLm1pY3Jvc29mdC5jb20vaWRlbnRpdHkvY2xhaW1zL29iamVjdGlkZW50aWZpZXIiOiI0ZWE3YzFmOS04MDVjLTQzZWMtOGY5MC03MTc2OTg4MWFlZDciLCJleHAiOjE2ODg4MzcyNDgsImlhdCI6MTY4ODgzMzY0OCwiZmlyc3RfbmFtZSI6IlRlc3QiLCJqdGkiOiIxMWQ3Y2M2YS02MDVlLTQ1ZTYtYjZjYy03OTcwMzU1MjU4YWEifQo=.WY-YlqlbvCVCK8CnHzr8TRuwvmswPyG-I8GuIGggfuO_i4uv42CQIa9sxijZUQfXvCmd2_x4UhrRzA0cWmiQKvvvVL_UEuWjGFIb7FCYKekmFdvxmyJaQ2UQH6JBQV16ri34Z4Hb01GKQMgl4wM5zwkiTZTThLq0Xj4pNv4A_Y10vp__Qfrgb6Wz-MMuWej0kBEPfhO8oGnK2V0BSFrUUq4jf4MQ8cMwDRQ73LX4y-pdN6SPhQuMIWjYX2JytP-ubMIdsCgHkjosf3HK_ds9GWNyhuAJdUo8XUnvVCy9vUT5UoSvkDUQrHR25MSMZcegmMGBUS5Q0T_40X31kU4JaQ" // nolint: lll,gosec
	tenantId          = "29y87kiy4iakrkbb/test"

	pathToSchemaFile       = "./testdata/schema.fga"
	pathToTenantDataFile   = "./assets/data.yaml"
	pathToUserTestDataFile = "./testdata/tuples.yaml"
	createUser             = `mutation($tenantId: String!, $input: UserInput!) {
		createUser(tenantId: $tenantId, input:$input){
			userId
			email
			firstName
			lastName
		}
	}`
	loginMutation = `mutation {
		login
	}`

	createUserQuery_userEmail = "test@mycorp.com"
	createUserQuery_firstName = "testFirstName"
	createUserQuery_lastName  = "testLastName"

	FGA_ROLE_PROJECT_OWNER            = "role:project/test/owner"
	FGA_ROLE_PROJECT_VAULT_MAINTAINER = "role:project/test/vault_maintainer"
	FGA_ROLE_PROJECT_MEMBER           = "role:project/test/member"
)

func generateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

type EntityInput struct {
	EntityType string `json:"entityType"`
	EntityId   string `json:"entityId"`
}

type InviteInput struct {
	Email  string      `json:"email"`
	Entity EntityInput `json:"entity"`
	Roles  []string    `json:"roles"`
}

type UserInput struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// Query: availableRolesForEntityType
func availableRolesForEntityTypeQuery() apitest.GraphQLRequestBody {
	const apiTestQuery = `query availableRolesForEntityType($tenantId: ID!, $entityType: String!) {
		availableRolesForEntityType(tenantId: $tenantId, entityType: $entityType) {
			displayName
			technicalName
			permissions {
				displayName
				relation
			}
		}
	}`

	return apitest.GraphQLRequestBody{
		Query: apiTestQuery,
		Variables: map[string]interface{}{
			"tenantId":   tenantId,
			"entityType": "account",
		},
	}
}

// Query: availableRolesForEntity
func availableRolesForEntityQuery() apitest.GraphQLRequestBody {
	const apiTestQuery = `query availableRolesForEntity($tenantId: ID!, $entity: EntityInput!) {
		availableRolesForEntity(tenantId: $tenantId, entity: $entity) {
			displayName
			technicalName
			permissions {
				displayName
				relation
			}
		}
	}`
	return apitest.GraphQLRequestBody{
		Query: apiTestQuery,
		Variables: map[string]interface{}{
			"tenantId": tenantId,
			"entity": EntityInput{
				"project",
				"test",
			},
		},
	}
}

// Query: User
func userQuery() apitest.GraphQLRequestBody {
	const apiTestQuery = `query user ($tenantId:String!, $userId:String!)
	{
		user(tenantId:$tenantId, userId:$userId)
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
			"tenantId": "29y87kiy4iakrkbb/test",
			"userId":   "OOD8JOOM2Z",
		},
	}
}

func tenantInfoQuery() apitest.GraphQLRequestBody {
	const query = `query tenantInfo($tenantId:String) {
		tenantInfo(tenantId:$tenantId) {
			tenantId
			subdomain
			emailDomain
			emailDomains
		}
	}`

	return apitest.GraphQLRequestBody{
		Query: query,
		Variables: map[string]interface{}{
			"tenantId": "29y87kiy4iakrkbb/test",
		},
	}
}

func rolesForUserOfEntityQuery(tenantId string, entityId string, entityType string, userId string) apitest.GraphQLRequestBody {
	const rolesForUserOfEntity = `query RolesForUserOfEntity($tenantId: ID!, $entity: EntityInput!, $userId: String!) {
		rolesForUserOfEntity(tenantId: $tenantId, entity: $entity, userId: $userId) {
			displayName
			technicalName
			permissions {
				displayName
				relation
			}
		}
	}`
	return apitest.GraphQLRequestBody{
		Query: rolesForUserOfEntity,
		Variables: map[string]interface{}{
			"tenantId": tenantId,
			"entity": EntityInput{
				entityType,
				entityId,
			},
			"userId": userId,
		},
	}
}

// Query: UsersConnection
func usersConnectionMutation() apitest.GraphQLRequestBody {
	const apiTestQuery = `query usersConnection($tenantId:String!)
	{
		usersConnection(tenantId:$tenantId)
		{
			user {
					userId
					email
					firstName
					lastName
					invitationOutstanding
				}
			pageInfo {
				totalCount
			}
		}
	}`

	return apitest.GraphQLRequestBody{
		Query: apiTestQuery,
		Variables: map[string]interface{}{
			"tenantId": "29y87kiy4iakrkbb/test",
		},
	}
}

// Query: InviteUser
func inviteUserQuery() apitest.GraphQLRequestBody {
	const apiTestQuery = `mutation inviteUser($tenantId:String!, $invite:Invite!, $notifyByEmail:Boolean!) {
		inviteUser(tenantId:$tenantId, invite:$invite, notifyByEmail:$notifyByEmail)
	}`

	return apitest.GraphQLRequestBody{
		Query: apiTestQuery,
		Variables: map[string]interface{}{
			"tenantId": "29y87kiy4iakrkbb/test",
			"invite": InviteInput{
				Email: "invited-admin-member@it.corp",
				Entity: EntityInput{
					EntityType: "project",
					EntityId:   "test",
				},
				Roles: []string{"owner"},
			},
			"notifyByEmail": true,
		},
	}
}

// Query: DeleteInvite
func deleteInviteMutation() apitest.GraphQLRequestBody {
	const apiTestQuery = `mutation deleteInvite($tenantId:String!, $invite:Invite!) {
		deleteInvite(tenantId:$tenantId, invite:$invite)
	}`

	return apitest.GraphQLRequestBody{
		Query: apiTestQuery,
		Variables: map[string]interface{}{
			"tenantId": "29y87kiy4iakrkbb/test",
			"invite": InviteInput{
				Email: "invited-admin-member@it.corp",
				Entity: EntityInput{
					EntityType: "project",
					EntityId:   "test",
				},
				Roles: []string{"owner"},
			},
		},
	}
}

// Query: CreateUser
func createUserQuery(id string) apitest.GraphQLRequestBody {
	const apiTestQuery = `mutation createUser($tenantId:String!, $input:UserInput!) {
		createUser(tenantId:$tenantId, input:$input)
		{
			userId
			email
			firstName
			lastName
			invitationOutstanding
		}
	}`

	var firstName = createUserQuery_firstName
	var lastName = createUserQuery_lastName

	return apitest.GraphQLRequestBody{
		Query: apiTestQuery,
		Variables: map[string]interface{}{
			"tenantId": tenantId,
			"input": graphql.UserInput{
				UserID:    id,
				Email:     createUserQuery_userEmail,
				FirstName: &firstName,
				LastName:  &lastName,
			},
		},
	}
}

func removeFromEntityMutation(tenantId string, entityId string, entityType string, userId string) apitest.GraphQLRequestBody {
	const removeFromEntity = `mutation RemoveFromEntity($tenantId: ID!, $entityType: String!, $userId: ID!, $entityId: ID!) {
		removeFromEntity(tenantId:$tenantId, entityId:$entityId, entityType:$entityType, userId:$userId)
	}`

	return apitest.GraphQLRequestBody{
		Query: removeFromEntity,
		Variables: map[string]interface{}{
			"tenantId":   tenantId,
			"entityId":   entityId,
			"entityType": entityType,
			"userId":     userId,
		},
	}
}

func assignRoleBindingsMutation(tenantId string, entityId string, // nolint: unparam
	entityType string, userId string, role string) apitest.GraphQLRequestBody { // nolint: unparam
	const assignRoleBindings = `mutation AssignRoleBindings($tenantId: ID!, $entityType: String!, $entityId: ID!, $input: [Change]!) {
		assignRoleBindings(tenantId: $tenantId, entityType: $entityType, entityId: $entityId, input: $input)
	}`
	return apitest.GraphQLRequestBody{
		Query: assignRoleBindings,
		Variables: map[string]interface{}{
			"tenantId":   tenantId,
			"entityType": entityType,
			"entityId":   entityId,
			"input": []map[string]interface{}{
				{
					"userId": userId,
					"roles":  []string{role},
				},
			},
		},
	}
}

func assignRoleBindingsMutation_EmptyRoles(tenantId string, entityId string, entityType string, userId string) apitest.GraphQLRequestBody {
	const assignRoleBindings = `mutation AssignRoleBindings($tenantId: ID!, $entityType: String!, $entityId: ID!, $input: [Change]!) {
		assignRoleBindings(tenantId: $tenantId, entityType: $entityType, entityId: $entityId, input: $input)
	}`
	return apitest.GraphQLRequestBody{
		Query: assignRoleBindings,
		Variables: map[string]interface{}{
			"tenantId":   tenantId,
			"entityType": entityType,
			"entityId":   entityId,
			"input": []map[string]interface{}{
				{
					"userId": userId,
					"roles":  []string{},
				},
			},
		},
	}
}

func createAccountMutation(tenantId string, entityType string, entityId string, owner string) apitest.GraphQLRequestBody {
	const createAccount = `mutation ($tenantId: ID!, $entityType: String!, $entityId: ID!, $owner: String!) {
		createAccount(tenantId: $tenantId, entityType: $entityType, entityId: $entityId, owner: $owner)
	  }`
	return apitest.GraphQLRequestBody{
		Query: createAccount,
		Variables: map[string]interface{}{
			"tenantId":   tenantId,
			"entityType": entityType,
			"entityId":   entityId,
			"owner":      owner,
		},
	}
}

func removeAccountMutation(tenantId string, entityType string, entityId string) apitest.GraphQLRequestBody {
	const removeAccount = `mutation ($tenantId: ID!, $entityType: String!, $entityId: ID!) {
		removeAccount(tenantId: $tenantId, entityType: $entityType, entityId: $entityId)
	  }`
	return apitest.GraphQLRequestBody{
		Query: removeAccount,
		Variables: map[string]interface{}{
			"tenantId":   tenantId,
			"entityType": entityType,
			"entityId":   entityId,
		},
	}
}

// Query: usersOfEntity
func usersOfEntity_filterSearchtermAndRoles_Query(tenantId string) apitest.GraphQLRequestBody {
	const apiTestQuery = ` query usersOfEntity(
    $tenantId: ID!
    $entity: EntityInput!
    $limit: Int
    $page: Int
    $showInvitees: Boolean
    $searchTerm: String
    $roles: [RoleInput]
  ) {
    usersOfEntity(
      tenantId: $tenantId
      entity: $entity
      limit: $limit
      page: $page
      showInvitees: $showInvitees
      searchTerm: $searchTerm
      roles: $roles
    ) {
      users {
        user {
          userId
          email
          firstName
          lastName
          invitationOutstanding
        }
        roles {
          displayName
          technicalName
        }
      }
      pageInfo {
        ownerCount
        totalCount
      }
    }
  }`

	ds1 := "Owner" // nolint: goconst
	tn1 := "owner" // nolint: goconst
	return apitest.GraphQLRequestBody{
		Query: apiTestQuery,
		Variables: map[string]interface{}{
			"tenantId": tenantId,
			"entity": EntityInput{
				"project",
				"test",
			},
			"limit":        10,
			"page":         1,
			"showInvitees": true,
			"searchTerm":   "Alice",
			"roles": []*graphql.RoleInput{
				{
					DisplayName:   ds1,
					TechnicalName: tn1,
				},
			},
		},
	}
}

func usersOfEntity_filter_BOB_and_Owner_Query(tenantId string) apitest.GraphQLRequestBody {
	const apiTestQuery = ` query usersOfEntity(
    $tenantId: ID!
    $entity: EntityInput!
    $limit: Int
    $page: Int
    $showInvitees: Boolean
    $searchTerm: String
    $roles: [RoleInput]
  ) {
    usersOfEntity(
      tenantId: $tenantId
      entity: $entity
      limit: $limit
      page: $page
      showInvitees: $showInvitees
      searchTerm: $searchTerm
      roles: $roles
    ) {
      users {
        user {
          userId
          email
          firstName
          lastName
          invitationOutstanding
        }
        roles {
          displayName
          technicalName
        }
      }
      pageInfo {
        ownerCount
        totalCount
      }
    }
  }`

	ds1 := "Owner"
	tn1 := "owner"
	_ = ds1
	_ = tn1
	return apitest.GraphQLRequestBody{
		Query: apiTestQuery,
		Variables: map[string]interface{}{
			"tenantId": tenantId,
			"page":     1,
			"limit":    10,
			"entity": EntityInput{
				"project",
				"test",
			},
			"showInvitees": true,
			"searchTerm":   "BOB",
			"roles": []*graphql.RoleInput{
				{
					DisplayName:   ds1,
					TechnicalName: tn1,
				},
			},
		},
	}
}

func usersOfEntityFiltered(vars map[string]interface{}) apitest.GraphQLRequestBody {
	const apiTestQuery = ` query usersOfEntity(
    $tenantId: ID!
    $entity: EntityInput!
    $limit: Int
    $page: Int
    $showInvitees: Boolean
    $searchTerm: String
    $roles: [RoleInput]
  ) {
    usersOfEntity(
      tenantId: $tenantId
      entity: $entity
      limit: $limit
      page: $page
      showInvitees: $showInvitees
      searchTerm: $searchTerm
      roles: $roles
    ) {
      users {
        user {
          userId
          email
          firstName
          lastName
          invitationOutstanding
        }
        roles {
          displayName
          technicalName
        }
      }
      pageInfo {
        ownerCount
        totalCount
      }
    }
  }`

	return apitest.GraphQLRequestBody{
		Query:     apiTestQuery,
		Variables: vars,
	}
}

func usersOfEntityFilteredSortby(vars map[string]interface{}) apitest.GraphQLRequestBody {
	const apiTestQuery = ` query usersOfEntity(
    $tenantId: ID!
    $entity: EntityInput!
    $limit: Int
    $page: Int
    $showInvitees: Boolean
    $searchTerm: String
    $roles: [RoleInput]
		$sortBy: SortByInput
  ) {
    usersOfEntity(
      tenantId: $tenantId
      entity: $entity
      limit: $limit
      page: $page
      showInvitees: $showInvitees
      searchTerm: $searchTerm
      roles: $roles,
			sortBy: $sortBy,
    ) {
      users {
        user {
          userId
          email
          firstName
          lastName
          invitationOutstanding
        }
        roles {
          displayName
          technicalName
        }
      }
      pageInfo {
        ownerCount
        totalCount
      }
    }
  }`

	return apitest.GraphQLRequestBody{
		Query:     apiTestQuery,
		Variables: vars,
	}
}

func usersOfEntity_filterRoles_Query(tenantId string) apitest.GraphQLRequestBody {
	const apiTestQuery = ` query usersOfEntity(
    $tenantId: ID!
    $entity: EntityInput!
    $limit: Int
    $page: Int
    $showInvitees: Boolean
    $searchTerm: String
    $roles: [RoleInput]
  ) {
    usersOfEntity(
      tenantId: $tenantId
      entity: $entity
      limit: $limit
      page: $page
      showInvitees: $showInvitees
      searchTerm: $searchTerm
      roles: $roles
    ) {
      users {
        user {
          userId
          email
          firstName
          lastName
          invitationOutstanding
        }
        roles {
          displayName
          technicalName
        }
      }
      pageInfo {
        ownerCount
        totalCount
      }
    }
  }`

	ds1 := "Owner"
	tn1 := "owner"
	_ = ds1
	_ = tn1
	return apitest.GraphQLRequestBody{
		Query: apiTestQuery,
		Variables: map[string]interface{}{
			"tenantId": tenantId,
			"entity": EntityInput{
				"project",
				"test",
			},
			"showInvitees": true,
			"roles": []*graphql.RoleInput{
				{
					DisplayName:   ds1,
					TechnicalName: tn1,
				},
			},
		},
	}
}
