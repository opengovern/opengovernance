package types

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strings"
)

type FindingHistory struct {
	ComplianceJobID   uint              `json:"jobId"`
	ConformanceStatus ConformanceStatus `json:"conformanceStatus"`
	EvaluatedAt       int64             `json:"evaluatedAt"`
}

type Finding struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	BenchmarkID           string            `json:"benchmarkID" example:"azure_cis_v140"`
	ControlID             string            `json:"controlID" example:"azure_cis_v140_7_5"`
	ConnectionID          string            `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	EvaluatedAt           int64             `json:"evaluatedAt" example:"1589395200"`
	StateActive           bool              `json:"stateActive" example:"true"`
	ConformanceStatus     ConformanceStatus `json:"conformanceStatus" example:"alarm"`
	Severity              FindingSeverity   `json:"severity" example:"low"`
	Evaluator             string            `json:"evaluator" example:"steampipe-v0.5"`
	Connector             source.Type       `json:"connector" example:"Azure"`
	KaytuResourceID       string            `json:"kaytuResourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceID            string            `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceName          string            `json:"resourceName" example:"vm-1"`
	ResourceLocation      string            `json:"resourceLocation" example:"eastus"`
	ResourceType          string            `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
	Reason                string            `json:"reason" example:"The VM is not using managed disks"`
	ComplianceJobID       uint              `json:"complianceJobID" example:"1"`
	ParentComplianceJobID uint              `json:"parentComplianceJobID" example:"1"`

	History []FindingHistory `json:"history"`

	ParentBenchmarks []string `json:"parentBenchmarks"`
}

func (r Finding) KeysAndIndex() ([]string, string) {
	index := FindingsIndex
	keys := []string{
		r.ResourceID,
		r.ConnectionID,
		r.ControlID,
		r.BenchmarkID,
	}
	if strings.HasPrefix(r.ConnectionID, "stack-") {
		index = StackFindingsIndex
	}
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

	KaytuResourceID  string      `json:"kaytuResourceID"`
	ResourceType     string      `json:"resourceType"`
	ResourceName     string      `json:"resourceName"`
	ResourceLocation string      `json:"resourceLocation"`
	Connector        source.Type `json:"connector"`

	//ConformanceStatusPerSeverity ConformanceStatusPerSeverity `json:"conformanceStatusPerSeverity"`
	Findings []Finding `json:"findings"`

	ResourceCollection    []string        `json:"resourceCollection" example:"azure_cis_v140_7_5"`
	ResourceCollectionMap map[string]bool `json:"-"` // for creation of the slice only

	JobId       uint  `json:"jobId" example:"1"`
	EvaluatedAt int64 `json:"evaluatedAt" example:"1589395200"`
}

func (r ResourceFinding) KeysAndIndex() ([]string, string) {
	return []string{
		r.KaytuResourceID,
	}, ResourceFindingsIndex
}
