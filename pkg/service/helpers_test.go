package service_test

import (
	"testing"

	"github.com/openmfp/iam-service/pkg/db"
	"github.com/openmfp/iam-service/pkg/graph"
	"github.com/openmfp/iam-service/pkg/service"
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
		var _limit int = 100
		var _page int = 2
		var limit *int = &_limit
		var page *int = &_page
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.Nil(t, err)
	})

	t.Run("Limit is within range, page is nil", func(t *testing.T) {
		var _limit int = 100
		var limit *int = &_limit
		var page *int = nil
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.Nil(t, err)
	})

	t.Run("Limit is nil, page is within range", func(t *testing.T) {
		var _page int = 2
		var limit *int = nil
		var page *int = &_page
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.Nil(t, err)
	})

	t.Run("Limit and page are outside valid range", func(t *testing.T) {
		var _limit int = 2000
		var _page int = -3
		var limit *int = &_limit
		var page *int = &_page
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.NotNil(t, err)
	})

	t.Run("Limit is outside valid range", func(t *testing.T) {
		var _limit int = 2000
		var _page int = 2
		var limit *int = &_limit
		var page *int = &_page
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.NotNil(t, err)
	})

	t.Run("Page is outside valid range", func(t *testing.T) {
		var _limit int = 50
		var _page int = -2
		var limit *int = &_limit
		var page *int = &_page
		err := service.VerifyLimitsWithOverride(limit, page)
		assert.NotNil(t, err)
	})

}

func Test_GeneratePaginationLimits(t *testing.T) {
	t.Run("Pagination limits OK", func(t *testing.T) {
		var limit int = 10
		var userIdToRolesLength int = 10
		var page int = 3
		var invitesLength int = 100

		start, end := service.GeneratePaginationLimits(limit, userIdToRolesLength, page, invitesLength)
		assert.Equal(t, 10, start)
		assert.Equal(t, 20, end)
	})

	t.Run("Pagination limits OK 2", func(t *testing.T) {
		var limit int = 10
		var userIdToRolesLength int = 10
		var page int = 30
		var invitesLength int = 100

		start, end := service.GeneratePaginationLimits(limit, userIdToRolesLength, page, invitesLength)
		assert.Equal(t, 100, start)
		assert.Equal(t, 100, end)
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

func Test_filterInvites(t *testing.T) {
	t.Run("Filter invites by email", func(t *testing.T) {
		invites := []db.Invite{
			{Email: "test1@example.com"},
			{Email: "test2@example.com"},
			{Email: "sample@example.com"},
		}
		searchTerm := "test"
		result := service.FilterInvites(invites, searchTerm)
		expected := []db.Invite{
			{Email: "test1@example.com"},
			{Email: "test2@example.com"},
		}
		assert.Equal(t, expected, result)
	})

	t.Run("No invites match search term", func(t *testing.T) {
		invites := []db.Invite{
			{Email: "test1@example.com"},
			{Email: "test2@example.com"},
			{Email: "sample@example.com"},
		}
		searchTerm := "nomatch"
		result := service.FilterInvites(invites, searchTerm)
		expected := []db.Invite{}
		assert.Equal(t, expected, result)
	})

	t.Run("Empty search term returns all invites", func(t *testing.T) {
		invites := []db.Invite{
			{Email: "test1@example.com"},
			{Email: "test2@example.com"},
			{Email: "sample@example.com"},
		}
		searchTerm := ""
		result := service.FilterInvites(invites, searchTerm)
		expected := invites
		assert.Equal(t, expected, result)
	})
}
