package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/demo"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	httpserver2 "github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-util/pkg/describe"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	analyticsDB "github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	insight "github.com/kaytu-io/kaytu-engine/pkg/insight/es"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/es"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	EsFetchPageSize = 10000
	MaxConns        = 100
	KafkaPageSize   = 5000
)

const (
	AWSLogoURI   = "https://raw.githubusercontent.com/kaytu-io/awsicons/master/svg-export/icons/AWS.svg"
	AzureLogoURI = "https://raw.githubusercontent.com/kaytu-io/Azure-Design/master/SVG_Azure_All/Azure.svg"
)

const (
	ConnectionIdParam    = "connectionId"
	ConnectionGroupParam = "connectionGroup"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	queryV1 := v1.Group("/query")
	queryV1.GET("", httpserver2.AuthorizeHandler(h.ListQueries, authApi.ViewerRole))
	queryV1.POST("/run", httpserver2.AuthorizeHandler(h.RunQuery, authApi.ViewerRole))
	queryV1.GET("/run/history", httpserver2.AuthorizeHandler(h.GetRecentRanQueries, authApi.ViewerRole))

	v2 := e.Group("/api/v2")

	resourcesV2 := v2.Group("/resources")
	resourcesV2.GET("/count", httpserver2.AuthorizeHandler(h.CountResources, authApi.ViewerRole))

	analyticsV2 := v2.Group("/analytics")
	analyticsV2.GET("/count", httpserver2.AuthorizeHandler(h.CountAnalytics, authApi.ViewerRole))
	analyticsV2.GET("/metrics/list", httpserver2.AuthorizeHandler(h.ListMetrics, authApi.ViewerRole))
	analyticsV2.GET("/metrics/:metric_id", httpserver2.AuthorizeHandler(h.GetMetric, authApi.ViewerRole))

	analyticsV2.GET("/metric", httpserver2.AuthorizeHandler(h.ListAnalyticsMetricsHandler, authApi.ViewerRole))
	analyticsV2.GET("/tag", httpserver2.AuthorizeHandler(h.ListAnalyticsTags, authApi.ViewerRole))
	analyticsV2.GET("/trend", httpserver2.AuthorizeHandler(h.ListAnalyticsMetricTrend, authApi.ViewerRole))
	analyticsV2.GET("/composition/:key", httpserver2.AuthorizeHandler(h.ListAnalyticsComposition, authApi.ViewerRole))
	analyticsV2.GET("/categories", httpserver2.AuthorizeHandler(h.ListAnalyticsCategories, authApi.ViewerRole))
	analyticsV2.GET("/table", httpserver2.AuthorizeHandler(h.GetAssetsTable, authApi.ViewerRole))

	analyticsSpend := analyticsV2.Group("/spend")
	analyticsSpend.GET("/count", httpserver2.AuthorizeHandler(h.CountAnalyticsSpend, authApi.ViewerRole))
	analyticsSpend.GET("/metric", httpserver2.AuthorizeHandler(h.ListAnalyticsSpendMetricsHandler, authApi.ViewerRole))
	analyticsSpend.GET("/composition", httpserver2.AuthorizeHandler(h.ListAnalyticsSpendComposition, authApi.ViewerRole))
	analyticsSpend.GET("/trend", httpserver2.AuthorizeHandler(h.GetAnalyticsSpendTrend, authApi.ViewerRole))
	analyticsSpend.GET("/table", httpserver2.AuthorizeHandler(h.GetSpendTable, authApi.ViewerRole))

	connectionsV2 := v2.Group("/connections")
	connectionsV2.GET("/data", httpserver2.AuthorizeHandler(h.ListConnectionsData, authApi.ViewerRole))

	insightsV2 := v2.Group("/insights")
	insightsV2.GET("", httpserver2.AuthorizeHandler(h.ListInsightResults, authApi.ViewerRole))
	insightsV2.GET("/:insightId/trend", httpserver2.AuthorizeHandler(h.GetInsightTrendResults, authApi.ViewerRole))
	insightsV2.GET("/:insightId", httpserver2.AuthorizeHandler(h.GetInsightResult, authApi.ViewerRole))

	resourceCollection := v2.Group("/resource-collection")
	resourceCollection.GET("", httpserver2.AuthorizeHandler(h.ListResourceCollections, authApi.ViewerRole))
	resourceCollection.GET("/:resourceCollectionId", httpserver2.AuthorizeHandler(h.GetResourceCollection, authApi.ViewerRole))
	resourceCollection.GET("/:resourceCollectionId/landscape", httpserver2.AuthorizeHandler(h.GetResourceCollectionLandscape, authApi.ViewerRole))

	metadata := v2.Group("/metadata")
	metadata.GET("/resourcetype", httpserver2.AuthorizeHandler(h.ListResourceTypeMetadata, authApi.ViewerRole))

	resourceCollectionMetadata := metadata.Group("/resource-collection")
	resourceCollectionMetadata.GET("", httpserver2.AuthorizeHandler(h.ListResourceCollectionsMetadata, authApi.ViewerRole))
	resourceCollectionMetadata.GET("/:resourceCollectionId", httpserver2.AuthorizeHandler(h.GetResourceCollectionMetadata, authApi.ViewerRole))
}

var tracer = otel.Tracer("new_inventory")

func (h *HttpHandler) getConnectionIdFilterFromParams(ctx echo.Context) ([]string, error) {
	connectionIds := httpserver2.QueryArrayParam(ctx, ConnectionIdParam)
	connectionGroup := httpserver2.QueryArrayParam(ctx, ConnectionGroupParam)
	if len(connectionIds) == 0 && len(connectionGroup) == 0 {
		return nil, nil
	}

	if len(connectionIds) > 0 && len(connectionGroup) > 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "connectionId and connectionGroup cannot be used together")
	}

	if len(connectionIds) > 0 {
		return connectionIds, nil
	}

	connectionMap := map[string]bool{}
	for _, connectionGroupID := range connectionGroup {
		connectionGroupObj, err := h.onboardClient.GetConnectionGroup(&httpclient.Context{UserRole: authApi.InternalRole}, connectionGroupID)
		if err != nil {
			return nil, err
		}
		for _, connectionID := range connectionGroupObj.ConnectionIds {
			connectionMap[connectionID] = true
		}
	}
	connectionIds = make([]string, 0, len(connectionMap))
	for connectionID := range connectionMap {
		connectionIds = append(connectionIds, connectionID)
	}
	if len(connectionIds) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "connectionGroup(s) do not have any connections")
	}

	return connectionIds, nil
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

func (h *HttpHandler) getConnectorTypesFromConnectionIDs(ctx echo.Context, connectorTypes []source.Type, connectionIDs []string) ([]source.Type, error) {
	if len(connectionIDs) == 0 {
		return connectorTypes, nil
	}
	if len(connectorTypes) != 0 {
		return connectorTypes, nil
	}
	connections, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), connectionIDs)
	if err != nil {
		return nil, err
	}

	enabledConnectors := make(map[source.Type]bool)
	for _, connection := range connections {
		enabledConnectors[connection.Connector] = true
	}
	connectorTypes = make([]source.Type, 0, len(enabledConnectors))
	for connectorType := range enabledConnectors {
		connectorTypes = append(connectorTypes, connectorType)
	}

	return connectorTypes, nil
}

func (h *HttpHandler) ListAnalyticsMetrics(ctx context.Context,
	metricIDs []string, metricType analyticsDB.MetricType, tagMap map[string][]string,
	connectorTypes []source.Type, connectionIDs []string, resourceCollections []string,
	minCount int, timeAt time.Time) (*int, []inventoryApi.Metric, error) {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	// tracer :
	_, span := tracer.Start(ctx, "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredMetrics")
	mts, err := aDB.ListFilteredMetrics(tagMap, metricType, metricIDs, connectorTypes, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, err
	}
	span.End()

	filteredMetricIDs := make([]string, 0, len(mts))
	for _, metric := range mts {
		filteredMetricIDs = append(filteredMetricIDs, metric.ID)
	}

	var metricIndexed map[string]es.CountWithTime
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionAnalyticMetricCountAtTime(h.logger, h.client, filteredMetricIDs, connectorTypes, connectionIDs, resourceCollections, timeAt, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorAnalyticMetricCountAtTime(h.logger, h.client, filteredMetricIDs, connectorTypes, resourceCollections, timeAt, EsFetchPageSize)
	}
	if err != nil {
		return nil, nil, err
	}

	apiMetrics := make([]inventoryApi.Metric, 0, len(mts))
	var totalCount *int
	for _, metric := range mts {
		apiMetric := inventoryApi.MetricToAPI(metric)
		if countWithTime, ok := metricIndexed[metric.ID]; ok && countWithTime.Count >= minCount {
			apiMetric.Count = &countWithTime.Count
			if apiMetric.LastEvaluated == nil || apiMetric.LastEvaluated.IsZero() || apiMetric.LastEvaluated.Before(countWithTime.Time) {
				apiMetric.LastEvaluated = &countWithTime.Time
			}
			totalCount = utils.PAdd(totalCount, &countWithTime.Count)
		}
		if (minCount == 0) || (apiMetric.Count != nil && *apiMetric.Count >= minCount) {
			apiMetrics = append(apiMetrics, apiMetric)
		}
	}

	return totalCount, apiMetrics, nil
}

