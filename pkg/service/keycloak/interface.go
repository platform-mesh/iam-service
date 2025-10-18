package keycloak

import (
	"context"

	"github.com/platform-mesh/iam-service/pkg/graph"
)

// KeycloakService defines the interface for Keycloak user management operations
type KeycloakService interface {
	// UserByMail retrieves a user by their email address
	UserByMail(ctx context.Context, userID string) (*graph.User, error)

	// GetUsersByEmails retrieves multiple users by their email addresses in a single batch call
	// Returns a map of email -> User for efficient lookups
	GetUsersByEmails(ctx context.Context, emails []string) (map[string]*graph.User, error)
}

// Ensure Service implements KeycloakService interface
var _ KeycloakService = (*Service)(nil)
