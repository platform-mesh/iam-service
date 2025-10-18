package fga

import (
	"context"
	"errors"
	"testing"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fgamocks "github.com/platform-mesh/iam-service/pkg/fga/mocks"
)

func TestNewStoreHelper(t *testing.T) {
	helper := NewStoreHelper()
	assert.NotNil(t, helper)
	assert.NotNil(t, helper.cache)
}

func TestStoreHelper_GetStoreID_Success(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	helper := NewStoreHelper()

	ctx := context.Background()
	orgID := "test-org"
	expectedStoreID := "store-123"

	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   "store-456",
				Name: "other-org",
			},
			{
				Id:   expectedStoreID,
				Name: orgID,
			},
		},
	}

	client.EXPECT().ListStores(ctx, &openfgav1.ListStoresRequest{}).Return(listStoresResponse, nil)

	storeID, err := helper.GetStoreID(ctx, client, orgID)

	assert.NoError(t, err)
	assert.Equal(t, expectedStoreID, storeID)
}

func TestStoreHelper_GetStoreID_CachedResult(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	helper := NewStoreHelper()

	ctx := context.Background()
	orgID := "test-org"
	expectedStoreID := "store-123"

	// Pre-populate cache
	helper.cache.Add("store-"+orgID, expectedStoreID)

	// Should not call ListStores since it's cached
	storeID, err := helper.GetStoreID(ctx, client, orgID)

	assert.NoError(t, err)
	assert.Equal(t, expectedStoreID, storeID)
}

func TestStoreHelper_GetStoreID_StoreNotFound(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	helper := NewStoreHelper()

	ctx := context.Background()
	orgID := "nonexistent-org"

	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   "store-123",
				Name: "other-org",
			},
		},
	}

	client.EXPECT().ListStores(ctx, &openfgav1.ListStoresRequest{}).Return(listStoresResponse, nil)

	storeID, err := helper.GetStoreID(ctx, client, orgID)

	assert.Error(t, err)
	assert.Empty(t, storeID)
	assert.Contains(t, err.Error(), "store with name nonexistent-org not found")
}

func TestStoreHelper_GetStoreID_ListStoresError(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	helper := NewStoreHelper()

	ctx := context.Background()
	orgID := "test-org"

	client.EXPECT().ListStores(ctx, &openfgav1.ListStoresRequest{}).Return(nil, errors.New("connection failed"))

	storeID, err := helper.GetStoreID(ctx, client, orgID)

	assert.Error(t, err)
	assert.Empty(t, storeID)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestStoreHelper_GetModelID_Success(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	helper := NewStoreHelper()

	ctx := context.Background()
	orgID := "test-org"
	storeID := "store-123"
	expectedModelID := "model-456"

	// Mock ListStores for GetStoreID
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: orgID,
			},
		},
	}
	client.EXPECT().ListStores(ctx, &openfgav1.ListStoresRequest{}).Return(listStoresResponse, nil)

	// Mock ReadAuthorizationModels
	readModelsResponse := &openfgav1.ReadAuthorizationModelsResponse{
		AuthorizationModels: []*openfgav1.AuthorizationModel{
			{
				Id: expectedModelID,
			},
			{
				Id: "model-789",
			},
		},
	}
	client.EXPECT().ReadAuthorizationModels(ctx, &openfgav1.ReadAuthorizationModelsRequest{
		StoreId: storeID,
	}).Return(readModelsResponse, nil)

	modelID, err := helper.GetModelID(ctx, client, orgID)

	assert.NoError(t, err)
	assert.Equal(t, expectedModelID, modelID)
}

func TestStoreHelper_GetModelID_CachedResult(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	helper := NewStoreHelper()

	ctx := context.Background()
	orgID := "test-org"
	expectedModelID := "model-456"

	// Pre-populate cache
	helper.cache.Add("model-"+orgID, expectedModelID)

	modelID, err := helper.GetModelID(ctx, client, orgID)

	assert.NoError(t, err)
	assert.Equal(t, expectedModelID, modelID)
}

