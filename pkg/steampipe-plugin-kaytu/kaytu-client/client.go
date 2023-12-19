package kaytu_client

import (
	pgsdk "github.com/kaytu-io/kaytu-engine/pkg/steampipe-plugin-kaytu/kaytu-sdk/pg"
	essdk "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

type Client struct {
	ES          essdk.Client
	PG          pgsdk.Client
	PGInventory pgsdk.Client
}
