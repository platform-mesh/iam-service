package service

import (
	"context"
	"math"
	"strings"

	"github.com/pkg/errors"

	"github.com/openmfp/golang-commons/context/keys"
	"github.com/openmfp/golang-commons/logger"
	"github.com/openmfp/iam-service/pkg/db"
	"github.com/openmfp/iam-service/pkg/graph"
)

const MAX_INT = math.MaxInt

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
		*limit = MAX_INT
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

func matchSearchTerm(user *graph.User, s *string) bool {
	if s == nil {
		return true
	}
	if strings.Contains(strings.ToLower(user.Email), strings.ToLower(*s)) {
		return true
	}
	if user.FirstName != nil && *user.FirstName != "" && strings.Contains(strings.ToLower(*user.FirstName), strings.ToLower(*s)) {
		return true
	}
	if user.LastName != nil && *user.LastName != "" && strings.Contains(strings.ToLower(*user.LastName), strings.ToLower(*s)) {
		return true
	}
	if strings.Contains(strings.ToLower(user.UserID), strings.ToLower(*s)) {
		return true
	}
	return false
}

func CheckFilterRoles(userRoles []*graph.Role, searchfilterRoles []*graph.RoleInput) bool {
	if len(searchfilterRoles) == 0 {
		return true
	}
	for _, searchRole := range searchfilterRoles {
		for _, userRole := range userRoles {
			if searchRole.TechnicalName == userRole.TechnicalName {
				return true
			}
		}
	}
	return false
}

func FilterInvites(invites []db.Invite, s string) []db.Invite {
	out := []db.Invite{}
	for _, invite := range invites {
		if strings.Contains(strings.ToLower(invite.Email), strings.ToLower(s)) {
			out = append(out, invite)
		}
	}
	return out
}
