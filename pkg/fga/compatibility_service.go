package fga

import (
	"context"
	"errors"
	"fmt"
	"strings"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/r3labs/diff/v3"
	"go.opentelemetry.io/otel"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	commonsCtx "github.com/platform-mesh/golang-commons/context"
	pmfga "github.com/platform-mesh/golang-commons/fga/store"
	commonsLogger "github.com/platform-mesh/golang-commons/logger"
	commonsSentry "github.com/platform-mesh/golang-commons/sentry"

	"github.com/platform-mesh/iam-service/internal/pkg/utils"
	"github.com/platform-mesh/iam-service/pkg/db"
	"github.com/platform-mesh/iam-service/pkg/fga/middleware/principal"
	"github.com/platform-mesh/iam-service/pkg/fga/types"
	graphql "github.com/platform-mesh/iam-service/pkg/graph"
)

type Service interface {
	UsersForEntity(ctx context.Context, tenantID string, entityID string, entityType string) (types.UserIDToRoles, error)
	UsersForEntityRolefilter(
		ctx context.Context, tenantID string, entityID string, entityType string, rolefilter []*graphql.RoleInput,
	) (types.UserIDToRoles, error)
	AssignRoleBindings(ctx context.Context, tenantID string, entityType string, entityID string, input []*graphql.Change) error
	RemoveFromEntity(ctx context.Context, tenantID string, entityType string, entityID string, userID string) error
	GetPermissionsForRole(ctx context.Context, tenantID string, entityType string, roleTechnicalName string) ([]*graphql.Permission, error)
}

type UserService interface {
	GetUserByID(ctx context.Context, tenantID string, userId string) (*graphql.User, error)
}

type CompatService struct {
	openfgav1.UnimplementedOpenFGAServiceServer
	upstream openfgav1.OpenFGAServiceClient
	helper   pmfga.FGAStoreHelper
	database db.Service
	events   FgaEvents
	roles    []string
}

type FgaEvents interface {
	UserRoleChanged(ctx context.Context, tenantID string, entityID string, entityType string,
		userID string, oldRoles []string, newRoles []string) error
}

func NewCompatClient(cl openfgav1.OpenFGAServiceClient, db db.Service, fgaEvents FgaEvents) (*CompatService, error) {
	return &CompatService{
		upstream: cl,
		helper:   pmfga.NewWithPrefix("tenant-"),
		database: db,
		events:   fgaEvents,
		roles:    types.AllRoleStrings(),
	}, nil
}

func (c *CompatService) WithFGAStoreHelper(helper pmfga.FGAStoreHelper) *CompatService {
	c.helper = helper
	return c
}

var _ openfgav1.OpenFGAServiceServer = (*CompatService)(nil)

func userIDFromContext(ctx context.Context) (string, error) {
	principalID, principalErr := principal.GetPrincipalFromContext(ctx)
	if principalErr != nil {
		return "", principalErr
	}

	token, err := commonsCtx.GetWebTokenFromContext(ctx)
	if err != nil && principalID == "" {
		return "", status.Error(codes.Unauthenticated, "unauthorized")
	}

	if token.Subject != "" {
		return token.Subject, nil
	}

	return principalID, nil
}

type Tuple interface {
	GetUser() string
	GetObject() string
	GetRelation() string
}

