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
	JobID        uint                `json:"jobID" example:"1"`                                           // Job ID
	InsightID    uint                `json:"insightID" example:"23"`                                      // Insight ID
	ConnectionID string              `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Connection ID
	ExecutedAt   time.Time           `json:"executedAt" example:"2023-04-21T08:53:09.928Z"`               // Time of Execution
	Result       int64               `json:"result" example:"1000"`                                       // Result
	Locations    []string            `json:"locations"`                                                   // Locations
	Connections  []InsightConnection `json:"connections,omitempty"`                                       // Connections
	Details      *InsightDetail      `json:"details,omitempty"`                                           // Insight Details
}

type Insight struct {
	ID          uint                `json:"id" example:"23"`                                                                         // Insight ID
	Query       Query               `json:"query"`                                                                                   // Query
	Connector   source.Type         `json:"connector" example:"Azure"`                                                               // Cloud Provider
	ShortTitle  string              `json:"shortTitle" example:"Clusters with no RBAC"`                                              // Short Title
	LongTitle   string              `json:"longTitle" example:"List clusters that have role-based access control (RBAC) disabled"`   // Long Title
	Description string              `json:"description" example:"List clusters that have role-based access control (RBAC) disabled"` // Description
	LogoURL     *string             `json:"logoURL"`                                                                                 // Logo URL
	Tags        map[string][]string `json:"tags"`                                                                                    // Tags
	Links       []string            `json:"links"`                                                                                   // Links
	Enabled     bool                `json:"enabled" example:"true"`
	Internal    bool                `json:"internal" example:"false"`

	TotalResultValue    *int64          `json:"totalResultValue,omitempty" example:"10"`   // Number of Total Result Value
	OldTotalResultValue *int64          `json:"oldTotalResultValue,omitempty" example:"0"` // Number of Old Total Result Value
	Results             []InsightResult `json:"result,omitempty"`                          // Insight Results
}

type InsightGroup struct {
	ID          uint                `json:"id" example:"23"`
	Connectors  []source.Type       `json:"connectors" example:"[\"Azure\", \"AWS\"]"`
	ShortTitle  string              `json:"shortTitle" example:"Clusters with no RBAC"`
	LongTitle   string              `json:"longTitle" example:"List clusters that have role-based access control (RBAC) disabled"`
	Description string              `json:"description" example:"List clusters that have role-based access control (RBAC) disabled"`
	LogoURL     *string             `json:"logoURL" example:"https://kaytu.io/logo.png"`
	Tags        map[string][]string `json:"tags"`

	Insights map[uint]Insight `json:"insights"`

	TotalResultValue    *int64 `json:"totalResultValue,omitempty" example:"10"`
	OldTotalResultValue *int64 `json:"oldTotalResultValue,omitempty" example:"0"`
}

type InsightTrendDatapoint struct {
	Timestamp int `json:"timestamp" example:"1686346668"` // Time
	Value     int `json:"value" example:"1000"`           // Resource Count
}

type InsightGroupTrendResponse struct {
	Trend           []InsightTrendDatapoint          `json:"trend"`
	TrendPerInsight map[uint][]InsightTrendDatapoint `json:"trendPerInsight"`
}
