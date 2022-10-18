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

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"

	insight "gitlab.com/keibiengine/keibi-engine/pkg/insight/es"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
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

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")
	v2 := e.Group("/api/v2")

	v1.GET("/locations/:provider", h.GetLocations)

	v1.POST("/resources", h.GetAllResources)
	v1.POST("/resources/azure", h.GetAzureResources)
	v1.POST("/resources/aws", h.GetAWSResources)
	v1.GET("/resources/count", h.CountResources)

	v1.POST("/resources/filters", h.GetResourcesFilters)

	v1.POST("/resource", h.GetResource)

	v1.GET("/resources/trend", h.GetResourceGrowthTrend)
	v1.GET("/resources/top/growing/accounts", h.GetTopFastestGrowingAccountsByResourceCount)
	v1.GET("/resources/top/accounts", h.GetTopAccountsByResourceCount)
	v1.GET("/resources/top/regions", h.GetTopRegionsByResourceCount)
	v1.GET("/resources/top/services", h.GetTopServicesByResourceCount)
	v1.GET("/resources/categories", h.GetCategories)
	v1.GET("/accounts/resource/count", h.GetAccountsResourceCount)

	v1.GET("/resources/distribution", h.GetResourceDistribution)
	v1.GET("/services/distribution", h.GetServiceDistribution)

	v1.GET("/cost/top/accounts", h.GetTopAccountsByCost)
	v1.GET("/cost/top/services", h.GetTopServicesByCost)

	v1.GET("/query", h.ListQueries)
	v1.GET("/query/count", h.CountQueries)
	v1.POST("/query/:queryId", h.RunQuery)

	v1.GET("/insight/results", h.ListInsightsResults)

	v1.GET("/metrics/summary", h.GetSummaryMetrics)
	v1.GET("/metrics/categorized", h.GetCategorizedMetrics)
	v1.GET("/categories", h.ListCategories)

	v2.GET("/metrics/categorized", h.GetCategorizedMetricsV2)
	v2.GET("/categories", h.ListCategoriesV2)
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
// @Success 200      {object} []api.CategoriesResponse
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

	var response []api.CategoriesResponse
	for region, count := range locationDistribution {
		response = append(response, api.CategoriesResponse{
			CategoryName:  region,
			ResourceCount: count,
		})
	}
	sort.Slice(response, func(i, j int) bool {
		return response[i].ResourceCount > response[j].ResourceCount
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

// GetCategories godoc
// @Summary Return resource categories and number of resources
// @Tags    inventory
// @Accept  json
// @Produce json
// @Param   provider query    string true  "Provider"
// @Param   sourceId query    string false "SourceID"
// @Success 200      {object} []api.CategoriesResponse
// @Router  /inventory/api/v1/resources/categories [get]
func (h *HttpHandler) GetCategories(ctx echo.Context) error {
	var sourceID *string
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	if sID := ctx.QueryParam("sourceId"); sID != "" {
		sourceUUID, err := uuid.Parse(sID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid sourceID")
		}
		s := sourceUUID.String()
		sourceID = &s
	}

	res, err := GetCategories(h.client, provider, sourceID)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, res)
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
			v, err := GetResources(h.client, provider, sourceID, []string{awsResourceType})
			if err != nil {
				return err
			}
			if len(v) > 0 {
				aws = v[0]
			}
		}

		if azureResourceType != "" {
			v, err := GetResources(h.client, provider, sourceID, []string{azureResourceType})
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

		v, err := GetResources(h.client, provider, sourceID, resourceList)
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
// @Param   category    query    string true  "Category"
// @Param   subCategory query    string true  "SubCategory"
// @Success 200         {object} api.CategorizedMetricsResponse
// @Router  /inventory/api/v2/metrics/categorized [get]
func (h *HttpHandler) GetCategorizedMetricsV2(ctx echo.Context) error {
	category := ctx.QueryParam("category")
	if len(strings.TrimSpace(category)) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "category is required")
	}

	subCategory := ctx.QueryParam("subCategory")
	if len(strings.TrimSpace(subCategory)) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "subCategory is required")
	}

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

	cats, err := h.db.GetCategories(category, subCategory)
	if err != nil {
		return err
	}

	var resourceList []string
	for _, v := range cats {
		resourceList = append(resourceList, cloudservice.ResourceListByServiceName(v.CloudService)...)
	}

	if len(resourceList) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid category/subcategory")
	}

	v, err := GetResources(h.client, provider, sourceID, resourceList)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, v)
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
	err = h.client.Search(context.Background(), describe.InventorySummaryIndex,
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
