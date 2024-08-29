package fga

import (
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openfga/openfga/pkg/middleware/requestid"
	"github.com/openmfp/golang-commons/policy_services"
	"github.com/openmfp/iam-service/pkg/db"
	loggermw "github.com/openmfp/iam-service/pkg/fga/middleware/logger"
	"github.com/openmfp/iam-service/pkg/fga/middleware/principal"
	"github.com/openmfp/iam-service/pkg/fga/middleware/user"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func NewFGAServer(grpcAddr string, db db.Service, fgaEvents FgaEvents, tr policy_services.TenantIdReader, isLocal bool) (*grpc.Server, *CompatService, error) {
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			requestid.NewUnaryInterceptor(),
			loggermw.NewUnaryInterceptor(),
			principal.NewUnaryInterceptor(),
			user.NewUnaryInterceptor(tr),
		),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	srv, err := NewFGAService(grpcAddr, db, fgaEvents)
	if err != nil {
		return nil, nil, err
	}

	openfgav1.RegisterOpenFGAServiceServer(grpcServer, srv)

	if isLocal {
		reflection.Register(grpcServer)
	}

	return grpcServer, srv, nil
}

func NewFGAService(grpcAddr string, db db.Service, fgaEvents FgaEvents) (*CompatService, error) {
	conn, err := grpc.NewClient(grpcAddr,
		grpc.EmptyDialOption{},
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)

	if err != nil {
		return nil, err
	}

	cl := openfgav1.NewOpenFGAServiceClient(conn)

	return NewCompatClient(cl, db, fgaEvents)
}
