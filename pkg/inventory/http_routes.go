package inventory

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/internal"
	"gitlab.com/keibiengine/keibi-engine/pkg/utils"

	api3 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"

	insight "gitlab.com/keibiengine/keibi-engine/pkg/insight/es"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

const EsFetchPageSize = 10000
const ApiDefaultPageSize = 20
const DefaultCurrency = "USD"
const InventorySummaryIndex = "inventory_summary"

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")
	v2 := e.Group("/api/v2")

	v1.GET("/locations/:provider", httpserver.AuthorizeHandler(h.GetLocations, api3.ViewerRole))

	v1.POST("/resources", httpserver.AuthorizeHandler(h.GetAllResources, api3.ViewerRole))
	v1.POST("/resources/azure", httpserver.AuthorizeHandler(h.GetAzureResources, api3.ViewerRole))
	v1.POST("/resources/aws", httpserver.AuthorizeHandler(h.GetAWSResources, api3.ViewerRole))
	v1.GET("/resources/count", httpserver.AuthorizeHandler(h.CountResources, api3.ViewerRole))

	v1.POST("/resources/filters", httpserver.AuthorizeHandler(h.GetResourcesFilters, api3.ViewerRole))

	v1.POST("/resource", httpserver.AuthorizeHandler(h.GetResource, api3.ViewerRole))

	v1.GET("/resources/trend", httpserver.AuthorizeHandler(h.GetResourceGrowthTrend, api3.ViewerRole))
	v1.GET("/resources/top/growing/accounts", httpserver.AuthorizeHandler(h.GetTopFastestGrowingAccountsByResourceCount, api3.ViewerRole))
	v1.GET("/resources/top/accounts", httpserver.AuthorizeHandler(h.GetTopAccountsByResourceCount, api3.ViewerRole))
	v1.GET("/resources/top/regions", httpserver.AuthorizeHandler(h.GetTopRegionsByResourceCount, api3.ViewerRole))
	v1.GET("/resources/top/services", httpserver.AuthorizeHandler(h.GetTopServicesByResourceCount, api3.ViewerRole))
	v2.GET("/resources/categories", httpserver.AuthorizeHandler(h.GetCategoriesV2, api3.ViewerRole))
	v2.GET("/resources/rootTemplates", httpserver.AuthorizeHandler(h.GetRootTemplates, api3.ViewerRole))
	v2.GET("/resources/rootCloudProviders", httpserver.AuthorizeHandler(h.GetRootCloudProviders, api3.ViewerRole))

	v2.GET("/resources/category", httpserver.AuthorizeHandler(h.GetCategoryNodeResourceCount, api3.ViewerRole))
	v2.GET("/cost/category", httpserver.AuthorizeHandler(h.GetCategoryNodeCost, api3.ViewerRole))
	v2.GET("/resources/composition", httpserver.AuthorizeHandler(h.GetCategoryNodeResourceCountComposition, api3.ViewerRole))
	v2.GET("/cost/composition", httpserver.AuthorizeHandler(h.GetCategoryNodeCostComposition, api3.ViewerRole))
	v2.GET("/resources/trend", httpserver.AuthorizeHandler(h.GetResourceGrowthTrendV2, api3.ViewerRole))
	v2.GET("/cost/trend", httpserver.AuthorizeHandler(h.GetCostGrowthTrendV2, api3.ViewerRole))

	v1.GET("/accounts/resource/count", httpserver.AuthorizeHandler(h.GetAccountsResourceCount, api3.ViewerRole))
	v2.GET("/accounts/summary", httpserver.AuthorizeHandler(h.GetAccountSummary, api3.ViewerRole))

	v1.GET("/resources/distribution", httpserver.AuthorizeHandler(h.GetResourceDistribution, api3.ViewerRole))
	v1.GET("/services/distribution", httpserver.AuthorizeHandler(h.GetServiceDistribution, api3.ViewerRole))
	v2.GET("/services/summary", httpserver.AuthorizeHandler(h.GetServiceSummary, api3.ViewerRole))

	v1.GET("/cost/top/accounts", httpserver.AuthorizeHandler(h.GetTopAccountsByCost, api3.ViewerRole))
	v1.GET("/cost/top/services", httpserver.AuthorizeHandler(h.GetTopServicesByCost, api3.ViewerRole))

	v1.GET("/query", httpserver.AuthorizeHandler(h.ListQueries, api3.ViewerRole))
	v1.GET("/query/count", httpserver.AuthorizeHandler(h.CountQueries, api3.ViewerRole))
	v1.POST("/query/:queryId", httpserver.AuthorizeHandler(h.RunQuery, api3.EditorRole))

	v1.GET("/insight/results", httpserver.AuthorizeHandler(h.ListInsightsResults, api3.ViewerRole))

	v1.GET("/metrics/summary", httpserver.AuthorizeHandler(h.GetSummaryMetrics, api3.ViewerRole))
	v1.GET("/metrics/categorized", httpserver.AuthorizeHandler(h.GetCategorizedMetrics, api3.ViewerRole))
	v1.GET("/categories", httpserver.AuthorizeHandler(h.ListCategories, api3.ViewerRole))

	v2.GET("/metrics/categorized", httpserver.AuthorizeHandler(h.GetCategorizedMetricsV2, api3.ViewerRole))
	v2.GET("/categories", httpserver.AuthorizeHandler(h.GetCategoriesV2, api3.ViewerRole))

	v2.GET("/metrics/resources/metric", httpserver.AuthorizeHandler(h.GetMetricsResourceCount, api3.ViewerRole))
	v2.GET("/metrics/cost/metric", httpserver.AuthorizeHandler(h.GetMetricsCost, api3.ViewerRole))
	v2.GET("/metrics/resources/composition", httpserver.AuthorizeHandler(h.GetMetricsResourceCountComposition, api3.ViewerRole))
	v2.GET("/metrics/cost/composition", httpserver.AuthorizeHandler(h.GetMetricsCostComposition, api3.ViewerRole))

	v1.GET("/connection/:connection_id/summary", httpserver.AuthorizeHandler(h.GetConnectionSummary, api3.ViewerRole))
	v1.GET("/provider/:provider/summary", httpserver.AuthorizeHandler(h.GetProviderSummary, api3.ViewerRole))
}

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}

// GetResourceGrowthTrend godoc
// @Summary Returns trend of resource growth for specific account
// @Tags    benchmarks
// @Accept  json
// @Produce json
// @Param   sourceId   query    string false "SourceID"
// @Param   provider   query    string false "Provider"
// @Param   timeWindow query    string false "Time Window" Enums(24h,1w,3m,1y,max)
// @Success 200        {object} []api.TrendDataPoint
// @Router  /inventory/api/v1/resources/trend [get]
func (h *HttpHandler) GetResourceGrowthTrend(ctx echo.Context) error {
	var err error
	var fromTime, toTime int64

	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")
	timeWindow := ctx.QueryParam("timeWindow")
	if timeWindow == "" {
		timeWindow = "24h"
	}

	toTime = time.Now().UnixMilli()
	tw, err := ParseTimeWindow(timeWindow)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid timeWindow")
	}
	fromTime = time.Now().Add(-1 * tw).UnixMilli()

	datapoints := map[int64]int{}
	sortMap := []map[string]interface{}{
		{
			"described_at": "asc",
		},
	}
	if sourceID != "" {
		hits, err := es.FetchConnectionTrendSummaryPage(h.client, &sourceID, fromTime, toTime, sortMap, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			datapoints[hit.DescribedAt] += hit.ResourceCount
		}
	} else {
		hits, err := es.FetchProviderTrendSummaryPage(h.client, provider, fromTime, toTime, sortMap, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			datapoints[hit.DescribedAt] += hit.ResourceCount
		}
	}

	var resp []api.TrendDataPoint
	for k, v := range datapoints {
		resp = append(resp, api.TrendDataPoint{
			Timestamp: k,
			Value:     int64(v),
		})
	}
	sort.SliceStable(resp, func(i, j int) bool {
		return resp[i].Timestamp < resp[j].Timestamp
	})
	return ctx.JSON(http.StatusOK, resp)
}

// GetResourceGrowthTrendV2 godoc
// @Summary Returns trend of resource growth for specific account
// @Tags    benchmarks
// @Accept  json
// @Produce json
// @Param   sourceId       query    string false "SourceID"
// @Param   provider       query    string false "Provider"
// @Param   timeWindow     query    string false "Time Window" Enums(24h,1w,3m,1y,max)
// @Param   startTime      query    string true  "start time for chart in epoch seconds"
// @Param   endTime        query    string true  "end time for chart in epoch seconds"
// @Param   category       query    string false "Category(Template) ID defaults to default template"
// @Param   dataPointCount query    int    false "Number of data points to return"
// @Param   importance     query    string false "Filter filters by importance if they have it (array format is supported with , separator | 'all' is also supported)"
// @Success 200            {object} []api.ResourceGrowthTrendResponse
// @Router  /inventory/api/v2/resources/trend [get]
func (h *HttpHandler) GetResourceGrowthTrendV2(ctx echo.Context) error {
	var err error

	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")

	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Now().AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid startTime")
		}
	}
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid endTime")
		}
	}

	importance := strings.ToLower(ctx.QueryParam("importance"))
	if importance == "" {
		importance = "critical"
	}
	importanceArray := strings.Split(importance, ",")

	dataPointCount := 10
	dataPointCountStr := ctx.QueryParam("dataPointCount")
	if dataPointCountStr != "" {
		dataPointCount, err = strconv.Atoi(dataPointCountStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid dataPointCount")
		}
	}

	category := ctx.QueryParam("category")
	var root *CategoryNode
	if category == "" {
		root, err = h.graphDb.GetPrimaryCategoryRootByName(ctx.Request().Context(), RootTypeTemplateRoot, DefaultTemplateRootName, importanceArray)
		if err != nil {
			return err
		}
	} else {
		root, err = h.graphDb.GetPrimaryCategory(ctx.Request().Context(), category, importanceArray)
		if err != nil {
			return err
		}
	}
	var resourceTypes []string
	for _, f := range root.SubTreeFilters {
		switch f.GetFilterType() {
		case FilterTypeCloudResourceType:
			filter := f.(*FilterCloudResourceTypeNode)
			resourceTypes = append(resourceTypes, filter.ResourceType)
		}
	}

	for i, sc := range root.Subcategories {
		cat, err := h.graphDb.GetCategory(ctx.Request().Context(), sc.ElementID, importanceArray)
		if err != nil {
			return err
		}
		root.Subcategories[i] = *cat
	}

	sortMap := []map[string]interface{}{
		{
			"described_at": "asc",
		},
	}

	trends := map[string]api.CategoryResourceTrend{}
	mainCategoryTrendsMap := map[int64]api.TrendDataPoint{}
	if sourceID != "" {
		hits, err := es.FetchConnectionResourceTypeTrendSummaryPage(h.client, &sourceID, resourceTypes, time.Unix(startTime, 0).UnixMilli(), time.Unix(endTime, 0).UnixMilli(), sortMap, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			for _, cat := range root.Subcategories {
				for _, f := range cat.SubTreeFilters {
					switch f.GetFilterType() {
					case FilterTypeCloudResourceType:
						filter := f.(*FilterCloudResourceTypeNode)
						if hit.ResourceType != filter.ResourceType {
							continue
						}

						v := trends[cat.ElementID]
						v.Trend = append(v.Trend, api.TrendDataPoint{
							Timestamp: hit.DescribedAt,
							Value:     int64(hit.ResourceCount),
						})
						if v, ok := mainCategoryTrendsMap[hit.DescribedAt]; ok {
							v.Value += int64(hit.ResourceCount)
							mainCategoryTrendsMap[hit.DescribedAt] = v
						} else {
							mainCategoryTrendsMap[hit.DescribedAt] = api.TrendDataPoint{
								Timestamp: hit.DescribedAt,
								Value:     int64(hit.ResourceCount),
							}
						}
						v.Name = cat.Name
						trends[cat.ElementID] = v
					}
				}
			}
		}
	} else {
		hits, err := es.FetchProviderResourceTypeTrendSummaryPage(h.client, provider, resourceTypes, time.Unix(startTime, 0).UnixMilli(), time.Unix(endTime, 0).UnixMilli(), sortMap, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			for _, cat := range root.Subcategories {
				for _, f := range cat.SubTreeFilters {
					switch f.GetFilterType() {
					case FilterTypeCloudResourceType:
						filter := f.(*FilterCloudResourceTypeNode)
						if hit.ResourceType != filter.ResourceType {
							continue
						}

						v := trends[cat.ElementID]
						v.Trend = append(v.Trend, api.TrendDataPoint{
							Timestamp: hit.DescribedAt,
							Value:     int64(hit.ResourceCount),
						})
						if v, ok := mainCategoryTrendsMap[hit.DescribedAt]; ok {
							v.Value += int64(hit.ResourceCount)
							mainCategoryTrendsMap[hit.DescribedAt] = v
						} else {
							mainCategoryTrendsMap[hit.DescribedAt] = api.TrendDataPoint{
								Timestamp: hit.DescribedAt,
								Value:     int64(hit.ResourceCount),
							}
						}
						v.Name = cat.Name
						trends[cat.ElementID] = v
					}
				}
			}
		}
	}

	var subcategoriesTrends []api.CategoryResourceTrend
	for _, v := range trends {
		// aggregate data points in the same category and same timestamp into one data point with the sum of the values
		timeValMap := map[int64]api.TrendDataPoint{}
		for _, trend := range v.Trend {
			if v, ok := timeValMap[trend.Timestamp]; ok {
				v.Value += trend.Value
				timeValMap[trend.Timestamp] = v
			} else {
				timeValMap[trend.Timestamp] = trend
			}
		}
		trendArr := make([]api.TrendDataPoint, 0, len(timeValMap))
		for _, v := range timeValMap {
			trendArr = append(trendArr, v)
		}
		// sort data points by timestamp
		sort.SliceStable(trendArr, func(i, j int) bool {
			return v.Trend[i].Timestamp < v.Trend[j].Timestamp
		})
		// overwrite the trend array with the aggregated and sorted data points
		v.Trend = internal.DownSampleResourceCounts(trendArr, dataPointCount)
		subcategoriesTrends = append(subcategoriesTrends, v)
	}

	mainCategoryTrends := make([]api.TrendDataPoint, 0, len(mainCategoryTrendsMap))
	for _, v := range mainCategoryTrendsMap {
		mainCategoryTrends = append(mainCategoryTrends, v)
	}
	sort.SliceStable(mainCategoryTrends, func(i, j int) bool {
		return mainCategoryTrends[i].Timestamp < mainCategoryTrends[j].Timestamp
	})
	mainCategoryTrends = internal.DownSampleResourceCounts(mainCategoryTrends, dataPointCount)
	return ctx.JSON(http.StatusOK, api.ResourceGrowthTrendResponse{
		CategoryName:  root.Name,
		Trend:         mainCategoryTrends,
		Subcategories: subcategoriesTrends,
	})
}

