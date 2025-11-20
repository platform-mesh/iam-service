package workspace

import (
	"fmt"
	"net/url"

	"github.com/platform-mesh/golang-commons/logger"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

// ClientFactory creates a client for a specific KCP workspace
type ClientFactory interface {
	New(accountPath string) (client.Client, error)
}

// KCPClient implements ClientFactory for KCP workspaces
type KCPClient struct {
	mgr mcmanager.Manager
	log *logger.Logger
}

// NewClientFactory creates a new workspace client factory
func NewClientFactory(mgr mcmanager.Manager, log *logger.Logger) *KCPClient {
	return &KCPClient{
		mgr: mgr,
		log: log,
	}
}

// New creates a new client for the specified workspace path
func (f *KCPClient) New(accountPath string) (client.Client, error) {
	cfg := rest.CopyConfig(f.mgr.GetLocalManager().GetConfig())

	parsed, err := url.Parse(cfg.Host)
	if err != nil { // coverage-ignore
		log.Error().Err(err).Msg("unable to parse host")
		return nil, err
	}

	parsed.Path = fmt.Sprintf("/clusters/%s", accountPath)
	cfg.Host = parsed.String()

	cl, err := client.New(cfg, client.Options{Scheme: f.mgr.GetLocalManager().GetScheme()})
	if err != nil { // coverage-ignore
		log.Error().Err(err).Msg("unable to construct root client")
		return nil, err
	}
	return cl, nil
}
