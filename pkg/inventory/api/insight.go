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

type InsightConnection struct {
	ConnectionID string `json:"connection_id"`
	OriginalID   string `json:"original_id"`
}

type InsightResult struct {
	JobID       uint                `json:"jobID"`
	InsightID   uint                `json:"insightID"`
	SourceID    string              `json:"sourceID"`
	ExecutedAt  time.Time           `json:"executedAt"`
	Result      int64               `json:"result"`
	Locations   []string            `json:"locations"`
	Connections []InsightConnection `json:"connections"`
	Details     *InsightDetail      `json:"details,omitempty"`
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

type InsightLink struct {
	ID   uint   `json:"id"`
	Text string `json:"text"`
	URI  string `json:"uri"`
}

type ListInsightResultType string

const (
	ListInsightResultTypePeerGroup ListInsightResultType = "peerGroup"
	ListInsightResultTypeInsight   ListInsightResultType = "insight"
)

type ListInsightResult interface {
	GetType() ListInsightResultType
	GetID() uint
}

type Insight struct {
	ID                    uint                  `json:"id"`
	Query                 string                `json:"query"`
	Category              string                `json:"category"`
	Provider              source.Type           `json:"provider"`
	ShortTitle            string                `json:"shortTitle"`
	LongTitle             string                `json:"longTitle"`
	Description           string                `json:"description"`
	LogoURL               *string               `json:"logoURL"`
	Labels                []InsightLabel        `json:"labels"`
	Links                 []InsightLink         `json:"links"`
	Enabled               bool                  `json:"enabled"`
	ExecutedAt            *time.Time            `json:"executedAt,omitempty"`
	TotalResults          int64                 `json:"totalResults"`
	Results               *InsightResult        `json:"results,omitempty"`
	ListInsightResultType ListInsightResultType `json:"listInsightResultType"`
}

func (i Insight) GetType() ListInsightResultType {
	return ListInsightResultTypeInsight
}

func (i Insight) GetID() uint {
	return i.ID
}

type InsightPeerGroup struct {
	ID                    uint                  `json:"id"`
	Category              string                `json:"category"`
	Insights              []Insight             `json:"insights,omitempty"`
	ShortTitle            string                `json:"shortTitle"`
	LongTitle             string                `json:"longTitle"`
	Description           string                `json:"description"`
	LogoURL               *string               `json:"logoURL"`
	Labels                []InsightLabel        `json:"labels"`
	Links                 []InsightLink         `json:"links"`
	TotalResults          int64                 `json:"totalResults"`
	ListInsightResultType ListInsightResultType `json:"listInsightResultType"`
}

func (i InsightPeerGroup) GetType() ListInsightResultType {
	return ListInsightResultTypePeerGroup
}

func (i InsightPeerGroup) GetID() uint {
	return i.ID
}

type InsightResultTrendResponse struct {
	Trend []TrendDataPoint `json:"trend"`
}
