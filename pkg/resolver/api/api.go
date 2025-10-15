package api

import (
	"context"

	"github.com/platform-mesh/iam-service/pkg/graph"
)

type ResolverService interface {
	UserById(ctx context.Context, userID string) (*graph.User, error)
}
