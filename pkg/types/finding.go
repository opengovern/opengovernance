package types

import (
	"fmt"
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/og-util/pkg/source"
)

type FindingEvent struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	FindingEsID               string            `json:"findingEsID"`
	ParentComplianceJobID     uint              `json:"parentComplianceJobID"`
	ComplianceJobID           uint              `json:"complianceJobID"`
	PreviousConformanceStatus ConformanceStatus `json:"previousConformanceStatus"`
	ConformanceStatus         ConformanceStatus `json:"conformanceStatus"`
	PreviousStateActive       bool              `json:"previousStateActive"`
	StateActive               bool              `json:"stateActive"`
	EvaluatedAt               int64             `json:"evaluatedAt"`
	Reason                    string            `json:"reason"`

	BenchmarkID               string           `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID                 string           `json:"controlID" example:"azure_cis_v140_7_5"`
	ConnectionID              string           `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	IntegrationType           integration.Type `json:"integrationType" example:"Azure"`
	Severity                  FindingSeverity  `json:"severity" example:"low"`
	OpenGovernanceResourceID  string           `json:"opengovernanceResourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceID                string           `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceType              string           `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
	ParentBenchmarkReferences []string         `json:"parentBenchmarkReferences"`
}

func (r FindingEvent) KeysAndIndex() ([]string, string) {
	return []string{
		r.FindingEsID,
		fmt.Sprintf("%d", r.ComplianceJobID),
		fmt.Sprintf("%d", r.EvaluatedAt),
	}, FindingEventsIndex
}

type Finding struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	BenchmarkID              string            `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID                string            `json:"controlID" example:"azure_cis_v140_7_5"`
	ConnectionID             string            `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	EvaluatedAt              int64             `json:"evaluatedAt" example:"1589395200"`
	StateActive              bool              `json:"stateActive" example:"true"`
	ConformanceStatus        ConformanceStatus `json:"conformanceStatus" example:"alarm"`
	Severity                 FindingSeverity   `json:"severity" example:"low"`
	Evaluator                string            `json:"evaluator" example:"steampipe-v0.5"`
	IntegrationType          integration.Type  `json:"integrationType" example:"Azure"`
	OpenGovernanceResourceID string            `json:"opengovernanceResourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceID               string            `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceName             string            `json:"resourceName" example:"vm-1"`
	ResourceLocation         string            `json:"resourceLocation" example:"eastus"`
	ResourceType             string            `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
	Reason                   string            `json:"reason" example:"The VM is not using managed disks"`
	CostOptimization         *float64          `json:"costOptimization"`
	ComplianceJobID          uint              `json:"complianceJobID" example:"1"`
	ParentComplianceJobID    uint              `json:"parentComplianceJobID" example:"1"`
	LastTransition           int64             `json:"lastTransition" example:"1589395200"`

	ParentBenchmarkReferences []string `json:"parentBenchmarkReferences"`
	ParentBenchmarks          []string `json:"parentBenchmarks"`
}

func (r Finding) KeysAndIndex() ([]string, string) {
	index := FindingsIndex
	keys := []string{
		r.OpenGovernanceResourceID,
		r.ResourceID,
		r.ConnectionID,
		r.ControlID,
		r.BenchmarkID,
	}
	keys = append(keys, r.ParentBenchmarks...)
	return keys, index
}

type ConformanceStatusPerSeverity struct {
	OkCount    SeverityResultWithTotal `json:"okCount"`
	AlarmCount SeverityResultWithTotal `json:"alarmCount"`
	InfoCount  SeverityResultWithTotal `json:"infoCount"`
	SkipCount  SeverityResultWithTotal `json:"skipCount"`
	ErrorCount SeverityResultWithTotal `json:"errorCount"`
}

type ResourceFinding struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	OpenGovernanceResourceID string      `json:"opengovernanceResourceID"`
	ResourceType             string      `json:"resourceType"`
	ResourceName             string      `json:"resourceName"`
	ResourceLocation         string      `json:"resourceLocation"`
	IntegrationType          source.Type `json:"integrationType"`

	//ConformanceStatusPerSeverity ConformanceStatusPerSeverity `json:"conformanceStatusPerSeverity"`
	Findings []Finding `json:"findings"`

	ResourceCollection    []string        `json:"resourceCollection" example:"azure_cis_v140_7_5"`
	ResourceCollectionMap map[string]bool `json:"-"` // for creation of the slice only

	JobId       uint  `json:"jobId" example:"1"`
	EvaluatedAt int64 `json:"evaluatedAt" example:"1589395200"`
}

func (r ResourceFinding) KeysAndIndex() ([]string, string) {
	return []string{
		r.OpenGovernanceResourceID,
		r.ResourceType,
	}, ResourceFindingsIndex
}
