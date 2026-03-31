package tuples

import (
	"fmt"
	"strings"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	fgamodel "github.com/platform-mesh/golang-commons/fga/model"

	"github.com/platform-mesh/iam-service/pkg/graph"
)

func GenerateContextualTuples(rctx *graph.ResourceContext, ai *accountsv1alpha1.AccountInfo) *openfgav1.ContextualTupleKeys {
	accountObject := fmt.Sprintf("%s:%s/%s",
		fgamodel.BuildObjectType("core.platform-mesh.io", "account"),
		ai.Spec.Account.OriginClusterId,
		ai.Spec.Account.Name,
	)

	var namespace string
	if rctx.Resource.Namespace != nil {
		namespace = *rctx.Resource.Namespace
	}

	// Skip the resource tuple for managed types (e.g. core.platform-mesh.io/Account);
	// they are their own FGA identity and only need the namespace tuple (if namespaced).
	if managedTuple(rctx.Group, rctx.Kind) {
		if namespace == "" {
			return &openfgav1.ContextualTupleKeys{}
		}
		nsObj := fgamodel.BuildObjectName("", "namespace", ai.Spec.Account.GeneratedClusterId, namespace, nil)
		return &openfgav1.ContextualTupleKeys{
			TupleKeys: []*openfgav1.TupleKey{
				{Object: nsObj, Relation: "parent", User: accountObject},
			},
		}
	}

	tupleKeys, err := fgamodel.BuildContextualTuples(accountObject, fgamodel.ResourceContext{
		Group:     rctx.Group,
		Kind:      strings.ToLower(rctx.Kind),
		ClusterID: ai.Spec.Account.GeneratedClusterId,
		Name:      rctx.Resource.Name,
		Namespace: namespace,
	})
	if err != nil {
		return &openfgav1.ContextualTupleKeys{}
	}

	return &openfgav1.ContextualTupleKeys{TupleKeys: tupleKeys}
}

func managedTuple(group, kind string) bool {
	switch strings.ToLower(group) {
	case "core.platform-mesh.io":
		switch strings.ToLower(kind) {
		case "account":
			return true
		}
	}
	return false
}