// ListAnalyticsMetricsHandler godoc
//
//	@Summary		List analytics metrics
//	@Description	Retrieving list of analytics with metrics of each type based on the given input filters.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			tag					query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			metricType			query		string			false	"Metric type, default: assets"	Enums(assets, spend)
//	@Param			connector			query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId		query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup		query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			resourceCollection	query		[]string		false	"Resource collection IDs to filter by"
//	@Param			metricIDs			query		[]string		false	"Metric IDs"
//	@Param			endTime				query		int64			false	"timestamp for resource count in epoch seconds"
//	@Param			startTime			query		int64			false	"timestamp for resource count change comparison in epoch seconds"
//	@Param			minCount			query		int				false	"Minimum number of resources with this tag value, default 0"
//	@Param			sortBy				query		string			false	"Sort by field - default is count"	Enums(name,count,growth,growth_rate)
//	@Param			pageSize			query		int				false	"page size - default is 20"
//	@Param			pageNumber			query		int				false	"page number - default is 1"
//	@Success		200					{object}	inventoryApi.ListMetricsResponse
//	@Router			/inventory/api/v2/analytics/metric [get]
func (h *HttpHandler) ListAnalyticsMetricsHandler(ctx echo.Context) error {
	var err error
	tagMap := model.TagStringsToTagMap(httpserver2.QueryArrayParam(ctx, "tag"))
	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	if metricType == "" {
		metricType = analyticsDB.MetricTypeAssets
	}
	connectorTypes := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	if len(connectionIDs) > MaxConns {
		return ctx.JSON(http.StatusBadRequest, "too many connections")
	}
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
	metricIDs := httpserver2.QueryArrayParam(ctx, "metricIDs")

	connectorTypes, err = h.getConnectorTypesFromConnectionIDs(ctx, connectorTypes, connectionIDs)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	endTime := time.Now()
	if endTimeStr := ctx.QueryParam("endTime"); endTimeStr != "" {
		endTimeVal, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid endTime value")
		}
		endTime = time.Unix(endTimeVal, 0)
	}
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr := ctx.QueryParam("startTime"); startTimeStr != "" {
		startTimeVal, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid startTime value")
		}
		startTime = time.Unix(startTimeVal, 0)
	}
	minCount := 0
	if minCountStr := ctx.QueryParam("minCount"); minCountStr != "" {
		minCountVal, err := strconv.ParseInt(minCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "minCount must be a number")
		}
		minCount = int(minCountVal)
	}
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	sortBy := strings.ToLower(ctx.QueryParam("sortBy"))
	if sortBy == "" {
		sortBy = "count"
	}
	if sortBy != "name" && sortBy != "count" &&
		sortBy != "growth" && sortBy != "growth_rate" {
		return ctx.JSON(http.StatusBadRequest, "invalid sortBy value")
	}

	totalCount, apiMetrics, err := h.ListAnalyticsMetrics(ctx.Request().Context(),
		metricIDs, metricType, tagMap, connectorTypes, connectionIDs, resourceCollections, minCount, endTime)
	if err != nil {
		return err
	}

	apiMetricsMap := make(map[string]inventoryApi.Metric, len(apiMetrics))
	for _, apiMetric := range apiMetrics {
		apiMetricsMap[apiMetric.ID] = apiMetric
	}

	totalOldCount, oldApiMetrics, err := h.ListAnalyticsMetrics(ctx.Request().Context(),
		metricIDs, metricType, tagMap, connectorTypes, connectionIDs, resourceCollections, 0, startTime)
	if err != nil {
		return err
	}
	for _, oldApiMetric := range oldApiMetrics {
		if apiMetric, ok := apiMetricsMap[oldApiMetric.ID]; ok {
			apiMetric.OldCount = oldApiMetric.Count
			apiMetricsMap[oldApiMetric.ID] = apiMetric
		}
	}

	apiMetrics = make([]inventoryApi.Metric, 0, len(apiMetricsMap))
	for _, apiMetric := range apiMetricsMap {
		apiMetrics = append(apiMetrics, apiMetric)
	}

	sort.Slice(apiMetrics, func(i, j int) bool {
		switch sortBy {
		case "name":
			return apiMetrics[i].Name < apiMetrics[j].Name
		case "count":
			if apiMetrics[i].Count == nil && apiMetrics[j].Count == nil {
				break
			}
			if apiMetrics[i].Count == nil {
				return false
			}
			if apiMetrics[j].Count == nil {
				return true
			}
			if *apiMetrics[i].Count != *apiMetrics[j].Count {
				return *apiMetrics[i].Count > *apiMetrics[j].Count
			}
		case "growth":
			diffi := utils.PSub(apiMetrics[i].Count, apiMetrics[i].OldCount)
			diffj := utils.PSub(apiMetrics[j].Count, apiMetrics[j].OldCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if *diffi != *diffj {
				return *diffi > *diffj
			}
		case "growth_rate":
			diffi := utils.PSub(apiMetrics[i].Count, apiMetrics[i].OldCount)
			diffj := utils.PSub(apiMetrics[j].Count, apiMetrics[j].OldCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if apiMetrics[i].OldCount == nil && apiMetrics[j].OldCount == nil {
				break
			}
			if apiMetrics[i].OldCount == nil {
				return true
			}
			if apiMetrics[j].OldCount == nil {
				return false
			}
			if *apiMetrics[i].OldCount == 0 && *apiMetrics[j].OldCount == 0 {
				break
			}
			if *apiMetrics[i].OldCount == 0 {
				return false
			}
			if *apiMetrics[j].OldCount == 0 {
				return true
			}
			if float64(*diffi)/float64(*apiMetrics[i].OldCount) != float64(*diffj)/float64(*apiMetrics[j].OldCount) {
				return float64(*diffi)/float64(*apiMetrics[i].OldCount) > float64(*diffj)/float64(*apiMetrics[j].OldCount)
			}
		}
		return apiMetrics[i].Name < apiMetrics[j].Name
	})

	result := inventoryApi.ListMetricsResponse{
		TotalCount:    totalCount,
		TotalOldCount: totalOldCount,
		TotalMetrics:  len(apiMetrics),
		Metrics:       utils.Paginate(pageNumber, pageSize, apiMetrics),
	}

	return ctx.JSON(http.StatusOK, result)
}

// ListAnalyticsTags godoc
//
//	@Summary		List analytics tags
//	@Description	Retrieving a list of tag keys with their possible values for all analytic metrics.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			connector			query		[]string	false	"Connector type to filter by"
//	@Param			connectionId		query		[]string	false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup		query		[]string	false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			resourceCollection	query		[]string	false	"Resource collection IDs to filter by"
//	@Param			minCount			query		int			false	"Minimum number of resources/spend with this tag value, default 1"
//	@Param			startTime			query		int64		false	"Start time in unix timestamp format, default now - 1 month"
//	@Param			endTime				query		int64		false	"End time in unix timestamp format, default now"
//	@Param			metricType			query		string		false	"Metric type, default: assets"	Enums(assets, spend)
//	@Success		200					{object}	map[string][]string
//	@Router			/inventory/api/v2/analytics/tag [get]
func (h *HttpHandler) ListAnalyticsTags(ctx echo.Context) error {
	connectorTypes := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if len(connectionIDs) > MaxConns {
		return ctx.JSON(http.StatusBadRequest, "too many connections")
	}
	connectorTypes, err = h.getConnectorTypesFromConnectionIDs(ctx, connectorTypes, connectionIDs)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	minCount := 1
	if minCountStr := ctx.QueryParam("minCount"); minCountStr != "" {
		minCountVal, err := strconv.ParseInt(minCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "minCount must be a number")
		}
		minCount = int(minCountVal)
	}
	minAmount := float64(minCount)
	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "endTime must be a number")
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "startTime must be a number")
	}

	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	if metricType == "" {
		metricType = analyticsDB.MetricTypeAssets
	}
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
	if len(resourceCollections) > 0 && metricType == analyticsDB.MetricTypeSpend {
		return ctx.JSON(http.StatusBadRequest, "ResourceCollections are not supported for spend metrics")
	}

	aDB := analyticsDB.NewDatabase(h.db.orm)
	fmt.Println("connectorTypes", connectorTypes)
	// trace :
	outputS1, span1 := tracer.Start(ctx.Request().Context(), "new_ListMetricTagsKeysWithPossibleValues", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListMetricTagsKeysWithPossibleValues")

	tags, err := aDB.ListMetricTagsKeysWithPossibleValues(connectorTypes)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()
	tags = model.TrimPrivateTags(tags)

	var metricCount map[string]es.CountWithTime
	var spend map[string]es.SpendMetricResp

	if metricType == analyticsDB.MetricTypeAssets {
		if len(connectionIDs) > 0 {
			metricCount, err = es.FetchConnectionAnalyticMetricCountAtTime(h.logger, h.client, nil, connectorTypes, connectionIDs, resourceCollections, endTime, EsFetchPageSize)
		} else {
			metricCount, err = es.FetchConnectorAnalyticMetricCountAtTime(h.logger, h.client, nil, connectorTypes, resourceCollections, endTime, EsFetchPageSize)
		}
		if err != nil {
			return err
		}
	} else {
		spend, err = es.FetchSpendByMetric(h.client, connectionIDs, connectorTypes, nil, startTime, endTime, EsFetchPageSize)
		if err != nil {
			return err
		}
	}

	fmt.Println("metricCount", metricCount)
	fmt.Println("spend", spend)
	fmt.Println("tags", tags)

	filteredTags := map[string][]string{}
	// tracer:
	outputS2, span2 := tracer.Start(outputS1, "new_ListFilteredMetrics(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_ListFilteredMetrics(loop)")

	for key, values := range tags {
		for _, tagValue := range values {
			_, span3 := tracer.Start(outputS2, "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
			span3.SetName("new_ListFilteredMetrics")

			metrics, err := aDB.ListFilteredMetrics(map[string][]string{
				key: {tagValue},
			}, metricType, nil, connectorTypes, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
			if err != nil {
				span3.RecordError(err)
				span3.SetStatus(codes.Error, err.Error())
				return err
			}
			span3.End()

			fmt.Println("metrics", key, tagValue, metrics)
			for _, metric := range metrics {
				if (metric.Type == analyticsDB.MetricTypeAssets && metricCount[metric.ID].Count >= minCount) ||
					(metric.Type == analyticsDB.MetricTypeSpend && spend[metric.ID].CostValue >= minAmount) {
					filteredTags[key] = append(filteredTags[key], tagValue)
					break
				}
			}
		}
	}
	tags = filteredTags
	fmt.Println("filteredTags", filteredTags)

	return ctx.JSON(http.StatusOK, tags)
}

// ListAnalyticsMetricTrend godoc
//
//	@Summary		Get metric trend
//
//	@Description	Retrieving a list of resource counts over the course of the specified time frame based on the given input filters
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			tag					query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			metricType			query		string			false	"Metric type, default: assets"	Enums(assets, spend)
//	@Param			ids					query		[]string		false	"Metric IDs to filter by"
//	@Param			connector			query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId		query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup		query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			resourceCollection	query		[]string		false	"Resource collection IDs to filter by"
//	@Param			startTime			query		int64			false	"timestamp for start in epoch seconds"
//	@Param			endTime				query		int64			false	"timestamp for end in epoch seconds"
//	@Param			granularity			query		string			false	"Granularity of the table, default is daily"	Enums(monthly, daily, yearly)
//	@Success		200					{object}	[]inventoryApi.ResourceTypeTrendDatapoint
//	@Router			/inventory/api/v2/analytics/trend [get]
func (h *HttpHandler) ListAnalyticsMetricTrend(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	tagMap := model.TagStringsToTagMap(httpserver2.QueryArrayParam(ctx, "tag"))
	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	if metricType == "" {
		metricType = analyticsDB.MetricTypeAssets
	}
	ids := httpserver2.QueryArrayParam(ctx, "ids")
	connectorTypes := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	if len(connectionIDs) > MaxConns {
		return echo.NewHTTPError(http.StatusBadRequest, "too many connections")
	}

	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")

	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now()
	if endTimeStr != "" {
		endTimeVal, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		endTime = time.Unix(endTimeVal, 0)
	}
	startTimeStr := ctx.QueryParam("startTime")
	startTime := endTime.AddDate(0, -1, 0)
	if startTimeStr != "" {
		startTimeVal, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		startTime = time.Unix(startTimeVal, 0)
	}

	granularity := inventoryApi.TableGranularityType(ctx.QueryParam("granularity"))
	if granularity == "" {
		granularity = inventoryApi.TableGranularityTypeDaily
	}
	if granularity != inventoryApi.TableGranularityTypeDaily &&
		granularity != inventoryApi.TableGranularityTypeMonthly &&
		granularity != inventoryApi.TableGranularityTypeYearly {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid granularity")
	}

	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredMetrics")

	metrics, err := aDB.ListFilteredMetrics(tagMap, metricType, ids, connectorTypes, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	metricIDs := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		metricIDs = append(metricIDs, metric.ID)
	}

	timeToCountMap := make(map[int]es.DatapointWithFailures)
	if endTime.Round(24 * time.Hour).Before(endTime) {
		endTime = endTime.Round(24 * time.Hour).Add(24 * time.Hour)
	} else {
		endTime = endTime.Round(24 * time.Hour)
	}
	if startTime.Round(24 * time.Hour).After(startTime) {
		startTime = startTime.Round(24 * time.Hour).Add(-24 * time.Hour)
	} else {
		startTime = startTime.Round(24 * time.Hour)
	}

	esDatapointCount := int(math.Floor(endTime.Sub(startTime).Hours() / 24))
	if esDatapointCount == 0 {
		esDatapointCount = 1
	}
	if len(connectionIDs) != 0 {
		timeToCountMap, err = es.FetchConnectionMetricTrendSummaryPage(h.logger, h.client, connectionIDs, connectorTypes, metricIDs, resourceCollections, startTime, endTime, esDatapointCount, EsFetchPageSize)
		if err != nil {
			return err
		}
	} else {
		timeToCountMap, err = es.FetchConnectorMetricTrendSummaryPage(h.logger, h.client, connectorTypes, metricIDs, resourceCollections, startTime, endTime, esDatapointCount, EsFetchPageSize)
		if err != nil {
			return err
		}
	}

	apiDatapoints := make([]inventoryApi.ResourceTypeTrendDatapoint, 0, len(timeToCountMap))
	for timeAt, val := range timeToCountMap {
		apiDatapoints = append(apiDatapoints, inventoryApi.ResourceTypeTrendDatapoint{
			Count:                                   val.Count,
			TotalDescribedConnectionCount:           val.TotalConnections,
			TotalSuccessfulDescribedConnectionCount: val.TotalSuccessfulConnections,
			Date:                                    time.UnixMilli(int64(timeAt)),
		})
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})

	filteredDatapointMap := make(map[int]inventoryApi.ResourceTypeTrendDatapoint)
	for _, apiDatapoint := range apiDatapoints {
		key := apiDatapoint.Date.Year()*10000 + int(apiDatapoint.Date.Month())*100 + apiDatapoint.Date.Day()
		switch granularity {
		case inventoryApi.TableGranularityTypeMonthly:
			key = apiDatapoint.Date.Year()*100 + int(apiDatapoint.Date.Month())
		case inventoryApi.TableGranularityTypeYearly:
			key = apiDatapoint.Date.Year()
		}
		if _, ok := filteredDatapointMap[key]; !ok {
			filteredDatapointMap[key] = apiDatapoint
		}
	}

	apiDatapoints = make([]inventoryApi.ResourceTypeTrendDatapoint, 0, len(filteredDatapointMap))
	for _, apiDatapoint := range filteredDatapointMap {
		apiDatapoints = append(apiDatapoints, apiDatapoint)
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})

	return ctx.JSON(http.StatusOK, apiDatapoints)
}

