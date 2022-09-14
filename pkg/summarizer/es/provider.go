package es

import (
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

const (
	ProviderSummaryIndex = "provider_summary"
)

type ProviderReportType string

const (
	ServiceProviderSummary      ProviderReportType = "ServicePerProvider"
	CategoryProviderSummary     ProviderReportType = "CategoryPerProvider"
	ResourceTypeProviderSummary ProviderReportType = "ResourceTypePerProvider"
	LocationProviderSummary     ProviderReportType = "LocationPerProvider"
	TrendProviderSummary        ProviderReportType = "TrendPerProviderHistory"
)

type ProviderServiceSummary struct {
	ScheduleJobID    uint               `json:"schedule_job_id"`
	ServiceName      string             `json:"service_name"`
	ResourceType     string             `json:"resource_type"`
	SourceType       source.Type        `json:"source_type"`
	DescribedAt      int64              `json:"described_at"`
	ResourceCount    int                `json:"resource_count"`
	LastDayCount     *int               `json:"last_day_count"`
	LastWeekCount    *int               `json:"last_week_count"`
	LastQuarterCount *int               `json:"last_quarter_count"`
	LastYearCount    *int               `json:"last_year_count"`
	ReportType       ProviderReportType `json:"report_type"`
}

func (r ProviderServiceSummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.ServiceName,
		r.SourceType.String(),
		string(ServiceProviderSummary),
	}, ProviderSummaryIndex
}

type ProviderCategorySummary struct {
	ScheduleJobID    uint               `json:"schedule_job_id"`
	CategoryName     string             `json:"category_name"`
	ResourceType     string             `json:"resource_type"`
	SourceType       source.Type        `json:"source_type"`
	DescribedAt      int64              `json:"described_at"`
	ResourceCount    int                `json:"resource_count"`
	LastDayCount     *int               `json:"last_day_count"`
	LastWeekCount    *int               `json:"last_week_count"`
	LastQuarterCount *int               `json:"last_quarter_count"`
	LastYearCount    *int               `json:"last_year_count"`
	ReportType       ProviderReportType `json:"report_type"`
}

func (r ProviderCategorySummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.CategoryName,
		r.SourceType.String(),
		string(CategoryProviderSummary),
	}, ProviderSummaryIndex
}

type ProviderTrendSummary struct {
	ScheduleJobID uint               `json:"schedule_job_id"`
	SourceType    source.Type        `json:"source_type"`
	DescribedAt   int64              `json:"described_at"`
	ResourceCount int                `json:"resource_count"`
	ReportType    ProviderReportType `json:"report_type"`
}

func (r ProviderTrendSummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.SourceType.String(),
		strconv.FormatInt(r.DescribedAt, 10),
		string(TrendProviderSummary),
	}, ProviderSummaryIndex
}

type ProviderResourceTypeSummary struct {
	ScheduleJobID    uint               `json:"schedule_job_id"`
	ResourceType     string             `json:"resource_type"`
	SourceType       source.Type        `json:"source_type"`
	ResourceCount    int                `json:"resource_count"`
	LastDayCount     *int               `json:"last_day_count"`
	LastWeekCount    *int               `json:"last_week_count"`
	LastQuarterCount *int               `json:"last_quarter_count"`
	LastYearCount    *int               `json:"last_year_count"`
	ReportType       ProviderReportType `json:"report_type"`
}

func (r ProviderResourceTypeSummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.ResourceType,
		r.SourceType.String(),
		string(ResourceTypeProviderSummary),
	}, ProviderSummaryIndex
}

type ProviderLocationSummary struct {
	ScheduleJobID        uint               `json:"schedule_job_id"`
	SourceType           source.Type        `json:"source_type"`
	LocationDistribution map[string]int     `json:"location_distribution"`
	ReportType           ProviderReportType `json:"report_type"`
}

func (r ProviderLocationSummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.SourceType.String(),
		string(LocationProviderSummary),
	}, ProviderSummaryIndex
}
