package fga

import (
	"context"
	"testing"

	"github.com/openmfp/golang-commons/fga/store/mocks"
	"github.com/openmfp/golang-commons/logger"
	"github.com/openmfp/iam-service/pkg/db"
	fgaMocks "github.com/openmfp/iam-service/pkg/fga/mocks"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewFGAEventer(t *testing.T) {
	clientMock := fgaMocks.NewOpenFGAServiceClient(t)
	inviteManagerMock := fgaMocks.NewInviteManger(t)
	helperMock := &mocks.FGAStoreHelper{}

	eventer, err := NewFGAEventer(clientMock, inviteManagerMock, helperMock, WithOpenFGAClient(clientMock))
	assert.NoError(t, err)
	assert.NotNil(t, eventer)
}

func TestHandleLogin(t *testing.T) {
	tests := []struct {
		name       string
		error      error
		setupMocks func(*fgaMocks.OpenFGAServiceClient, *fgaMocks.InviteManger, *mocks.FGAStoreHelper)
	}{
		{
			name: "should_be_able_to_handle_the_login_event_and_resolve_pending_invitations_for_a_user",
			setupMocks: func(c *fgaMocks.OpenFGAServiceClient, i *fgaMocks.InviteManger, h *mocks.FGAStoreHelper) {
				h.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("storeId", nil).Once()

				h.EXPECT().GetModelIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("modelId", nil).Once()

				// 1. grant viewer role
				c.EXPECT().Write(mock.Anything, mock.Anything).Once().Return(nil, assert.AnError).Once()
				h.EXPECT().IsDuplicateWriteError(assert.AnError).Return(false).Once()
				// 2. assign role to entity
				// 3. assign user to roles
				c.EXPECT().Write(mock.Anything, mock.Anything).Once().Return(nil, nil).Times(2)

				i.EXPECT().GetInvitesForEmail(mock.Anything, mock.Anything, mock.Anything).
					Return([]db.Invite{{
						TenantID:   "test-tenant",
						Email:      "user@sap.com",
						EntityType: "project",
						EntityID:   "test",
						Roles:      "owner",
					}}, nil).Once()

				i.EXPECT().DeleteInvitesForEmail(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:  "get_store_id_ERROR",
			error: assert.AnError,
			setupMocks: func(c *fgaMocks.OpenFGAServiceClient, i *fgaMocks.InviteManger, h *mocks.FGAStoreHelper) {
				h.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("", assert.AnError).Once()
			},
		},
		{
			name:  "get_model_id_ERROR",
			error: assert.AnError,
			setupMocks: func(c *fgaMocks.OpenFGAServiceClient, i *fgaMocks.InviteManger, h *mocks.FGAStoreHelper) {
				h.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("storeId", nil).Once()

				h.EXPECT().GetModelIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("", assert.AnError).Once()
			},
		},
		{
			name:  "get_invites_for_email_ERROR",
			error: assert.AnError,
			setupMocks: func(c *fgaMocks.OpenFGAServiceClient, i *fgaMocks.InviteManger, h *mocks.FGAStoreHelper) {
				h.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("storeId", nil).Once()

				h.EXPECT().GetModelIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("modelId", nil).Once()

				// 1. grant viewer role
				c.EXPECT().Write(mock.Anything, mock.Anything).Once().Return(nil, nil).Times(1)

				i.EXPECT().GetInvitesForEmail(mock.Anything, mock.Anything, mock.Anything).
					Return(nil, assert.AnError).Once()
			},
		},
		{
			name:  "invite_write_ERROR",
			error: assert.AnError,
			setupMocks: func(c *fgaMocks.OpenFGAServiceClient, i *fgaMocks.InviteManger, h *mocks.FGAStoreHelper) {
				h.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("storeId", nil).Once()

				h.EXPECT().GetModelIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("modelId", nil).Once()

				// 1. grant viewer role
				c.EXPECT().Write(mock.Anything, mock.Anything).Once().Return(nil, assert.AnError).Once()
				h.EXPECT().IsDuplicateWriteError(assert.AnError).Return(false).Once()

				i.EXPECT().GetInvitesForEmail(mock.Anything, mock.Anything, mock.Anything).
					Return([]db.Invite{{
						TenantID:   "test-tenant",
						Email:      "user@sap.com",
						EntityType: "project",
						EntityID:   "test",
						Roles:      "owner",
					}}, nil).Once()

				// 2. assign role to entity
				c.EXPECT().Write(mock.Anything, mock.Anything).Once().Return(nil, assert.AnError).Once()
				h.EXPECT().IsDuplicateWriteError(assert.AnError).Return(false).Once()
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			helperMock := &mocks.FGAStoreHelper{}
			inviteMock := fgaMocks.NewInviteManger(t)
			fgaClientMock := fgaMocks.NewOpenFGAServiceClient(t)

			eventer := &FGAEventer{
				upstream:     fgaClientMock,
				inviteManger: inviteMock,
				helper:       helperMock,
			}

			if test.setupMocks != nil {
				test.setupMocks(fgaClientMock, inviteMock, helperMock)
			}

			err := eventer.HandleLogin(context.Background(), logger.NewFromZerolog(zerolog.Nop()), "test-tenant", "I000000", "user@sap.com")
			assert.Equal(t, test.error, err)
		})
	}
}
