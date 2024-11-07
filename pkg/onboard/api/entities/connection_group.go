package entities

import (
	"context"
	"github.com/opengovern/og-util/pkg/steampipe"
	"github.com/opengovern/opengovernance/pkg/onboard/api"
)

func NewConnectionGroup(ctx context.Context, steampipe *steampipe.Database, cg any) (*api.ConnectionGroup, error) {
	apiCg := api.ConnectionGroup{}

	return &apiCg, nil
}
