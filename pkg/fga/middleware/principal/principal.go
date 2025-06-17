package principal

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/platform-mesh/golang-commons/jwt"
)

type principalCtxKey struct{}

func NewUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "middleware.Principal")
		defer span.End()

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, errors.New("could not extract metadata from context")
		}

		certHeader := md.Get(strings.ToLower(jwt.HeaderSpiffeValue))
		if len(certHeader) == 0 {
			return handler(ctx, req)
		}

		val := jwt.GetURIValue(certHeader[0])
		val = strings.TrimPrefix(val, "spiffe://")
		if val == "" {
			return handler(ctx, req)
		}

		ctx = SetPrincipalInContext(ctx, val)

		return handler(ctx, req)
	}
}

var (
	ErrNoPrincipalInContext = errors.New("no principal information found in the context")
	ErrPrincipalType        = errors.New("principal was not provided as string data")
)

func GetPrincipalFromContext(ctx context.Context) (string, error) {
	val := ctx.Value(principalCtxKey{})
	if val == nil {
		return "", ErrNoPrincipalInContext
	}
	strVal, ok := val.(string)
	if !ok {
		return "", ErrPrincipalType
	}

	return strVal, nil
}

func SetPrincipalInContext(ctx context.Context, val string) context.Context {
	return context.WithValue(ctx, principalCtxKey{}, val)
}
