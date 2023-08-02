package inventory

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	es2 "github.com/kaytu-io/kaytu-engine/pkg/analytics/es"
	es3 "github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
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
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/es"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/internal"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const EsFetchPageSize = 10000

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	queryV1 := v1.Group("/query")
	queryV1.GET("", httpserver.AuthorizeHandler(h.ListQueries, authApi.ViewerRole))
	queryV1.POST("/run", httpserver.AuthorizeHandler(h.RunQuery, authApi.EditorRole))
	queryV1.GET("/run/history", httpserver.AuthorizeHandler(h.GetRecentRanQueries, authApi.EditorRole))

	v2 := e.Group("/api/v2")

	resourcesV2 := v2.Group("/resources")
	resourcesV2.GET("/tag", httpserver.AuthorizeHandler(h.ListResourceTypeTags, authApi.ViewerRole))
	resourcesV2.GET("/count", httpserver.AuthorizeHandler(h.CountResources, authApi.ViewerRole))
	resourcesV2.GET("/metric/:resourceType", httpserver.AuthorizeHandler(h.GetResourceTypeMetricsHandler, authApi.ViewerRole))

	analyticsV2 := v2.Group("/analytics")
	analyticsV2.GET("/metric", httpserver.AuthorizeHandler(h.ListAnalyticsMetricsHandler, authApi.ViewerRole))
	analyticsV2.GET("/tag", httpserver.AuthorizeHandler(h.ListAnalyticsTags, authApi.ViewerRole))
	analyticsV2.GET("/trend", httpserver.AuthorizeHandler(h.ListAnalyticsMetricTrend, authApi.ViewerRole))
	analyticsV2.GET("/composition/:key", httpserver.AuthorizeHandler(h.ListAnalyticsComposition, authApi.ViewerRole))
	analyticsV2.GET("/regions/summary", httpserver.AuthorizeHandler(h.ListAnalyticsRegionsSummary, authApi.ViewerRole))
	analyticsV2.GET("/categories", httpserver.AuthorizeHandler(h.ListAnalyticsCategories, authApi.ViewerRole))

	servicesV2 := v2.Group("/services")
	servicesV2.GET("/cost/trend", httpserver.AuthorizeHandler(h.GetServiceCostTrend, authApi.ViewerRole))

	costV2 := v2.Group("/cost")
	costV2.GET("/metric", httpserver.AuthorizeHandler(h.ListCostMetricsHandler, authApi.ViewerRole))
	costV2.GET("/composition", httpserver.AuthorizeHandler(h.ListCostComposition, authApi.ViewerRole))
	costV2.GET("/trend", httpserver.AuthorizeHandler(h.GetCostTrend, authApi.ViewerRole))

	connectionsV2 := v2.Group("/connections")
	connectionsV2.GET("/data", httpserver.AuthorizeHandler(h.ListConnectionsData, authApi.ViewerRole))
	connectionsV2.GET("/data/:connectionId", httpserver.AuthorizeHandler(h.GetConnectionData, authApi.ViewerRole))

	insightsV2 := v2.Group("/insights")
	insightsV2.GET("", httpserver.AuthorizeHandler(h.ListInsightResults, authApi.ViewerRole))
	insightsV2.GET("/:insightId/trend", httpserver.AuthorizeHandler(h.GetInsightTrendResults, authApi.ViewerRole))
	insightsV2.GET("/:insightId", httpserver.AuthorizeHandler(h.GetInsightResult, authApi.ViewerRole))

	metadata := v2.Group("/metadata")
	metadata.GET("/resourcetype", httpserver.AuthorizeHandler(h.ListResourceTypeMetadata, authApi.ViewerRole))

	v1.GET("/migrate-analytics", httpserver.AuthorizeHandler(h.MigrateAnalytics, authApi.AdminRole))
}

