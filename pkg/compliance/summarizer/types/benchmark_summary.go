package types

import (
	"fmt"
	"github.com/axiomhq/hyperloglog"
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

	// these are not exported fields so they are not marshalled
	allResources    *hyperloglog.Sketch
	failedResources *hyperloglog.Sketch

	FailedConnectionCount int
	TotalConnectionCount  int

	// these are not exported fields so they are not marshalled
	allConnections    *hyperloglog.Sketch
	failedConnections *hyperloglog.Sketch
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
			allResources:      hyperloglog.New16(),
			failedResources:   hyperloglog.New16(),
			allConnections:    hyperloglog.New16(),
			failedConnections: hyperloglog.New16(),
		}
	}

	if !f.Result.IsPassed() {
		control.Passed = false

		control.failedResources.Insert([]byte(f.ResourceID))
		control.failedConnections.Insert([]byte(f.ConnectionID))
	}
	control.allResources.Insert([]byte(f.ResourceID))
	control.allConnections.Insert([]byte(f.ConnectionID))
	r.BenchmarkResult.Controls[f.ControlID] = control

	connectionControl, ok := connection.Controls[f.ControlID]
	if !ok {
		connectionControl = ControlResult{
			Passed:            true,
			allResources:      hyperloglog.New16(),
			failedResources:   hyperloglog.New16(),
			allConnections:    hyperloglog.New16(),
			failedConnections: hyperloglog.New16(),
		}
	}
	if !f.Result.IsPassed() {
		connectionControl.Passed = false
		connectionControl.failedResources.Insert([]byte(f.ResourceID))
		connectionControl.failedConnections.Insert([]byte(f.ConnectionID))
	}
	connectionControl.allResources.Insert([]byte(f.ResourceID))
	connectionControl.allConnections.Insert([]byte(f.ConnectionID))
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
		summary.FailedConnectionCount = int(summary.failedConnections.Estimate())
		summary.TotalConnectionCount = int(summary.allConnections.Estimate())

		summary.FailedResourcesCount = int(summary.failedResources.Estimate())
		summary.TotalResourcesCount = int(summary.allResources.Estimate())

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
			controlSummary.FailedConnectionCount = int(controlSummary.failedConnections.Estimate())
			controlSummary.TotalConnectionCount = int(controlSummary.allConnections.Estimate())

			controlSummary.FailedResourcesCount = int(controlSummary.failedResources.Estimate())
			controlSummary.TotalResourcesCount = int(controlSummary.allResources.Estimate())

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
