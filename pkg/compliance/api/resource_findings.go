package api

import (
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"time"
)

type ResourceFinding struct {
	KaytuResourceID   string      `json:"kaytuResourceID"`
	ResourceName      string      `json:"resourceName"`
	ResourceLocation  string      `json:"resourceLocation"`
	ResourceType      string      `json:"resourceType"`
	ResourceTypeLabel string      `json:"resourceTypeLabel"`
	Connector         source.Type `json:"connector"`

	FailedCount int `json:"failedCount"`
	TotalCount  int `json:"totalCount"`

	EvaluatedAt time.Time `json:"evaluatedAt"`

	Findings []Finding `json:"findings"`

	SortKey []any `json:"sortKey"`

	ConnectionID           string `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`           // Connection ID
	ProviderConnectionID   string `json:"providerConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`   // Connection ID
	ProviderConnectionName string `json:"providerConnectionName" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Connection ID
}

func GetAPIResourceFinding(resourceFinding types.ResourceFinding) ResourceFinding {
	apiRf := ResourceFinding{
		KaytuResourceID:  resourceFinding.KaytuResourceID,
		ResourceName:     resourceFinding.ResourceName,
		ResourceLocation: resourceFinding.ResourceLocation,
		ResourceType:     resourceFinding.ResourceType,
		Connector:        resourceFinding.Connector,

		FailedCount: 0,
		TotalCount:  len(resourceFinding.Findings),

		EvaluatedAt: time.UnixMilli(resourceFinding.EvaluatedAt),

		Findings: nil,
	}

	connectionIds := make(map[string]bool)

	for _, finding := range resourceFinding.Findings {
		if !finding.ConformanceStatus.IsPassed() {
			apiRf.FailedCount++
		}
		connectionIds[finding.ConnectionID] = true
		apiRf.ConnectionID = finding.ConnectionID
		f := Finding{
			BenchmarkID:           finding.BenchmarkID,
			ControlID:             finding.ControlID,
			ConnectionID:          finding.ConnectionID,
			EvaluatedAt:           finding.EvaluatedAt,
			StateActive:           finding.StateActive,
			ConformanceStatus:     "",
			Severity:              finding.Severity,
			Evaluator:             finding.Evaluator,
			Connector:             finding.Connector,
			KaytuResourceID:       finding.KaytuResourceID,
			ResourceID:            finding.ResourceID,
			ResourceName:          finding.ResourceName,
			ResourceLocation:      finding.ResourceLocation,
			ResourceType:          finding.ResourceType,
			Reason:                finding.Reason,
			ComplianceJobID:       finding.ComplianceJobID,
			ParentComplianceJobID: finding.ParentComplianceJobID,
			ParentBenchmarks:      finding.ParentBenchmarks,
		}
		if finding.ConformanceStatus.IsPassed() {
			f.ConformanceStatus = ConformanceStatusPassed
		} else {
			f.ConformanceStatus = ConformanceStatusFailed
		}
		apiRf.Findings = append(apiRf.Findings, f)
	}

	if len(connectionIds) > 1 {
		apiRf.ConnectionID = "Global (Multiple)"
		apiRf.ProviderConnectionID = "Global (Multiple)"
		apiRf.ProviderConnectionName = "Global (Multiple)"
	}

	return apiRf
}

type ResourceFindingsFilters struct {
	Connector          []source.Type           `json:"connector" example:"Azure"`
	ResourceID         []string                `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceTypeID     []string                `json:"resourceTypeID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines"`
	ConnectionID       []string                `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	ResourceCollection []string                `json:"resourceCollection" example:"example-rc"`
	BenchmarkID        []string                `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID          []string                `json:"controlID" example:"azure_cis_v140_7_5"`
	Severity           []types.FindingSeverity `json:"severity" example:"low"`
	ConformanceStatus  []ConformanceStatus     `json:"conformanceStatus" example:"alarm"`
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
