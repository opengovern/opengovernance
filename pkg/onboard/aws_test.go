package onboard

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
)

func TestDiscoverAwsAccounts(t *testing.T) {
	t.Skip()
	accounts, err := discoverAwsAccounts(context.Background(), api.DiscoverAWSAccountsRequest{
		AccessKey: "",
		SecretKey: "",
	})

	require.NoError(t, err)
	require.NotEmpty(t, accounts)
}
