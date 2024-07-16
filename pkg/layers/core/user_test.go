package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmfp/iam-service/pkg/layers/core/models"
	"github.com/openmfp/iam-service/pkg/mocks"
)

func getUserObject() *models.User {
	return &models.User{
		TenantID:  "tenant123",
		UserID:    "user123",
		Email:     "userEmail",
		FirstName: "userFirstName",
		LastName:  "userLastName",
	}
}

func TestService_CreateUser(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		mockSetup func(
			dbMock *mocks.DB,
			hooksMock *mocks.Hooks,
			verifierMock *mocks.Verifier,
		)
		error error
	}{
		{
			name:   "create_user_OK",
			userID: getUserObject().UserID,
			mockSetup: func(
				dbMock *mocks.DB,
				hooksMock *mocks.Hooks,
				verifierMock *mocks.Verifier,
			) {
				verifierMock.EXPECT().
					VerifyCreateUserInput(context.Background(), *getUserObject()).
					Return(nil).
					Once()

				dbMock.EXPECT().
					CreateUser(context.Background(), *getUserObject()).
					Return(getUserObject(), nil).
					Once()
				hooksMock.EXPECT().
					UserCreated(context.Background(), *getUserObject()).
					Return(nil).
					Once()
			},
		},
		{
			name:   "failed_to_validate_email_ERROR",
			userID: getUserObject().UserID,
			mockSetup: func(
				dbMock *mocks.DB,
				hooksMock *mocks.Hooks,
				verifierMock *mocks.Verifier,
			) {
				verifierMock.EXPECT().
					VerifyCreateUserInput(context.Background(), *getUserObject()).
					Return(assert.AnError).
					Once()
			},
			error: assert.AnError,
		},
		{
			name:   "email_is_not_valid_and_user_id_is_not_provided_ERROR",
			userID: "", // this is for purpose
			mockSetup: func(
				dbMock *mocks.DB,
				hooksMock *mocks.Hooks,
				verifierMock *mocks.Verifier,
			) {
				verifierMock.EXPECT().
					VerifyCreateUserInput(context.Background(),
						models.User{
							TenantID:  getUserObject().TenantID,
							UserID:    "",
							Email:     getUserObject().Email,
							FirstName: getUserObject().FirstName,
							LastName:  getUserObject().LastName,
						}).
					Return(assert.AnError).
					Once()
			},
			error: assert.AnError,
		},
		{
			name:   "create_user_db_ERROR",
			userID: getUserObject().UserID,
			mockSetup: func(
				dbMock *mocks.DB,
				hooksMock *mocks.Hooks,
				verifierMock *mocks.Verifier,
			) {
				verifierMock.EXPECT().
					VerifyCreateUserInput(context.Background(), *getUserObject()).
					Return(nil).
					Once()

				dbMock.EXPECT().
					CreateUser(context.Background(), *getUserObject()).
					Return(nil, errors.New("db error")).
					Once()
			},
			error: errors.New("db error"),
		},
		{
			name:   "hooks_user_created_ERROR",
			userID: getUserObject().UserID,
			mockSetup: func(
				dbMock *mocks.DB,
				hooksMock *mocks.Hooks,
				verifierMock *mocks.Verifier,
			) {
				verifierMock.EXPECT().
					VerifyCreateUserInput(context.Background(), *getUserObject()).
					Return(nil).
					Once()

				dbMock.EXPECT().
					CreateUser(context.Background(), *getUserObject()).
					Return(getUserObject(), nil).
					Once()
				hooksMock.EXPECT().
					UserCreated(context.Background(), *getUserObject()).
					Return(assert.AnError).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbMock := &mocks.DB{}
			hooksMock := &mocks.Hooks{}
			verifierMock := &mocks.Verifier{}
			s, err := New(dbMock, hooksMock, verifierMock)
			assert.Nil(t, err)

			tt.mockSetup(dbMock, hooksMock, verifierMock)

			_, err = s.CreateUser(context.Background(), models.User{
				TenantID:  getUserObject().TenantID,
				UserID:    tt.userID,
				Email:     getUserObject().Email,
				FirstName: getUserObject().FirstName,
				LastName:  getUserObject().LastName,
			})

			assert.Equal(t, tt.error, err)

			dbMock.AssertExpectations(t)
			hooksMock.AssertExpectations(t)
		})
	}
}
