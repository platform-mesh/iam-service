package keycloak

import (
	"regexp"

	"github.com/platform-mesh/golang-commons/errors"
)

type KeycloakIDMRetriever struct{}

func New() *KeycloakIDMRetriever {
	return &KeycloakIDMRetriever{}
}

func (k *KeycloakIDMRetriever) GetIDMTenant(issuer string) (string, error) {
	regex := regexp.MustCompile(`^.*\/realms\/(.*?)\/?$`)
	if !regex.MatchString(issuer) {
		return "", errors.New("token issuer is not valid")
	}

	realm := regex.FindStringSubmatch(issuer)[1]
	return realm, nil
}
