package cost_estimator

import (
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
)

func GetEC2InstanceCost(h *HttpHandler, resourceId string, timeInterval int) (float64, error) {
	resource, err := es.GetEC2Instance(h.client, resourceId)
	if err != nil {
		return 0, err
	}
	cost, err := h.workspaceClient.GetEC2InstanceCost(&httpclient.Context{UserRole: apiAuth.InternalRole}, resource, timeInterval)
	if err != nil {
		return 0, err
	}

	return cost, nil
}
