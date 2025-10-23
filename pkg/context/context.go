package context

import (
	"context"
	"fmt"

	"github.com/platform-mesh/golang-commons/errors"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	// kcpContextKey is the key for storing KCP user context
	kcpContextKey contextKey = "KCPContext"
	// clusterIdContextKey is the key for storing Cluster ID
	clusterIdContextKey contextKey = "clusterId"
)

// KCPContext holds KCP-related user information
type KCPContext struct {
	IDMTenant        string
	OrganizationName string
}

// SetKCPContext stores KCP context information in the request context
func SetKCPContext(ctx context.Context, kcpCtx KCPContext) context.Context {
	return context.WithValue(ctx, kcpContextKey, kcpCtx)
}

// GetKCPContext retrieves KCP context information from the request context
func GetKCPContext(ctx context.Context) (KCPContext, error) {
	val := ctx.Value(kcpContextKey)
	if val == nil {
		return KCPContext{}, errors.New("kcp user context not found in context")
	}

	kcpCtx, ok := val.(KCPContext)
	if !ok {
		return KCPContext{}, errors.New("invalid kcp user context type")
	}

	return kcpCtx, nil
}

// SetClusterId stores the Cluster ID in the request context
func SetClusterId(ctx context.Context, clusterId string) context.Context {
	return context.WithValue(ctx, clusterIdContextKey, clusterId)
}

// GetClusterId retrieves the Cluster ID from the request context
func GetClusterId(ctx context.Context) (string, error) {
	val := ctx.Value(clusterIdContextKey)
	if val == nil {
		return "", fmt.Errorf("account info not found in context")
	}
	clusterId, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("invalid account info type in context")
	}
	return clusterId, nil
}
