package kafka

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gopkg.in/Shopify/sarama.v1"
)

const (
	esIndexHeader          = "elasticsearch_index"
	ConnectionSummaryIndex = "connection_summary"
	ProviderSummaryIndex   = "provider_summary"
)

type ConnectionResourcesSummary struct {
	SummarizerJobID uint `json:"summarizer_job_id"`
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType source.Type `json:"source_type"`
	// SourceJobID is the DescribeSourceJob ID
	SourceJobID uint `json:"source_job_id"`
	// DescribedAt is when the DescribeSourceJob is created
	DescribedAt int64 `json:"described_at"`
	// ResourceCount is total of resources for specified account
	ResourceCount int `json:"resource_count"`
	// LastDayCount number of resources in the category at the same time yesterday
	LastDayCount *int `json:"last_day_count"`
	// LastWeekCount number of resources in the category at the same time a week ago
	LastWeekCount *int `json:"last_week_count"`
	// LastQuarterCount number of resources in the category at the same time a quarter ago
	LastQuarterCount *int `json:"last_quarter_count"`
	// LastYearCount number of resources in the category at the same time a year ago
	LastYearCount *int `json:"last_year_count"`
	// ReportType of document
	ReportType kafka.ResourceSummaryType `json:"report_type"`
}

func (r ConnectionResourcesSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return kafkaMsg(hashOf(r.SourceID, string(r.ReportType)), value, ConnectionSummaryIndex), nil
}

func (r ConnectionResourcesSummary) MessageID() string {
	return r.SourceID
}

type ConnectionServicesSummary struct {
	// ServiceName is service name of the resource
	ServiceName string `json:"service_name"`
	// ResourceType is type of the resource
	ResourceType string `json:"resource_type"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType source.Type `json:"source_type"`
	// DescribedAt is when the ScheduleJob is created
	DescribedAt int64 `json:"described_at"`
	// ResourceCount is total of resources for specified account
	ResourceCount int `json:"resource_count"`
	// LastDayCount number of resources in the category at the same time yesterday
	LastDayCount *int `json:"last_day_count"`
	// LastWeekCount number of resources in the category at the same time a week ago
	LastWeekCount *int `json:"last_week_count"`
	// LastQuarterCount number of resources in the category at the same time a quarter ago
	LastQuarterCount *int `json:"last_quarter_count"`
	// LastYearCount number of resources in the category at the same time a year ago
	LastYearCount *int `json:"last_year_count"`
	// ReportType of document
	ReportType kafka.ResourceSummaryType `json:"report_type"`
}

func (r ConnectionServicesSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return kafkaMsg(hashOf(r.ServiceName, r.SourceType.String(), string(r.ReportType)), value, ConnectionSummaryIndex), nil
}

func (r ConnectionServicesSummary) MessageID() string {
	return r.ServiceName
}

type ConnectionCategoriesSummary struct {
	// CategoryName is category name of the resource
	CategoryName string `json:"category_name"`
	// ResourceType is type of the resource
	ResourceType string `json:"resource_type"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType source.Type `json:"source_type"`
	// DescribedAt is when the ScheduleJob is created
	DescribedAt int64 `json:"described_at"`
	// ResourceCount is total of resources for specified account
	ResourceCount int `json:"resource_count"`
	// LastDayCount number of resources in the category at the same time yesterday
	LastDayCount *int `json:"last_day_count"`
	// LastWeekCount number of resources in the category at the same time a week ago
	LastWeekCount *int `json:"last_week_count"`
	// LastQuarterCount number of resources in the category at the same time a quarter ago
	LastQuarterCount *int `json:"last_quarter_count"`
	// LastYearCount number of resources in the category at the same time a year ago
	LastYearCount *int `json:"last_year_count"`
	// ReportType of document
	ReportType kafka.ResourceSummaryType `json:"report_type"`
}

func (r ConnectionCategoriesSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return kafkaMsg(hashOf(r.CategoryName, r.SourceType.String(), string(r.ReportType)), value, ConnectionSummaryIndex), nil
}

func (r ConnectionCategoriesSummary) MessageID() string {
	return r.CategoryName
}

type ConnectionLocationSummary struct {
	SummarizerJobID uint `json:"summarizer_job_id"`
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType source.Type `json:"source_type"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// LocationDistribution is total of resources per location for specified account
	LocationDistribution map[string]int `json:"location_distribution"`
	// ReportType of document
	ReportType kafka.ResourceSummaryType `json:"report_type"`
}

func (r ConnectionLocationSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return kafkaMsg(hashOf(r.SourceID, string(r.ReportType)), value, ConnectionSummaryIndex), nil
}

func (r ConnectionLocationSummary) MessageID() string {
	return r.SourceID
}

type ConnectionTrendSummary struct {
	SummarizerJobID uint `json:"summarizer_job_id"`
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType source.Type `json:"source_type"`
	// SourceJobID is the DescribeSourceJob ID
	SourceJobID uint `json:"source_job_id"`
	// DescribedAt is when the DescribeSourceJob is created
	DescribedAt int64 `json:"described_at"`
	// ResourceCount is total of resources for specified account
	ResourceCount int `json:"resource_count"`
	// ReportType of document
	ReportType kafka.ResourceSummaryType `json:"report_type"`
}

func (r ConnectionTrendSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return kafkaMsg(hashOf(r.SourceID, strconv.FormatInt(r.DescribedAt, 10), string(r.ReportType)), value, ConnectionSummaryIndex), nil
}

func (r ConnectionTrendSummary) MessageID() string {
	return fmt.Sprintf("%s [described at %d]", r.SourceID, r.DescribedAt)
}

type ProviderTrendSummary struct {
	SummarizerJobID uint `json:"summarizer_job_id"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType source.Type `json:"source_type"`
	// DescribedAt is when the DescribeSourceJob is created
	DescribedAt int64 `json:"described_at"`
	// ResourceCount is total of resources for specified account
	ResourceCount int `json:"resource_count"`
	// ReportType of document
	ReportType kafka.ResourceSummaryType `json:"report_type"`
}

func (r ProviderTrendSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return kafkaMsg(hashOf(r.SourceType.String(), strconv.FormatInt(r.DescribedAt, 10), string(r.ReportType)), value, ProviderSummaryIndex), nil
}

func (r ProviderTrendSummary) MessageID() string {
	return fmt.Sprintf("%s [described at %d]", r.SourceType, r.DescribedAt)
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

type SummaryDoc interface {
	AsProducerMessage() (*sarama.ProducerMessage, error)
	MessageID() string
}
