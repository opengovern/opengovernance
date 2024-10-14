package api

import (
	inventoryApi "github.com/opengovern/opengovernance/pkg/inventory/api"
	onboardApi "github.com/opengovern/opengovernance/pkg/onboard/api"
	"time"

	"github.com/opengovern/og-util/pkg/source"
)

const (
	QueryEngine_Odysseues    = "odysseus-v0.0.1"
	QueryEngine_OdysseusSQL  = "odysseus-sql"
	QueryEngine_OdysseusRego = "odysseus-rego"
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
	Connections []BenchmarkAssignedConnection `json:"connections"`
}

type TopFieldRecord struct {
	Connection   *onboardApi.Connection
	ResourceType *inventoryApi.ResourceType
	Control      *Control
	Service      *string

	Field *string `json:"field"`

	ControlCount      *int `json:"controlCount,omitempty"`
	ControlTotalCount *int `json:"controlTotalCount,omitempty"`

	ResourceCount      *int `json:"resourceCount,omitempty"`
	ResourceTotalCount *int `json:"resourceTotalCount,omitempty"`

	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
}

type BenchmarkRemediation struct {
	Remediation string `json:"remediation"`
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

type SortDirection string

const (
	SortDirectionAscending  SortDirection = "asc"
	SortDirectionDescending SortDirection = "desc"
)

type FilterWithMetadata struct {
	Key         string `json:"key" example:"key"`
	DisplayName string `json:"displayName" example:"displayName"`
	Count       *int   `json:"count" example:"10"`
}

type QueryParameter struct {
	Key      string `json:"key" example:"key"`
	Required bool   `json:"required" example:"true"`
}

type Query struct {
	ID             string           `json:"id" example:"azure_ad_manual_control"`
	QueryToExecute string           `json:"queryToExecute" example:"select\n  -- Required Columns\n  'active_directory' as resource,\n  'info' as status,\n  'Manual verification required.' as reason;\n"`
	Connector      []source.Type    `json:"connector" example:"Azure"`
	PrimaryTable   *string          `json:"primaryTable" example:"null"`
	ListOfTables   []string         `json:"listOfTables" example:"null"`
	Engine         string           `json:"engine" example:"steampipe-v0.5"`
	Parameters     []QueryParameter `json:"parameters"`
	Global         bool             `json:"Global"`
	CreatedAt      time.Time        `json:"createdAt" example:"2023-06-07T14:00:15.677558Z"`
	UpdatedAt      time.Time        `json:"updatedAt" example:"2023-06-16T14:58:08.759554Z"`
}
