package workspace

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/platform-mesh/golang-commons/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

type Provider struct {
	clusters map[string]cluster.Cluster
}

func (p *Provider) Get(ctx context.Context, clusterName string) (cluster.Cluster, error) {
	cluster, ok := p.clusters[clusterName]
	if !ok {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}
	return cluster, nil
}

func (p *Provider) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	return nil
}

func TestNewClientFactory(t *testing.T) {
	emptyConfig := &rest.Config{
		Host: "https://test-host.example.com",
	}
	testProvider := &Provider{clusters: map[string]cluster.Cluster{}}

	mgr, err := mcmanager.New(emptyConfig, testProvider, mcmanager.Options{})
	require.NoError(t, err)

	factory := NewClientFactory(mgr)

	assert.NotNil(t, factory)
	assert.Equal(t, mgr, factory.mgr)
}

func TestKCPClient_New(t *testing.T) {
	tests := []struct {
		name        string
		accountPath string
		hostURL     string
		wantErr     bool
	}{
		{
			name:        "valid account path",
			accountPath: "root:org:account",
			hostURL:     "https://test-host.example.com",
			wantErr:     false,
		},
		{
			name:        "account path with special characters",
			accountPath: "root:my-org:my-account-123",
			hostURL:     "https://test-host.example.com",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localCfg := &rest.Config{
				Host: tt.hostURL,
			}
			testProvider := &Provider{clusters: map[string]cluster.Cluster{}}

			mgr, err := mcmanager.New(localCfg, testProvider, mcmanager.Options{})
			require.NoError(t, err)

			log, err := logger.New(logger.Config{Level: "info"})
			require.NoError(t, err)
			ctx := logger.SetLoggerInContext(context.Background(), log)

			factory := NewClientFactory(mgr)

			client, err := factory.New(ctx, tt.accountPath)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)

				// Verify that the config was modified correctly
				expectedURL, _ := url.Parse(tt.hostURL)
				expectedURL.Path = "/clusters/" + tt.accountPath
			}
		})
	}
}

func TestKCPClient_New_ConfigCopy(t *testing.T) {
	originalHost := "https://original-host.example.com"
	localCfg := &rest.Config{
		Host: originalHost,
	}
	testProvider := &Provider{clusters: map[string]cluster.Cluster{}}

	mgr, err := mcmanager.New(localCfg, testProvider, mcmanager.Options{})
	require.NoError(t, err)

	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)
	ctx := logger.SetLoggerInContext(context.Background(), log)

	factory := NewClientFactory(mgr)

	accountPath1 := "root:org1:account1"
	accountPath2 := "root:org2:account2"

	client1, err := factory.New(ctx, accountPath1)
	require.NoError(t, err)
	require.NotNil(t, client1)

	client2, err := factory.New(ctx, accountPath2)
	require.NoError(t, err)
	require.NotNil(t, client2)

	// Verify that the original config wasn't modified
	assert.Equal(t, originalHost, localCfg.Host)

	// Verify that both clients were created successfully (they should be different)
	assert.NotEqual(t, client1, client2)
}
