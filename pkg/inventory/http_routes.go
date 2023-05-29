package inventory

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	keibiaws "github.com/kaytu-io/kaytu-aws-describer/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"gorm.io/gorm"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/internal"
	"gitlab.com/keibiengine/keibi-engine/pkg/utils"

	api3 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	apiOnboard "gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"

	insight "gitlab.com/keibiengine/keibi-engine/pkg/insight/es"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/es"

	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/turbot/steampipe-plugin-sdk/v4/grpc/proto"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

const EsFetchPageSize = 10000
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
	v1.GET("/resources/regions", httpserver.AuthorizeHandler(h.GetRegionsByResourceCount, api3.ViewerRole))
	v1.GET("/resources/top/services", httpserver.AuthorizeHandler(h.GetTopServicesByResourceCount, api3.ViewerRole))

	resourcesV2 := v2.Group("/resources")
	resourcesV2.GET("/tag", httpserver.AuthorizeHandler(h.ListResourceTypeTags, api3.ViewerRole))
	resourcesV2.GET("/tag/:key", httpserver.AuthorizeHandler(h.GetResourceTypeTag, api3.ViewerRole))
	resourcesV2.GET("/metric", httpserver.AuthorizeHandler(h.ListResourceTypeMetricsHandler, api3.ViewerRole))
	resourcesV2.GET("/composition/:key", httpserver.AuthorizeHandler(h.ListResourceTypeComposition, api3.ViewerRole))
	resourcesV2.GET("/trend", httpserver.AuthorizeHandler(h.ListResourceTypeTrend, api3.ViewerRole))

	servicesV2 := v2.Group("/services")
	servicesV2.GET("/tag", httpserver.AuthorizeHandler(h.ListServiceTags, api3.ViewerRole))
	servicesV2.GET("/tag/:key", httpserver.AuthorizeHandler(h.GetServiceTag, api3.ViewerRole))
	servicesV2.GET("/metric", httpserver.AuthorizeHandler(h.ListServiceMetricsHandler, api3.ViewerRole))
	servicesV2.GET("/composition/:key", httpserver.AuthorizeHandler(h.ListServiceComposition, api3.ViewerRole))
	servicesV2.GET("/cost/trend", httpserver.AuthorizeHandler(h.ListServiceCostTrend, api3.ViewerRole))
	servicesV2.GET("/summary", httpserver.AuthorizeHandler(h.ListServiceSummaries, api3.ViewerRole))
	servicesV2.GET("/summary/:serviceName", httpserver.AuthorizeHandler(h.GetServiceSummary, api3.ViewerRole))

	connectionsV2 := v2.Group("/connections")
	connectionsV2.GET("/summary", httpserver.AuthorizeHandler(h.ListConnectionsSummary, api3.ViewerRole))
	connectionsV2.GET("/summary/:connectionId", httpserver.AuthorizeHandler(h.GetConnectionSummary, api3.ViewerRole))

	v1.GET("/accounts/resource/count", httpserver.AuthorizeHandler(h.GetAccountsResourceCount, api3.ViewerRole))
	v1.GET("/resources/distribution", httpserver.AuthorizeHandler(h.GetResourceDistribution, api3.ViewerRole))
	v1.GET("/services/distribution", httpserver.AuthorizeHandler(h.GetServiceDistribution, api3.ViewerRole))

	v1.GET("/cost/top/accounts", httpserver.AuthorizeHandler(h.GetTopAccountsByCost, api3.ViewerRole))
	v1.GET("/cost/top/services", httpserver.AuthorizeHandler(h.GetTopServicesByCost, api3.ViewerRole))

	v1.GET("/query", httpserver.AuthorizeHandler(h.ListQueries, api3.ViewerRole))
	v1.GET("/query/count", httpserver.AuthorizeHandler(h.CountQueries, api3.ViewerRole))
	v1.POST("/query/:queryId", httpserver.AuthorizeHandler(h.RunQuery, api3.EditorRole))

	insightsV2 := v2.Group("/insights")
	insightsV2.GET("", httpserver.AuthorizeHandler(h.ListInsights, api3.ViewerRole))
	insightsV2.GET("/:insightId/trend", httpserver.AuthorizeHandler(h.GetInsightTrend, api3.ViewerRole))
	insightsV2.GET("/peer/:insightPeerGroupId", httpserver.AuthorizeHandler(h.GetInsightPeerGroup, api3.ViewerRole))
	insightsV2.GET("/:insightId", httpserver.AuthorizeHandler(h.GetInsight, api3.ViewerRole))

	metadata := v2.Group("/metadata")
	metadata.GET("/connectors", httpserver.AuthorizeHandler(h.ListConnectorMetadata, api3.ViewerRole))
	metadata.GET("/connectors/:connector", httpserver.AuthorizeHandler(h.GetConnectorMetadata, api3.ViewerRole))
	metadata.GET("/services", httpserver.AuthorizeHandler(h.ListServiceMetadata, api3.ViewerRole))
	metadata.GET("/services/:serviceName", httpserver.AuthorizeHandler(h.GetServiceMetadata, api3.ViewerRole))
	metadata.GET("/resourcetype", httpserver.AuthorizeHandler(h.ListResourceTypeMetadata, api3.ViewerRole))
	metadata.GET("/resourcetype/:resourceType", httpserver.AuthorizeHandler(h.GetResourceTypeMetadata, api3.ViewerRole))
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
//
//	@Summary		Returns trend of resource count growth for specific account
//	@Description	Returns trend of resource count in the specified time window
//	@Description	In case of not specifying SourceID, Provider is used for filtering
//	@Security		BearerToken
//	@Tags			benchmarks
//	@Accept			json
//	@Produce		json
//	@Param			sourceId	query		string	false	"SourceID"
//	@Param			provider	query		string	false	"Provider"
//	@Param			timeWindow	query		string	false	"Time Window"	Enums(24h,1w,3m,1y,max)
//	@Success		200			{object}	[]api.TrendDataPoint
//	@Router			/inventory/api/v1/resources/trend [get]
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
		hits, err := es.FetchConnectionTrendSummaryPage(h.client, []string{sourceID}, fromTime, toTime, sortMap, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			datapoints[hit.DescribedAt] += hit.ResourceCount
		}
	} else {
		hits, err := es.FetchProviderTrendSummaryPage(h.client, []source.Type{provider}, fromTime, toTime, sortMap, EsFetchPageSize)
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
//
//	@Summary	Returns top n accounts of specified provider by cost
//	@Security	BearerToken
//	@Tags		cost
//	@Accept		json
//	@Produce	json
//	@Param		count		query		int		true	"Number of top accounts returning."
//	@Param		provider	query		string	true	"Provider"
//	@Success	200			{object}	[]api.TopAccountCostResponse
//	@Router		/inventory/api/v1/cost/top/accounts [get]
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

		var response keibiaws.CostExplorerByAccountMonthlySearchResponse
		err = h.client.Search(context.Background(), "aws_costexplorer_byaccountmonthly", query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			accountId := hit.Source.SourceID
			cost := *hit.Source.Description.UnblendedCostAmount

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
//
//	@Summary	Returns top n services of specified provider by cost
//	@Security	BearerToken
//	@Tags		cost
//	@Accept		json
//	@Produce	json
//	@Param		count		query		int		true	"Number of top services returning."
//	@Param		provider	query		string	true	"Provider"
//	@Param		sourceId	query		string	true	"SourceID"
//	@Success	200			{object}	[]api.TopServiceCostResponse
//	@Router		/inventory/api/v1/cost/top/services [get]
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

		var response keibiaws.CostExplorerByServiceMonthlySearchResponse
		err = h.client.Search(context.Background(), "aws_costexplorer_byservicemonthly", query, &response)
		if err != nil {
			return err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			serviceName := *hit.Source.Description.Dimension1
			cost := *hit.Source.Description.UnblendedCostAmount

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
//
//	@Summary	Returns top n accounts of specified provider by resource count
//	@Security	BearerToken
//	@Tags		benchmarks
//	@Accept		json
//	@Produce	json
//	@Param		count		query		int		true	"Number of top accounts returning."
//	@Param		provider	query		string	true	"Provider"
//	@Success	200			{object}	[]api.TopAccountResponse
//	@Router		/inventory/api/v1/resources/top/accounts [get]
func (h *HttpHandler) GetTopAccountsByResourceCount(ctx echo.Context) error {
	providers := source.ParseTypes(ctx.QueryParams()["provider"])
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
	hits, err := es.FetchConnectionResourcesSummaryPage(h.client, providers, nil, srt, count)
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
//
//	@Summary	Returns top n fastest growing accounts of specified provider in the specified time window by resource count
//	@Security	BearerToken
//	@Tags		benchmarks
//	@Accept		json
//	@Produce	json
//	@Param		count		query		int		true	"Number of top accounts returning."
//	@Param		provider	query		string	true	"Provider"
//	@Param		timeWindow	query		string	true	"TimeWindow"	Enums(1d,1w,3m,1y)
//	@Success	200			{object}	[]api.TopAccountResponse
//	@Router		/inventory/api/v1/resources/top/growing/accounts [get]
func (h *HttpHandler) GetTopFastestGrowingAccountsByResourceCount(ctx echo.Context) error {
	providers := source.ParseTypes(ctx.QueryParams()["provider"])

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

	summaryList, err := es.FetchConnectionResourcesSummaryPage(h.client, providers, nil, nil, EsFetchPageSize)
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
//
//	@Summary	Returns top n regions of specified provider by resource count
//	@Security	BearerToken
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Param		count		query		int			true	"count"
//	@Param		provider	query		string		false	"Provider"
//	@Param		sourceId	query		[]string	false	"SourceId"
//	@Success	200			{object}	[]api.LocationResponse
//	@Router		/inventory/api/v1/resources/top/regions [get]
func (h *HttpHandler) GetTopRegionsByResourceCount(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}

	sourceIDs := ctx.QueryParams()["sourceId"]
	if len(sourceIDs) == 0 {
		sourceIDs = nil
	}

	locationDistribution := map[string]int{}

	hits, err := es.FetchConnectionLocationsSummaryPage(h.client, provider, sourceIDs, nil, EsFetchPageSize)
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

// GetRegionsByResourceCount godoc
//
//	@Summary	Returns top n regions of specified provider by resource count
//	@Security	BearerToken
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Param		provider	query		string		false	"Provider"
//	@Param		sourceId	query		[]string	false	"SourceId"
//	@Param		pageSize	query		int			false	"page size - default is 20"
//	@Param		pageNumber	query		int			false	"page number - default is 1"
//	@Success	200			{object}	[]api.LocationResponse
//	@Router		/inventory/api/v1/resources/regions [get]
func (h *HttpHandler) GetRegionsByResourceCount(ctx echo.Context) error {
	var err error
	provider, _ := source.ParseType(ctx.QueryParam("provider"))

	sourceIDs := ctx.QueryParams()["sourceId"]
	if len(sourceIDs) == 0 {
		sourceIDs = nil
	}
	pageNumber, pageSize, err := internal.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	locationDistribution := map[string]int{}

	hits, err := es.FetchConnectionLocationsSummaryPage(h.client, provider, sourceIDs, nil, EsFetchPageSize)
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
		if *response[i].ResourceCount != *response[j].ResourceCount {
			return *response[i].ResourceCount > *response[j].ResourceCount
		}
		return response[i].Location < response[j].Location
	})

	return ctx.JSON(http.StatusOK, api.RegionsByResourceCountResponse{
		TotalCount: len(response),
		Regions:    utils.Paginate(pageNumber, pageSize, response),
	})
}

// GetTopServicesByResourceCount godoc
//
//	@Summary	Returns top n services of specified provider by resource count
//	@Security	BearerToken
//	@Tags		benchmarks
//	@Accept		json
//	@Produce	json
//	@Param		count		query		int		true	"Number of top ser"
//	@Param		provider	query		string	true	"Provider"
//	@Param		sourceId	query		string	false	"SourceID"
//	@Success	200			{object}	[]api.TopServicesResponse
//	@Router		/inventory/api/v1/resources/top/services [get]
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

// ListResourceTypeTags godoc
//
//	@Summary	Return list of the keys with possible values for filtering resources types
//	@Security	BearerToken
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	map[string][]string
//	@Router		/inventory/api/v2/resources/tag [get]
func (h *HttpHandler) ListResourceTypeTags(ctx echo.Context) error {
	tags, err := h.db.ListResourceTypeTagsKeysWithPossibleValues()
	if err != nil {
		return err
	}
	tags = trimPrivateTags(tags)
	return ctx.JSON(http.StatusOK, tags)
}

// GetResourceTypeTag godoc
//
//	@Summary	Return list of the possible values for filtering resources types with specified key
//	@Security	BearerToken
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Param		key	path		string	true	"Tag key"
//	@Success	200	{object}	[]string
//	@Router		/inventory/api/v2/resources/tag/{key} [get]
func (h *HttpHandler) GetResourceTypeTag(ctx echo.Context) error {
	tagKey := ctx.Param("key")
	if tagKey == "" || strings.HasPrefix(tagKey, KaytuPrivateTagPrefix) {
		return echo.NewHTTPError(http.StatusBadRequest, "tag key is invalid")
	}

	tags, err := h.db.GetResourceTypeTagPossibleValues(tagKey)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, tags)
}

func (h *HttpHandler) ListResourceTypeMetrics(tagMap map[string][]string, serviceNames []string, connectorTypes []source.Type, connectionIDs []string, timeAt int64) (int, []api.ResourceType, error) {
	resourceTypes, err := h.db.ListFilteredResourceTypes(tagMap, serviceNames, connectorTypes)
	if err != nil {
		return 0, nil, err
	}
	resourceTypeStrings := make([]string, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		resourceTypeStrings = append(resourceTypeStrings, resourceType.ResourceType)
	}

	metricIndexed, err := es.FetchResourceTypeCountAtTime(h.client, connectorTypes, connectionIDs, time.Unix(timeAt, 0), resourceTypeStrings, EsFetchPageSize)
	if err != nil {
		return 0, nil, err
	}

	apiResourceTypes := make([]api.ResourceType, 0, len(resourceTypes))
	totalCount := 0
	for _, resourceType := range resourceTypes {
		rt := api.ResourceType{
			Connector:     resourceType.Connector,
			ResourceType:  resourceType.ResourceType,
			ResourceLabel: resourceType.ResourceLabel,
			ServiceName:   resourceType.ServiceName,
			Tags:          trimPrivateTags(resourceType.GetTagsMap()),
			LogoURI:       resourceType.LogoURI,
			Count:         nil,
		}
		if count, ok := metricIndexed[strings.ToLower(resourceType.ResourceType)]; ok {
			rt.Count = &count
			totalCount += count
		}
		apiResourceTypes = append(apiResourceTypes, rt)
	}

	return totalCount, apiResourceTypes, nil
}

// ListResourceTypeMetricsHandler godoc
//
//	@Summary	Returns list of resource types with metrics of each type based on the given input filters
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Param		tag				query		string		false	"Key-Value tags in key=value format to filter by"
//	@Param		servicename		query		string		false	"Service names to filter by"
//	@Param		connector		query		source.Type	false	"Connector type to filter by"
//	@Param		connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param		endTime			query		string		false	"timestamp for resource count in epoch seconds"
//	@Param		startTime		query		string		false	"timestamp for resource count change comparison in epoch seconds"
//	@Param		sortBy			query		string		false	"Sort by field - default is count"	Enums(name,count)
//	@Param		pageSize		query		int			false	"page size - default is 20"
//	@Param		pageNumber		query		int			false	"page number - default is 1"
//	@Success	200				{object}	api.ListResourceTypeMetricsResponse
//	@Router		/inventory/api/v2/resources/metric [get]
func (h *HttpHandler) ListResourceTypeMetricsHandler(ctx echo.Context) error {
	var err error
	tagMap := internal.TagStringsToTagMap(ctx.QueryParams()["tag"])
	serviceNames := ctx.QueryParams()["servicename"]
	connectorTypes := source.ParseTypes(ctx.QueryParams()["connector"])
	connectionIDs := ctx.QueryParams()["connectionId"]
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid endTime value")
		}
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Unix(endTime, 0).AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid startTime value")
		}
	}
	pageNumber, pageSize, err := internal.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	sortBy := strings.ToLower(ctx.QueryParam("sortBy"))
	if sortBy == "" {
		sortBy = "count"
	}
	if sortBy != "name" && sortBy != "count" {
		return ctx.JSON(http.StatusBadRequest, "invalid sortBy value")
	}

	totalCount, apiResourceTypes, err := h.ListResourceTypeMetrics(tagMap, serviceNames, connectorTypes, connectionIDs, endTime)
	if err != nil {
		return err
	}
	apiResourceTypesMap := make(map[string]api.ResourceType, len(apiResourceTypes))
	for _, apiResourceType := range apiResourceTypes {
		apiResourceTypesMap[apiResourceType.ResourceType] = apiResourceType
	}

	_, oldApiResourceTypes, err := h.ListResourceTypeMetrics(tagMap, serviceNames, connectorTypes, connectionIDs, startTime)
	if err != nil {
		return err
	}
	for _, oldApiResourceType := range oldApiResourceTypes {
		if apiResourceType, ok := apiResourceTypesMap[oldApiResourceType.ResourceType]; ok {
			if apiResourceType.Count == nil {
				continue
			}
			if oldApiResourceType.Count == nil {
				apiResourceType.CountChangePercent = utils.GetPointer(float64(1))
				continue
			}
			apiResourceType.CountChangePercent = utils.GetPointer(float64((float64(*apiResourceType.Count) - float64(*oldApiResourceType.Count)) / float64(*apiResourceType.Count)))
			apiResourceTypesMap[oldApiResourceType.ResourceType] = apiResourceType
		}
	}

	apiResourceTypes = make([]api.ResourceType, 0, len(apiResourceTypesMap))
	for _, apiResourceType := range apiResourceTypesMap {
		apiResourceTypes = append(apiResourceTypes, apiResourceType)
	}

	switch sortBy {
	case "name":
		sort.Slice(apiResourceTypes, func(i, j int) bool {
			return apiResourceTypes[i].ResourceType < apiResourceTypes[j].ResourceType
		})
	case "count":
		sort.Slice(apiResourceTypes, func(i, j int) bool {
			if apiResourceTypes[i].Count == nil {
				return false
			}
			if apiResourceTypes[j].Count == nil {
				return true
			}
			return *apiResourceTypes[i].Count > *apiResourceTypes[j].Count
		})
	}

	result := api.ListResourceTypeMetricsResponse{
		TotalCount:         totalCount,
		TotalResourceTypes: len(apiResourceTypes),
		ResourceTypes:      utils.Paginate(pageNumber, pageSize, apiResourceTypes),
	}

	return ctx.JSON(http.StatusOK, result)
}

