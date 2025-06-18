package contract_tests

import (
	"context"
	"net"
	"net/http"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/openmfp/iam-service/contract-tests/fga_test_data"

	"github.com/go-chi/chi/v5"
	"github.com/go-jose/go-jose/v4"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openfga/openfga/pkg/server"
	commonsCtx "github.com/openmfp/golang-commons/context"
	commonsLogger "github.com/openmfp/golang-commons/logger"
	"github.com/steinfletcher/apitest"
	"github.com/stretchr/testify/suite"
	"github.com/vrischmann/envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"

	"github.com/openmfp/iam-service/internal/pkg/config"
	gormlogger "github.com/openmfp/iam-service/internal/pkg/logger"
	iamRouter "github.com/openmfp/iam-service/internal/pkg/router"
	"github.com/openmfp/iam-service/pkg/db"
	dbMocks "github.com/openmfp/iam-service/pkg/db/mocks"
	"github.com/openmfp/iam-service/pkg/fga"
	openmfpservice "github.com/openmfp/iam-service/pkg/service"
)

type CommonTestSuite struct {
	suite.Suite
	appConfig     config.Config
	logger        *commonsLogger.Logger
	database      *db.Database
	conn          *grpc.ClientConn
	openfgaServer *server.Server
}

// closes database connections between tests - reclaims memory, so each test starts with fresh data
func (s *CommonTestSuite) TearDownTest() {

	err := s.conn.Close()
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to close connection")
		s.T().Fatal(err)
	}

	err = s.database.Close()
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to close database")
		s.T().Fatal(err)
	}

	s.conn = nil
	s.database = nil
}

type Middleware = func(http.Handler) http.Handler

func (s *CommonTestSuite) GqlApiTest(
	middleware *Middleware, userHooksMock *dbMocks.UserHooks, mockFgaEvents fga.FgaEvents,
) *apitest.Request {

	s.SetupLogger()

	// prevent calling this function twice in a test, before TearDownTest() is called
	if (s.conn != nil) || (s.database != nil) {
		s.logger.Error().Msg("conn and database should be nil")
		return nil
	}

	ctx := context.Background()

	// The order matters. Do not change it.
	s.setupConfig()
	s.setupOpenfgaServer(ctx)
	s.setupGrpcServer()
	s.setupDB(userHooksMock)

	r := s.getRouter(mockFgaEvents)

	result := apitest.New().
		Handler(r).
		Post("/query")

	return result
}

func (s *CommonTestSuite) setupConfig() {
	appConfig := config.Config{}
	err := envconfig.InitWithOptions(&appConfig, envconfig.Options{AllOptional: true})
	if err != nil {
		s.T().Fatal(err)
	}

	appConfig.Database.InMemory = true
	appConfig.Openfga.GRPCAddr = "localhost:8080"
	appConfig.IsLocal = true
	appConfig.Database.LocalData.DataPathUser = "../input/user.yaml"
	appConfig.Database.LocalData.DataPathInvitations = "../input/invitations.yaml"
	appConfig.Database.LocalData.DataPathTeam = "../input/team.yaml"
	appConfig.Database.LocalData.DataPathTenantConfiguration = "../input/tenantConfigurations.yaml"
	appConfig.Database.LocalData.DataPathDomainConfiguration = "../input/domainConfigurations.yaml"
	appConfig.Database.LocalData.DataPathRoles = "../input/roles.yaml"

	s.appConfig = appConfig
}

func (s *CommonTestSuite) SetupLogger() {
	logConfig := commonsLogger.DefaultConfig()
	logConfig.Level = "error"
	logger, err := commonsLogger.New(logConfig)
	if err != nil {
		s.Error(err)
	}
	s.logger = logger
}

func (s *CommonTestSuite) setupOpenfgaServer(ctx context.Context) {
	rawSchema, err := os.ReadFile(pathToSchemaFile)
	if err != nil {
		s.T().Fatal(err)
	}

	rawTenantData, err := os.ReadFile(pathToTenantDataFile)
	if err != nil {
		s.T().Fatal(err)
	}

	rawUserData, err := os.ReadFile(pathToUserTestDataFile)
	if err != nil {
		s.T().Fatal(err)
	}

	openfgaServer, err := fga_test_data.GetOpenfgaServer(
		ctx, tenantId, fga_test_data.FgaData{Schema: rawSchema, TenantRelations: rawTenantData, UserRelations: rawUserData},
	)
	if err != nil {
		s.T().Fatal(err)
	}

	s.openfgaServer = openfgaServer
}

func (s *CommonTestSuite) setupGrpcServer() {
	buffer := 101024 * 1024
	lis := bufconn.Listen(buffer)

	resolver.SetDefaultScheme("passthrough")
	grpcServer := grpc.NewServer()
	openfgav1.RegisterOpenFGAServiceServer(grpcServer, s.openfgaServer)

	go func() {
		grpcServer.Serve(lis) //nolint:all
	}()

	conn, err := grpc.NewClient("",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		s.T().Fatal(err)
	}

	s.conn = conn
}

func (s *CommonTestSuite) setupDB(hooks db.UserHooks) {
	// local sqlite db
	dsn := "file::memory:?cache=shared"
	dbDialect := sqlite.Open(dsn)
	dbConn, err := gorm.Open(dbDialect, &gorm.Config{
		Logger: gormlogger.NewFromLogger(s.logger.ComponentLogger("gorm")),
	})
	if err != nil {
		s.T().Fatal(err)
	}

	database, err := db.New(s.appConfig.Database, dbConn, s.logger, true, true)
	if err != nil {
		s.T().Fatal(err)
	}
	database.SetUserHooks(hooks)

	s.database = database
}

func (s *CommonTestSuite) getRouter(fgaEventHandler fga.FgaEvents) *chi.Mux {

	compatService, err := fga.NewCompatClient(openfgav1.NewOpenFGAServiceClient(s.conn), s.database, fgaEventHandler)
	if err != nil {
		s.T().Fatal(err)
	}

	// create openmfp Resolver
	mfpSvc := openmfpservice.New(s.database, compatService)
	router := iamRouter.CreateRouter(s.appConfig, mfpSvc, s.logger)
	return router

}

func getUserInjection(token, spiffe string) Middleware { // nolint: unparam
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ctx := commonsCtx.AddWebTokenToContext(r.Context(), token, []jose.SignatureAlgorithm{jose.RS256})
			ctx = commonsCtx.AddTenantToContext(ctx, tenantId)
			if spiffe != "" {
				ctx = commonsCtx.AddSpiffeToContext(ctx, spiffe)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