// GetCostGrowthTrendV2 godoc
// @Summary Returns trend of resource growth for specific account
// @Tags    benchmarks
// @Accept  json
// @Produce json
// @Param   sourceId       query    string false "SourceID"
// @Param   provider       query    string false "Provider"
// @Param   startTime      query    string true  "start time for chart in epoch seconds"
// @Param   endTime        query    string true  "end time for chart in epoch seconds"
// @Param   category       query    string false "Category(Template) ID defaults to default template"
// @Param   dataPointCount query    int    false "Number of data points to return"
// @Success 200            {object} []api.ResourceGrowthTrendResponse
// @Router  /inventory/api/v2/resources/trend [get]
func (h *HttpHandler) GetCostGrowthTrendV2(ctx echo.Context) error {
	var err error

	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")
	sourceIDPtr := &sourceID
	if sourceID == "" {
		sourceIDPtr = nil
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Now().AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid startTime")
		}
	}
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid endTime")
		}
	}

	dataPointCount := 10
	dataPointCountStr := ctx.QueryParam("dataPointCount")
	if dataPointCountStr != "" {
		dataPointCount, err = strconv.Atoi(dataPointCountStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid dataPointCount")
		}
	}

	category := ctx.QueryParam("category")
	var root *CategoryNode
	if category == "" {
		root, err = h.graphDb.GetPrimaryCategoryRootByName(ctx.Request().Context(), RootTypeTemplateRoot, DefaultTemplateRootName, []string{"all"})
		if err != nil {
			return err
		}
	} else {
		root, err = h.graphDb.GetPrimaryCategory(ctx.Request().Context(), category, []string{"all"})
		if err != nil {
			return err
		}
	}
	var serviceNames []string
	for _, f := range root.SubTreeFilters {
		switch f.GetFilterType() {
		case FilterTypeCost:
			filter := f.(*FilterCostNode)
			serviceNames = append(serviceNames, filter.ServiceName)
		}
	}
	if len(serviceNames) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no cost filters found")
	}

	for i, sc := range root.Subcategories {
		cat, err := h.graphDb.GetCategory(ctx.Request().Context(), sc.ElementID, []string{"all"})
		if err != nil {
			return err
		}
		root.Subcategories[i] = *cat
	}

	hits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, sourceIDPtr, &provider, serviceNames, time.Unix(endTime, 0), time.Unix(startTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}

	categories := append(root.Subcategories, *root)

	// Map of category ID to a trends mapped by their units
	trendsMap := map[string]map[string][]api.CostTrendDataPoint{}
	for serviceName, costArray := range hits {
		for _, cat := range categories {
			for _, f := range cat.SubTreeFilters {
				isProcessed := make(map[string]bool)
				switch f.GetFilterType() {
				case FilterTypeCost:
					filter := f.(*FilterCostNode)
					if filter.ServiceName != serviceName {
						continue
					}
					if _, ok := trendsMap[cat.ElementID]; !ok {
						trendsMap[cat.ElementID] = map[string][]api.CostTrendDataPoint{}
					}
					for _, cost := range costArray {
						costVal, costUnit := cost.GetCostAndUnit()
						if _, ok := trendsMap[cat.ElementID][costUnit]; !ok {
							trendsMap[cat.ElementID][costUnit] = []api.CostTrendDataPoint{}
						}
						processKey := fmt.Sprintf("%d---%s", cost.PeriodEnd, costUnit)
						if _, ok := isProcessed[processKey]; !ok {
							trendsMap[cat.ElementID][costUnit] = append(trendsMap[cat.ElementID][costUnit], api.CostTrendDataPoint{
								Timestamp: cost.PeriodEnd,
								Value: api.CostWithUnit{
									Cost: costVal,
									Unit: costUnit,
								},
							})
							isProcessed[processKey] = true
						}
					}
				}
			}
		}
	}

	var subcategoriesTrends []api.CategoryCostTrend
	var mainCategoryTrends map[string][]api.CostTrendDataPoint
	for _, cat := range categories {
		unitedTrendsMap := trendsMap[cat.ElementID]
		// aggregate data points in the same category and same timestamp into one data point with the sum of the values
		timeValMap := map[string]map[int64]api.CostTrendDataPoint{}
		for unit, unitedTrend := range unitedTrendsMap {
			timeValMap[unit] = map[int64]api.CostTrendDataPoint{}
			for _, trend := range unitedTrend {
				if v, ok := timeValMap[unit][trend.Timestamp]; !ok {
					timeValMap[unit][trend.Timestamp] = trend
				} else {
					v.Value.Cost += trend.Value.Cost
					timeValMap[unit][trend.Timestamp] = v
				}
			}
		}

		unitedTrendsMap = map[string][]api.CostTrendDataPoint{}
		for k, v := range timeValMap {
			unitedTrendsMap[k] = []api.CostTrendDataPoint{}
			for _, val := range v {
				unitedTrendsMap[k] = append(unitedTrendsMap[k], val)
			}
		}
		for unit, trends := range unitedTrendsMap {
			sort.Slice(trends, func(i, j int) bool {
				return trends[i].Timestamp < trends[j].Timestamp
			})
			unitedTrendsMap[unit] = trends
		}
		if cat.ElementID != root.ElementID {
			subcategoriesTrends = append(subcategoriesTrends, api.CategoryCostTrend{
				Name:  cat.Name,
				Trend: internal.DownSampleCosts(unitedTrendsMap, dataPointCount),
			})
		} else {
			mainCategoryTrends = internal.DownSampleCosts(unitedTrendsMap, dataPointCount)
		}
	}

	return ctx.JSON(http.StatusOK, api.CostGrowthTrendResponse{
		CategoryName:  root.Name,
		Trend:         mainCategoryTrends,
		Subcategories: subcategoriesTrends,
	})
}

// GetTopAccountsByCost godoc
// @Summary Returns top n accounts of specified provider by cost
// @Tags    cost
// @Accept  json
// @Produce json
// @Param   count    query    int    true "count"
// @Param   provider query    string true "Provider"
// @Success 200      {object} []api.TopAccountCostResponse
// @Router  /inventory/api/v1/cost/top/accounts [get]
func (h *HttpHandler) GetTopAccountsByCost(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}

	if provider != source.CloudAWS {
		return ctx.JSON(http.StatusNotImplemented, nil)
	}

	accountCostMap := map[string]float64{}
	var searchAfter []interface{}
	for {
		query, err := es.FindAWSCostQuery(nil, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response keibi.CostExplorerByAccountMonthlySearchResponse
		err = h.client.Search(context.Background(), "aws_costexplorer_byaccountmonthly", query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			accountId := hit.Source.SourceID
			cost, err := strconv.ParseFloat(*hit.Source.Description.UnblendedCostAmount, 64)
			if err != nil {
				return err
			}

			if v, ok := accountCostMap[accountId]; ok {
				cost += v
			}
			accountCostMap[accountId] = cost

			searchAfter = hit.Sort
		}
	}

	var accountCost []api.TopAccountCostResponse
	for key, value := range accountCostMap {
		src, err := h.onboardClient.GetSource(httpclient.FromEchoContext(ctx), key)
		if err != nil {
			if err.Error() == "source not found" { //source has been deleted
				continue
			}
			return err
		}
		accountCost = append(accountCost, api.TopAccountCostResponse{
			SourceID:               key,
			ProviderConnectionName: src.ConnectionName,
			ProviderConnectionID:   src.ConnectionID,
			Cost:                   value,
		})
	}

	if len(accountCost) > count {
		accountCost = accountCost[:count]
	}
	return ctx.JSON(http.StatusOK, accountCost)
}

// GetTopServicesByCost godoc
// @Summary Returns top n services of specified provider by cost
// @Tags    cost
// @Accept  json
// @Produce json
// @Param   count    query    int    true "count"
// @Param   provider query    string true "Provider"
// @Param   sourceId query    string true "SourceID"
// @Success 200      {object} []api.TopServiceCostResponse
// @Router  /inventory/api/v1/cost/top/services [get]
func (h *HttpHandler) GetTopServicesByCost(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}

	if provider != source.CloudAWS {
		return ctx.JSON(http.StatusNotImplemented, nil)
	}

	var sourceUUID *uuid.UUID
	sourceId := ctx.QueryParam("sourceId")
	if len(sourceId) > 0 {
		suuid, err := uuid.Parse(sourceId)
		if err != nil {
			return err
		}
		sourceUUID = &suuid
	}

	serviceCostMap := map[string]float64{}
	var searchAfter []interface{}
	for {
		query, err := es.FindAWSCostQuery(sourceUUID, EsFetchPageSize, searchAfter)
		if err != nil {
			return err
		}

		var response keibi.CostExplorerByServiceMonthlySearchResponse
		err = h.client.Search(context.Background(), "aws_costexplorer_byservicemonthly", query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			serviceName := *hit.Source.Description.Dimension1
			cost, err := strconv.ParseFloat(*hit.Source.Description.UnblendedCostAmount, 64)
			if err != nil {
				return err
			}

			if v, ok := serviceCostMap[serviceName]; ok {
				cost += v
			}
			serviceCostMap[serviceName] = cost
			searchAfter = hit.Sort
		}
	}

	var serviceCost []api.TopServiceCostResponse
	for key, value := range serviceCostMap {
		serviceCost = append(serviceCost, api.TopServiceCostResponse{
			ServiceName: key,
			Cost:        value,
		})
	}

	if len(serviceCost) > count {
		serviceCost = serviceCost[:count]
	}
	return ctx.JSON(http.StatusOK, serviceCost)
}

