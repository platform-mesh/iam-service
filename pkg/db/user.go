package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/openmfp/golang-commons/sentry"

	"github.com/openmfp/iam-service/pkg/graph"
)

// GetUserByID  returns a member by ID
func (d *Database) GetUserByID(ctx context.Context, tenantID string, userID string) (*graph.User, error) {
	var existingUser graph.User
	err := d.db.
		Where("tenant_id = ?", tenantID).
		Where("user_id = ?", userID).
		First(&existingUser).Error
	if err != nil {
		return nil, err
	}

	return &existingUser, nil
}

// GetUsersByUserIDs returns a member by ID, limiting response to the current page contents
// and using a search filter for user_id, first_name, last_name and email
func (d *Database) GetUsersByUserIDs(
	ctx context.Context, tenantID string, userIDs []string, limit, page int, searchTerm *string,
	sortBy *graph.SortBy,
) ([]*graph.User, error) {

	orderString := "first_name, last_name desc"
	var users []*graph.User
	if sortBy != nil {
		if sortBy.Direction != "asc" && sortBy.Direction != "desc" {
			return nil, errors.New("invalid sort direction")
		}
		orderString = fmt.Sprintf("first_name %s, last_name %s", sortBy.Direction, sortBy.Direction)
		d.logger.Debug().Str("direction", string(sortBy.Direction)).Str("orderString", orderString).Msg("GetUsersByUserIDs")
	}

	var query *gorm.DB
	if searchTerm == nil {
		query = d.db.
			Where("tenant_id = ?", tenantID).
			Where("user_id in ?", userIDs).
			Order(orderString)
	} else {
		dbLikeString := fmt.Sprintf("%%%s%%", *searchTerm)
		d.logger.Debug().Str("searchTerm", *searchTerm).Str("dbLikeString", dbLikeString).Msg("GetUsersByUserIDs")
		query = d.db.
			Where("tenant_id = ?", tenantID).
			Where("user_id in ?", userIDs).
			Where("user_id LIKE ? OR first_name LIKE ? OR last_name LIKE ? OR email LIKE ?", dbLikeString, dbLikeString, dbLikeString, dbLikeString).
			Order(orderString)
	}

	if limit > 0 {
		offset := (limit * page) - limit

		if offset < 0 {
			offset = 0
		}

		query.Limit(limit).Offset(offset)
	}

	err := query.Find(&users).Error

	return users, err
}

// GetUserByIDOrEmail returns a member by ID or email
func (d *Database) getUserByIDOrEmail(
	ctx context.Context, tenantID string, userID string, email string, // nolint: unparam
) (*graph.User, error) {
	if userID == "" && email == "" {
		return nil, errors.New("at least one of userId or a valid email has to be provided")
	}

	userSelector := d.db
	if userID != "" {
		userSelector = userSelector.Or("user_id = ?", userID)
	}
	if email != "" {
		userSelector = userSelector.Or("email = ?", email)
	}

	var existingUser graph.User
	err := d.db.
		Where("tenant_id = ?", tenantID).
		Where(userSelector).
		First(&existingUser).Error
	if err != nil {
		return nil, err
	}

	return &existingUser, nil
}

// GetUserByEmail returns user by email
func (d *Database) GetUserByEmail(ctx context.Context, tenantID string, email string) (*graph.User, error) {
	var existingUser graph.User
	err := d.db.
		Where("tenant_id = ?", tenantID).
		Where("email = ?", email).
		First(&existingUser).Error
	if err != nil {
		return nil, err
	}

	return &existingUser, nil
}

func (d *Database) GetOrCreateUser(ctx context.Context, tenantID string, input graph.UserInput) (*graph.User, error) {
	// Check if user already exists
	existingUser, err := d.getUserByIDOrEmail(ctx, tenantID, input.UserID, input.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		user := strings.Join([]string{input.UserID, input.Email}, ":")
		return nil, errors.Wrap(err, fmt.Sprintf("Failed to query for user: %s", user))
	}

	if existingUser != nil {
		return existingUser, nil
	}

	// User does not exist, go ahead and create it
	newUser := graph.User{
		UserID:                input.UserID,
		TenantID:              tenantID,
		Email:                 input.Email,
		FirstName:             input.FirstName,
		LastName:              input.LastName,
		InvitationOutstanding: input.InvitationOutstanding != nil && *input.InvitationOutstanding,
		Base: graph.Base{
			CreatedAt: time.Now().UTC(),
		},
	}

	err = d.db.Create(&newUser).Error
	if err != nil {
		return nil, errors.New("could not create user")
	}

	// Call UserCreated hook
	if d.userHooks != nil {
		d.userHooks.UserCreated(ctx, &newUser, tenantID)
	}

	return &newUser, err
}

// RemoveUser removes user
// it returns true if user entry is no longer exists.
func (d *Database) RemoveUser(ctx context.Context, tenantID string, userID string, email string) (bool, error) {
	if userID == "" && email == "" {
		return false, errors.New("either userId or email must be provided")
	}

	user, err := d.getUserByIDOrEmail(ctx, tenantID, userID, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, err
		}
		return false, err
	}

	err = d.db.Select("Teams").Delete(&user).Error
	if err != nil {
		d.logger.Error().Err(err).Str("tenant", tenantID).Str("user", user.ID).Str("email", user.Email).Msg("Failed to DeleteUser")
		sentry.CaptureError(err, sentry.Tags{"tenant": tenantID, "user": user.ID})
		return false, errors.New("could not delete user")
	}

	if d.userHooks != nil {
		d.userHooks.UserRemoved(ctx, user, tenantID)
	}

	return true, nil
}

func (d *Database) Save(user *graph.User) error {
	return d.db.Save(user).Error
}

// GetUsers returns all existing users for a tenant
func (d *Database) GetUsers(ctx context.Context, tenantID string, limit int, page int) (*graph.UserConnection, error) {
	var users []*graph.User
	offset := (limit * page) - limit
	err := d.db.Limit(limit).Offset(offset).
		Preload(clause.Associations).
		Where("tenant_id = ?", tenantID).
		Find(&users).Error

	var totalCount int64
	d.db.Model(users).Where("tenant_id = ?", tenantID).Count(&totalCount)

	userConnection := graph.UserConnection{
		User:     users,
		PageInfo: &graph.PageInfo{TotalCount: int(totalCount)},
	}

	return &userConnection, err
}

func (d *Database) SearchUsers(ctx context.Context, tenantId, query string) ([]*graph.User, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.cfg.MaxSearchUsersTimeout)*time.Second)
	defer cancel()

	fuzzedQuery := "%" + strings.ToLower(query) + "%"

	var users []*graph.User
	// don't use ILIKE, it is 2-3 times slower than combination of LOWER and LIKE
	// If this query becomes slow, consider using fuzzy search.
	// For this you need to enable pg_trgm extension and add GIN index on the columns you want to search.
	err := d.db.WithContext(ctx).
		Where("tenant_id = ?", tenantId).
		Where(
			d.db.Where("LOWER(user_id) LIKE ?", fuzzedQuery).
				Or("LOWER(email) LIKE ?", fuzzedQuery).
				Or("first_name IS NOT NULL AND LOWER(first_name) LIKE ?", fuzzedQuery).
				Or("last_name IS NOT NULL AND LOWER(last_name) LIKE ?", fuzzedQuery),
		).
		Limit(d.cfg.MaxSearchUsersLimit).
		Find(&users).Error

	return users, err
}
