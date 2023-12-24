package api

import (
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type BenchmarkAssignment struct {
	BenchmarkId          string    `json:"benchmarkId" example:"azure_cis_v140"`                        // Benchmark ID
	ConnectionId         *string   `json:"connectionId" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Connection ID
	ResourceCollectionId *string   `json:"resourceCollectionId" example:"example-rc"`                   // Resource Collection ID
	AssignedAt           time.Time `json:"assignedAt"`                                                  // Unix timestamp
}

type AssignedBenchmark struct {
	Benchmark Benchmark `json:"benchmarkId"`
	Status    bool      `json:"status" example:"true"` // Status
}

type BenchmarkAssignedConnection struct {
	ConnectionID           string      `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Connection ID
	ProviderConnectionID   string      `json:"providerConnectionID" example:"1283192749"`                   // Provider Connection ID
	ProviderConnectionName string      `json:"providerConnectionName"`                                      // Provider Connection Name
	Connector              source.Type `json:"connector" example:"Azure"`                                   // Clout Provider
	Status                 bool        `json:"status" example:"true"`                                       // Status
}

type BenchmarkAssignedResourceCollection struct {
	ResourceCollectionID   string `json:"resourceCollectionID"`   // Resource Collection ID
	ResourceCollectionName string `json:"resourceCollectionName"` // Resource Collection Name
	Status                 bool   `json:"status" example:"true"`  // Status
}

type BenchmarkAssignedEntities struct {
	Connections         []BenchmarkAssignedConnection         `json:"connections"`
	ResourceCollections []BenchmarkAssignedResourceCollection `json:"resourceCollections"`
}

type FindingFilters struct {
	Connector          []source.Type             `json:"connector" example:"Azure"`                                                                                    // Clout Provider
	ResourceID         []string                  `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"` // Resource unique identifier
	ResourceTypeID     []string                  `json:"resourceTypeID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines"`  // Resource type
	ConnectionID       []string                  `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`                                                  // Connection ID
	ResourceCollection []string                  `json:"resourceCollection" example:"example-rc"`                                                                      // Resource Collection ID
	BenchmarkID        []string                  `json:"benchmarkID" example:"azure_cis_v140"`                                                                         // Benchmark ID
	ControlID          []string                  `json:"controlID" example:"azure_cis_v140_7_5"`                                                                       // Control ID
	Severity           []string                  `json:"severity" example:"low"`                                                                                       // Severity
	ConformanceStatus  []types.ConformanceStatus `json:"conformanceStatus" example:"alarm"`
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

type GetFindingsRequest struct {
	Filters      FindingFilters    `json:"filters"`
	Sort         map[string]string `json:"sort"`
	Limit        int               `json:"limit" example:"100"`
	AfterSortKey []any             `json:"afterSortKey"`
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

type TopFieldRecord struct {
	Connection   *onboardApi.Connection
	ResourceType *inventoryApi.ResourceType
	Control      *Control
	Service      *string

	Field      *string `json:"field"`
	Count      int     `json:"count"`
	TotalCount int     `json:"totalCount"`
}

type BenchmarkRemediation struct {
	Remediation string `json:"remediation"`
}

type GetTopFieldResponse struct {
	TotalCount int              `json:"totalCount" example:"100"`
	Records    []TopFieldRecord `json:"records"`
}

type GetFieldCountResponse struct {
	Controls []struct {
		ControlName string           `json:"controlName"`
		FieldCounts []TopFieldRecord `json:"fieldCounts"`
	} `json:"controls"`
}

type AccountsFindingsSummary struct {
	AccountName     string  `json:"accountName"`
	AccountId       string  `json:"accountId"`
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
		Error  int `json:"error"`
		Info   int `json:"info"`
		Skip   int `json:"skip"`
	} `json:"conformanceStatusesCount"`
	LastCheckTime time.Time `json:"lastCheckTime"`
}

type GetAccountsFindingsSummaryResponse struct {
	Accounts []AccountsFindingsSummary `json:"accounts"`
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
		Error  int `json:"error"`
		Info   int `json:"info"`
		Skip   int `json:"skip"`
	} `json:"conformanceStatusesCount"`
}

type GetServicesFindingsSummaryResponse struct {
	Services []ServiceFindingsSummary `json:"services"`
}

type Finding struct {
	types.Finding

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