// GetTopAccountsByResourceCount godoc
// @Summary Returns top n accounts of specified provider by resource count
// @Tags    benchmarks
// @Accept  json
// @Produce json
// @Param   count    query    int    true "count"
// @Param   provider query    string true "Provider"
// @Success 200      {object} []api.TopAccountResponse
// @Router  /inventory/api/v1/resources/top/accounts [get]
func (h *HttpHandler) GetTopAccountsByResourceCount(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	count := EsFetchPageSize
	countStr := ctx.QueryParam("count")
	if len(countStr) > 0 {
		c, err := strconv.Atoi(countStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
		}
		count = c
	}

	var hits []summarizer.ConnectionResourcesSummary

	srt := []map[string]interface{}{{"resource_count": "desc"}}
	hits, err := es.FetchConnectionResourcesSummaryPage(h.client, provider, nil, srt, count)
	var res []api.TopAccountResponse
	for _, v := range hits {
		res = append(res, api.TopAccountResponse{
			SourceID:      v.SourceID,
			Provider:      string(v.SourceType),
			ResourceCount: v.ResourceCount,
		})
	}

	var sourceIds []string
	for _, r := range res {
		sourceIds = append(sourceIds, r.SourceID)
	}
	srcs, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), sourceIds)
	if err != nil {
		return err
	}

	for idx, r := range res {
		for _, src := range srcs {
			if r.SourceID == src.ID.String() {
				res[idx].ProviderConnectionID = src.ConnectionID
				res[idx].ProviderConnectionName = src.ConnectionName
				break
			}
		}
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetTopFastestGrowingAccountsByResourceCount godoc
// @Summary Returns top n accounts of specified provider by resource count
// @Tags    benchmarks
// @Accept  json
// @Produce json
// @Param   count      query    int    true "count"
// @Param   provider   query    string true "Provider"
// @Param   timeWindow query    string true "TimeWindow" Enums(1d,1w,3m,1y)
// @Success 200        {object} []api.TopAccountResponse
// @Router  /inventory/api/v1/resources/top/growing/accounts [get]
func (h *HttpHandler) GetTopFastestGrowingAccountsByResourceCount(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))

	timeWindow := ctx.QueryParam("timeWindow")
	switch timeWindow {
	case "1d", "1w", "3m", "1y":
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid timeWindow")
	}

	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}

	summaryList, err := es.FetchConnectionResourcesSummaryPage(h.client, provider, nil, nil, EsFetchPageSize)
	if err != nil {
		return err
	}

	sort.Slice(summaryList, func(i, j int) bool {
		var lastValueI, lastValueJ *int
		switch timeWindow {
		case "1d":
			lastValueI = summaryList[i].LastDayCount
			lastValueJ = summaryList[j].LastDayCount
		case "1w":
			lastValueI = summaryList[i].LastWeekCount
			lastValueJ = summaryList[j].LastWeekCount
		case "3m":
			lastValueI = summaryList[i].LastQuarterCount
			lastValueJ = summaryList[j].LastQuarterCount
		case "1y":
			lastValueI = summaryList[i].LastYearCount
			lastValueJ = summaryList[j].LastYearCount
		}

		if zero := 0; lastValueI == nil {
			lastValueI = &zero
		}
		if zero := 0; lastValueJ == nil {
			lastValueJ = &zero
		}

		diffI := summaryList[i].ResourceCount - *lastValueI
		diffJ := summaryList[j].ResourceCount - *lastValueJ

		return diffI > diffJ
	})

	if len(summaryList) > count {
		summaryList = summaryList[:count]
	}

	var sourceIds []string
	for _, r := range summaryList {
		sourceIds = append(sourceIds, r.SourceID)
	}
	srcs, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), sourceIds)
	if err != nil {
		return err
	}

	var res []api.TopAccountResponse
	for _, hit := range summaryList {
		connName := ""
		connID := ""
		for _, src := range srcs {
			if hit.SourceID == src.ID.String() {
				connID = src.ConnectionID
				connName = src.ConnectionName
				break
			}
		}

		res = append(res, api.TopAccountResponse{
			SourceID:               hit.SourceID,
			Provider:               string(hit.SourceType),
			ProviderConnectionName: connName,
			ProviderConnectionID:   connID,
			ResourceCount:          hit.ResourceCount,
		})
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetTopRegionsByResourceCount godoc
// @Summary Returns top n regions of specified provider by resource count
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   count    query    int    true  "count"
// @Param   provider query    string false "Provider"
// @Param   sourceId query    string false "SourceId"
// @Success 200      {object} []api.LocationResponse
// @Router  /inventory/api/v1/resources/top/regions [get]
func (h *HttpHandler) GetTopRegionsByResourceCount(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}

	var sourceID *string
	sourceId := ctx.QueryParam("sourceId")
	if len(sourceId) > 0 {
		sourceID = &sourceId
	}

	locationDistribution := map[string]int{}

	hits, err := es.FetchConnectionLocationsSummaryPage(h.client, provider, sourceID, nil, EsFetchPageSize)
	if err != nil {
		return err
	}
	for _, hit := range hits {
		for k, v := range hit.LocationDistribution {
			locationDistribution[k] += v
		}
	}

	var response []api.LocationResponse
	for region, count := range locationDistribution {
		response = append(response, api.LocationResponse{
			Location:      region,
			ResourceCount: &count,
		})
	}
	sort.Slice(response, func(i, j int) bool {
		return *response[i].ResourceCount > *response[j].ResourceCount
	})
	if len(response) > count {
		response = response[:count]
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetTopServicesByResourceCount godoc
// @Summary Returns top n services of specified provider by resource count
// @Tags    benchmarks
// @Accept  json
// @Produce json
// @Param   count    query    int    true  "count"
// @Param   provider query    string true  "Provider"
// @Param   sourceId query    string false "SourceID"
// @Success 200      {object} []api.TopServicesResponse
// @Router  /inventory/api/v1/resources/top/services [get]
func (h *HttpHandler) GetTopServicesByResourceCount(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))

	var count *int
	countStr := ctx.QueryParam("count")
	if len(countStr) > 0 {
		c, err := strconv.Atoi(countStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
		}
		count = &c
	}

	var sourceID *string
	sID := ctx.QueryParam("sourceId")
	if sID != "" {
		sourceUUID, err := uuid.Parse(sID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid sourceID")
		}
		s := sourceUUID.String()
		sourceID = &s
	}

	res, err := GetServices(h.client, provider, sourceID)
	if err != nil {
		return err
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ResourceCount > res[j].ResourceCount
	})
	if count != nil && len(res) > *count {
		res = res[:*count]
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetCategoriesV2 godoc
// @Summary Return list of the subcategories of the specified category
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   category query    string false "Category ID - defaults to default template category"
// @Success 200      {object} []api.CategoryNode
// @Router  /inventory/api/v2/resources/categories [get]
func (h *HttpHandler) GetCategoriesV2(ctx echo.Context) error {
	category := ctx.QueryParam("category")
	var (
		categoryNode *CategoryNode
		err          error
	)
	if category == "" {
		categoryNode, err = h.graphDb.GetCategoryRootSubcategoriesByName(ctx.Request().Context(), RootTypeTemplateRoot, DefaultTemplateRootName)
		if err != nil {
			return err
		}
	} else {
		categoryNode, err = h.graphDb.GetSubcategories(ctx.Request().Context(), category)
		if err != nil {
			return err
		}
	}

	res := api.CategoryNode{
		CategoryID:    categoryNode.ElementID,
		CategoryName:  categoryNode.Name,
		Subcategories: make([]api.CategoryNode, 0, len(categoryNode.Subcategories)),
	}
	for _, subcategory := range categoryNode.Subcategories {
		res.Subcategories = append(res.Subcategories, api.CategoryNode{
			CategoryID:   subcategory.ElementID,
			CategoryName: subcategory.Name,
		})
	}

	return ctx.JSON(http.StatusOK, res)
}

func (h *HttpHandler) GetCategoryNodeResourceCountHelper(ctx context.Context, depth int, category string, sourceID string, provider source.Type, t int64, importanceArray []string, nodeCacheMap map[string]api.CategoryNode) (*api.CategoryNode, error) {
	var (
		rootNode *CategoryNode
		err      error
	)
	if category == "" {
		rootNode, err = h.graphDb.GetCategoryRootByName(ctx, RootTypeTemplateRoot, DefaultTemplateRootName, importanceArray)
		if err != nil {
			return nil, err
		}
	} else {
		rootNode, err = h.graphDb.GetCategory(ctx, category, importanceArray)
		if err != nil {
			return nil, err
		}
	}
	sourceIDPtr := &sourceID
	if sourceID == "" {
		sourceIDPtr = nil
	}

	resourceTypes := GetResourceTypeListFromFilters(rootNode.SubTreeFilters, provider)

	metricIndexed, err := es.FetchResourceTypeCountAtTime(h.client, provider, sourceIDPtr, time.Unix(t, 0), resourceTypes, EsFetchPageSize)
	if err != nil {
		return nil, err
	}

	result, err := RenderCategoryResourceCountDFS(ctx,
		h.graphDb,
		rootNode,
		metricIndexed,
		depth,
		importanceArray,
		nodeCacheMap,
		map[string]api.Filter{})
	if err != nil {
		return nil, err
	}

	return result, err
}

func (h *HttpHandler) GetMetricsResourceCountHelper(ctx context.Context, category string, sourceID string, provider source.Type, t int64, importanceArray []string) (map[string]api.Filter, error) {
	var (
		filters []Filter
		err     error
	)
	sourceIDPtr := &sourceID
	if sourceID == "" {
		sourceIDPtr = nil
	}
	if category == "" {
		filterType := FilterTypeCloudResourceType
		filtersResourceTypes, err := h.graphDb.GetFilters(ctx, provider, importanceArray, &filterType)
		if err != nil {
			return nil, err
		}
		if sourceIDPtr == nil {
			filterType = FilterTypeInsight
			filtersInsight, err := h.graphDb.GetFilters(ctx, provider, importanceArray, &filterType)
			if err != nil {
				return nil, err
			}
			filters = append(filters, filtersInsight...)
		}
		filters = append(filters, filtersResourceTypes...)
	} else {
		rootNode, err := h.graphDb.GetCategory(ctx, category, importanceArray)
		if err != nil {
			return nil, err
		}
		filters = rootNode.SubTreeFilters
	}

	resourceTypes := GetResourceTypeListFromFilters(filters, provider)
	metricIndexed, err := es.FetchResourceTypeCountAtTime(h.client, provider, sourceIDPtr, time.Unix(t, 0), resourceTypes, EsFetchPageSize)
	if err != nil {
		return nil, err
	}

	insightIndexed := make(map[string]int)
	if sourceIDPtr == nil {
		insightIDs := GetInsightIDListFromFilters(filters, provider)
		insightIndexed, err = es.FetchInsightValueAtTime(h.client, time.Unix(t, 0), insightIDs)
		if err != nil {
			return nil, err
		}
	}

	result := make(map[string]api.Filter)
	for _, filter := range filters {
		switch filter.GetFilterType() {
		case FilterTypeCloudResourceType:
			f := filter.(*FilterCloudResourceTypeNode)
			if _, ok := metricIndexed[f.ResourceType]; !ok {
				continue
			}
			result[f.ElementID] = &api.FilterCloudResourceType{
				FilterType:    api.FilterTypeCloudResourceType,
				FilterID:      f.ElementID,
				CloudProvider: f.CloudProvider,
				ResourceType:  f.ResourceType,
				ResourceName:  f.ResourceName,
				PluralName:    f.PluralName,
				ResourceCount: metricIndexed[f.ResourceType],
			}
		case FilterTypeInsight:
			if sourceIDPtr != nil {
				continue
			}
			f := filter.(*FilterInsightNode)
			if _, ok := insightIndexed[f.InsightID]; !ok {
				continue
			}
			result[f.ElementID] = &api.FilterInsight{
				FilterType:    api.FilterTypeInsight,
				FilterID:      f.ElementID,
				CloudProvider: f.CloudProvider,
				Name:          f.Name,
				PluralName:    f.PluralName,
				Value:         insightIndexed[f.InsightID],
			}
		}
	}
	return result, err
}

// GetCategoryNodeResourceCount godoc
// @Summary Return category info by provided category id, info includes category name, subcategories names and ids and number of resources
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   category   query    string false "Category ID - defaults to default template category"
// @Param   depth      query    int    true  "Depth of rendering subcategories"
// @Param   provider   query    string false "Provider"
// @Param   sourceId   query    string false "SourceID"
// @Param   importance query    string false "Filter filters by importance if they have it (array format is supported with , separator | 'all' is also supported)"
// @Param   time       query    string false "timestamp for resource count in epoch seconds either timeWindow or time must be provided"
// @Param   timeWindow query    string false "time window either this or time must be provided" Enums(1d,1w,1m,3m,1y)
// @Success 200        {object} api.CategoryNode
// @Router  /inventory/api/v2/resources/category [get]
func (h *HttpHandler) GetCategoryNodeResourceCount(ctx echo.Context) error {
	depthStr := ctx.QueryParam("depth")
	depth, err := strconv.Atoi(depthStr)
	if err != nil || depth <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid depth")
	}
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")
	importance := strings.ToLower(ctx.QueryParam("importance"))
	if importance == "" {
		importance = "critical"
	}
	importanceArray := strings.Split(importance, ",")
	category := ctx.QueryParam("category")

	timeStr := ctx.QueryParam("time")
	timeWindowStr := ctx.QueryParam("timeWindow")
	timeVal := time.Now().Unix()

	if timeStr != "" && timeWindowStr != "" {
		return echo.NewHTTPError(http.StatusBadRequest, "only one of time or timeWindow should be provided")
	}
	if timeStr != "" {
		timeVal, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}

	var timeWindow time.Duration
	if timeWindowStr != "" {
		timeWindow, err = ParseTimeWindow(timeWindowStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid timeWindow")
		}
		timeVal = time.Now().Unix()
	}

	result, err := h.GetCategoryNodeResourceCountHelper(ctx.Request().Context(), depth, category, sourceID, provider, timeVal, importanceArray, make(map[string]api.CategoryNode))
	if err != nil {
		return err
	}
	if timeWindowStr != "" {
		nodeCacheMap := make(map[string]api.CategoryNode)
		_, err = h.GetCategoryNodeResourceCountHelper(ctx.Request().Context(), depth, category, sourceID, provider, time.Unix(timeVal, 0).Add(-1*timeWindow).Unix(), importanceArray, nodeCacheMap)
		if err != nil {
			return err
		}
		result = internal.CalculateResourceTypeCountPercentChanges(result, nodeCacheMap)
	}
	return ctx.JSON(http.StatusOK, result)
}

// GetMetricsResourceCount godoc
// @Summary Return category info by provided category id, info includes category name, subcategories names and ids and number of resources
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   category   query    string false "Category ID - defaults to default template category"
// @Param   provider   query    string false "Provider"
// @Param   sourceId   query    string false "SourceID"
// @Param   importance query    string false "Filter filters by importance if they have it (array format is supported with , separator | 'all' is also supported)"
// @Param   time       query    string false "timestamp for resource count in epoch seconds either timeWindow or time must be provided"
// @Param   timeWindow query    string false "time window either this or time must be provided" Enums(1d,1w,1m,3m,1y)
// @Success 200        {object} []api.Filter
// @Router  /inventory/api/v2/metrics/resources/metric [get]
func (h *HttpHandler) GetMetricsResourceCount(ctx echo.Context) error {
	var err error
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")
	importance := strings.ToLower(ctx.QueryParam("importance"))
	if importance == "" {
		importance = "critical"
	}
	importanceArray := strings.Split(importance, ",")
	category := ctx.QueryParam("category")

	timeStr := ctx.QueryParam("time")
	timeWindowStr := ctx.QueryParam("timeWindow")
	timeVal := time.Now().Unix()

	if timeStr != "" {
		timeVal, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}

	var timeWindow time.Duration
	if timeWindowStr != "" {
		timeWindow, err = ParseTimeWindow(timeWindowStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid timeWindow")
		}
	}

	result, err := h.GetMetricsResourceCountHelper(ctx.Request().Context(), category, sourceID, provider, timeVal, importanceArray)
	if err != nil {
		return err
	}
	if timeWindowStr != "" {
		historyResult, err := h.GetMetricsResourceCountHelper(ctx.Request().Context(), category, sourceID, provider, time.Unix(timeVal, 0).Add(-1*timeWindow).Unix(), importanceArray)
		if err != nil {
			return err
		}
		result = internal.CalculateMetricResourceTypeCountPercentChanges(result, historyResult)
	}

	resultAsArr := make([]api.Filter, 0)
	for _, v := range result {
		resultAsArr = append(resultAsArr, v)
	}

	return ctx.JSON(http.StatusOK, resultAsArr)
}

// GetCategoryNodeResourceCountComposition godoc
// @Summary Return category info by provided category id, info includes category name, subcategories names and ids and number of resources
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   category   query    string false "Category ID - defaults to default template category"
// @Param   top        query    int    true  "How many top categories to return"
// @Param   provider   query    string false "Provider"
// @Param   sourceId   query    string false "SourceID"
// @Param   importance query    string false "Filter filters by importance if they have it (array format is supported with , separator | 'all' is also supported)"
// @Param   time       query    string false "timestamp for resource count in epoch seconds"
// @Success 200        {object} api.CategoryNode
// @Router  /inventory/api/v2/resources/composition [get]
func (h *HttpHandler) GetCategoryNodeResourceCountComposition(ctx echo.Context) error {
	topStr := ctx.QueryParam("top")
	top, err := strconv.Atoi(topStr)
	if err != nil || top <= 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid top")
	}
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")
	importance := strings.ToLower(ctx.QueryParam("importance"))
	if importance == "" {
		importance = "critical"
	}
	importanceArray := strings.Split(importance, ",")
	category := ctx.QueryParam("category")

	timeStr := ctx.QueryParam("time")
	timeVal := time.Now().Unix()
	if timeStr != "" {
		timeVal, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}

	result, err := h.GetCategoryNodeResourceCountHelper(ctx.Request().Context(), 2, category, sourceID, provider, timeVal, importanceArray, make(map[string]api.CategoryNode))
	if err != nil {
		return err
	}
	// sort result.SubCategories by count desc
	sort.Slice(result.Subcategories, func(i, j int) bool {
		return *result.Subcategories[i].ResourceCount > *result.Subcategories[j].ResourceCount
	})
	// take top result and aggregate the rest into "other"
	if len(result.Subcategories) > top {
		other := api.CategoryNode{
			CategoryName:  "Others",
			ResourceCount: nil,
		}
		for i := top; i < len(result.Subcategories); i++ {
			other.ResourceCount = pointerAdd(other.ResourceCount, result.Subcategories[i].ResourceCount)
		}
		result.Subcategories = append(result.Subcategories[:top], other)
	}
	return ctx.JSON(http.StatusOK, result)
}

// GetMetricsResourceCountComposition godoc
// @Summary Return category info by provided category id, info includes category name, subcategories names and ids and number of resources
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   category   query    string false "Category ID - defaults to default template category"
// @Param   top        query    int    true  "How many top categories to return"
// @Param   provider   query    string false "Provider"
// @Param   sourceId   query    string false "SourceID"
// @Param   importance query    string false "Filter filters by importance if they have it (array format is supported with , separator | 'all' is also supported)"
// @Param   time       query    string false "timestamp for resource count in epoch seconds"
// @Success 200        {object} []api.Filter
// @Router  /inventory/api/v2/metrics/resources/composition [get]
func (h *HttpHandler) GetMetricsResourceCountComposition(ctx echo.Context) error {
	topStr := ctx.QueryParam("top")
	top, err := strconv.Atoi(topStr)
	if err != nil || top <= 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid top")
	}
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")
	importance := strings.ToLower(ctx.QueryParam("importance"))
	if importance == "" {
		importance = "critical"
	}
	importanceArray := strings.Split(importance, ",")
	category := ctx.QueryParam("category")

	timeStr := ctx.QueryParam("time")
	timeVal := time.Now().Unix()
	if timeStr != "" {
		timeVal, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}

	result, err := h.GetMetricsResourceCountHelper(ctx.Request().Context(), category, sourceID, provider, timeVal, importanceArray)
	if err != nil {
		return err
	}
	resultAsArr := make([]api.Filter, 0, len(result))
	for _, v := range result {
		resultAsArr = append(resultAsArr, v)
	}
	// sort result.SubCategories by count desc
	sort.Slice(resultAsArr, func(i, j int) bool {
		if resultAsArr[i].GetFilterType() == resultAsArr[j].GetFilterType() {
			switch resultAsArr[i].GetFilterType() {
			case api.FilterTypeCloudResourceType:
				return resultAsArr[i].(*api.FilterCloudResourceType).ResourceCount > resultAsArr[i].(*api.FilterCloudResourceType).ResourceCount
			}
		}
		if resultAsArr[i].GetFilterType() == api.FilterTypeCloudResourceType {
			return true
		}
		if resultAsArr[j].GetFilterType() == api.FilterTypeCloudResourceType {
			return false
		}
		return resultAsArr[i].GetFilterType() < resultAsArr[j].GetFilterType()
	})
	// take top result and aggregate the rest into "other"
	if len(resultAsArr) > top {
		other := &api.FilterCloudResourceType{
			FilterType:    api.FilterTypeCloudResourceType,
			FilterID:      "-others-",
			CloudProvider: provider,
			ResourceType:  "Others",
			ResourceName:  "Other",
			PluralName:    "Others",
			ResourceCount: 0,
		}
		for i := top; i < len(resultAsArr); i++ {
			switch resultAsArr[i].GetFilterType() {
			case api.FilterTypeCloudResourceType:
				other.ResourceCount += resultAsArr[i].(*api.FilterCloudResourceType).ResourceCount
			}
		}
		resultAsArr = append(resultAsArr[:top], other)
	}
	return ctx.JSON(http.StatusOK, resultAsArr)
}