// ListResourceTypeComposition godoc
//
//	@Summary	Return tag values with most resources for the given key
//	@Security	BearerToken
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Param		top				query		int			true	"How many top values to return default is 5"
//	@Param		connector		query		source.Type	false	"Connector types to filter by"
//	@Param		connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param		time			query		string		false	"timestamp for resource count in epoch seconds"
//	@Success	200				{object}	api.ListResourceTypeCompositionResponse
//	@Router		/inventory/api/v2/resources/composition/{key} [get]
func (h *HttpHandler) ListResourceTypeComposition(ctx echo.Context) error {
	var err error
	tagKey := ctx.Param("key")
	if tagKey == "" || strings.HasPrefix(tagKey, KaytuPrivateTagPrefix) {
		return echo.NewHTTPError(http.StatusBadRequest, "tag key is invalid")
	}
	topStr := ctx.QueryParam("top")
	top := int64(5)
	if topStr != "" {
		top, err = strconv.ParseInt(topStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid top value")
		}

	}
	connectorTypes := source.ParseTypes(ctx.QueryParams()["connector"])
	connectionIDs := ctx.QueryParams()["connectionId"]
	timeStr := ctx.QueryParam("time")
	timeAt := time.Now().Unix()
	if timeStr != "" {
		timeAt, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}

	resourceTypes, err := h.db.ListFilteredResourceTypes(map[string][]string{tagKey: nil}, nil, connectorTypes)
	if err != nil {
		return err
	}
	resourceTypeStrings := make([]string, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		resourceTypeStrings = append(resourceTypeStrings, resourceType.ResourceType)
	}
	metricIndexed, err := es.FetchResourceTypeCountAtTime(h.client, connectorTypes, connectionIDs, time.Unix(timeAt, 0), resourceTypeStrings, EsFetchPageSize)
	if err != nil {
		return err
	}

	valueCountMap := make(map[string]int)
	totalCount := 0
	for _, resourceType := range resourceTypes {
		for _, tagValue := range resourceType.GetTagsMap()[tagKey] {
			valueCountMap[tagValue] += metricIndexed[strings.ToLower(resourceType.ResourceType)]
			totalCount += metricIndexed[strings.ToLower(resourceType.ResourceType)]
			break
		}
	}

	type strIntPair struct {
		str     string
		integer int
	}
	valueCountPairs := make([]strIntPair, 0, len(valueCountMap))
	for value, count := range valueCountMap {
		valueCountPairs = append(valueCountPairs, strIntPair{str: value, integer: count})
	}
	sort.Slice(valueCountPairs, func(i, j int) bool {
		return valueCountPairs[i].integer > valueCountPairs[j].integer
	})

	apiResult := api.ListResourceTypeCompositionResponse{
		TotalCount:      totalCount,
		TotalValueCount: len(valueCountMap),
		TopValues:       make(map[string]int),
		Others:          0,
	}

	for i, pair := range valueCountPairs {
		if i < int(top) {
			apiResult.TopValues[pair.str] = pair.integer
		} else {
			apiResult.Others += pair.integer
		}
	}

	return ctx.JSON(http.StatusOK, apiResult)
}

// ListResourceTypeTrend godoc
//
//	@Summary	Returns list of resource counts over the course of the specified time frame based on the given input filters
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Param		tag				query		string		false	"Key-Value tags in key=value format to filter by"
//	@Param		servicename		query		string		false	"Service names to filter by"
//	@Param		connector		query		source.Type	false	"Connector type to filter by"
//	@Param		connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param		startTime		query		string		false	"timestamp for start in epoch seconds"
//	@Param		endTime			query		string		false	"timestamp for end in epoch seconds"
//	@Param		datapointCount	query		string		false	"maximum number of datapoints to return, default is 30"
//	@Success	200				{object}	[]api.ResourceTypeTrendDatapoint
//	@Router		/inventory/api/v2/resources/trend [get]
func (h *HttpHandler) ListResourceTypeTrend(ctx echo.Context) error {
	var err error
	tagMap := internal.TagStringsToTagMap(ctx.QueryParams()["tag"])
	serviceNames := ctx.QueryParams()["servicename"]
	connectorTypes := source.ParseTypes(ctx.QueryParams()["connector"])
	connectionIDs := ctx.QueryParams()["connectionId"]

	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Unix(endTime, 0).Add(-1 * 30 * 24 * time.Hour).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}

	datapointCountStr := ctx.QueryParam("datapointCount")
	datapointCount := int64(30)
	if datapointCountStr != "" {
		datapointCount, err = strconv.ParseInt(datapointCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid datapointCount")
		}
	}

	resourceTypes, err := h.db.ListFilteredResourceTypes(tagMap, serviceNames, connectorTypes)
	if err != nil {
		return err
	}
	resourceTypeStrings := make([]string, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		resourceTypeStrings = append(resourceTypeStrings, resourceType.ResourceType)
	}

	type countTimePair struct {
		count int
		time  time.Time
	}

	summarizeJobIDCountMap := make(map[uint]countTimePair)
	if len(connectionIDs) != 0 {
		hits, err := es.FetchConnectionResourceTypeTrendSummaryPage(h.client, connectionIDs, resourceTypeStrings, time.Unix(startTime, 0), time.Unix(endTime, 0), []map[string]any{{"described_at": "asc"}}, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			if v, ok := summarizeJobIDCountMap[hit.SummarizeJobID]; !ok {
				summarizeJobIDCountMap[hit.SummarizeJobID] = countTimePair{count: hit.ResourceCount, time: time.UnixMilli(hit.DescribedAt)}
			} else {
				v.count += hit.ResourceCount
				summarizeJobIDCountMap[hit.SummarizeJobID] = v
			}
		}
	} else {
		hits, err := es.FetchProviderResourceTypeTrendSummaryPage(h.client, connectorTypes, resourceTypeStrings, time.Unix(startTime, 0), time.Unix(endTime, 0), []map[string]any{{"described_at": "asc"}}, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			if v, ok := summarizeJobIDCountMap[hit.SummarizeJobID]; !ok {
				summarizeJobIDCountMap[hit.SummarizeJobID] = countTimePair{count: hit.ResourceCount, time: time.UnixMilli(hit.DescribedAt)}
			} else {
				v.count += hit.ResourceCount
				summarizeJobIDCountMap[hit.SummarizeJobID] = v
			}
		}
	}

	apiDatapoints := make([]api.ResourceTypeTrendDatapoint, 0, len(summarizeJobIDCountMap))
	for _, v := range summarizeJobIDCountMap {
		apiDatapoints = append(apiDatapoints, api.ResourceTypeTrendDatapoint{Count: v.count, Date: v.time})
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})
	apiDatapoints = internal.DownSampleResourceTypeTrendDatapoints(apiDatapoints, int(datapointCount))

	return ctx.JSON(http.StatusOK, apiDatapoints)
}

