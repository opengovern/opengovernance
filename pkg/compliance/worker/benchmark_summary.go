package worker

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
)

const BenchmarkSummaryIndex = "benchmark_summary"

type Result struct {
	QueryResult    map[types.ComplianceResult]int
	SeverityResult map[types.FindingSeverity]int

	SecurityScore float64
}

type PolicyResult struct {
	Passed bool

	FailedResourcesCount int
	TotalResourcesCount  int

	allResources    map[string]interface{}
	failedResources map[string]interface{}

	FailedConnectionCount int
	TotalConnectionCount  int

	allConnections    map[string]interface{}
	failedConnections map[string]interface{}
}

type BenchmarkSummary struct {
	BenchmarkID      string
	JobID            uint
	EvaluatedAtEpoch int64

	BenchmarkResult Result

	Connections   map[string]Result
	ResourceTypes map[string]Result
	Policies      map[string]PolicyResult
}

func (b BenchmarkSummary) KeysAndIndex() ([]string, string) {
	return []string{b.BenchmarkID, fmt.Sprintf("%d", b.JobID)}, BenchmarkSummaryIndex
}

func (b *BenchmarkSummary) AddFinding(f types.Finding) {
	if f.Severity == "" {
		f.Severity = types.FindingSeverityNone
	}
	if f.Result == "" {
		f.Result = types.ComplianceResultERROR
	}
	if f.ResourceType == "" {
		f.ResourceType = "-"
	}

	b.BenchmarkResult.SeverityResult[f.Severity]++
	b.BenchmarkResult.QueryResult[f.Result]++

	connection, ok := b.Connections[f.ConnectionID]
	if !ok {
		connection = Result{
			QueryResult:    map[types.ComplianceResult]int{},
			SeverityResult: map[types.FindingSeverity]int{},
		}
	}
	connection.SeverityResult[f.Severity]++
	connection.QueryResult[f.Result]++
	b.Connections[f.ConnectionID] = connection

	resourceType, ok := b.ResourceTypes[f.ResourceType]
	if !ok {
		resourceType = Result{
			QueryResult:    map[types.ComplianceResult]int{},
			SeverityResult: map[types.FindingSeverity]int{},
		}
	}
	resourceType.SeverityResult[f.Severity]++
	resourceType.QueryResult[f.Result]++
	b.ResourceTypes[f.ResourceType] = resourceType

	policy, ok := b.Policies[f.PolicyID]
	if !ok {
		policy = PolicyResult{
			Passed:            true,
			allResources:      map[string]interface{}{},
			failedResources:   map[string]interface{}{},
			allConnections:    map[string]interface{}{},
			failedConnections: map[string]interface{}{},
		}
	}

	if !f.Result.IsPassed() {
		policy.Passed = false

		policy.failedResources[f.ResourceID] = struct{}{}
		policy.failedConnections[f.ConnectionID] = struct{}{}
	}
	policy.allResources[f.ResourceID] = struct{}{}
	policy.allConnections[f.ConnectionID] = struct{}{}
	b.Policies[f.PolicyID] = policy
}

func (b *BenchmarkSummary) Summarize() {
	// update security scores
	for policyID, summary := range b.Policies {
		summary.FailedConnectionCount = len(summary.failedConnections)
		summary.TotalConnectionCount = len(summary.allConnections)

		summary.FailedResourcesCount = len(summary.failedResources)
		summary.TotalResourcesCount = len(summary.allResources)

		b.Policies[policyID] = summary
	}

	for connectionID, summary := range b.Connections {
		total := 0
		for _, count := range summary.QueryResult {
			total += count
		}

		if total > 0 {
			summary.SecurityScore = float64(summary.QueryResult[types.ComplianceResultOK]) / float64(total)
		}

		b.Connections[connectionID] = summary
	}

	for resourceType, summary := range b.ResourceTypes {
		total := 0
		for _, count := range summary.QueryResult {
			total += count
		}

		if total > 0 {
			summary.SecurityScore = float64(summary.QueryResult[types.ComplianceResultOK]) / float64(total)
		}

		b.ResourceTypes[resourceType] = summary
	}

	total := 0
	for _, count := range b.BenchmarkResult.QueryResult {
		total += count
	}
	if total > 0 {
		b.BenchmarkResult.SecurityScore = float64(b.BenchmarkResult.QueryResult[types.ComplianceResultOK]) / float64(total)
	}
}