func (h *HttpHandler) GetCategoryNodeCostHelper(ctx context.Context, depth int, category string, sourceID *string, providerPtr *source.Type, startTime, endTime int64, nodeCacheMap map[string]api.CategoryNode) (*api.CategoryNode, error) {
	var (
		rootNode *CategoryNode
		err      error
	)
	if category == "" {
		rootNode, err = h.graphDb.GetCategoryRootByName(ctx, RootTypeTemplateRoot, DefaultTemplateRootName, []string{"all"})
		if err != nil {
			return nil, err
		}
	} else {
		rootNode, err = h.graphDb.GetCategory(ctx, category, []string{"all"})
		if err != nil {
			return nil, err
		}
	}

	serviceNames := make([]string, 0)
	for _, filter := range rootNode.SubTreeFilters {
		if filter.GetFilterType() == FilterTypeCost {
			serviceNames = append(serviceNames, filter.(*FilterCostNode).ServiceName)
		}
	}
	if len(serviceNames) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "category has no cost filters")
	}

	costHits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, sourceID, providerPtr, serviceNames, time.Unix(endTime, 0), time.Unix(startTime, 0), EsFetchPageSize)
	aggregatedCostHits := internal.AggregateServiceCosts(costHits)
	if err != nil {
		return nil, err
	}

	result, err := RenderCategoryCostDFS(ctx,
		h.graphDb,
		rootNode,
		depth,
		aggregatedCostHits,
		nodeCacheMap,
		map[string]api.Filter{})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (h *HttpHandler) GetMetricsCostHelper(ctx context.Context, category string, sourceID *string, provider source.Type, startTime, endTime int64) (map[string]api.Filter, error) {
	var (
		filters []Filter
		err     error
	)
	if category == "" {
		filterType := FilterTypeCost
		filters, err = h.graphDb.GetFilters(ctx, provider, []string{"all"}, &filterType)
		if err != nil {
			return nil, err
		}
	} else {
		rootNode, err := h.graphDb.GetCategory(ctx, category, []string{"all"})
		if err != nil {
			return nil, err
		}
		filters = rootNode.SubTreeFilters
	}

	serviceNames := make([]string, 0)
	for _, filter := range filters {
		if filter.GetFilterType() == FilterTypeCost {
			serviceNames = append(serviceNames, filter.(*FilterCostNode).ServiceName)
		}
	}
	if len(serviceNames) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "category has no cost filters")
	}

	costHits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, sourceID, provider.AsPtr(), serviceNames, time.Unix(endTime, 0), time.Unix(startTime, 0), EsFetchPageSize)
	aggregatedCostHits := internal.AggregateServiceCosts(costHits)
	if err != nil {
		return nil, err
	}

	result := make(map[string]api.Filter)
	for _, filter := range filters {
		if filter.GetFilterType() == FilterTypeCost {
			costFilter := filter.(*FilterCostNode)

			if cost, ok := aggregatedCostHits[costFilter.ServiceName]; ok {
				result[costFilter.ServiceName] = &api.FilterCost{
					FilterType:    api.FilterTypeCost,
					FilterID:      costFilter.ElementID,
					ServiceName:   costFilter.Name,
					CloudProvider: costFilter.CloudProvider,
					Cost:          cost,
				}
			}
		}
	}

	return result, nil
}

// GetCategoryNodeCost godoc
// @Summary Return category cost info by provided category id, info includes category name, subcategories names and ids and their accumulated cost
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   category   query    string false "Category id - defaults to default template category"
// @Param   depth      query    int    true  "Depth of rendering subcategories"
// @Param   provider   query    string false "Provider"
// @Param   sourceId   query    string false "SourceID"
// @Param   startTime  query    string false "timestamp for start of cost window in epoch seconds"
// @Param   endTime    query    string false "timestamp for end of cost window in epoch seconds"
// @Param   timeWindow query    string false "time window either this or start & end time must be provided" Enums(1d,1w,1m,3m,1y)
// @Success 200        {object} api.CategoryNode
// @Router  /inventory/api/v2/cost/category [get]
func (h *HttpHandler) GetCategoryNodeCost(ctx echo.Context) error {
	depthStr := ctx.QueryParam("depth")
	depth, err := strconv.Atoi(depthStr)
	if err != nil || depth <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid depth")
	}
	provider := ctx.QueryParam("provider")
	var providerPtr *source.Type
	if provider != "" {
		providerType, err := source.ParseType(provider)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid provider")
		}
		providerPtr = &providerType
	}
	sourceID := ctx.QueryParam("sourceId")
	sourceIDPtr := &sourceID
	if sourceID == "" {
		sourceIDPtr = nil
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Now().AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid startTime")
		}
	}
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid endTime")
		}
	}

	timeWindowStr := ctx.QueryParam("timeWindow")
	var timeWindow time.Duration
	if timeWindowStr != "" {
		timeWindow, err = ParseTimeWindow(timeWindowStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid timeWindow")
		}
		if startTimeStr == "" {
			startTime = time.Unix(endTime, 0).Add(-1 * timeWindow).Unix()
		}
	}

	category := ctx.QueryParam("category")

	result, err := h.GetCategoryNodeCostHelper(ctx.Request().Context(), depth, category, sourceIDPtr, providerPtr, startTime, endTime, make(map[string]api.CategoryNode))
	if err != nil {
		return err
	}
	if timeWindowStr != "" {
		nodeCacheMap := make(map[string]api.CategoryNode)
		_, err = h.GetCategoryNodeCostHelper(ctx.Request().Context(), depth, category, sourceIDPtr, providerPtr, time.Unix(startTime, 0).Add(-1*timeWindow).Unix(), startTime, nodeCacheMap)
		if err != nil {
			return err
		}
		result = internal.CalculateCostPercentChanges(result, nodeCacheMap)
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetMetricsCost godoc
// @Summary Return category cost info by provided category id, info includes category name, subcategories names and ids and their accumulated cost
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   category   query    string false "Category id - defaults to default template category"
// @Param   provider   query    string false "Provider"
// @Param   sourceId   query    string false "SourceID"
// @Param   startTime  query    string false "timestamp for start of cost window in epoch seconds"
// @Param   endTime    query    string false "timestamp for end of cost window in epoch seconds"
// @Param   timeWindow query    string false "time window either this or start & end time must be provided" Enums(1d,1w,1m,3m,1y)
// @Success 200        {object} []api.Filter
// @Router  /inventory/api/v2/metrics/cost/metric [get]
func (h *HttpHandler) GetMetricsCost(ctx echo.Context) error {
	var err error
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")
	sourceIDPtr := &sourceID
	if sourceID == "" {
		sourceIDPtr = nil
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Now().AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid startTime")
		}
	}
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid endTime")
		}
	}

	timeWindowStr := ctx.QueryParam("timeWindow")
	var timeWindow time.Duration
	if timeWindowStr != "" {
		timeWindow, err = ParseTimeWindow(timeWindowStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid timeWindow")
		}
		if startTimeStr == "" {
			startTime = time.Unix(endTime, 0).Add(-1 * timeWindow).Unix()
		}
	}

	category := ctx.QueryParam("category")

	result, err := h.GetMetricsCostHelper(ctx.Request().Context(), category, sourceIDPtr, provider, startTime, endTime)
	if err != nil {
		return err
	}
	if timeWindowStr != "" {
		historyResult, err := h.GetMetricsCostHelper(ctx.Request().Context(), category, sourceIDPtr, provider, time.Unix(startTime, 0).Add(-1*timeWindow).Unix(), time.Unix(endTime, 0).Add(-1*timeWindow).Unix())
		if err != nil {
			return err
		}
		result = internal.CalculateMetricCostPercentChanges(result, historyResult)
	}

	resultAsArray := make([]api.Filter, 0, len(result))
	for _, v := range result {
		resultAsArray = append(resultAsArray, v)
	}

	return ctx.JSON(http.StatusOK, resultAsArray)
}

// GetCategoryNodeCostComposition godoc
// @Summary Return category info by provided category id, info includes category name, subcategories names and ids and number of resources
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   category  query    string false "Category ID - defaults to default template category"
// @Param   top       query    int    true  "How many top categories to return"
// @Param   provider  query    string false "Provider"
// @Param   sourceId  query    string false "SourceID"
// @Param   startTime query    string false "timestamp for start of cost window in epoch seconds"
// @Param   endTime   query    string false "timestamp for end of cost window in epoch seconds"
// @Param   costUnit  query    string true  "Unit of cost to filter by"
// @Success 200       {object} api.CategoryNode
// @Router  /inventory/api/v2/cost/composition [get]
func (h *HttpHandler) GetCategoryNodeCostComposition(ctx echo.Context) error {
	topStr := ctx.QueryParam("top")
	top, err := strconv.Atoi(topStr)
	if err != nil || top <= 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid top")
	}
	provider := ctx.QueryParam("provider")
	var providerPtr *source.Type
	if provider != "" {
		providerType, err := source.ParseType(provider)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid provider")
		}
		providerPtr = &providerType
	}
	sourceID := ctx.QueryParam("sourceId")
	sourceIDPtr := &sourceID
	if sourceID == "" {
		sourceIDPtr = nil
	}

	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Now().AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid startTime")
		}
	}
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid endTime")
		}
	}

	costUnit := ctx.QueryParam("costUnit")
	if costUnit == "" {
		costUnit = DefaultCurrency
	}

	category := ctx.QueryParam("category")

	result, err := h.GetCategoryNodeCostHelper(ctx.Request().Context(), 2, category, sourceIDPtr, providerPtr, startTime, endTime, make(map[string]api.CategoryNode))
	if err != nil {
		return err
	}
	// sort result.SubCategories by count desc
	sort.Slice(result.Subcategories, func(i, j int) bool {
		if _, ok := result.Subcategories[i].Cost[costUnit]; !ok {
			return false
		}
		if _, ok := result.Subcategories[j].Cost[costUnit]; !ok {
			return true
		}
		return result.Subcategories[i].Cost[costUnit].Cost > result.Subcategories[j].Cost[costUnit].Cost
	})
	// take top result and aggregate the rest into "other"
	if len(result.Subcategories) > top {
		other := api.CategoryNode{
			CategoryName: "Others",
			Cost: map[string]api.CostWithUnit{
				costUnit: {
					Cost: 0,
					Unit: costUnit,
				},
			},
		}
		for i := top; i < len(result.Subcategories); i++ {
			if _, ok := result.Subcategories[i].Cost[costUnit]; ok {
				v := other.Cost[costUnit]
				v.Cost += result.Subcategories[i].Cost[costUnit].Cost
				other.Cost[costUnit] = v
			}
		}
		result.Subcategories = append(result.Subcategories[:top], other)
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetMetricsCostComposition godoc
// @Summary Return category info by provided category id, info includes category name, subcategories names and ids and number of resources
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   category  query    string false "Category ID - defaults to default template category"
// @Param   top       query    int    true  "How many top categories to return"
// @Param   provider  query    string false "Provider"
// @Param   sourceId  query    string false "SourceID"
// @Param   startTime query    string false "timestamp for start of cost window in epoch seconds"
// @Param   endTime   query    string false "timestamp for end of cost window in epoch seconds"
// @Param   costUnit  query    string false "Unit of cost to filter by"
// @Success 200       {object} []api.Filter
// @Router  /inventory/api/v2/metrics/cost/composition [get]
func (h *HttpHandler) GetMetricsCostComposition(ctx echo.Context) error {
	topStr := ctx.QueryParam("top")
	top, err := strconv.Atoi(topStr)
	if err != nil || top <= 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid top")
	}
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")
	sourceIDPtr := &sourceID
	if sourceID == "" {
		sourceIDPtr = nil
	}

	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Now().AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid startTime")
		}
	}
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid endTime")
		}
	}

	costUnit := ctx.QueryParam("costUnit")
	if costUnit == "" {
		costUnit = DefaultCurrency
	}

	category := ctx.QueryParam("category")

	result, err := h.GetMetricsCostHelper(ctx.Request().Context(), category, sourceIDPtr, provider, startTime, endTime)
	if err != nil {
		return err
	}
	resultAsArr := make([]api.Filter, 0, len(result))
	for _, v := range result {
		resultAsArr = append(resultAsArr, v)
	}

	// sort result.SubCategories by count desc
	sort.Slice(resultAsArr, func(i, j int) bool {
		if resultAsArr[i].GetFilterType() == resultAsArr[j].GetFilterType() {
			switch resultAsArr[i].GetFilterType() {
			case api.FilterTypeCost:
				if _, ok := resultAsArr[i].(*api.FilterCost).Cost[costUnit]; !ok {
					return false
				}
				if _, ok := resultAsArr[j].(*api.FilterCost).Cost[costUnit]; !ok {
					return true
				}
				return resultAsArr[i].(*api.FilterCost).Cost[costUnit].Cost > resultAsArr[j].(*api.FilterCost).Cost[costUnit].Cost
			}
		}
		if resultAsArr[i].GetFilterType() == api.FilterTypeCost {
			return true
		}
		if resultAsArr[j].GetFilterType() == api.FilterTypeCost {
			return false
		}
		return resultAsArr[i].GetFilterType() < resultAsArr[j].GetFilterType()
	})
	// take top result and aggregate the rest into "other"
	if len(resultAsArr) > top {
		other := &api.FilterCost{
			FilterType:    api.FilterTypeCost,
			FilterID:      "-other-",
			ServiceName:   "Others",
			CloudProvider: provider,
			Cost: map[string]api.CostWithUnit{
				costUnit: {
					Cost: 0,
					Unit: costUnit,
				},
			},
		}
		for i := top; i < len(resultAsArr); i++ {
			switch resultAsArr[i].GetFilterType() {
			case api.FilterTypeCost:
				f := resultAsArr[i].(*api.FilterCost)
				if _, ok := f.Cost[costUnit]; ok {
					v := other.Cost[costUnit]
					v.Cost += f.Cost[costUnit].Cost
					other.Cost[costUnit] = v
				}
			}

		}
		resultAsArr = append(resultAsArr, other)
	}

	return ctx.JSON(http.StatusOK, resultAsArr)
}

