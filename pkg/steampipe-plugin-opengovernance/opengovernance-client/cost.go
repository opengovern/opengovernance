package opengovernance_client

import (
	"context"
	"github.com/opengovern/opengovernance/pkg/analytics/es/spend"
	"runtime"
	"time"

	essdk "github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/opengovernance/pkg/steampipe-plugin-opengovernance/opengovernance-sdk/config"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

type ConnectionCostSummaryHit struct {
	ID      string                              `json:"_id"`
	Score   float64                             `json:"_score"`
	Index   string                              `json:"_index"`
	Type    string                              `json:"_type"`
	Version int64                               `json:"_version,omitempty"`
	Source  spend.IntegrationMetricTrendSummary `json:"_source"`
	Sort    []any                               `json:"sort"`
}

type ConnectionCostSummaryHits struct {
	Total essdk.SearchTotal          `json:"total"`
	Hits  []ConnectionCostSummaryHit `json:"hits"`
}

type ConnectionCostSummarySearchResponse struct {
	PitID string                    `json:"pit_id"`
	Hits  ConnectionCostSummaryHits `json:"hits"`
}

type ConnectionCostSummaryPaginator struct {
	paginator *essdk.BaseESPaginator
}

func (k Client) NewConnectionCostSummaryPaginator(filters []essdk.BoolFilter, limit *int64) (ConnectionCostSummaryPaginator, error) {
	paginator, err := essdk.NewPaginator(k.ES.ES(), spend.AnalyticsSpendIntegrationSummaryIndex, filters, limit)
	if err != nil {
		return ConnectionCostSummaryPaginator{}, err
	}

	paginator.UpdatePageSize(100)
	if limit != nil {
		paginator.UpdatePageSize(*limit)
	}

	p := ConnectionCostSummaryPaginator{
		paginator: paginator,
	}

	return p, nil
}

func (p ConnectionCostSummaryPaginator) HasNext() bool {
	return !p.paginator.Done()
}

func (p ConnectionCostSummaryPaginator) Close(ctx context.Context) error {
	return p.paginator.Deallocate(ctx)
}

func (p ConnectionCostSummaryPaginator) NextPage(ctx context.Context) ([]spend.IntegrationMetricTrendSummary, error) {
	var response ConnectionCostSummarySearchResponse
	err := p.paginator.Search(ctx, &response)
	if err != nil {
		return nil, err
	}

	var values []spend.IntegrationMetricTrendSummary
	for _, hit := range response.Hits.Hits {
		values = append(values, hit.Source)
	}

	hits := int64(len(response.Hits.Hits))
	if hits > 0 {
		p.paginator.UpdateState(hits, response.Hits.Hits[hits-1].Sort, response.PitID)
	} else {
		p.paginator.UpdateState(hits, nil, "")
	}

	return values, nil
}

var costColumnMapping = map[string]string{
	"date":         "date",
	"date_epoch":   "date_epoch",
	"month":        "month",
	"year":         "year",
	"metric_id":    "metric_id",
	"metric_name":  "metric_name",
	"period_start": "period_start",
	"period_end":   "period_end",
}

func ListCostSummary(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (any, error) {
	plugin.Logger(ctx).Warn("ListCostSummary", d)
	runtime.GC()
	// create service
	cfg := config.GetConfig(d.Connection)
	ke, err := config.NewClientCached(cfg, d.ConnectionCache, ctx)
	if err != nil {
		plugin.Logger(ctx).Error("ListCostSummary NewClientCached", "error", err)
		return nil, err
	}
	k := Client{ES: ke}

	plugin.Logger(ctx).Warn("ListCostSummary: Building filter", d)
	filters := essdk.BuildFilter(ctx, d.QueryContext, costColumnMapping, "", nil, nil, nil)

	plugin.Logger(ctx).Warn("ListCostSummary: NewConnectionCostSummaryPaginator", d)
	plugin.Logger(ctx).Warn("ListCostSummary: Filters", filters)
	plugin.Logger(ctx).Warn("ListCostSummary: Limit", d.QueryContext.Limit)

	paginator, err := k.NewConnectionCostSummaryPaginator(filters, d.QueryContext.Limit)
	if err != nil {
		plugin.Logger(ctx).Error("ListCostSummary NewConnectionCostSummaryPaginator", "error", err)
		return nil, err
	}

	plugin.Logger(ctx).Warn("ListCostSummary: HasNext", d)
	for paginator.HasNext() {
		plugin.Logger(ctx).Warn("ListCostSummary: Next", d)
		page, err := paginator.NextPage(ctx)
		if err != nil {
			plugin.Logger(ctx).Error("ListCostSummary NextPage", "error", err)
			return nil, err
		}

		plugin.Logger(ctx).Warn("ListCostSummary: Stream", d)
		for _, v := range page {
			plugin.Logger(ctx).Warn("ListCostSummary: Page", v)
			for _, connRes := range v.Integrations {
				row := OpenGovernanceCostTableRow{
					IntegrationID:   connRes.IntegrationID,
					IntegrationName: connRes.IntegrationName,
					IntegrationType: connRes.IntegrationType.String(),
					Date:            v.Date,
					DateEpoch:       v.DateEpoch,
					Month:           v.Month,
					Year:            v.Year,
					MetricID:        v.MetricID,
					MetricName:      v.MetricName,
					CostValue:       connRes.CostValue,
					PeriodStart:     time.UnixMilli(v.PeriodStart),
					PeriodEnd:       time.UnixMilli(v.PeriodEnd),
				}
				d.StreamListItem(ctx, row)
			}
		}
	}

	plugin.Logger(ctx).Warn("ListCostSummary: Done", d)
	return nil, nil
}

type OpenGovernanceCostTableRow struct {
	IntegrationID   string    `json:"integration_id"`
	IntegrationName string    `json:"integration_name"`
	IntegrationType string    `json:"integration_type"`
	Date            string    `json:"date"`
	DateEpoch       int64     `json:"date_epoch"`
	Month           string    `json:"month"`
	Year            string    `json:"year"`
	MetricID        string    `json:"metric_id"`
	MetricName      string    `json:"metric_name"`
	CostValue       float64   `json:"cost_value"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
}
