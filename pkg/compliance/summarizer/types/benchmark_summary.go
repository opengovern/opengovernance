package types

import (
	"fmt"
	"github.com/axiomhq/hyperloglog"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"go.uber.org/zap"
	"strings"
)

type Result struct {
	QueryResult    map[types.ConformanceStatus]int
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
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	BenchmarkID      string
	JobID            uint
	EvaluatedAtEpoch int64

	Connections         BenchmarkSummaryResult
	ResourceCollections map[string]BenchmarkSummaryResult

	// caches, these are not marshalled and only used
	ResourceCollectionCache map[string]inventoryApi.ResourceCollection `json:"-"`
	ConnectionCache         map[string]onboardApi.Connection           `json:"-"`
}

func (b BenchmarkSummary) KeysAndIndex() ([]string, string) {
	return []string{b.BenchmarkID, fmt.Sprintf("%d", b.JobID)}, types.BenchmarkSummaryIndex
}

func (r *BenchmarkSummaryResult) addFinding(finding types.Finding) {
	if finding.ConformanceStatus != types.ComplianceResultOK {
		r.BenchmarkResult.Result.SeverityResult[finding.Severity]++
	}
	r.BenchmarkResult.Result.QueryResult[finding.ConformanceStatus]++

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
	if finding.ConformanceStatus != types.ComplianceResultOK {
		connection.Result.SeverityResult[finding.Severity]++
	}
	connection.Result.QueryResult[finding.ConformanceStatus]++
	r.Connections[finding.ConnectionID] = connection

	resourceType, ok := r.BenchmarkResult.ResourceTypes[finding.ResourceType]
	if !ok {
		resourceType = Result{
			QueryResult:    map[types.ConformanceStatus]int{},
			SeverityResult: map[types.FindingSeverity]int{},
			SecurityScore:  0,
		}
	}
	if finding.ConformanceStatus != types.ComplianceResultOK {
		resourceType.SeverityResult[finding.Severity]++
	}
	resourceType.QueryResult[finding.ConformanceStatus]++
	r.BenchmarkResult.ResourceTypes[finding.ResourceType] = resourceType

	connectionResourceType, ok := connection.ResourceTypes[finding.ResourceType]
	if !ok {
		connectionResourceType = Result{
			QueryResult:    map[types.ConformanceStatus]int{},
			SeverityResult: map[types.FindingSeverity]int{},
			SecurityScore:  0,
		}
	}
	if finding.ConformanceStatus != types.ComplianceResultOK {
		connectionResourceType.SeverityResult[finding.Severity]++
	}
	connectionResourceType.QueryResult[finding.ConformanceStatus]++
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

		control.failedResources.Insert([]byte(finding.ResourceID))
		control.failedConnections.Insert([]byte(finding.ConnectionID))
	}
	control.allResources.Insert([]byte(finding.ResourceID))
	control.allConnections.Insert([]byte(finding.ConnectionID))
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
		connectionControl.failedResources.Insert([]byte(finding.ResourceID))
		connectionControl.failedConnections.Insert([]byte(finding.ConnectionID))
	}
	connectionControl.allResources.Insert([]byte(finding.ResourceID))
	connectionControl.allConnections.Insert([]byte(finding.ConnectionID))
	connection.Controls[finding.ControlID] = connectionControl
}

func (b *BenchmarkSummary) AddFinding(logger *zap.Logger,
	finding types.Finding, resource *es.LookupResource) {
	if finding.Severity == "" {
		finding.Severity = types.FindingSeverityNone
	}
	if finding.ConformanceStatus == "" {
		finding.ConformanceStatus = types.ComplianceResultERROR
	}
	if finding.ResourceType == "" {
		finding.ResourceType = "-"
	}

	b.Connections.addFinding(finding)

	if resource == nil {
		logger.Warn("no resource found ignoring resource collection population for this finding",
			zap.String("kaytuResourceId", finding.KaytuResourceID),
			zap.String("resourceId", finding.ResourceID),
			zap.String("resourceType", finding.ResourceType),
			zap.String("connectionId", finding.ConnectionID),
			zap.String("benchmarkId", finding.BenchmarkID),
			zap.String("controlId", finding.ControlID),
		)
		return
	}

	for rcId, rc := range b.ResourceCollectionCache {
		// check if resource is in this resource collection
		isIn := false
		for _, filter := range rc.Filters {
			found := false

			for _, connector := range filter.Connectors {
				if strings.ToLower(connector) == strings.ToLower(finding.Connector.String()) {
					found = true
					break
				}
			}
			if !found && len(filter.Connectors) > 0 {
				continue
			}

			found = false
			for _, resourceType := range filter.ResourceTypes {
				if strings.ToLower(resourceType) == strings.ToLower(finding.ResourceType) {
					found = true
					break
				}
			}
			if !found && len(filter.ResourceTypes) > 0 {
				continue
			}

			found = false
			for _, accountId := range filter.AccountIDs {
				if conn, ok := b.ConnectionCache[strings.ToLower(accountId)]; ok {
					if strings.ToLower(conn.ID.String()) == strings.ToLower(finding.ConnectionID) {
						found = true
						break
					}
				}
			}
			if !found && len(filter.AccountIDs) > 0 {
				continue
			}

			found = false
			for _, region := range filter.Regions {
				if strings.ToLower(region) == strings.ToLower(resource.Location) {
					found = true
					break
				}
			}
			if !found && len(filter.Regions) > 0 {
				continue
			}

			found = false
			for k, v := range filter.Tags {
				k := strings.ToLower(k)
				v := strings.ToLower(v)

				isMatch := false
				for _, resourceTag := range resource.Tags {
					if strings.ToLower(resourceTag.Key) == k {
						if strings.ToLower(resourceTag.Value) == v {
							isMatch = true
							break
						}
					}
				}
				if !isMatch {
					found = false
					break
				}
				found = true
			}

			if !found && len(filter.Tags) > 0 {
				continue
			}

			isIn = true
			break
		}
		if !isIn {
			continue
		}

		benchmarkSummaryRc, ok := b.ResourceCollections[rcId]
		if !ok {
			benchmarkSummaryRc = BenchmarkSummaryResult{
				BenchmarkResult: ResultGroup{
					Result: Result{
						QueryResult:    map[types.ConformanceStatus]int{},
						SeverityResult: map[types.FindingSeverity]int{},
						SecurityScore:  0,
					},
					ResourceTypes: map[string]Result{},
					Controls:      map[string]ControlResult{},
				},
				Connections: map[string]ResultGroup{},
			}
		}
		benchmarkSummaryRc.addFinding(finding)
		b.ResourceCollections[rcId] = benchmarkSummaryRc
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
