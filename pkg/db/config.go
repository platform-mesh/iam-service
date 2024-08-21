package db

type ConfigDatabase struct {
	DSN               string `envconfig:"optional"`
	User              string `envconfig:"default=admin"`
	Password          string `envconfig:"default=apassword"`
	Name              string `envconfig:"default=aname"`
	IP                string `envconfig:"optional"`
	SSLMode           string `envconfig:"default=disable"`
	Instance          string `envconfig:"default=iam-service"`
	InstanceNamespace string `envconfig:"default=dxp-system"`
	SSLCertName       string `envconfig:"default=iam-service"`
	SSLCertNamespace  string `envconfig:"default=dxp-system"`
	TmpDir            string `envconfig:"default='.'"`
	MaxIdleConns      int    `envconfig:"default=5"`
	MaxOpenConns      int    `envconfig:"default=20"`
	MaxConnLifetime   string `envconfig:"default=30m"`
	LocalData         DatabaseLocalData
}

type DatabaseLocalData struct {
	DataPathUser                string `envconfig:"default=input/user.yaml"`
	DataPathInvitations         string `envconfig:"default=input/invitations.yaml"`
	DataPathTeam                string `envconfig:"default=input/team.yaml"`
	DataPathTenantConfiguration string `envconfig:"default=input/tenantConfigurations.yaml"`
	DataPathRoles               string `envconfig:"optional"`
	DataPathDomainConfiguration string `envconfig:"default=input/domainConfigurations.yaml"`
}
