package fga

import (
	"context"
	"fmt"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/platform-mesh/golang-commons/sentry"

	"go.opentelemetry.io/otel"

	"github.com/platform-mesh/iam-service/pkg/graph"
	"github.com/platform-mesh/iam-service/pkg/middleware/kcp"
)

type Service struct {
	client openfgav1.OpenFGAServiceClient
	helper *StoreHelper
}

func New(client openfgav1.OpenFGAServiceClient) *Service {
	return &Service{
		client: client,
		helper: NewStoreHelper(),
	}
}

func (s *Service) UsersForResource(
	ctx context.Context,
	resourceContext graph.ResourceContext,
	rolefilter []string,
) ([]string, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "fga.UsersForResource")
	defer span.End()

	kctx, err := kcp.GetKcpUserContext(ctx)
	if err != nil {
		return nil, err
	}

	storeID, err := s.helper.GetStoreID(ctx, s.client, kctx.OrganizationName)
	if err != nil {
		return nil, err
	}
	logger := logger.LoadLoggerFromContext(ctx)

	// core_platform-mesh_io_account:2lhwxtkkheamnri0/default
	object := fmt.Sprintf("%s:%s/%s", resourceContext.GroupResource, kctx.ClusterId, resourceContext.Resource.Name)

	rr, err := s.client.Read(ctx, &openfgav1.ReadRequest{
		StoreId: storeID,
		TupleKey: &openfgav1.ReadRequestTupleKey{
			Object: object,
		},
	})
	if err != nil {
		logger.Error().AnErr("openFGA read error", err).Send()
		sentry.CaptureError(err, nil)
		return nil, err
	}
	logger.Info().Msgf("fga.UsersForResource: got resource %v", rr)

	return nil, nil

	//allUserIDToRoles := types.UserIDToRoles{}
	//for _, role := range s.roles {
	//	var continuationToken string
	//	for {
	//		roleMembers, err := s.client.Read(ctx, &openfgav1.ReadRequest{
	//			StoreId: storeID,
	//			TupleKey: &openfgav1.ReadRequestTupleKey{
	//				Object:   fmt.Sprintf("role:%s/%s/%s", entityType, entityID, role),
	//				Relation: "assignee",
	//			},
	//			PageSize:          wrapperspb.Int32(100),
	//			ContinuationToken: continuationToken,
	//		})
	//		if err != nil {
	//			logger.Error().AnErr("openFGA read error", err).Send()
	//			sentry.CaptureError(err, stags)
	//			return nil, err
	//		}
	//		for _, tuple := range roleMembers.Tuples {
	//			user := tuple.Key.User
	//			userID := strings.TrimPrefix(user, "user:")
	//
	//			roleIdRaw := strings.Split(tuple.Key.Object, "/")
	//			if len(roleIdRaw) < 3 {
	//				logger.Error().Str("role", tuple.Key.Object).Msg("role ID is not in expected format")
	//				sentry.CaptureError(errors.New("role ID not in expected format"), stags)
	//				continue
	//			}
	//			roleTechnicalName := roleIdRaw[2]
	//			allUserIDToRoles[userID] = append(allUserIDToRoles[userID], roleTechnicalName)
	//		}
	//
	//		continuationToken = roleMembers.ContinuationToken
	//		if continuationToken == "" {
	//			break
	//		}
	//	}
	//}
	//
	//	filteredUserIDToRoles := make(types.UserIDToRoles, len(allUserIDToRoles))
	//	for userID, userRoles := range allUserIDToRoles {
	//		for _, userRole := range userRoles {
	//			if utils.CheckRolesFilter(userRole, rolefilter) {
	//				filteredUserIDToRoles[userID] = userRoles
	//				break
	//			}
	//		}
	//	}
	//
	//	return filteredUserIDToRoles, nil
}