// ListAnalyticsComposition godoc
//
//	@Summary		List analytics composition
//	@Description	Retrieving tag values with the most resources for the given key.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			key					path		string			true	"Tag key"
//	@Param			metricType			query		string			false	"Metric type, default: assets"	Enums(assets, spend)
//	@Param			top					query		int				true	"How many top values to return default is 5"
//	@Param			connector			query		[]source.Type	false	"Connector types to filter by"
//	@Param			connectionId		query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup		query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			resourceCollection	query		[]string		false	"Resource collection IDs to filter by"
//	@Param			endTime				query		int64			false	"timestamp for resource count in epoch seconds"
//	@Param			startTime			query		int64			false	"timestamp for resource count change comparison in epoch seconds"
//	@Success		200					{object}	inventoryApi.ListResourceTypeCompositionResponse
//	@Router			/inventory/api/v2/analytics/composition/{key} [get]
func (h *HttpHandler) ListAnalyticsComposition(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)

	var err error
	tagKey := ctx.Param("key")
	if tagKey == "" || strings.HasPrefix(tagKey, model.KaytuPrivateTagPrefix) {
		return echo.NewHTTPError(http.StatusBadRequest, "tag key is invalid")
	}
	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	if metricType == "" {
		metricType = analyticsDB.MetricTypeAssets
	}

	topStr := ctx.QueryParam("top")
	top := int64(5)
	if topStr != "" {
		top, err = strconv.ParseInt(topStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid top value")
		}

	}
	connectorTypes := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	if len(connectionIDs) > MaxConns {
		return ctx.JSON(http.StatusBadRequest, "too many connections")
	}

	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")

	endTime := time.Now()
	if endTimeStr := ctx.QueryParam("endTime"); endTimeStr != "" {
		endTimeVal, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid endTime value")
		}
		endTime = time.Unix(endTimeVal, 0)
	}
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr := ctx.QueryParam("startTime"); startTimeStr != "" {
		startTimeVal, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "invalid startTime value")
		}
		startTime = time.Unix(startTimeVal, 0)
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredMetrics")

	filteredMetrics, err := aDB.ListFilteredMetrics(map[string][]string{tagKey: nil}, metricType,
		nil, connectorTypes, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	var metrics []analyticsDB.AnalyticMetric
	for _, metric := range filteredMetrics {
		metrics = append(metrics, metric)
	}
	metricsIDs := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		metricsIDs = append(metricsIDs, metric.ID)
	}

	var metricIndexed map[string]es.CountWithTime
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionAnalyticMetricCountAtTime(h.logger, h.client, metricsIDs, connectorTypes, connectionIDs, resourceCollections, endTime, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorAnalyticMetricCountAtTime(h.logger, h.client, metricsIDs, connectorTypes, resourceCollections, endTime, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	var oldMetricIndexed map[string]es.CountWithTime
	if len(connectionIDs) > 0 {
		oldMetricIndexed, err = es.FetchConnectionAnalyticMetricCountAtTime(h.logger, h.client, metricsIDs, connectorTypes, connectionIDs, resourceCollections, startTime, EsFetchPageSize)
	} else {
		oldMetricIndexed, err = es.FetchConnectorAnalyticMetricCountAtTime(h.logger, h.client, metricsIDs, connectorTypes, resourceCollections, startTime, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	type currentAndOldCount struct {
		current int
		old     int
	}

	valueCountMap := make(map[string]currentAndOldCount)
	totalCount := 0
	totalOldCount := 0
	for _, metric := range metrics {
		for _, tagValue := range metric.GetTagsMap()[tagKey] {
			if _, ok := valueCountMap[tagValue]; !ok {
				valueCountMap[tagValue] = currentAndOldCount{}
			}
			v := valueCountMap[tagValue]
			v.current += metricIndexed[metric.ID].Count
			v.old += oldMetricIndexed[metric.ID].Count
			totalCount += metricIndexed[metric.ID].Count
			totalOldCount += oldMetricIndexed[metric.ID].Count
			valueCountMap[tagValue] = v
			break
		}
	}

	type strIntPair struct {
		str    string
		counts currentAndOldCount
	}
	valueCountPairs := make([]strIntPair, 0, len(valueCountMap))
	for value, count := range valueCountMap {
		valueCountPairs = append(valueCountPairs, strIntPair{str: value, counts: count})
	}
	sort.Slice(valueCountPairs, func(i, j int) bool {
		return valueCountPairs[i].counts.current > valueCountPairs[j].counts.current
	})

	apiResult := inventoryApi.ListResourceTypeCompositionResponse{
		TotalCount:      totalCount,
		TotalValueCount: len(valueCountMap),
		TopValues:       make(map[string]inventoryApi.CountPair),
		Others:          inventoryApi.CountPair{},
	}

	for i, pair := range valueCountPairs {
		if i < int(top) {
			apiResult.TopValues[pair.str] = inventoryApi.CountPair{
				Count:    pair.counts.current,
				OldCount: pair.counts.old,
			}
		} else {
			apiResult.Others.Count += pair.counts.current
			apiResult.Others.OldCount += pair.counts.old
		}
	}

	return ctx.JSON(http.StatusOK, apiResult)
}

// ListAnalyticsCategories godoc
//
//	@Summary		List Analytics categories
//	@Description	Retrieving list of categories for analytics
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			metricType	query		string	false	"Metric type, default: assets"	Enums(assets, spend)
//	@Param			minCount	query		int		false	"For assets minimum number of resources returned resourcetype must have, default 1"
//	@Success		200			{object}	inventoryApi.AnalyticsCategoriesResponse
//	@Router			/inventory/api/v2/analytics/categories [get]
func (h *HttpHandler) ListAnalyticsCategories(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	if metricType == "" {
		metricType = analyticsDB.MetricTypeAssets
	}
	minCount := 1
	if minCountStr := ctx.QueryParam("minCount"); minCountStr != "" {
		minCountVal, err := strconv.ParseInt(minCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "minCount must be a number")
		}
		minCount = int(minCountVal)
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListMetrics")

	metrics, err := aDB.ListMetrics([]analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	categoryResourceTypeMapMap := make(map[string]map[string]bool)
	for _, metric := range metrics {
		if metric.Type != metricType {
			continue
		}

		for _, tag := range metric.Tags {
			if tag.Key == "category" {
				for _, category := range tag.GetValue() {
					resourceTypeMap, ok := categoryResourceTypeMapMap[category]
					if !ok {
						resourceTypeMap = make(map[string]bool)
					}
					for _, table := range metric.Tables {
						resourceTypeMap[table] = true
					}
					categoryResourceTypeMapMap[category] = resourceTypeMap
				}
			}
		}
	}

	resourceTypeCountMap, err := es.GetResourceTypeCounts(h.client, nil, nil, nil, EsFetchPageSize)
	if err != nil {
		h.logger.Error("failed to get resource type counts", zap.Error(err))
		return err
	}

	for category, resourceTypes := range categoryResourceTypeMapMap {
		for resourceType, _ := range resourceTypes {
			if count, _ := resourceTypeCountMap[strings.ToLower(resourceType)]; count < minCount {
				delete(resourceTypes, resourceType)
			}
		}
		categoryResourceTypeMapMap[category] = resourceTypes
		if len(resourceTypes) == 0 {
			delete(categoryResourceTypeMapMap, category)
		}
	}

	categoryResourceTypeMap := make(map[string][]string)
	for category, resourceTypes := range categoryResourceTypeMapMap {
		for resourceType, _ := range resourceTypes {
			categoryResourceTypeMap[category] = append(categoryResourceTypeMap[category], resourceType)
		}
	}

	return ctx.JSON(http.StatusOK, inventoryApi.AnalyticsCategoriesResponse{
		CategoryResourceType: categoryResourceTypeMap,
	})
}

// GetAssetsTable godoc
//
//	@Summary		Get Assets Table
//	@Description	Returns asset table with respect to the dimension and granularity
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			startTime	query		int64	false	"timestamp for start in epoch seconds"
//	@Param			endTime		query		int64	false	"timestamp for end in epoch seconds"
//	@Param			granularity	query		string	false	"Granularity of the table, default is daily"	Enums(monthly, daily, yearly)
//	@Param			dimension	query		string	false	"Dimension of the table, default is metric"		Enums(connection, metric)
//
//	@Success		200			{object}	[]inventoryApi.AssetTableRow
//	@Router			/inventory/api/v2/analytics/table [get]
func (h *HttpHandler) GetAssetsTable(ctx echo.Context) error {
	var err error
	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	granularity := inventoryApi.TableGranularityType(ctx.QueryParam("granularity"))
	if granularity == "" {
		granularity = inventoryApi.TableGranularityTypeDaily
	}
	if granularity != inventoryApi.TableGranularityTypeDaily &&
		granularity != inventoryApi.TableGranularityTypeMonthly &&
		granularity != inventoryApi.TableGranularityTypeYearly {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid granularity")
	}
	dimension := inventoryApi.DimensionType(ctx.QueryParam("dimension"))
	if dimension == "" {
		dimension = inventoryApi.DimensionTypeMetric
	}
	if dimension != inventoryApi.DimensionTypeMetric &&
		dimension != inventoryApi.DimensionTypeConnection {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid dimension")
	}

	aDB := analyticsDB.NewDatabase(h.db.orm)
	ms, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeAssets,
		nil, nil, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		return err
	}
	var metricIds []string
	for _, m := range ms {
		metricIds = append(metricIds, m.ID)
	}
	mt, err := es.FetchAssetTableByDimension(h.logger, h.client, metricIds, granularity, dimension, startTime, endTime)
	if err != nil {
		return err
	}

	var table []inventoryApi.AssetTableRow
	for _, m := range mt {
		resourceCount := map[string]float64{}
		for dateKey, costItem := range m.Trend {
			resourceCount[dateKey] = costItem
		}
		table = append(table, inventoryApi.AssetTableRow{
			DimensionID:   m.DimensionID,
			DimensionName: m.DimensionName,
			ResourceCount: resourceCount,
		})
	}
	return ctx.JSON(http.StatusOK, table)
}

// ListAnalyticsSpendMetricsHandler godoc
//
//	@Summary		List spend metrics
//	@Description	Retrieving cost metrics with respect to specified filters. The API returns information such as the total cost and costs per each service based on the specified filters.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			filter			query		string			false	"Filter costs"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			startTime		query		int64			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		int64			false	"timestamp for end in epoch seconds"
//	@Param			sortBy			query		string			false	"Sort by field - default is cost"	Enums(dimension,cost,growth,growth_rate)
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Param			metricIDs		query		[]string		false	"Metric IDs"
//	@Success		200				{object}	inventoryApi.ListCostMetricsResponse
//	@Router			/inventory/api/v2/analytics/spend/metric [get]
func (h *HttpHandler) ListAnalyticsSpendMetricsHandler(ctx echo.Context) error {
	var err error
	connectorTypes := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	sortBy := strings.ToLower(ctx.QueryParam("sortBy"))
	if sortBy == "" {
		sortBy = "cost"
	}
	if sortBy != "dimension" && sortBy != "cost" &&
		sortBy != "growth" && sortBy != "growth_rate" {
		return ctx.JSON(http.StatusBadRequest, "invalid sortBy value")
	}

	aDB := analyticsDB.NewDatabase(h.db.orm)
	metricIds := httpserver2.QueryArrayParam(ctx, "metricIDs")
	metrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeSpend,
		metricIds, connectorTypes, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		return err
	}
	metricIds = []string{}
	for _, m := range metrics {
		metricIds = append(metricIds, m.ID)
	}
	metricsMap := make(map[string]analyticsDB.AnalyticMetric)
	for _, m := range metrics {
		metricsMap[m.ID] = m
	}

	filterStr := ctx.QueryParam("filter")
	if filterStr != "" {
		var filter map[string]interface{}
		err = json.Unmarshal([]byte(filterStr), &filter)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "could not parse filter")
		}
		connectionIDs, err = h.connectionsFilter(filter)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, fmt.Sprintf("invalid filter: %s", err.Error()))
		}
		h.logger.Warn(fmt.Sprintf("===Filtered Connections: %v", connectionIDs))
	}

	costMetricMap := make(map[string]inventoryApi.CostMetric)
	if filterStr != "" && len(connectionIDs) == 0 {
		return ctx.JSON(http.StatusOK, inventoryApi.ListCostMetricsResponse{
			TotalCount: 0,
			TotalCost:  0,
			Metrics:    []inventoryApi.CostMetric{},
		})
	} else if len(connectionIDs) > 0 {
		hits, err := es.FetchConnectionDailySpendHistoryByMetric(h.client, connectionIDs, connectorTypes, metricIds, startTime, endTime, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			localHit := hit
			if v, ok := costMetricMap[localHit.MetricID]; ok {
				exists := false
				for _, cnt := range v.Connector {
					if cnt.String() == localHit.Connector.String() {
						exists = true
						break
					}
				}
				if !exists {
					v.Connector = append(v.Connector, localHit.Connector)
				}
				v.TotalCost = utils.PAdd(v.TotalCost, &localHit.TotalCost)
				v.DailyCostAtStartTime = utils.PAdd(v.DailyCostAtStartTime, &localHit.StartDateCost)
				v.DailyCostAtEndTime = utils.PAdd(v.DailyCostAtEndTime, &localHit.EndDateCost)
				costMetricMap[localHit.MetricID] = v
			} else {
				costMetricMap[localHit.MetricID] = inventoryApi.CostMetric{
					Connector:                []source.Type{localHit.Connector},
					CostDimensionName:        localHit.MetricName,
					CostDimensionID:          localHit.MetricID,
					TotalCost:                &localHit.TotalCost,
					DailyCostAtStartTime:     &localHit.StartDateCost,
					DailyCostAtEndTime:       &localHit.EndDateCost,
					FinderQuery:              metricsMap[localHit.MetricID].FinderQuery,
					FinderPerConnectionQuery: metricsMap[localHit.MetricID].FinderPerConnectionQuery,
				}
			}
		}
	} else {
		hits, err := es.FetchConnectorDailySpendHistoryByMetric(h.client, connectorTypes, metricIds, startTime, endTime, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			localHit := hit
			connector, _ := source.ParseType(localHit.Connector)
			if v, ok := costMetricMap[localHit.MetricID]; ok {
				exists := false
				for _, cnt := range v.Connector {
					if cnt.String() == connector.String() {
						exists = true
						break
					}
				}
				if !exists {
					v.Connector = append(v.Connector, connector)
				}
				v.TotalCost = utils.PAdd(v.TotalCost, &localHit.TotalCost)
				v.DailyCostAtStartTime = utils.PAdd(v.DailyCostAtStartTime, &localHit.StartDateCost)
				v.DailyCostAtEndTime = utils.PAdd(v.DailyCostAtEndTime, &localHit.EndDateCost)
				costMetricMap[localHit.MetricID] = v
			} else {
				costMetricMap[localHit.MetricID] = inventoryApi.CostMetric{
					Connector:                []source.Type{connector},
					CostDimensionName:        localHit.MetricName,
					CostDimensionID:          localHit.MetricID,
					TotalCost:                &localHit.TotalCost,
					DailyCostAtStartTime:     &localHit.StartDateCost,
					DailyCostAtEndTime:       &localHit.EndDateCost,
					FinderQuery:              metricsMap[localHit.MetricID].FinderQuery,
					FinderPerConnectionQuery: metricsMap[localHit.MetricID].FinderPerConnectionQuery,
				}
			}
		}
	}

	var costMetrics []inventoryApi.CostMetric
	totalCost := float64(0)
	for _, costMetric := range costMetricMap {
		costMetrics = append(costMetrics, costMetric)
		if costMetric.TotalCost != nil {
			totalCost += *costMetric.TotalCost
		}
	}

	sort.Slice(costMetrics, func(i, j int) bool {
		switch sortBy {
		case "dimension":
			return costMetrics[i].CostDimensionName < costMetrics[j].CostDimensionName
		case "cost":
			if costMetrics[i].TotalCost == nil && costMetrics[j].TotalCost == nil {
				break
			}
			if costMetrics[i].TotalCost == nil {
				return false
			}
			if costMetrics[j].TotalCost == nil {
				return true
			}
			if *costMetrics[i].TotalCost != *costMetrics[j].TotalCost {
				return *costMetrics[i].TotalCost > *costMetrics[j].TotalCost
			}
		case "growth":
			diffi := utils.PSub(costMetrics[i].DailyCostAtEndTime, costMetrics[i].DailyCostAtStartTime)
			diffj := utils.PSub(costMetrics[j].DailyCostAtEndTime, costMetrics[j].DailyCostAtStartTime)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if *diffi != *diffj {
				return math.Abs(*diffi) > math.Abs(*diffj)
			}
		case "growth_rate":
			diffi := utils.PSub(costMetrics[i].DailyCostAtEndTime, costMetrics[i].DailyCostAtStartTime)
			diffj := utils.PSub(costMetrics[j].DailyCostAtEndTime, costMetrics[j].DailyCostAtStartTime)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if costMetrics[i].DailyCostAtStartTime == nil && costMetrics[j].DailyCostAtStartTime == nil {
				break
			}
			if costMetrics[i].DailyCostAtStartTime == nil {
				return true
			}
			if costMetrics[j].DailyCostAtStartTime == nil {
				return false
			}
			if *costMetrics[i].DailyCostAtStartTime == 0 && *costMetrics[j].DailyCostAtStartTime == 0 {
				break
			}
			if *costMetrics[i].DailyCostAtStartTime == 0 {
				return false
			}
			if *costMetrics[j].DailyCostAtStartTime == 0 {
				return true
			}
			if *diffi/(*costMetrics[i].DailyCostAtStartTime) != *diffj/(*costMetrics[j].DailyCostAtStartTime) {
				return math.Abs(*diffi/(*costMetrics[i].DailyCostAtStartTime)) > math.Abs(*diffj/(*costMetrics[j].DailyCostAtStartTime))
			}
		}
		return costMetrics[i].CostDimensionName < costMetrics[j].CostDimensionName
	})

	return ctx.JSON(http.StatusOK, inventoryApi.ListCostMetricsResponse{
		TotalCount: len(costMetrics),
		TotalCost:  totalCost,
		Metrics:    utils.Paginate(pageNumber, pageSize, costMetrics),
	})
}

