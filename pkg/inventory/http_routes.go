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

	"github.com/kaytu-io/kaytu-util/pkg/model"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
	"gorm.io/gorm"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/internal"
	"gitlab.com/keibiengine/keibi-engine/pkg/utils"

	api3 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	insight "gitlab.com/keibiengine/keibi-engine/pkg/insight/es"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/es"

	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/turbot/steampipe-plugin-sdk/v4/grpc/proto"

	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

const EsFetchPageSize = 10000

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.GET("/locations/:connector", httpserver.AuthorizeHandler(h.GetLocations, api3.ViewerRole))

	v1.POST("/resources", httpserver.AuthorizeHandler(h.GetAllResources, api3.ViewerRole))
	v1.POST("/resources/azure", httpserver.AuthorizeHandler(h.GetAzureResources, api3.ViewerRole))
	v1.POST("/resources/aws", httpserver.AuthorizeHandler(h.GetAWSResources, api3.ViewerRole))
	v1.GET("/resources/count", httpserver.AuthorizeHandler(h.CountResources, api3.ViewerRole))
	v1.POST("/resources/filters", httpserver.AuthorizeHandler(h.GetResourcesFilters, api3.ViewerRole))
	v1.POST("/resource", httpserver.AuthorizeHandler(h.GetResource, api3.ViewerRole))

	v1.GET("/resources/top/regions", httpserver.AuthorizeHandler(h.GetTopRegionsByResourceCount, api3.ViewerRole))
	v1.GET("/resources/regions", httpserver.AuthorizeHandler(h.GetRegionsByResourceCount, api3.ViewerRole))

	v1.GET("/query", httpserver.AuthorizeHandler(h.ListQueries, api3.ViewerRole))
	v1.GET("/query/count", httpserver.AuthorizeHandler(h.CountQueries, api3.ViewerRole))
	v1.POST("/query/:queryId", httpserver.AuthorizeHandler(h.RunQuery, api3.EditorRole))

	v2 := e.Group("/api/v2")

	resourcesV2 := v2.Group("/resources")
	resourcesV2.GET("/tag", httpserver.AuthorizeHandler(h.ListResourceTypeTags, api3.ViewerRole))
	resourcesV2.GET("/tag/:key", httpserver.AuthorizeHandler(h.GetResourceTypeTag, api3.ViewerRole))
	resourcesV2.GET("/metric", httpserver.AuthorizeHandler(h.ListResourceTypeMetricsHandler, api3.ViewerRole))
	resourcesV2.GET("/metric/:resourceType", httpserver.AuthorizeHandler(h.GetResourceTypeMetricsHandler, api3.ViewerRole))
	resourcesV2.GET("/composition/:key", httpserver.AuthorizeHandler(h.ListResourceTypeComposition, api3.ViewerRole))
	resourcesV2.GET("/trend", httpserver.AuthorizeHandler(h.ListResourceTypeTrend, api3.ViewerRole))

	servicesV2 := v2.Group("/services")
	servicesV2.GET("/tag", httpserver.AuthorizeHandler(h.ListServiceTags, api3.ViewerRole))
	servicesV2.GET("/tag/:key", httpserver.AuthorizeHandler(h.GetServiceTag, api3.ViewerRole))
	servicesV2.GET("/metric", httpserver.AuthorizeHandler(h.ListServiceMetricsHandler, api3.ViewerRole))
	servicesV2.GET("/metric/:serviceName", httpserver.AuthorizeHandler(h.GetServiceMetricsHandler, api3.ViewerRole))
	servicesV2.GET("/summary", httpserver.AuthorizeHandler(h.ListServiceSummaries, api3.ViewerRole))
	servicesV2.GET("/summary/:serviceName", httpserver.AuthorizeHandler(h.GetServiceSummary, api3.ViewerRole))

	costV2 := v2.Group("/cost")
	costV2.GET("/metric", httpserver.AuthorizeHandler(h.ListCostMetricsHandler, api3.ViewerRole))
	costV2.GET("/composition", httpserver.AuthorizeHandler(h.ListCostComposition, api3.ViewerRole))
	costV2.GET("/trend", httpserver.AuthorizeHandler(h.GetCostTrend, api3.ViewerRole))

	connectionsV2 := v2.Group("/connections")
	connectionsV2.GET("/data", httpserver.AuthorizeHandler(h.ListConnectionsData, api3.ViewerRole))
	connectionsV2.GET("/data/:connectionId", httpserver.AuthorizeHandler(h.GetConnectionData, api3.ViewerRole))

	insightsV2 := v2.Group("/insights")
	insightsV2.GET("", httpserver.AuthorizeHandler(h.ListInsightResults, api3.ViewerRole))
	insightsV2.GET("/job/:jobId", httpserver.AuthorizeHandler(h.GetInsightResultByJobId, api3.ViewerRole))
	insightsV2.GET("/:insightId/trend", httpserver.AuthorizeHandler(h.GetInsightTrendResults, api3.ViewerRole))
	insightsV2.GET("/:insightId", httpserver.AuthorizeHandler(h.GetInsightResult, api3.ViewerRole))

	metadata := v2.Group("/metadata")
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

