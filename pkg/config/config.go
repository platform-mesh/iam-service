package config

type ServiceConfig struct {
	Port    int `mapstructure:"port" default:"8080"`
	OpenFGA struct {
		GRPCAddr string `mapstructure:"openfga-grpc-addr" default:"openfga:8081"`
	} `mapstructure:",squash"`
	JWT struct {
		UserIDClaim string `mapstructure:"jwt-user-id-claim" default:"sub"`
	} `mapstructure:",squash"`
	IDM struct {
		ExcludedTenants []string `mapstructure:"idm-excluded-tenants"`
	} `mapstructure:",squash"`
	Keycloak struct {
		BaseURL      string `mapstructure:"keycloak-base-url" default:"https://portal.dev.local:8443/keycloak"`
		ClientID     string `mapstructure:"keycloak-client-id" default:"admin-cli"`
		User         string `mapstructure:"keycloak-user" default:"keycloak-admin"`
		PasswordFile string `mapstructure:"keycloak-password-file" default:".secret/keycloak/password"`
		Cache        struct {
			TTL     string `mapstructure:"keycloak-cache-ttl" default:"5m"`
			Enabled bool   `mapstructure:"keycloak-cache-enabled" default:"true"`
		} `mapstructure:",squash"`
	} `mapstructure:",squash"`
}
