package contract_tests

import (
	"testing"

	"github.com/openmfp/iam-service/pkg/db"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/openmfp/iam-service/cmd"
	"github.com/openmfp/iam-service/internal/pkg/config"
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
	// Set the flags (mimicking: `go run main.go dataload
	rootCmd.SetArgs([]string{
		"dataload",
	})

	// initialize suite fields(fga, db, etc)
	suite.GqlApiTest(nil, nil, nil)
	suite.appConfig = config.Config{
		IsLocal:  false,
		Database: db.ConfigDatabase{}}

	suite.appConfig.Database.LocalData.DataPathRoles = "../input/roles.yaml"

	cmd.NewDataLoader(
		rootCmd, suite.appConfig, suite.logger, suite.database)

	// Execute the command
	err := rootCmd.Execute()

	// Assert that there were no errors during execution
	assert.NoError(suite.T(), err)
}
