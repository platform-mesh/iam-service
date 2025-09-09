package cmd

import (
	"net/http"

	"github.com/rs/zerolog/log"

	_ "github.com/joho/godotenv/autoload"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/spf13/cobra"

	"github.com/platform-mesh/iam-service/internal/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/db"
)

// SetDataLoadCmd assigns cobra.Command to the DataLoader.dataLoadCmd field.
// I took it out of the constructor to increase readability.
func (d *DataLoader) SetDataLoadCmd(cfg config.Config) {
	var err error
	d.dataLoadCmd = &cobra.Command{
		Use:   "dataload",
		Short: "Load Initial Data",
		Long:  `Loads the initial data into the Database`,
		RunE: func(cmd *cobra.Command, args []string) error {

			if d.killIstio {
				defer executeKillIstio(cfg)
			}

			err = d.loadDataToDB()
			if err != nil {
				d.logger.Panic().Err(err).Msg("failed to seed db with data")
			}

			return nil
		},
	}
}

type DataLoader struct {
	cfg         config.Config
	logger      *logger.Logger
	Database    db.DataLoader
	dataLoadCmd *cobra.Command
	killIstio   bool
}

// InitDataLoader is an outer wrapper of the DataLoader constructor that is used in `command.go` init(). Not testable.
func InitDataLoader(rootCmd *cobra.Command) {
	cfg, log := initApp() // nolint: typecheck
	NewDataLoader(rootCmd, cfg, log, nil)
}

// NewDataLoader is an inner wrapper of the DataLoader constructor which accepts all dependencies as arguments. Testable.
func NewDataLoader(
	rootCmd *cobra.Command,
	cfg config.Config,
	logger *logger.Logger,
	database db.DataLoader,
) {
	d := &DataLoader{
		cfg:       cfg,
		logger:    logger,
		Database:  database,
		killIstio: false,
	}

	d.SetDataLoadCmd(cfg)

	rootCmd.AddCommand(d.dataLoadCmd)
	d.dataLoadCmd.Flags().BoolVar(&d.killIstio, "kill-istio", false, "indicates if the cli should kill the istio proxy after execution")
}

// loadDataToDB loads data to the database.
func (d *DataLoader) loadDataToDB() error {
	if d.Database == nil {
		var err error
		d.Database, err = initDB(d.cfg, d.logger)
		if err != nil {
			log.Panic().Err(err).Msg("failed to init a database")
		}
	}

	if d.cfg.Database.LocalData.DataPathRoles != "" {
		err := d.Database.LoadRoleData(d.cfg.Database.LocalData.DataPathRoles)
		if err != nil {
			log.Error().Err(err).Msg("failed to load data path roles")
			return err
		}
	}
	return nil
}

func executeKillIstio(cfg config.Config) {
	res, err := http.Post(cfg.Istio.QuitApi, "application/json", http.NoBody)
	if err != nil {
		log.Panic().Err(err).Msg("failed to kill istio")
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Error().Err(err).Msg("failed to close body")
		}
	}()
}
