package api

import (
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/opengovernance/pkg/types"
	"strings"
	"time"
)

type ComplianceResultFilters struct {
	JobID             []string                         `json:"jobID"`
	IntegrationType   []string                         `json:"integrationType" example:"Azure"`
	ResourceID        []string                         `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceTypeID    []string                         `json:"resourceTypeID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines"`
	ConnectionID      []string                         `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	NotConnectionID   []string                         `json:"notConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	ConnectionGroup   []string                         `json:"connectionGroup" example:"healthy"`
	BenchmarkID       []string                         `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID         []string                         `json:"controlID" example:"azure_cis_v140_7_5"`
	Severity          []types.ComplianceResultSeverity `json:"severity" example:"low"`
	ConformanceStatus []ConformanceStatus              `json:"conformanceStatus" example:"alarm"`
	StateActive       []bool                           `json:"stateActive" example:"true"`
	LastEvent         struct {
		From *int64 `json:"from"`
		To   *int64 `json:"to"`
	} `json:"lastEvent"`
	EvaluatedAt struct {
		From *int64 `json:"from"`
		To   *int64 `json:"to"`
	} `json:"evaluatedAt"`
	Interval *string `json:"interval" example:"5m"`
}

type ComplianceResultSummaryFilters struct {
	ConnectionID   []string `json:"connectionID"`
	ResourceTypeID []string `json:"resourceTypeID"`
}

type ComplianceResultFiltersWithMetadata struct {
	IntegrationType    []FilterWithMetadata `json:"integrationType"`
	BenchmarkID        []FilterWithMetadata `json:"benchmarkID"`
	ControlID          []FilterWithMetadata `json:"controlID"`
	ResourceTypeID     []FilterWithMetadata `json:"resourceTypeID"`
	ConnectionID       []FilterWithMetadata `json:"connectionID"`
	ResourceCollection []FilterWithMetadata `json:"resourceCollection"`
	Severity           []FilterWithMetadata `json:"severity"`
	ConformanceStatus  []FilterWithMetadata `json:"conformanceStatus"`
	StateActive        []FilterWithMetadata `json:"stateActive"`
}

type ComplianceResultsSort struct {
	IntegrationType          *SortDirection `json:"integrationType"`
	ResourceID               *SortDirection `json:"resourceID"`
	OpenGovernanceResourceID *SortDirection `json:"opengovernanceResourceID"`
	ResourceTypeID           *SortDirection `json:"resourceTypeID"`
	ConnectionID             *SortDirection `json:"connectionID"`
	BenchmarkID              *SortDirection `json:"benchmarkID"`
	ControlID                *SortDirection `json:"controlID"`
	Severity                 *SortDirection `json:"severity"`
	ConformanceStatus        *SortDirection `json:"conformanceStatus"`
	StateActive              *SortDirection `json:"stateActive"`
}

type GetComplianceResultsRequest struct {
	Filters      ComplianceResultFilters `json:"filters"`
	Sort         []ComplianceResultsSort `json:"sort"`
	Limit        int                     `json:"limit" example:"100"`
	AfterSortKey []any                   `json:"afterSortKey"`
}

type GetSingleComplianceResultRequest struct {
	BenchmarkID              string `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID                string `json:"controlID" example:"azure_cis_v140_7_5"`
	ConnectionID             string `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	OpenGovernanceResourceID string `json:"opengovernanceResourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceID               string `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
}

