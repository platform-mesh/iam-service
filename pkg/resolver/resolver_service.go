package resolver

import (
	"context"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	pmcontext "github.com/platform-mesh/golang-commons/context"

	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/resolver/api"
	"github.com/platform-mesh/iam-service/pkg/resolver/errors"
	"github.com/platform-mesh/iam-service/pkg/service/fga"
	"github.com/platform-mesh/iam-service/pkg/service/keycloak"
	"github.com/platform-mesh/iam-service/pkg/sorter"
)

var _ api.ResolverService = (*Service)(nil)

type Service struct {
	fgaService      *fga.Service
	keycloakService *keycloak.Service
	userSorter      sorter.UserSorter
}

func (s *Service) Me(ctx context.Context) (*graph.User, error) {
	// Get Current User
	webToken, err := pmcontext.GetWebTokenFromContext(ctx)
	if err != nil {
		return nil, errors.InternalError
	}
	return s.keycloakService.UserByMail(ctx, webToken.Mail)
}

func (s *Service) User(ctx context.Context, userID string) (*graph.User, error) {
	return s.keycloakService.UserByMail(ctx, userID)
}

func (s *Service) Users(ctx context.Context, context graph.ResourceContext, roleFilters []string, sortBy *graph.SortByInput, page *graph.PageInput) (*graph.UserConnection, error) {
	// Retrieve users with roles from fga
	allUserRoles, err := s.fgaService.ListUsers(ctx, context, roleFilters)
	if err != nil {
		return nil, err
	}

	// Fill users from keycloak with metadata using parallel processing
	s.enrichUsersWithKeycloakData(ctx, allUserRoles)

	// Apply sorting
	s.userSorter.SortUserRoles(allUserRoles, sortBy)

	totalCount := len(allUserRoles)

	// Apply pagination
	paginatedUserRoles, pageInfo := s.applyPagination(allUserRoles, page, totalCount)

	return &graph.UserConnection{Users: paginatedUserRoles, PageInfo: pageInfo}, nil
}

// enrichUsersWithKeycloakData enriches user data with information from Keycloak using batch call
func (s *Service) enrichUsersWithKeycloakData(ctx context.Context, userRoles []*graph.UserRoles) {
	if len(userRoles) == 0 {
		return
	}

	// Extract unique email addresses from user roles
	emailSet := make(map[string]bool)
	var emails []string

	for _, userRole := range userRoles {
		if userRole.User != nil && userRole.User.Email != "" {
			if !emailSet[userRole.User.Email] {
				emailSet[userRole.User.Email] = true
				emails = append(emails, userRole.User.Email)
			}
		}
	}

	if len(emails) == 0 {
		return
	}

	// Batch call to get all users at once
	userMap, err := s.keycloakService.GetUsersByEmails(ctx, emails)
	if err != nil {
		// Log error but continue with partial data
		// In a production system, you might want to add proper logging here
		return
	}

	// Update user roles with Keycloak data using the lookup map
	for _, userRole := range userRoles {
		if userRole.User != nil && userRole.User.Email != "" {
			if keycloakUser, exists := userMap[userRole.User.Email]; exists {
				// Update the user with complete information from Keycloak
				userRole.User.UserID = keycloakUser.UserID
				userRole.User.FirstName = keycloakUser.FirstName
				userRole.User.LastName = keycloakUser.LastName
				// Email is already set from OpenFGA
			}
		}
	}
}

// applyPagination applies pagination logic to the user roles list and returns the paginated slice and PageInfo
func (s *Service) applyPagination(allUserRoles []*graph.UserRoles, page *graph.PageInput, totalCount int) ([]*graph.UserRoles, *graph.PageInfo) {
	// Default pagination values
	defaultLimit := 10
	defaultPage := 1

	// Extract pagination parameters
	limit := defaultLimit
	if page != nil && page.Limit != nil {
		limit = *page.Limit
	}

	pageNum := defaultPage
	if page != nil && page.Page != nil {
		pageNum = *page.Page
	}

	// Ensure minimum values
	if limit < 1 {
		limit = defaultLimit
	}
	if pageNum < 1 {
		pageNum = defaultPage
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

func NewResolverService(fgaClient openfgav1.OpenFGAServiceClient, service *keycloak.Service) *Service {
	return &Service{
		fgaService:      fga.New(fgaClient),
		keycloakService: service,
		userSorter:      sorter.NewUserSorter(),
	}
}
