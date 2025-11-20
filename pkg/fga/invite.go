package fga

import (
	"context"
	"fmt"
	"strings"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/platform-mesh/golang-commons/errors"
	"github.com/platform-mesh/golang-commons/fga/util"
	"github.com/platform-mesh/golang-commons/logger"
	securityv1alpha1 "github.com/platform-mesh/security-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/roles"
)

// checkAndInviteUser checks if a user exists in Keycloak and creates an Invite if not
func (s *Service) checkAndInviteUser(ctx context.Context, userEmail, accountPath string) error {
	log := logger.LoadLoggerFromContext(ctx).MustChildLoggerWithAttributes("email", sanitizeUserID(userEmail))

	// Check if user exists in Keycloak
	_, err := s.kcChecker.UserByMail(ctx, userEmail)
	if err == nil {
		// User exists, no invite needed
		return nil
	}

	log.Debug().Msg("User not found in Keycloak, will create Invite")

	// User doesn't exist, need to create an Invite in the account workspace
	// Get workspace client for the account path
	wsClient, err := s.wsClientFactory.New(accountPath)
	if err != nil {
		return errors.Wrap(err, "failed to create workspace client for path %s", accountPath)
	}

	if err := s.createInviteIfNotExists(ctx, wsClient, userEmail); err != nil {
		return errors.Wrap(err, "failed to create Invite for user %s", sanitizeUserID(userEmail))
	}

	return nil
}

// createInviteIfNotExists creates or updates an Invite resource for the user
func (s *Service) createInviteIfNotExists(ctx context.Context, wsClient client.Client, userEmail string) error {
	// Generate invite name from email (replace @ and . with -)
	inviteName := strings.ReplaceAll(strings.ReplaceAll(userEmail, "@", "-"), ".", "-")
	log := logger.LoadLoggerFromContext(ctx).MustChildLoggerWithAttributes("email", sanitizeUserID(userEmail), "inviteName", inviteName)

	invite := &securityv1alpha1.Invite{
		ObjectMeta: metav1.ObjectMeta{
			Name: inviteName,
		},
	}

	result, err := controllerutil.CreateOrUpdate(ctx, wsClient, invite, func() error {
		// Set the email in the spec
		invite.Spec.Email = userEmail
		return nil
	})

	if err != nil {
		return errors.Wrap(err, "failed to create or update Invite resource for %s", sanitizeUserID(userEmail))
	}

	if result == controllerutil.OperationResultCreated {
		log.Info().Msg("Successfully created Invite resource")
	} else {
		log.Debug().Msg("Invite already exists")
	}

	return nil
}

// processInvites processes invite requests: checks if users exist, creates Invite resources if not, and assigns roles
func (s *Service) processInvites(ctx context.Context, rctx graph.ResourceContext, invites []*graph.InviteInput, storeID, fgaTypeName, clusterId string, log *logger.Logger) (int, []string) {
	var inviteErrors []string
	var assignedCount int

	for _, invite := range invites {
		inviteLog := log.MustChildLoggerWithAttributes("email", sanitizeUserID(invite.Email))
		inviteLog.Debug().Interface("roles", invite.Roles).Msg("Processing invite")

		// Check if user exists in Keycloak and create Invite if not
		if err := s.checkAndInviteUser(ctx, invite.Email, rctx.AccountPath); err != nil {
			errMsg := fmt.Sprintf("failed to create invite for user '%s': %v", sanitizeUserID(invite.Email), err)
			inviteErrors = append(inviteErrors, errMsg)
			inviteLog.Warn().Err(err).Msg("Failed to create Invite for user, continuing with role assignment")
		}

		// Assign roles to the invited user (using email as userID)
		// Validate that only available roles are being assigned
		roleDefinitions, err := s.rolesRetriever.GetRoleDefinitions(rctx)
		if err != nil { // coverage-ignore: difficult to test without mocking - requires nil config
			errMsg := fmt.Sprintf("failed to get role definitions for group resource '%s/%s': %v", rctx.Group, rctx.Kind, err)
			inviteErrors = append(inviteErrors, errMsg)
			log.Error().Err(err).Msg("Failed to retrieve role definitions")
			continue
		}
		availableRoles := roles.GetAvailableRoleIDs(roleDefinitions)

		for _, role := range invite.Roles {
			roleLog := inviteLog.MustChildLoggerWithAttributes("role", role)
			if !containsString(availableRoles, role) {
				errMsg := fmt.Sprintf("role '%s' is not allowed for user '%s'. Only roles %v are permitted", role, sanitizeUserID(invite.Email), availableRoles)
				inviteErrors = append(inviteErrors, errMsg)
				roleLog.Warn().Interface("availableRoles", availableRoles).Msg("Invalid role assignment attempted")
				continue
			}

			// Create the role tuple and assign role tuple for this user-role combination
			count, errs := s.assignRoleToUser(ctx, invite.Email, role, rctx, storeID, fgaTypeName, clusterId, roleLog)
			assignedCount += count
			inviteErrors = append(inviteErrors, errs...)
		}
	}

	return assignedCount, inviteErrors
}

// assignRoleToUser assigns a single role to a user by creating both the role assignment tuple and the permission tuple
func (s *Service) assignRoleToUser(ctx context.Context, userEmail, role string, rctx graph.ResourceContext, storeID, fgaTypeName, clusterId string, log *logger.Logger) (int, []string) {
	var errors []string
	var assignedCount int

	// Create the role assignment tuple (user -> role)
	roleTuple := &openfgav1.TupleKey{
		User:     fmt.Sprintf("user:%s", userEmail),
		Relation: "assignee",
		Object: fmt.Sprintf("role:%s/%s/%s/%s",
			fgaTypeName,
			clusterId,
			rctx.Resource.Name,
			role),
	}

	// Create the permission tuple (role -> resource)
	targetFGATypeName := util.ConvertToTypeName(rctx.Group, rctx.Kind)
	targetObject := fmt.Sprintf("%s:%s/%s", targetFGATypeName, clusterId, rctx.Resource.Name)
	if rctx.Resource.Namespace != nil {
		targetObject = fmt.Sprintf("%s:%s/%s/%s", targetFGATypeName, clusterId, *rctx.Resource.Namespace, rctx.Resource.Name)
	}
	assignRoleTuple := &openfgav1.TupleKey{
		User: fmt.Sprintf("role:%s/%s/%s/%s#assignee",
			fgaTypeName,
			clusterId,
			rctx.Resource.Name,
			role),
		Relation: role,
		Object:   targetObject,
	}

	// Write both tuples
	for _, write := range []*openfgav1.TupleKey{roleTuple, assignRoleTuple} {
		writeReq := &openfgav1.WriteRequest{
			StoreId: storeID,
			Writes: &openfgav1.WriteRequestWrites{
				TupleKeys: []*openfgav1.TupleKey{write},
			},
		}

		_, err := s.client.Write(ctx, writeReq)
		if err != nil {
			if isDuplicateWriteError(err) {
				log.Info().Msg("Role already assigned to user - skipping duplicate")
			} else { // coverage-ignore
				errMsg := fmt.Sprintf("failed to assign role '%s' to user '%s': %v", role, sanitizeUserID(userEmail), err)
				errors = append(errors, errMsg)
				log.Error().Err(err).Msg("Failed to write tuple to FGA")
			}
		} else {
			assignedCount++
			log.Info().Msg("Successfully assigned role to user")
		}
	}

	return assignedCount, errors
}
