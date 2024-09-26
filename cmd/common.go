package cmd

import (
	"fmt"
	"os"

	"github.com/openmfp/iam-service/pkg/db"

	"github.com/openmfp/golang-commons/logger"
	"github.com/openmfp/iam-service/internal/pkg/config"
)

func initApp() (config.Config, *logger.Logger) {
	appConfig, err := config.NewFromEnv()
	if err != nil {
		fmt.Printf("Error loading env file: %v\n", err) // nolint: forbidigo
		os.Exit(1)
	}

	logConfig := logger.DefaultConfig()
	logConfig.Name = "iam"
	logConfig.Level = appConfig.LogLevel
	logConfig.NoJSON = appConfig.IsLocal
	log, err := logger.New(logConfig)
	if err != nil {
		fmt.Printf("Error init logger: %v\n", err) // nolint: forbidigo
		os.Exit(1)
	}

	log.Info().Msgf("Logging on log level: %s", log.GetLevel().String())

	return appConfig, log
}

// initDB is a helper function to initialize the database connection.
func initDB(appConfig config.Config, log *logger.Logger) (*db.Database, error) {
	dbConn, err := getGormConn(log, appConfig.Database)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get a Database connection")
		return nil, err
	}

	database, err := db.New(appConfig.Database, dbConn, log, true, appConfig.IsLocal)
	if err != nil {
		log.Error().Err(err).Msg("Failed to init Database")
		return nil, err
	}

	return database, nil
}
