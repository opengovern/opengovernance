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
	esIndexHeader          = "elasticsearch_index"
	InventorySummaryIndex  = "inventory_summary"
	SourceResourcesSummary = "source_resources_summary"
)

type KafkaMessage interface {
	AsProducerMessage() (*sarama.ProducerMessage, error)
	MessageID() string
}

type ResourceSummaryType string

const (
	ResourceSummaryTypeResourceGrowthTrend  = "resource_growth_trend"
	ResourceSummaryTypeLocationDistribution = "location_distribution"
	ResourceSummaryTypeLastSummary          = "last_summary"
	ResourceSummaryTypeCompliancyTrend      = "compliancy_trend"
)

type KafkaResourceCompliancyTrendResource struct {
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

func (r KafkaResourceCompliancyTrendResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(uuid.New().String()),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummary),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}
func (r KafkaResourceCompliancyTrendResource) MessageID() string {
	return r.SourceID
}

type KafkaResource struct {
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

func (r KafkaResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(r.ID))

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(fmt.Sprintf("%x", h.Sum(nil))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(utils.ResourceTypeToESIndex(r.ResourceType)),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}
func (r KafkaResource) MessageID() string {
	return r.ID
}

type KafkaLookupResource struct {
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

func (r KafkaLookupResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(r.ResourceID))
	h.Write([]byte(r.SourceType))

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(fmt.Sprintf("%x", h.Sum(nil))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(InventorySummaryIndex),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}
func (r KafkaLookupResource) MessageID() string {
	return r.ResourceID
}

type KafkaSourceResourcesSummary struct {
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

func (r KafkaSourceResourcesSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(uuid.New().String()),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummary),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}
func (r KafkaSourceResourcesSummary) MessageID() string {
	return r.SourceID
}

type KafkaSourceResourcesLastSummary struct {
	KafkaSourceResourcesSummary
}

func (r KafkaSourceResourcesLastSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(r.SourceID))
	h.Write([]byte(r.ReportType))

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(fmt.Sprintf("%x", h.Sum(nil))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummary),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}

type KafkaLocationDistributionResource struct {
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

func (r KafkaLocationDistributionResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(r.SourceID))
	h.Write([]byte(r.ReportType))

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(fmt.Sprintf("%x", h.Sum(nil))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummary),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}
func (r KafkaLocationDistributionResource) MessageID() string {
	return r.SourceID
}
