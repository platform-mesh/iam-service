package idm

type IDMTenantRetriever interface {
	GetIDMTenant(issuer string) (string, error)
}
