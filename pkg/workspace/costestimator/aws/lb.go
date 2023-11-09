package aws

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

func LBCostByResource(db *db.Database, request api.GetLBCostRequest) (float64, error) {
	var cost float64
	var family string
	switch request.LBType {
	case "network":
		family = "Load Balancer-Network"
	case "gateway":
		family = "Load Balancer-Gateway"
	case "classic":
		family = "Load Balancer"
	default:
		family = "Load Balancer-Application"
	}
	lbPrice, err := db.FindLBPrice(request.RegionCode, family, "LoadBalancerUsage", "Hrs")
	if err != nil {
		return 0, err
	}
	cost += lbPrice.Price * costestimator.TimeInterval
	return cost, nil
}
