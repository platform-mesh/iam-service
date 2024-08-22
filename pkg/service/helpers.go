package service

import (
	"context"
	"math"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/openmfp/golang-commons/logger"
	"github.com/pkg/errors"
)

func setupLogger(ctx context.Context) *logger.Logger {
	log := logger.LoadLoggerFromContext(ctx)

	requestID := middleware.GetReqID(ctx)

	return logger.NewFromZerolog(
		log.With().Str("requestid", requestID).Logger(),
	)
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
