package api

import (
	"context"

	"github.com/platform-mesh/iam-service/pkg/graph"
)

type ResolverService interface {
	Me(ctx context.Context) (*graph.User, error)
	User(ctx context.Context, userID string) (*graph.User, error)
}
