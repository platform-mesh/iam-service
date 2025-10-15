package idm

import (
	"context"

	"github.com/platform-mesh/iam-service/pkg/graph"
)

type Service interface {
	UserById(ctx context.Context, userID string) (*graph.User, error)
}
