package es

import (
	"regexp"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/google/uuid"
)

const (
	InventorySummaryIndex       = "inventory_summary"
	SourceResourcesSummaryIndex = "source_resources_summary"
)

type ResourceSummaryType string

const (
	ResourceSummaryTypeCompliancyTrend = "compliancy_trend"
)

type ResourceCompliancyTrendResource struct {
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType source.Type `json:"source_type"`
	// ComplianceJobID is the ID that the Report was created for
	ComplianceJobID uint `json:"compliance_job_id"`
	// CompliantResourceCount is number of resources which is non-compliant
	CompliantResourceCount int `json:"compliant_resource_count"`
	// NonCompliantResourceCount is number of resources which is non-compliant
	NonCompliantResourceCount int `json:"non_compliant_resource_count"`
	// DescribedAt is when the resources is described
	DescribedAt int64 `json:"described_at"`
	// ResourceSummaryType of document
	ResourceSummaryType ResourceSummaryType `json:"report_type"`
}

func (r ResourceCompliancyTrendResource) KeysAndIndex() ([]string, string) {
	return []string{
		uuid.New().String(),
	}, SourceResourcesSummaryIndex
}

type Resource struct {
	// ID is the globally unique ID of the resource.
	ID string `json:"id"`
	// ID is the globally unique ID of the resource.
	ARN string `json:"arn"`
	// Description is the description of the resource based on the describe call.
	Description interface{} `json:"description"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType source.Type `json:"source_type"`
	// ResourceType is the type of the resource.
	ResourceType string `json:"resource_type"`
	// ResourceJobID is the DescribeResourceJob ID that described this resource
	ResourceJobID uint `json:"resource_job_id"`
	// SourceID is the Source ID that the resource belongs to
	SourceID string `json:"source_id"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// Metadata is arbitrary data associated with each resource
	Metadata map[string]string `json:"metadata"`
	// Name is the name of the resource.
	Name string `json:"name"`
	// ResourceGroup is the group of resource (Azure only)
	ResourceGroup string `json:"resource_group"`
	// Location is location/region of the resource
	Location string `json:"location"`
	// ScheduleJobID
	ScheduleJobID uint `json:"schedule_job_id"`
	// CreatedAt is when the DescribeSourceJob is created
	CreatedAt int64 `json:"created_at"`
}

func (r Resource) KeysAndIndex() ([]string, string) {
	return []string{
		r.ID,
	}, ResourceTypeToESIndex(r.ResourceType)
}

type LookupResource struct {
	// ResourceID is the globally unique ID of the resource.
	ResourceID string `json:"resource_id"`
	// Name is the name of the resource.
	Name string `json:"name"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType source.Type `json:"source_type"`
	// ResourceType is the type of the resource.
	ResourceType string `json:"resource_type"`
	// ServiceName is the service of the resource.
	ServiceName string `json:"service_name"`
	// Category is the category of the resource.
	Category string `json:"category"`
	// ResourceGroup is the group of resource (Azure only)
	ResourceGroup string `json:"resource_group"`
	// Location is location/region of the resource
	Location string `json:"location"`
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// ResourceJobID is the DescribeResourceJob ID that described this resource
	ResourceJobID uint `json:"resource_job_id"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// ScheduleJobID
	ScheduleJobID uint `json:"schedule_job_id"`
	// CreatedAt is when the DescribeSourceJob is created
	CreatedAt int64 `json:"created_at"`
	// IsCommon
	IsCommon bool `json:"is_common"`
	// Tags
	Tags map[string]string `json:"tags"`
}

func (r LookupResource) KeysAndIndex() ([]string, string) {
	return []string{
		r.ResourceID,
		string(r.SourceType),
		strings.ToLower(r.ResourceType),
	}, InventorySummaryIndex
}

var stopWordsRe = regexp.MustCompile(`\W+`)

func ResourceTypeToESIndex(t string) string {
	t = stopWordsRe.ReplaceAllString(t, "_")
	return strings.ToLower(t)
}