// CountAnalyticsSpend godoc
//
//	@Summary		Count analytics spend
//	@Description	Retrieving the count of resources and connections with respect to specified filters.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	inventoryApi.CountAnalyticsSpendResponse
//	@Router			/inventory/api/v2/analytics/spend/count [get]
func (h *HttpHandler) CountAnalyticsSpend(ctx echo.Context) error {
	counts, err := es.CountAnalyticsSpend(h.logger, h.client)
	if err != nil {
		h.logger.Error("failed to count analytics spend", zap.Error(err))
		return err
	}

	response := inventoryApi.CountAnalyticsSpendResponse{
		ConnectionCount: counts.Aggregations.ConnectionCount.Value,
		MetricCount:     counts.Aggregations.MetricCount.Value,
	}

	return ctx.JSON(http.StatusOK, response)
}

// CountAnalytics godoc
//
//	@Summary		Count analytics
//	@Description	Retrieving the count of resources and connections with respect to specified filters.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	inventoryApi.CountAnalyticsMetricsResponse
//	@Router			/inventory/api/v2/analytics/count [get]
func (h *HttpHandler) CountAnalytics(ctx echo.Context) error {
	counts, err := es.CountAnalyticsMetrics(h.logger, h.client)
	if err != nil {
		h.logger.Error("failed to count analytics metrics", zap.Error(err))
		return err
	}

	response := inventoryApi.CountAnalyticsMetricsResponse{
		ConnectionCount: counts.Aggregations.ConnectionCount.Value,
		MetricCount:     counts.Aggregations.MetricCount.Value,
	}

	return ctx.JSON(http.StatusOK, response)
}

