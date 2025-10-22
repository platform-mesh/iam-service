package accountinfo

import (
	"context"
	"fmt"

	kcpclientset "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/cluster"
	"github.com/kcp-dev/logicalcluster/v3"
	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	"github.com/platform-mesh/golang-commons/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

type Retriever interface {
	Get(ctx context.Context, accountPath string) (*accountsv1alpha1.AccountInfo, error)
}

type accountInfoRetriever struct {
	mgr           mcmanager.Manager
	clusterClient kcpclientset.ClusterInterface
}

func New(mgr mcmanager.Manager, clusterClient kcpclientset.ClusterInterface) (Retriever, error) {
	if clusterClient == nil || mgr == nil {
		return nil, fmt.Errorf("cluster client and manager cannot be nil")
	}
	return &accountInfoRetriever{
		mgr:           mgr,
		clusterClient: clusterClient,
	}, nil
}

func (a *accountInfoRetriever) Get(ctx context.Context, accountPath string) (*accountsv1alpha1.AccountInfo, error) {
	log := logger.LoadLoggerFromContext(ctx)
	lc, err := a.clusterClient.Cluster(logicalcluster.NewPath(accountPath)).CoreV1alpha1().LogicalClusters().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msg("failed to get logical cluster from kcp")
		return nil, err
	}

	cluster, err := a.mgr.GetCluster(ctx, logicalcluster.From(lc).String())
	if err != nil { // coverage-ignore
		log.Error().Err(err).Msg("failed to get cluster from manager")
		return nil, err
	}
	cl := cluster.GetClient()

	ai := &accountsv1alpha1.AccountInfo{}
	err = cl.Get(ctx, client.ObjectKey{Name: "account"}, ai)
	if err != nil {
		log.Error().Err(err).Msg("failed to get orgs workspace from kcp")
		return nil, err
	}
	return ai, nil
}
