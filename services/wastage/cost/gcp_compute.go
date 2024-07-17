package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/pennywise/pkg/cost"
	"github.com/kaytu-io/pennywise/pkg/schema"
	gcp "github.com/kaytu-io/plugin-gcp/plugin/proto/src/golang/gcp"
	"net/http"
	"time"
)

func (s *Service) GetGCPComputeInstanceCost(ctx context.Context, instance gcp.GcpComputeInstance) (float64, float64, error) {
	req := schema.Submission{
		ID:        "submission-1",
		CreatedAt: time.Now(),
		Resources: []schema.ResourceDef{},
	}

	valuesMap := map[string]any{}
	valuesMap["machine_type"] = instance.MachineType
	valuesMap["zone"] = instance.Zone

	purcharseOption := "on_demand"
	if instance.Preemptible {
		purcharseOption = "preemptible"
	}
	valuesMap["purchase_option"] = purcharseOption
	valuesMap["license"] = instance.InstanceOsLicense

	valuesMap["pennywise_usage"] = map[string]any{}

	req.Resources = append(req.Resources, schema.ResourceDef{
		Address:      instance.Id,
		Type:         "google_compute_instance",
		Name:         "",
		RegionCode:   instance.Zone,
		ProviderName: "google",
		Values:       valuesMap,
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return 0, 0, err
	}

	var response cost.State
	statusCode, err := httpclient.DoRequest(ctx, "GET", s.pennywiseBaseUrl+"/api/v1/cost/submission", nil, reqBody, &response)
	if err != nil {
		return 0, 0, err
	}

	if statusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("failed to get pennywise cost, status code = %d", statusCode)
	}

	resourceCost, err := response.Cost()
	if err != nil {
		return 0, 0, err
	}

	var licenseCost float64

	for _, resource := range response.Resources {
		for _, comps := range resource.Components {
			for _, comp := range comps {
				if comp.Name == "License Price" {
					licenseCost = comp.MonthlyQuantity.InexactFloat64() * comp.Rate.InexactFloat64()
				}
			}
		}
	}

	return resourceCost.Decimal.InexactFloat64(), licenseCost, nil
}

func (s *Service) GetGCPComputeDiskCost(ctx context.Context, disk gcp.GcpComputeDisk) (float64, error) {
	req := schema.Submission{
		ID:        "submission-1",
		CreatedAt: time.Now(),
		Resources: []schema.ResourceDef{},
	}

	valuesMap := map[string]any{}
	valuesMap["disk_type"] = disk.DiskType
	valuesMap["region"] = disk.Region
	if disk.DiskSize != nil {
		valuesMap["size"] = disk.DiskSize.Value
	}
	if disk.ProvisionedIops != nil {
		valuesMap["iops"] = disk.ProvisionedIops.Value
	}

	valuesMap["pennywise_usage"] = map[string]any{}

	req.Resources = append(req.Resources, schema.ResourceDef{
		Address:      disk.Id,
		Type:         "google_compute_disk",
		Name:         "",
		RegionCode:   disk.Region,
		ProviderName: "google",
		Values:       valuesMap,
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	var response cost.State
	statusCode, err := httpclient.DoRequest(ctx, "GET", s.pennywiseBaseUrl+"/api/v1/cost/submission", nil, reqBody, &response)
	if err != nil {
		return 0, err
	}

	if statusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get pennywise cost, status code = %d", statusCode)
	}

	resourceCost, err := response.Cost()
	if err != nil {
		return 0, err
	}

	return resourceCost.Decimal.InexactFloat64(), nil
}
