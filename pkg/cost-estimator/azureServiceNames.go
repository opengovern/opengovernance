package cost_estimator

import (
	"context"
	_ "encoding/json"
	azure "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	"github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
	essdk "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

// ======================== ComputeVirtualMachine ================================

type ComputeVirtualMachine struct {
	Description   azure.ComputeVirtualMachineDescription `json:"description"`
	Metadata      azure.Metadata                         `json:"metadata"`
	ResourceJobID int                                    `json:"resource_job_id"`
	SourceJobID   int                                    `json:"source_job_id"`
	ResourceType  string                                 `json:"resource_type"`
	SourceType    string                                 `json:"source_type"`
	ID            string                                 `json:"id"`
	ARN           string                                 `json:"arn"`
	SourceID      string                                 `json:"source_id"`
}

var GetComputeVirtualMachineFiltering = map[string]string{
	"id":   "description.VirtualMachine.ID",
	"type": "description.VirtualMachine.Type",
}

type ComputeVirtualMachinePaginator struct {
	paginator *essdk.BaseESPaginator
}

func (p ComputeVirtualMachinePaginator) HasNext() bool {
	return !p.paginator.Done()
}

func (p ComputeVirtualMachinePaginator) NextPage(ctx context.Context) ([]kaytu.ComputeVirtualMachine, error) {
	var response kaytu.ComputeVirtualMachineSearchResponse
	err := p.paginator.Search(ctx, &response)
	if err != nil {
		return nil, err
	}

	var values []kaytu.ComputeVirtualMachine
	for _, hit := range response.Hits.Hits {
		values = append(values, hit.Source)
	}

	hits := int64(len(response.Hits.Hits))
	if hits > 0 {
		p.paginator.UpdateState(hits, response.Hits.Hits[hits-1].Sort, response.PitID)
	} else {
		p.paginator.UpdateState(hits, nil, "")
	}

	return values, nil
}

func (h HttpHandler) GetComputeVirtualMachineResource(ctx context.Context) (string, string, string, error) {
	var arrayBoolFilter []essdk.BoolFilter
	for k, v := range GetComputeVirtualMachineFiltering {
		boolFilter := essdk.NewTermFilter(k, v)
		arrayBoolFilter = append(arrayBoolFilter, boolFilter)
	}

	p, err := essdk.NewPaginator(h.client.ES(), "microsoft_compute_virtualmachines", arrayBoolFilter, nil)
	if err != nil {
		return "", "", "", err
	}

	paginator := ComputeVirtualMachinePaginator{
		paginator: p,
	}

	if paginator.HasNext() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return "", "", "", err
		}

		for _, v := range page {
			location := v.Description.VirtualMachine.Location
			vmSize := v.Description.VirtualMachine.Properties.HardwareProfile.VMSize
			osType := v.Description.VirtualMachine.Properties.StorageProfile.OSDisk.OSType
			return *location, string(*vmSize), string(*osType), nil
		}
	}
	return "", "", "", nil
}

// ====================== VirtualNetwork ===========================

type VirtualNetwork struct {
	Description   azure.VirtualNetworkDescription `json:"description"`
	Metadata      azure.Metadata                  `json:"metadata"`
	ResourceJobID int                             `json:"resource_job_id"`
	SourceJobID   int                             `json:"source_job_id"`
	ResourceType  string                          `json:"resource_type"`
	SourceType    string                          `json:"source_type"`
	ID            string                          `json:"id"`
	ARN           string                          `json:"arn"`
	SourceID      string                          `json:"source_id"`
}

var getVirtualNetworkFilters = map[string]string{
	"address_prefixes":       "description.VirtualNetwork.Properties.AddressSpace.AddressPrefixes",
	"enable_ddos_protection": "description.VirtualNetwork.Properties.EnableDdosProtection",
	"enable_vm_protection":   "description.VirtualNetwork.Properties.EnableVMProtection",
	"etag":                   "description.VirtualNetwork.Etag",
	"id":                     "description.VirtualNetwork.ID",
	"kaytu_account_id":       "metadata.SourceID",
	"name":                   "description.VirtualNetwork.name",
	"network_peerings":       "description.VirtualNetwork.Properties.VirtualNetworkPeerings",
	"provisioning_state":     "description.VirtualNetwork.Properties.ProvisioningState",
	"resource_group":         "description.ResourceGroup",
	"resource_guid":          "description.VirtualNetwork.Properties.ResourceGUID",
	"subnets":                "description.VirtualNetwork.Properties.Subnets",
	"tags":                   "description.VirtualNetwork.Tags",
	"title":                  "description.VirtualNetwork.Name",
	"type":                   "description.VirtualNetwork.Type",
}

type VirtualNetworkPaginator struct {
	paginator *essdk.BaseESPaginator
}

func (p VirtualNetworkPaginator) HasNext() bool {
	return !p.paginator.Done()
}

func (p VirtualNetworkPaginator) NextPage(ctx context.Context) ([]kaytu.VirtualNetwork, error) {
	var response kaytu.VirtualNetworkSearchResponse
	err := p.paginator.Search(ctx, &response)
	if err != nil {
		return nil, err
	}

	var values []kaytu.VirtualNetwork
	for _, hit := range response.Hits.Hits {
		values = append(values, hit.Source)
	}

	hits := int64(len(response.Hits.Hits))
	if hits > 0 {
		p.paginator.UpdateState(hits, response.Hits.Hits[hits-1].Sort, response.PitID)
	} else {
		p.paginator.UpdateState(hits, nil, "")
	}

	return values, nil
}

func (h HttpHandler) GetVirtualNetworkResource(ctx context.Context) (string, string, string, error) {
	var arrayBoolFilter []essdk.BoolFilter
	for k, v := range GetComputeVirtualMachineFiltering {
		boolFilter := essdk.NewTermFilter(k, v)
		arrayBoolFilter = append(arrayBoolFilter, boolFilter)
	}

	p, err := essdk.NewPaginator(h.client.ES(), "microsoft_network_virtualnetworks", arrayBoolFilter, nil)
	if err != nil {
		return "", "", "", err
	}

	paginator := VirtualNetworkPaginator{
		paginator: p,
	}

	for paginator.HasNext() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return "", "", "", err
		}
		for _, v := range page {
			return v, nil
		}
	}
	return "", "", "", nil
}
