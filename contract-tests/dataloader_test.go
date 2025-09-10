package contract_tests

import (
	"context"
	"testing"

	"github.com/platform-mesh/iam-service/pkg/db"

	"github.com/stretchr/testify/mock"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/platform-mesh/iam-service/cmd"
	"github.com/platform-mesh/iam-service/internal/pkg/config"
	fgamock "github.com/platform-mesh/iam-service/pkg/fga/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DataLoaderTestSuite struct {
	CommonTestSuite
}

func TestDataLoaderTestSuite(t *testing.T) {
	suite.Run(t, new(DataLoaderTestSuite))
}

func (suite *DataLoaderTestSuite) TestLoadData() {
	storeID := "AAAAAAAAAAAAAAAAAAAAAAAAAA"
	// Initialize the root command or any parent command
	rootCmd := &cobra.Command{}
	// Set the flags (mimicking: `go run main.go dataload --schema=./assets/schema.fga
	// --file=./assets/data.yaml --tenants=tenant1,tenant2`)
	rootCmd.SetArgs([]string{
		"dataload",
		"--schema=./testdata/schema.fga",
		"--file=./assets/data.yaml",
		"--tenants=tenant1,tenant2",
	})

	// initialize suite fields(fga, db, etc)
	suite.GqlApiTest(nil, nil)
	suite.appConfig = config.Config{
		IsLocal: false,
		Database: db.ConfigDatabase{
			LocalData: db.DatabaseLocalData{
				DataPathTenantConfiguration: "../input/tenantConfigurations.yaml",
			},
		},
	}
	fgaStoreHelperMock := &fgamock.FGAStoreHelper{}

	fgaStoreHelperMock.EXPECT().
		GetStoreIDForTenant(mock.Anything, mock.Anything, "tenant1").
		Return(storeID, nil).Once()
	fgaStoreHelperMock.EXPECT().
		GetStoreIDForTenant(mock.Anything, mock.Anything, "tenant1").
		Return(storeID, nil).Once()
	fgaStoreHelperMock.EXPECT().
		GetStoreIDForTenant(mock.Anything, mock.Anything, "tenant2").
		Return(storeID, nil).Once()
	fgaStoreHelperMock.EXPECT().
		GetStoreIDForTenant(mock.Anything, mock.Anything, "tenant2").
		Return(storeID, nil).Once()

	fgaClient := openfgav1.NewOpenFGAServiceClient(suite.conn)
	cmd.NewDataLoader(
		rootCmd, suite.appConfig, suite.logger, fgaClient, fgaStoreHelperMock, suite.database)

	// Execute the command
	err := rootCmd.Execute()

	// Assert that there were no errors during execution
	assert.NoError(suite.T(), err)

	// Check DB. We must have 1 tenant configurations in the database
	var tenantConfigs []db.TenantConfiguration
	err = suite.database.GetGormDB().Find(&tenantConfigs).Error
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 1, len(tenantConfigs))
	assert.Equal(suite.T(), "example-tenant", tenantConfigs[0].TenantID)

	// Check FGA Schema.
	authorizationModels, err := fgaClient.ReadAuthorizationModels(context.TODO(), &openfgav1.ReadAuthorizationModelsRequest{
		StoreId: storeID,
	})
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 2, len(authorizationModels.AuthorizationModels),
		"must have 2 authorization models in the database, one model for each tenant")
	assert.Equal(suite.T(), 5, len(authorizationModels.AuthorizationModels[0].TypeDefinitions),
		"first auth model must have 5 type definitions")
	assert.Equal(suite.T(), 5, len(authorizationModels.AuthorizationModels[1].TypeDefinitions),
		"second auth model must have 5 type definitions")

	// Check FGA Data.
	readResponse, err := fgaClient.Read(context.TODO(), &openfgav1.ReadRequest{
		StoreId: storeID,
	})
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 4, len(readResponse.Tuples), "must have 4 tuples in the fga store")
	assert.Equal(suite.T(), readResponse.Tuples[0].Key, &openfgav1.TupleKey{
		User:     "role:tenant/tenant1/viewer#assignee",
		Relation: "member",
		Object:   "core_platform-mesh_io_account:tenant1",
	})
	assert.Equal(suite.T(), readResponse.Tuples[1].Key, &openfgav1.TupleKey{
		User:     "role:tenant/tenant1/external_viewer#assignee",
		Relation: "external_member",
		Object:   "core_platform-mesh_io_account:tenant1",
	})
	assert.Equal(suite.T(), readResponse.Tuples[2].Key, &openfgav1.TupleKey{
		User:     "role:tenant/tenant2/viewer#assignee",
		Relation: "member",
		Object:   "core_platform-mesh_io_account:tenant2",
	})
	assert.Equal(suite.T(), readResponse.Tuples[3].Key, &openfgav1.TupleKey{
		User:     "role:tenant/tenant2/external_viewer#assignee",
		Relation: "external_member",
		Object:   "core_platform-mesh_io_account:tenant2",
	})
}
