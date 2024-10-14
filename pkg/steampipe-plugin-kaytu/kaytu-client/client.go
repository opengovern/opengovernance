package kaytu_client

import (
	essdk "github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	pgsdk "github.com/opengovern/opengovernance/pkg/steampipe-plugin-kaytu/kaytu-sdk/pg"
)

type Client struct {
	ES          essdk.Client
	PG          pgsdk.Client
	PGInventory pgsdk.Client
}