// Write implements openfgav1.OpenFGAServiceServer.
func (c *CompatService) Write(ctx context.Context, in *openfgav1.WriteRequest) (*openfgav1.WriteResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	tupleLen := 0
	if in.Writes != nil {
		tupleLen += len(in.Writes.TupleKeys)
	}
	if in.Deletes != nil {
		tupleLen += len(in.Deletes.TupleKeys)
	}

	tuples := make([]Tuple, 0, tupleLen)
	if in.Writes != nil {
		for i := range in.Writes.TupleKeys {
			tuples = append(tuples, in.Writes.TupleKeys[i])
		}
	}
	if in.Deletes != nil {
		for i := range in.Deletes.TupleKeys {
			tuples = append(tuples, in.Deletes.TupleKeys[i])
		}
	}

	logger := commonsLogger.LoadLoggerFromContext(ctx)
	tags := commonsSentry.Tags{}
	for _, write := range tuples { // of course we can parallize this loop, but lets see if it works first 😅
		writeToEntityType := strings.Split(write.GetUser(), ":")[0]
		writeToEntityID := strings.Split(write.GetUser(), ":")[1]

		writeFromEntityType := strings.Split(write.GetObject(), ":")[0]

		tags = commonsSentry.Tags{"entityType": writeToEntityID, "entityID": writeToEntityID, "userID": userID}
		res, err := c.upstream.Check(ctx, &openfgav1.CheckRequest{
			StoreId:              in.StoreId,
			AuthorizationModelId: in.AuthorizationModelId,
			TupleKey: &openfgav1.CheckRequestTupleKey{
				User:     fmt.Sprintf("user:%s", userID),
				Relation: fmt.Sprintf("%s_%s_%s", writeFromEntityType, write.GetRelation(), writeToEntityType),
				Object:   fmt.Sprintf("%s:%s", writeToEntityType, writeToEntityID),
			},
		})
		if err != nil {
			logger.Error().AnErr("openFGA check error", err).Send()
			commonsSentry.CaptureError(err, tags)
			return nil, err
		}
		if !res.Allowed {
			logger.Debug().AnErr("openFGA unauthorized error", err).Send()
			commonsSentry.CaptureError(err, tags)
			return nil, status.Error(codes.Unauthenticated, "not authorized to perform this write operation")
		}
	}

	res, err := c.upstream.Write(ctx, in)
	if c.helper.IsDuplicateWriteError(err) {
		err = nil
	}
	if err != nil {
		logger.Error().AnErr("openFGA write error", err).Send()
		commonsSentry.CaptureError(err, tags)
	}

	return res, err
}

// TODO: to prevent mistakes, we should probably have a separate type for entityType and entityID.
//  Or at least create a separate struct with those two fields.

// UsersForEntity returns a map of user IDs to roles for a given entity.
func (s *CompatService) UsersForEntity(
	ctx context.Context, tenantID string, entityID string, entityType string,
) (types.UserIDToRoles, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "fga.GrantedUsers")
	defer span.End()

	storeID, err := s.helper.GetStoreIDForTenant(ctx, s.upstream, tenantID)
	if err != nil {
		return nil, err
	}

	logger := commonsLogger.LoadLoggerFromContext(ctx)
	userIDToRoles := types.UserIDToRoles{}
	for _, role := range s.roles {
		var continuationToken string
		for {
			roleMembers, err := s.upstream.Read(ctx, &openfgav1.ReadRequest{
				StoreId: storeID,
				TupleKey: &openfgav1.ReadRequestTupleKey{
					Object:   fmt.Sprintf("role:%s/%s/%s", entityType, entityID, role),
					Relation: "assignee",
				},
				PageSize:          wrapperspb.Int32(100),
				ContinuationToken: continuationToken,
			})
			if err != nil {
				logger.Error().AnErr("openFGA read error", err).Send()
				commonsSentry.CaptureError(err, commonsSentry.Tags{"tenantId": tenantID, "entityType": entityType, "entityID": entityID, "role": role})
				return nil, err
			}
			for _, tuple := range roleMembers.Tuples {
				user := tuple.Key.User
				userID := strings.TrimPrefix(user, "user:")

				roleIdRaw := strings.Split(tuple.Key.Object, "/")
				if len(roleIdRaw) < 3 {
					logger.Error().Str("role", tuple.Key.Object).Msg("role ID is not in expected format")
					commonsSentry.CaptureError(errors.New("role ID not in expected format"), commonsSentry.Tags{"role": tuple.Key.Object})
					continue
				}
				roleID := roleIdRaw[2]

				userIDToRoles[userID] = append(userIDToRoles[userID], roleID)
			}

			continuationToken = roleMembers.ContinuationToken
			if continuationToken == "" {
				break
			}
		}
	}

	return userIDToRoles, nil
}

