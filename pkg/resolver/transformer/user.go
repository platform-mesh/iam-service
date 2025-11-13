package transformer

import (
	"github.com/platform-mesh/iam-service/pkg/config"
	"github.com/platform-mesh/iam-service/pkg/graph"
)

type UserTransformer struct {
	serviceCfg *config.JWTConfig
}

func NewUserTransformer(serviceCfg *config.JWTConfig) *UserTransformer {
	return &UserTransformer{
		serviceCfg: serviceCfg,
	}
}

func (t *UserTransformer) Transform(user *graph.User) *graph.User {
	if user == nil {
		return nil
	}

	switch t.serviceCfg.UserIDClaim {
	case "email":
		user.UserID = user.Email
	default:
		// no-op
	}

	return user
}
