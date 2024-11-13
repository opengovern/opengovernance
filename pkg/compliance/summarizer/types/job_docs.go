package types

import (
	"github.com/opengovern/og-util/pkg/es"
	"github.com/opengovern/opengovernance/pkg/types"
	"go.uber.org/zap"
)

type JobDocs struct {
	BenchmarkSummary  BenchmarkSummary                 `json:"benchmarkSummary"`
	ResourcesFindings map[string]types.ResourceFinding `json:"resourcesFindings"`

	// these are used to track if the resource finding is done so we can remove it from the map and send it to queue to save memory
	ResourcesFindingsIsDone map[string]bool `json:"-"`
	LastResourceIdType      string          `json:"-"`
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
		jd.LastResourceIdType = resource.PlatformID
	} else if jd.LastResourceIdType != resource.PlatformID {
		jd.ResourcesFindingsIsDone[jd.LastResourceIdType] = true
		jd.LastResourceIdType = resource.PlatformID
	}

	logger.Info("creating the resource finding", zap.String("platform_resource_id", resource.PlatformID),
		zap.Any("resource", resource))
	resourceFinding, ok := jd.ResourcesFindings[resource.PlatformID]
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
		jd.ResourcesFindingsIsDone[resource.PlatformID] = false
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

	if _, ok := jd.ResourcesFindingsIsDone[resource.PlatformID]; !ok {
		jd.ResourcesFindingsIsDone[resource.PlatformID] = false
	}
	logger.Info("adding the resource finding", zap.String("platform_resource_id", resource.PlatformID),
		zap.Any("resource", resource))
	jd.ResourcesFindings[resource.PlatformID] = resourceFinding
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