// ListServiceTags godoc
//
//	@Summary	Return list of the keys with possible values for filtering services
//	@Security	BearerToken
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	map[string][]string
//	@Router		/inventory/api/v2/services/tag [get]
func (h *HttpHandler) ListServiceTags(ctx echo.Context) error {
	tags, err := h.db.ListServiceTagsKeysWithPossibleValues()
	if err != nil {
		return err
	}
	tags = trimPrivateTags(tags)
	return ctx.JSON(http.StatusOK, tags)
}

// GetServiceTag godoc
//
//	@Summary	Return list of the possible values for filtering services with specified key
//	@Security	BearerToken
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Param		key	path		string	true	"Tag key"
//	@Success	200	{object}	[]string
//	@Router		/inventory/api/v2/services/tag/{key} [get]
func (h *HttpHandler) GetServiceTag(ctx echo.Context) error {
	tagKey := ctx.Param("key")
	if tagKey == "" || strings.HasPrefix(tagKey, KaytuPrivateTagPrefix) {
		return echo.NewHTTPError(http.StatusBadRequest, "tag key is invalid")
	}

	tags, err := h.db.GetResourceTypeTagPossibleValues(tagKey)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, tags)
}

// ListServiceMetricsHandler godoc
//
//	@Summary	Returns list of resource types with metrics of each type based on the given input filters
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Param		tag				query		string		false	"Key-Value tags in key=value format to filter by"
//	@Param		connector		query		source.Type	false	"Connector type to filter by"
//	@Param		connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param		startTime		query		string		false	"timestamp for start of cost aggregation in epoch seconds"
//	@Param		endTime			query		string		false	"timestamp for end of cost aggregation in epoch seconds"
//	@Param		sortBy			query		string		false	"Sort by field - default is cost"	Enums(name,cost)
//	@Param		pageSize		query		int			false	"page size - default is 20"
//	@Param		pageNumber		query		int			false	"page number - default is 1"
//	@Success	200				{object}	api.ListServiceMetricsResponse
//	@Router		/inventory/api/v2/services/metric [get]
func (h *HttpHandler) ListServiceMetricsHandler(ctx echo.Context) error {
	var err error
	tagMap := internal.TagStringsToTagMap(ctx.QueryParams()["tag"])
	connectorTypes := source.ParseTypes(ctx.QueryParams()["connector"])
	connectionIDs := ctx.QueryParams()["connectionId"]
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, err.Error())
		}
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Unix(endTime, 0).AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, err.Error())
		}
	}
	pageNumber, pageSize, err := internal.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	sortBy := strings.ToLower(ctx.QueryParam("sortBy"))
	if sortBy == "" {
		sortBy = "cost"
	}
	if sortBy != "name" && sortBy != "cost" {
		return ctx.JSON(http.StatusBadRequest, "invalid sortBy value")
	}

	services, err := h.db.ListFilteredServices(tagMap, connectorTypes)
	if err != nil {
		return err
	}
	costFilterNamesMap := make(map[string]bool)
	for _, service := range services {
		if v, ok := service.GetTagsMap()[KaytuServiceCostTag]; ok {
			for _, costFilterName := range v {
				costFilterNamesMap[costFilterName] = true
			}
		}
	}
	costFilterNames := make([]string, 0, len(costFilterNamesMap))
	for costFilterName := range costFilterNamesMap {
		costFilterNames = append(costFilterNames, costFilterName)
	}

	costHits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, connectionIDs, connectorTypes, costFilterNames, time.Unix(endTime, 0), time.Unix(startTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}
	aggregatedCostHits := internal.AggregateServiceCosts(costHits)
	if err != nil {
		return err
	}

	apiServices := make([]api.Service, 0, len(services))
	totalCost := float64(0)
	for _, service := range services {
		srv := api.Service{
			Connector:    service.Connector,
			ServiceName:  service.ServiceName,
			ServiceLabel: service.ServiceLabel,
			Tags:         trimPrivateTags(service.GetTagsMap()),
			LogoURI:      service.LogoURI,
			Cost:         nil,
		}
		if v, ok := service.GetTagsMap()[KaytuServiceCostTag]; ok {
			for _, costFilterName := range v {
				if costWithUnit, ok := aggregatedCostHits[costFilterName]; ok {
					defaultCost := costWithUnit[DefaultCurrency]
					srv.Cost = utils.PAdd(srv.Cost, &defaultCost.Cost)
					totalCost += defaultCost.Cost
				}
			}
		}
		apiServices = append(apiServices, srv)
	}
	switch sortBy {
	case "name":
		sort.Slice(apiServices, func(i, j int) bool {
			return apiServices[i].ServiceName < apiServices[j].ServiceName
		})
	case "cost":
		sort.Slice(apiServices, func(i, j int) bool {
			if apiServices[i].Cost == nil {
				return false
			}
			if apiServices[j].Cost == nil {
				return true
			}
			return *apiServices[i].Cost > *apiServices[j].Cost
		})
	}

	result := api.ListServiceMetricsResponse{
		TotalCost:     totalCost,
		TotalServices: len(apiServices),
		Services:      utils.Paginate(pageNumber, pageSize, apiServices),
	}
	return ctx.JSON(http.StatusOK, result)
}

// ListServiceComposition godoc
//
//	@Summary	Return tag values with most cost for the given key
//	@Security	BearerToken
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Param		top				query		int			true	"How many top values to return default is 5"
//	@Param		connector		query		source.Type	false	"Connector type to filter by"
//	@Param		connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param		startTime		query		string		false	"timestamp for start of cost aggregation in epoch seconds"
//	@Param		endTime			query		string		false	"timestamp for end of cost aggregation in epoch seconds"
//	@Success	200				{object}	api.ListServiceCostCompositionResponse
//	@Router		/inventory/api/v2/services/composition/{key} [get]
func (h *HttpHandler) ListServiceComposition(ctx echo.Context) error {
	var err error
	tagKey := ctx.Param("key")
	if tagKey == "" || strings.HasPrefix(tagKey, KaytuPrivateTagPrefix) {
		return echo.NewHTTPError(http.StatusBadRequest, "tag key is invalid")
	}
	topStr := ctx.QueryParam("top")
	top := int64(5)
	if topStr != "" {
		top, err = strconv.ParseInt(topStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid top value")
		}

	}
	connectorTypes := source.ParseTypes(ctx.QueryParams()["connector"])
	connectionIDs := ctx.QueryParams()["connectionId"]
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Unix(endTime, 0).AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}

	services, err := h.db.ListFilteredServices(map[string][]string{tagKey: nil}, connectorTypes)
	if err != nil {
		return err
	}
	costFilterNamesMap := make(map[string]bool)
	for _, service := range services {
		if v, ok := service.GetTagsMap()[KaytuServiceCostTag]; ok {
			for _, costFilterName := range v {
				costFilterNamesMap[costFilterName] = true
			}
		}
	}
	costFilterNames := make([]string, 0, len(costFilterNamesMap))
	for costFilterName := range costFilterNamesMap {
		costFilterNames = append(costFilterNames, costFilterName)
	}

	costHits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, connectionIDs, connectorTypes, costFilterNames, time.Unix(endTime, 0), time.Unix(startTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}
	aggregatedCostHits := internal.AggregateServiceCosts(costHits)
	if err != nil {
		return err
	}

	valueCostMap := make(map[string]float64)
	totalCount := float64(0)
	for _, service := range services {
		for _, tagValue := range service.GetTagsMap()[tagKey] {
			for _, costFilterName := range service.GetTagsMap()[KaytuServiceCostTag] {
				valueCostMap[tagValue] += aggregatedCostHits[costFilterName][DefaultCurrency].Cost
				totalCount += aggregatedCostHits[costFilterName][DefaultCurrency].Cost
			}
			break
		}
	}

	type strFloatPair struct {
		str   string
		float float64
	}
	valueCostPairs := make([]strFloatPair, 0, len(valueCostMap))
	for value, count := range valueCostMap {
		valueCostPairs = append(valueCostPairs, strFloatPair{str: value, float: count})
	}
	sort.Slice(valueCostPairs, func(i, j int) bool {
		return valueCostPairs[i].float > valueCostPairs[j].float
	})

	apiResult := api.ListServiceCostCompositionResponse{
		TotalCost:       totalCount,
		TotalValueCount: len(valueCostMap),
		TopValues:       make(map[string]float64),
		Others:          0,
	}

	for i, pair := range valueCostPairs {
		if i < int(top) {
			apiResult.TopValues[pair.str] = pair.float
		} else {
			apiResult.Others += pair.float
		}
	}

	return ctx.JSON(http.StatusOK, apiResult)
}

// ListServiceCostTrend godoc
//
//	@Summary	Returns list of costs over the course of the specified time frame based on the given input filters
//	@Tags		inventory
//	@Accept		json
//	@Produce	json
//	@Param		tag				query		string		false	"Key-Value tags in key=value format to filter by"
//	@Param		connector		query		source.Type	false	"Connector type to filter by"
//	@Param		connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param		startTime		query		string		false	"timestamp for start in epoch seconds"
//	@Param		endTime			query		string		false	"timestamp for end in epoch seconds"
//	@Param		datapointCount	query		string		false	"maximum number of datapoints to return, default is 30"
//	@Success	200				{object}	[]api.CostTrendDatapoint
//	@Router		/inventory/api/v2/services/cost/trend [get]
func (h *HttpHandler) ListServiceCostTrend(ctx echo.Context) error {
	var err error
	tagMap := internal.TagStringsToTagMap(ctx.QueryParams()["tag"])
	connectorTypes := source.ParseTypes(ctx.QueryParams()["connector"])
	connectionIDs := ctx.QueryParams()["connectionId"]

	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Unix(endTime, 0).Add(-1 * 30 * 24 * time.Hour).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}

	datapointCountStr := ctx.QueryParam("datapointCount")
	datapointCount := int64(30)
	if datapointCountStr != "" {
		datapointCount, err = strconv.ParseInt(datapointCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid datapointCount")
		}
	}

	services, err := h.db.ListFilteredServices(tagMap, connectorTypes)
	if err != nil {
		return err
	}
	costFilterNamesMap := make(map[string]bool)
	for _, service := range services {
		if v, ok := service.GetTagsMap()[KaytuServiceCostTag]; ok {
			for _, costFilterName := range v {
				costFilterNamesMap[costFilterName] = true
			}
		}
	}
	costFilterNames := make([]string, 0, len(costFilterNamesMap))
	for costFilterName := range costFilterNamesMap {
		costFilterNames = append(costFilterNames, costFilterName)
	}

	costHits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, connectionIDs, connectorTypes, costFilterNames, time.Unix(endTime, 0), time.Unix(startTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}

	type costTimePair struct {
		cost float64
		time time.Time
	}

	summarizeJobIDCountMap := make(map[uint]costTimePair)

	for _, hitArr := range costHits {
		for _, hit := range hitArr {
			cost, _ := hit.GetCostAndUnit()
			if v, ok := summarizeJobIDCountMap[hit.SummarizeJobID]; !ok {
				summarizeJobIDCountMap[hit.SummarizeJobID] = costTimePair{cost: cost, time: time.Unix(hit.SummarizeJobTime, 0)}
			} else {
				v.cost += cost
				summarizeJobIDCountMap[hit.SummarizeJobID] = v
			}
		}
	}

	apiDatapoints := make([]api.CostTrendDatapoint, 0, len(summarizeJobIDCountMap))
	for _, v := range summarizeJobIDCountMap {
		apiDatapoints = append(apiDatapoints, api.CostTrendDatapoint{Cost: v.cost, Date: v.time})
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})
	apiDatapoints = internal.DownSampleCostTrendDatapoints(apiDatapoints, int(datapointCount))

	return ctx.JSON(http.StatusOK, apiDatapoints)
}

