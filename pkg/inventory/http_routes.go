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

	"github.com/labstack/echo/v4"
	"github.com/turbot/steampipe-plugin-sdk/v4/grpc/proto"
	"go.uber.org/zap"
	"gorm.io/gorm"

	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/cloudservice"
	insight "github.com/kaytu-io/kaytu-engine/pkg/insight/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/es"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory/internal"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	summarizer "github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
)

const EsFetchPageSize = 10000

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.GET("/locations/:connector", httpserver.AuthorizeHandler(h.GetLocations, authApi.ViewerRole))

	v1.POST("/resources", httpserver.AuthorizeHandler(h.GetResources, authApi.ViewerRole))
	v1.POST("/resources/filters", httpserver.AuthorizeHandler(h.GetResourcesFilters, authApi.ViewerRole))
	v1.POST("/resource", httpserver.AuthorizeHandler(h.GetResource, authApi.ViewerRole))

	v1.GET("/resources/top/regions", httpserver.AuthorizeHandler(h.GetTopRegionsByResourceCount, authApi.ViewerRole))
	v1.GET("/resources/regions", httpserver.AuthorizeHandler(h.GetRegionsByResourceCount, authApi.ViewerRole))

	v1.GET("/query", httpserver.AuthorizeHandler(h.ListQueries, authApi.ViewerRole))
	v1.GET("/query/count", httpserver.AuthorizeHandler(h.CountQueries, authApi.ViewerRole))
	v1.POST("/query/:queryId", httpserver.AuthorizeHandler(h.RunQuery, authApi.EditorRole))

	v2 := e.Group("/api/v2")

	resourcesV2 := v2.Group("/resources")
	resourcesV2.GET("/tag", httpserver.AuthorizeHandler(h.ListResourceTypeTags, authApi.ViewerRole))
	resourcesV2.GET("/tag/:key", httpserver.AuthorizeHandler(h.GetResourceTypeTag, authApi.ViewerRole))
	resourcesV2.GET("/count", httpserver.AuthorizeHandler(h.CountResources, authApi.ViewerRole))
	resourcesV2.GET("/metric", httpserver.AuthorizeHandler(h.ListResourceTypeMetricsHandler, authApi.ViewerRole))
	resourcesV2.GET("/metric/:resourceType", httpserver.AuthorizeHandler(h.GetResourceTypeMetricsHandler, authApi.ViewerRole))
	resourcesV2.GET("/composition/:key", httpserver.AuthorizeHandler(h.ListResourceTypeComposition, authApi.ViewerRole))
	resourcesV2.GET("/trend", httpserver.AuthorizeHandler(h.ListResourceTypeTrend, authApi.ViewerRole))
	resourcesV2.GET("/regions/summary", httpserver.AuthorizeHandler(h.ListResourcesRegionsSummary, authApi.ViewerRole))
	resourcesV2.GET("/regions/composition", httpserver.AuthorizeHandler(h.ListResourcesRegionsComposition, authApi.ViewerRole))
	resourcesV2.GET("/regions/trend", httpserver.AuthorizeHandler(h.ListResourcesRegionsTrend, authApi.ViewerRole))

	servicesV2 := v2.Group("/services")
	servicesV2.GET("/tag", httpserver.AuthorizeHandler(h.ListServiceTags, authApi.ViewerRole))
	servicesV2.GET("/tag/:key", httpserver.AuthorizeHandler(h.GetServiceTag, authApi.ViewerRole))
	servicesV2.GET("/metric", httpserver.AuthorizeHandler(h.ListServiceMetricsHandler, authApi.ViewerRole))
	servicesV2.GET("/metric/:serviceName", httpserver.AuthorizeHandler(h.GetServiceMetricsHandler, authApi.ViewerRole))
	servicesV2.GET("/summary", httpserver.AuthorizeHandler(h.ListServiceSummaries, authApi.ViewerRole))
	servicesV2.GET("/summary/:serviceName", httpserver.AuthorizeHandler(h.GetServiceSummary, authApi.ViewerRole))
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
	insightsV2.GET("/job/:jobId", httpserver.AuthorizeHandler(h.GetInsightResultByJobId, authApi.ViewerRole))
	insightsV2.GET("/:insightId/trend", httpserver.AuthorizeHandler(h.GetInsightTrendResults, authApi.ViewerRole))
	insightsV2.GET("/:insightId", httpserver.AuthorizeHandler(h.GetInsightResult, authApi.ViewerRole))

	metadata := v2.Group("/metadata")
	metadata.GET("/services", httpserver.AuthorizeHandler(h.ListServiceMetadata, authApi.ViewerRole))
	metadata.GET("/services/:serviceName", httpserver.AuthorizeHandler(h.GetServiceMetadata, authApi.ViewerRole))
	metadata.GET("/resourcetype", httpserver.AuthorizeHandler(h.ListResourceTypeMetadata, authApi.ViewerRole))
	metadata.GET("/resourcetype/:resourceType", httpserver.AuthorizeHandler(h.GetResourceTypeMetadata, authApi.ViewerRole))
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

