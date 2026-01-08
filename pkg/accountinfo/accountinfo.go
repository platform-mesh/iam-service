package accountinfo

import (
	"context"
	"fmt"
	"sync"

	"github.com/kcp-dev/logicalcluster/v3"
	"github.com/kcp-dev/sdk/apis/core/v1alpha1"
	kcpclientset "github.com/kcp-dev/sdk/client/clientset/versioned/cluster"
	accountsv1alpha1 "github.com/platform-mesh/account-operator/api/v1alpha1"
	"github.com/platform-mesh/account-operator/pkg/subroutines/accountinfo"
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
	clusterLocks  sync.Map
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
	lc, err := a.clusterClient.Cluster(logicalcluster.NewPath(accountPath)).CoreV1alpha1().LogicalClusters().Get(ctx, v1alpha1.LogicalClusterName, metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msg("failed to get logical cluster from kcp")
		return nil, err
	}
	clusterName := logicalcluster.From(lc).String()
	log = log.MustChildLoggerWithAttributes("cluster", clusterName)

	//FIXME: This lock was necessary as we saw race conditions when processing multiple requests in parallel
	// The issue occurs when a cluster is requested for the first time and multiple requests are processed simultaneously
	// We will work with the KCP team to identify the root cause and remove this lock in future
	mu := a.getClusterLock(clusterName)
	mu.Lock()
	defer mu.Unlock()

	cc, err := a.mgr.GetCluster(ctx, clusterName)
	if err != nil { // coverage-ignore
		log.Error().Err(err).Msg("failed to get cluster from manager")
		return nil, err
	}

	cl := cc.GetClient()
	ai := &accountsv1alpha1.AccountInfo{}
	err = cl.Get(ctx, client.ObjectKey{Name: accountinfo.DefaultAccountInfoName}, ai)
	if err != nil {
		log.Error().Err(err).Msg("failed to get orgs workspace from kcp")
		return nil, err
	}
	log.Debug().Msg("retrieved account info successfully")
	return ai, nil
}

func (a *accountInfoRetriever) getClusterLock(clusterName string) *sync.Mutex {
	lock, _ := a.clusterLocks.LoadOrStore(clusterName, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
