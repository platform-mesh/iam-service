package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/go-http-utils/headers"
	"github.com/go-jose/go-jose/v4"
	commonsCtx "github.com/openmfp/golang-commons/context"
	"github.com/openmfp/golang-commons/policy_services"
)

// REFACTOR: I am not happy with this...
func NewUnaryInterceptor(tenentIdReader policy_services.TenantIdReader) grpc.UnaryServerInterceptor {
	tr := policy_services.NewCustomTenantRetriever(tenentIdReader)

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "middleware.User")
		defer span.End()

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, errors.New("could not extract metadata from context")
		}

		authHeader := md.Get(headers.Authorization)
		if len(authHeader) == 0 {
			return handler(ctx, req)
		}

		header := strings.TrimPrefix(authHeader[0], "Bearer")
		header = strings.TrimSpace(header)

		ctx = commonsCtx.AddWebTokenToContext(ctx, header, []jose.SignatureAlgorithm{jose.RS256})
		ctx = commonsCtx.AddAuthHeaderToContext(ctx, authHeader[0])

		token, err := commonsCtx.GetWebTokenFromContext(ctx)
		if err != nil {
			return nil, err
		}

		key := policy_services.TenantKey(fmt.Sprintf("%s-%s", token.Issuer, strings.Join(token.Audiences, "-")))

		tenantID, err := tr.RetrieveOrAdd(ctx, key)
		if err != nil {
			return nil, err
		}

		ctx = commonsCtx.AddTenantToContext(ctx, tenantID)

		return handler(ctx, req)
	}
}
