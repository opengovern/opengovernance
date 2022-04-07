package onboard

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
)

func TestDiscoverAzureSubscriptions(t *testing.T) {
	t.Skip()
	subs, err := discoverAzureSubscriptions(context.Background(), api.DiscoverAzureSubscriptionsRequest{
		TenantId:     "",
		ClientId:     "",
		ClientSecret: "",
	})
	require.NoError(t, err)
	require.NotEmpty(t, subs)
}