// GetRootTemplates godoc
// @Summary Return root templates' info, info includes template name, template id, subcategories names and ids and number of resources
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   provider   query    string false "Provider"
// @Param   sourceId   query    string false "SourceID"
// @Param   importance query    string false "Filter filters by importance if they have it (array format is supported with , separator | 'all' is also supported)"
// @Param   time       query    string false "timestamp for resource count in epoch seconds"
// @Success 200        {object} []api.CategoryNode
// @Router  /inventory/api/v2/resources/rootTemplates [get]
func (h *HttpHandler) GetRootTemplates(ctx echo.Context) error {
	return GetCategoryRoots(ctx, h, RootTypeTemplateRoot)
}

// GetRootCloudProviders godoc
// @Summary Return root providers' info, info includes category name, category id, subcategories names and ids and number of resources
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   provider   query    string false "Provider"
// @Param   sourceId   query    string false "SourceID"
// @Param   importance query    string false "Filter filters by importance if they have it (array format is supported with , separator | 'all' is also supported)"
// @Param   time       query    string false "timestamp for resource count in epoch seconds"
// @Success 200        {object} []api.CategoryNode
// @Router  /inventory/api/v2/resources/rootCloudProviders [get]
func (h *HttpHandler) GetRootCloudProviders(ctx echo.Context) error {
	return GetCategoryRoots(ctx, h, RootTypeCloudProviderRoot)
}

func GetCategoryRoots(ctx echo.Context, h *HttpHandler, rootType CategoryRootType) error {
	var (
		err error
	)
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")
	sourceIDPtr := &sourceID
	if sourceID == "" {
		sourceIDPtr = nil
	}
	importance := strings.ToLower(ctx.QueryParam("importance"))
	if importance == "" {
		importance = "critical"
	}
	importanceArray := strings.Split(importance, ",")

	timeStr := ctx.QueryParam("time")
	timeVal := time.Now().Unix()
	if timeStr != "" {
		timeVal, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}

	templateRoots, err := h.graphDb.GetCategoryRoots(ctx.Request().Context(), rootType, importanceArray)
	if err != nil {
		return err
	}

	filters := make([]Filter, 0)
	for _, templateRoot := range templateRoots {
		filters = append(filters, templateRoot.SubTreeFilters...)
	}
	resourceTypes := GetResourceTypeListFromFilters(filters, provider)

	metricIndexed, err := es.FetchResourceTypeCountAtTime(h.client, provider, sourceIDPtr, time.Unix(timeVal, 0), resourceTypes, EsFetchPageSize)

	results := make([]api.CategoryNode, 0, len(templateRoots))
	cacheMap := map[string]api.Filter{}
	for _, templateRoot := range templateRoots {
		results = append(results, GetCategoryNodeResourceCountInfo(templateRoot, metricIndexed, cacheMap))
	}

	return ctx.JSON(http.StatusOK, results)
}

// GetSummaryMetrics godoc
// @Summary Return metrics, their value and their history
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   provider query    string false "Provider"
// @Param   sourceId query    string false "SourceID"
// @Success 200      {object} []api.MetricsResponse
// @Router  /inventory/api/v1/metrics/summary [get]
func (h *HttpHandler) GetSummaryMetrics(ctx echo.Context) error {
	var sourceID *string
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	if s := ctx.QueryParam("sourceId"); s != "" {
		sourceID = &s
	}

	includeAWS, includeAzure := false, false
	switch provider {
	case source.CloudAWS:
		includeAWS = true
	case source.CloudAzure:
		includeAzure = true
	default:
		includeAzure, includeAWS = true, true
	}

	var res []api.MetricsResponse
	padd := func(x, y *int) *int {
		var v *int
		if x != nil && y != nil {
			t := *x + *y
			v = &t
		} else if x != nil {
			v = x
		} else if y != nil {
			v = y
		}
		return v
	}
	padd64 := func(x, y *int64) *int {
		var v *int
		if x != nil && y != nil {
			t := int(*x) + int(*y)
			v = &t
		} else if x != nil {
			t := int(*x)
			v = &t
		} else if y != nil {
			t := int(*y)
			v = &t
		}
		return v
	}

	extractMetric := func(allProviderName, awsName, awsResourceType, azureName, azureResourceType string) error {
		awsResourceType = strings.ToLower(awsResourceType)
		azureResourceType = strings.ToLower(azureResourceType)

		var aws, azure api.ResourceTypeResponse
		metricName := allProviderName
		switch provider {
		case source.CloudAWS:
			metricName = awsName
		case source.CloudAzure:
			metricName = azureName
		}
		if metricName == "" {
			return nil
		}

		if awsResourceType != "" {
			v, err := GetResourcesFromPostgres(h.db, provider, sourceID, []string{awsResourceType})
			if err != nil {
				return err
			}
			if len(v) > 0 {
				aws = v[0]
			}
		}

		if azureResourceType != "" {
			v, err := GetResourcesFromPostgres(h.db, provider, sourceID, []string{azureResourceType})
			if err != nil {
				return err
			}
			if len(v) > 0 {
				azure = v[0]
			}
		}

		res = append(res, api.MetricsResponse{
			MetricsName:      metricName,
			Value:            azure.ResourceCount + aws.ResourceCount,
			LastDayValue:     padd(azure.LastDayCount, aws.LastDayCount),
			LastWeekValue:    padd(azure.LastWeekCount, aws.LastWeekCount),
			LastQuarterValue: padd(azure.LastQuarterCount, aws.LastQuarterCount),
			LastYearValue:    padd(azure.LastYearCount, aws.LastYearCount),
		})
		return nil
	}

	query, err := es.FindInsightResults(nil, nil)
	if err != nil {
		return err
	}

	var response es.InsightResultQueryResponse
	err = h.client.Search(context.Background(), insight.InsightsIndex,
		query, &response)
	if err != nil {
		return err
	}

	var awsAccountCount, azureAccountCount insight.InsightResource
	for _, item := range response.Hits.Hits {
		if includeAWS && item.Source.Description == "AWS Account Count" {
			awsAccountCount = item.Source
		}
		if includeAzure && item.Source.Description == "Azure Account Count" {
			azureAccountCount = item.Source
		}
	}

	totalAccounts, err := h.onboardClient.CountSources(httpclient.FromEchoContext(ctx), provider)
	if err != nil {
		return err
	}

	res = append(res, api.MetricsResponse{
		MetricsName:      "Total Accounts",
		Value:            int(totalAccounts),
		LastDayValue:     padd64(awsAccountCount.LastDayValue, azureAccountCount.LastDayValue),
		LastWeekValue:    padd64(awsAccountCount.LastWeekValue, azureAccountCount.LastWeekValue),
		LastQuarterValue: padd64(awsAccountCount.LastQuarterValue, azureAccountCount.LastQuarterValue),
		LastYearValue:    padd64(awsAccountCount.LastYearValue, azureAccountCount.LastYearValue),
	})

	var lastValue, lastDayResourceCount, lastWeekResourceCount, lastQuarterResourceCount, lastYearResourceCount *int
	for _, days := range []int{1, 2, 7, 93, 428} {
		fromTime := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
		toTime := fromTime.Add(24 * time.Hour)
		d, err := ExtractTrend(h.client, provider, sourceID, fromTime.UnixMilli(), toTime.UnixMilli())
		if err != nil {
			return err
		}
		if len(d) > 0 {
			max := 0
			var maxItem int64
			for k, v := range d {
				if k > maxItem {
					maxItem, max = k, v
				}
			}
			switch days {
			case 1:
				lastValue = &max
			case 2:
				lastDayResourceCount = &max
			case 7:
				lastWeekResourceCount = &max
			case 93:
				lastQuarterResourceCount = &max
			case 428:
				lastYearResourceCount = &max
			}
		}
	}
	if lastValue == nil {
		v := 0
		lastValue = &v
	}

	res = append(res, api.MetricsResponse{
		MetricsName:      "Cloud Resources",
		Value:            *lastValue,
		LastDayValue:     lastDayResourceCount,
		LastWeekValue:    lastWeekResourceCount,
		LastQuarterValue: lastQuarterResourceCount,
		LastYearValue:    lastYearResourceCount,
	})

	if err := extractMetric("Virtual Machines",
		"Virtual Machines", "aws::ec2::instance",
		"Virtual Machines", "Microsoft.Compute/virtualMachines"); err != nil {
		return err
	}

	if err := extractMetric("Networks",
		"Networks (VPC)", "aws::ec2::vpc",
		"Networks (vNets)", "Microsoft.Network/virtualNetworks"); err != nil {
		return err
	}

	if err := extractMetric("Disks",
		"Disks", "aws::ec2::volume",
		"Managed Disks", "Microsoft.Compute/disks"); err != nil {
		return err
	}

	var awsStorage, azureStorage insight.InsightResource
	for _, item := range response.Hits.Hits {
		if includeAWS && item.Source.Description == "AWS Storage" {
			awsStorage = item.Source
		}
		if includeAzure && item.Source.Description == "Azure Storage" {
			azureStorage = item.Source
		}
	}

	res = append(res, api.MetricsResponse{
		MetricsName:      "Total storage",
		Value:            int(awsStorage.Result + azureStorage.Result),
		LastDayValue:     padd64(awsStorage.LastDayValue, azureStorage.LastDayValue),
		LastWeekValue:    padd64(awsStorage.LastWeekValue, azureStorage.LastWeekValue),
		LastQuarterValue: padd64(awsStorage.LastQuarterValue, azureStorage.LastQuarterValue),
		LastYearValue:    padd64(awsStorage.LastYearValue, azureStorage.LastYearValue),
	})

	if err := extractMetric("DB Services",
		"RDS Instances", "aws::rds::dbinstance",
		"SQL Instances", "Microsoft.Sql/managedInstances"); err != nil {
		return err
	}

	if err := extractMetric("",
		"S3 Buckets", "aws::s3::bucket",
		"Storage Accounts", "Microsoft.Storage/storageAccounts"); err != nil {
		return err
	}

	if err := extractMetric("Kubernetes Cluster",
		"Kubernetes Cluster", "aws::eks::cluster",
		"Azure Kubernetes", "Microsoft.Kubernetes/connectedClusters"); err != nil {
		return err
	}

	if err := extractMetric("Serverless",
		"Lambda", "aws::lambda::function",
		"", ""); err != nil {
		return err
	}

	if err := extractMetric("PaaS",
		"Elastic Beanstalk", "aws::elasticbeanstalk::environment",
		"Apps", "microsoft.app/containerapps"); err != nil {
		return err
	}

	for i := 0; i < len(res); i++ {
		if res[i].MetricsName == "" {
			res = append(res[:i], res[i+1:]...)
		}
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetCategorizedMetrics godoc
// @Summary Return categorized metrics, their value and their history
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   provider query    string false "Provider"
// @Param   sourceId query    string false "SourceID"
// @Success 200      {object} api.CategorizedMetricsResponse
// @Router  /inventory/api/v1/metrics/categorized [get]
func (h *HttpHandler) GetCategorizedMetrics(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))

	var sourceID *string
	if sID := ctx.QueryParam("sourceId"); sID != "" {
		sourceUUID, err := uuid.Parse(sID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid sourceID")
		}
		s := sourceUUID.String()
		sourceID = &s
	}

	var res api.CategorizedMetricsResponse
	res.Category = make(map[string][]api.ResourceTypeResponse)
	for _, category := range cloudservice.ListCategories() {
		resourceList := cloudservice.ResourceListByCategory(category)
		if len(resourceList) == 0 {
			continue
		}

		v, err := GetResourcesFromPostgres(h.db, provider, sourceID, resourceList)
		if err != nil {
			return err
		}

		if v == nil || len(v) == 0 {
			continue
		}

		res.Category[category] = v
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetCategorizedMetricsV2 godoc
// @Summary Return categorized metrics, their value and their history
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   provider    query    string false "Provider"
// @Param   sourceId    query    string false "SourceID"
// @Param   category    query    string false "Category"
// @Param   subCategory query    string false "SubCategory"
// @Success 200         {object} api.CategoriesMetrics
// @Router  /inventory/api/v2/metrics/categorized [get]
func (h *HttpHandler) GetCategorizedMetricsV2(ctx echo.Context) error {
	startTime := time.Now().UnixMilli()

	category := ctx.QueryParam("category")
	subCategory := ctx.QueryParam("subCategory")

	provider, _ := source.ParseType(ctx.QueryParam("provider"))

	var sourceID *string
	if sID := ctx.QueryParam("sourceId"); sID != "" {
		sourceUUID, err := uuid.Parse(sID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid sourceID")
		}
		s := sourceUUID.String()
		sourceID = &s
	}

	var cats []Category
	var err error

	fmt.Println("getting categories", time.Now().UnixMilli()-startTime)
	if len(category) > 0 && len(subCategory) > 0 {
		cats, err = h.db.GetCategories(category, subCategory)
	} else if len(category) > 0 {
		cats, err = h.db.GetSubCategories(category)
	} else {
		cats, err = h.db.ListCategories()
	}
	if err != nil {
		return err
	}

	fmt.Println("getting resource types", time.Now().UnixMilli()-startTime)
	allResourceTypes := map[string]struct{}{}
	for _, v := range cats {
		resourceList := cloudservice.ResourceListByServiceName(v.CloudService)
		for _, r := range resourceList {
			allResourceTypes[r] = struct{}{}
		}
	}
	if len(allResourceTypes) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid category/subcategory")
	}

	var resourceTypeArr []string
	for r := range allResourceTypes {
		resourceTypeArr = append(resourceTypeArr, r)
	}

	fmt.Println("getting metrics", time.Now().UnixMilli()-startTime)
	metrics, err := GetResourcesFromPostgres(h.db, provider, sourceID, resourceTypeArr)
	if err != nil {
		return err
	}

	resp := api.CategoriesMetrics{
		Categories: map[string]api.CategoryMetric{},
	}

	fmt.Println("getting result", time.Now().UnixMilli()-startTime)
	for _, v := range cats {
		resourceList := cloudservice.ResourceListByServiceName(v.CloudService)
		for _, r := range resourceList {
			for _, m := range metrics {
				if m.ResourceType == r {
					c := resp.Categories[v.Name]
					c.ResourceCount += m.ResourceCount
					c.LastDayValue = utils.PAdd(c.LastDayValue, m.LastDayCount)
					c.LastWeekValue = utils.PAdd(c.LastWeekValue, m.LastWeekCount)
					c.LastQuarterValue = utils.PAdd(c.LastQuarterValue, m.LastQuarterCount)
					c.LastYearValue = utils.PAdd(c.LastYearValue, m.LastYearCount)
					if c.SubCategories == nil {
						c.SubCategories = map[string]api.HistoryCount{}
					}
					sub := c.SubCategories[v.SubCategory]
					sub.Count += m.ResourceCount
					sub.LastDayValue = utils.PAdd(sub.LastDayValue, m.LastDayCount)
					sub.LastWeekValue = utils.PAdd(sub.LastWeekValue, m.LastWeekCount)
					sub.LastQuarterValue = utils.PAdd(sub.LastQuarterValue, m.LastQuarterCount)
					sub.LastYearValue = utils.PAdd(sub.LastYearValue, m.LastYearCount)

					c.SubCategories[v.SubCategory] = sub
					resp.Categories[v.Name] = c
				}
			}
		}
	}
	fmt.Println("getting response", time.Now().UnixMilli()-startTime)
	return ctx.JSON(http.StatusOK, resp)
}

// ListCategories godoc
// @Summary Return list of categories
// @Tags    inventory
// @Accept  json
// @Produce json
// @Success 200 {object} []string
// @Router  /inventory/api/v1/categories [get]
func (h *HttpHandler) ListCategories(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, cloudservice.ListCategories())
}

