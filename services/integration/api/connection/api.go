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

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/demo"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/service/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/service"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type API struct {
	svc    service.Connection
	tracer trace.Tracer
	logger *zap.Logger
}

func New(
	svc service.Connection,
	logger *zap.Logger,
) API {
	return API{
		svc:    svc,
		tracer: otel.GetTracerProvider().Tracer("integration.http.sources"),
		logger: logger.Named("source"),
	}
}

func (h API) List(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "list")
	defer span.End()

	types := httpserver.QueryArrayParam(c, "connector")

	sources, err := h.svc.List(ctx, source.ParseTypes(types))
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
			if apiRes.Credential.Version == 2 {
				apiRes.Credential.Config, err = h.svc.CredentialV2ToV1(s.Credential.Secret)
				if err != nil {
					h.logger.Error("failed to provide credential from v2 to v1", zap.Error(err))

					return echo.ErrInternalServerError
				}
			}
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

	conns, err := h.svc.Get(ctx, req.SourceIDs)
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
			if apiRes.Credential.Version == 2 {
				apiRes.Credential.Config, err = h.svc.CredentialV2ToV1(conn.Credential.Secret)
				if err != nil {
					return err
				}
			}

		}

		res = append(res, apiRes)
	}
	return c.JSON(http.StatusOK, res)
}

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

	count, err := h.svc.Count(ctx, st)
	if err != nil {
		h.logger.Error("failed to read connections from the service", zap.Error(err))

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, count)
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
//	@Success		200					{object}	api.ListConnectionSummaryResponse
//	@Router			/integration/api/v1/connections/summary [get]
func (h API) Summaries(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "summaries")
	defer span.End()

	connectors := source.ParseTypes(httpserver.QueryArrayParam(c, "connector"))
	connectionIDs := httpserver.QueryArrayParam(c, "connectionId")
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

	connections, err := h.svc.ListWithFilter(ctx, connectors, connectionIDs, lifecycleStates, healthStates)
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

	connectionData := map[string]entity.ConnectionData{}
	if needResourceCount || needCost {
		connectionData, err = h.ListConnectionsData(httpclient.FromEchoContext(ctx), nil, resourceCollections, &startTime, &endTime, needCost, needResourceCount)
		if err != nil {
			return err
		}
	}

	pendingDescribeConnections, err := h.describeClient.ListPendingConnections(&httpclient.Context{UserRole: api3.InternalRole})
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
			diffi := utils.PSub(result.Connections[i].ResourceCount, result.Connections[i].OldResourceCount)
			diffj := utils.PSub(result.Connections[j].ResourceCount, result.Connections[j].OldResourceCount)
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
			diffi := utils.PSub(result.Connections[i].ResourceCount, result.Connections[i].OldResourceCount)
			diffj := utils.PSub(result.Connections[j].ResourceCount, result.Connections[j].OldResourceCount)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
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
			if float64(*diffi)/float64(*result.Connections[i].OldResourceCount) != float64(*diffj)/float64(*result.Connections[j].OldResourceCount) {
				return float64(*diffi)/float64(*result.Connections[i].OldResourceCount) > float64(*diffj)/float64(*result.Connections[j].OldResourceCount)
			}
		case "cost_growth":
			diffi := utils.PSub(result.Connections[i].DailyCostAtEndTime, result.Connections[i].DailyCostAtStartTime)
			diffj := utils.PSub(result.Connections[j].DailyCostAtEndTime, result.Connections[j].DailyCostAtStartTime)
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
		case "cost_growth_rate":
			diffi := utils.PSub(result.Connections[i].DailyCostAtEndTime, result.Connections[i].DailyCostAtStartTime)
			diffj := utils.PSub(result.Connections[j].DailyCostAtEndTime, result.Connections[j].DailyCostAtStartTime)
			if diffi == nil && diffj == nil {
				break
			}
			if diffi == nil {
				return false
			}
			if diffj == nil {
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
			if *diffi/(*result.Connections[i].DailyCostAtStartTime) != *diffj/(*result.Connections[j].DailyCostAtStartTime) {
				return *diffi/(*result.Connections[i].DailyCostAtStartTime) > *diffj/(*result.Connections[j].DailyCostAtStartTime)
			}
		}
		return result.Connections[i].ConnectionName < result.Connections[j].ConnectionName
	})

	result.Connections = utils.Paginate(pageNumber, pageSize, result.Connections)
	for idx, cnn := range result.Connections {
		cnn.ConnectionID = demo.EncodeResponseData(c, cnn.ConnectionID)
		cnn.ConnectionName = demo.EncodeResponseData(c, cnn.ConnectionName)
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

	allConnections, err := h.svc.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	var allConnectionIDs []string
	for _, c := range allConnections {
		allConnectionIDs = append(allConnectionIDs, c.ID.String())
	}

	for key, value := range filter {
		if key == "Match" {
			dimFilter := value.(map[string]interface{})
			if dimKey, ok := dimFilter["Key"]; ok {
				if dimKey == "ConnectionID" {
					connections, err = dimFilterFunction(dimFilter, allConnectionIDs)
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
						if contains(providers, c.Connector.Name.String()) {
							connections = append(connections, c.ID.String())
						}
					}
				} else if dimKey == "ConnectionGroup" {
					allGroups, err := h.db.ListConnectionGroups()
					if err != nil {
						return nil, err
					}
					allGroupsMap := make(map[string][]string)
					var allGroupsStr []string
					for _, group := range allGroups {
						g, err := group.ToAPI(ctx, h.steampipeConn)
						if err != nil {
							return nil, err
						}
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
		} else if key == "AND" {
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
		} else if key == "OR" {
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
		} else {
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

func (s API) Register(g *echo.Group) {
	g.GET("/", httpserver.AuthorizeHandler(s.List, api.ViewerRole))
	g.POST("/", httpserver.AuthorizeHandler(s.Get, api.KaytuAdminRole))
	g.GET("/count", httpserver.AuthorizeHandler(s.Count, api.ViewerRole))
}
