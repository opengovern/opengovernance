package kafka

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/utils"
	"gopkg.in/Shopify/sarama.v1"
)

const (
	esIndexHeader               = "elasticsearch_index"
	InventorySummaryIndex       = "inventory_summary"
	SourceResourcesSummaryIndex = "source_resources_summary"
)

type Message interface {
	AsProducerMessage() (*sarama.ProducerMessage, error)
	MessageID() string
}

type ResourceSummaryType string

const (
	ResourceSummaryTypeResourceGrowthTrend  = "resource_growth_trend"
	ResourceSummaryTypeLocationDistribution = "location_distribution"
	ResourceSummaryTypeLastSummary          = "last_summary"
	ResourceSummaryTypeLastServiceSummary   = "last_service_summary"
	ResourceSummaryTypeCompliancyTrend      = "compliancy_trend"
)

type ResourceCompliancyTrendResource struct {
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType utils.SourceType `json:"source_type"`
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

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(uuid.New().String()),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummaryIndex),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
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

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(hashOf(r.ID)),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(utils.ResourceTypeToESIndex(r.ResourceType)),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
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
}

func (r LookupResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(hashOf(r.ResourceID, string(r.SourceType))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(InventorySummaryIndex),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
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

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(uuid.New().String()),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummaryIndex),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
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
	// ReportType of document
	ReportType ResourceSummaryType `json:"report_type"`
}

func (r SourceServicesSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(hashOf(r.ServiceName, string(r.ReportType))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummaryIndex),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}
func (r SourceServicesSummary) MessageID() string {
	return r.ServiceName
}

type SourceResourcesLastSummary struct {
	SourceResourcesSummary
}

func (r SourceResourcesLastSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(hashOf(r.SourceID, string(r.ReportType))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummaryIndex),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
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

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(hashOf(r.SourceID, string(r.ReportType))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummaryIndex),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}
func (r LocationDistributionResource) MessageID() string {
	return r.SourceID
}

func hashOf(strings ...string) string {
	h := sha256.New()
	for _, s := range strings {
		h.Write([]byte(s))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
