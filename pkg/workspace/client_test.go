package workspace

import (
	"context"
	"fmt"
	"testing"

	"github.com/platform-mesh/golang-commons/logger"
	fgamocks "github.com/platform-mesh/iam-service/pkg/fga/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKCPClient_New_Success(t *testing.T) {
	tests := []struct {
		name        string
		accountPath string
	}{
		{
			name:        "valid account path",
			accountPath: "root:org:account",
		},
		{
			name:        "account path with special characters",
			accountPath: "root:my-org:my-account-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			expectedClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			mockFactory := fgamocks.NewClientFactory(t)
			mockFactory.On("New", mock.Anything, tt.accountPath).Return(expectedClient, nil)

			log, err := logger.New(logger.Config{Level: "info"})
			require.NoError(t, err)
			ctx := logger.SetLoggerInContext(context.Background(), log)

			result, err := mockFactory.New(ctx, tt.accountPath)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, expectedClient, result)
		})
	}
}

func TestKCPClient_New_Error(t *testing.T) {
	accountPath := "root:nonexistent:account"
	expectedErr := fmt.Errorf("cluster not found: %s", accountPath)

	mockFactory := fgamocks.NewClientFactory(t)
	mockFactory.On("New", mock.Anything, accountPath).Return(nil, expectedErr)

	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)
	ctx := logger.SetLoggerInContext(context.Background(), log)

	result, err := mockFactory.New(ctx, accountPath)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cluster not found")
}

func TestKCPClient_New_MultipleClients(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeClient1 := fake.NewClientBuilder().WithScheme(scheme).Build()
	fakeClient2 := fake.NewClientBuilder().WithScheme(scheme).Build()

	mockFactory := fgamocks.NewClientFactory(t)
	mockFactory.On("New", mock.Anything, "root:org1:account1").Return(fakeClient1, nil)
	mockFactory.On("New", mock.Anything, "root:org2:account2").Return(fakeClient2, nil)

	log, err := logger.New(logger.Config{Level: "info"})
	require.NoError(t, err)
	ctx := logger.SetLoggerInContext(context.Background(), log)

	client1, err := mockFactory.New(ctx, "root:org1:account1")
	require.NoError(t, err)
	require.NotNil(t, client1)

	client2, err := mockFactory.New(ctx, "root:org2:account2")
	require.NoError(t, err)
	require.NotNil(t, client2)

	// Verify that both clients were created successfully and are different
	assert.NotEqual(t, client1, client2)
	assert.Equal(t, fakeClient1, client1)
	assert.Equal(t, fakeClient2, client2)
}
