package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	inventoryAPI "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/service"
	"github.com/kaytu-io/kaytu-util/pkg/fp"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type API struct {
	connSvc service.Connection
	credSvc service.Credential
	tracer  trace.Tracer
	logger  *zap.Logger
}

func New(
	connSvc service.Connection,
	credSvc service.Credential,
	logger *zap.Logger,
) API {
	return API{
		connSvc: connSvc,
		credSvc: credSvc,
		tracer:  otel.GetTracerProvider().Tracer("integration.http.sources"),
		logger:  logger.Named("source"),
	}
}

// Delete godoc
//
//	@Summary		Delete connection
//	@Description	Deleting a single connection either AWS / Azure for the given connection id. it will delete its parent credential too, if it doesn't have any other child.
//	@Security		BearerToken
//	@Tags			connections
//	@Produce		json
//	@Success		200
//	@Param			connectionId	path	string	true	"Source ID"
//	@Router			/integration/api/v1/connections/{connectionId} [delete]
func (h API) Delete(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	connID, err := uuid.Parse(c.Param("connectionId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx, span := h.tracer.Start(ctx, "delete", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	connections, err := h.connSvc.Get(ctx, []string{connID.String()})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "connection not found")
		}

		return err
	}

	connection := connections[0]

	span.AddEvent("information", trace.WithAttributes(
		attribute.String("connection name", connection.Name),
	))

	if err := h.connSvc.Delete(ctx, connection); err != nil {
		h.logger.Error("cannot delete the given connection and its related credential", zap.Error(err))

		return err
	}

	return c.NoContent(http.StatusOK)
}

func (h API) List(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "list")
	defer span.End()

	types := httpserver.QueryArrayParam(c, "connector")

	sources, err := h.connSvc.List(ctx, source.ParseTypes(types))
	if err != nil {
		h.logger.Error("failed to read sources from the service", zap.Error(err))

		return echo.ErrInternalServerError
	}

	var resp entity.ListConnectionsResponse
	for _, s := range sources {
		apiRes := entity.NewConnection(s)
		if httpserver.GetUserRole(c) == api.InternalRole {
			apiRes.Credential = entity.NewCredential(s.Credential)
			apiRes.Credential.Config = s.Credential.Secret
		}
		resp = append(resp, apiRes)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h API) Get(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "get")
	defer span.End()

	var req entity.GetConnectionsRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	conns, err := h.connSvc.Get(ctx, req.SourceIDs)
	if err != nil {
		h.logger.Error("failed to read connections from the service", zap.Error(err))

		return echo.ErrInternalServerError
	}

	var res []entity.Connection
	for _, conn := range conns {
		apiRes := entity.NewConnection(conn)
		if httpserver.GetUserRole(c) == api.InternalRole {
			apiRes.Credential = entity.NewCredential(conn.Credential)
			apiRes.Credential.Config = conn.Credential.Secret
		}

		res = append(res, apiRes)
	}
	return c.JSON(http.StatusOK, res)
}

// Count godoc
//
//	@Summary		Count connections
//	@Description	Counting connections either for the given connection type or all types if not specified.
//	@Security		BearerToken
//	@Tags			connections
//	@Produce		json
//	@Param			connector	query		string	false	"Connector"
//	@Success		200			{object}	entity.CountConnectionsResponse
//	@Router			/integration/api/v1/connections/count [get]
func (h API) Count(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "count")
	defer span.End()

	sType := c.QueryParam("connector")

	var st *source.Type

	if sType != "" {
		t, err := source.ParseType(sType)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		st = &t
	}

	count, err := h.connSvc.Count(ctx, st)
	if err != nil {
		h.logger.Error("failed to read connections from the service", zap.Error(err))

		return echo.ErrInternalServerError
	}

	resp := entity.CountConnectionsResponse{
		Count: count,
	}

	return c.JSON(http.StatusOK, resp)
}

