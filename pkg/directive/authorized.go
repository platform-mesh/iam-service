package directive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/99designs/gqlgen/graphql"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	pmcontext "github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/errors"
	"github.com/platform-mesh/golang-commons/fga/util"
	"github.com/platform-mesh/golang-commons/jwt"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	"github.com/platform-mesh/iam-service/pkg/config"
	appcontext "github.com/platform-mesh/iam-service/pkg/context"
	"github.com/platform-mesh/iam-service/pkg/fga"
	"github.com/platform-mesh/iam-service/pkg/fga/tuples"
	"github.com/platform-mesh/iam-service/pkg/graph"
)

type AuthorizedDirective struct {
	mgr    mcmanager.Manager
	oc     openfgav1.OpenFGAServiceClient
	helper fga.StoreHelper
}

func NewAuthorizedDirective(mgr mcmanager.Manager, oc openfgav1.OpenFGAServiceClient, cfg *config.ServiceConfig) *AuthorizedDirective {
	return &AuthorizedDirective{mgr: mgr, oc: oc, helper: fga.NewFGAStoreHelper(cfg.OpenFGA.StoreCacheTTL)}
}
func (a AuthorizedDirective) Authorized(ctx context.Context, _ any, next graphql.Resolver, permission string) (any, error) {
	log := logger.LoadLoggerFromContext(ctx)
	log.Debug().Msg("Authorized directive called with permission: " + permission)

	token, err := pmcontext.GetWebTokenFromContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get web token from context")
	}

	kctx, err := appcontext.GetKCPContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kcp user context")
	}
	log.Debug().Str("context", fmt.Sprintf("%+v", kctx)).Msg("Retrieved kcp context")

	fieldCtx := graphql.GetFieldContext(ctx)
	rctx, err := extractResourceContextFromArguments(fieldCtx.Args)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("context", fmt.Sprintf("%+v", rctx)).Msg("Retrieved resource context")

	wsClient, err := getWSClient(rctx.AccountPath, log, a.mgr.GetLocalManager().GetConfig(), a.mgr.GetLocalManager().GetScheme())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace client")
	}

	// Retrieve account info from kcp workspace
	ai, err := getAccountInfoFromKcpContext(ctx, wsClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get account info from kcp context")
	}
	if ai.Spec.Organization.Name != kctx.OrganizationName {
		return nil, gqlerror.Errorf("unauthorized")
	}

	// Store account info in context for future use
	ctx = appcontext.SetAccountInfo(ctx, ai)

	// Test if resource exists
	exists, err := a.testIfResourceExists(ctx, rctx, wsClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to test if resource exists")
	}
	if !exists {
		return nil, gqlerror.Errorf("resource does not exist")
	}

	allowed, err := a.testIfAllowed(ctx, ai, rctx, permission, token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to test if action is allowed")
	}
	if !allowed {
		return nil, gqlerror.Errorf("unauthorized")
	}

	return next(ctx)
}

func (a AuthorizedDirective) testIfAllowed(ctx context.Context, ai *accountsv1alpha1.AccountInfo, rctx *graph.ResourceContext, permission string, token jwt.WebToken) (bool, error) {

	ct := tuples.GenerateContextualTuples(rctx, ai)

	fgaTypeName := util.ConvertToTypeName(rctx.Group, rctx.Kind)
	object := fmt.Sprintf("%s:%s/%s", fgaTypeName, ai.Spec.Account.GeneratedClusterId, rctx.Resource.Name)
	if rctx.Resource.Namespace != nil {
		object = fmt.Sprintf("%s:%s/%s/%s", fgaTypeName, ai.Spec.Account.GeneratedClusterId, *rctx.Resource.Namespace, rctx.Resource.Name)
	}
	user := fmt.Sprintf("user:%s", token.Mail)
	storeID, err := a.helper.GetStoreID(ctx, a.oc, ai.Spec.Organization.Name)
	if err != nil {
		return false, errors.Wrap(err, "failed to get store ID for organization %s", ai.Spec.Organization.Name)
	}

	req := openfgav1.CheckRequest{
		ContextualTuples: ct,
		StoreId:          storeID,
		TupleKey: &openfgav1.CheckRequestTupleKey{
			Object:   object,
			Relation: permission,
			User:     user,
		},
	}
	checkResp, err := a.oc.Check(ctx, &req)
	if err != nil {
		return false, errors.Wrap(err, "failed to check permission with openfga")
	}
	return checkResp.Allowed, nil
}

func (a AuthorizedDirective) testIfResourceExists(ctx context.Context, rctx *graph.ResourceContext, wsClient client.Client) (bool, error) {
	gvr := schema.GroupVersionResource{
		Group:    rctx.Group,
		Resource: rctx.Kind,
	}
	gvr, err := wsClient.RESTMapper().ResourceFor(gvr)
	if err != nil {
		return false, errors.Wrap(err, "failed to get GVR for resource")
	}
	gvk, err := wsClient.RESTMapper().KindFor(gvr)
	if err != nil {
		return false, errors.Wrap(err, "failed to get GVK for resource")
	}

	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(gvk)

	// Try to get the resource
	clObj := client.ObjectKey{Name: rctx.Resource.Name}
	if rctx.Resource.Namespace != nil {
		clObj.Namespace = *rctx.Resource.Namespace
	}
	err = wsClient.Get(ctx, clObj, resource)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return false, nil
		}
	}
	return true, nil
}

func getWSClient(accountPath string, log *logger.Logger, restcfg *rest.Config, scheme *runtime.Scheme) (client.Client, error) {
	cfg := rest.CopyConfig(restcfg)
	parsed, err := url.Parse(cfg.Host)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse host")
		return nil, err
	}

	parsed.Path = fmt.Sprintf("/clusters/%s", accountPath)
	cfg.Host = parsed.String()

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Error().Err(err).Msg("unable to construct root client")
		return nil, err
	}
	return cl, nil
}

func getAccountInfoFromKcpContext(ctx context.Context, cl client.Client) (*accountsv1alpha1.AccountInfo, error) {
	log := logger.LoadLoggerFromContext(ctx)
	ai := &accountsv1alpha1.AccountInfo{}
	err := cl.Get(ctx, client.ObjectKey{Name: "account"}, ai)
	if err != nil {
		log.Error().Err(err).Msg("failed to get orgs workspace from kcp")
		return nil, err
	}
	return ai, nil
}

const resourceContextParamName = "context"

func extractResourceContextFromArguments(args map[string]any) (*graph.ResourceContext, error) {
	o, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	var normalizedArgs map[string]any
	err = json.Unmarshal(o, &normalizedArgs)
	if err != nil {
		return nil, err
	}
	val, ok := normalizedArgs[resourceContextParamName]
	if !ok {
		return nil, fmt.Errorf("unable to extract param from request for given paramName %q", resourceContextParamName)
	}
	valBytes, err := json.Marshal(val)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal param value: %w", err)
	}
	var paramValue graph.ResourceContext
	err = json.Unmarshal(valBytes, &paramValue)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal param to ResourceContext: %w", err)
	}
	return &paramValue, nil
}
