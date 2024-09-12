package api

import (
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strings"
	"time"
)

type FindingFilters struct {
	Connector         []source.Type           `json:"connector" example:"Azure"`
	ResourceID        []string                `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceTypeID    []string                `json:"resourceTypeID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines"`
	ConnectionID      []string                `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	NotConnectionID   []string                `json:"notConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	BenchmarkID       []string                `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID         []string                `json:"controlID" example:"azure_cis_v140_7_5"`
	Severity          []types.FindingSeverity `json:"severity" example:"low"`
	ConformanceStatus []ConformanceStatus     `json:"conformanceStatus" example:"alarm"`
	StateActive       []bool                  `json:"stateActive" example:"true"`
	LastEvent         struct {
		From *int64 `json:"from"`
		To   *int64 `json:"to"`
	} `json:"lastEvent"`
	EvaluatedAt struct {
		From *int64 `json:"from"`
		To   *int64 `json:"to"`
	} `json:"evaluatedAt"`
}

type FindingSummaryFilters struct {
	ConnectionID   []string `json:"connectionID"`
	ResourceTypeID []string `json:"resourceTypeID"`
}

type FindingFiltersWithMetadata struct {
	Connector          []FilterWithMetadata `json:"connector"`
	BenchmarkID        []FilterWithMetadata `json:"benchmarkID"`
	ControlID          []FilterWithMetadata `json:"controlID"`
	ResourceTypeID     []FilterWithMetadata `json:"resourceTypeID"`
	ConnectionID       []FilterWithMetadata `json:"connectionID"`
	ResourceCollection []FilterWithMetadata `json:"resourceCollection"`
	Severity           []FilterWithMetadata `json:"severity"`
	ConformanceStatus  []FilterWithMetadata `json:"conformanceStatus"`
	StateActive        []FilterWithMetadata `json:"stateActive"`
}

type FindingsSort struct {
	Connector         *SortDirection `json:"connector"`
	ResourceID        *SortDirection `json:"resourceID"`
	KaytuResourceID   *SortDirection `json:"kaytuResourceID"`
	ResourceTypeID    *SortDirection `json:"resourceTypeID"`
	ConnectionID      *SortDirection `json:"connectionID"`
	BenchmarkID       *SortDirection `json:"benchmarkID"`
	ControlID         *SortDirection `json:"controlID"`
	Severity          *SortDirection `json:"severity"`
	ConformanceStatus *SortDirection `json:"conformanceStatus"`
	StateActive       *SortDirection `json:"stateActive"`
}

type GetFindingsRequest struct {
	Filters      FindingFilters `json:"filters"`
	Sort         []FindingsSort `json:"sort"`
	Limit        int            `json:"limit" example:"100"`
	AfterSortKey []any          `json:"afterSortKey"`
}

type GetSingleFindingRequest struct {
	BenchmarkID     string `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID       string `json:"controlID" example:"azure_cis_v140_7_5"`
	ConnectionID    string `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	KaytuResourceID string `json:"kaytuResourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceID      string `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
}

type GetSingleResourceFindingRequest struct {
	KaytuResourceId string  `json:"kaytuResourceId" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceType    *string `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
}

type ConformanceStatus string

const (
	ConformanceStatusFailed ConformanceStatus = "failed"
	ConformanceStatusPassed ConformanceStatus = "passed"
)

func ListConformanceStatuses() []ConformanceStatus {
	return []ConformanceStatus{ConformanceStatusFailed, ConformanceStatusPassed}
}

func (cs ConformanceStatus) GetEsConformanceStatuses() []types.ConformanceStatus {
	switch cs {
	case ConformanceStatusFailed:
		return types.GetFailedConformanceStatuses()
	case ConformanceStatusPassed:
		return types.GetPassedConformanceStatuses()
	}
	return nil
}

func ParseConformanceStatuses(conformanceStatuses []string) []ConformanceStatus {
	var result []ConformanceStatus
	for _, cs := range conformanceStatuses {
		switch strings.ToLower(cs) {
		case strings.ToLower(string(ConformanceStatusFailed)):
			result = append(result, ConformanceStatusFailed)
		case strings.ToLower(string(ConformanceStatusPassed)):
			result = append(result, ConformanceStatusPassed)
		}
	}
	return result
}

