package resolver

import (
	"context"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"

	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/service/idm"
)

type Service struct {
	fgaClient openfgav1.OpenFGAServiceClient
	idmClient idm.Service
}

func NewResolverService(fgaClient openfgav1.OpenFGAServiceClient, idmClient idm.Service) *Service {
	return &Service{
		fgaClient: fgaClient,
		idmClient: idmClient,
	}
}

func (s *Service) UserById(ctx context.Context, userID string) (*graph.User, error) {
	panic("implement me")
}
