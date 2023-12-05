package types

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
)

type Result struct {
	QueryResult    map[types.ComplianceResult]int
	SeverityResult map[types.FindingSeverity]int
	SecurityScore  float64
}

type ResultGroup struct {
	Result        Result
	ResourceTypes map[string]Result
	Controls      map[string]ControlResult
}

type ControlResult struct {
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
	BenchmarkResult ResultGroup
	Connections     map[string]ResultGroup
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
	r.BenchmarkResult.Result.SeverityResult[f.Severity]++
	r.BenchmarkResult.Result.QueryResult[f.Result]++

	connection, ok := r.Connections[f.ConnectionID]
	if !ok {
		connection = ResultGroup{
			Result: Result{
				QueryResult:    map[types.ComplianceResult]int{},
				SeverityResult: map[types.FindingSeverity]int{},
				SecurityScore:  0,
			},
			ResourceTypes: map[string]Result{},
			Controls:      map[string]ControlResult{},
		}
	}
	connection.Result.SeverityResult[f.Severity]++
	connection.Result.QueryResult[f.Result]++
	r.Connections[f.ConnectionID] = connection

	resourceType, ok := r.BenchmarkResult.ResourceTypes[f.ResourceType]
	if !ok {
		resourceType = Result{
			QueryResult:    map[types.ComplianceResult]int{},
			SeverityResult: map[types.FindingSeverity]int{},
			SecurityScore:  0,
		}
	}
	resourceType.SeverityResult[f.Severity]++
	resourceType.QueryResult[f.Result]++
	r.BenchmarkResult.ResourceTypes[f.ResourceType] = resourceType

	connectionResourceType, ok := connection.ResourceTypes[f.ResourceType]
	if !ok {
		connectionResourceType = Result{
			QueryResult:    map[types.ComplianceResult]int{},
			SeverityResult: map[types.FindingSeverity]int{},
			SecurityScore:  0,
		}
	}
	connectionResourceType.SeverityResult[f.Severity]++
	connectionResourceType.QueryResult[f.Result]++
	connection.ResourceTypes[f.ResourceType] = connectionResourceType

	control, ok := r.BenchmarkResult.Controls[f.ControlID]
	if !ok {
		control = ControlResult{
			Passed:            true,
			allResources:      map[string]any{},
			failedResources:   map[string]any{},
			allConnections:    map[string]any{},
			failedConnections: map[string]any{},
		}
	}

	if !f.Result.IsPassed() {
		control.Passed = false

		control.failedResources[f.ResourceID] = struct{}{}
		control.failedConnections[f.ConnectionID] = struct{}{}
	}
	control.allResources[f.ResourceID] = struct{}{}
	control.allConnections[f.ConnectionID] = struct{}{}
	r.BenchmarkResult.Controls[f.ControlID] = control

	connectionControl, ok := connection.Controls[f.ControlID]
	if !ok {
		connectionControl = ControlResult{
			Passed:            true,
			allResources:      map[string]any{},
			failedResources:   map[string]any{},
			allConnections:    map[string]any{},
			failedConnections: map[string]any{},
		}
	}
	if !f.Result.IsPassed() {
		connectionControl.Passed = false
		connectionControl.failedResources[f.ResourceID] = struct{}{}
		connectionControl.failedConnections[f.ConnectionID] = struct{}{}
	}
	connectionControl.allResources[f.ResourceID] = struct{}{}
	connectionControl.allConnections[f.ConnectionID] = struct{}{}
	connection.Controls[f.ControlID] = connectionControl
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
				BenchmarkResult: ResultGroup{
					Result: Result{
						QueryResult:    map[types.ComplianceResult]int{},
						SeverityResult: map[types.FindingSeverity]int{},
						SecurityScore:  0,
					},
					ResourceTypes: map[string]Result{},
					Controls:      map[string]ControlResult{},
				},
				Connections: map[string]ResultGroup{},
			}
		}

		rc.addFinding(f)
		b.ResourceCollections[*f.ResourceCollection] = rc
	}
}

func (r *BenchmarkSummaryResult) summarize() {
	// update security scores
	for controlID, summary := range r.BenchmarkResult.Controls {
		summary.FailedConnectionCount = len(summary.failedConnections)
		summary.TotalConnectionCount = len(summary.allConnections)

		summary.FailedResourcesCount = len(summary.failedResources)
		summary.TotalResourcesCount = len(summary.allResources)

		r.BenchmarkResult.Controls[controlID] = summary
	}

	for resourceType, summary := range r.BenchmarkResult.ResourceTypes {
		total := 0
		for _, count := range summary.QueryResult {
			total += count
		}

		if total > 0 {
			summary.SecurityScore = float64(summary.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
		}

		r.BenchmarkResult.ResourceTypes[resourceType] = summary
	}

	total := 0
	for _, count := range r.BenchmarkResult.Result.QueryResult {
		total += count
	}
	if total > 0 {
		r.BenchmarkResult.Result.SecurityScore = float64(r.BenchmarkResult.Result.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
	}

	for connectionID, summary := range r.Connections {
		for controlID, controlSummary := range summary.Controls {
			controlSummary.FailedConnectionCount = len(controlSummary.failedConnections)
			controlSummary.TotalConnectionCount = len(controlSummary.allConnections)

			controlSummary.FailedResourcesCount = len(controlSummary.failedResources)
			controlSummary.TotalResourcesCount = len(controlSummary.allResources)

			summary.Controls[controlID] = controlSummary
		}

		for resourceType, resourceTypeSummary := range summary.ResourceTypes {
			total := 0
			for _, count := range resourceTypeSummary.QueryResult {
				total += count
			}

			if total > 0 {
				resourceTypeSummary.SecurityScore = float64(resourceTypeSummary.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
			}

			summary.ResourceTypes[resourceType] = resourceTypeSummary
		}

		total := 0
		for _, count := range summary.Result.QueryResult {
			total += count
		}

		if total > 0 {
			summary.Result.SecurityScore = float64(summary.Result.QueryResult[types.ComplianceResultOK]) / float64(total) * 100.0
		}

		r.Connections[connectionID] = summary
	}
}

func (b *BenchmarkSummary) Summarize() {
	b.Connections.summarize()
	for rcId, rc := range b.ResourceCollections {
		rc.summarize()
		b.ResourceCollections[rcId] = rc
	}
}
