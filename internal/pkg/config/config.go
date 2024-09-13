package config

import (
	"time"

	"github.com/openmfp/iam-service/pkg/db"
	"github.com/vrischmann/envconfig"
)

type Config struct {
	Database        db.ConfigDatabase
	IsLocal         bool          `envconfig:"default=false"`
	LocalSsl        bool          `envconfig:"default=false"`
	LogLevel        string        `envconfig:"default=info,optional"`
	ShutdownTimeout time.Duration `envconfig:"default=1s"`
	Port            string        `envconfig:"default=8080,optional"`
	HealthPort      string        `envconfig:"default=3389,optional"`
	MetricsPort     string        `envconfig:"default=2112,optional"`
	Openfga         struct {
		EventingEnabled bool   `envconfig:"default=false"`
		GRPCAddr        string `envconfig:"default=openfga:8081"`
		ListenAddr      string `envconfig:"default=:8081"`
	}
}

// NewFromEnv creates a Config from environment values
func NewFromEnv() (Config, error) {
	appConfig := Config{}
	err := envconfig.Init(&appConfig)
	return appConfig, err
}
