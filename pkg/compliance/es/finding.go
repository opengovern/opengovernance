package es

import (
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

const (
	FindingsIndex = "findings"
)

type Status string

const (
	StatusAlarm = "alarm"
	StatusInfo  = "info"
	StatusOK    = "ok"
	StatusSkip  = "skip"
	StatusError = "error"
)

type Finding struct {
	ComplianceJobID  uint      `json:"complianceJobID"`
	ResourceID       string    `json:"resourceID"`
	ResourceName     string    `json:"resourceName"`
	ResourceLocation string    `json:"resourceLocation"`
	PolicyID         string    `json:"policyID"`
	BenchmarkID      string    `json:"benchmarkID"`
	Reason           string    `json:"reason"`
	Status           Status    `json:"status"`
	DescribedAt      int64     `json:"describedAt"`
	EvaluatedAt      int64     `json:"evaluatedAt"`
	SourceID         uuid.UUID `json:"sourceID"`
}

func (r Finding) KeysAndIndex() ([]string, string) {
	return []string{
		r.ResourceID,
		r.SourceID.String(),
		r.PolicyID,
		strconv.FormatInt(r.DescribedAt, 10),
	}, FindingsIndex
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
