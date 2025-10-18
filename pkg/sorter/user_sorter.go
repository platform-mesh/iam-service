package sorter

import (
	"sort"
	"strings"

	"github.com/platform-mesh/iam-service/pkg/graph"
)

// UserSorter defines the interface for sorting user-related data
type UserSorter interface {
	// SortUserRoles sorts a slice of UserRoles based on the provided sort criteria
	// If sortBy is nil, applies default sorting (LastName ASC)
	SortUserRoles(userRoles []*graph.UserRoles, sortBy *graph.SortByInput)
}

// DefaultUserSorter provides the default implementation for user sorting
type DefaultUserSorter struct{}

// NewUserSorter creates a new instance of DefaultUserSorter
func NewUserSorter() UserSorter {
	return &DefaultUserSorter{}
}

// SortUserRoles sorts the user roles list based on the sortBy parameter
// If sortBy is nil, defaults to sorting by LastName in ascending order
func (s *DefaultUserSorter) SortUserRoles(userRoles []*graph.UserRoles, sortBy *graph.SortByInput) {
	if len(userRoles) <= 1 {
		return
	}

	// Default sorting: LastName ASC
	field := graph.UserSortFieldLastName
	direction := graph.SortDirectionAsc

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