func (h *HttpHandler) getConnectorTypesFromConnectionIDs(ctx echo.Context, connectorTypes []source.Type, connectionIDs []string) ([]source.Type, error) {
	connections, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), connectionIDs)
	if err != nil {
		return nil, err
	}

	enabledConnectors := make(map[source.Type]bool)
	for _, connection := range connections {
		enabledConnectors[connection.Connector] = true
	}
	filteredConnectorType := make([]source.Type, 0)
	for _, connectorType := range connectorTypes {
		if enabledConnectors[connectorType] {
			filteredConnectorType = append(filteredConnectorType, connectorType)
		}
	}

	if len(connectorTypes) == 0 {
		for connectorType := range enabledConnectors {
			filteredConnectorType = append(filteredConnectorType, connectorType)
		}
	}

	return filteredConnectorType, nil
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
//	@Success	200				{object}	[]api.LocationResponse
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

	var response []api.LocationResponse
	for region, count := range locationDistribution {
		cnt := count
		response = append(response, api.LocationResponse{
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
//	@Success	200				{object}	[]api.LocationResponse
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

	var response []api.LocationResponse
	for region, count := range locationDistribution {
		cnt := count
		res := api.LocationResponse{
			Location:      region,
			ResourceCount: &cnt,
		}
		if oldLocationDistribution[region] != 0 {
			res.ResourceCountChangePercent = utils.GetPointer((float64(count) - float64(oldLocationDistribution[region])) / float64(oldLocationDistribution[region]) * 100)
		}
		response = append(response, res)
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
	tags, err := h.db.ListResourceTypeTagsKeysWithPossibleValues(connectorTypes, utils.GetPointer(true))
	if err != nil {
		return err
	}
	tags = model.TrimPrivateTags(tags)
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

	tags, err := h.db.GetResourceTypeTagPossibleValues(tagKey, connectorTypes, utils.GetPointer(true))
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, tags)
}

func (h *HttpHandler) ListResourceTypeMetrics(tagMap map[string][]string, serviceNames []string, connectorTypes []source.Type, connectionIDs []string, timeAt int64) (int, []api.ResourceType, error) {
	resourceTypes, err := h.db.ListFilteredResourceTypes(tagMap, serviceNames, connectorTypes, true)
	if err != nil {
		return 0, nil, err
	}
	resourceTypeStrings := make([]string, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		resourceTypeStrings = append(resourceTypeStrings, resourceType.ResourceType)
	}

	var metricIndexed map[string]int
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionResourceTypeCountAtTime(h.client, connectorTypes, connectionIDs, time.Unix(timeAt, 0), resourceTypeStrings, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorResourceTypeCountAtTime(h.client, connectorTypes, time.Unix(timeAt, 0), resourceTypeStrings, EsFetchPageSize)
	}
	if err != nil {
		return 0, nil, err
	}

	apiResourceTypes := make([]api.ResourceType, 0, len(resourceTypes))
	totalCount := 0
	for _, resourceType := range resourceTypes {
		apiResourceType := resourceType.ToApi()
		if count, ok := metricIndexed[strings.ToLower(resourceType.ResourceType)]; ok {
			apiResourceType.Count = &count
			totalCount += count
		}
		apiResourceTypes = append(apiResourceTypes, apiResourceType)
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
//	@Param			sortBy			query		string			false	"Sort by field - default is count"	Enums(name,count)
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Success		200				{object}	api.ListResourceTypeMetricsResponse
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
			apiResourceType.OldCount = oldApiResourceType.Count
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

func (h *HttpHandler) GetResourceTypeMetric(resourceTypeStr string, connectionIDs []string, timeAt int64) (*api.ResourceType, error) {
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
//	@Success		200				{object}	api.ResourceType
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
//	@Param			time			query		string			false	"timestamp for resource count in epoch seconds"
//	@Success		200				{object}	api.ListResourceTypeCompositionResponse
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
	timeStr := ctx.QueryParam("time")
	timeAt := time.Now().Unix()
	if timeStr != "" {
		timeAt, err = strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
	}

	resourceTypes, err := h.db.ListFilteredResourceTypes(map[string][]string{tagKey: nil}, nil, connectorTypes, true)
	if err != nil {
		return err
	}
	resourceTypeStrings := make([]string, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		resourceTypeStrings = append(resourceTypeStrings, resourceType.ResourceType)
	}

	var metricIndexed map[string]int
	if len(connectionIDs) > 0 {
		metricIndexed, err = es.FetchConnectionResourceTypeCountAtTime(h.client, connectorTypes, connectionIDs, time.Unix(timeAt, 0), resourceTypeStrings, EsFetchPageSize)
	} else {
		metricIndexed, err = es.FetchConnectorResourceTypeCountAtTime(h.client, connectorTypes, time.Unix(timeAt, 0), resourceTypeStrings, EsFetchPageSize)
	}
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
//	@Success		200				{object}	[]api.ResourceTypeTrendDatapoint
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

	resourceTypes, err := h.db.ListFilteredResourceTypes(tagMap, serviceNames, connectorTypes, true)
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
		timeToCountMap, err = es.FetchProviderResourceTypeTrendSummaryPage(h.client, connectorTypes, resourceTypeStrings, startTime, endTime, esDatapointCount, EsFetchPageSize)
		if err != nil {
			return err
		}
	}

	apiDatapoints := make([]api.ResourceTypeTrendDatapoint, 0, len(timeToCountMap))
	for timeAt, count := range timeToCountMap {
		apiDatapoints = append(apiDatapoints, api.ResourceTypeTrendDatapoint{Count: count, Date: time.UnixMilli(int64(timeAt))})
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})
	apiDatapoints = internal.DownSampleResourceTypeTrendDatapoints(apiDatapoints, int(datapointCount))

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
//	@Param			sortBy			query		string			false	"Sort by field - default is count"	Enums(name,count)
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Success		200				{object}	api.ListServiceMetricsResponse
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
	if sortBy != "name" && sortBy != "count" {
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
	apiServices := make([]api.Service, 0, len(services))
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
			}
		}
		apiServices = append(apiServices, apiService)
	}

	switch sortBy {
	case "name":
		sort.Slice(apiServices, func(i, j int) bool {
			return apiServices[i].ServiceName < apiServices[j].ServiceName
		})
	case "count":
		sort.Slice(apiServices, func(i, j int) bool {
			if apiServices[i].ResourceCount == nil {
				return false
			}
			if apiServices[j].ResourceCount == nil {
				return true
			}
			if *apiServices[i].ResourceCount == *apiServices[j].ResourceCount {
				return apiServices[i].ServiceName < apiServices[j].ServiceName
			}
			return *apiServices[i].ResourceCount > *apiServices[j].ResourceCount
		})
	}

	result := api.ListServiceMetricsResponse{
		TotalCount:    totalCount,
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
//	@Accept			json
//	@Produce		json
//	@Param			serviceName		path		string		true	"ServiceName"
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param			startTime		query		string		false	"timestamp for old values in epoch seconds"
//	@Param			endTime			query		string		false	"timestamp for current values in epoch seconds"
//	@Success		200				{object}	api.Service
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
//	@Param			sortBy			query		string			false	"Sort by field - default is cost"	Enums(dimension,cost)
//	@Param			pageSize		query		int				false	"page size - default is 20"
//	@Param			pageNumber		query		int				false	"page number - default is 1"
//	@Success		200				{object}	api.ListCostMetricsResponse
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
	if sortBy != "dimension" && sortBy != "cost" {
		return ctx.JSON(http.StatusBadRequest, "invalid sortBy value")
	}

	costHits, err := es.FetchDailyCostHistoryByServicesBetween(h.client, connectionIDs, connectorTypes, nil, time.Unix(startTime, 0), time.Unix(endTime, 0), EsFetchPageSize)
	if err != nil {
		return err
	}
	costMetricMap := make(map[string]api.CostMetric)
	for connector, serviceToCostMap := range costHits {
		for dimension, costVal := range serviceToCostMap {
			connectorTyped, _ := source.ParseType(connector)
			localCostVal := costVal
			costMetricMap[dimension] = api.CostMetric{
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

	var costMetrics []api.CostMetric
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
			if costMetrics[i].TotalCost == nil {
				return false
			}
			if costMetrics[j].TotalCost == nil {
				return true
			}
			if *costMetrics[i].TotalCost != *costMetrics[j].TotalCost {
				return *costMetrics[i].TotalCost > *costMetrics[j].TotalCost
			}
		}
		return costMetrics[i].CostDimensionName < costMetrics[j].CostDimensionName
	})

	return ctx.JSON(http.StatusOK, api.ListCostMetricsResponse{
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
//	@Success		200				{object}	api.ListCostCompositionResponse
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
	costMetricMap := make(map[string]api.CostMetric)
	for connector, serviceToCostMap := range costHits {
		for dimension, costVal := range serviceToCostMap {
			connectorTyped, _ := source.ParseType(connector)
			localCostVal := costVal
			costMetricMap[dimension] = api.CostMetric{
				Connector:         connectorTyped,
				CostDimensionName: dimension,
				TotalCost:         &localCostVal,
			}
		}
	}

	var costMetrics []api.CostMetric
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

	return ctx.JSON(http.StatusOK, api.ListCostCompositionResponse{
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
//	@Success		200				{object}	[]api.CostTrendDatapoint
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
	costTrendHits, err := es.FetchDailyCostTrendByServicesBetween(h.client, connectionIDs, connectorTypes, nil, startTime, endTime, esDataPointCount)
	if err != nil {
		return err
	}

	timepointToCost := make(map[int]float64)

	for _, serviceCosts := range costTrendHits {
		for timeAt, costAtTime := range serviceCosts {
			timepointToCost[timeAt] += costAtTime
		}
	}

	apiDatapoints := make([]api.CostTrendDatapoint, 0, len(timepointToCost))
	for timeAt, costVal := range timepointToCost {
		apiDatapoints = append(apiDatapoints, api.CostTrendDatapoint{Cost: costVal, Date: time.Unix(int64(timeAt), 0)})
	}
	sort.Slice(apiDatapoints, func(i, j int) bool {
		return apiDatapoints[i].Date.Before(apiDatapoints[j].Date)
	})
	apiDatapoints = internal.DownSampleCostTrendDatapoints(apiDatapoints, int(datapointCount))

	return ctx.JSON(http.StatusOK, apiDatapoints)
}

// ListConnectionsData godoc
//
//	@Summary	Returns cost and resource count data of the specified accounts at the specified time - internal use api,  for full result use onboard api
//	@Security	BearerToken
//	@Tags		connection
//	@Accept		json
//	@Produce	json
//	@Param		connectionId	query		[]string	true	"Connection IDs"
//	@Param		startTime		query		int			false	"start time in unix seconds"
//	@Param		endTime			query		int			false	"end time in unix seconds"
//	@Success	200				{object}	map[string]api.ConnectionData
//	@Router		/inventory/api/v2/connections/data [get]
func (h *HttpHandler) ListConnectionsData(ctx echo.Context) error {
	var err error
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
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

	res := map[string]api.ConnectionData{}
	for _, connectionID := range connectionIDs {
		res[connectionID] = api.ConnectionData{
			ConnectionID:  connectionID,
			Count:         0,
			LastInventory: nil,
			Cost:          0,
		}
	}

	hits, err := es.FetchConnectionResourcesCountAtTime(h.client, nil, connectionIDs, endTime, EsFetchPageSize)
	for _, hit := range hits {
		if v, ok := res[hit.SourceID]; ok {
			v.Count += hit.ResourceCount
			if v.LastInventory == nil || v.LastInventory.IsZero() || v.LastInventory.Before(time.UnixMilli(hit.DescribedAt)) {
				v.LastInventory = utils.GetPointer(time.UnixMilli(hit.DescribedAt))
			}
			res[hit.SourceID] = v
		}
	}

	costs, err := es.FetchDailyCostHistoryByAccountsBetween(h.client, nil, connectionIDs, endTime, startTime, EsFetchPageSize)
	if err != nil {
		return err
	}
	for connectionId, costValue := range costs {
		if v, ok := res[connectionId]; ok {
			v.Cost += costValue
			res[connectionId] = v
		}
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetConnectionData godoc
//
//	@Summary	Returns cost and resource count data of the specified account at the specified time - internal use api,  for full result use onboard api
//	@Security	BearerToken
//	@Tags		connection
//	@Accept		json
//	@Produce	json
//	@Param		startTime		query		int		false	"start time in unix seconds"
//	@Param		endTime			query		int		false	"end time in unix seconds"
//	@Param		connectionId	path		string	true	"ConnectionID"
//	@Success	200				{object}	api.ConnectionData
//	@Router		/inventory/api/v2/connections/data/{connectionId} [get]
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

	res := api.ConnectionData{
		ConnectionID: connectionId,
	}

	hits, err := es.FetchConnectionResourcesCountAtTime(h.client, nil, []string{connectionId}, endTime, EsFetchPageSize)
	for _, hit := range hits {
		if hit.SourceID == connectionId {
			res.Count += hit.ResourceCount
			if res.LastInventory == nil || res.LastInventory.IsZero() || res.LastInventory.Before(time.UnixMilli(hit.DescribedAt)) {
				res.LastInventory = utils.GetPointer(time.UnixMilli(hit.DescribedAt))
			}
		}
	}

	costs, err := es.FetchDailyCostHistoryByAccountsBetween(h.client, nil, []string{connectionId}, endTime, startTime, EsFetchPageSize)
	if err != nil {
		return err
	}
	for costConnectionId, costValue := range costs {
		if costConnectionId != connectionId {
			continue
		}
		res.Cost += costValue
	}

	return ctx.JSON(http.StatusOK, res)
}

// ListServiceSummaries godoc
//
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
//	@Success		200				{object}	api.ListServiceSummariesResponse
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

	serviceSummaries := make([]api.ServiceSummary, 0, len(services))
	for _, service := range services {
		serviceSummary := api.ServiceSummary{
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
	serviceSummariesFiltered := make([]api.ServiceSummary, 0, len(serviceSummaries))
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

	res := api.ListServiceSummariesResponse{
		TotalCount: len(serviceSummaries),
		Services:   utils.Paginate(pageNumber, pageSize, serviceSummaries),
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetServiceSummary godoc
//
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
//	@Success		200			{object}	api.ServiceSummary
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

	serviceSummary := api.ServiceSummary{
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
//	@Param			connector	path		string	true	"Connector"
//	@Success		200			{object}	[]api.LocationByProviderResponse
//	@Router			/inventory/api/v1/locations/{connector} [get]
func (h *HttpHandler) GetLocations(ctx echo.Context) error {
	connectorStr := ctx.Param("connector")
	connector, _ := source.ParseType(connectorStr)

	var locations []api.LocationByProviderResponse

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
			locations = append(locations, api.LocationByProviderResponse{
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
//	@Tags			resource
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
//	@Tags			resource
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
//	@Tags			resource
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
//	@Tags			resource
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
//	@Tags			resource
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
	err = h.client.Search(context.Background(), summarizer.InventorySummaryIndex,
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

// ListInsightResults godoc
//
//	@Summary		List insight results
//	@Description	List all insight results for the given insightIds - this mostly for internal usage, use compliance api for full api
//	@Security		BearerToken
//	@Tags			insight
//	@Produce		json
//	@Param			connector		query		[]source.Type	false	"filter insights by connector"
//	@Param			connectionId	query		[]string		false	"filter the result by source id"
//	@Param			insightId		query		[]string		true	"filter the result by insight id"
//	@Param			time			query		int				false	"unix seconds for the time to get the insight result for"
//	@Success		200				{object}	map[uint][]insight.InsightResource
//	@Router			/inventory/api/v2/insights [get]
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

	return ctx.JSON(http.StatusOK, insightValues)
}

// GetInsightResult godoc
//
//	@Summary		Get insight result by id
//	@Description	Get insight results for the given insightIds - this mostly for internal usage, use compliance api for full api
//	@Security		BearerToken
//	@Tags			insight
//	@Produce		json
//	@Param			insightId		path		string		true	"InsightID"
//	@Param			connectionId	query		[]string	false	"filter the result by source id"
//	@Param			time			query		int			false	"unix seconds for the time to get the insight result for"
//	@Success		200				{object}	[]insight.InsightResource
//	@Router			/inventory/api/v2/insights/{insightId} [get]
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

	if insightResult, ok := insightResults[uint(insightId)]; ok {
		return ctx.JSON(http.StatusOK, insightResult)
	} else {
		return echo.NewHTTPError(http.StatusNotFound, "no data for insight found")
	}
}

// GetInsightResultByJobId godoc
//
//	@Summary		Get insight result by Job ID
//	@Description	Get insight result for the given JobId - this mostly for internal usage, use compliance api for full api
//	@Security		BearerToken
//	@Tags			insight
//	@Produce		json
//	@Param			jobId	path		string	true	"JobId"
//	@Success		200		{object}	insight.InsightResource
//	@Router			/inventory/api/v2/insights/job/{jobId} [get]
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

// GetInsightTrendResults godoc
//
//	@Summary		Get insight trend data
//	@Description	Get an insight trend data by id and time window - this mostly for internal usage, use compliance api for full api
//	@Security		BearerToken
//	@Tags			insight
//	@Produce		json
//	@Param			insightId		path		string		true	"InsightID"
//	@Param			connectionId	query		[]string	false	"filter the result by source id"
//	@Param			startTime		query		int			false	"unix seconds for the start of the time window to get the insight trend for"
//	@Param			endTime			query		int			false	"unix seconds for the end of the time window to get the insight trend for"
//	@Success		200				{object}	map[int][]insight.InsightResource
//	@Router			/inventory/api/v2/insights/{insightId}/trend [get]
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
//	@Success		200			{object}	api.ListServiceMetadataResponse
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

	var serviceMetadata []api.Service
	for _, service := range services {
		serviceMetadata = append(serviceMetadata, service.ToApi())
	}

	sort.Slice(serviceMetadata, func(i, j int) bool {
		return serviceMetadata[i].ServiceName < serviceMetadata[j].ServiceName
	})

	result := api.ListServiceMetadataResponse{
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
//	@Success		200			{object}	api.Service
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
//	@Success		200			{object}	api.ListResourceTypeMetadataResponse
//	@Router			/inventory/api/v2/metadata/resourcetype [get]
func (h *HttpHandler) ListResourceTypeMetadata(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	serviceNames := httpserver.QueryArrayParam(ctx, "service")
	pageNumber, pageSize, err := utils.PageConfigFromStrings(ctx.QueryParam("pageNumber"), ctx.QueryParam("pageSize"))
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	resourceTypes, err := h.db.ListFilteredResourceTypes(tagMap, serviceNames, connectors, true)
	if err != nil {
		return err
	}

	var resourceTypeMetadata []api.ResourceType

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

	result := api.ListResourceTypeMetadataResponse{
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
//	@Success		200				{object}	api.ResourceType
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