// Summaries godoc
//
//	@Summary		List connections summaries
//	@Description	Retrieving a list of connections summaries
//	@Security		BearerToken
//	@Tags			connections
//	@Accept			json
//	@Produce		json
//	@Param			filter				query		string			false	"Filter costs"
//	@Param			connector			query		[]source.Type	false	"Connector"
//	@Param			connectionId		query		[]string		false	"Connection IDs"
//	@Param			resourceCollection	query		[]string		false	"Resource collection IDs to filter by"
//	@Param			connectionGroups	query		[]string		false	"Connection Groups"
//	@Param			lifecycleState		query		string			false	"lifecycle state filter"	Enums(DISABLED, DISCOVERED, IN_PROGRESS, ONBOARD, ARCHIVED)
//	@Param			healthState			query		string			false	"health state filter"		Enums(healthy,unhealthy)
//	@Param			pageSize			query		int				false	"page size - default is 20"
//	@Param			pageNumber			query		int				false	"page number - default is 1"
//	@Param			startTime			query		int				false	"start time in unix seconds"
//	@Param			endTime				query		int				false	"end time in unix seconds"
//	@Param			needCost			query		boolean			false	"for quicker inquiry send this parameter as false, default: true"
//	@Param			needResourceCount	query		boolean			false	"for quicker inquiry send this parameter as false, default: true"
//	@Param			sortBy				query		string			false	"column to sort by - default is cost"	Enums(onboard_date,resource_count,cost,growth,growth_rate,cost_growth,cost_growth_rate)
//	@Success		200					{object}	entity.ListConnectionsSummaryResponse
//	@Router			/integration/api/v1/connections/summaries [get]
func (h API) Summaries(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "summaries")
	defer span.End()

	connectors := source.ParseTypes(httpserver.QueryArrayParam(c, "connector"))
	connectionIDs := httpserver.QueryArrayParam(c, "connectionId")
	connectionIDs, err := httpserver.ResolveConnectionIDs(c, connectionIDs)
	if err != nil {
		return err
	}
	resourceCollections := httpserver.QueryArrayParam(c, "resourceCollection")

	endTimeStr := c.QueryParam("endTime")
	endTime := time.Now()
	if endTimeStr != "" {
		unix, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		endTime = time.Unix(unix, 0)
	}

	startTimeStr := c.QueryParam("startTime")
	startTime := endTime.AddDate(0, -1, 0)
	if startTimeStr != "" {
		unix, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		startTime = time.Unix(unix, 0)
	}

	pageNumber, pageSize, err := utils.PageConfigFromStrings(c.QueryParam("pageNumber"), c.QueryParam("pageSize"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	sortBy := strings.ToLower(c.QueryParam("sortBy"))
	if sortBy == "" {
		sortBy = "cost"
	}

	if sortBy != "cost" && sortBy != "growth" &&
		sortBy != "growth_rate" && sortBy != "cost_growth" &&
		sortBy != "cost_growth_rate" && sortBy != "onboard_date" &&
		sortBy != "resource_count" {
		return echo.NewHTTPError(http.StatusBadRequest, "sortBy is not a valid value")
	}

	var lifecycleStates []model.ConnectionLifecycleState
	lifecycleState := c.QueryParam("lifecycleState")
	if lifecycleState != "" {
		lifecycleStates = append(lifecycleStates, model.ConnectionLifecycleState(lifecycleState))
	}

	var healthStates []source.HealthStatus
	healthState := c.QueryParam("healthState")
	if healthState != "" {
		healthStates = append(healthStates, source.HealthStatus(healthState))
	}

	filterStr := c.QueryParam("filter")
	if filterStr != "" {
		var filter map[string]interface{}
		err = json.Unmarshal([]byte(filterStr), &filter)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		connectionIDs, err = h.Filter(ctx, filter)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		h.logger.Warn("Filtered Connections", zap.Strings("connection-ids", connectionIDs))
	}

	connections, err := h.connSvc.ListWithFilter(ctx, connectors, connectionIDs, lifecycleStates, healthStates)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if filterStr != "" && len(connectionIDs) == 0 {
		result := entity.ListConnectionsSummaryResponse{
			ConnectionCount:       len(connections),
			TotalCost:             0,
			TotalResourceCount:    0,
			TotalOldResourceCount: 0,
			TotalUnhealthyCount:   0,

			TotalDisabledCount:   0,
			TotalDiscoveredCount: 0,
			TotalOnboardedCount:  0,
			TotalArchivedCount:   0,
			Connections:          make([]entity.Connection, 0, len(connections)),
		}

		return c.JSON(http.StatusOK, result)
	}

	needCostStr := c.QueryParam("needCost")
	needCost := true
	// cost for resource collections is not supported yet
	if nc, err := strconv.ParseBool(needCostStr); (err == nil && !nc) || len(resourceCollections) > 0 {
		needCost = false
	}

	needResourceCountStr := c.QueryParam("needResourceCount")
	needResourceCount := true
	if nrc, err := strconv.ParseBool(needResourceCountStr); err == nil && !nrc {
		needResourceCount = false
	}

	connectionData := map[string]inventoryAPI.ConnectionData{}
	if needResourceCount || needCost {
		connectionData, err = h.connSvc.Data(httpclient.FromEchoContext(c), nil, resourceCollections, &startTime, &endTime, needCost, needResourceCount)
		if err != nil {
			return err
		}
	}

	pendingDescribeConnections, err := h.connSvc.Pending(&httpclient.Context{UserRole: api.InternalRole})
	if err != nil {
		return err
	}

	result := entity.ListConnectionsSummaryResponse{
		ConnectionCount:       len(connections),
		TotalCost:             0,
		TotalResourceCount:    0,
		TotalOldResourceCount: 0,
		TotalUnhealthyCount:   0,

		TotalDisabledCount:   0,
		TotalDiscoveredCount: 0,
		TotalOnboardedCount:  0,
		TotalArchivedCount:   0,
		Connections:          make([]entity.Connection, 0, len(connections)),
	}

	for _, connection := range connections {
		if data, ok := connectionData[connection.ID.String()]; ok {
			localData := data
			apiConn := entity.NewConnection(connection)
			apiConn.Cost = localData.TotalCost
			apiConn.DailyCostAtStartTime = localData.DailyCostAtStartTime
			apiConn.DailyCostAtEndTime = localData.DailyCostAtEndTime
			apiConn.ResourceCount = localData.Count
			apiConn.OldResourceCount = localData.OldCount
			apiConn.LastInventory = localData.LastInventory
			if localData.TotalCost != nil {
				result.TotalCost += *localData.TotalCost
			}
			if localData.Count != nil {
				result.TotalResourceCount += *localData.Count
			}
			if (localData.Count == nil || *localData.Count == 0) && len(resourceCollections) > 0 {
				continue
			}
			result.Connections = append(result.Connections, apiConn)
		} else {
			if len(resourceCollections) > 0 {
				continue
			}
			result.Connections = append(result.Connections, entity.NewConnection(connection))
		}
		switch connection.LifecycleState {
		case model.ConnectionLifecycleStateDiscovered:
			result.TotalDiscoveredCount++
		case model.ConnectionLifecycleStateDisabled:
			result.TotalDisabledCount++
		case model.ConnectionLifecycleStateInProgress:
			fallthrough
		case model.ConnectionLifecycleStateOnboard:
			result.TotalOnboardedCount++
		case model.ConnectionLifecycleStateArchived:
			result.TotalArchivedCount++
		}
		if connection.HealthState == source.HealthStatusUnhealthy {
			result.TotalUnhealthyCount++
		}
	}

	sort.Slice(result.Connections, func(i, j int) bool {
		switch sortBy {
		case "onboard_date":
			return result.Connections[i].OnboardDate.Before(result.Connections[j].OnboardDate)
		case "resource_count":
			if result.Connections[i].ResourceCount == nil && result.Connections[j].ResourceCount == nil {
				break
			}
			if result.Connections[i].ResourceCount == nil {
				return false
			}
			if result.Connections[j].ResourceCount == nil {
				return true
			}
			if *result.Connections[i].ResourceCount != *result.Connections[j].ResourceCount {
				return *result.Connections[i].ResourceCount > *result.Connections[j].ResourceCount
			}
		case "cost":
			if result.Connections[i].Cost == nil && result.Connections[j].Cost == nil {
				break
			}
			if result.Connections[i].Cost == nil {
				return false
			}
			if result.Connections[j].Cost == nil {
				return true
			}
			if *result.Connections[i].Cost != *result.Connections[j].Cost {
				return *result.Connections[i].Cost > *result.Connections[j].Cost
			}
		case "growth":
			diffI := utils.PSub(result.Connections[i].ResourceCount, result.Connections[i].OldResourceCount)
			diffJ := utils.PSub(result.Connections[j].ResourceCount, result.Connections[j].OldResourceCount)
			if diffI == nil && diffJ == nil {
				break
			}
			if diffI == nil {
				return false
			}
			if diffJ == nil {
				return true
			}
			if *diffI != *diffJ {
				return *diffI > *diffJ
			}
		case "growth_rate":
			diffI := utils.PSub(result.Connections[i].ResourceCount, result.Connections[i].OldResourceCount)
			diffJ := utils.PSub(result.Connections[j].ResourceCount, result.Connections[j].OldResourceCount)
			if diffI == nil && diffJ == nil {
				break
			}
			if diffI == nil {
				return false
			}
			if diffJ == nil {
				return true
			}
			if result.Connections[i].OldResourceCount == nil && result.Connections[j].OldResourceCount == nil {
				break
			}
			if result.Connections[i].OldResourceCount == nil {
				return true
			}
			if result.Connections[j].OldResourceCount == nil {
				return false
			}
			if *result.Connections[i].OldResourceCount == 0 && *result.Connections[j].OldResourceCount == 0 {
				break
			}
			if *result.Connections[i].OldResourceCount == 0 {
				return false
			}
			if *result.Connections[j].OldResourceCount == 0 {
				return true
			}
			if float64(*diffI)/float64(*result.Connections[i].OldResourceCount) != float64(*diffJ)/float64(*result.Connections[j].OldResourceCount) {
				return float64(*diffI)/float64(*result.Connections[i].OldResourceCount) > float64(*diffJ)/float64(*result.Connections[j].OldResourceCount)
			}
		case "cost_growth":
			diffI := utils.PSub(result.Connections[i].DailyCostAtEndTime, result.Connections[i].DailyCostAtStartTime)
			diffJ := utils.PSub(result.Connections[j].DailyCostAtEndTime, result.Connections[j].DailyCostAtStartTime)
			if diffI == nil && diffJ == nil {
				break
			}
			if diffI == nil {
				return false
			}
			if diffJ == nil {
				return true
			}
			if *diffI != *diffJ {
				return *diffI > *diffJ
			}
		case "cost_growth_rate":
			diffI := utils.PSub(result.Connections[i].DailyCostAtEndTime, result.Connections[i].DailyCostAtStartTime)
			diffJ := utils.PSub(result.Connections[j].DailyCostAtEndTime, result.Connections[j].DailyCostAtStartTime)
			if diffI == nil && diffJ == nil {
				break
			}
			if diffI == nil {
				return false
			}
			if diffJ == nil {
				return true
			}
			if result.Connections[i].DailyCostAtStartTime == nil && result.Connections[j].DailyCostAtStartTime == nil {
				break
			}
			if result.Connections[i].DailyCostAtStartTime == nil {
				return true
			}
			if result.Connections[j].DailyCostAtStartTime == nil {
				return false
			}
			if *result.Connections[i].DailyCostAtStartTime == 0 && *result.Connections[j].DailyCostAtStartTime == 0 {
				break
			}
			if *result.Connections[i].DailyCostAtStartTime == 0 {
				return false
			}
			if *result.Connections[j].DailyCostAtStartTime == 0 {
				return true
			}
			if *diffI/(*result.Connections[i].DailyCostAtStartTime) != *diffJ/(*result.Connections[j].DailyCostAtStartTime) {
				return *diffI/(*result.Connections[i].DailyCostAtStartTime) > *diffJ/(*result.Connections[j].DailyCostAtStartTime)
			}
		}
		return result.Connections[i].ConnectionName < result.Connections[j].ConnectionName
	})

	result.Connections = utils.Paginate(pageNumber, pageSize, result.Connections)
	for idx, cnn := range result.Connections {
		for _, pc := range pendingDescribeConnections {
			if cnn.ID.String() == pc {
				cnn.DescribeJobRunning = true
				break
			}
		}
		result.Connections[idx] = cnn
	}
	return c.JSON(http.StatusOK, result)
}

func (h API) Filter(ctx context.Context, filter map[string]interface{}) ([]string, error) {
	var connections []string

	allConnections, err := h.connSvc.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	var allConnectionIDs []string
	for _, c := range allConnections {
		allConnectionIDs = append(allConnectionIDs, c.ID.String())
	}

	for key, value := range filter {
		switch key {
		case "Match":
			dimFilter := value.(map[string]interface{})
			if dimKey, ok := dimFilter["Key"]; ok {
				switch dimKey {
				case "ConnectionID":
					connections, err = dimFilterFunction(dimFilter, allConnectionIDs)
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, connections))
				case "Provider":
					providers, err := dimFilterFunction(dimFilter, []string{"AWS", "Azure"})
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, providers))
					for _, c := range allConnections {
						if contains(providers, c.Connector.Name.String()) {
							connections = append(connections, c.ID.String())
						}
					}
				case "ConnectionName":
					var allConnectionsNames []string
					for _, c := range allConnections {
						allConnectionsNames = append(allConnectionsNames, c.Name)
					}
					connectionNames, err := dimFilterFunction(dimFilter, allConnectionsNames)
					if err != nil {
						return nil, err
					}
					h.logger.Warn(fmt.Sprintf("===Dim Filter Function on filter %v, result: %v", dimFilter, connectionNames))
					for _, conn := range allConnections {
						if contains(connectionNames, conn.Name) {
							connections = append(connections, conn.ID.String())
						}
					}
				}
			} else {
				return nil, fmt.Errorf("missing key")
			}
		case "AND":
			var andFilters []map[string]interface{}
			for _, v := range value.([]interface{}) {
				andFilter := v.(map[string]interface{})
				andFilters = append(andFilters, andFilter)
			}
			counter := make(map[string]int)
			for _, f := range andFilters {
				values, err := h.Filter(ctx, f)
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
		case "OR":
			var orFilters []map[string]interface{}
			for _, v := range value.([]interface{}) {
				orFilter := v.(map[string]interface{})
				orFilters = append(orFilters, orFilter)
			}
			for _, f := range orFilters {
				values, err := h.Filter(ctx, f)
				if err != nil {
					return nil, err
				}
				for _, v := range values {
					if !contains(connections, v) {
						connections = append(connections, v)
					}
				}
			}
		default:
			return nil, fmt.Errorf("invalid key: %s", key)
		}
	}
	return connections, nil
}

