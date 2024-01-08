package api

import (
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strings"
)

type FindingFilters struct {
	Connector         []source.Type           `json:"connector" example:"Azure"`                                                                                    // Clout Provider
	ResourceID        []string                `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"` // Resource unique identifier
	ResourceTypeID    []string                `json:"resourceTypeID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines"`  // Resource type
	ConnectionID      []string                `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`                                                  // Connection ID
	BenchmarkID       []string                `json:"benchmarkID" example:"azure_cis_v140"`                                                                         // Benchmark ID
	ControlID         []string                `json:"controlID" example:"azure_cis_v140_7_5"`                                                                       // Control ID
	Severity          []types.FindingSeverity `json:"severity" example:"low"`                                                                                       // Severity
	ConformanceStatus []ConformanceStatus     `json:"conformanceStatus" example:"alarm"`
}

type FindingFilterWithMetadata struct {
	Key         string `json:"key" example:"key"`                 // Key
	DisplayName string `json:"displayName" example:"displayName"` // Display Name
	Count       *int   `json:"count" example:"10"`                // Count
}

type FindingFiltersWithMetadata struct {
	Connector          []FindingFilterWithMetadata `json:"connector"`
	BenchmarkID        []FindingFilterWithMetadata `json:"benchmarkID"`
	ControlID          []FindingFilterWithMetadata `json:"controlID"`
	ResourceTypeID     []FindingFilterWithMetadata `json:"resourceTypeID"`
	ConnectionID       []FindingFilterWithMetadata `json:"connectionID"`
	ResourceCollection []FindingFilterWithMetadata `json:"resourceCollection"`
	Severity           []FindingFilterWithMetadata `json:"severity"`
	ConformanceStatus  []FindingFilterWithMetadata `json:"conformanceStatus"`
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
	KaytuResourceId string `json:"kaytuResourceId" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
}

type GetSingleResourceFindingResponse struct {
	Resource        es.Resource `json:"resource"`
	ControlFindings []Finding   `json:"controls"`
}

type ConformanceStatus string

const (
	ConformanceStatusFailed ConformanceStatus = "failed"
	ConformanceStatusPassed ConformanceStatus = "passed"
)

func (cs ConformanceStatus) GetEsConformanceStatuses() []types.ConformanceStatus {
	switch cs {
	case ConformanceStatusFailed:
		return []types.ConformanceStatus{types.ConformanceStatusALARM, types.ConformanceStatusERROR, types.ConformanceStatusINFO, types.ConformanceStatusSKIP}
	case ConformanceStatusPassed:
		return []types.ConformanceStatus{types.ConformanceStatusOK}
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
	BenchmarkID           string                `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID             string                `json:"controlID" example:"azure_cis_v140_7_5"`
	ConnectionID          string                `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	EvaluatedAt           int64                 `json:"evaluatedAt" example:"1589395200"`
	StateActive           bool                  `json:"stateActive" example:"true"`
	ConformanceStatus     ConformanceStatus     `json:"conformanceStatus" example:"alarm"`
	Severity              types.FindingSeverity `json:"severity" example:"low"`
	Evaluator             string                `json:"evaluator" example:"steampipe-v0.5"`
	Connector             source.Type           `json:"connector" example:"Azure"`
	KaytuResourceID       string                `json:"kaytuResourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceID            string                `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceName          string                `json:"resourceName" example:"vm-1"`
	ResourceLocation      string                `json:"resourceLocation" example:"eastus"`
	ResourceType          string                `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
	Reason                string                `json:"reason" example:"The VM is not using managed disks"`
	ComplianceJobID       uint                  `json:"complianceJobID" example:"1"`
	ParentComplianceJobID uint                  `json:"parentComplianceJobID" example:"1"`
	ParentBenchmarks      []string              `json:"parentBenchmarks"`

	ResourceTypeName            string   `json:"resourceTypeName" example:"Virtual Machine"`
	ParentBenchmarkNames        []string `json:"parentBenchmarkNames" example:"Azure CIS v1.4.0"`
	ParentBenchmarkDisplayCodes []string `json:"parentBenchmarkDisplayCodes" example:"Azure CIS v1.4.0"`
	ControlTitle                string   `json:"controlTitle"`
	ProviderConnectionID        string   `json:"providerConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`   // Connection ID
	ProviderConnectionName      string   `json:"providerConnectionName" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Connection ID
	NoOfOccurrences             int      `json:"noOfOccurrences" example:"1"`

	SortKey []any `json:"sortKey"`
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
