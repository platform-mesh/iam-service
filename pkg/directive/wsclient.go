package directive

import (
	"fmt"
	"net/url"

	"github.com/platform-mesh/golang-commons/logger"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

type WorkspaceClient interface {
	New(accountPath string) (client.Client, error)
}

type KCPClient struct {
	mgr mcmanager.Manager
	log *logger.Logger
}

func NewDefaultWSClientFactory(mgr mcmanager.Manager, log *logger.Logger) *KCPClient {
	return &KCPClient{
		mgr: mgr,
		log: log,
	}
}

func (f *KCPClient) New(accountPath string) (client.Client, error) {
	cfg := rest.CopyConfig(f.mgr.GetLocalManager().GetConfig())

	parsed, err := url.Parse(cfg.Host)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse host")
		return nil, err
	}

	parsed.Path = fmt.Sprintf("/clusters/%s", accountPath)
	cfg.Host = parsed.String()

	cl, err := client.New(cfg, client.Options{Scheme: f.mgr.GetLocalManager().GetScheme()})
	if err != nil {
		log.Error().Err(err).Msg("unable to construct root client")
		return nil, err
	}
	return cl, nil
}
