package entities

import (
	"context"
	"github.com/opengovern/og-util/pkg/steampipe"
	"github.com/opengovern/opengovernance/pkg/onboard/api"
	"github.com/opengovern/opengovernance/services/integration/model"
)

func NewConnectionGroup(ctx context.Context, steampipe *steampipe.Database, cg model.ConnectionGroup) (*api.ConnectionGroup, error) {
	apiCg := api.ConnectionGroup{
		Name:  cg.Name,
		Query: cg.Query,
	}

	if steampipe == nil || cg.Query == "" {
		return &apiCg, nil
	}

	connectionsQueryResult, err := steampipe.QueryAll(ctx, cg.Query)
	if err != nil {
		return nil, err
	}

	var connectionIds []string
	for i, header := range connectionsQueryResult.Headers {
		if header != "kaytu_id" {
			continue
		}
		for _, row := range connectionsQueryResult.Data {
			if len(row) <= i || row[i] == nil {
				continue
			}
			if strRow, ok := row[i].(string); ok {
				connectionIds = append(connectionIds, strRow)
			}
		}
	}

	apiCg.ConnectionIds = connectionIds

	return &apiCg, nil
}
