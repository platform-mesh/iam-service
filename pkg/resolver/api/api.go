package api

import (
	"context"

	"github.com/platform-mesh/iam-service/pkg/graph"
)

type ResolverService interface {
	UserByMail(ctx context.Context, userID string) (*graph.User, error)
}
