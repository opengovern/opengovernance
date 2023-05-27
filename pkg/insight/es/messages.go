package es

import (
	"fmt"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

const (
	InsightsIndex = "insights"
)

type InsightResourceType string

const (
	InsightResourceHistory         InsightResourceType = "history"
	InsightResourceLast            InsightResourceType = "last"
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
	// QueryID is the ID of steampipe query which has been executed
	QueryID uint `json:"query_id"`
	// SmartQueryID is the ID of smart query id which is connected to this insight
	SmartQueryID uint `json:"smart_query_id"`
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
	// Category
	Category string `json:"category"`
	// ExecutedAt is when the query is executed
	ExecutedAt int64 `json:"executed_at"`
	// ScheduleUUID
	ScheduleUUID string `json:"schedule_uuid"`
	// Result of query
	Result int64 `json:"result"`
	// LastDayValue result of the same query last day
	LastDayValue *int64 `json:"last_day_value"`
	// LastWeekValue result of the same query last week
	LastWeekValue *int64 `json:"last_week_value"`
	// LastMonthValue result of the same query last week
	LastMonthValue *int64 `json:"last_month_value"`
	// LastQuarterValue result of the same query last quarter
	LastQuarterValue *int64 `json:"last_quarter_value"`
	// LastYearValue result of the same query last year
	LastYearValue *int64 `json:"last_year_value"`
	// ResourceType shows which collection of docs this resource belongs to
	ResourceType InsightResourceType `json:"resource_type"`
	// Locations list of the locations of the resources included in this insight
	Locations []string `json:"locations,omitempty"`
	// IncludedConnections list of the connections ids of the resources included in this insight
	IncludedConnections []InsightConnection `json:"included_connections,omitempty"`

	S3Location string `json:"s3_location"`
}

func (r InsightResource) KeysAndIndex() ([]string, string) {
	keys := []string{
		string(r.ResourceType),
		fmt.Sprintf("%d", r.QueryID),
		fmt.Sprintf("%s", r.SourceID),
	}
	if r.ResourceType == InsightResourceHistory {
		keys = []string{
			string(r.ResourceType),
			fmt.Sprintf("%d", r.QueryID),
			fmt.Sprintf("%s", r.SourceID),
			fmt.Sprintf("%s", r.ScheduleUUID),
		}
	}

	return keys, InsightsIndex
}
