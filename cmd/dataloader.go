package cmd

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	_ "github.com/joho/godotenv/autoload"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openfga/language/pkg/go/transformer"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	fgastore "github.com/platform-mesh/golang-commons/fga/store"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/platform-mesh/iam-service/internal/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/db"
)

// SetDataLoadCmd assigns cobra.Command to the DataLoader.dataLoadCmd field.
// I took it out of the constructor to increase readability.
func (d *DataLoader) SetDataLoadCmd() {
	var err error
	d.dataLoadCmd = &cobra.Command{
		Use:   "dataload",
		Short: "Load Initial Data",
		Long:  `Loads the initial data into the Database`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if d.killIstio {
				defer executeKillIstio()
			}

			err = d.importSchemaToFga(ctx)
			if err != nil {
				d.logger.Panic().Err(err).Msg("failed to import fga-schema")
			}

			err = d.loadDataToFga(ctx)
			if err != nil {
				d.logger.Panic().Err(err).Msg("failed to import fga-data")
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
	fgaClient   openfgav1.OpenFGAServiceClient
	fgaStore    fgastore.FGAStoreHelper
	Database    db.DataLoader
	dataLoadCmd *cobra.Command
	file        string
	schemaFile  string
	tenants     string
	killIstio   bool
}

type Data struct {
	Tuples []Tuple `yaml:"tuples"`
}

type Tuple struct {
	Object   string `yaml:"object"`
	Relation string `yaml:"relation"`
	User     string `yaml:"user"`
}

// InitDataLoader is an outer wrapper of the DataLoader constructor that is used in `command.go` init(). Not testable.
func InitDataLoader(rootCmd *cobra.Command) {
	cfg, logger := initApp() // nolint: typecheck
	conn, err := grpc.NewClient(cfg.Openfga.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Panic().Err(err).Msg("failed to create fga client")
	}

	database, err := initDB(cfg, logger)
	if err != nil {
		logger.Panic().Err(err).Msg("failed to init a database")
	}

	NewDataLoader(rootCmd, cfg, logger, openfgav1.NewOpenFGAServiceClient(conn), fgastore.New(), database)
}

// NewDataLoader is an inner wrapper of the DataLoader constructor which accepts all dependencies as arguments. Testable.
func NewDataLoader(
	rootCmd *cobra.Command,
	cfg config.Config,
	logger *logger.Logger,
	fgaClient openfgav1.OpenFGAServiceClient,
	fgaStore fgastore.FGAStoreHelper,
	database db.DataLoader,
) {
	d := &DataLoader{
		cfg:       cfg,
		logger:    logger,
		fgaClient: fgaClient,
		fgaStore:  fgaStore,
		Database:  database,
		killIstio: false,
	}

	d.SetDataLoadCmd()

	rootCmd.AddCommand(d.dataLoadCmd)

	d.dataLoadCmd.Flags().StringVar(&d.file, "file", "", "file to import")
	d.dataLoadCmd.Flags().StringVar(&d.schemaFile, "schema", "", "schema to import")
	d.dataLoadCmd.Flags().StringVarP(&d.tenants, "tenants", "t", "", "tenant to import in")
	d.dataLoadCmd.Flags().BoolVar(&d.killIstio, "kill-istio", false, "indicates if the cli should kill the istio proxy after execution")

	var err error
	requiredFlags := []string{"file", "schema", "tenants"}
	for _, flag := range requiredFlags {
		err = d.dataLoadCmd.MarkFlagRequired(flag)
		if err != nil {
			d.logger.Panic().Err(err).Msg("failed to mark flag as required")
		}
	}
}

// importSchemaToFga imports schema.fga to the FGA server.
func (d *DataLoader) importSchemaToFga(ctx context.Context) error {
	bytes, err := os.ReadFile(d.schemaFile)
	if err != nil {
		d.logger.Error().Err(err).Msg("failed to read schema file")
		return err
	}

	model := transformer.MustTransformDSLToProto(string(bytes))

	var storeID string
	var res *openfgav1.WriteAuthorizationModelResponse
	for _, tenant := range strings.Split(d.tenants, ",") {
		storeID, err = d.fgaStore.GetStoreIDForTenant(ctx, d.fgaClient, tenant)
		if err != nil {
			d.logger.Error().Err(err).Msg("failed to get storeID")
			return err
		}

		res, err = d.fgaClient.WriteAuthorizationModel(ctx, &openfgav1.WriteAuthorizationModelRequest{
			StoreId:         storeID,
			TypeDefinitions: model.TypeDefinitions,
			SchemaVersion:   model.SchemaVersion,
		})
		if err != nil {
			d.logger.Error().Err(err).Msg("failed to import authorization model")
			return err
		}

		d.logger.Info().Msgf("stored new model version for tenant: %s, storeid: %s, modelid: %s", tenant, storeID, res.AuthorizationModelId)
	}

	return nil
}

// loadDataToFga loads data to the FGA server.
func (d *DataLoader) loadDataToFga(ctx context.Context) error {
	bytes, err := os.ReadFile(d.file)
	if err != nil {
		d.logger.Error().Err(err).Msg("failed to read data file")
		return err
	}

	data := &Data{}
	err = yaml.Unmarshal(bytes, data)
	if err != nil {
		d.logger.Error().Err(err).Msg("failed to parse data file")
		return err
	}

	var storeID string

	for _, tenant := range strings.Split(d.tenants, ",") {
		storeID, err = d.fgaStore.GetStoreIDForTenant(ctx, d.fgaClient, tenant)
		if err != nil {
			d.logger.Error().Err(err).Msg("failed to get storeID for tenant")
			return err
		}
		for _, tuple := range data.Tuples {
			err = d.processSingleTuple(ctx, storeID, tuple)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *DataLoader) processSingleTuple(ctx context.Context, storeID string, tuple Tuple) error {
	res, err := d.fgaClient.Read(ctx, &openfgav1.ReadRequest{
		StoreId: storeID,
		TupleKey: &openfgav1.ReadRequestTupleKey{
			Object:   tuple.Object,
			Relation: tuple.Relation,
			User:     tuple.User,
		},
	})
	if err != nil {
		d.logger.Error().Err(err).Msg("failed to read from fga")
		return err
	}

	if len(res.Tuples) > 0 {
		d.logger.Info().Msgf("skipped tuple %s#%s@%s", tuple.Object, tuple.Relation, tuple.User)
		return nil
	}

	_, err = d.fgaClient.Write(ctx, &openfgav1.WriteRequest{
		StoreId: storeID,
		Writes: &openfgav1.WriteRequestWrites{
			TupleKeys: []*openfgav1.TupleKey{
				{
					Object:   tuple.Object,
					Relation: tuple.Relation,
					User:     tuple.User,
				},
			},
		},
	})
	if err != nil {
		d.logger.Error().Err(err).Msg("failed to write to fga")
		return err
	}

	d.logger.Info().Msgf("wrote tuple %s#%s@%s", tuple.Object, tuple.Relation, tuple.User)

	return nil
}

// loadDataToDB loads data to the database.
func (d *DataLoader) loadDataToDB() error {
	err := d.Database.LoadTenantConfigData(d.cfg.Database.LocalData.DataPathTenantConfiguration)
	if err != nil {
		log.Error().Err(err).Msg("failed to load tenant config data")
		return err
	}

	if d.cfg.Database.LocalData.DataPathRoles != "" {
		err = d.Database.LoadRoleData(d.cfg.Database.LocalData.DataPathRoles)
		if err != nil {
			log.Error().Err(err).Msg("failed to load data path roles")
			return err
		}
	}

	return nil
}

func executeKillIstio() {
	res, err := http.Post("http://localhost:15020/quitquitquit", "application/json", http.NoBody) // nolint: noctx
	if err != nil {
		log.Panic().Err(err).Msg("failed to kill istio")
	}
	defer res.Body.Close() //nolint:errcheck
}
