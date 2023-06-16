package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
)

type Benchmark struct {
	ID          string              `json:"id" example:"azure_cis_v140"`                                                                                                                                                       // Benchmark ID
	Title       string              `json:"title" example:"Azure CIS v1.4.0"`                                                                                                                                                  // Benchmark title
	Description string              `json:"description" example:"The CIS Microsoft Azure Foundations Security Benchmark provides prescriptive guidance for establishing a secure baseline configuration for Microsoft Azure."` // Benchmark description
	LogoURI     string              `json:"logoURI"`                                                                                                                                                                           // Benchmark logo URI
	Category    string              `json:"category"`                                                                                                                                                                          // Benchmark category
	DocumentURI string              `json:"documentURI" example:"benchmarks/azure_cis_v140.md"`                                                                                                                                // Benchmark document URI
	Enabled     bool                `json:"enabled" example:"true"`                                                                                                                                                            // Whether the benchmark is enabled or not
	Managed     bool                `json:"managed" example:"true"`                                                                                                                                                            // Whether the benchmark is managed or not
	AutoAssign  bool                `json:"autoAssign" example:"true"`                                                                                                                                                         // Whether the benchmark is auto assigned or not
	Baseline    bool                `json:"baseline" example:"true"`                                                                                                                                                           // Whether the benchmark is baseline or not
	Tags        map[string][]string `json:"tags" example:"{category:[Compliance],cis:[true]}"`                                                                                                                                 // Benchmark tags
	Connectors  []source.Type       `json:"connectors" example:"[azure]"`                                                                                                                                                      // Benchmark connectors
	Children    []string            `json:"children" example:"[azure_cis_v140_1, azure_cis_v140_2]"`                                                                                                                           // Benchmark children
	Policies    []string            `json:"policies" example:"[azure_cis_v140_1_1, azure_cis_v140_1_2]"`                                                                                                                       // Benchmark policies
	CreatedAt   time.Time           `json:"createdAt" example:"2020-01-01T00:00:00Z"`                                                                                                                                          // Benchmark creation date
	UpdatedAt   time.Time           `json:"updatedAt" example:"2020-01-01T00:00:00Z"`                                                                                                                                          // Benchmark last update date
}

type Policy struct {
	ID                 string              `json:"id"`
	Title              string              `json:"title"`
	Description        string              `json:"description"`
	Tags               map[string][]string `json:"tags"`
	Connector          source.Type         `json:"connector"`
	Enabled            bool                `json:"enabled"`
	DocumentURI        string              `json:"documentURI"`
	QueryID            *string             `json:"queryID"`
	Severity           types.Severity      `json:"severity"`
	ManualVerification bool                `json:"manualVerification"`
	Managed            bool                `json:"managed"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
}

type Query struct {
	ID             string    `json:"id"`
	QueryToExecute string    `json:"queryToExecute"`
	Connector      string    `json:"connector"`
	ListOfTables   []string  `json:"listOfTables"`
	Engine         string    `json:"engine"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
