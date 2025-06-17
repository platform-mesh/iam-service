package fga

import (
	"context"
	"testing"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/platform-mesh/iam-service/pkg/fga/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	storeMocks "github.com/platform-mesh/golang-commons/fga/store/mocks"

	dbMocks "github.com/platform-mesh/iam-service/pkg/db/mocks"
	"github.com/platform-mesh/iam-service/pkg/fga/middleware/principal"
	"github.com/platform-mesh/iam-service/pkg/fga/mocks"
	"github.com/platform-mesh/iam-service/pkg/graph"
)

func TestNewCompatClient(t *testing.T) {
	cl := &mocks.OpenFGAServiceClient{}
	db := &dbMocks.DatabaseService{}
	fgaEvents := &mocks.FgaEvents{}
	s, err := NewCompatClient(cl, db, fgaEvents)
	assert.NoError(t, err)
	assert.NotNil(t, s)
}

func TestUserIDFromContext(t *testing.T) {
	tc := []struct {
		name   string
		ctx    context.Context
		result string
		error  error
	}{
		{
			name:   "success",
			ctx:    principal.SetPrincipalInContext(context.TODO(), "my-principal"),
			result: "my-principal",
			error:  nil,
		},
		{
			name:   "no_principal_ERROR",
			ctx:    context.TODO(),
			result: "",
			error:  principal.ErrNoPrincipalInContext,
		},
		{
			name:   "empty_principal_and_no_token",
			ctx:    principal.SetPrincipalInContext(context.TODO(), ""),
			result: "",
			error:  status.Error(codes.Unauthenticated, "unauthorized"),
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			res, err := userIDFromContext(tt.ctx)
			assert.Equal(t, tt.result, res)
			assert.Equal(t, tt.error, err)
		})
	}

}

