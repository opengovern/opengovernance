package kafka

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"gopkg.in/Shopify/sarama.v1"
)

const (
	esIndexHeader = "elasticsearch_index"
	InsightsIndex = "insights"
)

type InsightResourceType string

const (
	InsightResourceHistory = "history"
	InsightResourceLast    = "last"
)

type InsightResource struct {
	// JobID is the ID of the job which produced this resource
	JobID uint `json:"job_id"`
	// QueryID is the ID of steampipe query which has been executed
	QueryID uint `json:"query_id"`
	// SmartQueryID is the ID of smart query id which is connected to this insight
	SmartQueryID uint `json:"smart_query_id"`
	// Query
	Query string `json:"query"`
	// Description
	Description string `json:"description"`
	// Provider
	Provider string `json:"provider"`
	// Category
	Category string `json:"category"`
	// ExecutedAt is when the query is executed
	ExecutedAt int64 `json:"executed_at"`
	// Result of query
	Result int64 `json:"result"`
	// LastDayValue result of the same query last day
	LastDayValue *int64 `json:"last_day_value"`
	// LastWeekValue result of the same query last week
	LastWeekValue *int64 `json:"last_week_value"`
	// LastQuarterValue result of the same query last quarter
	LastQuarterValue *int64 `json:"last_quarter_value"`
	// LastYearValue result of the same query last year
	LastYearValue *int64 `json:"last_year_value"`
	// ResourceType shows which collection of docs this resource belongs to
	ResourceType InsightResourceType `json:"resource_type"`
}

func (r InsightResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	if r.ResourceType == InsightResourceHistory {
		return kafkaMsg(hashOf(string(r.ResourceType), fmt.Sprintf("%d", r.QueryID), fmt.Sprintf("%d", r.JobID)),
			value, InsightsIndex), nil
	}
	return kafkaMsg(hashOf(string(r.ResourceType), fmt.Sprintf("%d", r.QueryID)),
		value, InsightsIndex), nil
}
func (r InsightResource) MessageID() string {
	return fmt.Sprintf("%d", r.QueryID)
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