// UsersForEntityRolefilter returns a map of user IDs to roles for a given entity using a rolefilter
func (s *CompatService) UsersForEntityRolefilter(
	ctx context.Context,
	tenantID string,
	entityID string,
	entityType string,
	rolefilter []*graphql.RoleInput,
) (types.UserIDToRoles, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "fga.GrantedUsers")
	defer span.End()

	storeID, err := s.helper.GetStoreIDForTenant(ctx, s.upstream, tenantID)
	if err != nil {
		return nil, err
	}

	logger := commonsLogger.LoadLoggerFromContext(ctx)

	allUserIDToRoles := types.UserIDToRoles{}
	for _, role := range s.roles {
		var continuationToken string
		for {
			roleMembers, err := s.upstream.Read(ctx, &openfgav1.ReadRequest{
				StoreId: storeID,
				TupleKey: &openfgav1.ReadRequestTupleKey{
					Object:   fmt.Sprintf("role:%s/%s/%s", entityType, entityID, role),
					Relation: "assignee",
				},
				PageSize:          wrapperspb.Int32(100),
				ContinuationToken: continuationToken,
			})
			if err != nil {
				logger.Error().AnErr("openFGA read error", err).Send()
				commonsSentry.CaptureError(err, commonsSentry.Tags{"tenantId": tenantID, "entityType": entityType, "entityID": entityID, "role": role})
				return nil, err
			}
			for _, tuple := range roleMembers.Tuples {
				user := tuple.Key.User
				userID := strings.TrimPrefix(user, "user:")

				roleIdRaw := strings.Split(tuple.Key.Object, "/")
				if len(roleIdRaw) < 3 {
					logger.Error().Str("role", tuple.Key.Object).Msg("role ID is not in expected format")
					commonsSentry.CaptureError(errors.New("role ID not in expected format"), commonsSentry.Tags{"role": tuple.Key.Object})
					continue
				}
				roleTechnicalName := roleIdRaw[2]
				allUserIDToRoles[userID] = append(allUserIDToRoles[userID], roleTechnicalName)
			}

			continuationToken = roleMembers.ContinuationToken
			if continuationToken == "" {
				break
			}
		}
	}

	filteredUserIDToRoles := make(types.UserIDToRoles, len(allUserIDToRoles))
	for userID, userRoles := range allUserIDToRoles {
		for _, userRole := range userRoles {
			if utils.CheckRolesFilter(userRole, rolefilter) {
				filteredUserIDToRoles[userID] = userRoles
				break
			}
		}
	}

	return filteredUserIDToRoles, nil
}

