package types

import (
	"fmt"
	"github.com/opengovern/opengovernance/pkg/utils"

	"github.com/axiomhq/hyperloglog"
	"github.com/opengovern/opengovernance/pkg/types"
)

type Result struct {
	QueryResult      map[types.ConformanceStatus]int
	SeverityResult   map[types.FindingSeverity]int
	SecurityScore    float64
	CostOptimization *float64 `json:"CostOptimization,omitempty"`
}

func (r Result) IsFullyPassed() bool {
	for status, count := range r.QueryResult {
		if status.IsPassed() {
			continue
		}
		if count > 0 {
			return false
		}
	}
	return true
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

	CostOptimization *float64 `json:"CostOptimization,omitempty"`
}

type BenchmarkSummaryResult struct {
	BenchmarkResult ResultGroup
	Connections     map[string]ResultGroup
}

type BenchmarkSummary struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	BenchmarkID      string
	JobID            uint
	EvaluatedAtEpoch int64

	Connections         BenchmarkSummaryResult
	ResourceCollections map[string]BenchmarkSummaryResult
}

func (b BenchmarkSummary) KeysAndIndex() ([]string, string) {
	return []string{b.BenchmarkID, fmt.Sprintf("%d", b.JobID)}, types.BenchmarkSummaryIndex
}

func (r *BenchmarkSummaryResult) addFinding(finding types.Finding) {
	if !finding.ConformanceStatus.IsPassed() {
		r.BenchmarkResult.Result.SeverityResult[finding.Severity]++
	}
	r.BenchmarkResult.Result.QueryResult[finding.ConformanceStatus]++
	r.BenchmarkResult.Result.CostOptimization = utils.PAdd(r.BenchmarkResult.Result.CostOptimization, finding.CostOptimization)

	connection, ok := r.Connections[finding.ConnectionID]
	if !ok {
		connection = ResultGroup{
			Result: Result{
				QueryResult:    map[types.ConformanceStatus]int{},
				SeverityResult: map[types.FindingSeverity]int{},
				SecurityScore:  0,
			},
			ResourceTypes: map[string]Result{},
			Controls:      map[string]ControlResult{},
		}
	}
	if !finding.ConformanceStatus.IsPassed() {
		connection.Result.SeverityResult[finding.Severity]++
	}
	connection.Result.QueryResult[finding.ConformanceStatus]++
	connection.Result.CostOptimization = utils.PAdd(connection.Result.CostOptimization, finding.CostOptimization)
	r.Connections[finding.ConnectionID] = connection

	resourceType, ok := r.BenchmarkResult.ResourceTypes[finding.ResourceType]
	if !ok {
		resourceType = Result{
			QueryResult:    map[types.ConformanceStatus]int{},
			SeverityResult: map[types.FindingSeverity]int{},
			SecurityScore:  0,
		}
	}
	if !finding.ConformanceStatus.IsPassed() {
		resourceType.SeverityResult[finding.Severity]++
	}
	resourceType.QueryResult[finding.ConformanceStatus]++
	resourceType.CostOptimization = utils.PAdd(resourceType.CostOptimization, finding.CostOptimization)
	r.BenchmarkResult.ResourceTypes[finding.ResourceType] = resourceType

	connectionResourceType, ok := connection.ResourceTypes[finding.ResourceType]
	if !ok {
		connectionResourceType = Result{
			QueryResult:    map[types.ConformanceStatus]int{},
			SeverityResult: map[types.FindingSeverity]int{},
			SecurityScore:  0,
		}
	}
	if !finding.ConformanceStatus.IsPassed() {
		connectionResourceType.SeverityResult[finding.Severity]++
	}
	connectionResourceType.QueryResult[finding.ConformanceStatus]++
	connectionResourceType.CostOptimization = utils.PAdd(connectionResourceType.CostOptimization, finding.CostOptimization)
	connection.ResourceTypes[finding.ResourceType] = connectionResourceType

	control, ok := r.BenchmarkResult.Controls[finding.ControlID]
	if !ok {
		control = ControlResult{
			Passed:            true,
			allResources:      hyperloglog.New16(),
			failedResources:   hyperloglog.New16(),
			allConnections:    hyperloglog.New16(),
			failedConnections: hyperloglog.New16(),
		}
	}

	if !finding.ConformanceStatus.IsPassed() {
		control.Passed = false

		control.failedResources.Insert([]byte(finding.OpenGovernanceResourceID))
		control.failedConnections.Insert([]byte(finding.ConnectionID))
	}
	control.allResources.Insert([]byte(finding.OpenGovernanceResourceID))
	control.allConnections.Insert([]byte(finding.ConnectionID))
	control.CostOptimization = utils.PAdd(control.CostOptimization, finding.CostOptimization)
	r.BenchmarkResult.Controls[finding.ControlID] = control

	connectionControl, ok := connection.Controls[finding.ControlID]
	if !ok {
		connectionControl = ControlResult{
			Passed:            true,
			allResources:      hyperloglog.New16(),
			failedResources:   hyperloglog.New16(),
			allConnections:    hyperloglog.New16(),
			failedConnections: hyperloglog.New16(),
		}
	}
	if !finding.ConformanceStatus.IsPassed() {
		connectionControl.Passed = false
		connectionControl.failedResources.Insert([]byte(finding.OpenGovernanceResourceID))
		connectionControl.failedConnections.Insert([]byte(finding.ConnectionID))
	}
	connectionControl.allResources.Insert([]byte(finding.OpenGovernanceResourceID))
	connectionControl.allConnections.Insert([]byte(finding.ConnectionID))
	connectionControl.CostOptimization = utils.PAdd(connectionControl.CostOptimization, finding.CostOptimization)
	connection.Controls[finding.ControlID] = connectionControl
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
			summary.SecurityScore = float64(summary.QueryResult[types.ConformanceStatusOK]) / float64(total) * 100.0
		}

		r.BenchmarkResult.ResourceTypes[resourceType] = summary
	}

	total := 0
	for _, count := range r.BenchmarkResult.Result.QueryResult {
		total += count
	}
	if total > 0 {
		r.BenchmarkResult.Result.SecurityScore = float64(r.BenchmarkResult.Result.QueryResult[types.ConformanceStatusOK]) / float64(total) * 100.0
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
				resourceTypeSummary.SecurityScore = float64(resourceTypeSummary.QueryResult[types.ConformanceStatusOK]) / float64(total) * 100.0
			}

			summary.ResourceTypes[resourceType] = resourceTypeSummary
		}

		total := 0
		for _, count := range summary.Result.QueryResult {
			total += count
		}

		if total > 0 {
			summary.Result.SecurityScore = float64(summary.Result.QueryResult[types.ConformanceStatusOK]) / float64(total) * 100.0
		}

		r.Connections[connectionID] = summary
	}
}

func (b *BenchmarkSummary) summarize() {
	b.Connections.summarize()
	for rcId, rc := range b.ResourceCollections {
		rc.summarize()
		b.ResourceCollections[rcId] = rc
	}
}
