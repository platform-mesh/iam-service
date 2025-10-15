package idm

import (
	"context"

	"github.com/platform-mesh/iam-service/pkg/graph"
)

type Service interface {
	UserByMail(ctx context.Context, userID string) (*graph.User, error)
}
