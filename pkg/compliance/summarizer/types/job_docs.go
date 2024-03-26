package types

import (
	"fmt"
	"strings"

	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"go.uber.org/zap"
)

type JobDocs struct {
	BenchmarkSummary  BenchmarkSummary                 `json:"benchmarkSummary"`
	ResourcesFindings map[string]types.ResourceFinding `json:"resourcesFindings"`

	// these are used to track if the resource finding is done so we can remove it from the map and send it to queue to save memory
	ResourcesFindingsIsDone map[string]bool `json:"-"`
	LastResourceIdType      string          `json:"-"`
	// caches, these are not marshalled and only used
	ResourceCollectionCache map[string]inventoryApi.ResourceCollection `json:"-"`
	ConnectionCache         map[string]onboardApi.Connection           `json:"-"`
}

func (jd *JobDocs) AddFinding(logger *zap.Logger, job Job,
	finding types.Finding, resource *es.LookupResource,
) {
	if finding.Severity == "" {
		finding.Severity = types.FindingSeverityNone
	}
	if finding.ConformanceStatus == "" {
		finding.ConformanceStatus = types.ConformanceStatusERROR
	}
	if finding.ResourceType == "" {
		finding.ResourceType = "-"
	}

	if job.BenchmarkID == finding.BenchmarkID {
		jd.BenchmarkSummary.Connections.addFinding(finding)
	}

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

	if jd.LastResourceIdType == "" {
		jd.LastResourceIdType = fmt.Sprintf("%s-%s", resource.ResourceType, resource.ResourceID)
	} else if jd.LastResourceIdType != fmt.Sprintf("%s-%s", resource.ResourceType, resource.ResourceID) {
		jd.ResourcesFindingsIsDone[jd.LastResourceIdType] = true
		jd.LastResourceIdType = fmt.Sprintf("%s-%s", resource.ResourceType, resource.ResourceID)
	}
	resourceFinding, ok := jd.ResourcesFindings[fmt.Sprintf("%s-%s", resource.ResourceType, resource.ResourceID)]
	if !ok {
		resourceFinding = types.ResourceFinding{
			KaytuResourceID:       resource.ResourceID,
			ResourceType:          resource.ResourceType,
			ResourceName:          resource.Name,
			ResourceLocation:      resource.Location,
			Connector:             resource.SourceType,
			Findings:              nil,
			ResourceCollection:    nil,
			ResourceCollectionMap: make(map[string]bool),
			JobId:                 job.ID,
			EvaluatedAt:           job.CreatedAt.UnixMilli(),
		}
		jd.ResourcesFindingsIsDone[fmt.Sprintf("%s-%s", resource.ResourceType, resource.ResourceID)] = false
	}
	if resourceFinding.ResourceName == "" {
		resourceFinding.ResourceName = resource.Name
	}
	if resourceFinding.ResourceLocation == "" {
		resourceFinding.ResourceLocation = resource.Location
	}
	if resourceFinding.ResourceType == "" {
		resourceFinding.ResourceType = resource.ResourceType
	}
	resourceFinding.Findings = append(resourceFinding.Findings, finding)

	for rcId, rc := range jd.ResourceCollectionCache {
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
				if conn, ok := jd.ConnectionCache[strings.ToLower(accountId)]; ok {
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

		resourceFinding.ResourceCollectionMap[rcId] = true
		if job.BenchmarkID == finding.BenchmarkID {
			benchmarkSummaryRc, ok := jd.BenchmarkSummary.ResourceCollections[rcId]
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
			jd.BenchmarkSummary.ResourceCollections[rcId] = benchmarkSummaryRc
		}
	}

	jd.ResourcesFindings[fmt.Sprintf("%s-%s", resource.ResourceType, resource.ResourceID)] = resourceFinding
}

func (jd *JobDocs) SummarizeResourceFinding(logger *zap.Logger, resourceFinding types.ResourceFinding) types.ResourceFinding {
	resourceFinding.ResourceCollection = nil
	for rcId := range resourceFinding.ResourceCollectionMap {
		resourceFinding.ResourceCollection = append(resourceFinding.ResourceCollection, rcId)
	}
	return resourceFinding
}

func (jd *JobDocs) Summarize(logger *zap.Logger) {
	jd.BenchmarkSummary.summarize()
	for i, resourceFinding := range jd.ResourcesFindings {
		resourceFinding := resourceFinding
		for rcId := range resourceFinding.ResourceCollectionMap {
			resourceFinding.ResourceCollection = append(resourceFinding.ResourceCollection, rcId)
			jd.ResourcesFindings[i] = resourceFinding
		}
	}
}
