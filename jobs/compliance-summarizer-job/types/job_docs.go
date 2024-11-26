package types

import (
	"github.com/opengovern/og-util/pkg/es"
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/opencomply/pkg/types"
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

	var platformResourceID, resourceType, resourceName string
	var integrationType integration.Type
	if resource != nil {
		platformResourceID = resource.PlatformID
		resourceType = resource.ResourceType
		integrationType = resource.IntegrationType
		resourceName = resource.ResourceName
	} else {
		platformResourceID = complianceResult.PlatformResourceID
		resourceType = complianceResult.ResourceType
		integrationType = complianceResult.IntegrationType
		resourceName = complianceResult.ResourceName
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
		//return
	}

	if jd.LastResourceIdType == "" {
		jd.LastResourceIdType = platformResourceID
	} else if jd.LastResourceIdType != platformResourceID {
		jd.ResourcesFindingsIsDone[jd.LastResourceIdType] = true
		jd.LastResourceIdType = platformResourceID
	}

	logger.Info("creating the resource finding", zap.String("platform_resource_id", platformResourceID),
		zap.Any("resource", resource))
	resourceFinding, ok := jd.ResourcesFindings[platformResourceID]
	if !ok {
		resourceFinding = types.ResourceFinding{
			PlatformResourceID:    platformResourceID,
			ResourceType:          resourceType,
			ResourceName:          resourceName,
			IntegrationType:       integrationType,
			ComplianceResults:     nil,
			ResourceCollection:    nil,
			ResourceCollectionMap: make(map[string]bool),
			JobId:                 job.ID,
			EvaluatedAt:           job.CreatedAt.UnixMilli(),
		}
		jd.ResourcesFindingsIsDone[platformResourceID] = false
	} else {
		resourceFinding.JobId = job.ID
		resourceFinding.EvaluatedAt = job.CreatedAt.UnixMilli()
	}
	if resourceFinding.ResourceName == "" {
		resourceFinding.ResourceName = resourceName
	}
	if resourceFinding.ResourceType == "" {
		resourceFinding.ResourceType = resourceType
	}
	resourceFinding.ComplianceResults = append(resourceFinding.ComplianceResults, complianceResult)

	if _, ok := jd.ResourcesFindingsIsDone[platformResourceID]; !ok {
		jd.ResourcesFindingsIsDone[platformResourceID] = false
	}
	logger.Info("adding the resource finding", zap.String("platform_resource_id", platformResourceID),
		zap.Any("resource", resource))
	jd.ResourcesFindings[platformResourceID] = resourceFinding
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
