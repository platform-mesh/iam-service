package keycloak

import (
	"context"

	keycloakClient "github.com/platform-mesh/iam-service/pkg/keycloak/client"
)

// KeycloakClientInterface defines the subset of Keycloak client methods we use
type KeycloakClientInterface interface {
	GetUsersWithResponse(ctx context.Context, realm string, params *keycloakClient.GetUsersParams, reqEditors ...keycloakClient.RequestEditorFn) (*keycloakClient.GetUsersResponse, error)
}

// Ensure the generated client implements our interface
var _ KeycloakClientInterface = (*keycloakClient.ClientWithResponses)(nil)