type GetSingleResourceFindingRequest struct {
	OpenGovernanceResourceId string  `json:"opengovernanceResourceId" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceType             *string `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
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

type ComplianceResult struct {
	ID                        string                         `json:"id" example:"1"`
	BenchmarkID               string                         `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID                 string                         `json:"controlID" example:"azure_cis_v140_7_5"`
	ConnectionID              string                         `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	EvaluatedAt               int64                          `json:"evaluatedAt" example:"1589395200"`
	StateActive               bool                           `json:"stateActive" example:"true"`
	ConformanceStatus         ConformanceStatus              `json:"conformanceStatus" example:"alarm"`
	Severity                  types.ComplianceResultSeverity `json:"severity" example:"low"`
	Evaluator                 string                         `json:"evaluator" example:"steampipe-v0.5"`
	IntegrationType           integration.Type               `json:"integrationType" example:"Azure"`
	OpenGovernanceResourceID  string                         `json:"opengovernanceResourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceID                string                         `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceName              string                         `json:"resourceName" example:"vm-1"`
	ResourceLocation          string                         `json:"resourceLocation" example:"eastus"`
	ResourceType              string                         `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
	Reason                    string                         `json:"reason" example:"The VM is not using managed disks"`
	CostOptimization          *float64                       `json:"costOptimization" example:"0.5"`
	ComplianceJobID           uint                           `json:"complianceJobID" example:"1"`
	ParentComplianceJobID     uint                           `json:"parentComplianceJobID" example:"1"`
	ParentBenchmarkReferences []string                       `json:"parentBenchmarkReferences"`
	ParentBenchmarks          []string                       `json:"parentBenchmarks"`
	LastEvent                 time.Time                      `json:"lastEvent" example:"1589395200"`

	ResourceTypeName     string   `json:"resourceTypeName" example:"Virtual Machine"`
	ParentBenchmarkNames []string `json:"parentBenchmarkNames" example:"Azure CIS v1.4.0"`
	ControlTitle         string   `json:"controlTitle"`
	ProviderID           string   `json:"providerID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`      // Connection ID
	IntegrationName      string   `json:"integrationName" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Connection ID

	SortKey []any `json:"sortKey"`
}

func GetAPIComplianceResultFromESComplianceResult(complianceResult types.ComplianceResult) ComplianceResult {
	f := ComplianceResult{
		ID:                        complianceResult.EsID,
		BenchmarkID:               complianceResult.BenchmarkID,
		ControlID:                 complianceResult.ControlID,
		ConnectionID:              complianceResult.ConnectionID,
		EvaluatedAt:               complianceResult.EvaluatedAt,
		StateActive:               complianceResult.StateActive,
		ConformanceStatus:         "",
		Severity:                  complianceResult.Severity,
		Evaluator:                 complianceResult.Evaluator,
		IntegrationType:           complianceResult.IntegrationType,
		OpenGovernanceResourceID:  complianceResult.OpenGovernanceResourceID,
		ResourceID:                complianceResult.ResourceID,
		ResourceName:              complianceResult.ResourceName,
		ResourceLocation:          complianceResult.ResourceLocation,
		ResourceType:              complianceResult.ResourceType,
		Reason:                    complianceResult.Reason,
		CostOptimization:          complianceResult.CostOptimization,
		ComplianceJobID:           complianceResult.ComplianceJobID,
		ParentBenchmarkReferences: complianceResult.ParentBenchmarkReferences,
		ParentComplianceJobID:     complianceResult.ParentComplianceJobID,
		ParentBenchmarks:          complianceResult.ParentBenchmarks,
		LastEvent:                 time.UnixMilli(complianceResult.LastTransition),
	}
	if complianceResult.ConformanceStatus.IsPassed() {
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

type GetComplianceResultsResponse struct {
	ComplianceResults []ComplianceResult `json:"complianceResults"`
	TotalCount        int64              `json:"totalCount" example:"100"`
}

type ComplianceResultKPIResponse struct {
	FailedComplianceResultsCount int64 `json:"failedComplianceResultsCount"`
	FailedResourceCount          int64 `json:"failedResourceCount"`
	FailedControlCount           int64 `json:"failedControlCount"`
	FailedConnectionCount        int64 `json:"failedConnectionCount"`
}

type ServiceComplianceResultsSummary struct {
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

type GetServicesComplianceResultsSummaryResponse struct {
	Services []ServiceComplianceResultsSummary `json:"services"`
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

type CountComplianceResultsResponse struct {
	Count int64 `json:"count"`
}

type GetComplianceResultDriftEventsByComplianceResultIDResponse struct {
	ComplianceResultDriftEvents []ComplianceResultDriftEvent `json:"complianceResultDriftEvents"`
}

type ComplianceResultFiltersV2 struct {
	Integration     []IntegrationInfoFilter          `json:"integration"`
	BenchmarkID     []string                         `json:"benchmark_id"`
	NotBenchmarkID  []string                         `json:"not_benchmark_id"`
	ControlID       []string                         `json:"control_id"`
	NotControlID    []string                         `json:"not_control_id"`
	Severity        []types.ComplianceResultSeverity `json:"severity"`
	NotSeverity     []types.ComplianceResultSeverity `json:"not_severity"`
	ResourceID      []string                         `json:"resource_id"`
	NotResourceID   []string                         `json:"not_resource_id"`
	IsCompliant     []bool                           `json:"is_compliant"`
	IsActive        []bool                           `json:"is_active"`
	ResourceType    []string                         `json:"resource_type"`
	NotResourceType []string                         `json:"not_resource_type"`
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
	IntegrationType *string `json:"integration_type"`
	ProviderID      *string `json:"provider_id"`
	Name            *string `json:"name"`
	IntegrationID   *string `json:"integration_id"`
}

type ComplianceResultsSortV2 struct {
	BenchmarkID       *SortDirection `json:"benchmark_id"`
	ControlID         *SortDirection `json:"control_id"`
	Severity          *SortDirection `json:"severity"`
	ConformanceStatus *SortDirection `json:"conformance_status"`
	ResourceType      *SortDirection `json:"resource_type"`
	LastUpdated       *SortDirection `json:"last_updated"`
}

type GetComplianceResultsRequestV2 struct {
	Filters      ComplianceResultFiltersV2 `json:"filters"`
	Sort         []ComplianceResultsSortV2 `json:"sort"`
	Limit        int                       `json:"limit" example:"100"`
	AfterSortKey []any                     `json:"afterSortKey"`
}