// ListMetrics godoc
//
//	@Summary		List metrics
//	@Description	Returns list of metrics
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			connector	query		[]source.Type	false	"Connector type to filter by"
//	@Param			metricType	query		string			false	"Metric type, default: assets"	Enums(assets, spend)
//
//	@Success		200			{object}	[]inventoryApi.AnalyticsMetric
//	@Router			/inventory/api/v2/analytics/metrics/list [get]
func (h *HttpHandler) ListMetrics(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	var err error
	connectorTypes := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	metricType := analyticsDB.MetricType(ctx.QueryParam("metricType"))
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredMetrics")

	metrics, err := aDB.ListFilteredMetrics(nil, metricType,
		nil, connectorTypes, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	var apiMetrics []inventoryApi.AnalyticsMetric
	for _, metric := range metrics {
		apiMetric := inventoryApi.AnalyticsMetric{
			ID:                       metric.ID,
			Connectors:               source.ParseTypes(metric.Connectors),
			Type:                     metric.Type,
			Name:                     metric.Name,
			Query:                    metric.Query,
			Tables:                   metric.Tables,
			FinderQuery:              metric.FinderQuery,
			FinderPerConnectionQuery: metric.FinderPerConnectionQuery,
			Tags:                     metric.GetTagsMap(),
		}

		apiMetrics = append(apiMetrics, apiMetric)
	}
	return ctx.JSON(http.StatusOK, apiMetrics)
}

// GetMetric godoc
//
//	@Summary		List metrics
//	@Description	Returns list of metrics
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			metric_id	path		string	true	"MetricID"
//
//	@Success		200			{object}	inventoryApi.AnalyticsMetric
//	@Router			/inventory/api/v2/analytics/metrics/{metric_id} [get]
func (h *HttpHandler) GetMetric(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	var err error

	metricID := ctx.Param("metric_id")
	_, span := tracer.Start(ctx.Request().Context(), "new_GetMetric", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetMetric")

	metric, err := aDB.GetMetricByID(metricID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if metric == nil {
		return echo.NewHTTPError(http.StatusNotFound, "metric not found")
	}

	span.End()

	apiMetric := inventoryApi.AnalyticsMetric{
		ID:                       metric.ID,
		Connectors:               source.ParseTypes(metric.Connectors),
		Type:                     metric.Type,
		Name:                     metric.Name,
		Query:                    metric.Query,
		Tables:                   metric.Tables,
		FinderQuery:              metric.FinderQuery,
		FinderPerConnectionQuery: metric.FinderPerConnectionQuery,
		Tags:                     metric.GetTagsMap(),
	}
	return ctx.JSON(http.StatusOK, apiMetric)
}

// ListAnalyticsSpendComposition godoc
//
//	@Summary		List cost composition
//	@Description	Retrieving the cost composition with respect to specified filters. Retrieving information such as the total cost for the given time range, and the top services by cost.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			top				query		int				false	"How many top values to return default is 5"
//	@Param			startTime		query		int64			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		int64			false	"timestamp for end in epoch seconds"
//	@Success		200				{object}	inventoryApi.ListCostCompositionResponse
//	@Router			/inventory/api/v2/analytics/spend/composition [get]
func (h *HttpHandler) ListAnalyticsSpendComposition(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	var err error
	connectorTypes := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	topStr := ctx.QueryParam("top")
	top := int64(5)
	if topStr != "" {
		top, err = strconv.ParseInt(topStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid top value")
		}
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredMetrics")

	metrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeSpend,
		nil, nil, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	costMetricMap := make(map[string]inventoryApi.CostMetric)
	spends, err := es.FetchSpendByMetric(h.client, connectionIDs, connectorTypes, nil, startTime, endTime, EsFetchPageSize)
	if err != nil {
		return err
	}
	for metricID, spend := range spends {
		localSpend := spend

		var metric analyticsDB.AnalyticMetric
		for _, m := range metrics {
			if m.ID == metricID {
				metric = m
			}
		}

		categoryExists := false
		for _, tag := range metric.Tags {
			if tag.GetKey() == "category" {
				for _, value := range tag.GetValue() {
					categoryExists = true
					if v, ok := costMetricMap[value]; ok {
						v.TotalCost = utils.PAdd(v.TotalCost, &localSpend.CostValue)
						costMetricMap[value] = v
					} else {
						costMetricMap[value] = inventoryApi.CostMetric{
							CostDimensionName: value,
							TotalCost:         &localSpend.CostValue,
						}
					}
				}
			}
		}

		if !categoryExists {
			costMetricMap[metricID] = inventoryApi.CostMetric{
				CostDimensionName: localSpend.MetricName,
				TotalCost:         &localSpend.CostValue,
			}
		}
	}

	var costMetrics []inventoryApi.CostMetric
	totalCost := float64(0)
	for _, costMetric := range costMetricMap {
		costMetrics = append(costMetrics, costMetric)
		if costMetric.TotalCost != nil {
			totalCost += *costMetric.TotalCost
		}
	}

	sort.Slice(costMetrics, func(i, j int) bool {
		if costMetrics[i].TotalCost == nil {
			return false
		}
		if costMetrics[j].TotalCost == nil {
			return true
		}
		if *costMetrics[i].TotalCost != *costMetrics[j].TotalCost {
			return *costMetrics[i].TotalCost > *costMetrics[j].TotalCost
		}
		return costMetrics[i].CostDimensionName < costMetrics[j].CostDimensionName
	})

	topCostMap := make(map[string]float64)
	othersCost := float64(0)
	if top > int64(len(costMetrics)) {
		top = int64(len(costMetrics))
	}
	for _, costMetric := range costMetrics[:int(top)] {
		if costMetric.TotalCost != nil {
			topCostMap[costMetric.CostDimensionName] = *costMetric.TotalCost
		}
	}
	if len(costMetrics) > int(top) {
		for _, costMetric := range costMetrics[int(top):] {
			if costMetric.TotalCost != nil {
				othersCost += *costMetric.TotalCost
			}
		}
	}

	return ctx.JSON(http.StatusOK, inventoryApi.ListCostCompositionResponse{
		TotalCount:     len(costMetrics),
		TotalCostValue: totalCost,
		TopValues:      topCostMap,
		Others:         othersCost,
	})
}

// GetAnalyticsSpendTrend godoc
//
//	@Summary		Get Cost Trend
//	@Description	Retrieving a list of costs over the course of the specified time frame based on the given input filters. If startTime and endTime are empty, the API returns the last month trend.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string		false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			metricIds		query		[]string		false	"Metrics IDs"
//	@Param			startTime		query		int64			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		int64			false	"timestamp for end in epoch seconds"
//	@Param			granularity		query		string			false	"Granularity of the table, default is daily"	Enums(monthly, daily, yearly)
//	@Success		200				{object}	[]inventoryApi.CostTrendDatapoint
//	@Router			/inventory/api/v2/analytics/spend/trend [get]
func (h *HttpHandler) GetAnalyticsSpendTrend(ctx echo.Context) error {
	var err error
	metricIds := httpserver2.QueryArrayParam(ctx, "metricIds")
	connectorTypes := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))

	aDB := analyticsDB.NewDatabase(h.db.orm)
	metrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeSpend,
		metricIds, connectorTypes, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		return err
	}
	metricIds = nil
	for _, m := range metrics {
		metricIds = append(metricIds, m.ID)
	}

	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	granularity := inventoryApi.TableGranularityType(ctx.QueryParam("granularity"))
	if granularity == "" {
		granularity = inventoryApi.TableGranularityTypeDaily
	}
	if granularity != inventoryApi.TableGranularityTypeDaily &&
		granularity != inventoryApi.TableGranularityTypeMonthly &&
		granularity != inventoryApi.TableGranularityTypeYearly {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid granularity")
	}

	timepointToCost := map[string]es.DatapointWithFailures{}
	if len(connectionIDs) > 0 {
		timepointToCost, err = es.FetchConnectionSpendTrend(h.client, granularity, metricIds, connectionIDs, connectorTypes, startTime, endTime)
	} else {
		timepointToCost, err = es.FetchConnectorSpendTrend(h.client, granularity, metricIds, connectorTypes, startTime, endTime)
	}
	if err != nil {
		return err
	}

	apiDatapoints := make([]inventoryApi.CostTrendDatapoint, 0, len(timepointToCost))
	for timeAt, costVal := range timepointToCost {
		format := "2006-01-02"
		if granularity == inventoryApi.TableGranularityTypeMonthly {
			format = "2006-01"
		} else if granularity == inventoryApi.TableGranularityTypeYearly {
			format = "2006"
		}
		dt, _ := time.Parse(format, timeAt)
		var cost []inventoryApi.CostStackedItem
		for k, v := range costVal.CostStacked {
			metricName := ""
			for _, v := range metrics {
				if v.ID == k {
					metricName = v.Name
				}
			}
			cost = append(cost, inventoryApi.CostStackedItem{
				MetricID:   k,
				MetricName: metricName,
				Cost:       v,
			})
		}

		apiDatapoints = append(apiDatapoints, inventoryApi.CostTrendDatapoint{
			Cost:                                    costVal.Cost,
			CostStacked:                             cost,
			TotalDescribedConnectionCount:           costVal.TotalConnections,
			TotalSuccessfulDescribedConnectionCount: costVal.TotalSuccessfulConnections,
			Date:                                    dt,
		})
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})

	return ctx.JSON(http.StatusOK, apiDatapoints)
}

// GetSpendTable godoc
//
//	@Summary		Get Spend Trend
//	@Description	Returns spend table with respect to the dimension and granularity
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			startTime		query		int64		false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		int64		false	"timestamp for end in epoch seconds"
//	@Param			granularity		query		string		false	"Granularity of the table, default is daily"	Enums(monthly, daily, yearly)
//	@Param			dimension		query		string		false	"Dimension of the table, default is metric"		Enums(connection, metric)
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by - mutually exclusive with connectionGroup"
//	@Param			connectionGroup	query		[]string	false	"Connection group to filter by - mutually exclusive with connectionId"
//	@Param			connector		query		[]string	false	"Connector"
//	@Param			metricIds		query		[]string	false	"Metrics IDs"
//
//	@Success		200				{object}	[]inventoryApi.SpendTableRow
//	@Router			/inventory/api/v2/analytics/spend/table [get]
func (h *HttpHandler) GetSpendTable(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	var err error
	metricIds := httpserver2.QueryArrayParam(ctx, "metricIds")
	ms, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeSpend,
		metricIds, nil, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		return err
	}
	metricIds = nil
	for _, m := range ms {
		metricIds = append(metricIds, m.ID)
	}

	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	endTime, err := utils.TimeFromQueryParam(ctx, "endTime", time.Now())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime, err := utils.TimeFromQueryParam(ctx, "startTime", endTime.AddDate(0, -1, 0))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	granularity := inventoryApi.TableGranularityType(ctx.QueryParam("granularity"))
	if granularity == "" {
		granularity = inventoryApi.TableGranularityTypeDaily
	}
	if granularity != inventoryApi.TableGranularityTypeDaily &&
		granularity != inventoryApi.TableGranularityTypeMonthly &&
		granularity != inventoryApi.TableGranularityTypeYearly {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid granularity")
	}
	dimension := inventoryApi.DimensionType(ctx.QueryParam("dimension"))
	if dimension == "" {
		dimension = inventoryApi.DimensionTypeMetric
	}
	if dimension != inventoryApi.DimensionTypeMetric &&
		dimension != inventoryApi.DimensionTypeConnection {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid dimension")
	}

	connectionAccountIDMap := map[string]string{}
	var metrics []analyticsDB.AnalyticMetric

	if dimension == inventoryApi.DimensionTypeMetric {
		// trace :
		_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredMetrics", trace.WithSpanKind(trace.SpanKindServer))
		span.SetName("new_ListFilteredMetrics")

		metrics, err = aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeSpend,
			metricIds, nil, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.End()
	}

	mt, err := es.FetchSpendTableByDimension(h.client, dimension, connectionIDs, connectors, metricIds, startTime, endTime)
	if err != nil {
		return err
	}

	fmt.Println("FetchSpendTableByDimension res = ", len(mt))
	var table []inventoryApi.SpendTableRow
	for _, m := range mt {
		costValue := map[string]float64{}
		for dateKey, costItem := range m.Trend {
			dt, _ := time.Parse("2006-01-02", dateKey)
			monthKey := dt.Format("2006-01")
			yearKey := dt.Format("2006")
			if granularity == "daily" {
				costValue[dateKey] = costItem
			} else if granularity == "monthly" {
				if v, ok := costValue[monthKey]; ok {
					costValue[monthKey] = v + costItem
				} else {
					costValue[monthKey] = costItem
				}
			} else if granularity == "yearly" {
				if v, ok := costValue[yearKey]; ok {
					costValue[yearKey] = v + costItem
				} else {
					costValue[yearKey] = costItem
				}
			}
		}

		var category, accountID string
		dimensionName := m.DimensionName
		if dimension == inventoryApi.DimensionTypeMetric {
			for _, metric := range metrics {
				if m.DimensionID == metric.ID {
					for _, tag := range metric.Tags {
						if tag.GetKey() == "category" {
							for _, v := range tag.GetValue() {
								category = v
								break
							}
							break
						}
					}
					break
				}
			}
		} else if dimension == inventoryApi.DimensionTypeConnection {
			if v, ok := connectionAccountIDMap[m.DimensionID]; ok {
				accountID = demo.EncodeResponseData(ctx, v)
			} else {
				src, err := h.onboardClient.GetSource(&httpclient.Context{UserRole: authApi.InternalRole}, m.DimensionID)
				if err != nil {
					if !strings.Contains(err.Error(), "source not found") {
						return err
					}
					h.logger.Error("source not found", zap.String("connection_id", m.DimensionID))
				} else {
					accountID = demo.EncodeResponseData(ctx, src.ConnectionID)
				}
				connectionAccountIDMap[m.DimensionID] = accountID
			}
			dimensionName = demo.EncodeResponseData(ctx, dimensionName)
		}

		table = append(table, inventoryApi.SpendTableRow{
			DimensionID:   m.DimensionID,
			AccountID:     accountID,
			Connector:     m.Connector,
			Category:      category,
			DimensionName: dimensionName,
			CostValue:     costValue,
		})
	}
	return ctx.JSON(http.StatusOK, table)
}