func (h *HttpHandler) MigrateAnalytics(ctx echo.Context) error {
	for i := 0; i < 1000; i++ {
		err := h.MigrateAnalyticsPart(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *HttpHandler) MigrateAnalyticsPart(summarizerJobID int) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)

	connectionMap := map[string]es2.ConnectionMetricTrendSummary{}
	connectorMap := map[string]es2.ConnectorMetricTrendSummary{}

	resourceTypeMetricIDCache := map[string]string{}

	cctx := context.Background()

	pagination, err := es.NewConnectionResourceTypePaginator(
		h.client,
		[]keibi.BoolFilter{
			keibi.NewTermFilter("report_type", string(es3.ResourceTypeTrendConnectionSummary)),
			keibi.NewTermFilter("summarize_job_id", fmt.Sprintf("%d", summarizerJobID)),
		},
		nil,
	)
	if err != nil {
		return err
	}

	var docs []kafka.Doc
	for {
		if !pagination.HasNext() {
			fmt.Println("MigrateAnalytics = page done", summarizerJobID)
			break
		}

		fmt.Println("MigrateAnalytics = ask page", summarizerJobID)
		page, err := pagination.NextPage(cctx)
		if err != nil {
			return err
		}
		fmt.Println("MigrateAnalytics = next page", summarizerJobID)

		for _, hit := range page {
			connectionID, err := uuid.Parse(hit.SourceID)
			if err != nil {
				return err
			}

			var metricID string

			if v, ok := resourceTypeMetricIDCache[hit.ResourceType]; ok {
				metricID = v
			} else {
				metric, err := aDB.GetMetric(hit.ResourceType)
				if err != nil {
					return err
				}

				if metric == nil {
					return fmt.Errorf("resource type %s not found", hit.ResourceType)
				}

				resourceTypeMetricIDCache[hit.ResourceType] = metric.ID
				metricID = metric.ID
			}

			connection := es2.ConnectionMetricTrendSummary{
				ConnectionID:  connectionID,
				Connector:     hit.SourceType,
				EvaluatedAt:   hit.DescribedAt,
				MetricID:      metricID,
				ResourceCount: hit.ResourceCount,
				ReportType:    es3.MetricTrendConnectionSummary,
			}
			key := fmt.Sprintf("%s-%s-%d", connectionID.String(), metricID, hit.DescribedAt)
			if v, ok := connectionMap[key]; ok {
				v.ResourceCount += connection.ResourceCount
				connectionMap[key] = v
			} else {
				connectionMap[key] = connection
			}

			connector := es2.ConnectorMetricTrendSummary{
				Connector:     hit.SourceType,
				EvaluatedAt:   hit.DescribedAt,
				MetricID:      metricID,
				ResourceCount: hit.ResourceCount,
				ReportType:    es3.MetricTrendConnectorSummary,
			}
			key = fmt.Sprintf("%s-%s-%d", connector.Connector, metricID, hit.DescribedAt)
			if v, ok := connectorMap[key]; ok {
				v.ResourceCount += connector.ResourceCount
				connectorMap[key] = v
			} else {
				connectorMap[key] = connector
			}
		}
	}

	for _, c := range connectionMap {
		docs = append(docs, c)
	}

	for _, c := range connectorMap {
		docs = append(docs, c)
	}

	err = kafka.DoSend(h.kafkaProducer, "kaytu-resources", 0, docs, h.logger)
	if err != nil {
		return err
	}
	return nil
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

// ListResourceTypeTags godoc
//
//	@Summary		List resourcetype tags
//	@Description	This API allows users to retrieve a list of tag keys with their possible values for all resource types.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]string	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param			minCount		query		int			false	"Minimum number of resources with this tag value, default 1"
//	@Param			endTime			query		int			false	"End time in unix timestamp format, default now"
//	@Success		200				{object}	map[string][]string
//	@Router			/inventory/api/v2/resources/tag [get]
func (h *HttpHandler) ListResourceTypeTags(ctx echo.Context) error {
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
	}
	connectorTypes, err := h.getConnectorTypesFromConnectionIDs(ctx, connectorTypes, connectionIDs)
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
	endTime := time.Now()
	if endTimeStr := ctx.QueryParam("endTime"); endTimeStr != "" {
		endTimeVal, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "endTime must be a number")
		}
		endTime = time.Unix(endTimeVal, 0)
	}

	tags, err := h.db.ListResourceTypeTagsKeysWithPossibleValues(connectorTypes, utils.GetPointer(true))
	if err != nil {
		return err
	}
	tags = model.TrimPrivateTags(tags)

	var resourceTypeCount map[string]int
	if len(connectionIDs) > 0 {
		resourceTypeCount, err = es.FetchConnectionResourceTypeCountAtTime(h.client, connectorTypes, connectionIDs, endTime, nil, EsFetchPageSize)
	} else {
		resourceTypeCount, err = es.FetchConnectorResourceTypeCountAtTime(h.client, connectorTypes, endTime, nil, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	filteredTags := map[string][]string{}
	for key, values := range tags {
		for _, tagValue := range values {
			resourceTypes, err := h.db.ListFilteredResourceTypes(map[string][]string{key: {tagValue}}, nil, nil, connectorTypes, true)
			if err != nil {
				return err
			}
			for _, resourceType := range resourceTypes {
				if resourceTypeCount[strings.ToLower(resourceType.ResourceType)] >= minCount {
					filteredTags[key] = append(filteredTags[key], tagValue)
					break
				}
			}
		}
	}
	tags = filteredTags

	return ctx.JSON(http.StatusOK, tags)
}

func (h *HttpHandler) ListAnalyticsMetrics(metricIDs []string, tagMap map[string][]string, connectorTypes []source.Type, connectionIDs []string, minCount int, timeAt time.Time) (int, []inventoryApi.Metric, error) {
	aDB := analyticsDB.NewDatabase(h.db.orm)

	mts, err := aDB.ListFilteredMetrics(tagMap, metricIDs, connectorTypes)
	if err != nil {
		return 0, nil, err
	}
	filteredMetricIDs := make([]string, 0, len(mts))
	for _, metric := range mts {
		filteredMetricIDs = append(filteredMetricIDs, metric.ID)
	}

	var metricIndexed map[string]int
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionAnalyticMetricCountAtTime(h.client, connectorTypes, connectionIDs, timeAt, filteredMetricIDs, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorAnalyticMetricCountAtTime(h.client, connectorTypes, timeAt, filteredMetricIDs, EsFetchPageSize)
	}
	if err != nil {
		return 0, nil, err
	}

	apiMetrics := make([]inventoryApi.Metric, 0, len(mts))
	totalCount := 0
	for _, metric := range mts {
		apiMetric := inventoryApi.MetricToAPI(metric)
		if count, ok := metricIndexed[metric.ID]; ok && count >= minCount {
			apiMetric.Count = &count
			totalCount += count
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
//	@Description	Get list of analytics with metrics of each type based on the given input filters.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			tag				query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			metricIDs		query		[]string		false	"Metric IDs"
//	@Param			endTime			query		string			false	"timestamp for resource count in epoch seconds"
//	@Param			startTime		query		string			false	"timestamp for resource count change comparison in epoch seconds"
//	@Param			minCount		query		int				false	"Minimum number of resources with this tag value, default 1"
//	@Param			sortBy			query		string			false	"Sort by field - default is count"	Enums(name,count,growth,growth_rate)
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Success		200				{object}	inventoryApi.ListMetricsResponse
//	@Router			/inventory/api/v2/analytics/metric [get]
func (h *HttpHandler) ListAnalyticsMetricsHandler(ctx echo.Context) error {
	var err error
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
	}
	metricIDs := httpserver.QueryArrayParam(ctx, "metricIDs")

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
	minCount := 1
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

	totalCount, apiMetrics, err := h.ListAnalyticsMetrics(metricIDs, tagMap, connectorTypes, connectionIDs, minCount, endTime)
	if err != nil {
		return err
	}
	apiMetricsMap := make(map[string]inventoryApi.Metric, len(apiMetrics))
	for _, apiMetric := range apiMetrics {
		apiMetricsMap[apiMetric.ID] = apiMetric
	}

	totalOldCount, oldApiMetrics, err := h.ListAnalyticsMetrics(metricIDs, tagMap, connectorTypes, connectionIDs, 0, startTime)
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
//	@Description	This API allows users to retrieve a list of tag keys with their possible values for all analytic metrics.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]string	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param			minCount		query		int			false	"Minimum number of resources with this tag value, default 1"
//	@Param			endTime			query		int			false	"End time in unix timestamp format, default now"
//	@Success		200				{object}	map[string][]string
//	@Router			/inventory/api/v2/analytics/tag [get]
func (h *HttpHandler) ListAnalyticsTags(ctx echo.Context) error {
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
	}
	connectorTypes, err := h.getConnectorTypesFromConnectionIDs(ctx, connectorTypes, connectionIDs)
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
	endTime := time.Now()
	if endTimeStr := ctx.QueryParam("endTime"); endTimeStr != "" {
		endTimeVal, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "endTime must be a number")
		}
		endTime = time.Unix(endTimeVal, 0)
	}

	aDB := analyticsDB.NewDatabase(h.db.orm)
	fmt.Println("connectorTypes", connectorTypes)
	tags, err := aDB.ListMetricTagsKeysWithPossibleValues(connectorTypes)
	if err != nil {
		return err
	}
	tags = model.TrimPrivateTags(tags)

	var metricCount map[string]int
	if len(connectionIDs) > 0 {
		fmt.Println("FetchConnectionAnalyticMetricCountAtTime", connectorTypes, connectionIDs, endTime)
		metricCount, err = es.FetchConnectionAnalyticMetricCountAtTime(h.client, connectorTypes, connectionIDs, endTime, nil, EsFetchPageSize)
	} else {
		fmt.Println("FetchConnectorAnalyticMetricCountAtTime", connectorTypes, endTime)
		metricCount, err = es.FetchConnectorAnalyticMetricCountAtTime(h.client, connectorTypes, endTime, nil, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	fmt.Println("metricCount", metricCount)
	fmt.Println("tags", tags)
	filteredTags := map[string][]string{}
	for key, values := range tags {
		for _, tagValue := range values {
			metrics, err := aDB.ListFilteredMetrics(map[string][]string{key: {tagValue}}, nil, connectorTypes)
			if err != nil {
				return err
			}
			fmt.Println("metrics", key, tagValue, metrics)
			for _, metric := range metrics {
				if metricCount[metric.ID] >= minCount {
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
//	@Description	This API allows users to retrieve a list of resource counts over the course of the specified time frame based on the given input filters
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			tag				query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			ids				query		[]string		false	"Metric IDs to filter by"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			startTime		query		string			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		string			false	"timestamp for end in epoch seconds"
//	@Param			datapointCount	query		string			false	"maximum number of datapoints to return, default is 30"
//	@Success		200				{object}	[]inventoryApi.ResourceTypeTrendDatapoint
//	@Router			/inventory/api/v2/analytics/trend [get]
func (h *HttpHandler) ListAnalyticsMetricTrend(ctx echo.Context) error {
	var err error
	aDB := analyticsDB.NewDatabase(h.db.orm)
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	ids := httpserver.QueryArrayParam(ctx, "ids")
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "too many connection IDs")
	}

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

	datapointCountStr := ctx.QueryParam("datapointCount")
	datapointCount := int64(30)
	if datapointCountStr != "" {
		datapointCount, err = strconv.ParseInt(datapointCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid datapointCount")
		}
	}

	metrics, err := aDB.ListFilteredMetrics(tagMap, ids, connectorTypes)
	if err != nil {
		return err
	}
	metricIDs := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		metricIDs = append(metricIDs, metric.ID)
	}

	timeToCountMap := make(map[int]int)
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
		timeToCountMap, err = es.FetchConnectionMetricTrendSummaryPage(h.client, connectionIDs, metricIDs, startTime, endTime, esDatapointCount, EsFetchPageSize)
		if err != nil {
			return err
		}
	} else {
		timeToCountMap, err = es.FetchConnectorMetricTrendSummaryPage(h.client, connectorTypes, metricIDs, startTime, endTime, esDatapointCount, EsFetchPageSize)
		if err != nil {
			return err
		}
	}

	apiDatapoints := make([]inventoryApi.ResourceTypeTrendDatapoint, 0, len(timeToCountMap))
	for timeAt, count := range timeToCountMap {
		apiDatapoints = append(apiDatapoints, inventoryApi.ResourceTypeTrendDatapoint{Count: count, Date: time.UnixMilli(int64(timeAt))})
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})
	apiDatapoints = internal.DownSampleResourceTypeTrendDatapoints(apiDatapoints, int(datapointCount))

	return ctx.JSON(http.StatusOK, apiDatapoints)
}

// ListAnalyticsComposition godoc
//
//	@Summary		List analytics composition
//	@Description	This API allows users to retrieve tag values with the most resources for the given key.
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			key				path		string			true	"Tag key"
//	@Param			top				query		int				true	"How many top values to return default is 5"
//	@Param			connector		query		[]source.Type	false	"Connector types to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			endTime			query		string			false	"timestamp for resource count in epoch seconds"
//	@Param			startTime		query		string			false	"timestamp for resource count change comparison in epoch seconds"
//	@Success		200				{object}	inventoryApi.ListResourceTypeCompositionResponse
//	@Router			/inventory/api/v2/analytics/composition/{key} [get]
func (h *HttpHandler) ListAnalyticsComposition(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)

	var err error
	tagKey := ctx.Param("key")
	if tagKey == "" || strings.HasPrefix(tagKey, model.KaytuPrivateTagPrefix) {
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
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
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

	metrics, err := aDB.ListFilteredMetrics(map[string][]string{tagKey: nil}, nil, connectorTypes)
	if err != nil {
		return err
	}
	metricsIDs := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		metricsIDs = append(metricsIDs, metric.ID)
	}

	var metricIndexed map[string]int
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionAnalyticMetricCountAtTime(h.client, connectorTypes, connectionIDs, endTime, metricsIDs, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorAnalyticMetricCountAtTime(h.client, connectorTypes, endTime, metricsIDs, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	var oldMetricIndexed map[string]int
	if len(connectionIDs) > 0 {
		oldMetricIndexed, err = es.FetchConnectionAnalyticMetricCountAtTime(h.client, connectorTypes, connectionIDs, startTime, metricsIDs, EsFetchPageSize)
	} else {
		oldMetricIndexed, err = es.FetchConnectorAnalyticMetricCountAtTime(h.client, connectorTypes, startTime, metricsIDs, EsFetchPageSize)
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
			v.current += metricIndexed[metric.ID]
			v.old += oldMetricIndexed[metric.ID]
			totalCount += metricIndexed[metric.ID]
			totalOldCount += oldMetricIndexed[metric.ID]
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

// ListAnalyticsRegionsSummary godoc
//
//	@Summary		List Regions Summary
//	@Description	Returns list of regions analytics summary
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			startTime		query		int				false	"start time in unix seconds - default is now"
//	@Param			endTime			query		int				false	"end time in unix seconds - default is one week ago"
//	@Param			sortBy			query		string			false	"column to sort by - default is resource_count"	Enums(resource_count, growth, growth_rate)
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Success		200				{object}	inventoryApi.RegionsResourceCountResponse
//	@Router			/inventory/api/v2/analytics/regions/summary [get]
func (h *HttpHandler) ListAnalyticsRegionsSummary(ctx echo.Context) error {
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
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
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) == 0 {
		connectionIDs = nil
	}

	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	sortBy := ctx.QueryParam("sortBy")
	if sortBy == "" {
		sortBy = "resource_count"
	}

	currentLocationDistribution, err := es.FetchRegionSummaryPage(h.client, connectors, connectionIDs, nil, endTime, 10000)
	if err != nil {
		return err
	}

	oldLocationDistribution, err := es.FetchRegionSummaryPage(h.client, connectors, connectionIDs, nil, startTime, 10000)
	if err != nil {
		return err
	}

	var locationResponses []inventoryApi.LocationResponse
	for region, count := range currentLocationDistribution {
		cnt := count
		oldCount := 0
		if value, ok := oldLocationDistribution[region]; ok {
			oldCount = value
		}
		locationResponses = append(locationResponses, inventoryApi.LocationResponse{
			Location:         region,
			ResourceCount:    &cnt,
			ResourceOldCount: &oldCount,
		})
	}

	sort.Slice(locationResponses, func(i, j int) bool {
		switch sortBy {
		case "resource_count":
			if locationResponses[i].ResourceCount == nil && locationResponses[j].ResourceCount == nil {
				break
			}
			if locationResponses[i].ResourceCount == nil {
				return false
			}
			if locationResponses[j].ResourceCount == nil {
				return true
			}
			if *locationResponses[i].ResourceCount != *locationResponses[j].ResourceCount {
				return *locationResponses[i].ResourceCount > *locationResponses[j].ResourceCount
			}
		case "growth":
			diffi := utils.PSub(locationResponses[i].ResourceCount, locationResponses[i].ResourceOldCount)
			diffj := utils.PSub(locationResponses[j].ResourceCount, locationResponses[j].ResourceOldCount)
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
			diffi := utils.PSub(locationResponses[i].ResourceCount, locationResponses[i].ResourceOldCount)
			diffj := utils.PSub(locationResponses[j].ResourceCount, locationResponses[j].ResourceOldCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if locationResponses[i].ResourceOldCount == nil && locationResponses[j].ResourceOldCount == nil {
				break
			}
			if locationResponses[i].ResourceOldCount == nil {
				return true
			}
			if locationResponses[j].ResourceOldCount == nil {
				return false
			}
			if *locationResponses[i].ResourceOldCount == 0 && *locationResponses[j].ResourceOldCount == 0 {
				break
			}
			if *locationResponses[i].ResourceOldCount == 0 {
				return false
			}
			if *locationResponses[j].ResourceOldCount == 0 {
				return true
			}
			if float64(*diffi)/float64(*locationResponses[i].ResourceOldCount) != float64(*diffj)/float64(*locationResponses[j].ResourceOldCount) {
				return float64(*diffi)/float64(*locationResponses[i].ResourceOldCount) > float64(*diffj)/float64(*locationResponses[j].ResourceOldCount)
			}
		}
		return locationResponses[i].Location < locationResponses[j].Location
	})

	response := inventoryApi.RegionsResourceCountResponse{
		TotalCount: len(locationResponses),
		Regions:    utils.Paginate(pageNumber, pageSize, locationResponses),
	}

	return ctx.JSON(http.StatusOK, response)
}

// ListAnalyticsCategories godoc
//
//	@Summary		List Analytics categories
//	@Description	Returns list of categories for analytics summary
//	@Security		BearerToken
//	@Tags			analytics
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	inventoryApi.AnalyticsCategoriesResponse
//	@Router			/inventory/api/v2/analytics/categories [get]
func (h *HttpHandler) ListAnalyticsCategories(ctx echo.Context) error {
	aDB := analyticsDB.NewDatabase(h.db.orm)

	metrics, err := aDB.ListMetrics()
	if err != nil {
		return err
	}

	categoryResourceTypeMap := map[string][]string{}
	for _, metric := range metrics {
		for _, tag := range metric.Tags {
			if tag.Key == "category" {
				for _, category := range tag.GetValue() {
					categoryResourceTypeMap[category] = append(
						categoryResourceTypeMap[category],
						metric.Tables...,
					)
				}
			}
		}
	}

	return ctx.JSON(http.StatusOK, inventoryApi.AnalyticsCategoriesResponse{
		CategoryResourceType: categoryResourceTypeMap,
	})
}

// GetResourceTypeMetricsHandler godoc
//
//	@Summary		Get resource metrics
//	@Description	This API allows users to retrieve metrics for a specific resource type.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param			endTime			query		string		false	"timestamp for resource count in epoch seconds"
//	@Param			startTime		query		string		false	"timestamp for resource count change comparison in epoch seconds"
//	@Param			resourceType	path		string		true	"ResourceType"
//	@Success		200				{object}	inventoryApi.ResourceType
//	@Router			/inventory/api/v2/resources/metric/{resourceType} [get]
func (h *HttpHandler) GetResourceTypeMetricsHandler(ctx echo.Context) error {
	var err error
	resourceType := ctx.Param("resourceType")
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
	}
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

	apiResourceType, err := h.GetResourceTypeMetric(resourceType, connectionIDs, endTime)
	if err != nil {
		return err
	}

	oldApiResourceType, err := h.GetResourceTypeMetric(resourceType, connectionIDs, startTime)
	if err != nil {
		return err
	}
	apiResourceType.OldCount = oldApiResourceType.Count

	return ctx.JSON(http.StatusOK, *apiResourceType)
}

func (h *HttpHandler) GetResourceTypeMetric(resourceTypeStr string, connectionIDs []string, timeAt int64) (*inventoryApi.ResourceType, error) {
	resourceType, err := h.db.GetResourceType(resourceTypeStr)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, echo.NewHTTPError(http.StatusNotFound, "resource type not found")
		}
		return nil, err
	}

	var metricIndexed map[string]int
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionResourceTypeCountAtTime(h.client, nil, connectionIDs, time.Unix(timeAt, 0), []string{resourceTypeStr}, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorResourceTypeCountAtTime(h.client, nil, time.Unix(timeAt, 0), []string{resourceTypeStr}, EsFetchPageSize)
	}
	if err != nil {
		return nil, err
	}

	apiResourceType := resourceType.ToApi()
	if count, ok := metricIndexed[strings.ToLower(resourceType.ResourceType)]; ok {
		apiResourceType.Count = &count
	}

	return &apiResourceType, nil
}

// ListCostMetricsHandler godoc
//
//	@Summary		List cost metrics
//	@Description	This API allows users to retrieve cost metrics with respect to specified filters. The API returns information such as the total cost and costs per each service based on the specified filters.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			startTime		query		string			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		string			false	"timestamp for end in epoch seconds"
//	@Param			sortBy			query		string			false	"Sort by field - default is cost"	Enums(dimension,cost,growth,growth_rate)
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Success		200				{object}	inventoryApi.ListCostMetricsResponse
//	@Router			/inventory/api/v2/cost/metric [get]
func (h *HttpHandler) ListCostMetricsHandler(ctx echo.Context) error {
	var err error
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
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

	costHits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, connectionIDs, connectorTypes, nil, time.Unix(startTime, 0), time.Unix(endTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}
	costMetricMap := make(map[string]inventoryApi.CostMetric)
	for connector, serviceToCostMap := range costHits {
		for dimension, costVal := range serviceToCostMap {
			connectorTyped, _ := source.ParseType(connector)
			localCostVal := costVal
			costMetricMap[dimension] = inventoryApi.CostMetric{
				Connector:         connectorTyped,
				CostDimensionName: dimension,
				TotalCost:         &localCostVal,
			}

		}
	}

	endTimeCostHits, err := es.FetchDailyCostHistoryByServicesAtTime(h.client, connectionIDs, connectorTypes, nil, time.Unix(endTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}
	aggregatedEndTimeCostHits := internal.AggregateServiceCosts(endTimeCostHits)
	for dimension, costVal := range aggregatedEndTimeCostHits {
		if costMetric, ok := costMetricMap[dimension]; ok {
			localCostVal := costVal
			costMetric.DailyCostAtEndTime = utils.PAdd(costMetric.DailyCostAtEndTime, &localCostVal)
			costMetricMap[dimension] = costMetric
		}
	}

	startTimeCostHits, err := es.FetchDailyCostHistoryByServicesAtTime(h.client, connectionIDs, connectorTypes, nil, time.Unix(startTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}
	aggregatedStartTimeCostHits := internal.AggregateServiceCosts(startTimeCostHits)
	for dimension, costVal := range aggregatedStartTimeCostHits {
		if costMetric, ok := costMetricMap[dimension]; ok {
			localCostVal := costVal
			costMetric.DailyCostAtStartTime = utils.PAdd(costMetric.DailyCostAtStartTime, &localCostVal)
			costMetricMap[dimension] = costMetric
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
				return *diffi > *diffj
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
				return *diffi/(*costMetrics[i].DailyCostAtStartTime) > *diffj/(*costMetrics[j].DailyCostAtStartTime)
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

// ListCostComposition godoc
//
//	@Summary		List cost composition
//	@Description	This API allows users to retrieve the cost composition with respect to specified filters. The API returns information such as the total cost for the given time range, and the top services by cost.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			top				query		int				false	"How many top values to return default is 5"
//	@Param			startTime		query		string			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		string			false	"timestamp for end in epoch seconds"
//	@Success		200				{object}	inventoryApi.ListCostCompositionResponse
//	@Router			/inventory/api/v2/cost/composition [get]
func (h *HttpHandler) ListCostComposition(ctx echo.Context) error {
	var err error
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
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
	topStr := ctx.QueryParam("top")
	top := int64(5)
	if topStr != "" {
		top, err = strconv.ParseInt(topStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid top value")
		}
	}

	costHits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, connectionIDs, connectorTypes, nil, time.Unix(startTime, 0), time.Unix(endTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}
	costMetricMap := make(map[string]inventoryApi.CostMetric)
	for connector, serviceToCostMap := range costHits {
		for dimension, costVal := range serviceToCostMap {
			connectorTyped, _ := source.ParseType(connector)
			localCostVal := costVal
			costMetricMap[dimension] = inventoryApi.CostMetric{
				Connector:         connectorTyped,
				CostDimensionName: dimension,
				TotalCost:         &localCostVal,
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

// GetCostTrend godoc
//
//	@Summary		Get Cost Trend
//	@Description	This API allows users to retrieve a list of costs over the course of the specified time frame based on the given input filters. If startTime and endTime are empty, the API returns the last month trend.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			startTime		query		string			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		string			false	"timestamp for end in epoch seconds"
//	@Param			datapointCount	query		string			false	"maximum number of datapoints to return, default is 30"
//	@Success		200				{object}	[]inventoryApi.CostTrendDatapoint
//	@Router			/inventory/api/v2/cost/trend [get]
func (h *HttpHandler) GetCostTrend(ctx echo.Context) error {
	var err error
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")

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

	datapointCountStr := ctx.QueryParam("datapointCount")
	datapointCount := int64(30)
	if datapointCountStr != "" {
		datapointCount, err = strconv.ParseInt(datapointCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid datapointCount")
		}
	}

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

	esDataPointCount := int(endTime.Sub(startTime).Hours() / 24)
	if esDataPointCount == 0 {
		esDataPointCount = 1
	}
	timepointToCost, err := es.FetchDailyCostTrendBetween(h.client, connectionIDs, connectorTypes, startTime, endTime, esDataPointCount)
	if err != nil {
		return err
	}

	apiDatapoints := make([]inventoryApi.CostTrendDatapoint, 0, len(timepointToCost))
	for timeAt, costVal := range timepointToCost {
		apiDatapoints = append(apiDatapoints, inventoryApi.CostTrendDatapoint{Cost: costVal, Date: time.Unix(int64(timeAt), 0)})
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})
	apiDatapoints = internal.DownSampleCostTrendDatapoints(apiDatapoints, int(datapointCount))

	return ctx.JSON(http.StatusOK, apiDatapoints)
}

func (h *HttpHandler) ListConnectionsData(ctx echo.Context) error {
	var err error
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
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

	res := map[string]inventoryApi.ConnectionData{}
	resourceCounts, err := es.FetchConnectionAnalyticsResourcesCountAtTime(h.client, connectors, connectionIDs, endTime, EsFetchPageSize)
	if err != nil {
		return err
	}
	for _, hit := range resourceCounts {
		localHit := hit
		if _, ok := res[localHit.ConnectionID.String()]; !ok {
			res[localHit.ConnectionID.String()] = inventoryApi.ConnectionData{
				ConnectionID: localHit.ConnectionID.String(),
			}
		}
		v := res[localHit.ConnectionID.String()]
		v.Count = utils.PAdd(v.Count, &localHit.ResourceCount)
		if v.LastInventory == nil || v.LastInventory.IsZero() || v.LastInventory.Before(time.UnixMilli(localHit.EvaluatedAt)) {
			v.LastInventory = utils.GetPointer(time.UnixMilli(localHit.EvaluatedAt))
		}
		res[localHit.ConnectionID.String()] = v
	}
	oldResourceCount, err := es.FetchConnectionAnalyticsResourcesCountAtTime(h.client, connectors, connectionIDs, startTime, EsFetchPageSize)
	if err != nil {
		return err
	}
	for _, hit := range oldResourceCount {
		localHit := hit
		if _, ok := res[localHit.ConnectionID.String()]; !ok {
			res[localHit.ConnectionID.String()] = inventoryApi.ConnectionData{
				ConnectionID:  localHit.ConnectionID.String(),
				LastInventory: nil,
			}
		}
		v := res[localHit.ConnectionID.String()]
		v.OldCount = utils.PAdd(v.OldCount, &localHit.ResourceCount)
		if v.LastInventory == nil || v.LastInventory.IsZero() || v.LastInventory.Before(time.UnixMilli(localHit.EvaluatedAt)) {
			v.LastInventory = utils.GetPointer(time.UnixMilli(localHit.EvaluatedAt))
		}
		res[localHit.ConnectionID.String()] = v
	}

	costs, err := es.FetchDailyCostHistoryByAccountsBetween(h.client, connectors, connectionIDs, endTime, startTime, EsFetchPageSize)
	if err != nil {
		return err
	}

	startTimeCosts, err := es.FetchDailyCostHistoryByAccountsAtTime(h.client, connectors, connectionIDs, startTime)
	if err != nil {
		return err
	}
	endTimeCosts, err := es.FetchDailyCostHistoryByAccountsAtTime(h.client, connectors, connectionIDs, endTime)
	if err != nil {
		return err
	}

	for connectionId, costValue := range costs {
		localValue := costValue
		if _, ok := res[connectionId]; !ok {
			res[connectionId] = inventoryApi.ConnectionData{
				ConnectionID:  connectionId,
				LastInventory: nil,
			}
		}
		if v, ok := res[connectionId]; ok {
			v.TotalCost = utils.PAdd(v.TotalCost, &localValue)
			res[connectionId] = v
		}
	}
	for connectionId, costValue := range startTimeCosts {
		localValue := costValue
		if _, ok := res[connectionId]; !ok {
			res[connectionId] = inventoryApi.ConnectionData{
				ConnectionID:  connectionId,
				LastInventory: nil,
			}
		}
		if v, ok := res[connectionId]; ok {
			v.DailyCostAtStartTime = utils.PAdd(v.DailyCostAtStartTime, &localValue)
			res[connectionId] = v
		}
	}
	for connectionId, costValue := range endTimeCosts {
		if _, ok := res[connectionId]; !ok {
			res[connectionId] = inventoryApi.ConnectionData{
				ConnectionID:  connectionId,
				LastInventory: nil,
			}
		}
		localValue := costValue
		if v, ok := res[connectionId]; ok {
			v.DailyCostAtEndTime = utils.PAdd(v.DailyCostAtEndTime, &localValue)
			res[connectionId] = v
		}
	}

	return ctx.JSON(http.StatusOK, res)
}

func (h *HttpHandler) GetConnectionData(ctx echo.Context) error {
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

	res := inventoryApi.ConnectionData{
		ConnectionID: connectionId,
	}

	resourceCounts, err := es.FetchConnectionAnalyticsResourcesCountAtTime(h.client, nil, []string{connectionId}, endTime, EsFetchPageSize)
	for _, hit := range resourceCounts {
		if hit.ConnectionID.String() != connectionId {
			continue
		}
		localHit := hit
		res.Count = utils.PAdd(res.Count, &localHit.ResourceCount)
		if res.LastInventory == nil || res.LastInventory.IsZero() || res.LastInventory.Before(time.UnixMilli(localHit.EvaluatedAt)) {
			res.LastInventory = utils.GetPointer(time.UnixMilli(localHit.EvaluatedAt))
		}
	}

	oldResourceCounts, err := es.FetchConnectionAnalyticsResourcesCountAtTime(h.client, nil, []string{connectionId}, startTime, EsFetchPageSize)
	for _, hit := range oldResourceCounts {
		if hit.ConnectionID.String() != connectionId {
			continue
		}
		localHit := hit
		res.OldCount = utils.PAdd(res.OldCount, &localHit.ResourceCount)
		if res.LastInventory == nil || res.LastInventory.IsZero() || res.LastInventory.Before(time.UnixMilli(localHit.EvaluatedAt)) {
			res.LastInventory = utils.GetPointer(time.UnixMilli(localHit.EvaluatedAt))
		}
	}

	costs, err := es.FetchDailyCostHistoryByAccountsBetween(h.client, nil, []string{connectionId}, endTime, startTime, EsFetchPageSize)
	if err != nil {
		return err
	}
	startTimeCosts, err := es.FetchDailyCostHistoryByAccountsAtTime(h.client, nil, []string{connectionId}, startTime)
	if err != nil {
		return err
	}
	endTimeCosts, err := es.FetchDailyCostHistoryByAccountsAtTime(h.client, nil, []string{connectionId}, endTime)
	if err != nil {
		return err
	}

	for costConnectionId, costValue := range costs {
		if costConnectionId != connectionId {
			continue
		}
		localValue := costValue
		res.TotalCost = utils.PAdd(res.TotalCost, &localValue)
	}
	for costConnectionId, costValue := range startTimeCosts {
		if costConnectionId != connectionId {
			continue
		}
		localValue := costValue
		res.DailyCostAtStartTime = utils.PAdd(res.DailyCostAtStartTime, &localValue)
	}
	for costConnectionId, costValue := range endTimeCosts {
		if costConnectionId != connectionId {
			continue
		}
		localValue := costValue
		res.DailyCostAtEndTime = utils.PAdd(res.DailyCostAtEndTime, &localValue)
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetServiceCostTrend godoc
//
//	@Summary		Get Services Cost Trend
//	@Description	This API allows users to retrieve a list of costs over the course of the specified time frame for the given services. If startTime and endTime are empty, the API returns the last month trend.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			services		query		[]string		false	"Services to filter by"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			startTime		query		string			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		string			false	"timestamp for end in epoch seconds"
//	@Param			datapointCount	query		string			false	"maximum number of datapoints to return, default is 30"
//	@Success		200				{object}	[]inventoryApi.CostTrendDatapoint
//	@Router			/inventory/api/v2/services/cost/trend [get]
func (h *HttpHandler) GetServiceCostTrend(ctx echo.Context) error {
	var err error
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	services := httpserver.QueryArrayParam(ctx, "services")
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

	datapointCountStr := ctx.QueryParam("datapointCount")
	datapointCount := int64(30)
	if datapointCountStr != "" {
		datapointCount, err = strconv.ParseInt(datapointCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid datapointCount")
		}
	}

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

	esDataPointCount := int(endTime.Sub(startTime).Hours() / 24)
	if esDataPointCount == 0 {
		esDataPointCount = 1
	}
	servicesTimepointToCost, err := es.FetchDailyCostTrendByServicesBetween(h.client, connectionIDs, connectorTypes, services, startTime, endTime, esDataPointCount)
	if err != nil {
		return err
	}
	var response []inventoryApi.ListServicesCostTrendDatapoint
	for service, timepointToCost := range servicesTimepointToCost {
		apiDatapoints := make([]inventoryApi.CostTrendDatapoint, 0, len(timepointToCost))
		for timeAt, costVal := range timepointToCost {
			apiDatapoints = append(apiDatapoints, inventoryApi.CostTrendDatapoint{Cost: costVal, Date: time.Unix(int64(timeAt), 0)})
		}
		sort.Slice(apiDatapoints, func(i, j int) bool {
			return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
		})
		apiDatapoints = internal.DownSampleCostTrendDatapoints(apiDatapoints, int(datapointCount))
		response = append(response, inventoryApi.ListServicesCostTrendDatapoint{ServiceName: service, CostTrend: apiDatapoints})
	}
	return ctx.JSON(http.StatusOK, response)
}

// ListQueries godoc
//
//	@Summary		List smart queries
//	@Description	Listing smart queries by specified filters
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

	queries, err := h.db.GetQueriesWithFilters(search, req.Connectors)
	if err != nil {
		return err
	}

	var result []inventoryApi.SmartQueryItem
	for _, item := range queries {
		category := ""

		result = append(result, inventoryApi.SmartQueryItem{
			ID:          item.Model.ID,
			Provider:    item.Connector,
			Title:       item.Title,
			Category:    category,
			Description: item.Description,
			Query:       item.Query,
			Tags:        nil,
		})
	}
	return ctx.JSON(200, result)
}

// RunQuery godoc
//
//	@Summary		Run provided smart query and returns the result
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
	resp, err := h.RunSmartQuery(ctx.Request().Context(), *req.Query, *req.Query, &req)
	if err != nil {
		return err
	}
	return ctx.JSON(200, resp)
}

// GetRecentRanQueries godoc
//
//	@Summary		Get recently ran queries
//	@Description	Get recently ran queries.
//	@Security		BearerToken
//	@Tags			smart_query
//	@Accepts		json
//	@Produce		json
//	@Success		200	{object}	[]inventoryApi.SmartQueryHistory
//	@Router			/inventory/api/v1/query/run/history [get]
func (h *HttpHandler) GetRecentRanQueries(ctx echo.Context) error {
	smartQueryHistories, err := h.db.GetQueryHistory()
	if err != nil {
		h.logger.Error("Failed to get query history", zap.Error(err))
		return err
	}

	res := make([]inventoryApi.SmartQueryHistory, 0, len(smartQueryHistories))
	for _, history := range smartQueryHistories {
		res = append(res, history.ToApi())
	}

	return ctx.JSON(200, res)
}

func (h *HttpHandler) CountResources(ctx echo.Context) error {
	timeAt := time.Now()
	resourceTypes, err := h.db.ListFilteredResourceTypes(nil, nil, nil, nil, true)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	resourceTypeNames := make([]string, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		resourceTypeNames = append(resourceTypeNames, resourceType.ResourceType)
	}

	metricsIndexed, err := es.FetchConnectorResourceTypeCountAtTime(h.client, nil, timeAt, resourceTypeNames, EsFetchPageSize)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	totalCount := 0
	for _, count := range metricsIndexed {
		totalCount += count
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
		return nil, err
	}

	err = h.db.UpdateQueryHistory(query)
	if err != nil {
		h.logger.Error("failed to update query history", zap.Error(err))
		return nil, err
	}

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
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	timeStr := ctx.QueryParam("time")
	timeAt := time.Now().Unix()
	if timeStr != "" {
		timeAt, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")

	insightIdListStr := httpserver.QueryArrayParam(ctx, "insightId")
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
		insightValues, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), connectors, connectionIDs, insightIdList, true)
	} else {
		insightValues, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), connectors, connectionIDs, insightIdList, false)
	}
	if err != nil {
		return err
	}

	firstAvailable, err := es.FetchInsightValueAfter(h.client, time.Unix(timeAt, 0), connectors, connectionIDs, insightIdList)
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
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) == 0 {
		connectionIDs = nil
	}

	var insightResults map[uint][]insight.InsightResource
	if timeStr != "" {
		insightResults, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), nil, connectionIDs, []uint{uint(insightId)}, true)
	} else {
		insightResults, err = es.FetchInsightValueAtTime(h.client, time.Unix(timeAt, 0), nil, connectionIDs, []uint{uint(insightId)}, false)
	}
	if err != nil {
		return err
	}

	firstAvailable, err := es.FetchInsightValueAfter(h.client, time.Unix(timeAt, 0), nil, connectionIDs, []uint{uint(insightId)})
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

	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")

	dataPointCount := int(endTime.Sub(startTime).Hours() / 24)
	insightResults, err := es.FetchInsightAggregatedPerQueryValuesBetweenTimes(h.client, startTime, endTime, dataPointCount, nil, connectionIDs, []uint{uint(insightId)})
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
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	serviceNames := httpserver.QueryArrayParam(ctx, "service")
	resourceTypeNames := httpserver.QueryArrayParam(ctx, "resourceType")
	summarized := strings.ToLower(ctx.QueryParam("summarized")) == "true"
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	resourceTypes, err := h.db.ListFilteredResourceTypes(tagMap, resourceTypeNames, serviceNames, connectors, summarized)
	if err != nil {
		return err
	}

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