// GetTopRegionsByResourceCount godoc
//
//	@Summary	Returns top n regions of specified provider by resource count
//	@Security	BearerToken
//	@Tags		resource
//	@Accept		json
//	@Produce	json
//	@Param		count			query		int				true	"count"
//	@Param		connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param		connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Success	200				{object}	[]inventoryApi.LocationResponse
//	@Router		/inventory/api/v1/resources/top/regions [get]
func (h *HttpHandler) GetTopRegionsByResourceCount(ctx echo.Context) error {
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	count, err := strconv.Atoi(ctx.QueryParam("count"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid count")
	}

	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) == 0 {
		connectionIDs = nil
	}

	locationDistribution := map[string]int{}

	hits, err := es.FetchConnectionLocationsSummaryPage(h.client, connectors, connectionIDs, nil, time.Now())
	if err != nil {
		return err
	}
	for _, hit := range hits {
		for k, v := range hit.LocationDistribution {
			locationDistribution[k] += v
		}
	}

	var response []inventoryApi.LocationResponse
	for region, count := range locationDistribution {
		cnt := count
		response = append(response, inventoryApi.LocationResponse{
			Location:      region,
			ResourceCount: &cnt,
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
//	@Tags		resource
//	@Accept		json
//	@Produce	json
//	@Param		connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param		connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param		endTime			query		string			false	"timestamp for resource count per location in epoch seconds"
//	@Param		startTime		query		string			false	"timestamp for resource count per location change comparison in epoch seconds"
//	@Param		pageSize		query		int				false	"page size - default is 20"
//	@Param		pageNumber		query		int				false	"page number - default is 1"
//	@Success	200				{object}	inventoryApi.RegionsResourceCountResponse
//	@Router		/inventory/api/v1/resources/regions [get]
func (h *HttpHandler) GetRegionsByResourceCount(ctx echo.Context) error {
	var err error
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) == 0 {
		connectionIDs = nil
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
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	locationDistribution := map[string]int{}
	hits, err := es.FetchConnectionLocationsSummaryPage(h.client, connectors, connectionIDs, nil, time.Unix(endTime, 0))
	if err != nil {
		return err
	}
	for _, hit := range hits {
		for k, v := range hit.LocationDistribution {
			locationDistribution[k] += v
		}
	}
	oldLocationDistribution := map[string]int{}
	hits, err = es.FetchConnectionLocationsSummaryPage(h.client, connectors, connectionIDs, nil, time.Unix(startTime, 0))
	if err != nil {
		return err
	}
	for _, hit := range hits {
		for k, v := range hit.LocationDistribution {
			oldLocationDistribution[k] += v
		}
	}

	var response []inventoryApi.LocationResponse
	for region, count := range locationDistribution {
		cnt := count
		res := inventoryApi.LocationResponse{
			Location:      region,
			ResourceCount: &cnt,
		}
		if oldLocationDistribution[region] != 0 {
			res.ResourceOldCount = utils.GetPointer(oldLocationDistribution[region])
		}
		response = append(response, res)
	}
	sort.Slice(response, func(i, j int) bool {
		if *response[i].ResourceCount != *response[j].ResourceCount {
			return *response[i].ResourceCount > *response[j].ResourceCount
		}
		return response[i].Location < response[j].Location
	})

	return ctx.JSON(http.StatusOK, inventoryApi.RegionsResourceCountResponse{
		TotalCount: len(response),
		Regions:    utils.Paginate(pageNumber, pageSize, response),
	})
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

// GetResourceTypeTag godoc
//
//	@Summary		Get resourcetype tag
//	@Description	This API allows users to retrieve a list of possible values for a given tag key for all resource types.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]string	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param			minCount		query		int			false	"Minimum number of resources with this tag value, default 1"
//	@Param			endTime			query		int			false	"End time in unix timestamp format, default now"
//	@Param			key				path		string		true	"Tag key"
//	@Success		200				{object}	[]string
//	@Router			/inventory/api/v2/resources/tag/{key} [get]
func (h *HttpHandler) GetResourceTypeTag(ctx echo.Context) error {
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
	}
	connectorTypes, err := h.getConnectorTypesFromConnectionIDs(ctx, connectorTypes, connectionIDs)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	tagKey := ctx.Param("key")
	if tagKey == "" || strings.HasPrefix(tagKey, model.KaytuPrivateTagPrefix) {
		return echo.NewHTTPError(http.StatusBadRequest, "tag key is invalid")
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

	tags, err := h.db.GetResourceTypeTagPossibleValues(tagKey, connectorTypes, utils.GetPointer(true))
	if err != nil {
		return err
	}

	var resourceTypeCount map[string]int
	if len(connectionIDs) > 0 {
		resourceTypeCount, err = es.FetchConnectionResourceTypeCountAtTime(h.client, connectorTypes, connectionIDs, endTime, nil, EsFetchPageSize)
	} else {
		resourceTypeCount, err = es.FetchConnectorResourceTypeCountAtTime(h.client, connectorTypes, endTime, nil, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	filteredTags := make([]string, 0, len(tags))
	for _, tagValue := range tags {
		resourceTypes, err := h.db.ListFilteredResourceTypes(map[string][]string{tagKey: {tagValue}}, nil, nil, connectorTypes, true)
		if err != nil {
			return err
		}
		for _, resourceType := range resourceTypes {
			if resourceTypeCount[strings.ToLower(resourceType.ResourceType)] >= minCount {
				filteredTags = append(filteredTags, tagValue)
				break
			}
		}
	}
	tags = filteredTags

	return ctx.JSON(http.StatusOK, tags)
}

func (h *HttpHandler) ListResourceTypeMetrics(tagMap map[string][]string, serviceNames []string, connectorTypes []source.Type, connectionIDs []string, minCount int, timeAt time.Time) (int, []inventoryApi.ResourceType, error) {
	resourceTypes, err := h.db.ListFilteredResourceTypes(tagMap, nil, serviceNames, connectorTypes, true)
	if err != nil {
		return 0, nil, err
	}
	resourceTypeStrings := make([]string, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		resourceTypeStrings = append(resourceTypeStrings, resourceType.ResourceType)
	}

	var metricIndexed map[string]int
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionResourceTypeCountAtTime(h.client, connectorTypes, connectionIDs, timeAt, resourceTypeStrings, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorResourceTypeCountAtTime(h.client, connectorTypes, timeAt, resourceTypeStrings, EsFetchPageSize)
	}
	if err != nil {
		return 0, nil, err
	}

	apiResourceTypes := make([]inventoryApi.ResourceType, 0, len(resourceTypes))
	totalCount := 0
	for _, resourceType := range resourceTypes {
		apiResourceType := resourceType.ToApi()
		if count, ok := metricIndexed[strings.ToLower(resourceType.ResourceType)]; ok && count >= minCount {
			apiResourceType.Count = &count
			totalCount += count
		}
		if (minCount == 0) || (apiResourceType.Count != nil && *apiResourceType.Count >= minCount) {
			apiResourceTypes = append(apiResourceTypes, apiResourceType)
		}
	}

	return totalCount, apiResourceTypes, nil
}

// ListResourceTypeMetricsHandler godoc
//
//	@Summary		List resource metrics
//	@Description	This API allows users to retrieve a list of resource types with metrics of each type based on the given input filters.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			tag				query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			servicename		query		[]string		false	"Service names to filter by"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			endTime			query		string			false	"timestamp for resource count in epoch seconds"
//	@Param			startTime		query		string			false	"timestamp for resource count change comparison in epoch seconds"
//	@Param			minCount		query		int				false	"Minimum number of resources with this tag value, default 1"
//	@Param			sortBy			query		string			false	"Sort by field - default is count"	Enums(name,count,growth,growth_rate)
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Success		200				{object}	inventoryApi.ListResourceTypeMetricsResponse
//	@Router			/inventory/api/v2/resources/metric [get]
func (h *HttpHandler) ListResourceTypeMetricsHandler(ctx echo.Context) error {
	var err error
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	serviceNames := httpserver.QueryArrayParam(ctx, "servicename")
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
	}
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

	totalCount, apiResourceTypes, err := h.ListResourceTypeMetrics(tagMap, serviceNames, connectorTypes, connectionIDs, minCount, endTime)
	if err != nil {
		return err
	}
	apiResourceTypesMap := make(map[string]inventoryApi.ResourceType, len(apiResourceTypes))
	for _, apiResourceType := range apiResourceTypes {
		apiResourceTypesMap[apiResourceType.ResourceType] = apiResourceType
	}

	totalOldCount, oldApiResourceTypes, err := h.ListResourceTypeMetrics(tagMap, serviceNames, connectorTypes, connectionIDs, 0, startTime)
	if err != nil {
		return err
	}
	for _, oldApiResourceType := range oldApiResourceTypes {
		if apiResourceType, ok := apiResourceTypesMap[oldApiResourceType.ResourceType]; ok {
			apiResourceType.OldCount = oldApiResourceType.Count
			apiResourceTypesMap[oldApiResourceType.ResourceType] = apiResourceType
		}
	}

	apiResourceTypes = make([]inventoryApi.ResourceType, 0, len(apiResourceTypesMap))
	for _, apiResourceType := range apiResourceTypesMap {
		apiResourceTypes = append(apiResourceTypes, apiResourceType)
	}

	sort.Slice(apiResourceTypes, func(i, j int) bool {
		switch sortBy {
		case "name":
			return apiResourceTypes[i].ResourceType < apiResourceTypes[j].ResourceType
		case "count":
			if apiResourceTypes[i].Count == nil && apiResourceTypes[j].Count == nil {
				break
			}
			if apiResourceTypes[i].Count == nil {
				return false
			}
			if apiResourceTypes[j].Count == nil {
				return true
			}
			if *apiResourceTypes[i].Count != *apiResourceTypes[j].Count {
				return *apiResourceTypes[i].Count > *apiResourceTypes[j].Count
			}
		case "growth":
			diffi := utils.PSub(apiResourceTypes[i].Count, apiResourceTypes[i].OldCount)
			diffj := utils.PSub(apiResourceTypes[j].Count, apiResourceTypes[j].OldCount)
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
			diffi := utils.PSub(apiResourceTypes[i].Count, apiResourceTypes[i].OldCount)
			diffj := utils.PSub(apiResourceTypes[j].Count, apiResourceTypes[j].OldCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if apiResourceTypes[i].OldCount == nil && apiResourceTypes[j].OldCount == nil {
				break
			}
			if apiResourceTypes[i].OldCount == nil {
				return true
			}
			if apiResourceTypes[j].OldCount == nil {
				return false
			}
			if *apiResourceTypes[i].OldCount == 0 && *apiResourceTypes[j].OldCount == 0 {
				break
			}
			if *apiResourceTypes[i].OldCount == 0 {
				return false
			}
			if *apiResourceTypes[j].OldCount == 0 {
				return true
			}
			if float64(*diffi)/float64(*apiResourceTypes[i].OldCount) != float64(*diffj)/float64(*apiResourceTypes[j].OldCount) {
				return float64(*diffi)/float64(*apiResourceTypes[i].OldCount) > float64(*diffj)/float64(*apiResourceTypes[j].OldCount)
			}
		}
		return apiResourceTypes[i].ResourceType < apiResourceTypes[j].ResourceType
	})

	result := inventoryApi.ListResourceTypeMetricsResponse{
		TotalCount:         totalCount,
		TotalOldCount:      totalOldCount,
		TotalResourceTypes: len(apiResourceTypes),
		ResourceTypes:      utils.Paginate(pageNumber, pageSize, apiResourceTypes),
	}

	return ctx.JSON(http.StatusOK, result)
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

// ListResourceTypeComposition godoc
//
//	@Summary		List resource type composition
//	@Description	This API allows users to retrieve tag values with the most resources for the given key.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			key				path		string			true	"Tag key"
//	@Param			top				query		int				true	"How many top values to return default is 5"
//	@Param			connector		query		[]source.Type	false	"Connector types to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			endTime			query		string			false	"timestamp for resource count in epoch seconds"
//	@Param			startTime		query		string			false	"timestamp for resource count change comparison in epoch seconds"
//	@Success		200				{object}	inventoryApi.ListResourceTypeCompositionResponse
//	@Router			/inventory/api/v2/resources/composition/{key} [get]
func (h *HttpHandler) ListResourceTypeComposition(ctx echo.Context) error {
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

	resourceTypes, err := h.db.ListFilteredResourceTypes(map[string][]string{tagKey: nil}, nil, nil, connectorTypes, true)
	if err != nil {
		return err
	}
	resourceTypeStrings := make([]string, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		resourceTypeStrings = append(resourceTypeStrings, resourceType.ResourceType)
	}

	var metricIndexed map[string]int
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionResourceTypeCountAtTime(h.client, connectorTypes, connectionIDs, endTime, resourceTypeStrings, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorResourceTypeCountAtTime(h.client, connectorTypes, endTime, resourceTypeStrings, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	var oldMetricIndexed map[string]int
	if len(connectionIDs) > 0 {
		oldMetricIndexed, err = es.FetchConnectionResourceTypeCountAtTime(h.client, connectorTypes, connectionIDs, startTime, resourceTypeStrings, EsFetchPageSize)
	} else {
		oldMetricIndexed, err = es.FetchConnectorResourceTypeCountAtTime(h.client, connectorTypes, startTime, resourceTypeStrings, EsFetchPageSize)
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
	for _, resourceType := range resourceTypes {
		for _, tagValue := range resourceType.GetTagsMap()[tagKey] {
			if _, ok := valueCountMap[tagValue]; !ok {
				valueCountMap[tagValue] = currentAndOldCount{}
			}
			v := valueCountMap[tagValue]
			v.current += metricIndexed[strings.ToLower(resourceType.ResourceType)]
			v.old += oldMetricIndexed[strings.ToLower(resourceType.ResourceType)]
			totalCount += metricIndexed[strings.ToLower(resourceType.ResourceType)]
			totalOldCount += oldMetricIndexed[strings.ToLower(resourceType.ResourceType)]
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

// ListResourceTypeTrend godoc
//
//	@Summary		Get resource type trend
//
//	@Description	This API allows users to retrieve a list of resource counts over the course of the specified time frame based on the given input filters
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			tag				query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			servicename		query		[]string		false	"Service names to filter by"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			startTime		query		string			false	"timestamp for start in epoch seconds"
//	@Param			endTime			query		string			false	"timestamp for end in epoch seconds"
//	@Param			datapointCount	query		string			false	"maximum number of datapoints to return, default is 30"
//	@Success		200				{object}	[]inventoryApi.ResourceTypeTrendDatapoint
//	@Router			/inventory/api/v2/resources/trend [get]
func (h *HttpHandler) ListResourceTypeTrend(ctx echo.Context) error {
	var err error
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	serviceNames := httpserver.QueryArrayParam(ctx, "servicename")
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

	resourceTypes, err := h.db.ListFilteredResourceTypes(tagMap, nil, serviceNames, connectorTypes, true)
	if err != nil {
		return err
	}
	resourceTypeStrings := make([]string, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		resourceTypeStrings = append(resourceTypeStrings, resourceType.ResourceType)
	}

	timeToCountMap := make(map[int]int)
	esDatapointCount := int(math.Ceil(endTime.Sub(startTime).Hours() / 24))
	if len(connectionIDs) != 0 {
		timeToCountMap, err = es.FetchConnectionResourceTypeTrendSummaryPage(h.client, connectionIDs, resourceTypeStrings, startTime, endTime, esDatapointCount, EsFetchPageSize)
		if err != nil {
			return err
		}
	} else {
		timeToCountMap, err = es.FetchConnectorResourceTypeTrendSummaryPage(h.client, connectorTypes, resourceTypeStrings, startTime, endTime, esDatapointCount, EsFetchPageSize)
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

// ListResourcesRegionsSummary godoc
//
//	@Summary		List Regions Summary
//	@Description	Returns list of regions resources summary
//	@Security		BearerToken
//	@Tags			resource
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
//	@Router			/inventory/api/v2/resources/regions/summary [get]
func (h *HttpHandler) ListResourcesRegionsSummary(ctx echo.Context) error {
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
	currentLocationDistribution := map[string]int{}

	currentHits, err := es.FetchConnectionLocationsSummaryPage(h.client, connectors, connectionIDs, nil, endTime)
	if err != nil {
		return err
	}
	for _, hit := range currentHits {
		for k, v := range hit.LocationDistribution {
			currentLocationDistribution[k] += v
		}
	}

	oldLocationDistribution := map[string]int{}

	oldtHits, err := es.FetchConnectionLocationsSummaryPage(h.client, connectors, connectionIDs, nil, startTime)
	if err != nil {
		return err
	}
	for _, hit := range oldtHits {
		for k, v := range hit.LocationDistribution {
			oldLocationDistribution[k] += v
		}
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

// ListResourcesRegionsComposition godoc
//
//	@Summary		List resources regions composition
//	@Description	Returns list of top regions per given connector type and connection IDs
//	@Security		BearerToken
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			top				query		int				true	"How many top values to return default is 5"
//	@Param			startTime		query		int				false	"start time in unix seconds - default is now"
//	@Param			endTime			query		int				false	"end time in unix seconds - default is one week ago"
//	@Success		200				{object}	inventoryApi.ListRegionsResourceCountCompositionResponse
//	@Router			/inventory/api/v2/resources/regions/composition [get]
func (h *HttpHandler) ListResourcesRegionsComposition(ctx echo.Context) error {
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	endTime := time.Now()
	if endTimeStr := ctx.QueryParam("endTime"); endTimeStr != "" {
		endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid integer")
		}
		endTime = time.Unix(endTimeUnix, 0)
	}
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr := ctx.QueryParam("startTime"); startTimeStr != "" {
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

	top := 5
	if topStr := ctx.QueryParam("top"); topStr != "" {
		topVal, err := strconv.ParseInt(topStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "top is not a valid integer")
		}
		top = int(topVal)
	}

	currentLocationDistribution := map[string]int{}
	oldLocationDistribution := map[string]int{}

	currentHits, err := es.FetchConnectionLocationsSummaryPage(h.client, connectors, connectionIDs, nil, endTime)
	if err != nil {
		return err
	}
	for _, hit := range currentHits {
		for k, v := range hit.LocationDistribution {
			currentLocationDistribution[k] += v
		}
	}
	oldHits, err := es.FetchConnectionLocationsSummaryPage(h.client, connectors, connectionIDs, nil, startTime)
	if err != nil {
		return err
	}
	for _, hit := range oldHits {
		for k, v := range hit.LocationDistribution {
			oldLocationDistribution[k] += v
		}
	}

	type currentAndOldCount struct {
		current int
		old     int
	}
	valueCountMap := make(map[string]currentAndOldCount)
	totalCount := 0
	totalOldCount := 0
	for region, val := range currentLocationDistribution {
		valueCountMap[region] = currentAndOldCount{current: val, old: oldLocationDistribution[region]}
		totalCount += val
		totalOldCount += oldLocationDistribution[region]
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

	response := inventoryApi.ListRegionsResourceCountCompositionResponse{
		TotalCount:      len(valueCountPairs),
		TotalValueCount: totalCount,
		TopValues:       make(map[string]inventoryApi.CountPair),
		Others:          inventoryApi.CountPair{},
	}

	for i, pair := range valueCountPairs {
		if i < top {
			response.TopValues[pair.str] = inventoryApi.CountPair{
				Count:    pair.counts.current,
				OldCount: pair.counts.old,
			}
		} else {
			response.Others.Count += pair.counts.current
			response.Others.OldCount += pair.counts.old
		}
	}

	return ctx.JSON(http.StatusOK, response)
}

// ListResourcesRegionsTrend godoc
//
//	@Summary		Returns trend of resources count in given regions
//	@Description	Returns list of regions resources summary
//	@Security		BearerToken
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			startTime		query		int				false	"start time in unix seconds - default is now"
//	@Param			endTime			query		int				false	"end time in unix seconds - default is one week ago"
//	@Param			region			query		[]string		false	"Regions to filter by"
//	@Param			datapointCount	query		int				false	"Number of datapoints to return"
//	@Success		200				{object}	[]inventoryApi.ResourceTypeTrendDatapoint
//	@Router			/inventory/api/v2/resources/regions/trend [get]
func (h *HttpHandler) ListResourcesRegionsTrend(ctx echo.Context) error {
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	endTime := time.Now()
	if endTimeStr := ctx.QueryParam("endTime"); endTimeStr != "" {
		endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid integer")
		}
		endTime = time.Unix(endTimeUnix, 0)
	}
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr := ctx.QueryParam("startTime"); startTimeStr != "" {
		startTimeUnix, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "startTime is not a valid integer")
		}
		startTime = time.Unix(startTimeUnix, 0)
	}
	connectionIDs := ctx.QueryParams()["connectionId"]
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
	}
	datapointCount := int(endTime.Sub(startTime).Hours() / 24)
	if datapointCountStr := ctx.QueryParam("datapointCount"); datapointCountStr != "" {
		datapointCountVal, err := strconv.ParseInt(datapointCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid datapointCount")
		}
		datapointCount = int(datapointCountVal)
	}
	regions := ctx.QueryParams()["region"]
	filterRegionsMap := make(map[string]bool)
	for _, region := range regions {
		filterRegionsMap[region] = true
	}
	if len(regions) == 0 {
		filterRegionsMap = nil
	}

	esDatapointCount := int(endTime.Sub(startTime).Hours() / 24)
	timeToCountsMap, err := es.ConnectionResourceTypeRegionsTrendSummaryPage(h.client, connectors, connectionIDs, startTime, endTime, esDatapointCount, EsFetchPageSize)
	if err != nil {
		return err
	}

	apiDatapoints := make([]inventoryApi.ResourceTypeTrendDatapoint, 0, len(timeToCountsMap))
	for timeAt, regionStrToCountMap := range timeToCountsMap {
		count := 0
		for regionStr, regionCount := range regionStrToCountMap {
			if filterRegionsMap != nil && !filterRegionsMap[regionStr] {
				continue
			}
			count += regionCount
		}
		apiDatapoints = append(apiDatapoints, inventoryApi.ResourceTypeTrendDatapoint{
			Count: count,
			Date:  time.Unix(int64(timeAt), 0),
		})
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})
	apiDatapoints = internal.DownSampleResourceTypeTrendDatapoints(apiDatapoints, datapointCount)

	return ctx.JSON(http.StatusOK, apiDatapoints)
}

// ListServiceTags godoc
//
//	@Summary		List resourcetype tags
//	@Description	This API allows users to retrieve a list of possible values for a given tag key for all services.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]string
//	@Router			/inventory/api/v2/services/tag [get]
func (h *HttpHandler) ListServiceTags(ctx echo.Context) error {
	tags, err := h.db.ListServiceTagsKeysWithPossibleValues()
	if err != nil {
		return err
	}
	tags = model.TrimPrivateTags(tags)
	return ctx.JSON(http.StatusOK, tags)
}

// GetServiceTag godoc
//
//	@Summary		Get resourcetype tag
//	@Description	This API allows users to retrieve a list of possible values for a given tag key for all resource types.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			key	path		string	true	"Tag key"
//	@Success		200	{object}	[]string
//	@Router			/inventory/api/v2/services/tag/{key} [get]
func (h *HttpHandler) GetServiceTag(ctx echo.Context) error {
	tagKey := ctx.Param("key")
	if tagKey == "" || strings.HasPrefix(tagKey, model.KaytuPrivateTagPrefix) {
		return echo.NewHTTPError(http.StatusBadRequest, "tag key is invalid")
	}

	tags, err := h.db.GetServiceTagPossibleValues(tagKey)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, tags)
}

// ListServiceMetricsHandler godoc
//
//	@Summary		List services metrics
//	@Description	This API allows users to retrieve a list of services with metrics of each type based on the given input filters.
//	@Security		BearerToken
//	@Tags			inventory
//	@Accept			json
//	@Produce		json
//	@Param			tag				query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			startTime		query		string			false	"timestamp for old values in epoch seconds"
//	@Param			endTime			query		string			false	"timestamp for current values in epoch seconds"
//	@Param			sortBy			query		string			false	"Sort by field - default is count"	Enums(name,count,growth,growth_rate)
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Success		200				{object}	inventoryApi.ListServiceMetricsResponse
//	@Router			/inventory/api/v2/services/metric [get]
func (h *HttpHandler) ListServiceMetricsHandler(ctx echo.Context) error {
	var err error
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	connectorTypes := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
	}
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

	services, err := h.db.ListFilteredServices(tagMap, connectorTypes)
	if err != nil {
		return err
	}
	resourceTypeMap := make(map[string]int)
	for _, service := range services {
		for _, resourceType := range service.ResourceTypes {
			resourceTypeMap[strings.ToLower(resourceType.ResourceType)] = 0
		}
	}
	resourceTypeNames := make([]string, 0, len(resourceTypeMap))
	for resourceType := range resourceTypeMap {
		resourceTypeNames = append(resourceTypeNames, resourceType)
	}

	var resourceTypeCounts map[string]int
	if len(connectionIDs) > 0 {
		resourceTypeCounts, err = es.FetchConnectionResourceTypeCountAtTime(h.client, connectorTypes, connectionIDs, time.Unix(endTime, 0), resourceTypeNames, EsFetchPageSize)
	} else {
		resourceTypeCounts, err = es.FetchConnectorResourceTypeCountAtTime(h.client, connectorTypes, time.Unix(endTime, 0), resourceTypeNames, EsFetchPageSize)
	}
	if err != nil {
		return err
	}
	var oldResourceTypeCounts map[string]int
	if len(connectionIDs) > 0 {
		oldResourceTypeCounts, err = es.FetchConnectionResourceTypeCountAtTime(h.client, connectorTypes, connectionIDs, time.Unix(startTime, 0), resourceTypeNames, EsFetchPageSize)
	} else {
		oldResourceTypeCounts, err = es.FetchConnectorResourceTypeCountAtTime(h.client, connectorTypes, time.Unix(startTime, 0), resourceTypeNames, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	totalCount := 0
	totalOldCount := 0
	apiServices := make([]inventoryApi.Service, 0, len(services))
	for _, service := range services {
		apiService := service.ToApi()
		for _, resourceType := range service.ResourceTypes {
			if resourceTypeCount, ok := resourceTypeCounts[strings.ToLower(resourceType.ResourceType)]; ok {
				cnt := &resourceTypeCount
				apiService.ResourceCount = utils.PAdd(apiService.ResourceCount, cnt)
				totalCount += resourceTypeCount
			}
			if oldResourceTypeCount, ok := oldResourceTypeCounts[strings.ToLower(resourceType.ResourceType)]; ok {
				cnt := &oldResourceTypeCount
				apiService.OldResourceCount = utils.PAdd(apiService.OldResourceCount, cnt)
				totalOldCount += oldResourceTypeCount
			}
		}
		apiServices = append(apiServices, apiService)
	}

	sort.Slice(apiServices, func(i, j int) bool {
		switch sortBy {
		case "name":
			return apiServices[i].ServiceName < apiServices[j].ServiceName
		case "count":
			if apiServices[i].ResourceCount == nil && apiServices[j].ResourceCount == nil {
				break
			}
			if apiServices[i].ResourceCount == nil {
				return false
			}
			if apiServices[j].ResourceCount == nil {
				return true
			}
			if *apiServices[i].ResourceCount != *apiServices[j].ResourceCount {
				return *apiServices[i].ResourceCount > *apiServices[j].ResourceCount
			}
		case "growth":
			diffi := utils.PSub(apiServices[i].ResourceCount, apiServices[i].OldResourceCount)
			diffj := utils.PSub(apiServices[j].ResourceCount, apiServices[j].OldResourceCount)
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
			diffi := utils.PSub(apiServices[i].ResourceCount, apiServices[i].OldResourceCount)
			diffj := utils.PSub(apiServices[j].ResourceCount, apiServices[j].OldResourceCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
				return true
			}
			if apiServices[i].OldResourceCount == nil && apiServices[j].OldResourceCount == nil {
				break
			}
			if apiServices[i].OldResourceCount == nil {
				return true
			}
			if apiServices[j].OldResourceCount == nil {
				return false
			}
			if *apiServices[i].OldResourceCount == 0 && *apiServices[j].OldResourceCount == 0 {
				break
			}
			if *apiServices[i].OldResourceCount == 0 {
				return false
			}
			if *apiServices[j].OldResourceCount == 0 {
				return true
			}
			if float64(*diffi)/float64(*apiServices[i].OldResourceCount) != float64(*diffj)/float64(*apiServices[j].OldResourceCount) {
				return float64(*diffi)/float64(*apiServices[i].OldResourceCount) > float64(*diffj)/float64(*apiServices[j].OldResourceCount)
			}
		}
		return apiServices[i].ServiceName < apiServices[j].ServiceName
	})

	result := inventoryApi.ListServiceMetricsResponse{
		TotalCount:    totalCount,
		TotalOldCount: totalOldCount,
		TotalServices: len(apiServices),
		Services:      utils.Paginate(pageNumber, pageSize, apiServices),
	}
	return ctx.JSON(http.StatusOK, result)
}

// GetServiceMetricsHandler godoc
//
//	@Summary		Get service metrics
//	@Description	This API allows users to retrieve a service with metrics.
//	@Tags			inventory
//	@Security		BearerToken
//	@Accept			json
//	@Produce		json
//	@Param			serviceName		path		string		true	"ServiceName"
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param			startTime		query		string		false	"timestamp for old values in epoch seconds"
//	@Param			endTime			query		string		false	"timestamp for current values in epoch seconds"
//	@Success		200				{object}	inventoryApi.Service
//	@Router			/inventory/api/v2/services/metric/{serviceName} [get]
func (h *HttpHandler) GetServiceMetricsHandler(ctx echo.Context) error {
	var err error
	serviceName := ctx.Param("serviceName")
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
	}
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

	service, err := h.db.GetService(serviceName)
	if err != nil {
		return err
	}
	resourceTypeMap := make(map[string]int)
	for _, resourceType := range service.ResourceTypes {
		resourceTypeMap[strings.ToLower(resourceType.ResourceType)] = 0
	}
	resourceTypeNames := make([]string, 0, len(resourceTypeMap))
	for resourceType := range resourceTypeMap {
		resourceTypeNames = append(resourceTypeNames, resourceType)
	}

	var resourceTypeCounts map[string]int
	if len(connectionIDs) > 0 {
		resourceTypeCounts, err = es.FetchConnectionResourceTypeCountAtTime(h.client, nil, connectionIDs, time.Unix(endTime, 0), resourceTypeNames, EsFetchPageSize)
	} else {
		resourceTypeCounts, err = es.FetchConnectorResourceTypeCountAtTime(h.client, nil, time.Unix(endTime, 0), resourceTypeNames, EsFetchPageSize)
	}
	if err != nil {
		return err
	}
	var oldResourceTypeCounts map[string]int
	if len(connectionIDs) > 0 {
		oldResourceTypeCounts, err = es.FetchConnectionResourceTypeCountAtTime(h.client, nil, connectionIDs, time.Unix(startTime, 0), resourceTypeNames, EsFetchPageSize)
	} else {
		oldResourceTypeCounts, err = es.FetchConnectorResourceTypeCountAtTime(h.client, nil, time.Unix(startTime, 0), resourceTypeNames, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	apiService := service.ToApi()
	for _, resourceType := range service.ResourceTypes {
		if resourceTypeCount, ok := resourceTypeCounts[strings.ToLower(resourceType.ResourceType)]; ok {
			cnt := &resourceTypeCount
			apiService.ResourceCount = utils.PAdd(apiService.ResourceCount, cnt)
		}
		if oldResourceTypeCount, ok := oldResourceTypeCounts[strings.ToLower(resourceType.ResourceType)]; ok {
			cnt := &oldResourceTypeCount
			apiService.OldResourceCount = utils.PAdd(apiService.OldResourceCount, cnt)
		}
	}
	return ctx.JSON(http.StatusOK, apiService)
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

	esDataPointCount := int(endTime.Sub(startTime).Hours() / 24)
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
	resourceCounts, err := es.FetchConnectionResourcesCountAtTime(h.client, connectors, connectionIDs, endTime, EsFetchPageSize)
	if err != nil {
		return err
	}
	for _, hit := range resourceCounts {
		localHit := hit
		if _, ok := res[localHit.SourceID]; !ok {
			res[localHit.SourceID] = inventoryApi.ConnectionData{
				ConnectionID: localHit.SourceID,
			}
		}
		v := res[localHit.SourceID]
		v.Count = utils.PAdd(v.Count, &localHit.ResourceCount)
		if v.LastInventory == nil || v.LastInventory.IsZero() || v.LastInventory.Before(time.UnixMilli(localHit.DescribedAt)) {
			v.LastInventory = utils.GetPointer(time.UnixMilli(localHit.DescribedAt))
		}
		res[localHit.SourceID] = v
	}
	oldResourceCount, err := es.FetchConnectionResourcesCountAtTime(h.client, connectors, connectionIDs, startTime, EsFetchPageSize)
	if err != nil {
		return err
	}
	for _, hit := range oldResourceCount {
		localHit := hit
		if _, ok := res[localHit.SourceID]; !ok {
			res[localHit.SourceID] = inventoryApi.ConnectionData{
				ConnectionID:  localHit.SourceID,
				LastInventory: nil,
			}
		}
		v := res[localHit.SourceID]
		v.OldCount = utils.PAdd(v.OldCount, &localHit.ResourceCount)
		if v.LastInventory == nil || v.LastInventory.IsZero() || v.LastInventory.Before(time.UnixMilli(localHit.DescribedAt)) {
			v.LastInventory = utils.GetPointer(time.UnixMilli(localHit.DescribedAt))
		}
		res[localHit.SourceID] = v
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

	resourceCounts, err := es.FetchConnectionResourcesCountAtTime(h.client, nil, []string{connectionId}, endTime, EsFetchPageSize)
	for _, hit := range resourceCounts {
		if hit.SourceID != connectionId {
			continue
		}
		localHit := hit
		res.Count = utils.PAdd(res.Count, &localHit.ResourceCount)
		if res.LastInventory == nil || res.LastInventory.IsZero() || res.LastInventory.Before(time.UnixMilli(localHit.DescribedAt)) {
			res.LastInventory = utils.GetPointer(time.UnixMilli(localHit.DescribedAt))
		}
	}

	oldResourceCounts, err := es.FetchConnectionResourcesCountAtTime(h.client, nil, []string{connectionId}, startTime, EsFetchPageSize)
	for _, hit := range oldResourceCounts {
		if hit.SourceID != connectionId {
			continue
		}
		localHit := hit
		res.OldCount = utils.PAdd(res.OldCount, &localHit.ResourceCount)
		if res.LastInventory == nil || res.LastInventory.IsZero() || res.LastInventory.Before(time.UnixMilli(localHit.DescribedAt)) {
			res.LastInventory = utils.GetPointer(time.UnixMilli(localHit.DescribedAt))
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

// ListServiceSummaries godoc
//
//	@Deprecated		use /inventory/api/v2/services/metric instead
//	@Summary		List Cloud Services Summary
//	@Description	Retrieves list of summaries of the services including the number of them and the API filters and a list of services with more details. Including connector and the resource counts.
//	@Security		BearerToken
//	@Tags			services
//	@Accept			json
//	@Produce		json
//	@Param			connectionId	query		[]string	false	"filter: Connection ID"
//	@Param			connector		query		[]string	false	"filter: Connector"
//	@Param			tag				query		[]string	false	"filter: tag for the services"
//	@Param			endTime			query		string		false	"time for resource count in epoch seconds"
//	@Param			pageSize		query		int			false	"page size - default is 20"
//	@Param			pageNumber		query		int			false	"page number - default is 1"
//	@Param			sortBy			query		string		false	"column to sort by - default is resourcecount"	Enums(servicecode,resourcecount)
//	@Success		200				{object}	inventoryApi.ListServiceSummariesResponse
//	@Router			/inventory/api/v2/services/summary [get]
func (h *HttpHandler) ListServiceSummaries(ctx echo.Context) error {
	var err error
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))

	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
	}
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))

	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid epoch time")
		}
	}

	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	sortBy := ctx.QueryParam("sortBy")
	if sortBy == "" {
		sortBy = "resourcecount"
	}

	resourceTypeMap := make(map[string]int64)
	services, err := h.db.ListFilteredServices(tagMap, connectors)
	if err != nil {
		return err
	}

	for _, service := range services {
		for _, resourceType := range service.ResourceTypes {
			resourceTypeMap[strings.ToLower(resourceType.ResourceType)] = 0
		}
	}
	resourceTypeNames := make([]string, 0, len(resourceTypeMap))
	for resourceTypeName := range resourceTypeMap {
		resourceTypeNames = append(resourceTypeNames, resourceTypeName)
	}

	var resourceTypeCounts map[string]int
	if len(connectionIDs) > 0 {
		resourceTypeCounts, err = es.FetchConnectionResourceTypeCountAtTime(h.client, connectors, connectionIDs, time.Unix(endTime, 0), resourceTypeNames, EsFetchPageSize)
	} else {
		resourceTypeCounts, err = es.FetchConnectorResourceTypeCountAtTime(h.client, connectors, time.Unix(endTime, 0), resourceTypeNames, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	serviceSummaries := make([]inventoryApi.ServiceSummary, 0, len(services))
	for _, service := range services {
		serviceSummary := inventoryApi.ServiceSummary{
			Connector:     service.Connector,
			ServiceLabel:  service.ServiceLabel,
			ServiceName:   service.ServiceName,
			ResourceCount: nil,
		}
		for _, resourceType := range service.ResourceTypes {
			if resourceTypeCount, ok := resourceTypeCounts[strings.ToLower(resourceType.ResourceType)]; ok {
				rtC := resourceTypeCount
				serviceSummary.ResourceCount = utils.PAdd(serviceSummary.ResourceCount, &rtC)
			}
		}
		serviceSummaries = append(serviceSummaries, serviceSummary)
	}

	// remove services with no resource count
	serviceSummariesFiltered := make([]inventoryApi.ServiceSummary, 0, len(serviceSummaries))
	for _, serviceSummary := range serviceSummaries {
		if serviceSummary.ResourceCount != nil {
			serviceSummariesFiltered = append(serviceSummariesFiltered, serviceSummary)
		}
	}
	serviceSummaries = serviceSummariesFiltered

	sort.Slice(serviceSummaries, func(i, j int) bool {
		switch sortBy {
		case "resourcecount":
			if serviceSummaries[i].ResourceCount == nil {
				return false
			}
			if serviceSummaries[j].ResourceCount == nil {
				return true
			}
			if *serviceSummaries[i].ResourceCount != *serviceSummaries[j].ResourceCount {
				return *serviceSummaries[i].ResourceCount > *serviceSummaries[j].ResourceCount
			}
		case "servicecode":
			return serviceSummaries[i].ServiceName < serviceSummaries[j].ServiceName
		}
		return serviceSummaries[i].ServiceName < serviceSummaries[j].ServiceName
	})

	res := inventoryApi.ListServiceSummariesResponse{
		TotalCount: len(serviceSummaries),
		Services:   utils.Paginate(pageNumber, pageSize, serviceSummaries),
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetServiceSummary godoc
//
//	@Deprecated		use /inventory/api/v2/services/metric/{serviceName} instead
//	@Summary		Get Cloud Service Summary
//	@Description	Retrieves Cloud Service Summary for the specified service name. Including connector, the resource counts.
//	@Security		BearerToken
//	@Tags			services
//	@Accepts		json
//	@Produce		json
//	@Param			serviceName	path		string		true	"Service Name"
//	@Param			connectorId	query		[]string	false	"filter: connectorId"
//	@Param			connector	query		[]string	false	"filter: connector"
//	@Param			endTime		query		string		true	"time for resource count in epoch seconds"
//	@Success		200			{object}	inventoryApi.ServiceSummary
//	@Router			/inventory/api/v2/services/summary/{serviceName} [get]
func (h *HttpHandler) GetServiceSummary(ctx echo.Context) error {
	var err error
	serviceName := ctx.Param("serviceName")
	if serviceName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "service_name is required")
	}

	connectionIDs := httpserver.QueryArrayParam(ctx, "connectorId")
	if len(connectionIDs) > 20 {
		return ctx.JSON(http.StatusBadRequest, "too many connection IDs")
	}
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))

	endTimeStr := ctx.QueryParam("endTime")
	endTime := time.Now().Unix()
	if endTimeStr != "" {
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, "endTime is not a valid epoch time")
		}
	}

	resourceTypeMap := make(map[string]int64)
	service, err := h.db.GetService(serviceName)
	if err != nil {
		return err
	}

	for _, resourceType := range service.ResourceTypes {
		resourceTypeMap[strings.ToLower(resourceType.ResourceType)] = 0
	}

	resourceTypeNames := make([]string, 0, len(resourceTypeMap))
	for resourceTypeName := range resourceTypeMap {
		resourceTypeNames = append(resourceTypeNames, resourceTypeName)
	}

	var resourceTypeCounts map[string]int
	if len(connectionIDs) > 0 {
		resourceTypeCounts, err = es.FetchConnectionResourceTypeCountAtTime(h.client, connectors, connectionIDs, time.Unix(endTime, 0), resourceTypeNames, EsFetchPageSize)
	} else {
		resourceTypeCounts, err = es.FetchConnectorResourceTypeCountAtTime(h.client, connectors, time.Unix(endTime, 0), resourceTypeNames, EsFetchPageSize)
	}
	if err != nil {
		return err
	}

	serviceSummary := inventoryApi.ServiceSummary{
		Connector:     service.Connector,
		ServiceLabel:  service.ServiceLabel,
		ServiceName:   service.ServiceName,
		ResourceCount: nil,
	}
	for _, resourceType := range service.ResourceTypes {
		if resourceTypeCount, ok := resourceTypeCounts[strings.ToLower(resourceType.ResourceType)]; ok {
			serviceSummary.ResourceCount = utils.PAdd(serviceSummary.ResourceCount, &resourceTypeCount)
		}
	}

	return ctx.JSON(http.StatusOK, serviceSummary)
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

	esDataPointCount := int(endTime.Sub(startTime).Hours() / 24)
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

// GetResource godoc
//
//	@Summary		Get details of a Resource
//	@Description	Getting resource details by id and resource type
//	@Security		BearerToken
//	@Tags			resource
//	@Accepts		json
//	@Produce		json
//	@Param			request	body		inventoryApi.GetResourceRequest	true	"Request Body"
//	@Success		200		{object}	map[string]string
//	@Router			/inventory/api/v1/resource [post]
func (h *HttpHandler) GetResource(ctx echo.Context) error {
	var req inventoryApi.GetResourceRequest
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

	var response inventoryApi.GenericQueryResponse
	err = h.client.Search(ctx.Request().Context(), index, string(queryBytes), &response)
	if err != nil {
		return err
	}

	var sourceMap map[string]interface{}
	for _, hit := range response.Hits.Hits {
		sourceMap = hit.Source
	}

	var cells map[string]*proto.Column
	pluginProvider := steampipe.ExtractPlugin(req.ResourceType)
	if pluginProvider == steampipe.SteampipePluginAWS {
		pluginTableName := awsSteampipe.ExtractTableName(req.ResourceType)
		desc, err := steampipe.ConvertToDescription(req.ResourceType, sourceMap, awsSteampipe.AWSDescriptionMap)
		if err != nil {
			return err
		}

		cells, err = awsSteampipe.AWSDescriptionToRecord(desc, pluginTableName)
		if err != nil {
			return err
		}
	} else if pluginProvider == steampipe.SteampipePluginAzure || pluginProvider == steampipe.SteampipePluginAzureAD {
		pluginTableName := azureSteampipe.ExtractTableName(req.ResourceType)
		desc, err := steampipe.ConvertToDescription(req.ResourceType, sourceMap, azureSteampipe.AzureDescriptionMap)
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

	queries, err := h.db.GetQueriesWithFilters(search, req.Labels, req.ProviderFilter)
	if err != nil {
		return err
	}

	var result []inventoryApi.SmartQueryItem
	for _, item := range queries {
		category := ""

		result = append(result, inventoryApi.SmartQueryItem{
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
//	@Param			request	body		inventoryApi.ListQueryRequest	true	"Request Body"
//	@Success		200		{object}	int
//	@Router			/inventory/api/v1/query/count [get]
func (h *HttpHandler) CountQueries(ctx echo.Context) error {
	var req inventoryApi.ListQueryRequest
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
//	@Param			queryId	path		string							true	"QueryID"
//	@Param			request	body		inventoryApi.RunQueryRequest	true	"Request Body"
//	@Param			accept	header		string							true	"Accept header"	Enums(application/json,text/csv)
//	@Success		200		{object}	inventoryApi.RunQueryResponse
//	@Router			/inventory/api/v1/query/{queryId} [post]
func (h *HttpHandler) RunQuery(ctx echo.Context) error {
	var req inventoryApi.RunQueryRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	queryId := ctx.Param("queryId")

	if accepts := ctx.Request().Header.Get("accept"); accepts != "" {
		mediaType, _, err := mime.ParseMediaType(accepts)
		if err == nil && mediaType == "text/csv" {
			req.Page = inventoryApi.Page{
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
//	@Param			connector	path		string	true	"Connector"
//	@Success		200			{object}	[]inventoryApi.LocationByProviderResponse
//	@Router			/inventory/api/v1/locations/{connector} [get]
func (h *HttpHandler) GetLocations(ctx echo.Context) error {
	connectorStr := ctx.Param("connector")
	connector, _ := source.ParseType(connectorStr)

	var locations []inventoryApi.LocationByProviderResponse

	if connectorStr == "all" || connector == source.CloudAWS {
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
			locations = append(locations, inventoryApi.LocationByProviderResponse{
				Name: regionName,
			})
		}
	}

	if connectorStr == "all" || connector == source.CloudAzure {
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
			locations = append(locations, inventoryApi.LocationByProviderResponse{
				Name: regionName,
			})
		}
	}

	return ctx.JSON(http.StatusOK, locations)
}

// GetResources godoc
//
//	@Summary		Get resources
//	@Description	Getting all cloud providers resources by filters
//	@Security		BearerToken
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			request	body		inventoryApi.GetResourcesRequest	true	"Request Body"
//	@Success		200		{object}	inventoryApi.GetResourcesResponse
//	@Router			/inventory/api/v1/resources [post]
func (h *HttpHandler) GetResources(ctx echo.Context) error {
	var req inventoryApi.GetResourcesRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	h.logger.Info("GetResources got called", zap.Any("req", req))

	resourceTypes, err := h.db.ListFilteredResourceTypes(nil, req.Filters.ResourceType, req.Filters.Service, req.Filters.Connectors, true)
	if err != nil {
		h.logger.Error("failed to list resource types", zap.Error(err))
		return err
	}
	req.Filters.ResourceType = make([]string, 0, len(resourceTypes))
	resourceTypeMap := make(map[string]ResourceType)
	for _, resourceType := range resourceTypes {
		req.Filters.ResourceType = append(req.Filters.ResourceType, strings.ToLower(resourceType.ResourceType))
		resourceTypeMap[strings.ToLower(resourceType.ResourceType)] = resourceType
	}

	h.logger.Info("list of resourceType", zap.Any("resourceType", req.Filters.ResourceType))

	res, err := inventoryApi.QueryResources(ctx.Request().Context(), h.client, &req, req.Filters.Connectors)
	if err != nil {
		return err
	}

	h.logger.Info("list of resources", zap.Int("len", len(res.AllResources)))

	var connections []onboardApi.Connection
	if len(req.Filters.ConnectionID) == 0 {
		connections, err = h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), req.Filters.Connectors)
		if err != nil {
			h.logger.Error("Failed to list sources", zap.Error(err))
			return err
		}
	} else {
		connections, err = h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), req.Filters.ConnectionID)
		if err != nil {
			h.logger.Error("Failed to get sources", zap.Error(err))
			return err
		}
	}
	connectionsMap := map[string]onboardApi.Connection{}
	for _, connection := range connections {
		connectionsMap[connection.ID.String()] = connection
	}
	for idx := range res.AllResources {
		id := res.AllResources[idx].ConnectionID
		res.AllResources[idx].ProviderConnectionID = connectionsMap[id].ConnectionID
		res.AllResources[idx].ProviderConnectionName = connectionsMap[id].ConnectionName
		res.AllResources[idx].ResourceTypeLabel = resourceTypeMap[res.AllResources[idx].ResourceType].ResourceLabel
	}

	result := inventoryApi.GetResourcesResponse{
		Resources:  res.AllResources,
		TotalCount: res.TotalCount,
	}

	return ctx.JSON(http.StatusOK, result)
}

// CountResources godoc
//
//	@Deprecated
//	@Summary		Count resources
//	@Description	Number of all resources
//	@Security		BearerToken
//	@Tags			resource
//	@Accept			json
//	@Produce		json,text/csv
//	@Success		200	{object}	int64
//	@Router			/inventory/api/v2/resources/count [get]
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

// GetResourcesFilters godoc
//
//	@Summary		Get resource filters
//	@Description	Getting resource filters by filters.
//	@Security		BearerToken
//	@Tags			resource
//	@Accept			json
//	@Produce		json,text/csv
//	@Param			request	body		inventoryApi.GetFiltersRequest	true	"Request Body"
//	@Param			common	query		string							false	"Common filter"	Enums(true,false,all)
//	@Success		200		{object}	inventoryApi.GetFiltersResponse
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

	var req inventoryApi.GetFiltersRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	query, err := es.BuildFilterQuery(req.Query, req.Filters, common)
	if err != nil {
		return err
	}

	var response es.LookupResourceAggregationResponse
	err = h.client.Search(context.Background(), summarizer.InventorySummaryIndex,
		query, &response)
	if err != nil {
		return err
	}

	resp := inventoryApi.GetFiltersResponse{}
	for _, item := range response.Aggregations.ResourceTypeFilter.Buckets {
		resp.Filters.ResourceType = append(resp.Filters.ResourceType, inventoryApi.ResourceTypeFull{
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
	if !inventoryApi.FilterIsEmpty(req.Filters.Service) {
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
	if !inventoryApi.FilterIsEmpty(req.Filters.Category) {
		categoriesMap := make(map[string]string)
		for _, category := range req.Filters.Category {
			if _, ok := resp.Filters.Category[category]; ok {
				categoriesMap[category] = resp.Filters.Category[category]
			}
		}
		resp.Filters.Category = categoriesMap
	}

	var connectionIDs []string
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
		resp.Filters.Connections = append(resp.Filters.Connections, inventoryApi.ConnectionFull{
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
	req *inventoryApi.RunQueryRequest) (*inventoryApi.RunQueryResponse, error) {

	var err error
	lastIdx := (req.Page.No - 1) * req.Page.Size

	if req.Sorts == nil || len(req.Sorts) == 0 {
		req.Sorts = []inventoryApi.SmartQuerySortItem{
			{
				Field:     "1",
				Direction: inventoryApi.DirectionAscending,
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

func (h *HttpHandler) GetInsightResultByJobId(ctx echo.Context) error {
	jobIdStr := ctx.Param("jobId")
	jobId, err := strconv.ParseUint(jobIdStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	job, err := h.schedulerClient.GetInsightJobById(httpclient.FromEchoContext(ctx), jobIdStr)
	if err != nil {
		return err
	}
	if job.ID == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "No job found")
	}
	insightResult, err := es.FetchInsightByJobIDAndInsightID(h.client, uint(jobId), job.InsightID)
	if err != nil {
		return err
	}

	if insightResult == nil {
		return echo.NewHTTPError(http.StatusNotFound, "no data for insight found")
	}

	return echo.NewHTTPError(http.StatusOK, *insightResult)
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

// ListServiceMetadata godoc
//
//	@Summary		Get List of Cloud Services
//	@Description	Gets a list of all workspace cloud services and their metadata, and list of resource types.
//	@Description	The results could be filtered by tags.
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			connector	query		[]source.Type	false	"Connector"
//	@Param			tag			query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			pageSize	query		int				false	"page size - default is 20"
//	@Param			pageNumber	query		int				false	"page number - default is 1"
//	@Success		200			{object}	inventoryApi.ListServiceMetadataResponse
//	@Router			/inventory/api/v2/metadata/services [get]
func (h *HttpHandler) ListServiceMetadata(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	services, err := h.db.ListFilteredServices(tagMap, connectors)
	if err != nil {
		return err
	}

	var serviceMetadata []inventoryApi.Service
	for _, service := range services {
		serviceMetadata = append(serviceMetadata, service.ToApi())
	}

	sort.Slice(serviceMetadata, func(i, j int) bool {
		return serviceMetadata[i].ServiceName < serviceMetadata[j].ServiceName
	})

	result := inventoryApi.ListServiceMetadataResponse{
		TotalServiceCount: len(serviceMetadata),
		Services:          utils.Paginate(pageNumber, pageSize, serviceMetadata),
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetServiceMetadata godoc
//
//	@Summary		Get Cloud Service Details
//	@Description	Gets a single cloud service details and its metadata and list of resource types.
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			serviceName	path		string	true	"ServiceName"
//	@Success		200			{object}	inventoryApi.Service
//	@Router			/inventory/api/v2/metadata/services/{serviceName} [get]
func (h *HttpHandler) GetServiceMetadata(ctx echo.Context) error {
	serviceName := ctx.Param("serviceName")

	service, err := h.db.GetService(serviceName)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, service.ToApi())
}

// ListResourceTypeMetadata godoc
//
//	@Summary		Get List of Resource Types
//	@Description	Gets a list of all resource types in workspace and their metadata including service name.
//	@Description	The results could be filtered by provider name and service name.
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			connector	query		[]source.Type	true	"Filter by Connector"
//	@Param			service		query		[]string		false	"Filter by service name"
//	@Param			tag			query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			pageSize	query		int				false	"page size - default is 20"
//	@Param			pageNumber	query		int				false	"page number - default is 1"
//	@Success		200			{object}	inventoryApi.ListResourceTypeMetadataResponse
//	@Router			/inventory/api/v2/metadata/resourcetype [get]
func (h *HttpHandler) ListResourceTypeMetadata(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	serviceNames := httpserver.QueryArrayParam(ctx, "service")
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	resourceTypes, err := h.db.ListFilteredResourceTypes(tagMap, nil, serviceNames, connectors, true)
	if err != nil {
		return err
	}

	var resourceTypeMetadata []inventoryApi.ResourceType

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
			insightList, err := h.complianceClient.ListInsightsMetadata(httpclient.FromEchoContext(ctx), []source.Type{resourceType.Connector})
			if err != nil {
				return err
			}
			for _, insightEntity := range insightList {
				for _, insightTable := range insightEntity.Query.ListOfTables {
					if insightTable == table {
						insightTableCount++
						break
					}
				}
			}
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

// GetResourceTypeMetadata godoc
//
//	@Summary		Get Resource Type
//	@Description	Get a single resource type metadata and its details including service name and insights list. Specified by resource type name.
//	@Security		BearerToken
//	@Tags			metadata
//	@Produce		json
//	@Param			resourceType	path		string	true	"ResourceType"
//	@Success		200				{object}	inventoryApi.ResourceType
//	@Router			/inventory/api/v2/metadata/resourcetype/{resourceType} [get]
func (h *HttpHandler) GetResourceTypeMetadata(ctx echo.Context) error {
	resourceTypeStr := ctx.Param("resourceType")
	resourceType, err := h.db.GetResourceType(resourceTypeStr)
	if err != nil {
		return err
	}

	result := resourceType.ToApi()
	var table string
	switch resourceType.Connector {
	case source.CloudAWS:
		table = awsSteampipe.ExtractTableName(resourceType.ResourceType)
	case source.CloudAzure:
		table = azureSteampipe.ExtractTableName(resourceType.ResourceType)
	}
	if table != "" {
		insightTables := make([]uint, 0)
		insightList, err := h.complianceClient.ListInsightsMetadata(httpclient.FromEchoContext(ctx), []source.Type{resourceType.Connector})
		if err != nil {
			return err
		}
		for _, insightEntity := range insightList {
			for _, insightTable := range insightEntity.Query.ListOfTables {
				if insightTable == table {
					insightTables = append(insightTables, insightEntity.ID)
					break
				}
			}
		}
		result.Insights = insightTables
		result.InsightsCount = utils.GetPointerOrNil(len(insightTables))

		// TODO: add compliance list & count

		switch resourceType.Connector {
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
