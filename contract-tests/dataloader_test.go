package contract_tests

import (
	"testing"

	"github.com/platform-mesh/iam-service/pkg/db"

	"github.com/platform-mesh/iam-service/cmd"
	"github.com/platform-mesh/iam-service/internal/pkg/config"
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
	// Initialize the root command or any parent command
	rootCmd := &cobra.Command{}
	// Set the flags (mimicking: `go run main.go dataload --schema=./assets/schema.fga
	// --file=./assets/data.yaml --tenants=tenant1,tenant2`)
	rootCmd.SetArgs([]string{
		"dataload",
	})

	// initialize suite fields(fga, db, etc)
	suite.GqlApiTest(nil, nil, nil)

	// Set up config with proper paths for the data files
	suite.appConfig = config.Config{
		Database: db.ConfigDatabase{
			LocalData: db.DatabaseLocalData{
				DataPathTenantConfiguration: "../input/tenantConfigurations.yaml",
				DataPathRoles:               "../input/roles.yaml",
			},
		},
	}

	cmd.NewDataLoader(
		rootCmd, suite.appConfig, suite.logger, suite.database)

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

}