// ListCategoriesV2 godoc
// @Summary Return list of categories
// @Tags    inventory
// @Accept  json
// @Produce json
// @Success 200 {object} []string
// @Router  /inventory/api/v2/categories [get]
func (h *HttpHandler) ListCategoriesV2(ctx echo.Context) error {
	cats, err := h.db.ListCategories()
	if err != nil {
		return err
	}

	cmap := map[string][]string{}
	for _, c := range cats {
		exists := false
		for _, s := range cmap[c.Name] {
			if s == c.SubCategory {
				exists = true
			}
		}
		if exists {
			continue
		}
		cmap[c.Name] = append(cmap[c.Name], c.SubCategory)
	}
	var resp []api.Category
	for k, v := range cmap {
		resp = append(resp, api.Category{
			Name:        k,
			SubCategory: v,
		})
	}
	return ctx.JSON(http.StatusOK, resp)
}

// GetAccountsResourceCount godoc
// @Summary Returns resource count of accounts
// @Tags    benchmarks
// @Accept  json
// @Produce json
// @Param   provider query    string true  "Provider"
// @Param   sourceId query    string false "SourceID"
// @Success 200      {object} []api.AccountResourceCountResponse
// @Router  /inventory/api/v1/accounts/resource/count [get]
func (h *HttpHandler) GetAccountsResourceCount(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceId := ctx.QueryParam("sourceId")
	var sourceIdPtr *string
	if sourceId != "" {
		sourceIdPtr = &sourceId
	}

	res := map[string]api.AccountResourceCountResponse{}

	var err error
	var allSources []api2.Source
	if sourceId == "" {
		allSources, err = h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), provider.AsPtr())
	} else {
		allSources, err = h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), []string{sourceId})
	}
	if err != nil {
		return err
	}

	for _, src := range allSources {
		res[src.ID.String()] = api.AccountResourceCountResponse{
			SourceID:               src.ID.String(),
			SourceType:             source.Type(src.Type),
			ProviderConnectionName: src.ConnectionName,
			ProviderConnectionID:   src.ConnectionID,
			Enabled:                src.Enabled,
			OnboardDate:            src.OnboardDate,
		}
	}

	hits, err := es.FetchConnectionResourcesSummaryPage(h.client, provider, sourceIdPtr, nil, EsFetchPageSize)
	for _, hit := range hits {
		if v, ok := res[hit.SourceID]; ok {
			v.ResourceCount += hit.ResourceCount
			v.LastInventory = time.UnixMilli(hit.DescribedAt)
			res[hit.SourceID] = v
		}
	}
	var response []api.AccountResourceCountResponse
	for _, v := range res {
		response = append(response, v)
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetAccountSummary godoc
// @Summary Returns resource count of accounts
// @Tags    benchmarks
// @Accept  json
// @Produce json
// @Param   provider   query    string true  "Provider"
// @Param   sourceId   query    string false "SourceID"
// @Param   pageSize   query    int    false "page size - default is 20"
// @Param   pageNumber query    int    false "page number - default is 1"
// @Param   sortBy     query    string false "column to sort by - default is sourceid" Enums(sourceid,resourcecount,cost)
// @Success 200        {object} []api.AccountSummaryResponse
// @Router  /inventory/api/v2/accounts/summary [get]
func (h *HttpHandler) GetAccountSummary(ctx echo.Context) error {
	var err error

	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceId := ctx.QueryParam("sourceId")
	var sourceIdPtr *string
	if sourceId != "" {
		sourceIdPtr = &sourceId
	}

	pageSizeStr := ctx.QueryParam("pageSize")
	pageSize := ApiDefaultPageSize
	if pageSizeStr != "" {
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "pageSize is not a valid integer")
		}
	}
	pageNumberStr := ctx.QueryParam("pageNumber")
	pageNumber := 1
	if pageNumberStr != "" {
		pageNumber, err = strconv.Atoi(pageNumberStr)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "pageNumber is not a valid integer")
		}
	}
	sortBy := ctx.QueryParam("sortBy")
	if sortBy == "" {
		sortBy = "sourceid"
	}

	res := map[string]api.AccountSummary{}

	var allSources []api2.Source
	if sourceId == "" {
		allSources, err = h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), provider.AsPtr())
	} else {
		allSources, err = h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), []string{sourceId})
	}
	if err != nil {
		return err
	}

	for _, src := range allSources {
		res[src.ID.String()] = api.AccountSummary{
			SourceID:               src.ID.String(),
			SourceType:             source.Type(src.Type),
			ProviderConnectionName: src.ConnectionName,
			ProviderConnectionID:   src.ConnectionID,
			Enabled:                src.Enabled,
			OnboardDate:            src.OnboardDate,
		}
	}

	hits, err := es.FetchConnectionResourcesSummaryPage(h.client, provider, sourceIdPtr, nil, EsFetchPageSize)
	for _, hit := range hits {
		if v, ok := res[hit.SourceID]; ok {
			v.ResourceCount += hit.ResourceCount
			v.LastInventory = time.UnixMilli(hit.DescribedAt)
			res[hit.SourceID] = v
		}
	}

	costs, err := es.FetchDailyCostHistoryByAccountsBetween(h.client, sourceIdPtr, provider.AsPtr(), time.Now(), time.Now().AddDate(0, 0, -7), EsFetchPageSize)
	if err != nil {
		return err
	}
	for sourceID, costArr := range costs {
		if v, ok := res[sourceID]; ok {
			if v.Cost == nil {
				v.Cost = make(map[string]float64)
			}
			if v.LastCost == nil {
				v.LastCost = make(map[string]time.Time)
			}
			for _, cost := range costArr {
				costValue, costUnit := cost.GetCostAndUnit()
				lastCostDate, ok := v.LastCost[costUnit]
				if !ok || lastCostDate.Before(time.Unix(cost.PeriodEnd, 0)) {
					v.Cost[costUnit] = 0
					lastCostDate = time.Unix(cost.PeriodEnd, 0)
					v.LastCost[costUnit] = lastCostDate
				}
				if lastCostDate.Equal(time.Unix(cost.PeriodEnd, 0)) {
					v.Cost[costUnit] += costValue
				}
			}
			res[sourceID] = v
		}
	}

	var accountSummaries []api.AccountSummary
	for _, v := range res {
		accountSummaries = append(accountSummaries, v)
	}

	switch sortBy {
	case "sourceid":
		sort.Slice(accountSummaries, func(i, j int) bool {
			return accountSummaries[i].SourceID < accountSummaries[j].SourceID
		})
	case "resourcecount":
		sort.Slice(accountSummaries, func(i, j int) bool {
			return accountSummaries[i].ResourceCount > accountSummaries[j].ResourceCount
		})
	case "cost":
		sort.Slice(accountSummaries, func(i, j int) bool {
			return accountSummaries[i].Cost[DefaultCurrency] > accountSummaries[j].Cost[DefaultCurrency]
		})
	default:
		sort.Slice(accountSummaries, func(i, j int) bool {
			return accountSummaries[i].SourceID < accountSummaries[j].SourceID
		})
	}

	response := api.AccountSummaryResponse{
		TotalCount: len(accountSummaries),
		Accounts:   internal.Paginate(pageNumber, pageSize, accountSummaries),
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetResourceDistribution godoc
// @Summary Returns distribution of resource for specific account
// @Tags    benchmarks
// @Accept  json
// @Produce json
// @Param   sourceId   query    string true "SourceID"
// @Param   provider   query    string true "Provider"    Enums(AWS,Azure,all)
// @Param   timeWindow query    string true "Time Window" Enums(24h,1w,3m,1y,max)
// @Success 200        {object} map[string]int
// @Router  /inventory/api/v1/resources/distribution [get]
func (h *HttpHandler) GetResourceDistribution(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceID := ctx.QueryParam("sourceId")

	var sourceIDPtr *string
	if sourceID != "" {
		sourceIDPtr = &sourceID
	}
	locationDistribution := map[string]int{}

	hits, err := es.FetchConnectionLocationsSummaryPage(h.client, provider, sourceIDPtr, nil, EsFetchPageSize)
	if err != nil {
		return err
	}

	for _, hit := range hits {
		for k, v := range hit.LocationDistribution {
			locationDistribution[k] += v
		}
	}
	return ctx.JSON(http.StatusOK, locationDistribution)
}

// GetServiceDistribution godoc
// @Summary Returns distribution of services for specific account
// @Tags    benchmarks
// @Accept  json
// @Produce json
// @Param   sourceId query    string true "SourceID"
// @Param   provider query    string true "Provider"
// @Success 200      {object} []api.ServiceDistributionItem
// @Router  /inventory/api/v1/services/distribution [get]
func (h *HttpHandler) GetServiceDistribution(ctx echo.Context) error {
	sourceID := ctx.QueryParam("sourceId")

	hits, err := es.FetchConnectionServiceLocationsSummaryPage(h.client, source.Nil, &sourceID, nil, EsFetchPageSize)
	if err != nil {
		return err
	}

	var res []api.ServiceDistributionItem
	for _, hit := range hits {
		res = append(res, api.ServiceDistributionItem{
			ServiceName:  hit.ServiceName,
			Distribution: hit.LocationDistribution,
		})
	}
	return ctx.JSON(http.StatusOK, res)
}

// GetServiceSummary godoc
// @Summary Returns distribution of services for specific account
// @Tags    benchmarks
// @Accept  json
// @Produce json
// @Param   sourceId   query    string false "SourceID"
// @Param   provider   query    string false "Provider"
// @Param   startTime  query    string true  "start time for cost calculation in epoch seconds"
// @Param   endTime    query    string true  "end time for cost calculation and time resource count in epoch seconds"
// @Param   pageSize   query    int    false "page size - default is 20"
// @Param   pageNumber query    int    false "page number - default is 1"
// @Param   sortBy     query    string false "column to sort by - default is servicecode" Enums(servicecode,resourcecount,cost)
// @Success 200        {object} []api.ServiceSummaryResponse
// @Router  /inventory/api/v2/services/summary [get]
func (h *HttpHandler) GetServiceSummary(ctx echo.Context) error {
	sourceID := ctx.QueryParam("sourceId")
	var sourceIDPtr *string
	if sourceID != "" {
		sourceIDPtr = &sourceID
	}
	provider, _ := source.ParseType(ctx.QueryParam("provider"))

	startTime, err := strconv.ParseInt(ctx.QueryParam("startTime"), 10, 64)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "startTime is not a valid epoch time")
	}
	endTime, err := strconv.ParseInt(ctx.QueryParam("endTime"), 10, 64)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, "endTime is not a valid epoch time")
	}

	pageSizeStr := ctx.QueryParam("pageSize")
	pageSize := ApiDefaultPageSize
	if pageSizeStr != "" {
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "pageSize is not a valid integer")
		}
	}
	pageNumberStr := ctx.QueryParam("pageNumber")
	pageNumber := 1
	if pageNumberStr != "" {
		pageNumber, err = strconv.Atoi(pageNumberStr)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "pageNumber is not a valid integer")
		}
	}
	sortBy := ctx.QueryParam("sortBy")
	if sortBy == "" {
		sortBy = "servicecode"
	}

	serviceNodes, err := h.graphDb.GetCloudServiceNodes(ctx.Request().Context(), provider, []string{"all"})
	if err != nil {
		return err
	}
	costFilterMap := make(map[string]map[string]api.CostWithUnit)
	resourceTypeFilterMap := make(map[string]int64)
	for _, serviceNode := range serviceNodes {
		for _, f := range serviceNode.SubTreeFilters {
			switch f.GetFilterType() {
			case FilterTypeCost:
				filter := f.(*FilterCostNode)
				if provider.IsNull() || provider.String() == filter.CloudProvider.String() {
					costFilterMap[filter.ServiceName] = map[string]api.CostWithUnit{}
				}
			case FilterTypeCloudResourceType:
				filter := f.(*FilterCloudResourceTypeNode)
				if provider.IsNull() || provider.String() == filter.CloudProvider.String() {
					resourceTypeFilterMap[filter.ResourceType] = 0
				}
			}
		}
	}

	costFilters := make([]string, 0, len(costFilterMap))
	for k := range costFilterMap {
		costFilters = append(costFilters, k)
	}
	costHits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, sourceIDPtr, provider.AsPtr(), costFilters, time.Unix(endTime, 0), time.Unix(startTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}
	aggregatedCostHits := internal.AggregateServiceCosts(costHits)
	for k, hit := range aggregatedCostHits {
		costFilterMap[k] = hit
	}

	resourceTypeFilters := make([]string, 0, len(resourceTypeFilterMap))
	for k := range resourceTypeFilterMap {
		resourceTypeFilters = append(resourceTypeFilters, k)
	}

	sortMap := []map[string]interface{}{
		{
			"described_at": "desc",
		},
	}
	if sourceID != "" {
		hits, err := es.FetchConnectionResourceTypeTrendSummaryPage(h.client, &sourceID, resourceTypeFilters, time.Unix(endTime, 0).AddDate(0, 0, -1).UnixMilli(), time.Unix(endTime, 0).UnixMilli(), sortMap, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			if v, ok := resourceTypeFilterMap[hit.ResourceType]; ok && v == 0 {
				resourceTypeFilterMap[hit.ResourceType] = int64(hit.ResourceCount)
			}
		}
	} else {
		hits, err := es.FetchProviderResourceTypeTrendSummaryPage(h.client, provider, resourceTypeFilters, time.Unix(endTime, 0).AddDate(0, 0, -7).UnixMilli(), time.Unix(endTime, 0).UnixMilli(), sortMap, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			if v, ok := resourceTypeFilterMap[hit.ResourceType]; ok && v == 0 {
				resourceTypeFilterMap[hit.ResourceType] = int64(hit.ResourceCount)
			}
		}
	}

	var serviceSummaries []api.ServiceSummary
	for _, serviceNode := range serviceNodes {
		serviceSummary := api.ServiceSummary{
			CloudProvider: api.SourceType(serviceNode.CloudProvider.String()),
			ServiceName:   serviceNode.Name,
			ServiceCode:   serviceNode.ServiceCode,
			ResourceCount: nil,
			Cost:          nil,
		}
		for _, f := range serviceNode.SubTreeFilters {
			switch f.GetFilterType() {
			case FilterTypeCost:
				filter := f.(*FilterCostNode)
				if provider.IsNull() || provider.String() == filter.CloudProvider.String() {
					serviceSummary.Cost = internal.MergeCostMaps(serviceSummary.Cost, costFilterMap[filter.ServiceName])
				}
			case FilterTypeCloudResourceType:
				filter := f.(*FilterCloudResourceTypeNode)
				if provider.IsNull() || provider.String() == filter.CloudProvider.String() {
					count := int(resourceTypeFilterMap[filter.ResourceType])
					serviceSummary.ResourceCount = pointerAdd(serviceSummary.ResourceCount, &count)
				}
			}
		}
		serviceSummaries = append(serviceSummaries, serviceSummary)
	}

	switch sortBy {
	case "servicecode":
		sort.Slice(serviceSummaries, func(i, j int) bool {
			return serviceSummaries[i].ServiceCode < serviceSummaries[j].ServiceCode
		})
	case "resourcecount":
		sort.Slice(serviceSummaries, func(i, j int) bool {
			if serviceSummaries[i].ResourceCount == nil {
				return false
			}
			if serviceSummaries[j].ResourceCount == nil {
				return true
			}
			return *serviceSummaries[i].ResourceCount > *serviceSummaries[j].ResourceCount
		})
	case "cost":
		sort.Slice(serviceSummaries, func(i, j int) bool {
			if serviceSummaries[i].Cost == nil {
				return false
			}
			if serviceSummaries[j].Cost == nil {
				return true
			}
			if _, ok := serviceSummaries[i].Cost[DefaultCurrency]; !ok {
				return false
			}
			if _, ok := serviceSummaries[j].Cost[DefaultCurrency]; !ok {
				return true
			}

			return serviceSummaries[i].Cost[DefaultCurrency].Cost > serviceSummaries[j].Cost[DefaultCurrency].Cost
		})
	default:
		sort.Slice(serviceSummaries, func(i, j int) bool {
			return serviceSummaries[i].ServiceCode < serviceSummaries[j].ServiceCode
		})
	}

	res := api.ServiceSummaryResponse{
		TotalCount: len(serviceSummaries),
		Services:   internal.Paginate(pageNumber, pageSize, serviceSummaries),
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetResource godoc
// @Summary     Get details of a Resource
// @Description Getting resource details by id and resource type
// @Tags        resource
// @Accepts     json
// @Produce     json
// @Param       request body api.GetResourceRequest true "Request Body"
// @Router      /inventory/api/v1/resource [post]
func (h *HttpHandler) GetResource(ctx echo.Context) error {
	var req api.GetResourceRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	hash := sha256.New()
	hash.Write([]byte(req.ID))

	index := strings.ToLower(req.ResourceType)
	index = strings.ReplaceAll(index, "::", "_")
	index = strings.ReplaceAll(index, ".", "_")
	index = strings.ReplaceAll(index, "/", "_")
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"_id": fmt.Sprintf("%x", hash.Sum(nil)),
			},
		},
	}
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return err
	}

	var response api.GenericQueryResponse
	err = h.client.Search(ctx.Request().Context(), index, string(queryBytes), &response)
	if err != nil {
		return err
	}

	var source map[string]interface{}
	for _, hit := range response.Hits.Hits {
		source = hit.Source
	}

	var cells map[string]*proto.Column
	pluginProvider := steampipe.ExtractPlugin(req.ResourceType)
	pluginTableName := steampipe.ExtractTableName(req.ResourceType)
	if pluginProvider == steampipe.SteampipePluginAWS {
		desc, err := steampipe.ConvertToDescription(req.ResourceType, source)
		if err != nil {
			return err
		}

		cells, err = steampipe.AWSDescriptionToRecord(desc, pluginTableName)
		if err != nil {
			return err
		}
	} else if pluginProvider == steampipe.SteampipePluginAzure || pluginProvider == steampipe.SteampipePluginAzureAD {
		desc, err := steampipe.ConvertToDescription(req.ResourceType, source)
		if err != nil {
			return err
		}

		if pluginProvider == steampipe.SteampipePluginAzure {
			cells, err = steampipe.AzureDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				return err
			}
		} else {
			cells, err = steampipe.AzureADDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				return err
			}
		}
	} else {
		return errors.New("invalid provider")
	}

	resp := map[string]interface{}{}
	for k, v := range cells {
		if k == "tags" {
			var respTags []interface{}
			if jsonBytes := v.GetJsonValue(); jsonBytes != nil {
				var tags map[string]interface{}
				err = json.Unmarshal(jsonBytes, &tags)
				if err != nil {
					return err
				}
				for tagKey, tagValue := range tags {
					respTags = append(respTags, map[string]interface{}{
						"key":   tagKey,
						"value": tagValue,
					})
				}
			}
			resp["tags"] = respTags
			continue
		}

		var val string
		if x, ok := v.GetValue().(*proto.Column_DoubleValue); ok {
			val = fmt.Sprintf("%f", x.DoubleValue)
		} else if x, ok := v.GetValue().(*proto.Column_IntValue); ok {
			val = fmt.Sprintf("%d", x.IntValue)
		} else if x, ok := v.GetValue().(*proto.Column_StringValue); ok {
			val = x.StringValue
		} else if x, ok := v.GetValue().(*proto.Column_BoolValue); ok {
			val = fmt.Sprintf("%v", x.BoolValue)
		} else if x, ok := v.GetValue().(*proto.Column_TimestampValue); ok {
			val = fmt.Sprintf("%v", x.TimestampValue.AsTime())
		} else if x, ok := v.GetValue().(*proto.Column_IpAddrValue); ok {
			val = x.IpAddrValue
		} else if x, ok := v.GetValue().(*proto.Column_CidrRangeValue); ok {
			val = x.CidrRangeValue
		} else if x, ok := v.GetValue().(*proto.Column_JsonValue); ok {
			val = string(x.JsonValue)
		} else {
			val = fmt.Sprintf("unknown type: %v", v.GetValue())
		}

		if len(val) > 0 {
			resp[k] = val
		}
	}

	return ctx.JSON(200, resp)
}