func (s *CompatService) AssignRoleBindings(ctx context.Context, tenantID string, entityType string, entityID string, input []*graphql.Change) error { // nolint: gocognit, cyclop,funlen,lll
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "fga.AssignRoleBindings")
	defer span.End()

	logger := commonsLogger.LoadLoggerFromContext(ctx)

	storeID, err := s.helper.GetStoreIDForTenant(ctx, s.upstream, tenantID)
	if err != nil {
		return err
	}

	modelID, err := s.helper.GetModelIDForTenant(ctx, s.upstream, tenantID)
	if err != nil {
		return err
	}

	for _, change := range input {
		tags := commonsSentry.Tags{"tenant": tenantID, "entityType": entityType, "entityID": entityID, "change.userID": change.UserID}

		previousUserRoles, err := s.upstream.ListObjects(ctx, &openfgav1.ListObjectsRequest{
			StoreId:              storeID,
			AuthorizationModelId: modelID,
			User:                 fmt.Sprintf("user:%s", change.UserID),
			Relation:             "assignee",
			Type:                 "role",
		})
		if err != nil {
			logger.Error().AnErr("openFGA read error", err).Send()
			commonsSentry.CaptureError(err, tags)
			return err
		}

		previousUserRolesForEntity := make([]string, 0, len(previousUserRoles.Objects))
		for _, role := range previousUserRoles.Objects {
			if strings.Contains(role, fmt.Sprintf("%s/%s", entityType, entityID)) {
				previousUserRolesForEntity = append(previousUserRolesForEntity, role)
			}
		}

		adjustedRequestRoles := make([]string, len(change.Roles))
		for i, r := range change.Roles {
			adjustedRequestRoles[i] = fmt.Sprintf("role:%s/%s/%s", entityType, entityID, r)
		}

		changelog, err := diff.Diff(previousUserRolesForEntity, adjustedRequestRoles)
		if err != nil {
			return err
		}

		writes := []*openfgav1.TupleKey{}
		deletes := []*openfgav1.TupleKeyWithoutCondition{}
		for _, diffChange := range changelog {
			switch diffChange.Type {
			case diff.CREATE:
				writes = append(writes, &openfgav1.TupleKey{
					Object:   diffChange.To.(string),
					Relation: "assignee",
					User:     fmt.Sprintf("user:%s", change.UserID),
				})
			case diff.UPDATE:
				writes = append(writes, &openfgav1.TupleKey{
					Object:   diffChange.To.(string),
					Relation: "assignee",
					User:     fmt.Sprintf("user:%s", change.UserID),
				})

				deletes = append(deletes, &openfgav1.TupleKeyWithoutCondition{
					Object:   diffChange.From.(string),
					Relation: "assignee",
					User:     fmt.Sprintf("user:%s", change.UserID),
				})
			case diff.DELETE:
				deletes = append(deletes, &openfgav1.TupleKeyWithoutCondition{
					Object:   diffChange.From.(string),
					Relation: "assignee",
					User:     fmt.Sprintf("user:%s", change.UserID),
				})
			}
		}

		req := &openfgav1.WriteRequest{
			StoreId:              storeID,
			AuthorizationModelId: modelID,
		}
		if len(writes) > 0 {
			req.Writes = &openfgav1.WriteRequestWrites{
				TupleKeys: writes,
			}
		}
		if len(deletes) > 0 {
			req.Deletes = &openfgav1.WriteRequestDeletes{
				TupleKeys: deletes,
			}
		}

		if req.Writes != nil || req.Deletes != nil { // nolint: nestif
			_, err = s.upstream.Write(ctx, req)
			if s.helper.IsDuplicateWriteError(err) {
				err = nil
			}
			if err != nil {
				logger.Error().AnErr("openFGA write error", err).Send()
				commonsSentry.CaptureError(err, tags)
				return err
			}

			requestedNewRoles := make([]string, 0, len(change.Roles))
			for _, r := range change.Roles {
				requestedNewRoles = append(requestedNewRoles, fmt.Sprintf("role:%s/%s/%s", entityType, entityID, r))
			}
			if s.events != nil {
				err = s.events.UserRoleChanged(ctx, tenantID, entityID, entityType, change.UserID, previousUserRolesForEntity, requestedNewRoles)
				if err != nil {
					logger.Error().AnErr("error calling UserRoleChanged event handler", err).Send()
					commonsSentry.CaptureError(err, tags)
				}
			}
		}

		rolesForEntity, err := s.database.GetRolesForEntity(ctx, entityType, "")
		if err != nil {
			logger.Error().Str("entityType", entityType).AnErr("GetRolesForEntity", err).Send()
			commonsSentry.CaptureError(err, tags)
			return err
		}

		for _, r := range rolesForEntity {
			_, err = s.upstream.Write(ctx, &openfgav1.WriteRequest{
				StoreId:              storeID,
				AuthorizationModelId: modelID,
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{
						{
							Object:   fmt.Sprintf("%s:%s", entityType, entityID),
							Relation: r.TechnicalName,
							User:     fmt.Sprintf("role:%s/%s/%s#assignee", entityType, entityID, r.TechnicalName),
						},
					},
				},
			})
			if s.helper.IsDuplicateWriteError(err) {
				err = nil
			}
			if err != nil {
				logger.Error().AnErr("openFGA write error", err).Send()
				commonsSentry.CaptureError(err, tags)
				return err
			}
		}
	}

	return nil
}

