package es

import (
	"fmt"
	"strings"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

const (
	InsightsIndex                    = "insights"
	ResourceCollectionsInsightsIndex = "rc_insights"
	StacksInsightsIndex              = "stacks-insights"
)

type InsightResourceType string

const (
	InsightResourceProviderHistory InsightResourceType = "provider_history"
	InsightResourceProviderLast    InsightResourceType = "provider_last"
)

type InsightConnection struct {
	ConnectionID string `json:"connection_id"`
	OriginalID   string `json:"original_id"`
}

type InsightResource struct {
	// JobID is the ID of the job which produced this resource
	JobID uint `json:"job_id"`
	// InsightID is the ID of insight which has been executed
	InsightID uint `json:"insight_id"`
	// Query
	Query string `json:"query"`
	// Internal hidden to user
	Internal bool `json:"internal"`
	// Description
	Description string `json:"description"`
	// SourceID
	SourceID string `json:"source_id"`
	// AccountID
	AccountID string `json:"account_id"`
	// Provider
	Provider source.Type `json:"provider"`
	// ExecutedAt is when the query is executed
	ExecutedAt int64 `json:"executed_at"`
	// Result of query
	Result int64 `json:"result"`
	// ResourceType shows which collection of docs this resource belongs to
	ResourceType InsightResourceType `json:"resource_type"`
	// Locations list of the locations of the resources included in this insight
	Locations []string `json:"locations,omitempty"`
	// IncludedConnections list of the connections ids of the resources included in this insight
	IncludedConnections []InsightConnection `json:"included_connections,omitempty"`
	// PerConnectionCount is the count of insight results per connection
	PerConnectionCount map[string]int64 `json:"per_connection_count,omitempty"`

	S3Location string `json:"s3_location"`

	ResourceCollection *string `json:"resource_collection,omitempty"`
}

func (r InsightResource) KeysAndIndex() ([]string, string) {
	keys := []string{
		string(r.ResourceType),
		fmt.Sprintf("%d", r.InsightID),
		fmt.Sprintf("%s", r.SourceID),
	}
	if strings.HasSuffix(strings.ToLower(string(r.ResourceType)), "history") {
		keys = append(keys, fmt.Sprintf("%d", r.JobID))
	}
	idx := InsightsIndex
	if r.ResourceCollection != nil {
		keys = append(keys, *r.ResourceCollection)
		idx = ResourceCollectionsInsightsIndex
	}

	if strings.HasPrefix(r.SourceID, "stack-") {
		idx = StacksInsightsIndex
	}

	return keys, idx
}
