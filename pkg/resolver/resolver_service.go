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
)

var _ api.ResolverService = (*Service)(nil)

type Service struct {
	fgaService      *fga.Service
	keycloakService *keycloak.Service
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

func (s *Service) Users(ctx context.Context, context graph.ResourceContext, roleFilters []string, sortBy *graph.SortByInput) (*graph.UserConnection, error) {
	// Retrieve users with roles from fga
	userRoles, err := s.fgaService.ListUsers(ctx, context, roleFilters)
	if err != nil {
		return nil, err
	}

	// Fill users from keycloak with metadata using parallel processing
	s.enrichUsersWithKeycloakData(ctx, userRoles)

	nr := len(userRoles)
	return &graph.UserConnection{Users: userRoles, PageInfo: &graph.PageInfo{Count: nr, TotalCount: nr, HasNextPage: false, HasPreviousPage: false}}, nil
}

// enrichUsersWithKeycloakData enriches user data with information from Keycloak in parallel
func (s *Service) enrichUsersWithKeycloakData(ctx context.Context, userRoles []*graph.UserRoles) {
	if len(userRoles) == 0 {
		return
	}

	// Result structure for parallel processing
	type keycloakResult struct {
		index        int
		keycloakUser *graph.User
		err          error
	}

	// Create channel for goroutine communication
	resultChan := make(chan keycloakResult, len(userRoles))

	// Launch goroutines for each user
	for i, userRole := range userRoles {
		if userRole.User != nil && userRole.User.Email != "" {
			go func(index int, email string) {
				keycloakUser, err := s.keycloakService.UserByMail(ctx, email)
				resultChan <- keycloakResult{
					index:        index,
					keycloakUser: keycloakUser,
					err:          err,
				}
			}(i, userRole.User.Email)
		} else {
			// Send a nil result for users without email
			resultChan <- keycloakResult{index: i, keycloakUser: nil, err: nil}
		}
	}

	// Collect results from all goroutines
	for i := 0; i < len(userRoles); i++ {
		result := <-resultChan

		if result.err != nil {
			// Log error but continue with partial data - could add logging here
			continue
		}

		if result.keycloakUser != nil && result.index < len(userRoles) {
			userRole := userRoles[result.index]
			if userRole.User != nil {
				// Update the user with complete information from Keycloak
				userRole.User.UserID = result.keycloakUser.UserID
				userRole.User.FirstName = result.keycloakUser.FirstName
				userRole.User.LastName = result.keycloakUser.LastName
				// Email is already set from OpenFGA
			}
		}
	}
}

func NewResolverService(fgaClient openfgav1.OpenFGAServiceClient, service *keycloak.Service) *Service {
	return &Service{
		fgaService:      fga.New(fgaClient),
		keycloakService: service,
	}
}