// ListQueries godoc
// @Summary     List smart queries
// @Description Listing smart queries
// @Tags        smart_query
// @Produce     json
// @Param       request body     api.ListQueryRequest true "Request Body"
// @Success     200     {object} []api.SmartQueryItem
// @Router      /inventory/api/v1/query [get]
func (h *HttpHandler) ListQueries(ctx echo.Context) error {
	var req api.ListQueryRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var search *string
	if len(req.TitleFilter) > 0 {
		search = &req.TitleFilter
	}

	queries, err := h.db.GetQueriesWithFilters(search, req.Labels, req.ProviderFilter)
	if err != nil {
		return err
	}

	var result []api.SmartQueryItem
	for _, item := range queries {
		tags := map[string]string{}
		category := ""

		for _, tag := range item.Tags {
			tags[tag.Key] = tag.Value
			if strings.ToLower(tag.Key) == "category" {
				category = tag.Value
			}
		}
		result = append(result, api.SmartQueryItem{
			ID:          item.Model.ID,
			Provider:    item.Provider,
			Title:       item.Title,
			Category:    category,
			Description: item.Description,
			Query:       item.Query,
			Tags:        tags,
		})
	}
	return ctx.JSON(200, result)
}

// ListInsightsResults godoc
// @Summary List insight results for specified account
// @Tags    insights
// @Produce json
// @Param   request body api.ListInsightResultsRequest true "Request Body"
// @Success 200
// @Router  /inventory/api/v1/insight/results [get]
func (h *HttpHandler) ListInsightsResults(ctx echo.Context) error {
	var req api.ListInsightResultsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	query, err := es.FindInsightResults(req.DescriptionFilter, req.Labels)
	if err != nil {
		return err
	}

	var response es.InsightResultQueryResponse
	err = h.client.Search(context.Background(), insight.InsightsIndex,
		query, &response)
	if err != nil {
		return err
	}

	resp := api.ListInsightResultsResponse{}
	for _, item := range response.Hits.Hits {
		if item.Source.Internal {
			continue
		}

		resp.Results = append(resp.Results, api.InsightResult{
			SmartQueryID:     item.Source.SmartQueryID,
			Description:      item.Source.Description,
			Provider:         item.Source.Provider,
			Category:         item.Source.Category,
			Query:            item.Source.Query,
			ExecutedAt:       item.Source.ExecutedAt,
			Result:           item.Source.Result,
			LastDayValue:     item.Source.LastDayValue,
			LastWeekValue:    item.Source.LastWeekValue,
			LastQuarterValue: item.Source.LastQuarterValue,
			LastYearValue:    item.Source.LastYearValue,
		})
	}
	return ctx.JSON(200, resp)
}

// CountQueries godoc
// @Summary     Count smart queries
// @Description Counting smart queries
// @Tags        smart_query
// @Produce     json
// @Param       request body     api.ListQueryRequest true "Request Body"
// @Success     200     {object} int
// @Router      /inventory/api/v1/query/count [get]
func (h *HttpHandler) CountQueries(ctx echo.Context) error {
	var req api.ListQueryRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var search *string
	if len(req.TitleFilter) > 0 {
		search = &req.TitleFilter
	}

	c, err := h.db.CountQueriesWithFilters(search, req.Labels, req.ProviderFilter)
	if err != nil {
		return err
	}
	return ctx.JSON(200, *c)
}

// RunQuery godoc
// @Summary     Run a specific smart query
// @Description Run a specific smart query.
// @Description In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
// @Description Note that csv output doesn't process pagination and returns first 5000 records.
// @Tags        smart_query
// @Accepts     json
// @Produce     json,text/csv
// @Param       queryId path     string              true "QueryID"
// @Param       request body     api.RunQueryRequest true "Request Body"
// @Param       accept  header   string              true "Accept header" Enums(application/json,text/csv)
// @Success     200     {object} api.RunQueryResponse
// @Router      /inventory/api/v1/query/{queryId} [post]
func (h *HttpHandler) RunQuery(ctx echo.Context) error {
	var req api.RunQueryRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	queryId := ctx.Param("queryId")

	if accepts := ctx.Request().Header.Get("accept"); accepts != "" {
		mediaType, _, err := mime.ParseMediaType(accepts)
		if err == nil && mediaType == "text/csv" {
			req.Page = api.Page{
				No:   1,
				Size: 5000,
			}

			ctx.Response().Header().Set(echo.HeaderContentType, "text/csv")
			ctx.Response().WriteHeader(http.StatusOK)

			query, err := h.db.GetQuery(queryId)
			if err != nil {
				if err == pgx.ErrNoRows {
					return echo.NewHTTPError(http.StatusNotFound, "Query not found")
				}
				return err
			}

			resp, err := h.RunSmartQuery(query.Title, query.Query, &req)
			if err != nil {
				return err
			}

			err = Csv(resp.Headers, ctx.Response())
			if err != nil {
				return err
			}

			for _, row := range resp.Result {
				var cells []string
				for _, cell := range row {
					cells = append(cells, fmt.Sprint(cell))
				}
				err := Csv(cells, ctx.Response())
				if err != nil {
					return err
				}
			}

			ctx.Response().Flush()
			return nil
		}
	}

	query, err := h.db.GetQuery(queryId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "Query not found")
		}
		return err
	}
	resp, err := h.RunSmartQuery(query.Title, query.Query, &req)
	if err != nil {
		return err
	}
	return ctx.JSON(200, resp)
}

// GetLocations godoc
// @Summary     Get locations
// @Description Getting locations by provider
// @Tags        location
// @Produce     json
// @Param       provider path     string true "Provider" Enums(aws,azure)
// @Success     200      {object} []api.LocationByProviderResponse
// @Router      /inventory/api/v1/locations/{provider} [get]
func (h *HttpHandler) GetLocations(ctx echo.Context) error {
	provider := ctx.Param("provider")

	var locations []api.LocationByProviderResponse

	if provider == "aws" || provider == "all" {
		regions, err := h.client.NewEC2RegionPaginator(nil, nil)
		if err != nil {
			return err
		}

		res := map[string]interface{}{}
		for regions.HasNext() {
			regions, err := regions.NextPage(ctx.Request().Context())
			if err != nil {
				return err
			}

			for _, region := range regions {
				res[*region.Description.Region.RegionName] = 0
			}
		}
		for regionName := range res {
			locations = append(locations, api.LocationByProviderResponse{
				Name: regionName,
			})
		}
	}

	if provider == "azure" || provider == "all" {
		locs, err := h.client.NewLocationPaginator(nil, nil)
		if err != nil {
			return err
		}

		res := map[string]interface{}{}
		for locs.HasNext() {
			locpage, err := locs.NextPage(ctx.Request().Context())
			if err != nil {
				return err
			}

			for _, location := range locpage {
				res[*location.Description.Location.Name] = 0
			}
		}
		for regionName := range res {
			locations = append(locations, api.LocationByProviderResponse{
				Name: regionName,
			})
		}
	}

	return ctx.JSON(http.StatusOK, locations)
}

// GetAzureResources godoc
// @Summary     Get Azure resources
// @Description Getting Azure resources by filters.
// @Description In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
// @Description Note that csv output doesn't process pagination and returns first 5000 records.
// @Tags        inventory
// @Accept      json
// @Produce     json,text/csv
// @Param       request body     api.GetResourcesRequest true  "Request Body"
// @Param       accept  header   string                  true  "Accept header" Enums(application/json,text/csv)
// @Param       common  query    string                  false "Common filter" Enums(true,false,all)
// @Success     200     {object} api.GetAzureResourceResponse
// @Router      /inventory/api/v1/resources/azure [post]
func (h *HttpHandler) GetAzureResources(ctx echo.Context) error {
	provider := api.SourceCloudAzure
	commonQuery := ctx.QueryParam("common")
	var common *bool
	if commonQuery == "" || commonQuery == "true" {
		v := true
		common = &v
	} else if commonQuery == "false" {
		v := false
		common = &v
	}

	if accepts := ctx.Request().Header.Get("accept"); accepts != "" {
		mediaType, _, err := mime.ParseMediaType(accepts)
		if err == nil && mediaType == "text/csv" {
			return h.GetResourcesCSV(ctx, &provider, common)
		}
	}
	return h.GetResources(ctx, &provider, common)
}

// GetAWSResources godoc
// @Summary     Get AWS resources
// @Description Getting AWS resources by filters.
// @Description In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
// @Description Note that csv output doesn't process pagination and returns first 5000 records.
// @Tags        inventory
// @Accept      json
// @Produce     json,text/csv
// @Param       request body     api.GetResourcesRequest true  "Request Body"
// @Param       accept  header   string                  true  "Accept header" Enums(application/json,text/csv)
// @Param       common  query    string                  false "Common filter" Enums(true,false,all)
// @Success     200     {object} api.GetAWSResourceResponse
// @Router      /inventory/api/v1/resources/aws [post]
func (h *HttpHandler) GetAWSResources(ctx echo.Context) error {
	provider := api.SourceCloudAWS
	commonQuery := ctx.QueryParam("common")
	var common *bool
	if commonQuery == "" || commonQuery == "true" {
		v := true
		common = &v
	} else if commonQuery == "false" {
		v := false
		common = &v
	}

	if accepts := ctx.Request().Header.Get("accept"); accepts != "" {
		mediaType, _, err := mime.ParseMediaType(accepts)
		if err == nil && mediaType == "text/csv" {
			return h.GetResourcesCSV(ctx, &provider, common)
		}
	}
	return h.GetResources(ctx, &provider, common)
}

