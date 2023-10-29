package api

import (
	types2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/worker/types"
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
	Connector          []source.Type            `json:"connector" example:"Azure"`                                                                                    // Clout Provider
	ResourceID         []string                 `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"` // Resource unique identifier
	ResourceTypeID     []string                 `json:"resourceTypeID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines"`  // Resource type
	ConnectionID       []string                 `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`                                                  // Connection ID
	ResourceCollection []string                 `json:"resourceCollection" example:"example-rc"`                                                                      // Resource Collection ID
	BenchmarkID        []string                 `json:"benchmarkID" example:"azure_cis_v140"`                                                                         // Benchmark ID
	PolicyID           []string                 `json:"policyID" example:"azure_cis_v140_7_5"`                                                                        // Policy ID
	Severity           []string                 `json:"severity" example:"low"`                                                                                       // Severity
	Status             []types.ComplianceResult `json:"status" example:"alarm"`                                                                                       // Compliance result status
	ActiveOnly         bool                     `json:"activeOnly"`
}

type GetFindingsRequest struct {
	Filters FindingFilters `json:"filters"`
}

type TopFieldRecord struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type BenchmarkRemediation struct {
	Remediation string `json:"remediation"`
}

type GetTopFieldResponse struct {
	TotalCount int              `json:"totalCount" example:"100"`
	Records    []TopFieldRecord `json:"records"`
}

type GetFieldCountResponse struct {
	Policies []struct {
		PolicyName  string           `json:"policyName"`
		FieldCounts []TopFieldRecord `json:"fieldCounts"`
	} `json:"policies"`
}

type AccountsFindingsSummary struct {
	AccountName     string  `json:"accountName"`
	AccountId       string  `json:"accountId"`
	SecurityScore   float64 `json:"securityScore"`
	SeveritiesCount struct {
		Critical int `json:"critical"`
		High     int `json:"high"`
		Low      int `json:"low"`
		Medium   int `json:"medium"`
	} `json:"severitiesCount"`
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
		Low      int `json:"low"`
		Medium   int `json:"medium"`
	} `json:"severitiesCount"`
}

type GetServicesFindingsSummaryResponse struct {
	Services []ServiceFindingsSummary `json:"services"`
}

type Finding struct {
	types.Finding

	PolicyTitle            string `json:"policyTitle"`
	ProviderConnectionID   string `json:"providerConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`   // Connection ID
	ProviderConnectionName string `json:"providerConnectionName" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Connection ID
}

type GetFindingsResponse struct {
	Findings   []Finding `json:"findings"`
	TotalCount int64     `json:"totalCount" example:"100"`
}

type GetBenchmarksSummaryResponse struct {
	BenchmarkSummary []BenchmarkEvaluationSummary `json:"benchmarkSummary"`

	TotalResult types.ComplianceResultSummary `json:"totalResult"`
	TotalChecks types.SeverityResult          `json:"totalChecks"`
}

type BenchmarkEvaluationSummary struct {
	ID          string                        `json:"id" example:"azure_cis_v140"`                                                                                                                                                       // Benchmark ID
	Title       string                        `json:"title" example:"Azure CIS v1.4.0"`                                                                                                                                                  // Benchmark title
	Description string                        `json:"description" example:"The CIS Microsoft Azure Foundations Security Benchmark provides prescriptive guidance for establishing a secure baseline configuration for Microsoft Azure."` // Benchmark description
	Connectors  []source.Type                 `json:"connectors" example:"[Azure]"`                                                                                                                                                      // Cloud providers
	Tags        map[string][]string           `json:"tags" `                                                                                                                                                                             // Tags
	Enabled     bool                          `json:"enabled" example:"true"`                                                                                                                                                            // Enabled
	Result      types.ComplianceResultSummary `json:"result"`                                                                                                                                                                            // Compliance result summary
	Checks      types.SeverityResult          `json:"checks"`                                                                                                                                                                            // Checks summary
	EvaluatedAt time.Time                     `json:"evaluatedAt" example:"2020-01-01T00:00:00Z"`                                                                                                                                        // Evaluated at
}

type PolicySummary struct {
	Policy
	types2.PolicyResult
}