func TestStoreHelper_GetModelID_GetStoreIDError(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	helper := NewStoreHelper()

	ctx := context.Background()
	orgID := "test-org"

	// Mock ListStores to fail
	client.EXPECT().ListStores(ctx, &openfgav1.ListStoresRequest{}).Return(nil, errors.New("store error"))

	modelID, err := helper.GetModelID(ctx, client, orgID)

	assert.Error(t, err)
	assert.Empty(t, modelID)
	assert.Contains(t, err.Error(), "failed to get store ID")
}

func TestStoreHelper_GetModelID_NoModels(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	helper := NewStoreHelper()

	ctx := context.Background()
	orgID := "test-org"
	storeID := "store-123"

	// Mock ListStores for GetStoreID
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: orgID,
			},
		},
	}
	client.EXPECT().ListStores(ctx, &openfgav1.ListStoresRequest{}).Return(listStoresResponse, nil)

	// Mock ReadAuthorizationModels with empty response
	readModelsResponse := &openfgav1.ReadAuthorizationModelsResponse{
		AuthorizationModels: []*openfgav1.AuthorizationModel{},
	}
	client.EXPECT().ReadAuthorizationModels(ctx, &openfgav1.ReadAuthorizationModelsRequest{
		StoreId: storeID,
	}).Return(readModelsResponse, nil)

	modelID, err := helper.GetModelID(ctx, client, orgID)

	assert.Error(t, err)
	assert.Empty(t, modelID)
	assert.Contains(t, err.Error(), "no authorization models in response")
}

func TestStoreHelper_GetModelID_ReadModelsError(t *testing.T) {
	client := fgamocks.NewOpenFGAServiceClient(t)
	helper := NewStoreHelper()

	ctx := context.Background()
	orgID := "test-org"
	storeID := "store-123"

	// Mock ListStores for GetStoreID
	listStoresResponse := &openfgav1.ListStoresResponse{
		Stores: []*openfgav1.Store{
			{
				Id:   storeID,
				Name: orgID,
			},
		},
	}
	client.EXPECT().ListStores(ctx, &openfgav1.ListStoresRequest{}).Return(listStoresResponse, nil)

	// Mock ReadAuthorizationModels to fail
	client.EXPECT().ReadAuthorizationModels(ctx, &openfgav1.ReadAuthorizationModelsRequest{
		StoreId: storeID,
	}).Return(nil, errors.New("read models failed"))

	modelID, err := helper.GetModelID(ctx, client, orgID)

	assert.Error(t, err)
	assert.Empty(t, modelID)
	assert.Contains(t, err.Error(), "read models failed")
}

func TestStoreHelper_IsDuplicateWriteError_True(t *testing.T) {
	helper := NewStoreHelper()

	// Create a gRPC status error with the specific error code
	grpcError := status.Error(codes.Code(openfgav1.ErrorCode_write_failed_due_to_invalid_input), "duplicate write")

	result := helper.IsDuplicateWriteError(grpcError)
	assert.True(t, result)
}

func TestStoreHelper_IsDuplicateWriteError_False_DifferentCode(t *testing.T) {
	helper := NewStoreHelper()

	// Create a gRPC status error with a different error code
	grpcError := status.Error(codes.NotFound, "not found")

	result := helper.IsDuplicateWriteError(grpcError)
	assert.False(t, result)
}

func TestStoreHelper_IsDuplicateWriteError_False_NonGRPCError(t *testing.T) {
	helper := NewStoreHelper()

	// Create a regular error
	regularError := errors.New("regular error")

	result := helper.IsDuplicateWriteError(regularError)
	assert.False(t, result)
}

func TestStoreHelper_IsDuplicateWriteError_False_NilError(t *testing.T) {
	helper := NewStoreHelper()

	result := helper.IsDuplicateWriteError(nil)
	assert.False(t, result)
}
