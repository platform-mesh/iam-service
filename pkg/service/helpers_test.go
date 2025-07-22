package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/platform-mesh/golang-commons/context/keys"
	"github.com/platform-mesh/iam-service/pkg/db"
	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/service"
	"github.com/stretchr/testify/assert"
)

func Test_verifyLimitsWithOverride(t *testing.T) {
	t.Run("Limit and page are nil", func(t *testing.T) {
		var limit *int = nil
		var page *int = nil
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.Nil(t, err)
	})

	t.Run("Limit and page are within valid range", func(t *testing.T) {
		var _limit = 100
		var _page = 2
		var limit = &_limit
		var page = &_page
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.Nil(t, err)
	})

	t.Run("Limit is within range, page is nil", func(t *testing.T) {
		var _limit = 100
		var limit = &_limit
		var page *int = nil
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.Nil(t, err)
	})

	t.Run("Limit is nil, page is within range", func(t *testing.T) {
		var _page = 2
		var limit *int = nil
		var page = &_page
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.Nil(t, err)
	})

	t.Run("Limit and page are outside valid range", func(t *testing.T) {
		var _limit = 2000
		var _page = -3
		var limit = &_limit
		var page = &_page
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.NotNil(t, err)
	})

	t.Run("Limit is outside valid range", func(t *testing.T) {
		var _limit = 2000
		var _page = 2
		var limit = &_limit
		var page = &_page
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.NotNil(t, err)
	})

	t.Run("Page is outside valid range", func(t *testing.T) {
		var _limit = 50
		var _page = -2
		var limit = &_limit
		var page = &_page
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.NotNil(t, err)
	})

}

func Test_GeneratePaginationLimits(t *testing.T) {
	t.Run("Pagination limits OK", func(t *testing.T) {
		var limit = 10
		var userIdToRolesLength = 10
		var page = 3
		var invitesLength = 100

		start, end := service.GeneratePaginationLimits(limit, userIdToRolesLength, page, invitesLength)
		assert.Equal(t, 10, start)
		assert.Equal(t, 20, end)
	})

	t.Run("Pagination limits OK 2", func(t *testing.T) {
		var limit = 10
		var userIdToRolesLength = 10
		var page = 30
		var invitesLength = 100

		start, end := service.GeneratePaginationLimits(limit, userIdToRolesLength, page, invitesLength)
		assert.Equal(t, 100, start)
		assert.Equal(t, 100, end)
	})

	t.Run("Pagination limits OK 3", func(t *testing.T) {
		var limit = 10
		var userIdToRolesLength = 23
		var page = 1
		var invitesLength = 7

		start, end := service.GeneratePaginationLimits(limit, userIdToRolesLength, page, invitesLength)
		assert.Equal(t, 0, start)
		assert.Equal(t, 0, end)
	})
}
func Test_checkFilterRoles(t *testing.T) {

	dnAdmin := "Admin"
	tnAdmin := "admin"
	dnGuest := "Guest"
	tnGuest := "guest"
	t.Run("No search filter roles", func(t *testing.T) {
		userRoles := []*graph.Role{
			{TechnicalName: "admin", DisplayName: "Admin"},
			{TechnicalName: "user", DisplayName: "User"},
		}
		searchfilterRoles := []*graph.RoleInput{
			{
				DisplayName:   dnAdmin,
				TechnicalName: tnAdmin,
			},
		}
		result := service.CheckFilterRoles(userRoles, searchfilterRoles)
		assert.True(t, result)
	})

	t.Run("Matching role found", func(t *testing.T) {
		userRoles := []*graph.Role{
			{TechnicalName: "admin"},
			{TechnicalName: "user"},
		}
		searchfilterRoles := []*graph.RoleInput{}
		result := service.CheckFilterRoles(userRoles, searchfilterRoles)
		assert.True(t, result)
	})

	t.Run("No matching role found", func(t *testing.T) {
		userRoles := []*graph.Role{
			{TechnicalName: "admin", DisplayName: "Admin"},
			{TechnicalName: "user", DisplayName: "User"},
		}
		searchfilterRoles := []*graph.RoleInput{
			{TechnicalName: tnGuest, DisplayName: dnGuest},
		}
		result := service.CheckFilterRoles(userRoles, searchfilterRoles)
		assert.False(t, result)
	})
}

