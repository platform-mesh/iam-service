package fga

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openmfp/golang-commons/errors"
	openmfpfga "github.com/openmfp/golang-commons/fga/store"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/status"
)

type OpenMFPStoreHelper struct {
	cache *expirable.LRU[string, string]
}

func NewOpenMFPStoreHelper() *OpenMFPStoreHelper {
	return &OpenMFPStoreHelper{cache: expirable.NewLRU[string, string](10, nil, 10*time.Minute)}
}

var _ openmfpfga.FGAStoreHelper = (*OpenMFPStoreHelper)(nil)

func (d OpenMFPStoreHelper) GetStoreIDForTenant(ctx context.Context, conn openfgav1.OpenFGAServiceClient, orgID string) (string, error) {

	cacheKey := "store-" + orgID
	s, ok := d.cache.Get(cacheKey)
	if ok && s != "" {
		return s, nil
	}

	orgIDSplit := strings.Split(orgID, "/")
	if len(orgIDSplit) != 2 {
		return "", fmt.Errorf("invalid tenant id expecting format `cluster/name`")
	}
	orgName := orgIDSplit[1]

	stores, err := conn.ListStores(ctx, &openfgav1.ListStoresRequest{})
	if err != nil {
		log.Error().Err(err).Msg("Failed to list stores")
		return "", err
	}

	idx := slices.IndexFunc(stores.Stores, func(store *openfgav1.Store) bool {
		return store.Name == orgName
	})
	if idx == -1 {
		return "", fmt.Errorf("store with name %s not found", orgID)
	}

	storeID := stores.Stores[idx].Id
	d.cache.Add(cacheKey, storeID)
	return storeID, nil
}

func (d OpenMFPStoreHelper) GetModelIDForTenant(ctx context.Context, conn openfgav1.OpenFGAServiceClient, orgID string) (string, error) {

	cacheKey := "model-" + orgID
	s, ok := d.cache.Get(cacheKey)
	if ok && s != "" {
		return s, nil
	}

	storeID, err := d.GetStoreIDForTenant(ctx, conn, orgID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get store ID for tenant %s", orgID)
	}
	res, err := conn.ReadAuthorizationModels(ctx, &openfgav1.ReadAuthorizationModelsRequest{StoreId: storeID})
	if err != nil {
		return "", err
	}

	if len(res.AuthorizationModels) < 1 {
		return "", errors.New("no authorization models in response. Cannot determine proper AuthorizationModelId")
	}

	modelID := res.AuthorizationModels[0].Id
	d.cache.Add(cacheKey, modelID)

	return modelID, nil
}
func (d OpenMFPStoreHelper) IsDuplicateWriteError(err error) bool {
	if err == nil {
		return false
	}

	s, ok := status.FromError(err)
	return ok && int32(s.Code()) == int32(openfgav1.ErrorCode_write_failed_due_to_invalid_input)
}
