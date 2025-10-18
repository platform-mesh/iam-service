package fga

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/platform-mesh/golang-commons/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/status"
)

type StoreHelper struct {
	cache *expirable.LRU[string, string]
}

func NewStoreHelper() *StoreHelper {
	return &StoreHelper{cache: expirable.NewLRU[string, string](10, nil, 10*time.Minute)}
}

func NewStoreHelperWithTTL(ttl time.Duration) *StoreHelper {
	return &StoreHelper{cache: expirable.NewLRU[string, string](10, nil, ttl)}
}

func (d StoreHelper) GetStoreID(ctx context.Context, conn openfgav1.OpenFGAServiceClient, orgID string) (string, error) {

	cacheKey := "store-" + orgID
	s, ok := d.cache.Get(cacheKey)
	if ok && s != "" {
		return s, nil
	}

	stores, err := conn.ListStores(ctx, &openfgav1.ListStoresRequest{})
	if err != nil {
		log.Error().Err(err).Msg("Failed to list stores")
		return "", err
	}

	idx := slices.IndexFunc(stores.Stores, func(store *openfgav1.Store) bool {
		return store.Name == orgID
	})
	if idx == -1 {
		return "", fmt.Errorf("store with name %s not found", orgID)
	}

	storeID := stores.Stores[idx].Id
	d.cache.Add(cacheKey, storeID)
	return storeID, nil
}

func (d StoreHelper) GetModelID(ctx context.Context, conn openfgav1.OpenFGAServiceClient, orgID string) (string, error) {

	cacheKey := "model-" + orgID
	s, ok := d.cache.Get(cacheKey)
	if ok && s != "" {
		return s, nil
	}

	storeID, err := d.GetStoreID(ctx, conn, orgID)
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
func (d StoreHelper) IsDuplicateWriteError(err error) bool {
	if err == nil {
		return false
	}

	s, ok := status.FromError(err)
	return ok && int32(s.Code()) == int32(openfgav1.ErrorCode_write_failed_due_to_invalid_input)
}
