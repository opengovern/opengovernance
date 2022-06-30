package kafka

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gopkg.in/Shopify/sarama.v1"
)

const (
	esIndexHeader               = "elasticsearch_index"
	InventorySummaryIndex       = "inventory_summary"
	SourceResourcesSummaryIndex = "source_resources_summary"
)

type DescribedResource interface {
	AsProducerMessage() (*sarama.ProducerMessage, error)
	MessageID() string
}

type ResourceSummaryType string

const (
	ResourceSummaryTypeResourceGrowthTrend        = "resource_growth_trend"
	ResourceSummaryTypeLocationDistribution       = "location_distribution"
	ResourceSummaryTypeLastSummary                = "last_summary"
	ResourceSummaryTypeServiceHistorySummary      = "service_history_summary"
	ResourceSummaryTypeLastServiceSummary         = "last_service_summary"
	ResourceSummaryTypeServiceDistributionSummary = "service_distribution_summary"
	ResourceSummaryTypeCategoryHistorySummary     = "category_history_summary"
	ResourceSummaryTypeLastCategorySummary        = "last_category_summary"
	ResourceSummaryTypeCompliancyTrend            = "compliancy_trend"
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

func (r ResourceCompliancyTrendResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return kafkaMsg(uuid.New().String(),
		value, SourceResourcesSummaryIndex), nil
}
func (r ResourceCompliancyTrendResource) MessageID() string {
	return r.SourceID
}

type Resource struct {
	// ID is the globally unique ID of the resource.
	ID string `json:"id"`
	// Description is the description of the resource based on the describe call.
	Description interface{} `json:"description"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType api.SourceType `json:"source_type"`
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
}

func (r Resource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return kafkaMsg(hashOf(r.ID),
		value, ResourceTypeToESIndex(r.ResourceType)), nil
}
func (r Resource) MessageID() string {
	return r.ID
}

type LookupResource struct {
	// ResourceID is the globally unique ID of the resource.
	ResourceID string `json:"resource_id"`
	// Name is the name of the resource.
	Name string `json:"name"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType api.SourceType `json:"source_type"`
	// ResourceType is the type of the resource.
	ResourceType string `json:"resource_type"`
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
	// CreatedAt is when the DescribeSourceJob is created
	CreatedAt int64 `json:"created_at"`
	// IsCommon
	IsCommon bool `json:"is_common"`
}

func (r LookupResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return kafkaMsg(hashOf(r.ResourceID, string(r.SourceType)),
		value, InventorySummaryIndex), nil
}
func (r LookupResource) MessageID() string {
	return r.ResourceID
}

type SourceResourcesSummary struct {
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType api.SourceType `json:"source_type"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// DescribedAt is when the DescribeSourceJob is created
	DescribedAt int64 `json:"described_at"`
	// ResourceCount is total of resources for specified account
	ResourceCount int `json:"resource_count"`
	// ReportType of document
	ReportType ResourceSummaryType `json:"report_type"`
}

func (r SourceResourcesSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return kafkaMsg(uuid.New().String(),
		value, SourceResourcesSummaryIndex), nil
}
func (r SourceResourcesSummary) MessageID() string {
	return r.SourceID
}

type SourceServicesSummary struct {
	// ServiceName is service name of the resource
	ServiceName string `json:"service_name"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType api.SourceType `json:"source_type"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// DescribedAt is when the DescribeSourceJob is created
	DescribedAt int64 `json:"described_at"`
	// ResourceCount is total of resources for specified account
	ResourceCount int `json:"resource_count"`
	// LastDayCount number of resources in the category at the same time yesterday
	LastDayCount int `json:"last_day_count"`
	// LastWeekCount number of resources in the category at the same time a week ago
	LastWeekCount int `json:"last_week_count"`
	// LastQuarterCount number of resources in the category at the same time a quarter ago
	LastQuarterCount int `json:"last_quarter_count"`
	// LastYearCount number of resources in the category at the same time a year ago
	LastYearCount int `json:"last_year_count"`
	// ReportType of document
	ReportType ResourceSummaryType `json:"report_type"`
}

func (r SourceServicesSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	if r.ReportType == ResourceSummaryTypeCategoryHistorySummary {
		return kafkaMsg(hashOf(r.ServiceName, fmt.Sprintf("%d", r.SourceJobID), string(r.ReportType)),
			value, SourceResourcesSummaryIndex), nil
	}

	return kafkaMsg(hashOf(r.ServiceName, string(r.ReportType)),
		value, SourceResourcesSummaryIndex), nil
}
func (r SourceServicesSummary) MessageID() string {
	return r.ServiceName
}

type SourceCategorySummary struct {
	// CategoryName is category name of the resource
	CategoryName string `json:"category_name"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType api.SourceType `json:"source_type"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// DescribedAt is when the DescribeSourceJob is created
	DescribedAt int64 `json:"described_at"`
	// ResourceCount is total of resources for specified account
	ResourceCount int `json:"resource_count"`
	// LastDayCount number of resources in the category at the same time yesterday
	LastDayCount int `json:"last_day_count"`
	// LastWeekCount number of resources in the category at the same time a week ago
	LastWeekCount int `json:"last_week_count"`
	// LastQuarterCount number of resources in the category at the same time a quarter ago
	LastQuarterCount int `json:"last_quarter_count"`
	// LastYearCount number of resources in the category at the same time a year ago
	LastYearCount int `json:"last_year_count"`
	// ReportType of document
	ReportType ResourceSummaryType `json:"report_type"`
}

func (r SourceCategorySummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	if r.ReportType == ResourceSummaryTypeCategoryHistorySummary {
		return kafkaMsg(hashOf(r.CategoryName, fmt.Sprintf("%d", r.SourceJobID), string(r.ReportType)),
			value, SourceResourcesSummaryIndex), nil
	}

	return kafkaMsg(hashOf(r.CategoryName, string(r.ReportType)),
		value, SourceResourcesSummaryIndex), nil
}
func (r SourceCategorySummary) MessageID() string {
	return r.CategoryName
}

type SourceResourcesLastSummary struct {
	SourceResourcesSummary
}

func (r SourceResourcesLastSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return kafkaMsg(hashOf(r.SourceID, string(r.ReportType)),
		value, SourceResourcesSummaryIndex), nil
}

type LocationDistributionResource struct {
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType api.SourceType `json:"source_type"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// LocationDistribution is total of resources per location for specified account
	LocationDistribution map[string]int `json:"location_distribution"`
	// ReportType of document
	ReportType ResourceSummaryType `json:"report_type"`
}

func (r LocationDistributionResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return kafkaMsg(hashOf(r.SourceID, string(r.ReportType)),
		value, SourceResourcesSummaryIndex), nil
}
func (r LocationDistributionResource) MessageID() string {
	return r.SourceID
}

type SourceServiceDistributionResource struct {
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// ServiceName is name of service
	ServiceName string `json:"service_name"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType api.SourceType `json:"source_type"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// LocationDistribution is total of resources per location for specified account
	LocationDistribution map[string]int `json:"location_distribution"`
	// ReportType of document
	ReportType ResourceSummaryType `json:"report_type"`
}

func (r SourceServiceDistributionResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return kafkaMsg(hashOf(r.SourceID, r.ServiceName, string(r.ReportType)),
		value, SourceResourcesSummaryIndex), nil
}
func (r SourceServiceDistributionResource) MessageID() string {
	return r.ServiceName
}

func hashOf(strings ...string) string {
	h := sha256.New()
	for _, s := range strings {
		h.Write([]byte(s))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func kafkaMsg(key string, value []byte, index string) *sarama.ProducerMessage {
	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(key),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(index),
			},
		},
		Value: sarama.ByteEncoder(value),
	}
}

var stopWordsRe = regexp.MustCompile(`\W+`)

func ResourceTypeToESIndex(t string) string {
	t = stopWordsRe.ReplaceAllString(t, "_")
	return strings.ToLower(t)
}
