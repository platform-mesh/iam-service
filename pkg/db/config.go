package db

type ConfigDatabase struct {
	InMemory              bool   `envconfig:"default=true"`
	DSN                   string `envconfig:"optional"`
	User                  string `envconfig:"default=admin"`
	Name                  string `envconfig:"default=aname"`
	SSLMode               string `envconfig:"default=disable"`
	Password              string `envconfig:"default=apassword"`
	IP                    string `envconfig:"optional"`
	Instance              string `envconfig:"default=iam-service"`
	InstanceNamespace     string `envconfig:"default=dxp-system"`
	SSLCertName           string `envconfig:"default=iam-service"`
	SSLCertNamespace      string `envconfig:"default=dxp-system"`
	TmpDir                string `envconfig:"default='.'"`
	MaxIdleConns          int    `envconfig:"default=5"`
	MaxOpenConns          int    `envconfig:"default=20"`
	MaxConnLifetime       string `envconfig:"default=30m"`
	LocalData             DatabaseLocalData
	MaxSearchUsersLimit   int `envconfig:"default=5"` // defines the maximum number of users to return in a search
	MaxSearchUsersTimeout int `envconfig:"default=5"` // defines the maximum time to wait for a search to complete
}

type DatabaseLocalData struct {
	//DataPathUser                string `envconfig:"default=input/user.yaml"`
	//DataPathInvitations         string `envconfig:"default=input/invitations.yaml"`
	//DataPathTeam                string `envconfig:"default=input/team.yaml"`
	//DataPathTenantConfiguration string `envconfig:"default=input/tenantConfigurations.yaml"`
	DataPathRoles string `envconfig:"default=input/roles.yaml"`
	//DataPathDomainConfiguration string `envconfig:"default=input/domainConfigurations.yaml"`
}