// GetAccountsResourceCount godoc
//
//	@Summary	Returns resource count of accounts
//	@Security	BearerToken
//	@Tags		benchmarks
//	@Accept		json
//	@Produce	json
//	@Param		provider	query		string		true	"Provider"
//	@Param		sourceId	query		[]string	false	"SourceID"
//	@Success	200			{object}	[]api.ConnectionResourceCountResponse
//	@Router		/inventory/api/v1/accounts/resource/count [get]
func (h *HttpHandler) GetAccountsResourceCount(ctx echo.Context) error {
	connectors := source.ParseTypes(ctx.QueryParams()["provider"])
	sourceId := ctx.QueryParam("sourceId")
	var sourceIdPtr *string
	if sourceId != "" {
		sourceIdPtr = &sourceId
	}

	res := map[string]api.ConnectionResourceCountResponse{}

	var err error
	var allSources []apiOnboard.Source
	if sourceId == "" {
		allSources, err = h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), connectors)
	} else {
		allSources, err = h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), []string{sourceId})
	}
	if err != nil {
		return err
	}

	for _, src := range allSources {
		res[src.ID.String()] = api.ConnectionResourceCountResponse{
			SourceID:                src.ID.String(),
			Connector:               src.Type,
			ConnectorConnectionName: src.ConnectionName,
			ConnectorConnectionID:   src.ConnectionID,
			LifecycleState:          string(src.LifecycleState),
			OnboardDate:             src.OnboardDate,
		}
	}

	hits, err := es.FetchConnectionResourcesSummaryPage(h.client, connectors, sourceIdPtr, nil, EsFetchPageSize)
	for _, hit := range hits {
		if v, ok := res[hit.SourceID]; ok {
			v.ResourceCount += hit.ResourceCount
			v.LastInventory = time.UnixMilli(hit.DescribedAt)
			res[hit.SourceID] = v
		}
	}
	var response []api.ConnectionResourceCountResponse
	for _, v := range res {
		response = append(response, v)
	}
	return ctx.JSON(http.StatusOK, response)
}

// ListConnectionsSummary godoc
//
//	@Summary	Returns resource count of accounts
//	@Security	BearerToken
//	@Tags		benchmarks
//	@Accept		json
//	@Produce	json
//	@Param		connector		query		source.Type	true	"Connector"
//	@Param		connectionId	query		string		false	"Connection IDs"
//	@Param		healthState		query		string		false	"Source Healthstate"	Enums(healthy,unhealthy)
//	@Param		lifecycleState	query		string		false	"lifecycle state filter"
//	@Param		pageSize		query		int			false	"page size - default is 20"
//	@Param		pageNumber		query		int			false	"page number - default is 1"
//	@Param		startTime		query		int			false	"start time in unix seconds"
//	@Param		endTime			query		int			false	"end time in unix seconds"
//	@Param		sortBy			query		string		false	"column to sort by - default is cost"	Enums(onboard_date,resource_count,cost)
//	@Success	200				{object}	api.ListConnectionsResponse
//	@Router		/inventory/api/v2/connections/summary [get]
func (h *HttpHandler) ListConnectionsSummary(ctx echo.Context) error {
	var err error

	connectors := source.ParseTypes(ctx.QueryParams()["connector"])
	connectionIDs := ctx.QueryParams()["connectionId"]

	pageNumber, pageSize, err := internal.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	sortBy := ctx.QueryParam("sortBy")
	if sortBy == "" {
		sortBy = "cost"
	}
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now()
	if endTimeStr != "" {
		endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid integer")
		}
		endTime = time.Unix(endTimeUnix, 0)
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr != "" {
		startTimeUnix, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "startTime is not a valid integer")
		}
		startTime = time.Unix(startTimeUnix, 0)
	}

	healthState := ctx.QueryParam("healthState")
	enabledState := ctx.QueryParam("enabledState")

	res := map[string]api.Connection{}

	var allSources []apiOnboard.Source
	if len(connectionIDs) == 0 {
		allSources, err = h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), connectors)
	} else {
		allSources, err = h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), connectionIDs)
	}
	if err != nil {
		return err
	}

	unhealthyCount := 0
	disabledCount := 0
	for _, src := range allSources {
		if healthState != "" && healthState != string(src.HealthState) {
			continue
		}
		if enabledState != "" && strings.ToLower(enabledState) != strings.ToLower(string(src.LifecycleState)) {
			continue
		}
		res[src.ID.String()] = api.Connection{
			ConnectionID:            src.ID.String(),
			Connector:               src.Type,
			ConnectorConnectionName: src.ConnectionName,
			ConnectorConnectionID:   src.ConnectionID,
			LifecycleState:          string(src.LifecycleState),
			OnboardDate:             src.OnboardDate,
			HealthState:             src.HealthState,
			LastHealthCheckTime:     src.LastHealthCheckTime,
			HealthReason:            src.HealthReason,
			Metadata:                src.Metadata,
		}

		if src.HealthState == source.HealthStatusUnhealthy {
			unhealthyCount++
		}
		if src.LifecycleState == apiOnboard.ConnectionLifecycleStateDisabled {
			disabledCount++
		}
	}

	hits, err := es.FetchConnectionResourcesCountAtTime(h.client, connectors, connectionIDs, endTime, []map[string]any{{"described_at": "asc"}}, EsFetchPageSize)
	for _, hit := range hits {
		if v, ok := res[hit.SourceID]; ok {
			v.ResourceCount += hit.ResourceCount
			if v.LastInventory.IsZero() || v.LastInventory.Before(time.UnixMilli(hit.DescribedAt)) {
				v.LastInventory = time.UnixMilli(hit.DescribedAt)
			}
			res[hit.SourceID] = v
		}
	}

	costs, err := es.FetchDailyCostHistoryByAccountsBetween(h.client, connectors, connectionIDs, endTime, startTime, EsFetchPageSize)
	aggregatedCostHits := internal.AggregateConnectionCosts(costs)
	if err != nil {
		return err
	}
	for sourceID, costArr := range aggregatedCostHits {
		if v, ok := res[sourceID]; ok {
			if v.Cost == nil {
				v.Cost = make(map[string]float64)
			}
			for _, cost := range costArr {
				val, _ := v.Cost[cost.Unit]
				val += cost.Cost
				v.Cost[cost.Unit] = val
			}
			res[sourceID] = v
		}
	}

	totalCost := make(map[string]float64)
	var connectionSummaries []api.Connection
	for _, v := range res {
		connectionSummaries = append(connectionSummaries, v)
		if v.Cost != nil {
			for k, v := range v.Cost {
				if _, ok := totalCost[k]; !ok {
					totalCost[k] = 0
				}
				totalCost[k] += v
			}
		}
	}

	switch sortBy {
	case "onboard_date":
		sort.Slice(connectionSummaries, func(i, j int) bool {
			return connectionSummaries[i].OnboardDate.Before(connectionSummaries[j].OnboardDate)
		})
	case "resource_count":
		sort.Slice(connectionSummaries, func(i, j int) bool {
			if connectionSummaries[i].ResourceCount == connectionSummaries[j].ResourceCount {
				return connectionSummaries[i].ConnectionID < connectionSummaries[j].ConnectionID
			}
			return connectionSummaries[i].ResourceCount > connectionSummaries[j].ResourceCount
		})
	case "cost":
		sort.Slice(connectionSummaries, func(i, j int) bool {
			if connectionSummaries[i].Cost[DefaultCurrency] == connectionSummaries[j].Cost[DefaultCurrency] {
				return connectionSummaries[i].ConnectionID < connectionSummaries[j].ConnectionID
			}
			return connectionSummaries[i].Cost[DefaultCurrency] > connectionSummaries[j].Cost[DefaultCurrency]
		})
	default:
		sort.Slice(connectionSummaries, func(i, j int) bool {
			return connectionSummaries[i].ConnectionID < connectionSummaries[j].ConnectionID
		})
	}

	response := api.ListConnectionsResponse{
		TotalCount:          len(connectionSummaries),
		TotalUnhealthyCount: unhealthyCount,
		TotalDisabledCount:  disabledCount,
		TotalCost:           totalCost,
		Connections:         utils.Paginate(pageNumber, pageSize, connectionSummaries),
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetConnectionSummary godoc
//
//	@Summary	Returns resource count of accounts
//	@Tags		benchmarks
//	@Accept		json
//	@Produce	json
//	@Param		startTime	query		int	false	"start time in unix seconds"
//	@Param		endTime		query		int	false	"end time in unix seconds"
//	@Success	200			{object}	api.Connection
//	@Router		/inventory/api/v2/connections/summary/{connectionId} [get]
func (h *HttpHandler) GetConnectionSummary(ctx echo.Context) error {
	var err error
	connectionId := ctx.Param("connectionId")
	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now()
	if endTimeStr != "" {
		endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid integer")
		}
		endTime = time.Unix(endTimeUnix, 0)
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr != "" {
		startTimeUnix, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "startTime is not a valid integer")
		}
		startTime = time.Unix(startTimeUnix, 0)
	}

	res := api.Connection{}
	src, err := h.onboardClient.GetSource(httpclient.FromEchoContext(ctx), connectionId)
	if err != nil {
		return err
	}

	res = api.Connection{
		ConnectionID:            src.ID.String(),
		Connector:               src.Type,
		ConnectorConnectionName: src.ConnectionName,
		ConnectorConnectionID:   src.ConnectionID,
		LifecycleState:          string(src.LifecycleState),
		OnboardDate:             src.OnboardDate,
		HealthState:             src.HealthState,
		LastHealthCheckTime:     src.LastHealthCheckTime,
		HealthReason:            src.HealthReason,
		Metadata:                src.Metadata,
	}

	hits, err := es.FetchConnectionResourcesCountAtTime(h.client, []source.Type{src.Type}, []string{connectionId}, endTime, []map[string]any{{"described_at": "asc"}}, EsFetchPageSize)
	for _, hit := range hits {
		if hit.SourceID == connectionId {
			res.ResourceCount += hit.ResourceCount
			if res.LastInventory.IsZero() || res.LastInventory.Before(time.UnixMilli(hit.DescribedAt)) {
				res.LastInventory = time.UnixMilli(hit.DescribedAt)
			}
		}
	}

	costs, err := es.FetchDailyCostHistoryByAccountsBetween(h.client, []source.Type{src.Type}, []string{connectionId}, endTime, startTime, EsFetchPageSize)
	aggregatedCostHits := internal.AggregateConnectionCosts(costs)
	if err != nil {
		return err
	}
	for sourceID, costArr := range aggregatedCostHits {
		if sourceID == connectionId {
			if res.Cost == nil {
				res.Cost = make(map[string]float64)
			}
			for _, cost := range costArr {
				val, _ := res.Cost[cost.Unit]
				val += cost.Cost
				res.Cost[cost.Unit] = val
			}
		}
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetResourceDistribution godoc
//
//	@Summary	Returns distribution of resource for specific account
//	@Security	BearerToken
//	@Tags		benchmarks
//	@Accept		json
//	@Produce	json
//	@Param		sourceId	query		[]string	true	"SourceID"
//	@Param		provider	query		string		true	"Provider"		Enums(AWS,Azure,all)
//	@Param		timeWindow	query		string		true	"Time Window"	Enums(24h,1w,3m,1y,max)
//	@Success	200			{object}	map[string]int
//	@Router		/inventory/api/v1/resources/distribution [get]
func (h *HttpHandler) GetResourceDistribution(ctx echo.Context) error {
	provider, _ := source.ParseType(ctx.QueryParam("provider"))
	sourceIDs := ctx.QueryParams()["sourceId"]

	if len(sourceIDs) != 0 {
		sourceIDs = nil
	}
	locationDistribution := map[string]int{}

	hits, err := es.FetchConnectionLocationsSummaryPage(h.client, provider, sourceIDs, nil, EsFetchPageSize)
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
//
//	@Summary	Returns distribution of services for specific account
//	@Security	BearerToken
//	@Tags		benchmarks
//	@Accept		json
//	@Produce	json
//	@Param		sourceId	query		[]string	true	"SourceID"
//	@Param		provider	query		string		true	"Provider"
//	@Success	200			{object}	[]api.ServiceDistributionItem
//	@Router		/inventory/api/v1/services/distribution [get]
func (h *HttpHandler) GetServiceDistribution(ctx echo.Context) error {
	sourceIDs := ctx.QueryParams()["sourceId"]
	if len(sourceIDs) == 0 {
		sourceIDs = nil
	}

	hits, err := es.FetchConnectionServiceLocationsSummaryPage(h.client, source.Nil, sourceIDs, nil, EsFetchPageSize)
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

// ListServiceSummaries godoc
//
//	@Summary		Get Cloud Services Summary
//	@Description	Gets a summary of the services including the number of them and the API filters and a list of services with more details. Including connector, the resource counts and the cost.
//	@Security		BearerToken
//	@Tags			benchmarks
//	@Accept			json
//	@Produce		json
//	@Param			connectionId	query		string	false	"filter: Connection ID"
//	@Param			connector		query		string	false	"filter: Connector"
//	@Param			tag				query		string	false	"filter: Category for the services"
//	@Param			startTime		query		string	true	"start time for cost calculation in epoch seconds"
//	@Param			endTime			query		string	true	"end time for cost calculation and time resource count in epoch seconds"
//	@Param			minSpent		query		int		false	"filter: minimum spent amount for the service in the specified time"
//	@Param			pageSize		query		int		false	"page size - default is 20"
//	@Param			pageNumber		query		int		false	"page number - default is 1"
//	@Param			sortBy			query		string	false	"column to sort by - default is cost"	Enums(servicecode,resourcecount,cost)
//	@Success		200				{object}	api.ListServiceSummariesResponse
//	@Router			/inventory/api/v2/services/summary [get]
func (h *HttpHandler) ListServiceSummaries(ctx echo.Context) error {
	var err error
	tagMap := internal.TagStringsToTagMap(ctx.QueryParams()["tag"])

	connectionIDs := ctx.QueryParams()["connectionId"]
	if len(connectionIDs) == 0 {
		connectionIDs = nil
	}
	connectors := source.ParseTypes(ctx.QueryParams()["connector"])

	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid epoch time")
		}
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Unix(endTime, 0).AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "startTime is not a valid epoch time")
		}
	}

	minSpentStr := ctx.QueryParam("minSpent")
	var minSpent *float64
	if minSpentStr != "" {
		minSpentF, err := strconv.ParseFloat(minSpentStr, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "minSpent is not a valid integer")
		}
		minSpent = &minSpentF
	}

	pageNumber, pageSize, err := internal.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	sortBy := ctx.QueryParam("sortBy")
	if sortBy == "" {
		sortBy = "cost"
	}

	costFilterMap := make(map[string]float64)
	resourceTypeMap := make(map[string]int64)
	services, err := h.db.ListFilteredServices(tagMap, connectors)
	if err != nil {
		return err
	}

	for _, service := range services {
		for _, costFilterName := range service.GetTagsMap()[KaytuServiceCostTag] {
			costFilterMap[costFilterName] = 0
		}
		for _, resourceType := range service.ResourceTypes {
			resourceTypeMap[resourceType.ResourceType] = 0
		}
	}
	costFilterNames := make([]string, 0, len(costFilterMap))
	for costFilterName := range costFilterMap {
		costFilterNames = append(costFilterNames, costFilterName)
	}
	resourceTypeNames := make([]string, 0, len(resourceTypeMap))
	for resourceTypeName := range resourceTypeMap {
		resourceTypeNames = append(resourceTypeNames, resourceTypeName)
	}

	costHits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, connectionIDs, connectors, costFilterNames, time.Unix(endTime, 0), time.Unix(startTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}
	costs := internal.AggregateServiceCosts(costHits)

	resourceTypeCounts, err := es.FetchResourceTypeCountAtTime(h.client, connectors, connectionIDs, time.Unix(endTime, 0), resourceTypeNames, EsFetchPageSize)
	if err != nil {
		return err
	}

	serviceSummaries := make([]api.ServiceSummary, 0, len(services))
	for _, service := range services {
		serviceSummary := api.ServiceSummary{
			Connector:     service.Connector,
			ServiceLabel:  service.ServiceLabel,
			ServiceName:   service.ServiceName,
			ResourceCount: nil,
			Cost:          nil,
		}
		for _, costFilterName := range service.GetTagsMap()[KaytuServiceCostTag] {
			if cost, ok := costs[costFilterName]; ok {
				serviceSummary.Cost = utils.PAdd(serviceSummary.Cost, utils.GetPointer(cost[DefaultCurrency].Cost))
			}
		}
		for _, resourceType := range service.ResourceTypes {
			if resourceTypeCount, ok := resourceTypeCounts[resourceType.ResourceType]; ok {
				serviceSummary.ResourceCount = utils.PAdd(serviceSummary.ResourceCount, &resourceTypeCount)
			}
		}
		serviceSummaries = append(serviceSummaries, serviceSummary)
	}

	if minSpent != nil {
		filteredServiceSummaries := make([]api.ServiceSummary, 0, len(serviceSummaries))
		for _, serviceSummary := range serviceSummaries {
			if serviceSummary.Cost != nil && *serviceSummary.Cost >= *minSpent {
				filteredServiceSummaries = append(filteredServiceSummaries, serviceSummary)
			}
		}
		serviceSummaries = filteredServiceSummaries
	}

	switch sortBy {
	case "servicecode":
		sort.Slice(serviceSummaries, func(i, j int) bool {
			return serviceSummaries[i].ServiceName < serviceSummaries[j].ServiceName
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

			return *serviceSummaries[i].Cost > *serviceSummaries[j].Cost
		})
	default:
		sort.Slice(serviceSummaries, func(i, j int) bool {
			return serviceSummaries[i].ServiceName < serviceSummaries[j].ServiceName
		})
	}

	res := api.ListServiceSummariesResponse{
		TotalCount: len(serviceSummaries),
		Services:   utils.Paginate(pageNumber, pageSize, serviceSummaries),
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetServiceSummary godoc
//
//	@Summary		Get Cloud Service Summary
//	@Description	Get Cloud Service Summary for the specified service name. Including connector, the resource counts and the cost.
//	@Security		BearerToken
//	@Tags			benchmarks
//	@Accepts		json
//	@Produce		json
//	@Param			connectorId	query		string	false	"filter: SourceIDs"
//	@Param			connector	query		string	false	"filter: Provider"
//	@Param			startTime	query		string	true	"start time for cost calculation in epoch seconds"
//	@Param			endTime		query		string	true	"end time for cost calculation and time resource count in epoch seconds"
//	@Success		200			{object}	api.ServiceSummary
//	@Router			/inventory/api/v2/services/summary/{serviceName} [get]
func (h *HttpHandler) GetServiceSummary(ctx echo.Context) error {
	var err error
	serviceName := ctx.Param("serviceName")
	if serviceName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "service_name is required")
	}

	connectionIDs := ctx.QueryParams()["connectorId"]
	if len(connectionIDs) == 0 {
		connectionIDs = nil
	}
	connectors := source.ParseTypes(ctx.QueryParams()["connector"])

	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid epoch time")
		}
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := time.Unix(endTime, 0).AddDate(0, 0, -7).Unix()
	if startTimeStr != "" {
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "startTime is not a valid epoch time")
		}
	}

	costFilterMap := make(map[string]float64)
	resourceTypeMap := make(map[string]int64)
	service, err := h.db.GetService(serviceName)
	if err != nil {
		return err
	}

	for _, costFilterName := range service.GetTagsMap()[KaytuServiceCostTag] {
		costFilterMap[costFilterName] = 0
	}
	for _, resourceType := range service.ResourceTypes {
		resourceTypeMap[resourceType.ResourceType] = 0
	}

	costFilterNames := make([]string, 0, len(costFilterMap))
	for costFilterName := range costFilterMap {
		costFilterNames = append(costFilterNames, costFilterName)
	}
	resourceTypeNames := make([]string, 0, len(resourceTypeMap))
	for resourceTypeName := range resourceTypeMap {
		resourceTypeNames = append(resourceTypeNames, resourceTypeName)
	}

	costHits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, connectionIDs, connectors, costFilterNames, time.Unix(endTime, 0), time.Unix(startTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}
	costs := internal.AggregateServiceCosts(costHits)
	resourceTypeCounts, err := es.FetchResourceTypeCountAtTime(h.client, connectors, connectionIDs, time.Unix(endTime, 0), resourceTypeNames, EsFetchPageSize)
	if err != nil {
		return err
	}

	serviceSummary := api.ServiceSummary{
		Connector:     service.Connector,
		ServiceLabel:  service.ServiceLabel,
		ServiceName:   service.ServiceName,
		ResourceCount: nil,
		Cost:          nil,
	}
	for _, costFilterName := range service.GetTagsMap()[KaytuServiceCostTag] {
		if cost, ok := costs[costFilterName]; ok {
			serviceSummary.Cost = utils.PAdd(serviceSummary.Cost, utils.GetPointer(cost[DefaultCurrency].Cost))
		}
	}
	for _, resourceType := range service.ResourceTypes {
		if resourceTypeCount, ok := resourceTypeCounts[resourceType.ResourceType]; ok {
			serviceSummary.ResourceCount = utils.PAdd(serviceSummary.ResourceCount, &resourceTypeCount)
		}
	}

	return ctx.JSON(http.StatusOK, serviceSummary)
}

