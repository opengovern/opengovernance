package onboard

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
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
