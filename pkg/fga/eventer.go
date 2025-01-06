package fga

import (
	"context"
	"fmt"
	"strings"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	openmfpFga "github.com/openmfp/golang-commons/fga/store"
	commonsLogger "github.com/openmfp/golang-commons/logger"
	"github.com/openmfp/iam-service/pkg/db"
	"go.opentelemetry.io/otel"
)

type InviteManger interface {
	GetInvitesForEmail(ctx context.Context, tenantID, email string) ([]db.Invite, error)
	DeleteInvitesForEmail(ctx context.Context, tenantID, email string) error
}

type FGALoginHandler interface {
	HandleLogin(ctx context.Context, logger *commonsLogger.Logger, tenantID string, userId string, userEmail string) error
}

type FGAEventer struct {
	FGALoginHandler
	upstream     openfgav1.OpenFGAServiceClient
	helper       openmfpFga.FGAStoreHelper
	inviteManger InviteManger
}

type FGAEventerOption func(f *FGAEventer)

func WithOpenFGAClient(cl openfgav1.OpenFGAServiceClient) FGAEventerOption {
	return func(f *FGAEventer) {
		f.upstream = cl
	}
}

func NewFGAEventer(
	client openfgav1.OpenFGAServiceClient,
	inviteManager InviteManger,
	helper openmfpFga.FGAStoreHelper,
	opts ...FGAEventerOption,
) (*FGAEventer, error) {
	fgaEventer := &FGAEventer{
		upstream:     client,
		inviteManger: inviteManager,
		helper:       helper,
	}

	for _, opt := range opts {
		opt(fgaEventer)
	}

	return fgaEventer, nil
}

// HandleLogin Handles the login event whenever a user logs into the portal. This makes sure that the user gets the appropriate tenant role
func (s *FGAEventer) HandleLogin(ctx context.Context, logger *commonsLogger.Logger,
	tenantID string, userId string, userEmail string) error {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "fga.handleLogin")
	defer span.End()

	storeID, err := s.helper.GetStoreIDForTenant(ctx, s.upstream, tenantID)
	if err != nil {
		logger.Error().Err(err).Msg("could not retrieve matching store for tenant")
		return err
	}

	modelID, err := s.helper.GetModelIDForTenant(ctx, s.upstream, tenantID)
	if err != nil {
		logger.Error().Err(err).Msg("could not retrieve matching store authorization model for tenant")
		return err
	}

	tenantRole := "external_viewer"
	if strings.HasPrefix(strings.ToLower(userId), "i") || strings.HasPrefix(strings.ToLower(userId), "d") {
		tenantRole = "viewer"
	}

	_, err = s.upstream.Write(ctx, &openfgav1.WriteRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		Writes: &openfgav1.WriteRequestWrites{
			TupleKeys: []*openfgav1.TupleKey{
				{
					Object:   fmt.Sprintf("role:tenant/%s/%s", tenantID, tenantRole),
					Relation: "assignee",
					User:     fmt.Sprintf("user:%s", userId),
				},
			},
		},
	})
	if err != nil && !s.helper.IsDuplicateWriteError(err) {
		logger.Debug().AnErr("openFGA write error", err).Send()
	}

	invites, err := s.inviteManger.GetInvitesForEmail(ctx, tenantID, userEmail)
	logger.Info().Str("tenantID", tenantID).Str("userEmail", userEmail).Int("invites to process", len(invites)).Msg("invites to process")

	if err != nil {
		logger.Error().Err(err).Str("userEmail", userEmail).Str("tenantID", tenantID).Msg("unable to load invites for user")
		return err
	}

	err = s.writeInvites(ctx, logger, storeID, modelID, userEmail, userId, invites)
	if err != nil {
		logger.Error().Err(err).Str("userEmail", userEmail).Msg("unable to write invites")
		return err
	}

	return s.inviteManger.DeleteInvitesForEmail(ctx, tenantID, userEmail)
}

func (s *FGAEventer) writeInvites(ctx context.Context, logger *commonsLogger.Logger,
	storeID string, modelID string, userEmail string, userId string, invites []db.Invite) error {
	for _, invite := range invites {
		var tupleKeys []*openfgav1.TupleKey
		for _, role := range strings.Split(invite.Roles, ",") {
			tupleKeys = append(tupleKeys, &openfgav1.TupleKey{
				Object:   fmt.Sprintf("role:%s/%s/%s", invite.EntityType, invite.EntityID, role),
				Relation: "assignee",
				User:     fmt.Sprintf("user:%s", userId),
			})

			// make sure role is assigned
			_, err := s.upstream.Write(ctx, &openfgav1.WriteRequest{
				StoreId:              storeID,
				AuthorizationModelId: modelID,
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{
						{
							Object:   fmt.Sprintf("%s:%s", invite.EntityType, invite.EntityID),
							Relation: role,
							User:     fmt.Sprintf("role:%s/%s/%s#assignee", invite.EntityType, invite.EntityID, role),
						},
					},
				},
			})
			logger.Info().Str("inviteEmail", invite.Email).Str("inviteEntityID", invite.EntityID).Msg("openFGA write user role for invited user")
			if err != nil && !s.helper.IsDuplicateWriteError(err) {
				logger.Debug().Str("userEmail", userEmail).AnErr("openFGA write error", err).Send()
				return err
			}
		}

		// assign user to roles
		_, err := s.upstream.Write(ctx, &openfgav1.WriteRequest{
			StoreId:              storeID,
			AuthorizationModelId: modelID,
			Writes: &openfgav1.WriteRequestWrites{
				TupleKeys: tupleKeys,
			},
		})
		logger.Info().Str("userEmail", userEmail).Msg("openFGA write user role")
		if err != nil && !s.helper.IsDuplicateWriteError(err) {
			logger.Debug().Str("userEmail", userEmail).AnErr("openFGA write error", err).Send()
			return err
		}
	}
	return nil
}
