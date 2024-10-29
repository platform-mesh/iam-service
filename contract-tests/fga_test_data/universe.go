package fga_test_data

import (
	"context"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openfga/language/pkg/go/transformer"
	"github.com/openfga/openfga/pkg/server"
	"github.com/openfga/openfga/pkg/storage/memory"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
)

type FgaData struct {
	Schema          []byte
	TenantRelations []byte
	UserRelations   []byte
}

type fgaRelations struct {
	Tuples []struct {
		Object   string `json:"object"`
		Relation string `json:"relation"`
		User     string `json:"user"`
	} `yaml:"tuples"`
}

func GetOpenfgaServer(ctx context.Context, tenantId string, input FgaData) (*server.Server, error) {
	openfgaServer := server.MustNewServerWithOpts(server.WithDatastore(memory.New()))
	storeRes, err := openfgaServer.CreateStore(ctx, &openfgav1.CreateStoreRequest{
		Name: "tenant-" + tenantId,
	})
	if err != nil {
		return nil, err
	}

	model := transformer.MustTransformDSLToProto(string(input.Schema))
	_, err = openfgaServer.WriteAuthorizationModel(ctx, &openfgav1.WriteAuthorizationModelRequest{
		StoreId:         storeRes.Id,
		TypeDefinitions: model.TypeDefinitions,
		SchemaVersion:   model.SchemaVersion,
	})
	if err != nil {
		return nil, err
	}

	var tenantRelations fgaRelations
	err = yaml.Unmarshal(input.TenantRelations, &tenantRelations)
	if err != nil {
		return nil, err
	}

	var userRelations fgaRelations
	err = yaml.Unmarshal(input.UserRelations, &userRelations)
	if err != nil {
		return nil, err
	}

	for _, tuple := range append(tenantRelations.Tuples, userRelations.Tuples...) {
		_, err = openfgaServer.Write(ctx, &openfgav1.WriteRequest{
			StoreId: storeRes.Id,
			Writes: &openfgav1.WriteRequestWrites{
				TupleKeys: []*openfgav1.TupleKey{
					{
						Object:   tuple.Object,
						Relation: tuple.Relation,
						User:     tuple.User,
					},
				},
			},
		})
		if IsDuplicateWriteError(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
	}

	return openfgaServer, nil
}

func IsDuplicateWriteError(err error) bool {
	if err == nil {
		return false
	}

	s, ok := status.FromError(err)
	return ok && int32(s.Code()) == int32(openfgav1.ErrorCode_write_failed_due_to_invalid_input) // nolint: gosec
}
