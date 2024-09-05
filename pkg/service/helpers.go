package service

import (
	"context"
	"math"

	"github.com/pkg/errors"

	"github.com/openmfp/golang-commons/context/keys"
	"github.com/openmfp/golang-commons/logger"
)

func setupLogger(ctx context.Context) *logger.Logger {
	log := logger.LoadLoggerFromContext(ctx)

	requestID := getRequestId(ctx)

	return logger.NewFromZerolog(
		log.With().Str("request_id", requestID).Logger(),
	)
}

func getRequestId(ctx context.Context) string {
	if val, ok := ctx.Value(keys.RequestIdCtxKey).(string); ok {
		return val
	}
	return "no_request_id_error"
}

func VerifyLimitsWithOverride(limit *int, page *int) error {
	zero := 0
	minusOne := -1

	if limit == nil {
		limit = &minusOne
	}
	if page == nil {
		page = &zero
	}

	if *limit == -1 {
		*page = 0
		*limit = math.MaxInt
		return nil
	}
	if *page < 0 {
		return errors.Errorf("page: page cannot be smaller than 0")
	}
	if *limit < 1 || *limit > 1000 {
		return errors.Errorf("limit: limit cannot be smaller than 1 or greater than 1000")
	}
	return nil
}

func GeneratePaginationLimits(limit int, userIdToRolesLength int, page int, invitesLength int) (int, int) {
	memberPages := int(math.Ceil(float64(userIdToRolesLength) / float64(limit)))
	freeSlots := limit*(memberPages) - userIdToRolesLength

	sliceStart := limit*(page-memberPages) - (limit - freeSlots)
	sliceStart = maxInt(0, sliceStart)
	sliceEnd := limit*(page-memberPages) + freeSlots
	sliceEnd = minInt(sliceEnd, invitesLength)

	sliceEnd = maxInt(sliceEnd, 0)
	sliceStart = minInt(sliceStart, sliceEnd)

	return sliceStart, sliceEnd
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