func TestWrite(t *testing.T) {
	tc := []struct {
		name       string
		ctx        context.Context
		in         *openfgav1.WriteRequest
		setupMocks func(context.Context, *openfgav1.WriteRequest, *mocks.OpenFGAServiceClient, *storeMocks.FGAStoreHelper)
		error      error
	}{
		{
			name: "success",
			ctx:  principal.SetPrincipalInContext(context.TODO(), "my-principal"),
			in: &openfgav1.WriteRequest{
				StoreId:              "storeId",
				AuthorizationModelId: "authorizationModelId",
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{
						{
							User:     "user:alice",
							Relation: "reader",
							Object:   "document:doc1",
						},
					},
				},
				Deletes: &openfgav1.WriteRequestDeletes{
					TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
						{
							User:     "user:alice",
							Relation: "reader",
							Object:   "document:doc2",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest,
				openFGAServiceClientMock *mocks.OpenFGAServiceClient, fgaStoreHelperMock *storeMocks.FGAStoreHelper) {
				openFGAServiceClientMock.EXPECT().Check(ctx, &openfgav1.CheckRequest{
					StoreId:              in.StoreId,
					AuthorizationModelId: in.AuthorizationModelId,
					TupleKey: &openfgav1.CheckRequestTupleKey{
						User:     "user:my-principal",
						Relation: "document_reader_user",
						Object:   "user:alice",
					},
				}).Return(&openfgav1.CheckResponse{
					Allowed: true,
				}, nil).Twice()

				openFGAServiceClientMock.EXPECT().Write(ctx, in).
					Return(&openfgav1.WriteResponse{}, nil).
					Once()

				fgaStoreHelperMock.EXPECT().IsDuplicateWriteError(mock.Anything).
					Return(true).
					Once()
			},
			error: nil,
		},
		{
			name:  "no_principal_ERROR",
			ctx:   context.TODO(),
			error: principal.ErrNoPrincipalInContext,
		},
		{
			name: "check_ERROR",
			ctx:  principal.SetPrincipalInContext(context.TODO(), "my-principal"),
			in: &openfgav1.WriteRequest{
				StoreId:              "storeId",
				AuthorizationModelId: "authorizationModelId",
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{
						{
							User:     "user:alice",
							Relation: "reader",
							Object:   "document:doc1",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest,
				openFGAServiceClientMock *mocks.OpenFGAServiceClient, fgaStoreHelperMock *storeMocks.FGAStoreHelper) {
				openFGAServiceClientMock.EXPECT().Check(ctx, &openfgav1.CheckRequest{
					StoreId:              in.StoreId,
					AuthorizationModelId: in.AuthorizationModelId,
					TupleKey: &openfgav1.CheckRequestTupleKey{
						User:     "user:my-principal",
						Relation: "document_reader_user",
						Object:   "user:alice",
					},
				}).Return(nil, assert.AnError).Once()
			},
			error: assert.AnError,
		},
		{
			name: "check_response_is_not_allowed_ERROR",
			ctx:  principal.SetPrincipalInContext(context.TODO(), "my-principal"),
			in: &openfgav1.WriteRequest{
				StoreId:              "storeId",
				AuthorizationModelId: "authorizationModelId",
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{
						{
							User:     "user:alice",
							Relation: "reader",
							Object:   "document:doc1",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest,
				openFGAServiceClientMock *mocks.OpenFGAServiceClient, fgaStoreHelperMock *storeMocks.FGAStoreHelper) {
				openFGAServiceClientMock.EXPECT().Check(ctx, &openfgav1.CheckRequest{
					StoreId:              in.StoreId,
					AuthorizationModelId: in.AuthorizationModelId,
					TupleKey: &openfgav1.CheckRequestTupleKey{
						User:     "user:my-principal",
						Relation: "document_reader_user",
						Object:   "user:alice",
					},
				}).Return(&openfgav1.CheckResponse{
					Allowed: false,
				}, nil).Once()
			},
			error: status.Error(codes.Unauthenticated, "not authorized to perform this write operation"),
		},
		{
			name: "upstream_write_ERROR",
			ctx:  principal.SetPrincipalInContext(context.TODO(), "my-principal"),
			in: &openfgav1.WriteRequest{
				StoreId:              "storeId",
				AuthorizationModelId: "authorizationModelId",
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{
						{
							User:     "user:alice",
							Relation: "reader",
							Object:   "document:doc1",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest,
				openFGAServiceClientMock *mocks.OpenFGAServiceClient, fgaStoreHelperMock *storeMocks.FGAStoreHelper) {
				openFGAServiceClientMock.EXPECT().Check(ctx, &openfgav1.CheckRequest{
					StoreId:              in.StoreId,
					AuthorizationModelId: in.AuthorizationModelId,
					TupleKey: &openfgav1.CheckRequestTupleKey{
						User:     "user:my-principal",
						Relation: "document_reader_user",
						Object:   "user:alice",
					},
				}).Return(&openfgav1.CheckResponse{
					Allowed: true,
				}, nil).Once()

				fgaStoreHelperMock.EXPECT().IsDuplicateWriteError(mock.Anything).
					Return(false).
					Once()

				openFGAServiceClientMock.EXPECT().Write(ctx, in).
					Return(nil, assert.AnError).
					Once()
			},
			error: assert.AnError,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			// setup service
			openFGAServiceClientMock := &mocks.OpenFGAServiceClient{}
			fgaStoreHelperMock := &storeMocks.FGAStoreHelper{}
			s := CompatService{
				upstream: openFGAServiceClientMock,
				helper:   fgaStoreHelperMock,
				roles:    getRoles(),
			}

			// setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(tt.ctx, tt.in, openFGAServiceClientMock, fgaStoreHelperMock)
			}

			// execute
			_, err := s.Write(tt.ctx, tt.in)
			assert.Equal(t, tt.error, err)
		})
	}
}

func TestUsersForEntity(t *testing.T) {
	tc := []struct {
		name       string
		setupMocks func(*mocks.OpenFGAServiceClient, *storeMocks.FGAStoreHelper)
		error      error
		result     types.UserIDToRoles
	}{
		{
			name: "success",
			result: types.UserIDToRoles{
				"alice": []string{"member", "vault_maintainer"},
				"bob":   []string{"member"},
			},
			setupMocks: func(client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("storeId", nil).Once()

				client.EXPECT().Read(mock.Anything, &openfgav1.ReadRequest{
					StoreId: "storeId",
					TupleKey: &openfgav1.ReadRequestTupleKey{
						Relation: "assignee",
						Object:   "role:entityType/entityId/member",
					},
					PageSize: wrapperspb.Int32(100),
				}).Return(&openfgav1.ReadResponse{
					Tuples: []*openfgav1.Tuple{
						{
							Key: &openfgav1.TupleKey{
								User:   "user:alice",
								Object: "role:entityType/entityId/member",
							},
						},
						{
							Key: &openfgav1.TupleKey{
								User:   "user:bob",
								Object: "role:entityType/entityId/member",
							},
						},
					},
				}, nil).Once()

				client.EXPECT().Read(mock.Anything, &openfgav1.ReadRequest{
					StoreId: "storeId",
					TupleKey: &openfgav1.ReadRequestTupleKey{
						Relation: "assignee",
						Object:   "role:entityType/entityId/owner",
					},
					PageSize: wrapperspb.Int32(100),
				}).Return(&openfgav1.ReadResponse{
					Tuples: []*openfgav1.Tuple{
						{
							Key: &openfgav1.TupleKey{
								User:   "user:alice",
								Object: "NOT_VALID_OBJECT",
							},
						},
					},
				}, nil).Once()

				client.EXPECT().Read(mock.Anything, &openfgav1.ReadRequest{
					StoreId: "storeId",
					TupleKey: &openfgav1.ReadRequestTupleKey{
						Relation: "assignee",
						Object:   "role:entityType/entityId/vault_maintainer",
					},
					PageSize: wrapperspb.Int32(100),
				}).Return(&openfgav1.ReadResponse{
					Tuples: []*openfgav1.Tuple{
						{
							Key: &openfgav1.TupleKey{
								User:   "user:alice",
								Object: "role:entityType/entityId/vault_maintainer",
							},
						},
					},
				}, nil).Once()
			},
		},
		{
			name:  "GetStoreIDForTenant_ERROR",
			error: assert.AnError,
			setupMocks: func(client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("", assert.AnError).Once()
			},
		},
		{
			name:  "read_ERROR",
			error: assert.AnError,
			setupMocks: func(client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("storeId", nil).Once()

				client.EXPECT().Read(mock.Anything, &openfgav1.ReadRequest{
					StoreId: "storeId",
					TupleKey: &openfgav1.ReadRequestTupleKey{
						Relation: "assignee",
						Object:   "role:entityType/entityId/owner",
					},
					PageSize: wrapperspb.Int32(100),
				}).Return(nil, assert.AnError).Once()
			},
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			// setup service
			openFGAServiceClientMock := &mocks.OpenFGAServiceClient{}
			fgaStoreHelperMock := &storeMocks.FGAStoreHelper{}
			s := CompatService{
				upstream: openFGAServiceClientMock,
				helper:   fgaStoreHelperMock,
				roles:    getRoles(),
			}

			// setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(openFGAServiceClientMock, fgaStoreHelperMock)
			}

			// execute
			res, err := s.UsersForEntity(context.TODO(), "tenantID", "entityId", "entityType")
			assert.Equal(t, tt.error, err)
			assert.Equal(t, tt.result, res)
		})
	}
}

func TestCreateAccount(t *testing.T) {
	tc := []struct {
		name       string
		ctx        context.Context
		setupMocks func(context.Context, *mocks.OpenFGAServiceClient, *storeMocks.FGAStoreHelper)
		error      error
	}{
		{
			name: "success",
			ctx:  context.TODO(),
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(ctx, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(ctx, mock.Anything, "tenantID").
					Return("modelId", nil).Once()

				client.EXPECT().Write(mock.Anything, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Writes: &openfgav1.WriteRequestWrites{
						TupleKeys: []*openfgav1.TupleKey{
							{
								User:     "user:ownerUserID",
								Relation: "assignee",
								Object:   "role:entitytype/entityID/owner",
							},
						},
					},
				}).Return(nil, nil).Once()

				client.EXPECT().Write(mock.Anything, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Writes: &openfgav1.WriteRequestWrites{
						TupleKeys: []*openfgav1.TupleKey{
							{
								User:     "role:entitytype/entityID/owner#assignee",
								Relation: "owner",
								Object:   "entitytype:entityID",
							},
						},
					},
				}).Return(nil, nil).Once()

				helper.EXPECT().IsDuplicateWriteError(mock.Anything).Return(false).Twice()
			},
		},
		{
			name:  "get_store_id_for_tenant_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(ctx, mock.Anything, "tenantID").
					Return("", assert.AnError).Once()
			},
		},
		{
			name:  "get_model_id_for_tenant_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(ctx, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(ctx, mock.Anything, "tenantID").
					Return("", assert.AnError).Once()
			},
		},
		{
			name:  "is_duplicated_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(ctx, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(ctx, mock.Anything, "tenantID").
					Return("modelId", nil).Once()

				client.EXPECT().Write(mock.Anything, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Writes: &openfgav1.WriteRequestWrites{
						TupleKeys: []*openfgav1.TupleKey{
							{
								User:     "user:ownerUserID",
								Relation: "assignee",
								Object:   "role:entitytype/entityID/owner",
							},
						},
					},
				}).Return(nil, assert.AnError).Once()

				helper.EXPECT().IsDuplicateWriteError(assert.AnError).Return(true).Once()
			},
		},
		{
			name:  "write_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(ctx, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(ctx, mock.Anything, "tenantID").
					Return("modelId", nil).Once()

				client.EXPECT().Write(mock.Anything, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Writes: &openfgav1.WriteRequestWrites{
						TupleKeys: []*openfgav1.TupleKey{
							{
								User:     "user:ownerUserID",
								Relation: "assignee",
								Object:   "role:entitytype/entityID/owner",
							},
						},
					},
				}).Return(nil, assert.AnError).Once()

				helper.EXPECT().IsDuplicateWriteError(assert.AnError).Return(false).Once()
			},
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			// setup service
			openFGAServiceClientMock := &mocks.OpenFGAServiceClient{}
			fgaStoreHelperMock := &storeMocks.FGAStoreHelper{}
			s := CompatService{
				upstream: openFGAServiceClientMock,
				helper:   fgaStoreHelperMock,
				roles:    getRoles(),
			}

			// setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(tt.ctx, openFGAServiceClientMock, fgaStoreHelperMock)
			}

			// execute
			err := s.CreateAccount(tt.ctx, "tenantID", "entityType", "entityID", "ownerUserID")
			assert.Equal(t, tt.error, err)
		})
	}
}

