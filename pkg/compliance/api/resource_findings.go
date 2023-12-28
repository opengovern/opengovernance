package api

import (
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"time"
)

type ResourceFinding struct {
	KaytuResourceID  string `json:"kaytuResourceID"`
	ResourceName     string `json:"resourceName"`
	ResourceLocation string `json:"resourceLocation"`
	ResourceType     string `json:"resourceType"`

	FailedCount int `json:"failedCount"`
	TotalCount  int `json:"totalCount"`

	EvaluatedAt time.Time `json:"evaluatedAt"`

	Findings []Finding `json:"findings"`

	SortKey []any `json:"sortKey"`
}

func GetAPIResourceFinding(resourceFinding types.ResourceFinding) ResourceFinding {
	apiRf := ResourceFinding{
		KaytuResourceID:  resourceFinding.KaytuResourceID,
		ResourceName:     resourceFinding.ResourceName,
		ResourceLocation: resourceFinding.ResourceLocation,
		ResourceType:     resourceFinding.ResourceType,

		FailedCount: 0,
		TotalCount:  len(resourceFinding.Findings),

		EvaluatedAt: time.UnixMilli(resourceFinding.EvaluatedAt),

		Findings: nil,
	}

	for _, finding := range resourceFinding.Findings {
		if !finding.ConformanceStatus.IsPassed() {
			apiRf.FailedCount++
		}
		apiRf.Findings = append(apiRf.Findings, Finding{Finding: finding})
	}

	return apiRf
}

type ResourceFindingsFilters struct {
	Connector          []source.Type             `json:"connector" example:"Azure"`
	ResourceID         []string                  `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceTypeID     []string                  `json:"resourceTypeID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines"`
	ConnectionID       []string                  `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	ResourceCollection []string                  `json:"resourceCollection" example:"example-rc"`
	BenchmarkID        []string                  `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID          []string                  `json:"controlID" example:"azure_cis_v140_7_5"`
	Severity           []types.FindingSeverity   `json:"severity" example:"low"`
	ConformanceStatus  []types.ConformanceStatus `json:"conformanceStatus" example:"alarm"`
}

type ResourceFindingsSort struct {
	KaytuResourceID  *SortDirection `json:"kaytuResourceID"`
	ResourceType     *SortDirection `json:"resourceType"`
	ResourceName     *SortDirection `json:"resourceName"`
	ResourceLocation *SortDirection `json:"resourceLocation"`
	FailedCount      *SortDirection `json:"failedCount"`
}

type ListResourceFindingsRequest struct {
	Filters      ResourceFindingsFilters `json:"filters"`
	Sort         []ResourceFindingsSort  `json:"sort"`
	Limit        int                     `json:"limit" example:"100"`
	AfterSortKey []any                   `json:"afterSortKey"`
}

type ListResourceFindingsResponse struct {
	TotalCount       int               `json:"totalCount"`
	ResourceFindings []ResourceFinding `json:"resourceFindings"`
}
