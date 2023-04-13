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
	JobID       uint                `json:"jobID"`             // Job ID
	InsightID   uint                `json:"insightID"`         // Insight ID
	SourceID    string              `json:"sourceID"`          // Source ID
	ExecutedAt  time.Time           `json:"executedAt"`        // Time of Execution
	Result      int64               `json:"result"`            // Result
	Locations   []string            `json:"locations"`         // Locations
	Connections []InsightConnection `json:"connections"`       // Connections
	Details     *InsightDetail      `json:"details,omitempty"` // Insight Details
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

type InsightTag struct {
	ID    uint   `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
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

type Query struct {
	ID             string    `json:"id"`
	QueryToExecute string    `json:"queryToExecute"`
	Connector      string    `json:"connector"`
	ListOfTables   string    `json:"listOfTables"`
	Engine         string    `json:"engine"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Insight struct {
	ID                    uint                  `json:"id"`                    // Insight Id
	Query                 Query                 `json:"query"`                 // Query
	Category              string                `json:"category"`              // Category
	Provider              source.Type           `json:"provider"`              // Provider
	ShortTitle            string                `json:"shortTitle"`            // Short Title
	LongTitle             string                `json:"longTitle"`             // Long Title
	Description           string                `json:"description"`           // Description
	LogoURL               *string               `json:"logoURL"`               // Logo URL
	Labels                []InsightTag          `json:"labels"`                // List of insight tags
	Links                 []InsightLink         `json:"links"`                 // List of links
	Enabled               bool                  `json:"enabled"`               // Enabled
	ExecutedAt            *time.Time            `json:"executedAt,omitempty"`  // Time of Execution
	TotalResults          int64                 `json:"totalResults"`          // Total Results
	Results               *InsightResult        `json:"results,omitempty"`     // Insight Results and Details
	ListInsightResultType ListInsightResultType `json:"listInsightResultType"` // PeerGroup or Insight
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
	Labels                []InsightTag          `json:"labels"`
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