func TestRemoveAccount(t *testing.T) {
	tc := []struct {
		name       string
		ctx        context.Context
		setupMocks func(context.Context, *mocks.OpenFGAServiceClient, *storeMocks.FGAStoreHelper)
		error      error
	}{
		{
			name: "success",
			ctx:  context.TODO(),
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(ctx, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(ctx, mock.Anything, "tenantID").
					Return("modelId", nil).Once()

				client.EXPECT().Read(ctx, &openfgav1.ReadRequest{
					StoreId: "storeId",
					TupleKey: &openfgav1.ReadRequestTupleKey{
						Relation: "assignee",
						Object:   "role:entitytype/entityID/owner",
					},
					PageSize: wrapperspb.Int32(100),
				}).Return(&openfgav1.ReadResponse{
					Tuples: []*openfgav1.Tuple{{
						Key: &openfgav1.TupleKey{
							User:     "user:alice",
							Object:   "role:entitytype/entityID/owner",
							Relation: "assignee",
						}},
					},
				}, nil).Once()

				client.EXPECT().Write(ctx, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Deletes: &openfgav1.WriteRequestDeletes{
						TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
							{
								User:     "user:alice",
								Relation: "assignee",
								Object:   "role:entitytype/entityID/owner",
							},
						},
					},
				}).Return(nil, nil).Once()

				client.EXPECT().Read(ctx, &openfgav1.ReadRequest{
					StoreId: "storeId",
					TupleKey: &openfgav1.ReadRequestTupleKey{
						Relation: "assignee",
						Object:   "role:entitytype/entityID/member",
					},
					PageSize: wrapperspb.Int32(100),
				}).Return(&openfgav1.ReadResponse{}, nil).Once()

				client.EXPECT().Read(ctx, &openfgav1.ReadRequest{
					StoreId: "storeId",
					TupleKey: &openfgav1.ReadRequestTupleKey{
						Relation: "assignee",
						Object:   "role:entitytype/entityID/vault_maintainer",
					},
					PageSize: wrapperspb.Int32(100),
				}).Return(&openfgav1.ReadResponse{}, nil).Once()

				client.EXPECT().Write(ctx, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Deletes: &openfgav1.WriteRequestDeletes{
						TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
							{
								User:     "role:entitytype/entityID/owner#assignee",
								Relation: "owner",
								Object:   "entitytype:entityID",
							},
						},
					},
				}).Return(nil, nil).Once()

				client.EXPECT().Write(ctx, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Deletes: &openfgav1.WriteRequestDeletes{
						TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
							{
								User:     "role:entitytype/entityID/member#assignee",
								Relation: "member",
								Object:   "entitytype:entityID",
							},
						},
					},
				}).Return(nil, nil).Once()

				client.EXPECT().Write(ctx, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Deletes: &openfgav1.WriteRequestDeletes{
						TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
							{
								User:     "role:entitytype/entityID/vault_maintainer#assignee",
								Relation: "vault_maintainer",
								Object:   "entitytype:entityID",
							},
						},
					},
				}).Return(nil, nil).Once()

				helper.EXPECT().IsDuplicateWriteError(mock.Anything).Return(false).Times(3)
			},
		},
		{
			name:  "get_store_id_for_tenant_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(ctx, mock.Anything, "tenantID").
					Return("", assert.AnError).Once()
			},
		},
		{
			name:  "get_model_id_for_tenant_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(ctx, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(ctx, mock.Anything, "tenantID").
					Return("", assert.AnError).Once()
			},
		},
		{
			name:  "upstream_read_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(ctx, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(ctx, mock.Anything, "tenantID").
					Return("modelId", nil).Once()

				client.EXPECT().Read(ctx, mock.Anything).Return(nil, assert.AnError).Once()
			},
		},
		{
			name:  "upstream_write_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(ctx, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(ctx, mock.Anything, "tenantID").
					Return("modelId", nil).Once()

				client.EXPECT().Read(ctx, mock.Anything).Return(&openfgav1.ReadResponse{
					Tuples: []*openfgav1.Tuple{{
						Key: &openfgav1.TupleKey{
							User:     "user:alice",
							Object:   "role:entitytype/entityID/owner",
							Relation: "assignee",
						}},
					},
				}, nil).Times(3)

				client.EXPECT().Write(ctx, mock.Anything).Return(nil, assert.AnError).Once()
			},
		},
		{
			name:  "delete_role_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(ctx, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(ctx, mock.Anything, "tenantID").
					Return("modelId", nil).Once()

				client.EXPECT().Read(ctx, mock.Anything).Return(&openfgav1.ReadResponse{
					Tuples: []*openfgav1.Tuple{{
						Key: &openfgav1.TupleKey{
							User:     "user:alice",
							Object:   "role:entitytype/entityID/owner",
							Relation: "assignee",
						}},
					},
				}, nil).Once()

				client.EXPECT().Write(ctx, mock.Anything).Return(nil, nil).Once()

				client.EXPECT().Read(ctx, mock.Anything).Return(&openfgav1.ReadResponse{}, nil).Once()

				client.EXPECT().Read(ctx, mock.Anything).Return(&openfgav1.ReadResponse{}, nil).Once()

				client.EXPECT().Write(ctx, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Deletes: &openfgav1.WriteRequestDeletes{
						TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
							{
								User:     "role:entitytype/entityID/owner#assignee",
								Relation: "owner",
								Object:   "entitytype:entityID",
							},
						},
					},
				}).Return(nil, assert.AnError).Once()

				helper.EXPECT().IsDuplicateWriteError(mock.Anything).Return(false).Once()
			},
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			// setup service
			openFGAServiceClientMock := &mocks.OpenFGAServiceClient{}
			fgaStoreHelperMock := &storeMocks.FGAStoreHelper{}
			s := CompatService{
				upstream: openFGAServiceClientMock,
				helper:   fgaStoreHelperMock,
				roles:    getRoles(),
			}

			// setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(tt.ctx, openFGAServiceClientMock, fgaStoreHelperMock)
			}

			// execute
			err := s.RemoveAccount(tt.ctx, "tenantID", "entityType", "entityID")
			assert.Equal(t, tt.error, err)
		})
	}
}