func (h *HttpHandler) ListConnectionsData(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)
	performanceStartTime := time.Now()
	var err error
	connectionIDs := httpserver2.QueryArrayParam(ctx, "connectionId")
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
	connectors, err := h.getConnectorTypesFromConnectionIDs(ctx, nil, connectionIDs)
	if err != nil {
		return err
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
	needCostStr := ctx.QueryParam("needCost")
	needCost := true
	if needCostStr == "false" {
		needCost = false
	}
	needResourceCountStr := ctx.QueryParam("needResourceCount")
	needResourceCount := true
	if needResourceCountStr == "false" {
		needResourceCount = false
	}

	fmt.Println("ListConnectionsData part1 ", time.Now().Sub(performanceStartTime).Milliseconds())
	res := map[string]inventoryApi.ConnectionData{}
	if needResourceCount {
		metrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeAssets,
			nil, connectors, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
		if err != nil {
			return err
		}
		var metricIDs []string
		for _, m := range metrics {
			metricIDs = append(metricIDs, m.ID)
		}

		resourceCountsMap, err := es.FetchConnectionAnalyticsResourcesCountAtTime(h.logger, h.client, connectors, connectionIDs, resourceCollections, metricIDs, endTime, EsFetchPageSize)
		if err != nil {
			return err
		}
		for connectionId, resourceCountAndEvaluated := range resourceCountsMap {
			if _, ok := res[connectionId]; !ok {
				res[connectionId] = inventoryApi.ConnectionData{
					ConnectionID: connectionId,
				}
			}
			v := res[connectionId]
			localCount := resourceCountAndEvaluated
			v.Count = utils.PAdd(v.Count, &localCount.ResourceCountsSum)
			if v.LastInventory == nil || v.LastInventory.IsZero() || v.LastInventory.Before(time.UnixMilli(localCount.LatestEvaluatedAt)) {
				v.LastInventory = utils.GetPointer(time.UnixMilli(localCount.LatestEvaluatedAt))
			}
			res[connectionId] = v
		}
		fmt.Println("ListConnectionsData part2 ", time.Now().Sub(performanceStartTime).Milliseconds())
		oldResourceCount, err := es.FetchConnectionAnalyticsResourcesCountAtTime(h.logger, h.client, connectors, connectionIDs, resourceCollections, metricIDs, startTime, EsFetchPageSize)
		if err != nil {
			return err
		}
		for connectionId, resourceCountAndEvaluated := range oldResourceCount {
			if _, ok := res[connectionId]; !ok {
				res[connectionId] = inventoryApi.ConnectionData{
					ConnectionID:  connectionId,
					LastInventory: nil,
				}
			}
			v := res[connectionId]
			localCount := resourceCountAndEvaluated
			v.OldCount = utils.PAdd(v.OldCount, &localCount.ResourceCountsSum)
			if v.LastInventory == nil || v.LastInventory.IsZero() || v.LastInventory.Before(time.UnixMilli(localCount.LatestEvaluatedAt)) {
				v.LastInventory = utils.GetPointer(time.UnixMilli(localCount.LatestEvaluatedAt))
			}
			res[connectionId] = v
		}
		fmt.Println("ListConnectionsData part3 ", time.Now().Sub(performanceStartTime).Milliseconds())
	}

	if needCost {
		hits, err := es.FetchConnectionDailySpendHistory(h.client, connectionIDs, connectors, nil, startTime, endTime, EsFetchPageSize)
		if err != nil {
			return err
		}
		for _, hit := range hits {
			localHit := hit
			if v, ok := res[localHit.ConnectionID]; ok {
				v.TotalCost = utils.PAdd(v.TotalCost, &localHit.TotalCost)
				v.DailyCostAtStartTime = utils.PAdd(v.DailyCostAtStartTime, &localHit.StartDateCost)
				v.DailyCostAtEndTime = utils.PAdd(v.DailyCostAtEndTime, &localHit.EndDateCost)
				res[localHit.ConnectionID] = v
			} else {
				res[localHit.ConnectionID] = inventoryApi.ConnectionData{
					ConnectionID:         localHit.ConnectionID,
					Count:                nil,
					OldCount:             nil,
					LastInventory:        nil,
					TotalCost:            &localHit.TotalCost,
					DailyCostAtStartTime: &localHit.StartDateCost,
					DailyCostAtEndTime:   &localHit.EndDateCost,
				}
			}
		}
		fmt.Println("ListConnectionsData part4 ", time.Now().Sub(performanceStartTime).Milliseconds())
	}

	return ctx.JSON(http.StatusOK, res)
}

// ListQueries godoc
//
//	@Summary		List smart queries
//	@Description	Retrieving list of smart queries by specified filters
//	@Security		BearerToken
//	@Tags			smart_query
//	@Produce		json
//	@Param			request	body		inventoryApi.ListQueryRequest	true	"Request Body"
//	@Success		200		{object}	[]inventoryApi.SmartQueryItem
//	@Router			/inventory/api/v1/query [get]
func (h *HttpHandler) ListQueries(ctx echo.Context) error {
	var req inventoryApi.ListQueryRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var search *string
	if len(req.TitleFilter) > 0 {
		search = &req.TitleFilter
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_GetQueriesWithFilters", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetQueriesWithFilters")

	queries, err := h.db.GetQueriesWithFilters(search)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	var result []inventoryApi.SmartQueryItem
	for _, item := range queries {
		category := ""

		tags := map[string]string{}
		if item.IsPopular {
			tags["popular"] = "true"
		}
		result = append(result, inventoryApi.SmartQueryItem{
			ID:         item.ID,
			Connectors: source.ParseTypes(item.Connectors),
			Title:      item.Title,
			Category:   category,
			Query:      item.Query,
			Tags:       tags,
		})
	}
	return ctx.JSON(200, result)
}

// RunQuery godoc
//
//	@Summary		Run query
//	@Description	Run provided smart query and returns the result.
//	@Security		BearerToken
//	@Tags			smart_query
//	@Accepts		json
//	@Produce		json
//	@Param			request	body		inventoryApi.RunQueryRequest	true	"Request Body"
//	@Param			accept	header		string							true	"Accept header"	Enums(application/json,text/csv)
//	@Success		200		{object}	inventoryApi.RunQueryResponse
//	@Router			/inventory/api/v1/query/run [post]
func (h *HttpHandler) RunQuery(ctx echo.Context) error {
	var req inventoryApi.RunQueryRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.Query == nil || *req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Query is required")
	}
	// tracer :
	outputS, span := tracer.Start(ctx.Request().Context(), "new_RunSmartQuery", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_RunSmartQuery")

	resp, err := h.RunSmartQuery(outputS, *req.Query, *req.Query, &req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.AddEvent("information", trace.WithAttributes(
		attribute.String("query title ", resp.Title),
	))
	span.End()
	return ctx.JSON(200, resp)
}

// GetRecentRanQueries godoc
//
//	@Summary		List recently ran queries
//	@Description	List queries which have been run recently
//	@Security		BearerToken
//	@Tags			smart_query
//	@Accepts		json
//	@Produce		json
//	@Success		200	{object}	[]inventoryApi.SmartQueryHistory
//	@Router			/inventory/api/v1/query/run/history [get]
func (h *HttpHandler) GetRecentRanQueries(ctx echo.Context) error {
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_GetQueryHistory", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_GetQueryHistory")

	smartQueryHistories, err := h.db.GetQueryHistory()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.logger.Error("Failed to get query history", zap.Error(err))
		return err
	}
	span.End()

	res := make([]inventoryApi.SmartQueryHistory, 0, len(smartQueryHistories))
	for _, history := range smartQueryHistories {
		res = append(res, history.ToApi())
	}

	return ctx.JSON(200, res)
}

func (h *HttpHandler) CountResources(ctx echo.Context) error {
	metricsIndexed, err := es.FetchConnectorAnalyticMetricCountAtTime(h.logger, h.client, nil, nil, nil, time.Now(), EsFetchPageSize)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	totalCount := 0
	for _, countWithTime := range metricsIndexed {
		totalCount += countWithTime.Count
	}
	return ctx.JSON(http.StatusOK, totalCount)
}

func (h *HttpHandler) RunSmartQuery(ctx context.Context, title, query string, req *inventoryApi.RunQueryRequest) (*inventoryApi.RunQueryResponse, error) {
	var err error
	lastIdx := (req.Page.No - 1) * req.Page.Size

	direction := inventoryApi.DirectionType("")
	orderBy := ""
	if req.Sorts != nil && len(req.Sorts) > 0 {
		direction = req.Sorts[0].Direction
		orderBy = req.Sorts[0].Field
	}
	if len(req.Sorts) > 1 {
		return nil, errors.New("multiple sort items not supported")
	}

	h.logger.Info("executing smart query", zap.String("query", query))
	res, err := h.steampipeConn.Query(ctx, query, &lastIdx, &req.Page.Size, orderBy, steampipe.DirectionType(direction))
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// tracer :
	connections, err := h.onboardClient.ListSources(&httpclient.Context{UserRole: authApi.InternalRole}, nil)
	if err != nil {
		return nil, err
	}
	connectionToNameMap := make(map[string]string)
	for _, connection := range connections {
		connectionToNameMap[connection.ID.String()] = connection.ConnectionName
	}

	accountIDExists := false
	for _, header := range res.Headers {
		if header == "kaytu_account_id" {
			accountIDExists = true
		}
	}

	if accountIDExists {
		// Add account name
		res.Headers = append(res.Headers, "account_name")
		for colIdx, header := range res.Headers {
			if strings.ToLower(header) != "kaytu_account_id" {
				continue
			}
			for rowIdx, row := range res.Data {
				if len(row) <= colIdx {
					continue
				}
				if row[colIdx] == nil {
					continue
				}
				if accountID, ok := row[colIdx].(string); ok {
					if accountName, ok := connectionToNameMap[accountID]; ok {
						res.Data[rowIdx] = append(res.Data[rowIdx], accountName)
					} else {
						res.Data[rowIdx] = append(res.Data[rowIdx], "null")
					}
				}
			}
		}
	}

	_, span := tracer.Start(ctx, "new_UpdateQueryHistory", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_UpdateQueryHistory")

	err = h.db.UpdateQueryHistory(query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.logger.Error("failed to update query history", zap.Error(err))
		return nil, err
	}
	span.End()

	resp := inventoryApi.RunQueryResponse{
		Title:   title,
		Query:   query,
		Headers: res.Headers,
		Result:  res.Data,
	}
	return &resp, nil
}

func (h *HttpHandler) ListInsightResults(ctx echo.Context) error {
	var err error
	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	timeStr := ctx.QueryParam("time")
	timeAt := time.Now().Unix()
	if timeStr != "" {
		timeAt, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}
	connectionIDs := httpserver2.QueryArrayParam(ctx, "connectionId")

	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")

	insightIdListStr := httpserver2.QueryArrayParam(ctx, "insightId")
	if len(insightIdListStr) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "insight id is required")
	}
	insightIdList := make([]uint, 0, len(insightIdListStr))
	for _, idStr := range insightIdListStr {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid insight id")
		}
		insightIdList = append(insightIdList, uint(id))
	}

	var insightValues map[uint][]insight.InsightResource
	if timeStr != "" {
		insightValues, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), connectors, connectionIDs, resourceCollections, insightIdList, true)
	} else {
		insightValues, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), connectors, connectionIDs, resourceCollections, insightIdList, false)
	}
	if err != nil {
		return err
	}

	firstAvailable, err := es.FetchInsightValueAfter(h.client, time.Unix(timeAt, 0), connectors, connectionIDs, resourceCollections, insightIdList)
	if err != nil {
		return err
	}

	for insightId, _ := range firstAvailable {
		if results, ok := insightValues[insightId]; ok && len(results) > 0 {
			continue
		}
		insightValues[insightId] = firstAvailable[insightId]
	}

	return ctx.JSON(http.StatusOK, insightValues)
}

