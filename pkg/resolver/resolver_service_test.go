package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/platform-mesh/iam-service/pkg/graph"
)

func TestService_applyPagination_DefaultValues(t *testing.T) {
	service := &Service{}

	// Create test data
	userRoles := createTestUserRoles(25) // 25 users

	// Test with nil page input (should use defaults)
	paginatedUsers, pageInfo := service.applyPagination(userRoles, nil, len(userRoles))

	// Should return first 10 users (default limit)
	assert.Equal(t, 10, len(paginatedUsers))
	assert.Equal(t, 10, pageInfo.Count)
	assert.Equal(t, 25, pageInfo.TotalCount)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)

	// Verify we got the first 10 users
	for i, userRole := range paginatedUsers {
		expectedEmail := userRoles[i].User.Email
		assert.Equal(t, expectedEmail, userRole.User.Email)
	}
}

func TestService_applyPagination_CustomLimitAndPage(t *testing.T) {
	service := &Service{}

	// Create test data
	userRoles := createTestUserRoles(25) // 25 users

	limit := 5
	page := 3
	pageInput := &graph.PageInput{
		Limit: &limit,
		Page:  &page,
	}

	// Test with custom page input
	paginatedUsers, pageInfo := service.applyPagination(userRoles, pageInput, len(userRoles))

	// Should return 5 users from page 3 (users 10-14, 0-indexed)
	assert.Equal(t, 5, len(paginatedUsers))
	assert.Equal(t, 5, pageInfo.Count)
	assert.Equal(t, 25, pageInfo.TotalCount)
	assert.True(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)

	// Verify we got the correct users (page 3 with limit 5 = offset 10)
	expectedOffset := (page - 1) * limit // (3-1) * 5 = 10
	for i, userRole := range paginatedUsers {
		expectedEmail := userRoles[expectedOffset+i].User.Email
		assert.Equal(t, expectedEmail, userRole.User.Email)
	}
}

func TestService_applyPagination_LastPage(t *testing.T) {
	service := &Service{}

	// Create test data
	userRoles := createTestUserRoles(23) // 23 users

	limit := 10
	page := 3 // Page 3 with limit 10 should have 3 users (20-22)
	pageInput := &graph.PageInput{
		Limit: &limit,
		Page:  &page,
	}

	paginatedUsers, pageInfo := service.applyPagination(userRoles, pageInput, len(userRoles))

	// Should return 3 users (the remainder)
	assert.Equal(t, 3, len(paginatedUsers))
	assert.Equal(t, 3, pageInfo.Count)
	assert.Equal(t, 23, pageInfo.TotalCount)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
}

func TestService_applyPagination_EmptyResults(t *testing.T) {
	service := &Service{}

	// Empty user roles
	userRoles := []*graph.UserRoles{}

	paginatedUsers, pageInfo := service.applyPagination(userRoles, nil, 0)

	assert.Equal(t, 0, len(paginatedUsers))
	assert.Equal(t, 0, pageInfo.Count)
	assert.Equal(t, 0, pageInfo.TotalCount)
	assert.False(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
}

func TestService_applyPagination_PageBeyondTotal(t *testing.T) {
	service := &Service{}

	// Create test data
	userRoles := createTestUserRoles(5) // 5 users

	limit := 10
	page := 2 // Page 2 with limit 10 and only 5 users total
	pageInput := &graph.PageInput{
		Limit: &limit,
		Page:  &page,
	}

	paginatedUsers, pageInfo := service.applyPagination(userRoles, pageInput, len(userRoles))

	// Should return empty results but maintain correct pagination info
	assert.Equal(t, 0, len(paginatedUsers))
	assert.Equal(t, 0, pageInfo.Count)
	assert.Equal(t, 5, pageInfo.TotalCount)
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)
}

func TestService_applyPagination_InvalidValues(t *testing.T) {
	service := &Service{}

	// Create test data
	userRoles := createTestUserRoles(15) // 15 users

	// Test with invalid limit and page values
	limit := -5
	page := 0
	pageInput := &graph.PageInput{
		Limit: &limit,
		Page:  &page,
	}

	paginatedUsers, pageInfo := service.applyPagination(userRoles, pageInput, len(userRoles))

	// Should use default values (limit=10, page=1)
	assert.Equal(t, 10, len(paginatedUsers))
	assert.Equal(t, 10, pageInfo.Count)
	assert.Equal(t, 15, pageInfo.TotalCount)
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)
}

func TestService_applySorting_DefaultLastNameAsc(t *testing.T) {
	service := &Service{}

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
	service.applySorting(userRoles, nil)

	// Verify order: Anderson, Brown, Wilson
	assert.Equal(t, "Anderson", *userRoles[0].User.LastName)
	assert.Equal(t, "Brown", *userRoles[1].User.LastName)
	assert.Equal(t, "Wilson", *userRoles[2].User.LastName)
}

func TestService_applySorting_FirstNameDesc(t *testing.T) {
	service := &Service{}

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

	service.applySorting(userRoles, sortBy)

	// Verify order: Charlie, Bob, Alice (FirstName DESC)
	assert.Equal(t, "Charlie", *userRoles[0].User.FirstName)
	assert.Equal(t, "Bob", *userRoles[1].User.FirstName)
	assert.Equal(t, "Alice", *userRoles[2].User.FirstName)
}

func TestService_applySorting_EmailAsc(t *testing.T) {
	service := &Service{}

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

	service.applySorting(userRoles, sortBy)

	// Verify order: alice, bob, charlie (Email ASC)
	assert.Equal(t, "alice@example.com", userRoles[0].User.Email)
	assert.Equal(t, "bob@example.com", userRoles[1].User.Email)
	assert.Equal(t, "charlie@example.com", userRoles[2].User.Email)
}

func TestService_applySorting_NilValues(t *testing.T) {
	service := &Service{}

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
	service.applySorting(userRoles, nil)

	// Nil values should sort first (empty string comparison)
	// Order should be: user1 (nil LastName), user2 (Brown), user3 (Wilson)
	assert.Equal(t, "user1@example.com", userRoles[0].User.Email)
	assert.Equal(t, "user2@example.com", userRoles[1].User.Email)
	assert.Equal(t, "user3@example.com", userRoles[2].User.Email)
}

func TestService_applySorting_EmptyList(t *testing.T) {
	service := &Service{}

	userRoles := []*graph.UserRoles{}

	// Should not panic with empty list
	service.applySorting(userRoles, nil)

	assert.Equal(t, 0, len(userRoles))
}

func TestService_applySorting_SingleItem(t *testing.T) {
	service := &Service{}

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
	service.applySorting(userRoles, nil)

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
					TechnicalName: "member",
					DisplayName:   "Member",
				},
			},
		}
	}
	return userRoles
}
