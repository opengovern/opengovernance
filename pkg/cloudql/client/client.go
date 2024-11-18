package opengovernance_client

import (
	essdk "github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	pgsdk "github.com/opengovern/opengovernance/pkg/cloudql/sdk/pg"
)

type Client struct {
	ES essdk.Client
	PG pgsdk.Client
}