// GetResource godoc
//
//	@Summary		Get details of a Resource
//	@Description	Getting resource details by id and resource type
//	@Security		BearerToken
//	@Tags			resource
//	@Accepts		json
//	@Produce		json
//	@Param			request	body		api.GetResourceRequest	true	"Request Body"
//	@Success		200		{object}	map[string]string
//	@Router			/inventory/api/v1/resource [post]
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
	if pluginProvider == steampipe.SteampipePluginAWS {
		pluginTableName := awsSteampipe.ExtractTableName(req.ResourceType)
		desc, err := steampipe.ConvertToDescription(req.ResourceType, source, awsSteampipe.AWSDescriptionMap)
		if err != nil {
			return err
		}

		cells, err = awsSteampipe.AWSDescriptionToRecord(desc, pluginTableName)
		if err != nil {
			return err
		}
	} else if pluginProvider == steampipe.SteampipePluginAzure || pluginProvider == steampipe.SteampipePluginAzureAD {
		pluginTableName := azureSteampipe.ExtractTableName(req.ResourceType)
		desc, err := steampipe.ConvertToDescription(req.ResourceType, source, azureSteampipe.AzureDescriptionMap)
		if err != nil {
			return err
		}

		if pluginProvider == steampipe.SteampipePluginAzure {
			cells, err = azureSteampipe.AzureDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				return err
			}
		} else {
			cells, err = azureSteampipe.AzureADDescriptionToRecord(desc, pluginTableName)
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
		} else if _, ok := v.GetValue().(*proto.Column_NullValue); ok {
			val = ""
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
//
//	@Summary		List smart queries
//	@Description	Listing smart queries by specified filters
//	@Security		BearerToken
//	@Tags			smart_query
//	@Produce		json
//	@Param			request	body		api.ListQueryRequest	true	"Request Body"
//	@Success		200		{object}	[]api.SmartQueryItem
//	@Router			/inventory/api/v1/query [get]
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
		category := ""

		result = append(result, api.SmartQueryItem{
			ID:          item.Model.ID,
			Provider:    item.Provider,
			Title:       item.Title,
			Category:    category,
			Description: item.Description,
			Query:       item.Query,
			Tags:        nil,
		})
	}
	return ctx.JSON(200, result)
}

