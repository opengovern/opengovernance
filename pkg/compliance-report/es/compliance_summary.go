package es

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gopkg.in/Shopify/sarama.v1"
)

const (
	esIndexHeader          = "elasticsearch_index"
	CompliancySummaryIndex = "compliance_summary"
)

type CompliancySummaryType string

const (
	CompliancySummaryTypeServiceSummary CompliancySummaryType = "service"
)

type CompliancySummary struct {
	CompliancySummaryType CompliancySummaryType `json:"compliancySummaryType"`
	ReportJobId           uint                  `json:"reportJobID"`
	Provider              source.Type           `json:"provider"`
	DescribedAt           int64                 `json:"describedAt"`
}

func (r CompliancySummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return kafkaMsg(
		hashOf(string(r.CompliancySummaryType), strconv.FormatInt(int64(r.ReportJobId), 10)),
		value,
		CompliancySummaryIndex,
	), nil
}

func (r CompliancySummary) MessageID() string {
	return strconv.FormatInt(int64(r.ReportJobId), 10)
}

type ServiceCompliancySummary struct {
	ServiceName string `json:"serviceName"`

	TotalResources       int     `json:"totalResources"`
	TotalCompliant       int     `json:"totalCompliant"`
	CompliancePercentage float64 `json:"compliancePercentage"`

	CompliancySummary
}

func (r ServiceCompliancySummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return kafkaMsg(
		hashOf(r.ServiceName, string(r.CompliancySummaryType), strconv.FormatInt(int64(r.ReportJobId), 10)),
		value,
		CompliancySummaryIndex,
	), nil
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

func hashOf(strings ...string) string {
	h := sha256.New()
	for _, s := range strings {
		h.Write([]byte(s))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

type ServiceCompliancySummaryQueryResponse struct {
	Hits ServiceCompliancySummaryQueryHits `json:"hits"`
}
type ServiceCompliancySummaryQueryHits struct {
	Total keibi.SearchTotal                  `json:"total"`
	Hits  []ServiceCompliancySummaryQueryHit `json:"hits"`
}
type ServiceCompliancySummaryQueryHit struct {
	ID      string                   `json:"_id"`
	Score   float64                  `json:"_score"`
	Index   string                   `json:"_index"`
	Type    string                   `json:"_type"`
	Version int64                    `json:"_version,omitempty"`
	Source  ServiceCompliancySummary `json:"_source"`
	Sort    []interface{}            `json:"sort"`
}

func ServiceComplianceScoreByProviderQuery(provider source.Type, size int, order string, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"provider": {string(provider)}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"compliancySummaryType": {CompliancySummaryTypeServiceSummary}},
	})

	res["size"] = size
	res["sort"] = []map[string]interface{}{
		{
			"compliancePercentage": order,
		},
		{
			"_id": "asc",
		},
	}
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	return string(b), err
}
