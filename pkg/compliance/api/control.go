package api

import (
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"time"
)

type Query struct {
	ID             string    `json:"id" example:"azure_ad_manual_control"`
	QueryToExecute string    `json:"queryToExecute" example:"select\n  -- Required Columns\n  'active_directory' as resource,\n  'info' as status,\n  'Manual verification required.' as reason;\n"`
	Connector      string    `json:"connector" example:"Azure"`
	PrimaryTable   *string   `json:"primaryTable" example:"null"`
	ListOfTables   []string  `json:"listOfTables" example:"null"`
	Engine         string    `json:"engine" example:"steampipe-v0.5"`
	CreatedAt      time.Time `json:"createdAt" example:"2023-06-07T14:00:15.677558Z"`
	UpdatedAt      time.Time `json:"updatedAt" example:"2023-06-16T14:58:08.759554Z"`
}

type Control struct {
	ID          string              `json:"id" example:"azure_cis_v140_1_1"`
	Title       string              `json:"title" example:"1.1 Ensure that multi-factor authentication status is enabled for all privileged users"`
	Description string              `json:"description" example:"Enable multi-factor authentication for all user credentials who have write access to Azure resources. These include roles like 'Service Co-Administrators', 'Subscription Owners', 'Contributors'."`
	Tags        map[string][]string `json:"tags"`

	Explanation       string `json:"explanation" example:"Multi-factor authentication adds an additional layer of security by requiring users to enter a code from a mobile device or phone in addition to their username and password when signing into Azure."`
	NonComplianceCost string `json:"nonComplianceCost" example:"Non-compliance to this control could result in several costs including..."`
	UsefulExample     string `json:"usefulExample" example:"Access to resources must be closely controlled to prevent malicious activity like data theft..."`

	Connector          source.Type           `json:"connector" example:"Azure"`
	Enabled            bool                  `json:"enabled" example:"true"`
	DocumentURI        string                `json:"documentURI" example:"benchmarks/azure_cis_v140_1_1.md"`
	Query              *Query                `json:"query"`
	Severity           types.FindingSeverity `json:"severity" example:"low"`
	ManualVerification bool                  `json:"manualVerification" example:"true"`
	Managed            bool                  `json:"managed" example:"true"`
	CreatedAt          time.Time             `json:"createdAt" example:"2020-01-01T00:00:00Z"`
	UpdatedAt          time.Time             `json:"updatedAt" example:"2020-01-01T00:00:00Z"`
}

type ControlSummary struct {
	Control      Control                    `json:"control"`
	ResourceType *inventoryApi.ResourceType `json:"resourceType"`

	Benchmarks []Benchmark `json:"benchmarks"`

	Passed                bool  `json:"passed"`
	FailedResourcesCount  int   `json:"failedResourcesCount"`
	TotalResourcesCount   int   `json:"totalResourcesCount"`
	FailedConnectionCount int   `json:"failedConnectionCount"`
	TotalConnectionCount  int   `json:"totalConnectionCount"`
	EvaluatedAt           int64 `json:"evaluatedAt"`
}

type ControlTrendDatapoint struct {
	Timestamp             int `json:"timestamp" example:"1686346668"` // Time
	FailedResourcesCount  int `json:"failedResourcesCount"`
	TotalResourcesCount   int `json:"totalResourcesCount"`
	FailedConnectionCount int `json:"failedConnectionCount"`
	TotalConnectionCount  int `json:"totalConnectionCount"`
}
