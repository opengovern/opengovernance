package types

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

type BenchmarkSummaryResult struct {
	BenchmarkResult Result
	Connections     map[string]Result
	ResourceTypes   map[string]Result
	Policies        map[string]PolicyResult
}

type BenchmarkSummary struct {
	BenchmarkID      string
	JobID            uint
	EvaluatedAtEpoch int64

	Connections         BenchmarkSummaryResult
	ResourceCollections map[string]BenchmarkSummaryResult
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

	if f.ResourceCollection == nil {
		b.Connections.BenchmarkResult.SeverityResult[f.Severity]++
		b.Connections.BenchmarkResult.QueryResult[f.Result]++

		connection, ok := b.Connections.Connections[f.ConnectionID]
		if !ok {
			connection = Result{
				QueryResult:    map[types.ComplianceResult]int{},
				SeverityResult: map[types.FindingSeverity]int{},
			}
		}
		connection.SeverityResult[f.Severity]++
		connection.QueryResult[f.Result]++
		b.Connections.Connections[f.ConnectionID] = connection

		resourceType, ok := b.Connections.ResourceTypes[f.ResourceType]
		if !ok {
			resourceType = Result{
				QueryResult:    map[types.ComplianceResult]int{},
				SeverityResult: map[types.FindingSeverity]int{},
			}
		}
		resourceType.SeverityResult[f.Severity]++
		resourceType.QueryResult[f.Result]++
		b.Connections.ResourceTypes[f.ResourceType] = resourceType

		policy, ok := b.Connections.Policies[f.PolicyID]
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
		b.Connections.Policies[f.PolicyID] = policy

	} else {
		rc, ok := b.ResourceCollections[*f.ResourceCollection]
		if !ok {
			rc = BenchmarkSummaryResult{
				BenchmarkResult: Result{
					QueryResult:    map[types.ComplianceResult]int{},
					SeverityResult: map[types.FindingSeverity]int{},
					SecurityScore:  0,
				},
				Connections:   map[string]Result{},
				ResourceTypes: map[string]Result{},
				Policies:      map[string]PolicyResult{},
			}
		}
		rc.BenchmarkResult.SeverityResult[f.Severity]++
		rc.BenchmarkResult.QueryResult[f.Result]++

		resourceType, ok := b.Connections.ResourceTypes[f.ResourceType]
		if !ok {
			resourceType = Result{
				QueryResult:    map[types.ComplianceResult]int{},
				SeverityResult: map[types.FindingSeverity]int{},
			}
		}
		resourceType.SeverityResult[f.Severity]++
		resourceType.QueryResult[f.Result]++

		rc.ResourceTypes[f.ResourceType] = resourceType

		policy, ok := rc.Policies[f.PolicyID]
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
		rc.Policies[f.PolicyID] = policy

		b.ResourceCollections[*f.ResourceCollection] = rc
	}
}

func (b *BenchmarkSummary) Summarize() {
	// update security scores
	for policyID, summary := range b.Connections.Policies {
		summary.FailedConnectionCount = len(summary.failedConnections)
		summary.TotalConnectionCount = len(summary.allConnections)

		summary.FailedResourcesCount = len(summary.failedResources)
		summary.TotalResourcesCount = len(summary.allResources)

		b.Connections.Policies[policyID] = summary
	}

	for connectionID, summary := range b.Connections.Connections {
		total := 0
		for _, count := range summary.QueryResult {
			total += count
		}

		if total > 0 {
			summary.SecurityScore = float64(summary.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
		}

		b.Connections.Connections[connectionID] = summary
	}

	for resourceType, summary := range b.Connections.ResourceTypes {
		total := 0
		for _, count := range summary.QueryResult {
			total += count
		}

		if total > 0 {
			summary.SecurityScore = float64(summary.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
		}

		b.Connections.ResourceTypes[resourceType] = summary
	}

	total := 0
	for _, count := range b.Connections.BenchmarkResult.QueryResult {
		total += count
	}
	if total > 0 {
		b.Connections.BenchmarkResult.SecurityScore = float64(b.Connections.BenchmarkResult.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
	}

	for rcId, rc := range b.ResourceCollections {
		for policyID, summary := range rc.Policies {
			summary.FailedConnectionCount = len(summary.failedConnections)
			summary.TotalConnectionCount = len(summary.allConnections)

			summary.FailedResourcesCount = len(summary.failedResources)
			summary.TotalResourcesCount = len(summary.allResources)

			rc.Policies[policyID] = summary
		}

		for connectionID, summary := range rc.Connections {
			total := 0
			for _, count := range summary.QueryResult {
				total += count
			}

			if total > 0 {
				summary.SecurityScore = float64(summary.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
			}

			rc.Connections[connectionID] = summary
		}

		for resourceType, summary := range rc.ResourceTypes {
			total := 0
			for _, count := range summary.QueryResult {
				total += count
			}

			if total > 0 {
				summary.SecurityScore = float64(summary.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
			}

			rc.ResourceTypes[resourceType] = summary
		}

		total := 0
		for _, count := range rc.BenchmarkResult.QueryResult {
			total += count
		}
		if total > 0 {
			rc.BenchmarkResult.SecurityScore = float64(rc.BenchmarkResult.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
		}

		b.ResourceCollections[rcId] = rc
	}
}
