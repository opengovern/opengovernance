package kaytu_client

import (
	essdk "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	pgsdk "github.com/kaytu-io/open-governance/pkg/steampipe-plugin-kaytu/kaytu-sdk/pg"
)

type Client struct {
	ES          essdk.Client
	PG          pgsdk.Client
	PGInventory pgsdk.Client
}
