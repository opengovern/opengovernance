package costestimator

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/cost"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/postgresql"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
	kaytuResources "github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/resources"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"go.uber.org/zap"
)

func CalcCosts(db *db.Database, logger *zap.Logger, provider string, resourceType string, request kaytuResources.ResourceRequest) (float64, error) {
	resource, err := kaytuResources.GetResource(provider, resourceType, request)
	if err != nil {
		return 0, nil
	}
	resources := []query.Resource{*resource}

	backend := postgresql.NewBackend(db)
	state, err := cost.NewState(backend, resources)
	if err != nil {
		logger.Error("Error while making cost state", zap.Error(err))
		return 0, err
	}
	cost, err := state.Cost()
	if err != nil {
		logger.Error("Error while calculating cost", zap.Error(err))
		return 0, err
	}
	return cost.Decimal.InexactFloat64(), nil
}
