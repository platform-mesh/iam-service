package logger

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	commonsLogger "github.com/platform-mesh/golang-commons/logger"
)

const (
	requestIDCtxKey    = "request-id-context-key"
	requestIDLoggerKey = "request-id"
)

func NewUnaryInterceptor() grpc.UnaryServerInterceptor {
	return interceptors.UnaryServerInterceptor(reportable())
}

func reportable() interceptors.CommonReportableFunc {
	return func(ctx context.Context, c interceptors.CallMeta) (interceptors.Reporter, context.Context) {
		logger := commonsLogger.LoadLoggerFromContext(ctx)

		md, exists := metadata.FromOutgoingContext(ctx)
		if exists {
			requestIds := md.Get(requestIDCtxKey)
			if len(requestIds) == 1 {
				logger = logger.ChildLogger(requestIDLoggerKey, requestIds[0])
			}
		}
		ctx = commonsLogger.SetLoggerInContext(ctx, logger)
		return interceptors.NoopReporter{}, ctx
	}
}