// GetAllResources godoc
// @Summary     Get resources
// @Description Getting all cloud providers resources by filters.
// @Description In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
// @Description Note that csv output doesn't process pagination and returns first 5000 records.
// @Description If sort by is empty, result will be sorted by the first column in ascending order.
// @Tags        inventory
// @Accept      json
// @Produce     json,text/csv
// @Param       request body     api.GetResourcesRequest true  "Request Body"
// @Param       accept  header   string                  true  "Accept header" Enums(application/json,text/csv)
// @Param       common  query    string                  false "Common filter" Enums(true,false,all)
// @Success     200     {object} api.GetResourcesResponse
// @Router      /inventory/api/v1/resources [post]
func (h *HttpHandler) GetAllResources(ctx echo.Context) error {
	commonQuery := ctx.QueryParam("common")
	var common *bool
	if commonQuery == "" || commonQuery == "true" {
		v := true
		common = &v
	} else if commonQuery == "false" {
		v := false
		common = &v
	}

	if accepts := ctx.Request().Header.Get("accept"); accepts != "" {
		mediaType, _, err := mime.ParseMediaType(accepts)
		if err == nil && mediaType == "text/csv" {
			return h.GetResourcesCSV(ctx, nil, common)
		}
	}
	return h.GetResources(ctx, nil, common)
}

// CountResources godoc
// @Summary Count resources
// @Tags    inventory
// @Accept  json
// @Produce json,text/csv
// @Success 200 {object} int64
// @Router  /inventory/api/v1/resources/count [post]
func (h *HttpHandler) CountResources(ctx echo.Context) error {
	value := 0
	toTime := time.Now()
	fromTime := toTime.Add(-24 * time.Hour)
	d, err := ExtractTrend(h.client, source.Nil, nil, fromTime.UnixMilli(), toTime.UnixMilli())
	if err != nil {
		return err
	}
	if len(d) > 0 {
		var maxItem int64
		for k, v := range d {
			if k > maxItem {
				maxItem, value = k, v
			}
		}
	}

	return ctx.JSON(http.StatusOK, value)
}

// GetConnectionSummary godoc
// @Summary Get connection summary
// @Tags    inventory
// @Accept  json
// @Produce json,text/csv
// @Success 200 {object} api.ConnectionSummaryResponse
// @Router  /inventory/api/v1/connection/{connection_id}/summary [post]
func (h *HttpHandler) GetConnectionSummary(ctx echo.Context) error {
	connectionID := ctx.Param("connection_id")

	metrics, err := h.db.FetchConnectionAllMetrics(connectionID)
	if err != nil {
		return err
	}

	cats, err := h.db.ListCategories()
	if err != nil {
		return err
	}

	resp := api.ConnectionSummaryResponse{
		Categories:    map[string]api.ConnectionSummaryCategory{},
		CloudServices: map[string]int{},
		ResourceTypes: map[string]int{},
	}
	for _, m := range metrics {
		cloudService := cloudservice.ServiceNameByResourceType(m.ResourceType)
		resp.ResourceTypes[m.ResourceType] += m.Count
		resp.CloudServices[cloudService] += m.Count
		for _, c := range cats {
			if c.CloudService == cloudService {
				v, ok := resp.Categories[c.Name]
				if !ok {
					v = api.ConnectionSummaryCategory{
						ResourceCount: 0,
						SubCategories: map[string]int{},
					}
				}

				v.ResourceCount += m.Count
				v.SubCategories[c.SubCategory] += m.Count

				resp.Categories[c.Name] = v
			}
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetProviderSummary godoc
// @Summary Get provider summary
// @Tags    inventory
// @Accept  json
// @Produce json,text/csv
// @Success 200 {object} api.ConnectionSummaryResponse
// @Router  /inventory/api/v1/provider/{provider}/summary [post]
func (h *HttpHandler) GetProviderSummary(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.Param("provider"))

	metrics, err := h.db.FetchProviderAllMetrics(provider)
	if err != nil {
		return err
	}

	cats, err := h.db.ListCategories()
	if err != nil {
		return err
	}

	resp := api.ConnectionSummaryResponse{
		Categories:    map[string]api.ConnectionSummaryCategory{},
		CloudServices: map[string]int{},
		ResourceTypes: map[string]int{},
	}
	for _, m := range metrics {
		cloudService := cloudservice.ServiceNameByResourceType(m.ResourceType)
		resp.ResourceTypes[m.ResourceType] += m.Count
		resp.CloudServices[cloudService] += m.Count
		for _, c := range cats {
			if c.CloudService == cloudService {
				v, ok := resp.Categories[c.Name]
				if !ok {
					v = api.ConnectionSummaryCategory{
						ResourceCount: 0,
						SubCategories: map[string]int{},
					}
				}

				v.ResourceCount += m.Count
				v.SubCategories[c.SubCategory] += m.Count

				resp.Categories[c.Name] = v
			}
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetResourcesFilters godoc
// @Summary     Get resource filters
// @Description Getting resource filters by filters.
// @Tags        inventory
// @Accept      json
// @Produce     json,text/csv
// @Param       request body     api.GetFiltersRequest true  "Request Body"
// @Param       common  query    string                false "Common filter" Enums(true,false,all)
// @Success     200     {object} api.GetFiltersResponse
// @Router      /inventory/api/v1/resources/filters [post]
func (h *HttpHandler) GetResourcesFilters(ctx echo.Context) error {
	commonQuery := ctx.QueryParam("common")
	var common *bool
	if commonQuery == "" || commonQuery == "true" {
		v := true
		common = &v
	} else if commonQuery == "false" {
		v := false
		common = &v
	}

	var req api.GetFiltersRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	query, err := es.BuildFilterQuery(req.Query, req.Filters, common)
	if err != nil {
		return err
	}

	var response es.LookupResourceAggregationResponse
	err = h.client.Search(context.Background(), InventorySummaryIndex,
		query, &response)
	if err != nil {
		return err
	}

	resp := api.GetFiltersResponse{}
	for _, item := range response.Aggregations.CategoryFilter.Buckets {
		resp.Filters.Category = append(resp.Filters.Category, item.Key)
	}
	for _, item := range response.Aggregations.ServiceFilter.Buckets {
		resp.Filters.Service = append(resp.Filters.Service, item.Key)
	}
	for _, item := range response.Aggregations.ResourceTypeFilter.Buckets {
		resp.Filters.ResourceType = append(resp.Filters.ResourceType, api.ResourceTypeFull{
			ResourceTypeARN:  item.Key,
			ResourceTypeName: cloudservice.ResourceTypeName(item.Key),
		})
	}

	connectionIDs := []string{}
	for _, item := range response.Aggregations.ConnectionFilter.Buckets {
		connectionIDs = append(connectionIDs, item.Key)
	}
	connections, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), connectionIDs)
	if err != nil {
		return err
	}
	for _, item := range response.Aggregations.ConnectionFilter.Buckets {
		connName := item.Key
		for _, c := range connections {
			if c.ID.String() == item.Key {
				connName = c.ConnectionName
			}
		}
		resp.Filters.Connections = append(resp.Filters.Connections, api.ConnectionFull{
			ID:   item.Key,
			Name: connName,
		})
	}
	for _, item := range response.Aggregations.LocationFilter.Buckets {
		resp.Filters.Location = append(resp.Filters.Location, item.Key)
	}
	for _, item := range response.Aggregations.SourceTypeFilter.Buckets {
		resp.Filters.Provider = append(resp.Filters.Provider, item.Key)
	}

	if len(req.Filters.TagKeys) > 0 {
		resp.Filters.TagValues = make(map[string][]string)
		for _, key := range req.Filters.TagKeys {
			set, err := h.rdb.SMembers(context.Background(), "tag-"+key).Result()
			if err != nil {
				return err
			}
			resp.Filters.TagValues[key] = set
		}
	} else {
		var cursor uint64 = 0
		for {
			var keys []string
			cmd := h.rdb.Scan(context.Background(), cursor, "tag-*", 0)
			fmt.Println(cmd)
			keys, cursor, err = cmd.Result()
			if err != nil {
				return err
			}

			if cursor == 0 {
				break
			}

			for _, key := range keys {
				resp.Filters.TagKeys = append(resp.Filters.TagKeys, key[4:])
			}
		}
	}

	return ctx.JSON(200, resp)
}

func (h *HttpHandler) RunSmartQuery(title, query string,
	req *api.RunQueryRequest) (*api.RunQueryResponse, error) {

	var err error
	lastIdx := (req.Page.No - 1) * req.Page.Size

	if req.Sorts == nil || len(req.Sorts) == 0 {
		req.Sorts = []api.SmartQuerySortItem{
			{
				Field:     "1",
				Direction: api.DirectionAscending,
			},
		}
	}
	if len(req.Sorts) > 1 {
		return nil, errors.New("multiple sort items not supported")
	}

	fmt.Println("smart query is: ", query)
	res, err := h.steampipeConn.Query(query, lastIdx, req.Page.Size, req.Sorts[0].Field, steampipe.DirectionType(req.Sorts[0].Direction))
	if err != nil {
		return nil, err
	}

	resp := api.RunQueryResponse{
		Title:   title,
		Query:   query,
		Headers: res.Headers,
		Result:  res.Data,
	}
	return &resp, nil
}

func (h *HttpHandler) GetResources(ctx echo.Context, provider *api.SourceType, commonFilter *bool) error {
	var req api.GetResourcesRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	res, err := api.QueryResources(ctx.Request().Context(), h.client, &req, provider, commonFilter)
	if err != nil {
		return err
	}

	if provider == nil {
		connectionID := map[string]string{}
		connectionName := map[string]string{}
		var sourceIds []string
		for _, resource := range res.AllResources {
			connectionName[resource.ProviderConnectionID] = "Unknown"
			connectionID[resource.ProviderConnectionID] = ""
			sourceIds = append(sourceIds, resource.ProviderConnectionID)
		}
		srcs, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), sourceIds)
		if err != nil {
			return err
		}
		for sourceId := range connectionName {
			for _, src := range srcs {
				if sourceId == src.ID.String() {
					connectionName[sourceId] = src.ConnectionName
					connectionID[sourceId] = src.ConnectionID
				}
			}
		}
		for idx := range res.AllResources {
			id := res.AllResources[idx].ProviderConnectionID
			res.AllResources[idx].ProviderConnectionID = connectionID[id]
			res.AllResources[idx].ProviderConnectionName = connectionName[id]
		}
		return ctx.JSON(http.StatusOK, api.GetResourcesResponse{
			Resources:  res.AllResources,
			TotalCount: res.TotalCount,
		})
	} else if *provider == api.SourceCloudAWS {
		connectionID := map[string]string{}
		connectionName := map[string]string{}
		var sourceIds []string
		for _, resource := range res.AWSResources {
			connectionName[resource.ProviderConnectionID] = "Unknown"
			connectionID[resource.ProviderConnectionID] = ""
			sourceIds = append(sourceIds, resource.ProviderConnectionID)
		}
		srcs, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), sourceIds)
		if err != nil {
			return err
		}
		for sourceId := range connectionName {
			for _, src := range srcs {
				if sourceId == src.ID.String() {
					connectionName[sourceId] = src.ConnectionName
					connectionID[sourceId] = src.ConnectionID
				}
			}
		}
		for idx := range res.AWSResources {
			id := res.AWSResources[idx].ProviderConnectionID
			res.AWSResources[idx].ProviderConnectionID = connectionID[id]
			res.AWSResources[idx].ProviderConnectionName = connectionName[id]
		}
		return ctx.JSON(http.StatusOK, api.GetAWSResourceResponse{
			Resources:  res.AWSResources,
			TotalCount: res.TotalCount,
		})
	} else if *provider == api.SourceCloudAzure {
		connectionID := map[string]string{}
		connectionName := map[string]string{}
		var sourceIds []string
		for _, resource := range res.AzureResources {
			connectionName[resource.ProviderConnectionID] = "Unknown"
			connectionID[resource.ProviderConnectionID] = ""
			sourceIds = append(sourceIds, resource.ProviderConnectionID)
		}
		srcs, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), sourceIds)
		if err != nil {
			return err
		}
		for sourceId := range connectionName {
			for _, src := range srcs {
				if sourceId == src.ID.String() {
					connectionName[sourceId] = src.ConnectionName
					connectionID[sourceId] = src.ConnectionID
				}
			}
		}
		for idx := range res.AzureResources {
			id := res.AzureResources[idx].ProviderConnectionID
			res.AzureResources[idx].ProviderConnectionID = connectionID[id]
			res.AzureResources[idx].ProviderConnectionName = connectionName[id]
		}
		return ctx.JSON(http.StatusOK, api.GetAzureResourceResponse{
			Resources:  res.AzureResources,
			TotalCount: res.TotalCount,
		})
	} else {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid provider")
	}
}

func Csv(record []string, w io.Writer) error {
	wr := csv.NewWriter(w)
	err := wr.Write(record)
	if err != nil {
		return err
	}
	wr.Flush()
	return nil
}

func (h *HttpHandler) GetResourcesCSV(ctx echo.Context, provider *api.SourceType, commonFilter *bool) error {
	var req api.GetResourcesRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Page = api.Page{
		No:   1,
		Size: 10000,
	}

	ctx.Response().Header().Set(echo.HeaderContentType, "text/csv")
	ctx.Response().WriteHeader(http.StatusOK)

	res, err := api.QueryResources(ctx.Request().Context(), h.client, &req, provider, commonFilter)
	if err != nil {
		return err
	}

	if provider == nil {
		err := Csv(api.AllResource{}.ToCSVHeaders(), ctx.Response())
		if err != nil {
			return err
		}

		for _, resource := range res.AllResources {
			err := Csv(resource.ToCSVRecord(), ctx.Response())
			if err != nil {
				return err
			}
		}
	} else if *provider == api.SourceCloudAWS {
		err := Csv(api.AWSResource{}.ToCSVHeaders(), ctx.Response())
		if err != nil {
			return err
		}

		for _, resource := range res.AWSResources {
			err := Csv(resource.ToCSVRecord(), ctx.Response())
			if err != nil {
				return err
			}
		}
	} else if *provider == api.SourceCloudAzure {
		err := Csv(api.AzureResource{}.ToCSVHeaders(), ctx.Response())
		if err != nil {
			return err
		}

		for _, resource := range res.AzureResources {
			err := Csv(resource.ToCSVRecord(), ctx.Response())
			if err != nil {
				return err
			}
		}
	} else {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid provider")
	}
	ctx.Response().Flush()
	return nil
}