func TestAssignRoleBindings(t *testing.T) {
	tc := []struct {
		name       string
		ctx        context.Context
		setupMocks func(context.Context, *mocks.OpenFGAServiceClient, *storeMocks.FGAStoreHelper)
		error      error
	}{
		{
			// success case assigns role "member"
			name: "success",
			ctx:  context.TODO(),
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("modelId", nil).Once()

				client.EXPECT().ListObjects(mock.Anything, &openfgav1.ListObjectsRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Type:                 "role",
					Relation:             "assignee",
					User:                 "user:alice",
				}).Return(&openfgav1.ListObjectsResponse{
					Objects: []string{
						"role:entityType/entityID/owner",
						"role:entityType/entityID/vault_maintainer",
					},
				}, nil).Once()

				client.EXPECT().Write(mock.Anything, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Writes: &openfgav1.WriteRequestWrites{
						TupleKeys: []*openfgav1.TupleKey{
							{
								User:     "user:alice",
								Relation: "assignee",
								Object:   "role:entityType/entityID/member",
							},
						},
					},
					Deletes: &openfgav1.WriteRequestDeletes{
						TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
							{
								User:     "user:alice",
								Relation: "assignee",
								Object:   "role:entityType/entityID/owner",
							},
							{
								User:     "user:alice",
								Relation: "assignee",
								Object:   "role:entityType/entityID/vault_maintainer",
							},
						},
					},
				}).Return(nil, assert.AnError).Once()

				client.EXPECT().Write(mock.Anything, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Writes: &openfgav1.WriteRequestWrites{
						TupleKeys: []*openfgav1.TupleKey{
							{
								User:     "role:entityType/entityID/owner#assignee",
								Relation: "owner",
								Object:   "entityType:entityID",
							},
						},
					},
				}).Return(nil, nil).Once()

				client.EXPECT().Write(mock.Anything, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Writes: &openfgav1.WriteRequestWrites{
						TupleKeys: []*openfgav1.TupleKey{
							{
								User:     "role:entityType/entityID/member#assignee",
								Relation: "member",
								Object:   "entityType:entityID",
							},
						},
					},
				}).Return(nil, nil).Once()

				client.EXPECT().Write(mock.Anything, &openfgav1.WriteRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Writes: &openfgav1.WriteRequestWrites{
						TupleKeys: []*openfgav1.TupleKey{
							{
								User:     "role:entityType/entityID/vault_maintainer#assignee",
								Relation: "vault_maintainer",
								Object:   "entityType:entityID",
							},
						},
					},
				}).Return(nil, nil).Once()

				helper.EXPECT().IsDuplicateWriteError(mock.Anything).Return(true).Times(4)
			},
		},
		{
			name:  "get_store_id_for_tenant_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("", assert.AnError).Once()
			},
		},
		{
			name:  "get_model_id_for_tenant_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("", assert.AnError).Once()
			},
		},
		{
			name:  "list_objects_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("modelId", nil).Once()

				client.EXPECT().ListObjects(mock.Anything, mock.Anything).Return(nil, assert.AnError).Once()
			},
		},
		{
			// success case assigns role "member"
			name:  "success",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				helper.EXPECT().GetModelIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("modelId", nil).Once()

				client.EXPECT().ListObjects(mock.Anything, &openfgav1.ListObjectsRequest{
					StoreId:              "storeId",
					AuthorizationModelId: "modelId",
					Type:                 "role",
					Relation:             "assignee",
					User:                 "user:alice",
				}).Return(&openfgav1.ListObjectsResponse{
					Objects: []string{
						"role:entityType/entityID/owner",
						"role:entityType/entityID/vault_maintainer",
					},
				}, nil).Once()

				client.EXPECT().Write(mock.Anything, mock.Anything).Return(nil, assert.AnError).Once()

				helper.EXPECT().IsDuplicateWriteError(mock.Anything).Return(false).Once()
			},
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			// setup service
			openFGAServiceClientMock := &mocks.OpenFGAServiceClient{}
			fgaStoreHelperMock := &storeMocks.FGAStoreHelper{}
			s := CompatService{
				upstream: openFGAServiceClientMock,
				helper:   fgaStoreHelperMock,
				roles:    getRoles(),
			}

			// setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(tt.ctx, openFGAServiceClientMock, fgaStoreHelperMock)
			}

			// execute
			err := s.AssignRoleBindings(tt.ctx, "tenantID", "entityType", "entityID", []*graph.Change{
				{
					UserID: "alice",
					Roles:  []string{"member"},
				},
			})
			assert.Equal(t, tt.error, err)
		})
	}
}