func (s *CompatService) RemoveFromEntity(ctx context.Context, tenantID string, entityType string, entityID string, userID string) error {
	storeID, err := s.helper.GetStoreIDForTenant(ctx, s.upstream, tenantID)
	if err != nil {
		return err
	}

	deletes := make([]*openfgav1.TupleKeyWithoutCondition, len(s.roles))
	for i, role := range s.roles {
		deletes[i] = &openfgav1.TupleKeyWithoutCondition{
			Object:   fmt.Sprintf("role:%s/%s/%s", entityType, entityID, role),
			Relation: "assignee",
			User:     fmt.Sprintf("user:%s", userID),
		}
	}

	logger := commonsLogger.LoadLoggerFromContext(ctx)
	for _, d := range deletes {
		_, err = s.upstream.Write(ctx, &openfgav1.WriteRequest{
			StoreId: storeID,
			Deletes: &openfgav1.WriteRequestDeletes{
				TupleKeys: []*openfgav1.TupleKeyWithoutCondition{d},
			},
		})
		if s.helper.IsDuplicateWriteError(err) {
			err = nil
		}
		if err != nil {
			logger.Error().AnErr("openFGA write error", err).Send()
			commonsSentry.CaptureError(err, commonsSentry.Tags{"tenant": tenantID, "entityType": entityType, "entityID": entityID, "userID": userID})
			return err
		}
	}

	return err
}

const ParentEntry = "parent"

func (s *CompatService) GetPermissionsForRole(ctx context.Context, tenantID string, entityType string, roleTechnicalName string) ([]*graphql.Permission, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "fga.GetPermissionsForRole")
	defer span.End()

	storeID, err := s.helper.GetStoreIDForTenant(ctx, s.upstream, tenantID)
	if err != nil {
		return nil, err
	}

	modelResponse, err := s.upstream.ReadAuthorizationModels(ctx, &openfgav1.ReadAuthorizationModelsRequest{
		StoreId: storeID,
	})
	if err != nil {
		return nil, err
	}

	if len(modelResponse.AuthorizationModels) == 0 {
		return []*graphql.Permission{}, nil
	}

	model := modelResponse.AuthorizationModels[0]
	var permissions []*graphql.Permission

	for _, typeDef := range model.TypeDefinitions {
		if typeDef.Type == entityType {
			for relationName, relationDef := range typeDef.Relations {
				// skip relations that are not permissions
				if relationName == ParentEntry || types.ParseRole(relationName) != types.Invalid {
					continue
				}

				if s.roleHasPermission(relationDef, roleTechnicalName) {
					permissions = append(permissions, &graphql.Permission{
						DisplayName: s.formatSnakeToTitle(relationName),
						Relation:    relationName,
					})
				}
			}
			break
		}
	}

	return permissions, nil
}

func (s *CompatService) roleHasPermission(relationDef *openfgav1.Userset, roleTechnicalName string) bool {
	union := relationDef.GetUnion()
	if union != nil {
		for _, child := range union.Child {
			computedUserset := child.GetComputedUserset()
			if computedUserset != nil && computedUserset.Relation == roleTechnicalName {
				return true
			}
		}
		return false
	}

	computedUserset := relationDef.GetComputedUserset()
	return computedUserset != nil && computedUserset.Relation == roleTechnicalName
}

func (s *CompatService) formatSnakeToTitle(relation string) string {
	caser := cases.Title(language.English)
	words := strings.Split(relation, "_")
	for i, word := range words {
		words[i] = caser.String(word)
	}
	return strings.Join(words, " ")
}
