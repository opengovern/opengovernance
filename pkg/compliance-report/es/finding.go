package es

import (
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"gopkg.in/Shopify/sarama.v1"
)

const (
	FindingsIndex = "findings"
)

type Finding struct {
	ID                 uuid.UUID `json:"id"`
	ReportJobID        uint      `json:"reportJobID"`
	ReportID           uint      `json:"reportID"`
	ResourceID         string    `json:"resourceID"`
	ResourceName       string    `json:"resourceName"`
	ResourceLocation   string    `json:"resourceLocation"`
	SourceID           uuid.UUID `json:"accountID"`
	ControlID          string    `json:"controlID"`
	ParentBenchmarkIDs []string  `json:"parentBenchmarkIDs"`
	Status             string    `json:"status"`
	DescribedAt        int64     `json:"describedAt"`
}

func (r Finding) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return kafkaMsg(
		hashOf(
			r.ResourceID,
			r.SourceID.String(),
			r.ControlID,
			strconv.FormatInt(int64(r.ReportJobID), 10),
		),
		value,
		FindingsIndex,
	), nil
}

func (r Finding) MessageID() string {
	return strconv.FormatInt(int64(r.ReportJobID), 10)
}

type FindingsQueryResponse struct {
	Hits FindingsQueryHits `json:"hits"`
}
type FindingsQueryHits struct {
	Total keibi.SearchTotal  `json:"total"`
	Hits  []FindingsQueryHit `json:"hits"`
}
type FindingsQueryHit struct {
	ID      string        `json:"_id"`
	Score   float64       `json:"_score"`
	Index   string        `json:"_index"`
	Type    string        `json:"_type"`
	Version int64         `json:"_version,omitempty"`
	Source  Finding       `json:"_source"`
	Sort    []interface{} `json:"sort"`
}

func FindingsByBenchmarkID(benchmarkID string, status *string, reportID uint, size int, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"parentBenchmarkIDs": {benchmarkID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"reportID": {reportID}},
	})
	if status != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"status": {*status}},
		})
	}
	res["size"] = size
	res["sort"] = []map[string]interface{}{
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

func FindingsByPolicyID(benchmarkID string, policyID string, reportID uint, size int, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"parentBenchmarkIDs": {benchmarkID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"reportID": {reportID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"controlID": {policyID}},
	})
	res["size"] = size
	res["sort"] = []map[string]interface{}{
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