type Finding struct {
	ID                        string                `json:"id" example:"1"`
	BenchmarkID               string                `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID                 string                `json:"controlID" example:"azure_cis_v140_7_5"`
	ConnectionID              string                `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	EvaluatedAt               int64                 `json:"evaluatedAt" example:"1589395200"`
	StateActive               bool                  `json:"stateActive" example:"true"`
	ConformanceStatus         ConformanceStatus     `json:"conformanceStatus" example:"alarm"`
	Severity                  types.FindingSeverity `json:"severity" example:"low"`
	Evaluator                 string                `json:"evaluator" example:"steampipe-v0.5"`
	Connector                 source.Type           `json:"connector" example:"Azure"`
	KaytuResourceID           string                `json:"kaytuResourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceID                string                `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceName              string                `json:"resourceName" example:"vm-1"`
	ResourceLocation          string                `json:"resourceLocation" example:"eastus"`
	ResourceType              string                `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
	Reason                    string                `json:"reason" example:"The VM is not using managed disks"`
	CostOptimization          *float64              `json:"costOptimization" example:"0.5"`
	ComplianceJobID           uint                  `json:"complianceJobID" example:"1"`
	ParentComplianceJobID     uint                  `json:"parentComplianceJobID" example:"1"`
	ParentBenchmarkReferences []string              `json:"parentBenchmarkReferences"`
	ParentBenchmarks          []string              `json:"parentBenchmarks"`
	LastEvent                 time.Time             `json:"lastEvent" example:"1589395200"`

	ResourceTypeName       string   `json:"resourceTypeName" example:"Virtual Machine"`
	ParentBenchmarkNames   []string `json:"parentBenchmarkNames" example:"Azure CIS v1.4.0"`
	ControlTitle           string   `json:"controlTitle"`
	ProviderConnectionID   string   `json:"providerConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`   // Connection ID
	ProviderConnectionName string   `json:"providerConnectionName" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Connection ID

	SortKey []any `json:"sortKey"`
}

func GetAPIFindingFromESFinding(finding types.Finding) Finding {
	f := Finding{
		ID:                        finding.EsID,
		BenchmarkID:               finding.BenchmarkID,
		ControlID:                 finding.ControlID,
		ConnectionID:              finding.ConnectionID,
		EvaluatedAt:               finding.EvaluatedAt,
		StateActive:               finding.StateActive,
		ConformanceStatus:         "",
		Severity:                  finding.Severity,
		Evaluator:                 finding.Evaluator,
		Connector:                 finding.Connector,
		KaytuResourceID:           finding.KaytuResourceID,
		ResourceID:                finding.ResourceID,
		ResourceName:              finding.ResourceName,
		ResourceLocation:          finding.ResourceLocation,
		ResourceType:              finding.ResourceType,
		Reason:                    finding.Reason,
		CostOptimization:          finding.CostOptimization,
		ComplianceJobID:           finding.ComplianceJobID,
		ParentBenchmarkReferences: finding.ParentBenchmarkReferences,
		ParentComplianceJobID:     finding.ParentComplianceJobID,
		ParentBenchmarks:          finding.ParentBenchmarks,
		LastEvent:                 time.UnixMilli(finding.LastTransition),
	}
	if finding.ConformanceStatus.IsPassed() {
		f.ConformanceStatus = ConformanceStatusPassed
	} else {
		f.ConformanceStatus = ConformanceStatusFailed
	}
	if f.ResourceType == "" {
		f.ResourceType = "Unknown"
		f.ResourceTypeName = "Unknown"
	}

	return f
}

type GetFindingsResponse struct {
	Findings   []Finding `json:"findings"`
	TotalCount int64     `json:"totalCount" example:"100"`
}

type FindingKPIResponse struct {
	FailedFindingsCount   int64 `json:"failedFindingsCount"`
	FailedResourceCount   int64 `json:"failedResourceCount"`
	FailedControlCount    int64 `json:"failedControlCount"`
	FailedConnectionCount int64 `json:"failedConnectionCount"`
}

type ServiceFindingsSummary struct {
	ServiceName     string  `json:"serviceName"`
	ServiceLabel    string  `json:"serviceLabel"`
	SecurityScore   float64 `json:"securityScore"`
	SeveritiesCount struct {
		Critical int `json:"critical"`
		High     int `json:"high"`
		Medium   int `json:"medium"`
		Low      int `json:"low"`
		None     int `json:"none"`
	} `json:"severitiesCount"`
	ConformanceStatusesCount struct {
		Passed int `json:"passed"`
		Failed int `json:"failed"`
	} `json:"conformanceStatusesCount"`
}

type GetServicesFindingsSummaryResponse struct {
	Services []ServiceFindingsSummary `json:"services"`
}

type GetFieldCountResponse struct {
	Controls []struct {
		ControlName string           `json:"controlName"`
		FieldCounts []TopFieldRecord `json:"fieldCounts"`
	} `json:"controls"`
}

type GetTopFieldResponse struct {
	TotalCount int              `json:"totalCount" example:"100"`
	Records    []TopFieldRecord `json:"records"`
}

type CountFindingsResponse struct {
	Count int64 `json:"count"`
}

type GetFindingEventsByFindingIDResponse struct {
	FindingEvents []FindingEvent `json:"findingEvents"`
}

type FindingFiltersV2 struct {
	Integration     []IntegrationInfoFilter `json:"integration"`
	BenchmarkID     []string                `json:"benchmark_id"`
	NotBenchmarkID  []string                `json:"not_benchmark_id"`
	ControlID       []string                `json:"control_id"`
	NotControlID    []string                `json:"not_control_id"`
	Severity        []types.FindingSeverity `json:"severity"`
	NotSeverity     []types.FindingSeverity `json:"not_severity"`
	ResourceID      []string                `json:"resource_id"`
	NotResourceID   []string                `json:"not_resource_id"`
	IsCompliant     []bool                  `json:"is_compliant"`
	IsActive        []bool                  `json:"is_active"`
	ResourceType    []string                `json:"resource_type"`
	NotResourceType []string                `json:"not_resource_type"`
	LastUpdated     struct {
		From *int64 `json:"from"`
		To   *int64 `json:"to"`
	} `json:"last_event"`
	NotLastUpdated struct {
		From *int64 `json:"from"`
		To   *int64 `json:"to"`
	} `json:"not_last_event"`
}

type IntegrationInfoFilter struct {
	Integration        *string `json:"integration"`
	ID                 *string `json:"id"`
	IDName             *string `json:"id_name"`
	IntegrationTracker *string `json:"integration_tracker"`
}

type FindingsSortV2 struct {
	BenchmarkID       *SortDirection `json:"benchmark_id"`
	ControlID         *SortDirection `json:"control_id"`
	Severity          *SortDirection `json:"severity"`
	ConformanceStatus *SortDirection `json:"conformance_status"`
	ResourceType      *SortDirection `json:"resource_type"`
	LastUpdated       *SortDirection `json:"last_updated"`
}

type GetFindingsRequestV2 struct {
	Filters      FindingFiltersV2 `json:"filters"`
	Sort         []FindingsSortV2 `json:"sort"`
	Limit        int              `json:"limit" example:"100"`
	AfterSortKey []any            `json:"afterSortKey"`
}
