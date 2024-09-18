package types

import "github.com/kaytu-io/open-governance/pkg/metadata/models"

type QueryRunResult struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	RunId       string                  `json:"runID"`
	CreatedBy   string                  `json:"createdBy"`
	TriggeredAt int64                   `json:"triggeredAt"`
	EvaluatedAt int64                   `json:"evaluatedAt"`
	QueryID     string                  `json:"queryID"`
	Parameters  []models.QueryParameter `json:"parameters"`
	ColumnNames []string                `json:"columnNames"`
	Result      [][]any                 `json:"result"`
}

func (r QueryRunResult) KeysAndIndex() ([]string, string) {
	return []string{
		r.RunId,
	}, QueryRunIndex
}
