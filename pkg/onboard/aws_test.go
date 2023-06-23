package onboard

import (
	"context"
	"testing"

	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/stretchr/testify/require"
)

func TestDiscoverAwsAccounts(t *testing.T) {
	accounts, err := discoverAwsAccounts(context.Background(), api.DiscoverAWSAccountsRequest{
		AccessKey: "",
		SecretKey: "",
	})

	require.NoError(t, err)
	require.NotEmpty(t, accounts)
}
