package api

import (
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type ListInsightResultsRequest struct {
	Provider   *source.Type `json:"provider"`
	SourceID   *string      `json:"sourceID"`
	ExecutedAt *int64       `json:"executedAt"`
}

type ListInsightResultsResponse struct {
	Results []InsightResult `json:"results"`
}

type InsightResult struct {
	JobID      uint      `json:"jobID"`
	InsightID  uint      `json:"insightID"`
	SourceID   string    `json:"sourceID"`
	ExecutedAt time.Time `json:"executedAt"`
	Result     int64     `json:"result"`
}

type InsightDetail struct {
	Headers []string `json:"headers"`
	Rows    [][]any  `json:"rows"`
}

type GetInsightResultTrendRequest struct {
	QueryID  uint         `json:"queryID"`
	Provider *source.Type `json:"provider"`
	SourceID *string      `json:"sourceID"`
}

type GetInsightResultTrendResponse struct {
	Trend []TrendDataPoint `json:"trend"`
}

type InsightLabel struct {
	ID    uint   `json:"id"`
	Label string `json:"label"`
}

type Insight struct {
	ID          uint           `json:"id"`
	Query       string         `json:"query"`
	Category    string         `json:"category"`
	Provider    source.Type    `json:"provider"`
	ShortTitle  string         `json:"shortTitle"`
	LongTitle   string         `json:"longTitle"`
	Description string         `json:"description"`
	LogoURL     *string        `json:"logoURL"`
	Labels      []InsightLabel `json:"labels"`
	Enabled     bool           `json:"enabled"`

	TotalResults int64           `json:"totalResults"`
	Results      []InsightResult `json:"results,omitempty"`
}
