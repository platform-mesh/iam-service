package sorter

import (
	"sort"
	"strings"

	"github.com/platform-mesh/iam-service/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/graph"
)

// UserSorter defines the interface for sorting user-related data
type UserSorter interface {
	// SortUserRoles sorts a slice of UserRoles based on the provided sort criteria
	// If sortBy is nil, applies default sorting (LastName ASC)
	SortUserRoles(userRoles []*graph.UserRoles, sortBy *graph.SortByInput)
}

// DefaultUserSorter provides the default implementation for user sorting
type DefaultUserSorter struct {
	defaultField     graph.UserSortField
	defaultDirection graph.SortDirection
}

// NewUserSorter creates a new instance of DefaultUserSorter with default values
func NewUserSorter() UserSorter {
	return &DefaultUserSorter{
		defaultField:     graph.UserSortFieldLastName,
		defaultDirection: graph.SortDirectionAsc,
	}
}

// NewUserSorterWithConfig creates a new instance of DefaultUserSorter with configurable values
func NewUserSorterWithConfig(cfg *config.ServiceConfig) UserSorter {
	return &DefaultUserSorter{
		defaultField:     parseUserSortField(cfg.Sorting.DefaultField),
		defaultDirection: parseSortDirection(cfg.Sorting.DefaultDirection),
	}
}

// SortUserRoles sorts the user roles list based on the sortBy parameter
// If sortBy is nil, defaults to sorting by LastName in ascending order
func (s *DefaultUserSorter) SortUserRoles(userRoles []*graph.UserRoles, sortBy *graph.SortByInput) {
	if len(userRoles) <= 1 {
		return
	}

	// Use configured defaults
	field := s.defaultField
	direction := s.defaultDirection

	// Override with provided sortBy if available
	if sortBy != nil {
		field = sortBy.Field
		direction = sortBy.Direction
	}

	// Perform sorting using the sort package
	sort.Slice(userRoles, func(i, j int) bool {
		userI := userRoles[i].User
		userJ := userRoles[j].User

		compareResult := s.compareUsers(userI, userJ, field)

		// Apply direction
		if direction == graph.SortDirectionDesc {
			return compareResult > 0
		}
		return compareResult < 0
	})
}

// compareUsers compares two users based on the specified field
// Returns:
//   - negative value if userI < userJ
//   - zero if userI == userJ
//   - positive value if userI > userJ
func (s *DefaultUserSorter) compareUsers(userI, userJ *graph.User, field graph.UserSortField) int {
	switch field {
	case graph.UserSortFieldUserID:
		return strings.Compare(userI.UserID, userJ.UserID)
	case graph.UserSortFieldEmail:
		return strings.Compare(userI.Email, userJ.Email)
	case graph.UserSortFieldFirstName:
		return strings.Compare(s.getStringValue(userI.FirstName), s.getStringValue(userJ.FirstName))
	case graph.UserSortFieldLastName:
		return strings.Compare(s.getStringValue(userI.LastName), s.getStringValue(userJ.LastName))
	default:
		// Fallback to LastName if invalid field
		return strings.Compare(s.getStringValue(userI.LastName), s.getStringValue(userJ.LastName))
	}
}

// getStringValue safely extracts string value from a string pointer
// Returns empty string if the pointer is nil
func (s *DefaultUserSorter) getStringValue(strPtr *string) string {
	if strPtr == nil {
		return ""
	}
	return *strPtr
}

// parseUserSortField converts a string configuration to UserSortField
func parseUserSortField(field string) graph.UserSortField {
	switch strings.ToLower(field) {
	case "userid", "user_id":
		return graph.UserSortFieldUserID
	case "email":
		return graph.UserSortFieldEmail
	case "firstname", "first_name":
		return graph.UserSortFieldFirstName
	case "lastname", "last_name":
		return graph.UserSortFieldLastName
	default:
		// Default to LastName if invalid field
		return graph.UserSortFieldLastName
	}
}

// parseSortDirection converts a string configuration to SortDirection
func parseSortDirection(direction string) graph.SortDirection {
	switch strings.ToUpper(direction) {
	case "DESC", "DESCENDING":
		return graph.SortDirectionDesc
	case "ASC", "ASCENDING":
		return graph.SortDirectionAsc
	default:
		// Default to ASC if invalid direction
		return graph.SortDirectionAsc
	}
}
