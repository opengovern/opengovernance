package es

import (
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

const (
	ConnectionSummaryIndex = "connection_summary"
	CostSummeryIndex       = "cost_summary"
)

type ConnectionReportType string

const (
	ServiceSummary                     ConnectionReportType = "ServicePerSource"
	CategorySummary                    ConnectionReportType = "CategoryPerSource"
	ResourceSummary                    ConnectionReportType = "ResourcesPerSource"
	ResourceTypeSummary                ConnectionReportType = "ResourceTypePerSource"
	LocationConnectionSummary          ConnectionReportType = "LocationPerSource"
	ServiceLocationConnectionSummary   ConnectionReportType = "ServiceLocationPerSource"
	TrendConnectionSummary             ConnectionReportType = "TrendPerSourceHistory"
	ResourceTypeTrendConnectionSummary ConnectionReportType = "ResourceTypeTrendPerSourceHistory"
	CostConnectionSummaryMonthly       ConnectionReportType = "CostPerSource"
	CostConnectionSummaryDaily         ConnectionReportType = "CostPerSourceDaily"
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
	keys := []string{
		r.SourceID,
		string(ResourceSummary),
	}
	if strings.HasSuffix(string(r.ReportType), "History") {
		keys = append(keys, fmt.Sprintf("%d", r.ScheduleJobID))
	}
	return keys, ConnectionSummaryIndex
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
	keys := []string{
		r.SourceID,
		string(LocationConnectionSummary),
	}
	if strings.HasSuffix(string(r.ReportType), "History") {
		keys = append(keys, fmt.Sprintf("%d", r.ScheduleJobID))
	}
	return keys, ConnectionSummaryIndex
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
	keys := []string{
		r.ServiceName,
		r.SourceID,
		string(ServiceLocationConnectionSummary),
	}
	if strings.HasSuffix(string(r.ReportType), "History") {
		keys = append(keys, fmt.Sprintf("%d", r.ScheduleJobID))
	}
	return keys, ConnectionSummaryIndex
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
	keys := []string{
		strconv.FormatInt(r.DescribedAt, 10),
		r.SourceID,
		string(TrendConnectionSummary),
	}
	if strings.HasSuffix(string(r.ReportType), "History") {
		keys = append(keys, fmt.Sprintf("%d", r.ScheduleJobID))
	}
	return keys, ConnectionSummaryIndex
}

type ConnectionResourceTypeTrendSummary struct {
	ScheduleJobID uint                 `json:"schedule_job_id"`
	SourceID      string               `json:"source_id"`
	SourceType    source.Type          `json:"source_type"`
	SourceJobID   uint                 `json:"source_job_id"`
	DescribedAt   int64                `json:"described_at"`
	ResourceType  string               `json:"resource_type"`
	ResourceCount int                  `json:"resource_count"`
	ReportType    ConnectionReportType `json:"report_type"`
}

func (r ConnectionResourceTypeTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		strconv.FormatInt(r.DescribedAt, 10),
		r.SourceID,
		r.ResourceType,
		string(ResourceTypeTrendConnectionSummary),
	}
	if strings.HasSuffix(string(r.ReportType), "History") {
		keys = append(keys, fmt.Sprintf("%d", r.ScheduleJobID))
	}
	return keys, ConnectionSummaryIndex
}

type ConnectionCostSummary struct {
	AccountID     string               `json:"account_id"`
	ScheduleJobID uint                 `json:"schedule_job_id"`
	SourceID      string               `json:"source_id"`
	SourceType    source.Type          `json:"source_type"`
	SourceJobID   uint                 `json:"source_job_id"`
	ResourceType  string               `json:"resource_type"`
	Cost          any                  `json:"cost"`
	PeriodStart   int64                `json:"period_start"`
	PeriodEnd     int64                `json:"period_end"`
	ReportType    ConnectionReportType `json:"report_type"`
}

func (c ConnectionCostSummary) GetCostAndUnit() (float64, string) {
	switch c.ResourceType {
	case "aws::internal::byaccountmonthly":
	case "aws::internal::byaccountdaily":
		costFloat, err := strconv.ParseFloat(c.Cost.(map[string]interface{})["UnblendedCostAmount"].(string), 64)
		if err != nil {
			return 0, ""
		}
		costUnit, ok := c.Cost.(map[string]interface{})["UnblendedCostUnit"]
		if !ok {
			return 0, ""
		}

		return costFloat, costUnit.(string)
	}
	return 0, ""
	return 0, ""
}

func (c ConnectionCostSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		c.AccountID,
		c.SourceID,
		c.ResourceType,
		fmt.Sprint(c.PeriodStart),
		fmt.Sprint(c.PeriodEnd),
	}
	switch c.ResourceType {
	case "aws::internal::byaccountmonthly":
		keys = append(keys, string(CostConnectionSummaryMonthly))
	case "aws::internal::byaccountdaily":
		keys = append(keys, string(CostConnectionSummaryDaily))
	}

	return keys, CostSummeryIndex
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
	keys := []string{
		r.ResourceType,
		r.SourceID,
		string(ResourceTypeSummary),
	}
	if strings.HasSuffix(string(r.ReportType), "History") {
		keys = append(keys, fmt.Sprintf("%d", r.ScheduleJobID))
	}
	return keys, ConnectionSummaryIndex
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
	keys := []string{
		r.ServiceName,
		r.SourceID,
		string(ServiceSummary),
	}
	if strings.HasSuffix(string(r.ReportType), "History") {
		keys = append(keys, fmt.Sprintf("%d", r.ScheduleJobID))
	}
	return keys, ConnectionSummaryIndex
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
	keys := []string{
		r.CategoryName,
		r.SourceID,
		string(CategorySummary),
	}
	if strings.HasSuffix(string(r.ReportType), "History") {
		keys = append(keys, fmt.Sprintf("%d", r.ScheduleJobID))
	}
	return keys, ConnectionSummaryIndex
}
