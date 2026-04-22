package workspace

import (
	"context"
	"fmt"

	"github.com/platform-mesh/golang-commons/logger"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

// ClientFactory creates a client for a specific KCP workspace
type ClientFactory interface {
	New(ctx context.Context, accountPath string) (client.Client, error)
}

// KCPClient implements ClientFactory for KCP workspaces
type KCPClient struct {
	mgr mcmanager.Manager
}

// NewClientFactory creates a new workspace client factory
func NewClientFactory(mgr mcmanager.Manager) *KCPClient {
	return &KCPClient{
		mgr: mgr,
	}
}

// New creates a new client for the specified workspace path
func (f *KCPClient) New(ctx context.Context, accountPath string) (client.Client, error) {
	log := logger.LoadLoggerFromContext(ctx)
	cluster, err := f.mgr.GetCluster(ctx, accountPath)
	if err != nil {
		log.Err(err).Msg(fmt.Sprintf("failed to get cluster: %s", accountPath))
		return nil, err
	}

	return cluster.GetClient(), nil
}
