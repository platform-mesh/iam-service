package db

import (
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ServiceTestSuite struct {
	suite.Suite
	db   *gorm.DB
	mock sqlmock.Sqlmock
	svc  *Service
}

func (suite *ServiceTestSuite) SetupTest() {
	var err error
	var sqlDB *sql.DB

	// Create a new sqlmock database connection
	sqlDB, suite.mock, err = sqlmock.New()
	assert.NoError(suite.T(), err)

	// Disable GORM's internal logger to avoid cluttering test output
	suite.db, err = gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(suite.T(), err)

	// Initialize the service with the mocked Interface
	suite.svc = &Service{conn: suite.db}
}
