package context

import (
	"context"
	"fmt"

	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	"github.com/platform-mesh/golang-commons/errors"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	// kcpContextKey is the key for storing KCP user context
	kcpContextKey contextKey = "KCPContext"
	// accountInfoContextKey is the key for storing account info
	accountInfoContextKey contextKey = "accountInfo"
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

// SetAccountInfo stores account info in the request context
func SetAccountInfo(ctx context.Context, ai *accountsv1alpha1.AccountInfo) context.Context {
	return context.WithValue(ctx, accountInfoContextKey, ai)
}

// GetAccountInfo retrieves account info from the request context
func GetAccountInfo(ctx context.Context) (*accountsv1alpha1.AccountInfo, error) {
	val := ctx.Value(accountInfoContextKey)
	if val == nil {
		return nil, fmt.Errorf("account info not found in context")
	}
	ai, ok := val.(*accountsv1alpha1.AccountInfo)
	if !ok {
		return nil, fmt.Errorf("invalid account info type in context")
	}
	return ai, nil
}