// CountQueries godoc
//
//	@Summary		Count smart queries
//	@Description	Counting smart queries
//	@Security		BearerToken
//	@Tags			smart_query
//	@Produce		json
//	@Param			request	body		api.ListQueryRequest	true	"Request Body"
//	@Success		200		{object}	int
//	@Router			/inventory/api/v1/query/count [get]
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
//
//	@Summary		Run a specific smart query
//	@Description	Run a specific smart query.
//	@Description	In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
//	@Description	Note that csv output doesn't process pagination and returns first 5000 records.
//	@Security		BearerToken
//	@Tags			smart_query
//	@Accepts		json
//	@Produce		json,text/csv
//	@Param			queryId	path		string				true	"QueryID"
//	@Param			request	body		api.RunQueryRequest	true	"Request Body"
//	@Param			accept	header		string				true	"Accept header"	Enums(application/json,text/csv)
//	@Success		200		{object}	api.RunQueryResponse
//	@Router			/inventory/api/v1/query/{queryId} [post]
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
				if err == gorm.ErrRecordNotFound {
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
		if err == gorm.ErrRecordNotFound {
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
//
//	@Summary		Get locations
//	@Description	Getting locations by provider
//	@Security		BearerToken
//	@Tags			location
//	@Produce		json
//	@Param			provider	path		string	true	"Provider"	Enums(aws,azure,all)
//	@Success		200			{object}	[]api.LocationByProviderResponse
//	@Router			/inventory/api/v1/locations/{provider} [get]
func (h *HttpHandler) GetLocations(ctx echo.Context) error {
	provider := ctx.Param("provider")

	var locations []api.LocationByProviderResponse

	if provider == "aws" || provider == "all" {
		regions, err := h.awsClient.NewEC2RegionPaginator(nil, nil)
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
		locs, err := h.azureClient.NewLocationPaginator(nil, nil)
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
//
//	@Summary		Get Azure resources
//	@Description	Getting Azure resources by filters.
//	@Description	In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
//	@Description	Note that csv output doesn't process pagination and returns first 5000 records.
//	@Description	If sort by is empty, result will be sorted by the first column in ascending order.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json,text/csv
//	@Param			request	body		api.GetResourcesRequest	true	"Request Body"
//	@Param			accept	header		string					true	"Accept header"	Enums(application/json,text/csv)
//	@Param			common	query		string					false	"Common filter"	Enums(true,false,all)
//	@Success		200		{object}	api.GetAzureResourceResponse
//	@Router			/inventory/api/v1/resources/azure [post]
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
//
//	@Summary		Get AWS resources
//	@Description	Getting AWS resources by filters.
//	@Description	In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
//	@Description	Note that csv output doesn't process pagination and returns first 5000 records.
//	@Description	If sort by is empty, result will be sorted by the first column in ascending order.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json,text/csv
//	@Param			request	body		api.GetResourcesRequest	true	"Request Body"
//	@Param			accept	header		string					true	"Accept header"	Enums(application/json,text/csv)
//	@Param			common	query		string					false	"Common filter"	Enums(true,false,all)
//	@Success		200		{object}	api.GetAWSResourceResponse
//	@Router			/inventory/api/v1/resources/aws [post]
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
//
//	@Summary		Get resources
//	@Description	Getting all cloud providers resources by filters.
//	@Description	In order to get the results in CSV format, Accepts header must be filled with `text/csv` value.
//	@Description	Note that csv output doesn't process pagination and returns first 5000 records.
//	@Description	If sort by is empty, result will be sorted by the first column in ascending order.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json,text/csv
//	@Param			request	body		api.GetResourcesRequest	true	"Request Body"
//	@Param			accept	header		string					true	"Accept header"	Enums(application/json,text/csv)
//	@Param			common	query		string					false	"Common filter"	Enums(true,false,all)
//	@Success		200		{object}	api.GetResourcesResponse
//	@Router			/inventory/api/v1/resources [post]
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
//
//	@Summary		Count resources
//	@Description	Number of all resources
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json,text/csv
//	@Success		200	{object}	int64
//	@Router			/inventory/api/v1/resources/count [get]
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
//
//	@Summary		Get resource filters
//	@Description	Getting resource filters by filters.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json,text/csv
//	@Param			request	body		api.GetFiltersRequest	true	"Request Body"
//	@Param			common	query		string					false	"Common filter"	Enums(true,false,all)
//	@Success		200		{object}	api.GetFiltersResponse
//	@Router			/inventory/api/v1/resources/filters [post]
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
	for _, item := range response.Aggregations.ResourceTypeFilter.Buckets {
		resp.Filters.ResourceType = append(resp.Filters.ResourceType, api.ResourceTypeFull{
			ResourceTypeARN:  item.Key,
			ResourceTypeName: cloudservice.ResourceTypeName(item.Key),
		})
	}

	services, err := h.graphDb.GetCloudServiceNodes(ctx.Request().Context(), source.Nil)
	if err != nil {
		return err
	}
	resp.Filters.Service = make(map[string]string)
	for _, service := range services {
		resp.Filters.Service[service.ServiceName] = service.Name
	}
	if !api.FilterIsEmpty(req.Filters.Service) {
		servicesMap := make(map[string]string)
		for _, service := range req.Filters.Service {
			if _, ok := resp.Filters.Service[service]; ok {
				servicesMap[service] = resp.Filters.Service[service]
			}
		}
		resp.Filters.Service = servicesMap
	}

	categories, err := h.graphDb.GetNormalCategoryNodes(ctx.Request().Context(), source.Nil)
	if err != nil {
		return err
	}
	resp.Filters.Category = make(map[string]string)
	for _, category := range categories {
		resp.Filters.Category[category.ElementID] = category.Name
	}
	if !api.FilterIsEmpty(req.Filters.Category) {
		categoriesMap := make(map[string]string)
		for _, category := range req.Filters.Category {
			if _, ok := resp.Filters.Category[category]; ok {
				categoriesMap[category] = resp.Filters.Category[category]
			}
		}
		resp.Filters.Category = categoriesMap
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

	if !api.FilterIsEmpty(req.Filters.Service) && api.FilterIsEmpty(req.Filters.ResourceType) {
		pvd := source.Nil
		if provider != nil {
			pvd, _ = source.ParseType(string(*provider))
		}
		filterType := FilterTypeCloudResourceType
		resourceFilters, err := h.graphDb.GetFilters(ctx.Request().Context(), pvd, req.Filters.Service, &filterType)
		if err != nil {
			return err
		}
		req.Filters.ResourceType = make([]string, 0)
		for _, filter := range resourceFilters {
			switch filter.GetFilterType() {
			case FilterTypeCloudResourceType:
				f := filter.(*FilterCloudResourceTypeNode)
				req.Filters.ResourceType = append(req.Filters.ResourceType, f.ResourceType)
			}
		}
	}

	if !api.FilterIsEmpty(req.Filters.Category) && api.FilterIsEmpty(req.Filters.ResourceType) {
		resourceTypesMap := make(map[string]bool)
		for _, category := range req.Filters.Category {
			cat, err := h.graphDb.GetCategory(ctx.Request().Context(), category)
			if err != nil {
				return err
			}
			for _, filter := range cat.SubTreeFilters {
				switch filter.GetFilterType() {
				case FilterTypeCloudResourceType:
					f := filter.(*FilterCloudResourceTypeNode)
					resourceTypesMap[f.ResourceType] = true
				}
			}
		}
		req.Filters.ResourceType = make([]string, 0)
		for resourceType := range resourceTypesMap {
			req.Filters.ResourceType = append(req.Filters.ResourceType, resourceType)
		}
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

// ListInsights godoc
//
//	@Summary		List insights
//	@Description	List all insights
//	@Security		BearerToken
//	@Tags			insight
//	@Produce		json
//	@Param			connector	query		source.Type	false	"filter insights by connector"
//	@Param			sourceId	query		[]string	false	"filter the result by source id"
//	@Param			time		query		int			false	"unix seconds for the time to get the insight result for"
//	@Success		200			{object}	[]api.ListInsightResult
//	@Router			/inventory/api/v2/insights [get]
func (h *HttpHandler) ListInsights(ctx echo.Context) error {
	connector, _ := source.ParseType(ctx.QueryParam("connector"))
	var resultTime *time.Time
	if timeStr := ctx.QueryParam("time"); timeStr != "" {
		timeInt, err := strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		t := time.Unix(timeInt, 0)
		resultTime = &t
	}
	sourceIDs := ctx.QueryParams()["sourceId"]
	if len(sourceIDs) == 0 {
		sourceIDs = nil
	}

	insightList, err := h.complianceClient.GetInsights(httpclient.FromEchoContext(ctx), connector)
	if err != nil {
		return err
	}

	insightPeerGroupList, err := h.complianceClient.GetInsightPeerGroups(httpclient.FromEchoContext(ctx), connector)
	if err != nil {
		return err
	}

	insightIdList := make([]uint, 0, len(insightList))
	insightResultMap := make(map[uint]*api.Insight)
	for _, insightRow := range insightList {
		insightIdList = append(insightIdList, insightRow.ID)
		tags := make([]api.InsightTag, 0, len(insightRow.Tags))
		for _, tag := range insightRow.Tags {
			tags = append(tags, api.InsightTag{
				ID:    tag.ID,
				Key:   tag.Key,
				Value: tag.Value,
			})
		}
		links := make([]api.InsightLink, 0, len(insightRow.Links))
		for _, link := range insightRow.Links {
			links = append(links, api.InsightLink{
				ID:   link.ID,
				Text: link.Text,
				URI:  link.URI,
			})
		}
		insightResultMap[insightRow.ID] = &api.Insight{
			ID: insightRow.ID,
			Query: api.Query{
				ID:             insightRow.Query.ID,
				QueryToExecute: insightRow.Query.QueryToExecute,
				Connector:      insightRow.Query.Connector,
				ListOfTables:   insightRow.Query.ListOfTables,
				Engine:         insightRow.Query.Engine,
				CreatedAt:      insightRow.Query.CreatedAt,
				UpdatedAt:      insightRow.Query.UpdatedAt,
			},
			Category:              insightRow.Category,
			Provider:              insightRow.Connector,
			ShortTitle:            insightRow.ShortTitle,
			LongTitle:             insightRow.LongTitle,
			Description:           insightRow.Description,
			LogoURL:               insightRow.LogoURL,
			Labels:                tags,
			Links:                 links,
			Enabled:               insightRow.Enabled,
			TotalResults:          0,
			ListInsightResultType: api.ListInsightResultTypeInsight,
		}
	}

	var insightValues map[uint]insight.InsightResource
	if resultTime != nil {
		insightValues, err = es.FetchInsightValueAtTime(h.client, *resultTime, connector, sourceIDs, insightIdList, true)
	} else {
		insightValues, err = es.FetchInsightValueAtTime(h.client, time.Now(), connector, sourceIDs, insightIdList, false)
	}
	if err != nil {
		return err
	}

	for insightId, insightResult := range insightValues {
		if v, ok := insightResultMap[insightId]; ok {
			v.TotalResults += insightResult.Result
			if insightResult.ExecutedAt != 0 {
				exAt := time.UnixMilli(insightResult.ExecutedAt)
				v.ExecutedAt = &exAt
			}
		}
	}

	result := make([]api.ListInsightResult, 0)
	usedInPeerGroup := make(map[uint]bool)
	for _, insightPeerGroup := range insightPeerGroupList {
		tags := make([]api.InsightTag, 0, len(insightPeerGroup.Tags))
		for _, tag := range insightPeerGroup.Tags {
			tags = append(tags, api.InsightTag{
				ID:    tag.ID,
				Key:   tag.Key,
				Value: tag.Value,
			})
		}
		links := make([]api.InsightLink, 0, len(insightPeerGroup.Links))
		for _, link := range insightPeerGroup.Links {
			links = append(links, api.InsightLink{
				ID:   link.ID,
				Text: link.Text,
				URI:  link.URI,
			})
		}
		peerGroup := &api.InsightPeerGroup{
			ID:                    insightPeerGroup.ID,
			Category:              insightPeerGroup.Category,
			Insights:              make([]api.Insight, 0, len(insightPeerGroup.Insights)),
			ShortTitle:            insightPeerGroup.ShortTitle,
			LongTitle:             insightPeerGroup.LongTitle,
			Description:           insightPeerGroup.Description,
			LogoURL:               insightPeerGroup.LogoURL,
			Labels:                tags,
			Links:                 links,
			TotalResults:          0,
			ListInsightResultType: api.ListInsightResultTypePeerGroup,
		}
		for _, apiInsight := range insightPeerGroup.Insights {
			if v, ok := insightResultMap[apiInsight.ID]; ok {
				peerGroup.Insights = append(peerGroup.Insights, *v)
				peerGroup.TotalResults += v.TotalResults
				usedInPeerGroup[apiInsight.ID] = true
			}
		}
		result = append(result, peerGroup)
	}

	for _, v := range insightResultMap {
		if _, ok := usedInPeerGroup[v.ID]; ok {
			continue
		}
		result = append(result, v)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].GetType() == result[j].GetType() {
			return result[i].GetID() < result[j].GetID()
		} else if result[i].GetType() == api.ListInsightResultTypePeerGroup {
			return true
		} else if result[j].GetType() == api.ListInsightResultTypePeerGroup {
			return false
		} else {
			return result[i].GetID() < result[j].GetID()
		}
	})

	return ctx.JSON(http.StatusOK, result)
}

// GetInsight godoc
//
//	@Summary		Get insight
//	@Description	Get an insight by id
//	@Security		BearerToken
//	@Tags			insight
//	@Produce		json
//	@Param			sourceId	query		[]string	false	"filter the result by source id"
//	@Param			time		query		int			false	"unix seconds for the time to get the insight result for"
//	@Success		200			{object}	api.Insight
//	@Router			/inventory/api/v2/insights/{insightId} [get]
func (h *HttpHandler) GetInsight(ctx echo.Context) error {
	insightId, err := strconv.ParseUint(ctx.Param("insightId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid insight id")
	}
	var resultTime *time.Time
	if timeStr := ctx.QueryParam("time"); timeStr != "" {
		timeInt, err := strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		t := time.Unix(timeInt, 0)
		resultTime = &t
	}
	sourceIDs := ctx.QueryParams()["sourceId"]
	if len(sourceIDs) == 0 {
		sourceIDs = nil
	}

	insightRow, err := h.complianceClient.GetInsightById(httpclient.FromEchoContext(ctx), uint(insightId))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return echo.NewHTTPError(http.StatusNotFound, "insight not found")
		}
		return err
	}

	tags := make([]api.InsightTag, 0, len(insightRow.Tags))
	for _, tag := range insightRow.Tags {
		tags = append(tags, api.InsightTag{
			ID:    tag.ID,
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	links := make([]api.InsightLink, 0, len(insightRow.Links))
	for _, link := range insightRow.Links {
		links = append(links, api.InsightLink{
			ID:   link.ID,
			Text: link.Text,
			URI:  link.URI,
		})
	}
	result := api.Insight{
		ID: insightRow.ID,
		Query: api.Query{
			ID:             insightRow.Query.ID,
			QueryToExecute: insightRow.Query.QueryToExecute,
			Connector:      insightRow.Query.Connector,
			ListOfTables:   insightRow.Query.ListOfTables,
			Engine:         insightRow.Query.Engine,
			CreatedAt:      insightRow.Query.CreatedAt,
			UpdatedAt:      insightRow.Query.UpdatedAt,
		},
		Category:              insightRow.Category,
		Provider:              insightRow.Connector,
		ShortTitle:            insightRow.ShortTitle,
		LongTitle:             insightRow.LongTitle,
		Description:           insightRow.Description,
		LogoURL:               insightRow.LogoURL,
		Labels:                tags,
		Links:                 links,
		Enabled:               insightRow.Enabled,
		TotalResults:          0,
		Results:               nil,
		ListInsightResultType: api.ListInsightResultTypeInsight,
	}

	var insightResults map[uint]insight.InsightResource
	if resultTime != nil {
		insightResults, err = es.FetchInsightValueAtTime(h.client, *resultTime, source.Nil, sourceIDs, []uint{uint(insightId)}, true)
	} else {
		insightResults, err = es.FetchInsightValueAtTime(h.client, time.Now(), source.Nil, sourceIDs, []uint{uint(insightId)}, false)
	}
	if err != nil {
		return err
	}

	if insightResult, ok := insightResults[uint(insightId)]; ok {
		result.TotalResults = insightResult.Result
		exAt := time.UnixMilli(insightResult.ExecutedAt)
		result.ExecutedAt = &exAt

		bucket, key, err := utils.ParseHTTPSubpathS3URIToBucketAndKey(insightResult.S3Location)
		objectBuffer := aws.NewWriteAtBuffer(make([]byte, 0, 1024*1024))
		_, err = h.s3Downloader.Download(objectBuffer, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		var results steampipe.Result
		err = json.Unmarshal(objectBuffer.Bytes(), &results)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		connections := make([]api.InsightConnection, 0, len(insightResult.IncludedConnections))
		for _, connection := range insightResult.IncludedConnections {
			connections = append(connections, api.InsightConnection{
				ConnectionID: connection.ConnectionID,
				OriginalID:   connection.OriginalID,
			})
		}

		result.Results = &api.InsightResult{
			JobID:       insightResult.JobID,
			InsightID:   insightResult.QueryID,
			SourceID:    insightResult.SourceID,
			ExecutedAt:  time.UnixMilli(insightResult.ExecutedAt),
			Locations:   insightResult.Locations,
			Connections: connections,
			Result:      insightResult.Result,
			Details: &api.InsightDetail{
				Headers: results.Headers,
				Rows:    results.Data,
			},
		}
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetInsightPeerGroup godoc
//
//	@Summary		Get insight peer
//	@Description	Get an insight peer by id
//	@Security		BearerToken
//	@Tags			insight
//	@Produce		json
//	@Param			sourceId	query		[]string	false	"filter the result by source id"
//	@Param			time		query		int			false	"unix seconds for the time to get the insight result for"
//	@Success		200			{object}	api.InsightPeerGroup
//	@Router			/inventory/api/v2/insights/peer/{insightPeerGroupId} [get]
func (h *HttpHandler) GetInsightPeerGroup(ctx echo.Context) error {
	insightPeerGroupId, err := strconv.ParseUint(ctx.Param("insightPeerGroupId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid insight peer group id")
	}
	var resultTime *time.Time
	if timeStr := ctx.QueryParam("time"); timeStr != "" {
		timeInt, err := strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		t := time.Unix(timeInt, 0)
		resultTime = &t
	}
	sourceIDs := ctx.QueryParams()["sourceId"]
	if len(sourceIDs) == 0 {
		sourceIDs = nil
	}

	insightPeerGroup, err := h.complianceClient.GetInsightPeerGroupById(httpclient.FromEchoContext(ctx), uint(insightPeerGroupId))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return echo.NewHTTPError(http.StatusNotFound, "insight peer group not found")
		}
		return err
	}

	tags := make([]api.InsightTag, 0, len(insightPeerGroup.Tags))
	for _, tag := range insightPeerGroup.Tags {
		tags = append(tags, api.InsightTag{
			ID:    tag.ID,
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	links := make([]api.InsightLink, 0, len(insightPeerGroup.Links))
	for _, link := range insightPeerGroup.Links {
		links = append(links, api.InsightLink{
			ID:   link.ID,
			Text: link.Text,
			URI:  link.URI,
		})
	}
	insights := make([]api.Insight, 0, len(insightPeerGroup.Insights))
	insightIds := make([]uint, 0, len(insightPeerGroup.Insights))
	for _, insightRow := range insightPeerGroup.Insights {
		tags := make([]api.InsightTag, 0, len(insightRow.Tags))
		for _, tag := range insightRow.Tags {
			tags = append(tags, api.InsightTag{
				ID:    tag.ID,
				Key:   tag.Key,
				Value: tag.Value,
			})
		}
		links := make([]api.InsightLink, 0, len(insightRow.Links))
		for _, link := range insightRow.Links {
			links = append(links, api.InsightLink{
				ID:   link.ID,
				Text: link.Text,
				URI:  link.URI,
			})
		}
		insightIds = append(insightIds, insightRow.ID)
		insights = append(insights, api.Insight{
			ID: insightRow.ID,
			Query: api.Query{
				ID:             insightRow.Query.ID,
				QueryToExecute: insightRow.Query.QueryToExecute,
				Connector:      insightRow.Query.Connector,
				ListOfTables:   insightRow.Query.ListOfTables,
				Engine:         insightRow.Query.Engine,
				CreatedAt:      insightRow.Query.CreatedAt,
				UpdatedAt:      insightRow.Query.UpdatedAt,
			},
			Category:              insightRow.Category,
			Provider:              insightRow.Connector,
			ShortTitle:            insightRow.ShortTitle,
			LongTitle:             insightRow.LongTitle,
			Description:           insightRow.Description,
			LogoURL:               insightRow.LogoURL,
			Labels:                tags,
			Links:                 links,
			Enabled:               insightRow.Enabled,
			ExecutedAt:            nil,
			TotalResults:          0,
			Results:               nil,
			ListInsightResultType: api.ListInsightResultTypeInsight,
		})
	}

	result := api.InsightPeerGroup{
		ID:                    insightPeerGroup.ID,
		Category:              insightPeerGroup.Category,
		Insights:              nil,
		ShortTitle:            insightPeerGroup.ShortTitle,
		LongTitle:             insightPeerGroup.LongTitle,
		Description:           insightPeerGroup.Description,
		LogoURL:               insightPeerGroup.LogoURL,
		Labels:                tags,
		Links:                 links,
		TotalResults:          0,
		ListInsightResultType: api.ListInsightResultTypePeerGroup,
	}

	var insightResults map[uint]insight.InsightResource
	if resultTime != nil {
		insightResults, err = es.FetchInsightValueAtTime(h.client, *resultTime, source.Nil, sourceIDs, insightIds, true)
	} else {
		insightResults, err = es.FetchInsightValueAtTime(h.client, time.Now(), source.Nil, sourceIDs, insightIds, false)
	}
	if err != nil {
		return err
	}

	for i, insightRow := range insights {
		if insightResult, ok := insightResults[insightRow.ID]; ok {
			result.TotalResults = insightResult.Result
			exAt := time.UnixMilli(insightResult.ExecutedAt)
			insights[i].ExecutedAt = &exAt

			bucket, key, err := utils.ParseHTTPSubpathS3URIToBucketAndKey(insightResult.S3Location)
			objectBuffer := aws.NewWriteAtBuffer(make([]byte, 0, 1024*1024))
			_, err = h.s3Downloader.Download(objectBuffer, &s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}

			var results steampipe.Result
			err = json.Unmarshal(objectBuffer.Bytes(), &results)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}

			connections := make([]api.InsightConnection, 0, len(insightResult.IncludedConnections))
			for _, connection := range insightResult.IncludedConnections {
				connections = append(connections, api.InsightConnection{
					ConnectionID: connection.ConnectionID,
					OriginalID:   connection.OriginalID,
				})
			}

			insights[i].Results = &api.InsightResult{
				JobID:       insightResult.JobID,
				InsightID:   insightResult.QueryID,
				SourceID:    insightResult.SourceID,
				ExecutedAt:  time.UnixMilli(insightResult.ExecutedAt),
				Locations:   insightResult.Locations,
				Connections: connections,
				Result:      insightResult.Result,
				Details: &api.InsightDetail{
					Headers: results.Headers,
					Rows:    results.Data,
				},
			}
			result.TotalResults += insightResult.Result
		}
	}
	result.Insights = insights

	return ctx.JSON(http.StatusOK, result)
}

// GetInsightTrend godoc
//
//	@Summary		Get insight trend
//	@Description	Get an insight trend by id
//	@Security		BearerToken
//	@Tags			insight
//	@Produce		json
//	@Param			sourceId		query		string	false	"filter the result by source id"
//	@Param			startTime		query		int		false	"unix seconds for the start of the time window to get the insight trend for"
//	@Param			endTime			query		int		false	"unix seconds for the end of the time window to get the insight trend for"
//	@Param			dataPointCount	query		int		false	"Number of data points to return"
//	@Success		200				{object}	api.InsightResultTrendResponse
//	@Router			/inventory/api/v2/insights/{insightId}/trend [get]
func (h *HttpHandler) GetInsightTrend(ctx echo.Context) error {
	insightId, err := strconv.ParseUint(ctx.Param("insightId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid insight id")
	}
	var startTime, endTime time.Time
	endTime = time.Now()
	if timeStr := ctx.QueryParam("endTime"); timeStr != "" {
		timeInt, err := strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		endTime = time.Unix(timeInt, 0)
	}
	if timeStr := ctx.QueryParam("startTime"); timeStr != "" {
		timeInt, err := strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		startTime = time.Unix(timeInt, 0)
	} else {
		startTime = endTime.Add(-time.Hour * 24 * 30)
	}
	// default to distance between start and end time in days or 30, whichever is smaller
	dataPointCount := int(math.Min(math.Ceil(endTime.Sub(startTime).Hours()/24), 30))
	if dataPointCountStr := ctx.QueryParam("dataPointCount"); dataPointCountStr != "" {
		dataPointCountInt, err := strconv.ParseInt(dataPointCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid dataPointCount")
		}
		dataPointCount = int(dataPointCountInt)
	}

	sourceIDs := ctx.QueryParams()["sourceId"]
	if len(sourceIDs) == 0 {
		sourceIDs = nil
	}

	_, err = h.complianceClient.GetInsightById(httpclient.FromEchoContext(ctx), uint(insightId))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return echo.NewHTTPError(http.StatusNotFound, "insight not found")
		}
		return err
	}

	insightResults, err := es.FetchInsightAggregatedPerQueryValuesBetweenTimes(h.client, startTime, endTime, source.Nil, sourceIDs, []uint{uint(insightId)})
	if err != nil {
		return err
	}

	result := api.InsightResultTrendResponse{
		Trend: make([]api.TrendDataPoint, 0),
	}

	if values, ok := insightResults[uint(insightId)]; ok {
		for _, value := range values {
			result.Trend = append(result.Trend, api.TrendDataPoint{
				Timestamp: value.ExecutedAt / 1000, /* convert to seconds */
				Value:     value.Result,
			})
		}
	}

	result.Trend = internal.DownSampleTrendDataPoints(result.Trend, dataPointCount)
	sort.SliceStable(result.Trend, func(i, j int) bool {
		return result.Trend[i].Timestamp < result.Trend[j].Timestamp
	})

	return ctx.JSON(http.StatusOK, result)
}

// ListConnectorMetadata godoc
//
//	@Summary		Get List of Connectors
//	@Description	Gets a list of all connectors in workspace and their metadata including list of their resource types and services names.
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Success		200	{object}	[]api.ConnectorMetadata
//	@Router			/inventory/api/v2/metadata/connectors [get]
func (h *HttpHandler) ListConnectorMetadata(ctx echo.Context) error {
	var result []api.ConnectorMetadata

	for _, connector := range source.List {
		rootNode, err := h.graphDb.GetCategoryRootByName(ctx.Request().Context(), RootTypeConnectorRoot, connector.String())
		if err != nil {
			return err
		}
		resourceTypesCount := 0
		for _, filter := range rootNode.SubTreeFilters {
			if filter.GetFilterType() == FilterTypeCloudResourceType {
				resourceTypesCount++
			}
		}

		serviceNodes, err := h.graphDb.GetCloudServiceNodes(ctx.Request().Context(), connector)
		if err != nil {
			return err
		}

		result = append(result, api.ConnectorMetadata{
			Connector:          connector,
			ConnectorLabel:     connector.String(),
			ResourceTypesCount: utils.GetPointer(resourceTypesCount),
			ServicesCount:      utils.GetPointer(len(serviceNodes)),
			LogoURI:            rootNode.LogoURI,
		})
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetConnectorMetadata godoc
//
//	@Summary		Get Connector
//	@Description	Gets a single connector and its metadata including list of their resource types and services names by the connector name.
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			connector	path		string	true	"connector"
//	@Success		200			{object}	api.ConnectorMetadata
//	@Router			/inventory/api/v2/metadata/connectors/{connector} [get]
func (h *HttpHandler) GetConnectorMetadata(ctx echo.Context) error {
	connector, err := source.ParseType(ctx.Param("connector"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connector")
	}

	rootNode, err := h.graphDb.GetCategoryRootByName(ctx.Request().Context(), RootTypeConnectorRoot, connector.String())
	if err != nil {
		return err
	}
	resourceTypes := make([]string, 0)
	for _, filter := range rootNode.SubTreeFilters {
		if filter.GetFilterType() == FilterTypeCloudResourceType {
			resourceTypes = append(resourceTypes, filter.(*FilterCloudResourceTypeNode).ResourceType)
		}
	}

	serviceNodes, err := h.graphDb.GetCloudServiceNodes(ctx.Request().Context(), connector)
	if err != nil {
		return err
	}
	services := make([]string, 0)
	for _, serviceNode := range serviceNodes {
		services = append(services, serviceNode.ServiceName)
	}

	result := api.ConnectorMetadata{
		Connector:      connector,
		ConnectorLabel: connector.String(),
		ResourceTypes:  resourceTypes,
		Services:       services,
		LogoURI:        rootNode.LogoURI,
	}

	return ctx.JSON(http.StatusOK, result)
}

// ListServiceMetadata godoc
//
//	@Summary		Get List of Cloud Services
//	@Description	Gets a list of all workspace cloud services and their metadata inclouding parent service, list of resource types and cost support.
//	@Description	The results could be filtered by cost support and resource type.
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			connector		query		source.Type	true	"Connector"
//	@Param			costSupport		query		boolean		false	"Filter by cost support"
//	@Param			resourceType	query		string		false	"Filter by resource types"
//
//	@Success		200				{object}	[]api.ServiceMetadata
//	@Router			/inventory/api/v2/metadata/services [get]
func (h *HttpHandler) ListServiceMetadata(ctx echo.Context) error {
	connector, err := source.ParseType(ctx.QueryParam("connector"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connector")
	}

	costSupportFilterStr := ctx.QueryParam("costSupport")
	costSupportFilter := false
	if costSupportFilterStr != "" {
		costSupportFilter, err = strconv.ParseBool(costSupportFilterStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid costSupport")
		}
	}

	resourceTypes := ctx.QueryParams()["resourceType"]
	if len(resourceTypes) == 0 {
		resourceTypes = nil
	}

	services, err := h.graphDb.GetCloudServiceNodes(ctx.Request().Context(), connector)
	if err != nil {
		return err
	}

	var result []api.ServiceMetadata
	for _, service := range services {
		costSupport := false
		for _, filter := range service.Filters {
			if filter.GetFilterType() == FilterTypeCost {
				costSupport = true
				break
			}
		}
		if costSupportFilterStr != "" && (costSupport != costSupportFilter) {
			continue
		}
		serviceResourceTypes := make([]string, 0)
		for _, filter := range service.Filters {
			if filter.GetFilterType() == FilterTypeCloudResourceType {
				serviceResourceTypes = append(serviceResourceTypes, filter.(*FilterCloudResourceTypeNode).ResourceType)
			}
		}
		if resourceTypes != nil {
			if !utils.IncludesAll(serviceResourceTypes, resourceTypes) {
				continue
			}
		}
		result = append(result, api.ServiceMetadata{
			Connector:          service.Connector,
			ServiceName:        service.ServiceName,
			ServiceLabel:       service.Name,
			ParentService:      service.GetParentService(),
			ResourceTypesCount: utils.GetPointer(len(serviceResourceTypes)),
			CostSupport:        costSupport,
			LogoURI:            service.LogoURI,
		})
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetServiceMetadata godoc
//
//	@Summary		Get Cloud Service Details
//	@Description	Gets a single cloud service details and its metadata inclouding parent service, list of resource types, cost support and costmap service names.
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			serviceName	path		string	true	"serviceName"
//	@Success		200			{object}	api.ServiceMetadata
//	@Router			/inventory/api/v2/metadata/services/{serviceName} [get]
func (h *HttpHandler) GetServiceMetadata(ctx echo.Context) error {
	serviceName := ctx.Param("serviceName")

	service, err := h.graphDb.GetCloudServiceNode(ctx.Request().Context(), source.Nil, serviceName)
	if err != nil {
		return err
	}

	costSupport := false
	costMapServiceNames := make([]string, 0)
	for _, filter := range service.Filters {
		if filter.GetFilterType() == FilterTypeCost {
			costSupport = true
			costMapServiceNames = append(costMapServiceNames, filter.(*FilterCostNode).CostServiceName)
		}
	}
	if costSupport == false {
		costMapServiceNames = nil
	}
	serviceResourceTypes := make([]string, 0)
	for _, filter := range service.Filters {
		if filter.GetFilterType() == FilterTypeCloudResourceType {
			serviceResourceTypes = append(serviceResourceTypes, filter.(*FilterCloudResourceTypeNode).ResourceType)
		}
	}

	result := api.ServiceMetadata{
		Connector:           service.Connector,
		ServiceName:         service.ServiceName,
		ServiceLabel:        service.Name,
		ParentService:       service.GetParentService(),
		ResourceTypes:       serviceResourceTypes,
		CostSupport:         costSupport,
		CostMapServiceNames: costMapServiceNames,
		LogoURI:             service.LogoURI,
	}

	return ctx.JSON(http.StatusOK, result)
}

// ListResourceTypeMetadata godoc
//
//	@Summary		Get List of Resource Types
//	@Description	Gets a list of all resource types in workspace and their metadata including service name.
//	@Description	The results could be filtered by provider name and service name.
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			connector	query		source.Type	true	"Filter by Connector"
//	@Param			service		query		string		false	"Filter by service name"
//	@Success		200			{object}	[]api.ResourceTypeMetadata
//	@Router			/inventory/api/v2/metadata/resourcetype [get]
func (h *HttpHandler) ListResourceTypeMetadata(ctx echo.Context) error {
	connector, err := source.ParseType(ctx.QueryParam("connector"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connector")
	}

	serviceNames := ctx.QueryParams()["service"]
	filterTypeCloudResourceType := FilterTypeCloudResourceType
	filters, err := h.graphDb.GetFilters(ctx.Request().Context(), connector, serviceNames, &filterTypeCloudResourceType)
	if err != nil {
		return err
	}

	var result []api.ResourceTypeMetadata

	for _, filter := range filters {
		resourceTypeNode := filter.(*FilterCloudResourceTypeNode)

		res := api.ResourceTypeMetadata{
			Connector:         resourceTypeNode.Connector,
			ResourceTypeName:  resourceTypeNode.ResourceType,
			ResourceTypeLabel: resourceTypeNode.ResourceLabel,
			ServiceName:       resourceTypeNode.ServiceName,
			DiscoveryEnabled:  true,
			LogoURI:           resourceTypeNode.LogoURI,
		}

		var table string
		switch connector {
		case source.CloudAWS:
			table = awsSteampipe.ExtractTableName(resourceTypeNode.ResourceType)
		case source.CloudAzure:
			table = azureSteampipe.ExtractTableName(resourceTypeNode.ResourceType)
		}
		insightTableCount := 0
		if table != "" {
			insightList, err := h.complianceClient.GetInsights(httpclient.FromEchoContext(ctx), resourceTypeNode.Connector)
			if err != nil {
				return err
			}
			for _, insightEntity := range insightList {
				for _, insightTable := range strings.Split(insightEntity.Query.ListOfTables, ",") {
					if insightTable == table {
						insightTableCount++
						break
					}
				}
			}
		}
		res.InsightsCount = utils.GetPointerOrNil(insightTableCount)

		result = append(result, res)
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetResourceTypeMetadata godoc
//
//	@Summary		Get Resource Type
//	@Description	Get a single resource type metadata and its details including service name and insights list. Specified by resource type name.
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			resourceType	path		string	true	"resourceType"
//	@Success		200				{object}	[]api.ResourceTypeMetadata
//	@Router			/inventory/api/v2/metadata/resourcetype/{resourceType} [get]
func (h *HttpHandler) GetResourceTypeMetadata(ctx echo.Context) error {
	resourceType := ctx.Param("resourceType")

	resourceTypeNode, err := h.graphDb.GetResourceType(ctx.Request().Context(), source.Nil, resourceType)
	if err != nil {
		return err
	}

	result := api.ResourceTypeMetadata{
		Connector:         resourceTypeNode.Connector,
		ResourceTypeName:  resourceTypeNode.ResourceType,
		ResourceTypeLabel: resourceTypeNode.ResourceLabel,
		ServiceName:       resourceTypeNode.ServiceName,
		DiscoveryEnabled:  true,
		LogoURI:           resourceTypeNode.LogoURI,
	}

	var table string
	switch resourceTypeNode.Connector {
	case source.CloudAWS:
		table = awsSteampipe.ExtractTableName(resourceTypeNode.ResourceType)
	case source.CloudAzure:
		table = azureSteampipe.ExtractTableName(resourceTypeNode.ResourceType)
	}
	if table != "" {
		insightTables := make([]uint, 0)
		insightList, err := h.complianceClient.GetInsights(httpclient.FromEchoContext(ctx), resourceTypeNode.Connector)
		if err != nil {
			return err
		}
		for _, insightEntity := range insightList {
			for _, insightTable := range strings.Split(insightEntity.Query.ListOfTables, ",") {
				if insightTable == table {
					insightTables = append(insightTables, insightEntity.ID)
					break
				}
			}
		}
		result.Insights = insightTables

		switch resourceTypeNode.Connector {
		case source.CloudAWS:
			result.Attributes, _ = steampipe.Cells(h.awsPlg, table)
		case source.CloudAzure:
			result.Attributes, err = steampipe.Cells(h.azurePlg, table)
			if err != nil {
				result.Attributes, _ = steampipe.Cells(h.azureADPlg, table)
			}
		}
	}

	return ctx.JSON(http.StatusOK, result)
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
