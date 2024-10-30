package api

import (
	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/opengovernance/pkg/types"
	"time"
)

type ResourceFinding struct {
	ID                       string      `json:"id"`
	OpenGovernanceResourceID string      `json:"opengovernanceResourceID"`
	ResourceName             string      `json:"resourceName"`
	ResourceLocation         string      `json:"resourceLocation"`
	ResourceType             string      `json:"resourceType"`
	ResourceTypeLabel        string      `json:"resourceTypeLabel"`
	Connector                source.Type `json:"connector"`
	ComplianceJobID          string      `json:"complianceJobID"`

	FailedCount int `json:"failedCount"`
	TotalCount  int `json:"totalCount"`

	EvaluatedAt time.Time `json:"evaluatedAt"`

	Findings []Finding `json:"findings"`

	SortKey []any `json:"sortKey"`

	ConnectionID         string `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`    // Connection ID
	ProviderConnectionID string `json:"providerID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`      // Connection ID
	IntegrationName      string `json:"integrationName" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Connection ID
}

func GetAPIResourceFinding(resourceFinding types.ResourceFinding) ResourceFinding {
	apiRf := ResourceFinding{
		ID:                       resourceFinding.EsID,
		OpenGovernanceResourceID: resourceFinding.OpenGovernanceResourceID,
		ResourceName:             resourceFinding.ResourceName,
		ResourceLocation:         resourceFinding.ResourceLocation,
		ResourceType:             resourceFinding.ResourceType,
		Connector:                resourceFinding.Connector,

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
		apiRf.Findings = append(apiRf.Findings, GetAPIFindingFromESFinding(finding))
	}

	if len(connectionIds) > 1 {
		apiRf.ConnectionID = "Global (Multiple)"
		apiRf.ProviderConnectionID = "Global (Multiple)"
		apiRf.IntegrationName = "Global (Multiple)"
	}

	return apiRf
}

type ResourceFindingFilters struct {
	ComplianceJobId    []string                `json:"compliance_job_id"`
	Connector          []source.Type           `json:"connector" example:"Azure"`
	ResourceID         []string                `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceTypeID     []string                `json:"resourceTypeID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines"`
	ConnectionID       []string                `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	NotConnectionID    []string                `json:"notConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	ConnectionGroup    []string                `json:"connectionGroup" example:"healthy"`
	ResourceCollection []string                `json:"resourceCollection" example:"example-rc"`
	BenchmarkID        []string                `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID          []string                `json:"controlID" example:"azure_cis_v140_7_5"`
	Severity           []types.FindingSeverity `json:"severity" example:"low"`
	ConformanceStatus  []ConformanceStatus     `json:"conformanceStatus" example:"alarm"`
	EvaluatedAt        struct {
		From *int64 `json:"from"`
		To   *int64 `json:"to"`
	}
	Interval *string `json:"interval"`
}

type ResourceFindingsSort struct {
	OpenGovernanceResourceID *SortDirection `json:"opengovernanceResourceID"`
	ResourceType             *SortDirection `json:"resourceType"`
	ResourceName             *SortDirection `json:"resourceName"`
	ResourceLocation         *SortDirection `json:"resourceLocation"`
	FailedCount              *SortDirection `json:"failedCount"`
	ConformanceStatus        *SortDirection `json:"conformanceStatus"`
}

type ListResourceFindingsRequest struct {
	Filters      ResourceFindingFilters `json:"filters"`
	Sort         []ResourceFindingsSort `json:"sort"`
	Limit        int                    `json:"limit" example:"100"`
	AfterSortKey []any                  `json:"afterSortKey"`
}

type ListResourceFindingsResponse struct {
	TotalCount       int               `json:"totalCount"`
	ResourceFindings []ResourceFinding `json:"resourceFindings"`
}