func TestRemoveFromEntity(t *testing.T) {
	tc := []struct {
		name       string
		ctx        context.Context
		setupMocks func(context.Context, *mocks.OpenFGAServiceClient, *storeMocks.FGAStoreHelper)
		error      error
	}{
		{
			name: "success",
			ctx:  context.TODO(),
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				client.EXPECT().Write(mock.Anything, &openfgav1.WriteRequest{
					StoreId: "storeId",
					Deletes: &openfgav1.WriteRequestDeletes{
						TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
							{
								User:     "user:alice",
								Relation: "assignee",
								Object:   "role:entityType/entityID/member",
							},
						},
					},
				}).Return(nil, assert.AnError).Once()

				client.EXPECT().Write(mock.Anything, &openfgav1.WriteRequest{
					StoreId: "storeId",
					Deletes: &openfgav1.WriteRequestDeletes{
						TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
							{
								User:     "user:alice",
								Relation: "assignee",
								Object:   "role:entityType/entityID/owner",
							},
						},
					},
				}).Return(nil, assert.AnError).Once()

				client.EXPECT().Write(mock.Anything, &openfgav1.WriteRequest{
					StoreId: "storeId",
					Deletes: &openfgav1.WriteRequestDeletes{
						TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
							{
								User:     "user:alice",
								Relation: "assignee",
								Object:   "role:entityType/entityID/vault_maintainer",
							},
						},
					},
				}).Return(nil, assert.AnError).Once()

				helper.EXPECT().IsDuplicateWriteError(mock.Anything).Return(true).Times(3)
			},
		},
		{
			name:  "get_store_id_for_tenant_ERROR",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("", assert.AnError).Once()
			},
		},
		{
			name:  "success",
			ctx:   context.TODO(),
			error: assert.AnError,
			setupMocks: func(ctx context.Context, client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("storeId", nil).Once()

				client.EXPECT().Write(mock.Anything, mock.Anything).Return(nil, assert.AnError).Once()

				helper.EXPECT().IsDuplicateWriteError(assert.AnError).Return(false).Once()
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			// setup service
			openFGAServiceClientMock := &mocks.OpenFGAServiceClient{}
			fgaStoreHelperMock := &storeMocks.FGAStoreHelper{}
			s := CompatService{
				upstream: openFGAServiceClientMock,
				helper:   fgaStoreHelperMock,
				roles:    getRoles(),
			}

			// setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(tt.ctx, openFGAServiceClientMock, fgaStoreHelperMock)
			}

			// execute
			err := s.RemoveFromEntity(tt.ctx, "tenantID", "entityType", "entityID", "alice")
			assert.Equal(t, tt.error, err)
		})
	}
}

