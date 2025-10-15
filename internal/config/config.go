package config

type ServiceConfig struct {
	Port    int `mapstructure:"port" default:"8080"`
	OpenFGA struct {
		GRPCAddr string `mapstructure:"openfga-grpc-addr" default:"openfga:8081"`
	} `mapstructure:",squash"`
	JWT struct {
		UserIDClaim string `mapstructure:"jwt-user-id-claim" default:"sub"`
	} `mapstructure:",squash"`
	Keycloak struct {
		BaseURL      string `mapstructure:"keycloak-base-url" default:"https://portal.dev.local:8443/keycloak"`
		ClientID     string `mapstructure:"keycloak-client-id" default:"iam-service"`
		User         string `mapstructure:"keycloak-user" default:"keycloak-admin"`
		PasswordFile string `mapstructure:"keycloak-password-file" default:".secret/keycloak/password"`
	} `mapstructure:",squash"`
}
