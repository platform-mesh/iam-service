package config

import "time"

type IDMConfig struct {
	ExcludedTenants []string `mapstructure:"excluded-tenants" default:"welcome"`
}

type OpenFGAConfig struct {
	GRPCAddr      string        `mapstructure:"openfga-grpc-addr" default:"openfga:8081"`
	StoreCacheTTL time.Duration `mapstructure:"openfga-store-cache-ttl" default:"5m"`
}

type JWTConfig struct {
	UserIDClaim string `mapstructure:"jwt-user-id-claim" default:"sub"`
}

type KeycloakCacheConfig struct {
	Enabled bool          `mapstructure:"keycloak-cache-enabled" default:"true"`
	TTL     time.Duration `mapstructure:"keycloak-user-cache-ttl" default:"1h"`
}

type KeycloakConfig struct {
	BaseURL      string              `mapstructure:"keycloak-base-url" default:"https://portal.dev.local:8443/keycloak"`
	ClientID     string              `mapstructure:"keycloak-client-id" default:"admin-cli"`
	User         string              `mapstructure:"keycloak-user" default:"keycloak-admin"`
	PasswordFile string              `mapstructure:"keycloak-password-file" default:".secret/keycloak/password"`
	Cache        KeycloakCacheConfig `mapstructure:",squash"`
}

type PaginationConfig struct {
	DefaultLimit int `mapstructure:"pagination-default-limit" default:"10"`
	DefaultPage  int `mapstructure:"pagination-default-page" default:"1"`
}

type SortingConfig struct {
	DefaultField     string `mapstructure:"sorting-default-field" default:"LastName"`
	DefaultDirection string `mapstructure:"sorting-default-direction" default:"ASC"`
}

type RolesConfig struct {
	FilePath string `mapstructure:"roles-file-path" default:"input/roles.yaml"`
}

type ServiceConfig struct {
	Port       int              `mapstructure:"port" default:"8080"`
	OpenFGA    OpenFGAConfig    `mapstructure:",squash"`
	JWT        JWTConfig        `mapstructure:",squash"`
	IDM        IDMConfig        `mapstructure:",squash"`
	Keycloak   KeycloakConfig   `mapstructure:",squash"`
	Pagination PaginationConfig `mapstructure:",squash"`
	Sorting    SortingConfig    `mapstructure:",squash"`
	Roles      RolesConfig      `mapstructure:",squash"`
}
