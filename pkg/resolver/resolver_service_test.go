package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/sorter"
)

func TestService_applySorting_DefaultLastNameAsc(t *testing.T) {
	userSorter := sorter.NewUserSorter()

	// Create test data with different last names
	userRoles := []*graph.UserRoles{
		{
			User: &graph.User{
				UserID:    "3",
				Email:     "charlie@example.com",
				FirstName: stringPtr("Charlie"),
				LastName:  stringPtr("Wilson"),
			},
		},
		{
			User: &graph.User{
				UserID:    "1",
				Email:     "alice@example.com",
				FirstName: stringPtr("Alice"),
				LastName:  stringPtr("Anderson"),
			},
		},
		{
			User: &graph.User{
				UserID:    "2",
				Email:     "bob@example.com",
				FirstName: stringPtr("Bob"),
				LastName:  stringPtr("Brown"),
			},
		},
	}

	// Apply default sorting (should be LastName ASC)
	userSorter.SortUserRoles(userRoles, nil)

	// Verify order: Anderson, Brown, Wilson
	assert.Equal(t, "Anderson", *userRoles[0].User.LastName)
	assert.Equal(t, "Brown", *userRoles[1].User.LastName)
	assert.Equal(t, "Wilson", *userRoles[2].User.LastName)
}

func TestService_applySorting_FirstNameDesc(t *testing.T) {
	userSorter := sorter.NewUserSorter()

	// Create test data
	userRoles := []*graph.UserRoles{
		{
			User: &graph.User{
				UserID:    "1",
				Email:     "alice@example.com",
				FirstName: stringPtr("Alice"),
				LastName:  stringPtr("Anderson"),
			},
		},
		{
			User: &graph.User{
				UserID:    "3",
				Email:     "charlie@example.com",
				FirstName: stringPtr("Charlie"),
				LastName:  stringPtr("Wilson"),
			},
		},
		{
			User: &graph.User{
				UserID:    "2",
				Email:     "bob@example.com",
				FirstName: stringPtr("Bob"),
				LastName:  stringPtr("Brown"),
			},
		},
	}

	sortBy := &graph.SortByInput{
		Field:     graph.UserSortFieldFirstName,
		Direction: graph.SortDirectionDesc,
	}

	userSorter.SortUserRoles(userRoles, sortBy)

	// Verify order: Charlie, Bob, Alice (FirstName DESC)
	assert.Equal(t, "Charlie", *userRoles[0].User.FirstName)
	assert.Equal(t, "Bob", *userRoles[1].User.FirstName)
	assert.Equal(t, "Alice", *userRoles[2].User.FirstName)
}

func TestService_applySorting_EmailAsc(t *testing.T) {
	userSorter := sorter.NewUserSorter()

	// Create test data
	userRoles := []*graph.UserRoles{
		{
			User: &graph.User{
				UserID:    "3",
				Email:     "charlie@example.com",
				FirstName: stringPtr("Charlie"),
				LastName:  stringPtr("Wilson"),
			},
		},
		{
			User: &graph.User{
				UserID:    "1",
				Email:     "alice@example.com",
				FirstName: stringPtr("Alice"),
				LastName:  stringPtr("Anderson"),
			},
		},
		{
			User: &graph.User{
				UserID:    "2",
				Email:     "bob@example.com",
				FirstName: stringPtr("Bob"),
				LastName:  stringPtr("Brown"),
			},
		},
	}

	sortBy := &graph.SortByInput{
		Field:     graph.UserSortFieldEmail,
		Direction: graph.SortDirectionAsc,
	}

	userSorter.SortUserRoles(userRoles, sortBy)

	// Verify order: alice, bob, charlie (Email ASC)
	assert.Equal(t, "alice@example.com", userRoles[0].User.Email)
	assert.Equal(t, "bob@example.com", userRoles[1].User.Email)
	assert.Equal(t, "charlie@example.com", userRoles[2].User.Email)
}

func TestService_applySorting_NilValues(t *testing.T) {
	userSorter := sorter.NewUserSorter()

	// Create test data with nil first/last names
	userRoles := []*graph.UserRoles{
		{
			User: &graph.User{
				UserID:    "1",
				Email:     "user1@example.com",
				FirstName: stringPtr("Alice"),
				LastName:  nil, // nil LastName
			},
		},
		{
			User: &graph.User{
				UserID:    "2",
				Email:     "user2@example.com",
				FirstName: nil, // nil FirstName
				LastName:  stringPtr("Brown"),
			},
		},
		{
			User: &graph.User{
				UserID:    "3",
				Email:     "user3@example.com",
				FirstName: stringPtr("Charlie"),
				LastName:  stringPtr("Wilson"),
			},
		},
	}

	// Sort by LastName ASC (default)
	userSorter.SortUserRoles(userRoles, nil)

	// Nil values should sort first (empty string comparison)
	// Order should be: user1 (nil LastName), user2 (Brown), user3 (Wilson)
	assert.Equal(t, "user1@example.com", userRoles[0].User.Email)
	assert.Equal(t, "user2@example.com", userRoles[1].User.Email)
	assert.Equal(t, "user3@example.com", userRoles[2].User.Email)
}

func TestService_applySorting_EmptyList(t *testing.T) {
	userSorter := sorter.NewUserSorter()

	userRoles := []*graph.UserRoles{}

	// Should not panic with empty list
	userSorter.SortUserRoles(userRoles, nil)

	assert.Equal(t, 0, len(userRoles))
}

func TestService_applySorting_SingleItem(t *testing.T) {
	userSorter := sorter.NewUserSorter()

	userRoles := []*graph.UserRoles{
		{
			User: &graph.User{
				UserID:    "1",
				Email:     "user@example.com",
				FirstName: stringPtr("Test"),
				LastName:  stringPtr("User"),
			},
		},
	}

	// Should not panic with single item
	userSorter.SortUserRoles(userRoles, nil)

	assert.Equal(t, 1, len(userRoles))
	assert.Equal(t, "user@example.com", userRoles[0].User.Email)
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Helper function to create test user roles data
func createTestUserRoles(count int) []*graph.UserRoles {
	userRoles := make([]*graph.UserRoles, count)
	for i := 0; i < count; i++ {
		userRoles[i] = &graph.UserRoles{
			User: &graph.User{
				UserID: "",
				Email:  "user" + string(rune('0'+i%10)) + "@example.com", // Simple pattern for testing
			},
			Roles: []*graph.Role{
				{
					ID:          "member",
					DisplayName: "Member",
					Description: "Limited access to resources",
				},
			},
		}
	}
	return userRoles
}
