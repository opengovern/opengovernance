package types

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
)

type Result struct {
	QueryResult    map[types.ComplianceResult]int
	SeverityResult map[types.FindingSeverity]int

	SecurityScore float64
}

type PolicyResult struct {
	Passed bool

	FailedResourcesCount int
	TotalResourcesCount  int

	allResources    map[string]any
	failedResources map[string]any

	FailedConnectionCount int
	TotalConnectionCount  int

	allConnections    map[string]any
	failedConnections map[string]any
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
	return []string{b.BenchmarkID, fmt.Sprintf("%d", b.JobID)}, types.BenchmarkSummaryIndex
}

func (r *BenchmarkSummaryResult) addFinding(f types.Finding) {
	r.BenchmarkResult.SeverityResult[f.Severity]++
	r.BenchmarkResult.QueryResult[f.Result]++

	connection, ok := r.Connections[f.ConnectionID]
	if !ok {
		connection = Result{
			QueryResult:    map[types.ComplianceResult]int{},
			SeverityResult: map[types.FindingSeverity]int{},
		}
	}
	connection.SeverityResult[f.Severity]++
	connection.QueryResult[f.Result]++
	r.Connections[f.ConnectionID] = connection

	resourceType, ok := r.ResourceTypes[f.ResourceType]
	if !ok {
		resourceType = Result{
			QueryResult:    map[types.ComplianceResult]int{},
			SeverityResult: map[types.FindingSeverity]int{},
		}
	}
	resourceType.SeverityResult[f.Severity]++
	resourceType.QueryResult[f.Result]++
	r.ResourceTypes[f.ResourceType] = resourceType

	policy, ok := r.Policies[f.PolicyID]
	if !ok {
		policy = PolicyResult{
			Passed:            true,
			allResources:      map[string]any{},
			failedResources:   map[string]any{},
			allConnections:    map[string]any{},
			failedConnections: map[string]any{},
		}
	}

	if !f.Result.IsPassed() {
		policy.Passed = false

		policy.failedResources[f.ResourceID] = struct{}{}
		policy.failedConnections[f.ConnectionID] = struct{}{}
	}
	policy.allResources[f.ResourceID] = struct{}{}
	policy.allConnections[f.ConnectionID] = struct{}{}
	r.Policies[f.PolicyID] = policy
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
		b.Connections.addFinding(f)
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

		rc.addFinding(f)
		b.ResourceCollections[*f.ResourceCollection] = rc
	}
}

func (r *BenchmarkSummaryResult) summarize() {
	// update security scores
	for policyID, summary := range r.Policies {
		summary.FailedConnectionCount = len(summary.failedConnections)
		summary.TotalConnectionCount = len(summary.allConnections)

		summary.FailedResourcesCount = len(summary.failedResources)
		summary.TotalResourcesCount = len(summary.allResources)

		r.Policies[policyID] = summary
	}

	for connectionID, summary := range r.Connections {
		total := 0
		for _, count := range summary.QueryResult {
			total += count
		}

		if total > 0 {
			summary.SecurityScore = float64(summary.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
		}

		r.Connections[connectionID] = summary
	}

	for resourceType, summary := range r.ResourceTypes {
		total := 0
		for _, count := range summary.QueryResult {
			total += count
		}

		if total > 0 {
			summary.SecurityScore = float64(summary.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
		}

		r.ResourceTypes[resourceType] = summary
	}

	total := 0
	for _, count := range r.BenchmarkResult.QueryResult {
		total += count
	}
	if total > 0 {
		r.BenchmarkResult.SecurityScore = float64(r.BenchmarkResult.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
	}
}

func (b *BenchmarkSummary) Summarize() {
	b.Connections.summarize()
	for rcId, rc := range b.ResourceCollections {
		rc.summarize()
		b.ResourceCollections[rcId] = rc
	}
}
