package directives

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	gqlgen "github.com/99designs/gqlgen/graphql"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	openmfpctx "github.com/openmfp/golang-commons/context"
	"github.com/openmfp/golang-commons/logger"
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

var relationMapping = map[string]string{
	"project_create": "create_core_openmfp_org_accounts",
}

func Authorized(openfgaClient openfgav1.OpenFGAServiceClient, log *logger.Logger) func(ctx context.Context, obj any, next gqlgen.Resolver, relation string, entityType *string, entityTypeParamName *string, entityParamName string) (res any, err error) {
	return func(ctx context.Context, obj any, next gqlgen.Resolver, relation string, entityType, entityTypeParamName *string, entityParamName string) (any, error) {

		if mappedRelation, ok := relationMapping[relation]; ok {
			relation = mappedRelation
		}

		if openfgaClient == nil {
			return nil, errors.New("OpenFGAServiceClient is nil. Cannot process request")
		}

		fieldCtx := gqlgen.GetFieldContext(ctx)

		entityID, err := extractNestedKeyFromArgs(fieldCtx.Args, entityParamName)
		if err != nil {
			return nil, err
		}

		orgID, err := extractNestedKeyFromArgs(fieldCtx.Args, "tenantId")
		if err != nil {
			return nil, err
		}

		orgIDSplit := strings.Split(orgID, "/")
		if len(orgIDSplit) != 2 {
			return nil, fmt.Errorf("invalid tenant id expecting format `cluster/name`")
		}
		orgName := orgIDSplit[1]

		stores, err := openfgaClient.ListStores(ctx, &openfgav1.ListStoresRequest{})
		if err != nil {
			log.Error().Err(err).Msg("Failed to list stores")
			return nil, err
		}

		idx := slices.IndexFunc(stores.Stores, func(store *openfgav1.Store) bool {
			return store.Name == orgName
		})
		if idx == -1 {
			return nil, fmt.Errorf("store with name %s not found", orgID)
		}

		storeID := stores.Stores[idx].Id

		user, err := openmfpctx.GetWebTokenFromContext(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get auth token from context")
			return nil, gqlerror.Errorf("unauthorized")
		}

		req := &openfgav1.CheckRequest{
			StoreId: storeID,
			TupleKey: &openfgav1.CheckRequestTupleKey{
				User:     fmt.Sprintf("user:%s", user.Mail), // FIXME: for now, as the email is not the subject of the token
				Relation: relation,
				Object:   fmt.Sprintf("account:%s", entityID),
			},
		}

		res, err := openfgaClient.Check(ctx, req)
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
}