func Test_UsersForEntityRolefilter(t *testing.T) {
	tc := []struct {
		name       string
		setupMocks func(*mocks.OpenFGAServiceClient, *storeMocks.FGAStoreHelper)
		error      error
		result     types.UserIDToRoles
	}{
		{
			name: "success",
			result: types.UserIDToRoles{
				"alice": []string{"member"},
				"bob":   []string{"member"},
			},
			setupMocks: func(client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("storeId", nil).Once()

				client.EXPECT().Read(mock.Anything, &openfgav1.ReadRequest{
					StoreId: "storeId",
					TupleKey: &openfgav1.ReadRequestTupleKey{
						Relation: "assignee",
						Object:   "role:entityType/entityId/member",
					},
					PageSize: wrapperspb.Int32(100),
				}).Return(&openfgav1.ReadResponse{
					Tuples: []*openfgav1.Tuple{
						{
							Key: &openfgav1.TupleKey{
								User:   "user:alice",
								Object: "role:entityType/entityId/member",
							},
						},
						{
							Key: &openfgav1.TupleKey{
								User:   "user:bob",
								Object: "role:entityType/entityId/member",
							},
						},
					},
				}, nil).Once()

				client.EXPECT().Read(mock.Anything, &openfgav1.ReadRequest{
					StoreId: "storeId",
					TupleKey: &openfgav1.ReadRequestTupleKey{
						Relation: "assignee",
						Object:   "role:entityType/entityId/owner",
					},
					PageSize: wrapperspb.Int32(100),
				}).Return(&openfgav1.ReadResponse{
					Tuples: []*openfgav1.Tuple{
						{
							Key: &openfgav1.TupleKey{
								User:   "user:alice",
								Object: "NOT_VALID_OBJECT",
							},
						},
					},
				}, nil).Once()

				client.EXPECT().Read(mock.Anything, &openfgav1.ReadRequest{
					StoreId: "storeId",
					TupleKey: &openfgav1.ReadRequestTupleKey{
						Relation: "assignee",
						Object:   "role:entityType/entityId/vault_maintainer",
					},
					PageSize: wrapperspb.Int32(100),
				}).Return(&openfgav1.ReadResponse{
					Tuples: []*openfgav1.Tuple{
						{
							Key: &openfgav1.TupleKey{
								User:   "user:alice",
								Object: "role:entityType/entityId/vault_maintainer",
							},
						},
					},
				}, nil).Once()
			},
		},
		{
			name:  "GetStoreIDForTenant_ERROR",
			error: assert.AnError,
			setupMocks: func(client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, "tenantID").
					Return("", assert.AnError).Once()
			},
		},
		{
			name:  "read_ERROR",
			error: assert.AnError,
			setupMocks: func(client *mocks.OpenFGAServiceClient, helper *storeMocks.FGAStoreHelper) {
				helper.EXPECT().GetStoreIDForTenant(mock.Anything, mock.Anything, mock.Anything).
					Return("storeId", nil).Once()

				client.EXPECT().Read(mock.Anything, &openfgav1.ReadRequest{
					StoreId: "storeId",
					TupleKey: &openfgav1.ReadRequestTupleKey{
						Relation: "assignee",
						Object:   "role:entityType/entityId/owner",
					},
					PageSize: wrapperspb.Int32(100),
				}).Return(nil, assert.AnError).Once()
			},
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			// setup service
			openFGAServiceClientMock := &mocks.OpenFGAServiceClient{}
			fgaStoreHelperMock := &storeMocks.FGAStoreHelper{}
			s := CompatService{
				upstream: openFGAServiceClientMock,
				helper:   fgaStoreHelperMock,
				roles:    getRoles(),
			}

			// setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(openFGAServiceClientMock, fgaStoreHelperMock)
			}

			// execute
			filefilter := []*graph.RoleInput{
				{DisplayName: "Member", TechnicalName: "member"},
			}
			res, err := s.UsersForEntityRolefilter(context.TODO(), "tenantID", "entityId", "entityType", filefilter)
			assert.Equal(t, tt.error, err)
			assert.Equal(t, tt.result, res)
		})
	}
}
