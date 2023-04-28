package onboard

import (
	"context"
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