func (h *HttpHandler) GetInsightResult(ctx echo.Context) error {
	insightId, err := strconv.ParseUint(ctx.Param("insightId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid insight id")
	}
	timeStr := ctx.QueryParam("time")
	timeAt := time.Now().Unix()
	if timeStr != "" {
		timeAt, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}
	connectionIDs := httpserver2.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) == 0 {
		connectionIDs = nil
	}

	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")

	var insightResults map[uint][]insight.InsightResource
	if timeStr != "" {
		insightResults, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), nil, connectionIDs, resourceCollections, []uint{uint(insightId)}, true)
	} else {
		insightResults, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), nil, connectionIDs, resourceCollections, []uint{uint(insightId)}, false)
	}
	if err != nil {
		return err
	}

	firstAvailable, err := es.FetchInsightValueAfter(h.client, time.Unix(timeAt, 0), nil, connectionIDs, resourceCollections, []uint{uint(insightId)})
	if err != nil {
		return err
	}

	for insightId, _ := range firstAvailable {
		if results, ok := insightResults[insightId]; ok && len(results) > 0 {
			continue
		}
		insightResults[insightId] = firstAvailable[insightId]
	}

	if insightResult, ok := insightResults[uint(insightId)]; ok {
		return ctx.JSON(http.StatusOK, insightResult)
	} else {
		return echo.NewHTTPError(http.StatusNotFound, "no data for insight found")
	}
}

func (h *HttpHandler) GetInsightTrendResults(ctx echo.Context) error {
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

	connectionIDs := httpserver2.QueryArrayParam(ctx, "connectionId")
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")

	dataPointCount := int(endTime.Sub(startTime).Hours() / 24)
	insightResults, err := es.FetchInsightAggregatedPerQueryValuesBetweenTimes(h.client, startTime, endTime, dataPointCount, nil, connectionIDs, resourceCollections, []uint{uint(insightId)})
	if err != nil {
		return err
	}
	if insightResult, ok := insightResults[uint(insightId)]; ok {
		return ctx.JSON(http.StatusOK, insightResult)
	} else {
		return echo.NewHTTPError(http.StatusNotFound, "no data for insight found")
	}
}

func (h *HttpHandler) ListResourceTypeMetadata(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver2.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	serviceNames := httpserver2.QueryArrayParam(ctx, "service")
	resourceTypeNames := httpserver2.QueryArrayParam(ctx, "resourceType")
	summarized := strings.ToLower(ctx.QueryParam("summarized")) == "true"
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	// trace :
	_, span := tracer.Start(ctx.Request().Context(), "new_ListFilteredResourceTypes", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListFilteredResourceTypes")

	resourceTypes, err := h.db.ListFilteredResourceTypes(tagMap, resourceTypeNames, serviceNames, connectors, summarized)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.End()

	var resourceTypeMetadata []inventoryApi.ResourceType
	tableCountMap := make(map[string]int)
	insightList, err := h.complianceClient.ListInsightsMetadata(httpclient.FromEchoContext(ctx), connectors)
	if err != nil {
		return err
	}
	for _, insightEntity := range insightList {
		for _, insightTable := range insightEntity.Query.ListOfTables {
			tableCountMap[insightTable]++
		}
	}

	for _, resourceType := range resourceTypes {
		apiResourceType := resourceType.ToApi()

		var table string
		switch resourceType.Connector {
		case source.CloudAWS:
			table = awsSteampipe.ExtractTableName(resourceType.ResourceType)
		case source.CloudAzure:
			table = azureSteampipe.ExtractTableName(resourceType.ResourceType)
		}
		insightTableCount := 0
		if table != "" {
			insightTableCount = tableCountMap[table]
		}
		apiResourceType.InsightsCount = utils.GetPointerOrNil(insightTableCount)

		// TODO: add compliance count

		resourceTypeMetadata = append(resourceTypeMetadata, apiResourceType)
	}

	sort.Slice(resourceTypeMetadata, func(i, j int) bool {
		return resourceTypeMetadata[i].ResourceType < resourceTypeMetadata[j].ResourceType
	})

	result := inventoryApi.ListResourceTypeMetadataResponse{
		TotalResourceTypeCount: len(resourceTypeMetadata),
		ResourceTypes:          utils.Paginate(pageNumber, pageSize, resourceTypeMetadata),
	}

	return ctx.JSON(http.StatusOK, result)
}

// ListResourceCollections godoc
//
//	@Summary		List resource collections with inventory data
//	@Description	Retrieving list of resource collections by specified filters with inventory data
//	@Security		BearerToken
//	@Tags			resource_collection
//	@Produce		json
//	@Param			id		query		[]string								false	"Resource collection IDs"
//	@Param			status	query		[]inventoryApi.ResourceCollectionStatus	false	"Resource collection status"
//	@Success		200		{object}	[]inventoryApi.ResourceCollection
//	@Router			/inventory/api/v2/resource-collection [get]
func (h *HttpHandler) ListResourceCollections(ctx echo.Context) error {
	ids := httpserver2.QueryArrayParam(ctx, "id")

	statuesString := httpserver2.QueryArrayParam(ctx, "status")
	var statuses []ResourceCollectionStatus
	for _, statusString := range statuesString {
		statuses = append(statuses, ResourceCollectionStatus(statusString))
	}

	resourceCollections, err := h.db.ListResourceCollections(ids, statuses)
	if err != nil {
		h.logger.Error("failed to list resource collections", zap.Error(err))
		return err
	}

	res := make(map[string]inventoryApi.ResourceCollection)
	collectionIds := make([]string, 0, len(resourceCollections))
	for _, collection := range resourceCollections {
		res[collection.ID] = collection.ToApi()
		collectionIds = append(collectionIds, collection.ID)
	}

	aDB := analyticsDB.NewDatabase(h.db.orm)
	filteredMetrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeAssets,
		nil, nil, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		h.logger.Error("failed to list filtered metrics", zap.Error(err))
		return err
	}
	filteredMetricIds := make([]string, 0, len(filteredMetrics))
	filteredMetricMap := make(map[string]analyticsDB.AnalyticMetric)
	for _, metric := range filteredMetrics {
		filteredMetricIds = append(filteredMetricIds, metric.ID)
		filteredMetricMap[metric.ID] = metric
	}

	perRcMetricResult, err := es.FetchPerResourceCollectionConnectorAnalyticMetricCountAtTime(h.logger, h.client,
		filteredMetricIds, nil, collectionIds, time.Now(), EsFetchPageSize)
	if err != nil {
		h.logger.Error("failed to fetch per resource collection metric count", zap.Error(err))
		return err
	}

	for collectionId, metricCount := range perRcMetricResult {
		if _, ok := res[collectionId]; !ok {
			continue
		}
		v := res[collectionId]
		for metricId, countWithTime := range metricCount {
			if countWithTime.Count == 0 {
				continue
			}
			countWithTime := countWithTime

			metric := filteredMetricMap[metricId]
			for _, connector := range metric.Connectors {
				found := false
				for _, c := range v.Connectors {
					if c.String() == connector {
						found = true
						break
					}
				}
				if !found {
					v.Connectors = append(v.Connectors, source.Type(connector))
				}
			}
			v.ResourceCount = utils.PAdd(v.ResourceCount, &countWithTime.Count)
			if v.LastEvaluatedAt == nil || v.LastEvaluatedAt.IsZero() || v.LastEvaluatedAt.Before(countWithTime.Time) {
				v.LastEvaluatedAt = &countWithTime.Time
			}
		}
		res[collectionId] = v
	}

	resArray := make([]inventoryApi.ResourceCollection, 0, len(res))
	for _, collection := range res {
		resArray = append(resArray, collection)
	}

	return ctx.JSON(http.StatusOK, resArray)
}

