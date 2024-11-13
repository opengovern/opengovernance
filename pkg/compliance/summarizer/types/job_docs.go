package types

import (
	"fmt"
	"strings"

	"github.com/opengovern/og-util/pkg/es"
	inventoryApi "github.com/opengovern/opengovernance/pkg/inventory/api"
	"github.com/opengovern/opengovernance/pkg/types"
	integrationApi "github.com/opengovern/opengovernance/services/integration/api/models"
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
	IntegrationCache        map[string]integrationApi.Integration      `json:"-"`
}

func (jd *JobDocs) AddComplianceResult(logger *zap.Logger, job Job,
	complianceResult types.ComplianceResult, resource *es.LookupResource,
) {
	if complianceResult.Severity == "" {
		complianceResult.Severity = types.ComplianceResultSeverityNone
	}
	if complianceResult.ComplianceStatus == "" {
		complianceResult.ComplianceStatus = types.ComplianceStatusERROR
	}
	if complianceResult.ResourceType == "" {
		complianceResult.ResourceType = "-"
	}

	if job.BenchmarkID == complianceResult.BenchmarkID {
		jd.BenchmarkSummary.Integrations.addComplianceResult(complianceResult)
	}

	if resource == nil {
		logger.Warn("no resource found ignoring resource collection population for this complianceResult",
			zap.String("platformResourceID", complianceResult.PlatformResourceID),
			zap.String("resourceId", complianceResult.ResourceID),
			zap.String("resourceType", complianceResult.ResourceType),
			zap.String("integrationID", complianceResult.IntegrationID),
			zap.String("benchmarkId", complianceResult.BenchmarkID),
			zap.String("controlId", complianceResult.ControlID),
		)
		return
	}

	if jd.LastResourceIdType == "" {
		jd.LastResourceIdType = fmt.Sprintf("%s-%s", resource.ResourceType, resource.PlatformID)
	} else if jd.LastResourceIdType != fmt.Sprintf("%s-%s", resource.ResourceType, resource.PlatformID) {
		jd.ResourcesFindingsIsDone[jd.LastResourceIdType] = true
		jd.LastResourceIdType = fmt.Sprintf("%s-%s", resource.ResourceType, resource.PlatformID)
	}
	resourceFinding, ok := jd.ResourcesFindings[fmt.Sprintf("%s-%s", resource.ResourceType, resource.PlatformID)]
	if !ok {
		resourceFinding = types.ResourceFinding{
			PlatformResourceID:    resource.PlatformID,
			ResourceType:          resource.ResourceType,
			ResourceName:          resource.ResourceName,
			IntegrationType:       resource.IntegrationType,
			ComplianceResults:     nil,
			ResourceCollection:    nil,
			ResourceCollectionMap: make(map[string]bool),
			JobId:                 job.ID,
			EvaluatedAt:           job.CreatedAt.UnixMilli(),
		}
		jd.ResourcesFindingsIsDone[fmt.Sprintf("%s-%s", resource.ResourceType, resource.PlatformID)] = false
	} else {
		resourceFinding.JobId = job.ID
		resourceFinding.EvaluatedAt = job.CreatedAt.UnixMilli()
	}
	if resourceFinding.ResourceName == "" {
		resourceFinding.ResourceName = resource.ResourceName
	}
	if resourceFinding.ResourceType == "" {
		resourceFinding.ResourceType = resource.ResourceType
	}
	resourceFinding.ComplianceResults = append(resourceFinding.ComplianceResults, complianceResult)

	for rcId, rc := range jd.ResourceCollectionCache {
		// check if resource is in this resource collection
		isIn := false
		for _, filter := range rc.Filters {
			found := false

			for _, integrationType := range filter.Connectors {
				if strings.ToLower(integrationType) == strings.ToLower(complianceResult.IntegrationType.String()) {
					found = true
					break
				}
			}
			if !found && len(filter.Connectors) > 0 {
				continue
			}

			found = false
			for _, resourceType := range filter.ResourceTypes {
				if strings.ToLower(resourceType) == strings.ToLower(complianceResult.ResourceType) {
					found = true
					break
				}
			}
			if !found && len(filter.ResourceTypes) > 0 {
				continue
			}

			found = false
			for _, accountId := range filter.AccountIDs {
				if integration, ok := jd.IntegrationCache[strings.ToLower(accountId)]; ok {
					if strings.ToLower(integration.IntegrationID) == strings.ToLower(complianceResult.IntegrationID) {
						found = true
						break
					}
				}
			}
			if !found && len(filter.AccountIDs) > 0 {
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
		if job.BenchmarkID == complianceResult.BenchmarkID {
			benchmarkSummaryRc, ok := jd.BenchmarkSummary.ResourceCollections[rcId]
			if !ok {
				benchmarkSummaryRc = BenchmarkSummaryResult{
					BenchmarkResult: ResultGroup{
						Result: Result{
							QueryResult:    map[types.ComplianceStatus]int{},
							SeverityResult: map[types.ComplianceResultSeverity]int{},
							SecurityScore:  0,
						},
						ResourceTypes: map[string]Result{},
						Controls:      map[string]ControlResult{},
					},
					Integrations: map[string]ResultGroup{},
				}
			}
			benchmarkSummaryRc.addComplianceResult(complianceResult)
			jd.BenchmarkSummary.ResourceCollections[rcId] = benchmarkSummaryRc
		}
	}

	jd.ResourcesFindings[fmt.Sprintf("%s-%s", resource.ResourceType, resource.PlatformID)] = resourceFinding
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
