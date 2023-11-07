package cost_estimator

import (
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/calculator/aws"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
)

func GetEC2InstanceCost(h *HttpHandler, resourceId string, timeInterval int64) (float64, error) {
	resource, err := es.GetEC2Instance(h.client, resourceId)
	if err != nil {
		return 0, err
	}
	cost, err := aws.EC2InstanceCostByResource(h.db, resource, timeInterval)
	if err != nil {
		return 0, err
	}

	return cost, nil
}

func GetRDSInstanceCost(h *HttpHandler, resourceId string, timeInterval int64) (float64, error) {
	resource, err := es.GetRDSInstance(h.client, resourceId)
	if err != nil {
		return 0, err
	}

}