func contains(array []string, key string) bool {
	for _, v := range array {
		if v == key {
			return true
		}
	}
	return false
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
						if !contains(output, conn) {
							output = append(output, conn)
						}
					}
				}
			}
		case strings.Contains(matchOption.(string), "ENDS_WITH"):
			for _, v := range values {
				for _, conn := range allValues {
					if strings.HasSuffix(conn, v) {
						if !contains(output, conn) {
							output = append(output, conn)
						}
					}
				}
			}
		case strings.Contains(matchOption.(string), "CONTAINS"):
			for _, v := range values {
				for _, conn := range allValues {
					if strings.Contains(conn, v) {
						if !contains(output, conn) {
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
				if !contains(output, v) {
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

// AzureHealthCheck godoc
//
//	@Summary		Get Azure connection health
//	@Description	Get live connection health status with given connection ID for Azure.
//	@Security		BearerToken
//	@Tags			connections
//	@Produce		json
//	@Param			connectionId	path		string	true	"connection ID"
//	@Param			updateMetadata	query		bool	false	"Whether to update metadata or not"	default(true)
//	@Success		200				{object}	entity.Connection
//	@Router			/integration/api/v1/connections/{connectionId}/azure/healthcheck [get]
func (h API) AzureHealthCheck(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "healthcheck.azure")
	defer span.End()

	id, err := uuid.Parse(c.Param("connectionId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connection uuid")
	}

	err = httpserver.CheckAccessToConnectionID(c, id.String())
	if err != nil {
		return err
	}
	// means by default we are considering updateMetadata as true and makes it false only
	// when we have a query parameter name updateMetadata equals to "false"
	updateMetadata := strings.ToLower(c.QueryParam("updateMetadata")) != "false"

	connections, err := h.connSvc.Get(ctx, []string{id.String()})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		h.logger.Error("failed to get connection", zap.Error(err), zap.String("connectionId", id.String()))

		return err
	}

	// we are passing only one id to the get method,
	// so we are expecting exactly one response.
	connection := connections[0]

	span.SetAttributes(
		attribute.String("connection name", connection.Name),
	)

	if !connection.LifecycleState.IsEnabled() {
		connection, err = h.connSvc.UpdateHealth(ctx, connection, source.HealthStatusNil, fp.Optional("Connection is not enabled"), fp.Optional(false), fp.Optional(false), true)
		if err != nil {
			h.logger.Error("failed to update connection health", zap.Error(err), zap.String("connectionId", connection.SourceId))
			return err
		}
	} else {
		isHealthy, err := h.credSvc.AzureHealthCheck(ctx, &connection.Credential)
		if err != nil {
			h.logger.Error("failed to check credential health",
				zap.Error(err),
				zap.String("connectionId", connection.SourceId),
			)

			return err
		}

		if !isHealthy {
			connection, err = h.connSvc.UpdateHealth(ctx, connection, source.HealthStatusUnhealthy, fp.Optional("Credential is not healthy"), fp.Optional(false), fp.Optional(false), true)
			if err != nil {
				h.logger.Error("failed to update connection health", zap.Error(err), zap.String("connectionId", connection.SourceId))
				return err
			}
		} else {
			connection, err = h.connSvc.AzureHealth(ctx, connection, updateMetadata)
			if err != nil {
				h.logger.Error("connection healthcheck failed", zap.Error(err))

				return err
			}
		}
	}

	return c.JSON(http.StatusOK, entity.NewConnection(connection))
}

// AWSHealthCheck godoc
//
//	@Summary		Get AWS connection health
//	@Description	Get live connection health status with given connection ID for AWS.
//	@Security		BearerToken
//	@Tags			connections
//	@Produce		json
//	@Param			connectionId	path		string	true	"connection ID"
//	@Success		200				{object}	entity.Connection
//	@Router			/integration/api/v1/connections/{connectionId}/aws/healthcheck [get]
func (h API) AWSHealthCheck(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "healthcheck.aws")
	defer span.End()

	id, err := uuid.Parse(c.Param("connectionId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connection uuid")
	}
	err = httpserver.CheckAccessToConnectionID(c, id.String())
	if err != nil {
		return err
	}

	connections, err := h.connSvc.Get(ctx, []string{id.String()})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		h.logger.Error("failed to get connection", zap.Error(err), zap.String("connectionId", id.String()))

		return err
	}

	// we are passing only one id to the get method,
	// so we are expecting exactly one response.
	connection := connections[0]

	span.SetAttributes(
		attribute.String("connection name", connection.Name),
	)

	if !connection.LifecycleState.IsEnabled() {
		connection, err = h.connSvc.UpdateHealth(ctx, connection, source.HealthStatusNil, fp.Optional("Connection is not enabled"), fp.Optional(false), fp.Optional(false), true)
		if err != nil {
			h.logger.Error("failed to update connection health", zap.Error(err), zap.String("connectionId", connection.SourceId))
			return err
		}
	} else {
		isHealthy, err := h.credSvc.AWSHealthCheck(ctx, &connection.Credential, true)
		if err != nil {
			h.logger.Error("failed to check credential health",
				zap.Error(err),
				zap.String("connectionId", connection.SourceId),
			)

			return err
		}

		if !isHealthy {
			connection, err = h.connSvc.UpdateHealth(ctx, connection, source.HealthStatusUnhealthy, fp.Optional("Credential is not healthy"), fp.Optional(false), fp.Optional(false), true)
			if err != nil {
				h.logger.Error("failed to update connection health", zap.Error(err), zap.String("connectionId", connection.SourceId))
				return err
			}
		} else {
			connection, err = h.connSvc.AWSHealthCheck(ctx, connection, true)
			if err != nil {
				h.logger.Error("connection healthcheck failed", zap.Error(err))

				return err
			}
		}
	}

	return c.JSON(http.StatusOK, entity.NewConnection(connection))
}

// AWSCreate godoc
//
//	@Summary		Create AWS connection [standalone]
//	@Description	Creating AWS source [standalone]
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Success		200		{object}	entity.CreateConnectionResponse
//	@Param			request	body		entity.CreateAWSConnectionRequest	true	"Request"
//	@Router			/integration/api/v1/connections/aws [post]
func (h API) AWSCreate(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "create.aws")
	defer span.End()

	var req entity.CreateAWSConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	cfg, err := h.credSvc.AWSSDKConfig(ctx, aws.GetRoleArnFromName(req.Config.AccountID, req.Config.AssumeRoleName), req.Config.ExternalId)
	if err != nil {
		return err
	}

	acc, err := service.AWSCurrentAccount(ctx, cfg)
	if err != nil {
		return err
	}
	if req.Name != "" {
		acc.AccountName = &req.Name
	}

	src, err := h.connSvc.NewAWS(ctx, *acc, req.Description, *req.Config)
	if err != nil {
		h.logger.Error("cannot build an aws connection", zap.Error(err))

		return err
	}

	src, err = h.connSvc.AWSHealthCheck(ctx, src, false)
	if err != nil {
		h.logger.Error("connection health check failed", zap.Error(err))

		return err
	}

	err = h.connSvc.Create(ctx, src)
	if err != nil {
		h.logger.Error("cannot create an aws connection", zap.Error(err))

		return err
	}

	return c.JSON(http.StatusOK, entity.CreateConnectionResponse{
		ID: src.ID,
	})
}

func (s API) Register(g *echo.Group) {
	g.GET("", httpserver.AuthorizeHandler(s.List, api.ViewerRole))
	g.POST("", httpserver.AuthorizeHandler(s.Get, api.KaytuAdminRole))
	g.GET("/count", httpserver.AuthorizeHandler(s.Count, api.ViewerRole))
	g.GET("/summaries", httpserver.AuthorizeHandler(s.Summaries, api.ViewerRole))
	g.POST("/aws", httpserver.AuthorizeHandler(s.AWSCreate, api.EditorRole))
	g.GET("/:connectionId/azure/healthcheck", httpserver.AuthorizeHandler(s.AzureHealthCheck, api.EditorRole))
	g.GET("/:connectionId/aws/healthcheck", httpserver.AuthorizeHandler(s.AWSHealthCheck, api.EditorRole))
}
