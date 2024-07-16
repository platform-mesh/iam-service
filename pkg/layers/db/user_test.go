package db

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/openmfp/iam-service/pkg/layers/core/models"
)

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (suite *ServiceTestSuite) TestGetUserByIDOrEmail() {
	ctx := context.Background()

	// Define the input and expected result
	input := models.User{
		UserID:   "userID123",
		TenantID: "tenant123",
		Email:    "user@example.com",
	}

	expectedSQL := "SELECT * FROM `users` WHERE tenant_id = ? AND (user_id = ? OR email = ?) ORDER BY `users`.`id` LIMIT ?"

	// Define the expected query and result
	rows := sqlmock.NewRows([]string{"id", "user_id", "tenant_id", "email", "first_name", "last_name"}).
		AddRow("1", "userID123", "tenant123", "user@example.com", "John", "Doe")
	suite.mock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs("tenant123", "userID123", "user@example.com", 1).
		WillReturnRows(rows)

	// Execute the function
	user, err := suite.svc.GetUserByIDOrEmail(ctx, input)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), user)

	// Ensure all expectations were met
	err = suite.mock.ExpectationsWereMet()
	assert.NoError(suite.T(), err)
}

func (suite *ServiceTestSuite) TestCreateUser() {
	tests := []struct {
		name      string
		input     models.User
		setupMock func()
		error     error
	}{
		{
			name: "CreateUser_OK",
			input: models.User{
				UserID:    "userID123",
				TenantID:  "tenant123",
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
			},
			setupMock: func() {
				expectedSelectSQL := "SELECT * FROM `users` WHERE tenant_id = ? AND (user_id = ? OR email = ?) ORDER BY `users`.`id` LIMIT ?"
				expectedInsertSQL := "INSERT INTO `users` (`id`,`tenant_id`,`user_id`,`email`,`first_name`,`last_name`) VALUES (?,?,?,?,?,?)"

				// Define the expected query and result for checking if user already exists
				existingRows := sqlmock.NewRows([]string{"id", "user_id", "tenant_id", "email", "first_name", "last_name"})
				suite.mock.ExpectQuery(regexp.QuoteMeta(expectedSelectSQL)).
					WithArgs("tenant123", "userID123", "user@example.com", 1).
					WillReturnRows(existingRows)

				// Define the expected query and result for creating a new user
				suite.mock.ExpectBegin()
				suite.mock.ExpectExec(regexp.QuoteMeta(expectedInsertSQL)).
					WithArgs(sqlmock.AnyArg(), "tenant123", "userID123", "user@example.com", "John", "Doe").
					WillReturnResult(sqlmock.NewResult(1, 1))
				suite.mock.ExpectCommit()
			},
		},
		{
			name: "tenantID_is_required_ERROR",
			input: models.User{
				UserID: "userID123",
				Email:  "user@example.com",
			},
			error: errors.New("tenantID is required"),
		},
		{
			name: "tenantID_is_required_ERROR",
			input: models.User{
				TenantID: "tenant123",
			},
			error: errors.New("at least one of userId and email has to be provided"),
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			if tt.setupMock != nil {
				tt.setupMock()
			}

			// Execute the function
			_, err := suite.svc.CreateUser(ctx, tt.input)
			assert.Equal(t, tt.error, err)
			//assert.NotNil(suite.T(), user)

			// Ensure all expectations were met
			err = suite.mock.ExpectationsWereMet()
			assert.NoError(suite.T(), err)
		})
	}
}