// GetResourceCollection godoc
//
//	@Summary		Get resource collection with inventory data
//	@Description	Retrieving resource collection by specified ID with inventory data
//	@Security		BearerToken
//	@Tags			resource_collection
//	@Produce		json
//	@Param			resourceCollectionId	path		string	true	"Resource collection ID"
//	@Success		200						{object}	inventoryApi.ResourceCollection
//	@Router			/inventory/api/v2/resource-collection/{resourceCollectionId} [get]
func (h *HttpHandler) GetResourceCollection(ctx echo.Context) error {
	collectionID := ctx.Param("resourceCollectionId")
	resourceCollection, err := h.db.GetResourceCollection(collectionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "resource collection not found")
		}
		return err
	}

	aDB := analyticsDB.NewDatabase(h.db.orm)
	filteredMetrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeAssets,
		nil, nil, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		h.logger.Error("failed to list filtered metrics", zap.Error(err))
		return err
	}

	filteredMetricIds := make([]string, 0, len(filteredMetrics))
	filteredMetricMap := make(map[string]analyticsDB.AnalyticMetric)
	for _, metric := range filteredMetrics {
		filteredMetricIds = append(filteredMetricIds, metric.ID)
		filteredMetricMap[metric.ID] = metric
	}

	metricIndexed, err := es.FetchPerResourceCollectionConnectorAnalyticMetricCountAtTime(h.logger, h.client,
		filteredMetricIds, nil, []string{collectionID}, time.Now(), EsFetchPageSize)
	if err != nil {
		h.logger.Error("failed to fetch per resource collection metric count", zap.Error(err))
		return err
	}

	result := resourceCollection.ToApi()
	for metricId, count := range metricIndexed[collectionID] {
		if count.Count == 0 {
			continue
		}
		countWithTime := count

		metric := filteredMetricMap[metricId]
		for _, connector := range metric.Connectors {
			found := false
			for _, c := range result.Connectors {
				if c.String() == connector {
					found = true
					break
				}
			}
			if !found {
				result.Connectors = append(result.Connectors, source.Type(connector))
			}
		}
		result.ResourceCount = utils.PAdd(result.ResourceCount, &countWithTime.Count)
		result.MetricCount = utils.PAdd(result.MetricCount, utils.GetPointer(1))
		if result.LastEvaluatedAt == nil || result.LastEvaluatedAt.IsZero() || result.LastEvaluatedAt.Before(countWithTime.Time) {
			result.LastEvaluatedAt = &countWithTime.Time
		}
	}

	perConnectionMetric, err := es.FetchConnectionAnalyticsResourcesCountAtTime(h.logger, h.client, nil, nil,
		[]string{collectionID}, filteredMetricIds, time.Now(), EsFetchPageSize)
	if err != nil {
		h.logger.Error("failed to fetch per connection metric count", zap.Error(err))
		return err
	}

	for _, metricCount := range perConnectionMetric {
		if metricCount.ResourceCountsSum == 0 {
			continue
		}
		result.ConnectionCount = utils.PAdd(result.ConnectionCount, utils.GetPointer(1))
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetResourceCollectionLandscape godoc
//
//	@Summary		Get resource collection landscape
//	@Description	Retrieving resource collection landscape by specified ID
//	@Security		BearerToken
//	@Tags			resource_collection
//	@Produce		json
//	@Param			resourceCollectionId	path		string	true	"Resource collection ID"
//	@Success		200						{object}	inventoryApi.ResourceCollectionLandscape
//	@Router			/inventory/api/v2/resource-collection/{resourceCollectionId}/landscape [get]
func (h *HttpHandler) GetResourceCollectionLandscape(ctx echo.Context) error {
	resourceCollectionID := ctx.Param("resourceCollectionId")

	aDB := analyticsDB.NewDatabase(h.db.orm)
	metrics, err := aDB.ListFilteredMetrics(nil, analyticsDB.MetricTypeAssets,
		nil, nil, []analyticsDB.AnalyticMetricStatus{analyticsDB.AnalyticMetricStatusActive})
	if err != nil {
		return err
	}

	metricsMap := make(map[string]analyticsDB.AnalyticMetric)
	filteredMetricIDs := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		filteredMetricIDs = append(filteredMetricIDs, metric.ID)
		metricsMap[metric.ID] = metric
	}
	metricIndexed, err := es.FetchConnectorAnalyticMetricCountAtTime(h.logger, h.client, filteredMetricIDs, nil, []string{resourceCollectionID}, time.Now(), EsFetchPageSize)
	if err != nil {
		return err
	}

	includedResourceTypes := make(map[string]describe.ResourceType)
	for metricID, countWithTime := range metricIndexed {
		if countWithTime.Count == 0 {
			continue
		}
		metric := metricsMap[metricID]

		for _, table := range metric.Tables {
			if awsResourceType, err := kaytuAws.GetResourceType(table); err == nil && awsResourceType != nil {
				includedResourceTypes[awsResourceType.ResourceName] = awsResourceType
			} else if azureResourceType, err := kaytuAzure.GetResourceType(table); err == nil && azureResourceType != nil {
				includedResourceTypes[azureResourceType.ResourceName] = azureResourceType
			}
		}
	}

	var awsLandscapesSubcategories = make(map[string]inventoryApi.ResourceCollectionLandscapeSubcategory)
	var azureLandscapesSubcategories = make(map[string]inventoryApi.ResourceCollectionLandscapeSubcategory)
	for _, resourceType := range includedResourceTypes {
		category := "Other"
		if resourceType.GetTags() != nil && len(resourceType.GetTags()["category"]) > 0 {
			category = resourceType.GetTags()["category"][0]
		}
		item := inventoryApi.ResourceCollectionLandscapeItem{
			ID:          resourceType.GetResourceName(),
			Name:        resourceType.GetResourceLabel(),
			Description: "", //TODO
			LogoURI:     "", //TODO
		}
		if resourceType.GetTags() != nil && len(resourceType.GetTags()["logo_uri"]) > 0 {
			item.LogoURI = resourceType.GetTags()["logo_uri"][0]
		}
		switch resourceType.GetConnector() {
		case source.CloudAWS:
			subcategory, ok := awsLandscapesSubcategories[category]
			if !ok {
				subcategory = inventoryApi.ResourceCollectionLandscapeSubcategory{
					ID:          fmt.Sprintf("%s-%s", source.CloudAWS.String(), category),
					Name:        category,
					Description: "",
					Items:       nil,
				}
			}
			if item.LogoURI == "" {
				item.LogoURI = AWSLogoURI
			}
			subcategory.Items = append(subcategory.Items, item)
			awsLandscapesSubcategories[category] = subcategory
		case source.CloudAzure:
			subcategory, ok := azureLandscapesSubcategories[category]
			if !ok {
				subcategory = inventoryApi.ResourceCollectionLandscapeSubcategory{
					ID:          fmt.Sprintf("%s-%s", source.CloudAzure.String(), category),
					Name:        category,
					Description: "",
					Items:       nil,
				}
			}
			if item.LogoURI == "" {
				item.LogoURI = AzureLogoURI
			}
			subcategory.Items = append(subcategory.Items, item)
			azureLandscapesSubcategories[category] = subcategory
		}
	}

	var awsLandscapesCategory = inventoryApi.ResourceCollectionLandscapeCategory{
		ID:            source.CloudAWS.String(),
		Name:          "AWS",
		Description:   "AWS resources",
		Subcategories: nil,
	}
	for _, subcategory := range awsLandscapesSubcategories {
		awsLandscapesCategory.Subcategories = append(awsLandscapesCategory.Subcategories, subcategory)
	}
	var azureLandscapesCategory = inventoryApi.ResourceCollectionLandscapeCategory{
		ID:            source.CloudAzure.String(),
		Name:          "Azure",
		Description:   "Azure resources",
		Subcategories: nil,
	}
	for _, subcategory := range azureLandscapesSubcategories {
		azureLandscapesCategory.Subcategories = append(azureLandscapesCategory.Subcategories, subcategory)
	}

	landscape := inventoryApi.ResourceCollectionLandscape{
		Categories: make([]inventoryApi.ResourceCollectionLandscapeCategory, 0, 2),
	}
	if len(awsLandscapesCategory.Subcategories) > 0 {
		landscape.Categories = append(landscape.Categories, awsLandscapesCategory)
	}
	if len(azureLandscapesCategory.Subcategories) > 0 {
		landscape.Categories = append(landscape.Categories, azureLandscapesCategory)
	}

	return ctx.JSON(http.StatusOK, landscape)
}

// ListResourceCollectionsMetadata godoc
//
//	@Summary		List resource collections
//	@Description	Retrieving list of resource collections by specified filters
//	@Security		BearerToken
//	@Tags			resource_collection
//	@Produce		json
//	@Param			id		query		[]string								false	"Resource collection IDs"
//	@Param			status	query		[]inventoryApi.ResourceCollectionStatus	false	"Resource collection status"
//	@Success		200		{object}	[]inventoryApi.ResourceCollection
//	@Router			/inventory/api/v2/metadata/resource-collection [get]
func (h *HttpHandler) ListResourceCollectionsMetadata(ctx echo.Context) error {
	ids := httpserver2.QueryArrayParam(ctx, "id")

	statuesString := httpserver2.QueryArrayParam(ctx, "status")
	var statuses []ResourceCollectionStatus
	for _, statusString := range statuesString {
		statuses = append(statuses, ResourceCollectionStatus(statusString))
	}

	resourceCollections, err := h.db.ListResourceCollections(ids, nil)
	if err != nil {
		return err
	}

	res := make([]inventoryApi.ResourceCollection, 0, len(resourceCollections))
	for _, collection := range resourceCollections {
		res = append(res, collection.ToApi())
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetResourceCollectionMetadata godoc
//
//	@Summary		Get resource collection
//	@Description	Retrieving resource collection by specified ID
//	@Security		BearerToken
//	@Tags			resource_collection
//	@Produce		json
//	@Param			resourceCollectionId	path		string	true	"Resource collection ID"
//	@Success		200						{object}	inventoryApi.ResourceCollection
//	@Router			/inventory/api/v2/metadata/resource-collection/{resourceCollectionId} [get]
func (h *HttpHandler) GetResourceCollectionMetadata(ctx echo.Context) error {
	collectionID := ctx.Param("resourceCollectionId")
	resourceCollection, err := h.db.GetResourceCollection(collectionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "resource collection not found")
		}
		return err
	}
	return ctx.JSON(http.StatusOK, resourceCollection.ToApi())
}

func (h *HttpHandler) connectionsFilter(filter map[string]interface{}) ([]string, error) {
	var connections []string
	allConnections, err := h.onboardClient.ListSources(&httpclient.Context{UserRole: authApi.InternalRole}, []source.Type{source.CloudAWS, source.CloudAzure})
	if err != nil {
		return nil, err
	}
	var allConnectionsStr []string
	for _, c := range allConnections {
		allConnectionsStr = append(allConnectionsStr, c.ID.String())
	}
	for key, value := range filter {
		if key == "Match" {
			dimFilter := value.(map[string]interface{})
			if dimKey, ok := dimFilter["Key"]; ok {
				if dimKey == "ConnectionID" {
					connections, err = dimFilterFunction(dimFilter, allConnectionsStr)
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, connections))
				} else if dimKey == "Provider" {
					providers, err := dimFilterFunction(dimFilter, []string{"AWS", "Azure"})
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, providers))
					for _, c := range allConnections {
						if arrayContains(providers, c.Connector.String()) {
							connections = append(connections, c.ID.String())
						}
					}
				} else if dimKey == "ConnectionGroup" {
					allGroups, err := h.onboardClient.ListConnectionGroups(&httpclient.Context{UserRole: authApi.InternalRole})
					if err != nil {
						return nil, err
					}
					allGroupsMap := make(map[string][]string)
					var allGroupsStr []string
					for _, g := range allGroups {
						allGroupsMap[g.Name] = make([]string, 0, len(g.ConnectionIds))
						for _, cid := range g.ConnectionIds {
							allGroupsMap[g.Name] = append(allGroupsMap[g.Name], cid)
							allGroupsStr = append(allGroupsStr, cid)
						}
					}
					groups, err := dimFilterFunction(dimFilter, allGroupsStr)
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, groups))

					for _, g := range groups {
						for _, conn := range allGroupsMap[g] {
							if !arrayContains(connections, conn) {
								connections = append(connections, conn)
							}
						}
					}
				} else if dimKey == "ConnectionName" {
					var allConnectionsNames []string
					for _, c := range allConnections {
						allConnectionsNames = append(allConnectionsNames, c.ConnectionName)
					}
					connectionNames, err := dimFilterFunction(dimFilter, allConnectionsNames)
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, connectionNames))
					for _, conn := range allConnections {
						if arrayContains(connectionNames, conn.ConnectionName) {
							connections = append(connections, conn.ID.String())
						}
					}

				}
			} else {
				return nil, fmt.Errorf("missing key")
			}
		} else if key == "AND" {
			var andFilters []map[string]interface{}
			for _, v := range value.([]interface{}) {
				andFilter := v.(map[string]interface{})
				andFilters = append(andFilters, andFilter)
			}
			counter := make(map[string]int)
			for _, f := range andFilters {
				values, err := h.connectionsFilter(f)
				if err != nil {
					return nil, err
				}
				for _, v := range values {
					if c, ok := counter[v]; ok {
						counter[v] = c + 1
					} else {
						counter[v] = 1
					}
					if counter[v] == len(andFilters) {
						connections = append(connections, v)
					}
				}
			}
		} else if key == "OR" {
			var orFilters []map[string]interface{}
			for _, v := range value.([]interface{}) {
				orFilter := v.(map[string]interface{})
				orFilters = append(orFilters, orFilter)
			}
			for _, f := range orFilters {
				values, err := h.connectionsFilter(f)
				if err != nil {
					return nil, err
				}
				for _, v := range values {
					if !arrayContains(connections, v) {
						connections = append(connections, v)
					}
				}
			}
		} else {
			return nil, fmt.Errorf("invalid key: ", key)
		}
	}
	return connections, nil
}

func dimFilterFunction(dimFilter map[string]interface{}, allValues []string) ([]string, error) {
	var values []string
	for _, v := range dimFilter["Values"].([]interface{}) {
		values = append(values, fmt.Sprintf("%v", v))
	}
	var output []string
	if matchOption, ok := dimFilter["MatchOption"]; ok {
		switch {
		case strings.Contains(matchOption.(string), "EQUAL"):
			output = values
		case strings.Contains(matchOption.(string), "STARTS_WITH"):
			for _, v := range values {
				for _, conn := range allValues {
					if strings.HasPrefix(conn, v) {
						if !arrayContains(output, conn) {
							output = append(output, conn)
						}
					}
				}
			}
		case strings.Contains(matchOption.(string), "ENDS_WITH"):
			for _, v := range values {
				for _, conn := range allValues {
					if strings.HasSuffix(conn, v) {
						if !arrayContains(output, conn) {
							output = append(output, conn)
						}
					}
				}
			}
		case strings.Contains(matchOption.(string), "CONTAINS"):
			for _, v := range values {
				for _, conn := range allValues {
					if strings.Contains(conn, v) {
						if !arrayContains(output, conn) {
							output = append(output, conn)
						}
					}
				}
			}
		default:
			return nil, fmt.Errorf("invalid option")
		}
		if strings.HasPrefix(matchOption.(string), "~") {
			var notOutput []string
			for _, v := range allValues {
				if !arrayContains(output, v) {
					notOutput = append(notOutput, v)
				}
			}
			return notOutput, nil
		}
	} else {
		output = values
	}
	return output, nil
}

func arrayContains(array []string, key string) bool {
	for _, v := range array {
		if v == key {
			return true
		}
	}
	return false
}
