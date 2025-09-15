package directives

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	gqlgen "github.com/99designs/gqlgen/graphql"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	pmctx "github.com/platform-mesh/golang-commons/context"
	commonsfga "github.com/platform-mesh/golang-commons/fga/store"
	"github.com/rs/zerolog/log"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func extractNestedKeyFromArgs(args map[string]any, paramName string) (string, error) {
	o, err := json.Marshal(args)
	if err != nil {
		return "", err
	}

	var normalizedArgs map[string]any
	err = json.Unmarshal(o, &normalizedArgs)
	if err != nil {
		return "", err
	}

	var paramValue string
	parts := strings.Split(paramName, ".")
	for i, key := range parts {
		val, ok := normalizedArgs[key]
		if !ok {
			return "", fmt.Errorf("unable to extract param from request for given paramName %q", paramName)
		}

		if i == len(strings.Split(paramName, "."))-1 {
			paramValue, ok = val.(string)
			if !ok || paramValue == "" {
				return "", fmt.Errorf("unable to extract param from request for given paramName %q, param is of wrong type", paramName)
			}

			return paramValue, nil
		}

		normalizedArgs, ok = val.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("unable to extract param from request for given paramName %q, param is of wrong type", paramName)
		}
	}

	return paramValue, nil
}

type AuthorizedDirective struct {
	StoreHelper   commonsfga.FGAStoreHelper
	openfgaClient openfgav1.OpenFGAServiceClient
}

func NewAuthorizedDirective(storeHelper commonsfga.FGAStoreHelper, openfgaClient openfgav1.OpenFGAServiceClient) AuthorizedDirective {
	return AuthorizedDirective{
		StoreHelper:   storeHelper,
		openfgaClient: openfgaClient,
	}
}
func (a AuthorizedDirective) Authorized(ctx context.Context, _ any, next gqlgen.Resolver, relation string, _ *string, _ *string, entityParamName string) (any, error) {
	var relationMapping = map[string]string{
		"project_create": "create_core_platform-mesh_io_accounts",
	}

	if mappedRelation, ok := relationMapping[relation]; ok {
		relation = mappedRelation
	}

	if a.openfgaClient == nil {
		return nil, errors.New("OpenFGAServiceClient is nil. Cannot process request")
	}

	fieldCtx := gqlgen.GetFieldContext(ctx)

	entityID, err := extractNestedKeyFromArgs(fieldCtx.Args, entityParamName)
	if err != nil {
		return nil, err
	}

	tenantID, err := extractNestedKeyFromArgs(fieldCtx.Args, "tenantId")
	if err != nil {
		return nil, err
	}

	storeID, err := a.StoreHelper.GetStoreIDForTenant(ctx, a.openfgaClient, tenantID)
	if err != nil {
		return nil, err
	}

	user, err := pmctx.GetWebTokenFromContext(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get auth token from context")
		return nil, gqlerror.Errorf("unauthorized")
	}

	req := &openfgav1.CheckRequest{
		StoreId: storeID,
		TupleKey: &openfgav1.CheckRequestTupleKey{
			User:     fmt.Sprintf("user:%s", user.Mail), // FIXME: for now, as the email is not the subject of the token
			Relation: relation,
			Object:   fmt.Sprintf("core_platform-mesh_io_account:%s", entityID),
		},
	}

	res, err := a.openfgaClient.Check(ctx, req)
	if err != nil {
		log.Error().Err(err).Str("user", req.TupleKey.User).Msg("authorization check failed")
		return nil, err
	}

	if !res.Allowed {
		log.Warn().Bool("allowed", res.Allowed).Any("req", req).Msg("not allowed")
		return nil, gqlerror.Errorf("unauthorized")
	}

	return next(ctx)
}
