package es

import (
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

const (
	ConnectionSummaryIndex = "connection_summary"
)

type ConnectionReportType string

const (
	ServiceSummary                   ConnectionReportType = "ServicePerSource"
	CategorySummary                  ConnectionReportType = "CategoryPerSource"
	ResourceSummary                  ConnectionReportType = "ResourcesPerSource"
	ResourceTypeSummary              ConnectionReportType = "ResourceTypePerSource"
	LocationConnectionSummary        ConnectionReportType = "LocationPerSource"
	ServiceLocationConnectionSummary ConnectionReportType = "ServiceLocationPerSource"
	TrendConnectionSummary           ConnectionReportType = "TrendPerSourceHistory"
)

type ConnectionResourcesSummary struct {
	ScheduleJobID    uint                 `json:"schedule_job_id"`
	SourceID         string               `json:"source_id"`
	SourceType       source.Type          `json:"source_type"`
	SourceJobID      uint                 `json:"source_job_id"`
	DescribedAt      int64                `json:"described_at"`
	ResourceCount    int                  `json:"resource_count"`
	LastDayCount     *int                 `json:"last_day_count"`
	LastWeekCount    *int                 `json:"last_week_count"`
	LastQuarterCount *int                 `json:"last_quarter_count"`
	LastYearCount    *int                 `json:"last_year_count"`
	ReportType       ConnectionReportType `json:"report_type"`
}

func (r ConnectionResourcesSummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.SourceID,
		string(ResourceSummary),
	}, ConnectionSummaryIndex
}

type ConnectionLocationSummary struct {
	ScheduleJobID        uint                 `json:"schedule_job_id"`
	SourceID             string               `json:"source_id"`
	SourceType           source.Type          `json:"source_type"`
	SourceJobID          uint                 `json:"source_job_id"`
	LocationDistribution map[string]int       `json:"location_distribution"`
	ReportType           ConnectionReportType `json:"report_type"`
}

func (r ConnectionLocationSummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.SourceID,
		string(LocationConnectionSummary),
	}, ConnectionSummaryIndex
}

type ConnectionServiceLocationSummary struct {
	ScheduleJobID        uint                 `json:"schedule_job_id"`
	SourceID             string               `json:"source_id"`
	SourceType           source.Type          `json:"source_type"`
	SourceJobID          uint                 `json:"source_job_id"`
	ServiceName          string               `json:"service_name"`
	LocationDistribution map[string]int       `json:"location_distribution"`
	ReportType           ConnectionReportType `json:"report_type"`
}

func (r ConnectionServiceLocationSummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.ServiceName,
		r.SourceID,
		string(ServiceLocationConnectionSummary),
	}, ConnectionSummaryIndex
}

type ConnectionTrendSummary struct {
	ScheduleJobID uint                 `json:"schedule_job_id"`
	SourceID      string               `json:"source_id"`
	SourceType    source.Type          `json:"source_type"`
	SourceJobID   uint                 `json:"source_job_id"`
	DescribedAt   int64                `json:"described_at"`
	ResourceCount int                  `json:"resource_count"`
	ReportType    ConnectionReportType `json:"report_type"`
}

func (r ConnectionTrendSummary) KeysAndIndex() ([]string, string) {
	return []string{
		strconv.FormatInt(r.DescribedAt, 10),
		r.SourceID,
		string(TrendConnectionSummary),
	}, ConnectionSummaryIndex
}

type ConnectionResourceTypeSummary struct {
	ScheduleJobID    uint                 `json:"schedule_job_id"`
	SourceID         string               `json:"source_id"`
	SourceJobID      uint                 `json:"source_job_id"`
	ResourceType     string               `json:"resource_type"`
	SourceType       source.Type          `json:"source_type"`
	ResourceCount    int                  `json:"resource_count"`
	LastDayCount     *int                 `json:"last_day_count"`
	LastWeekCount    *int                 `json:"last_week_count"`
	LastQuarterCount *int                 `json:"last_quarter_count"`
	LastYearCount    *int                 `json:"last_year_count"`
	ReportType       ConnectionReportType `json:"report_type"`
}

func (r ConnectionResourceTypeSummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.ResourceType,
		r.SourceID,
		string(ResourceTypeSummary),
	}, ConnectionSummaryIndex
}

type ConnectionServiceSummary struct {
	ScheduleJobID    uint                 `json:"schedule_job_id"`
	SourceID         string               `json:"source_id"`
	SourceJobID      uint                 `json:"source_job_id"`
	ServiceName      string               `json:"service_name"`
	ResourceType     string               `json:"resource_type"`
	SourceType       source.Type          `json:"source_type"`
	DescribedAt      int64                `json:"described_at"`
	ResourceCount    int                  `json:"resource_count"`
	LastDayCount     *int                 `json:"last_day_count"`
	LastWeekCount    *int                 `json:"last_week_count"`
	LastQuarterCount *int                 `json:"last_quarter_count"`
	LastYearCount    *int                 `json:"last_year_count"`
	ReportType       ConnectionReportType `json:"report_type"`
}

func (r ConnectionServiceSummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.ServiceName,
		r.SourceID,
		string(ServiceSummary),
	}, ConnectionSummaryIndex
}

type ConnectionCategorySummary struct {
	ScheduleJobID    uint                 `json:"schedule_job_id"`
	SourceID         string               `json:"source_id"`
	SourceJobID      uint                 `json:"source_job_id"`
	CategoryName     string               `json:"category_name"`
	ResourceType     string               `json:"resource_type"`
	SourceType       source.Type          `json:"source_type"`
	DescribedAt      int64                `json:"described_at"`
	ResourceCount    int                  `json:"resource_count"`
	LastDayCount     *int                 `json:"last_day_count"`
	LastWeekCount    *int                 `json:"last_week_count"`
	LastQuarterCount *int                 `json:"last_quarter_count"`
	LastYearCount    *int                 `json:"last_year_count"`
	ReportType       ConnectionReportType `json:"report_type"`
}

func (r ConnectionCategorySummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.CategoryName,
		r.SourceID,
		string(CategorySummary),
	}, ConnectionSummaryIndex
}
