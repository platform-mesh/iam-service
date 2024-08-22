package service_test

import (
	"testing"

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
