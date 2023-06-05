package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type InsightDetail struct {
	Headers []string `json:"headers"`
	Rows    [][]any  `json:"rows"`
}

type InsightConnection struct {
	ConnectionID string `json:"connection_id"`
	OriginalID   string `json:"original_id"`
}

type InsightResult struct {
	JobID        uint                `json:"jobID"`                 // Job ID
	InsightID    uint                `json:"insightID"`             // Insight ID
	ConnectionID string              `json:"connectionID"`          // Connection ID
	ExecutedAt   time.Time           `json:"executedAt"`            // Time of Execution
	Result       int64               `json:"result"`                // Result
	Locations    []string            `json:"locations"`             // Locations
	Connections  []InsightConnection `json:"connections,omitempty"` // Connections
	Details      *InsightDetail      `json:"details,omitempty"`     // Insight Details
}

type Insight struct {
	ID          uint                `json:"id"`
	Query       Query               `json:"query"`
	Connector   source.Type         `json:"connector"`
	ShortTitle  string              `json:"shortTitle"`
	LongTitle   string              `json:"longTitle"`
	Description string              `json:"description"`
	LogoURL     *string             `json:"logoURL"`
	Tags        map[string][]string `json:"labels"`
	Links       []string            `json:"links"`
	Enabled     bool                `json:"enabled"`
	Internal    bool                `json:"internal"`

	TotalResultValue              *int64          `json:"totalResultValue,omitempty"`
	TotalResultValueChangePercent *float64        `json:"totalResultValueChangePercent,omitempty"`
	Results                       []InsightResult `json:"result,omitempty"`
}

type InsightTrendDatapoint struct {
	Timestamp int `json:"timestamp"` // Time
	Value     int `json:"value"`     // Resource Count
}
