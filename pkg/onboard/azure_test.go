package onboard

import (
	"context"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscoverAzureSubscriptions(t *testing.T) {
	subs, err := discoverAzureSubscriptions(context.Background(), azure.AuthConfig{
		TenantID:     "",
		ClientID:     "",
		ClientSecret: "",
	})
	require.NoError(t, err)
	require.NotEmpty(t, subs)
}