func Test_FilterInvites(t *testing.T) {
	// sample invites for testing
	invite1 := db.Invite{
		Email: "admin@example.com",
		Roles: "admin,owner",
	}
	invite2 := db.Invite{
		Email: "user@example.com",
		Roles: "user",
	}
	invite3 := db.Invite{
		Email: "manager@example.com",
		Roles: "manager,admin",
	}
	invite4 := db.Invite{
		Email: "other@example.org",
		Roles: "user,owner",
	}
	invites := []db.Invite{invite1, invite2, invite3, invite4}

	t.Run("Empty search string with no role filter", func(t *testing.T) {
		// when no search string and no role filter, every invite should match
		searchTerm := ""
		filtered, owners := service.FilterInvites(invites, &searchTerm, nil)
		// Expect all invites are returned
		assert.Equal(t, len(invites), len(filtered))
		// Count owner occurrences
		expectedOwners := 0
		for _, inv := range invites {
			if strings.Contains(inv.Roles, "owner") {
				expectedOwners++
			}
		}
		assert.Equal(t, expectedOwners, owners)
	})

	t.Run("Search filter matches email case insensitively", func(t *testing.T) {
		searchTerm := "EXAMPLE.COM"
		// search string is set to a part of the email
		filtered, _ := service.FilterInvites(invites, &searchTerm, nil)
		// All emails ending with example.com should match (invites 1,2,3 match, invite4 does not)
		assert.Equal(t, 3, len(filtered))
	})

	t.Run("Role filter applied - matching role", func(t *testing.T) {
		// filter by role "admin"
		roleFilter := []*graph.RoleInput{
			{TechnicalName: "admin", DisplayName: "Administrator"},
		}
		searchTerm := "example.com"
		filtered, owners := service.FilterInvites(invites, &searchTerm, roleFilter)
		// Only invite1 and invite3 have "admin" in Roles.
		assert.Equal(t, 2, len(filtered))
		// invite1 contains "owner" so owners count should be 1
		assert.Equal(t, 1, owners)
	})

	t.Run("Role filter applied - no matching role", func(t *testing.T) {
		// filter by role "nonexistent"
		roleFilter := []*graph.RoleInput{
			{TechnicalName: "nonexistent", DisplayName: "Nonexistent"},
		}
		searchTerm := "example.com"
		filtered, owners := service.FilterInvites(invites, &searchTerm, roleFilter)
		// None of the invites contain the role "nonexistent"
		assert.Equal(t, 0, len(filtered))
		assert.Equal(t, 0, owners)
	})

	t.Run("Combined email and role filter, non-matching search string", func(t *testing.T) {
		// When search string does not match, no invites returned irrespective of roles.
		roleFilter := []*graph.RoleInput{
			{TechnicalName: "admin", DisplayName: "Administrator"},
		}
		searchTerm := "nomatch"
		filtered, owners := service.FilterInvites(invites, &searchTerm, roleFilter)
		assert.Equal(t, 0, len(filtered))
		assert.Equal(t, 0, owners)
	})

	t.Run("Search filter nil, should only filter by roles", func(t *testing.T) {
		roleFilter := []*graph.RoleInput{
			{TechnicalName: "admin", DisplayName: "Administrator"},
		}
		// search string is set to a part of the email
		filtered, _ := service.FilterInvites(invites, nil, roleFilter)
		// All emails ending with example.com should match (invites 1,2,3 match, invite4 does not)
		assert.Equal(t, 2, len(filtered))
	})
}
func Test_GetUserIDsFromUserIDRoles(t *testing.T) {
	t.Run("Empty map returns empty slice", func(t *testing.T) {
		result := service.GetUserIDsFromUserIDRoles(map[string][]string{})
		assert.Empty(t, result)
	})

	t.Run("Single user without owner", func(t *testing.T) {
		input := map[string][]string{
			"user1": {"member"},
		}
		result := service.GetUserIDsFromUserIDRoles(input)
		assert.ElementsMatch(t, []string{"user1"}, result)
	})

	t.Run("Multiple users with varying roles", func(t *testing.T) {
		input := map[string][]string{
			"user1": {"owner", "admin"},
			"user2": {"member"},
			"user3": {"owner"},
		}
		expected := []string{"user1", "user2", "user3"}
		result := service.GetUserIDsFromUserIDRoles(input)
		assert.ElementsMatch(t, expected, result)
	})
}

func TestGetRequestId_WithValue(t *testing.T) {
	expected := "test-request-id"
	ctx := context.WithValue(context.Background(), keys.RequestIdCtxKey, expected)
	id := service.GetRequestId(ctx)
	assert.Equal(t, expected, id)
}

func TestGetRequestId_NoValue(t *testing.T) {
	ctx := context.Background()
	id := service.GetRequestId(ctx)
	assert.Equal(t, "no_request_id_error", id)
}
