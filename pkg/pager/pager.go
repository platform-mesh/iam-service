package pager

import "github.com/platform-mesh/iam-service/pkg/graph"

// Pager provides pagination functionality for slices
type Pager interface {
	// PaginateUserRoles applies pagination to a slice of user roles
	PaginateUserRoles(allUserRoles []*graph.UserRoles, page *graph.PageInput, totalCount int) ([]*graph.UserRoles, *graph.PageInfo)
}

// DefaultPager implements the Pager interface with standard pagination logic
type DefaultPager struct {
	defaultLimit int
	defaultPage  int
}

// NewDefaultPager creates a new DefaultPager with default values
func NewDefaultPager() *DefaultPager {
	return &DefaultPager{
		defaultLimit: 10,
		defaultPage:  1,
	}
}

// PaginateUserRoles applies pagination logic to the user roles list and returns the paginated slice and PageInfo
func (p *DefaultPager) PaginateUserRoles(allUserRoles []*graph.UserRoles, page *graph.PageInput, totalCount int) ([]*graph.UserRoles, *graph.PageInfo) {
	// Extract pagination parameters
	limit := p.defaultLimit
	if page != nil && page.Limit != nil {
		limit = *page.Limit
	}

	pageNum := p.defaultPage
	if page != nil && page.Page != nil {
		pageNum = *page.Page
	}

	// Ensure minimum values
	if limit < 1 {
		limit = p.defaultLimit
	}
	if pageNum < 1 {
		pageNum = p.defaultPage
	}

	// Calculate pagination bounds
	offset := (pageNum - 1) * limit
	end := offset + limit

	// Handle empty result set
	if totalCount == 0 {
		return []*graph.UserRoles{}, &graph.PageInfo{
			Count:           0,
			TotalCount:      0,
			HasNextPage:     false,
			HasPreviousPage: false,
		}
	}

	// Handle offset beyond total count
	if offset >= totalCount {
		return []*graph.UserRoles{}, &graph.PageInfo{
			Count:           0,
			TotalCount:      totalCount,
			HasNextPage:     false,
			HasPreviousPage: pageNum > 1,
		}
	}

	// Adjust end boundary
	if end > totalCount {
		end = totalCount
	}

	// Extract the paginated slice
	paginatedUserRoles := allUserRoles[offset:end]

	// Calculate pagination info
	count := len(paginatedUserRoles)
	hasNextPage := end < totalCount
	hasPreviousPage := pageNum > 1

	pageInfo := &graph.PageInfo{
		Count:           count,
		TotalCount:      totalCount,
		HasNextPage:     hasNextPage,
		HasPreviousPage: hasPreviousPage,
	}

	return paginatedUserRoles, pageInfo
}
