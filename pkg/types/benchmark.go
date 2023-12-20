package types

import (
	"strings"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type FullBenchmark struct {
	ID    string `json:"ID" example:"azure_cis_v140"` // Benchmark ID
	Title string `json:"title" example:"CIS v1.4.0"`  // Benchmark title
}

type Finding struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	BenchmarkID           string           `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID             string           `json:"controlID" example:"azure_cis_v140_7_5"`
	ConnectionID          string           `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	EvaluatedAt           int64            `json:"evaluatedAt" example:"1589395200"`
	StateActive           bool             `json:"stateActive" example:"true"`
	Result                ComplianceResult `json:"result" example:"alarm"`
	Severity              FindingSeverity  `json:"severity" example:"low"`
	Evaluator             string           `json:"evaluator" example:"steampipe-v0.5"`
	Connector             source.Type      `json:"connector" example:"Azure"`
	KaytuResourceID       string           `json:"kaytuResourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceID            string           `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceName          string           `json:"resourceName" example:"vm-1"`
	ResourceLocation      string           `json:"resourceLocation" example:"eastus"`
	ResourceType          string           `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
	Reason                string           `json:"reason" example:"The VM is not using managed disks"`
	ComplianceJobID       uint             `json:"complianceJobID" example:"1"`
	ParentComplianceJobID uint             `json:"parentComplianceJobID" example:"1"`

	ResourceCollection *string  `json:"resourceCollection"` // Resource collection
	ParentBenchmarks   []string `json:"parentBenchmarks"`
}

func (r Finding) KeysAndIndex() ([]string, string) {
	index := FindingsIndex
	keys := []string{
		r.ResourceID,
		r.ConnectionID,
		r.ControlID,
	}
	if r.ResourceCollection != nil {
		keys = append(keys, *r.ResourceCollection)
		index = ResourceCollectionsFindingsIndex
	}
	if strings.HasPrefix(r.ConnectionID, "stack-") {
		index = StackFindingsIndex
	}
	return keys, index
}
