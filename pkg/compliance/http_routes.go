package compliance

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"

	"gitlab.com/keibiengine/keibi-engine/pkg/types"

	api3 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	es2 "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/query"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
	"gorm.io/gorm"

	"github.com/labstack/echo/v4"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.GET("/benchmarks", httpserver.AuthorizeHandler(h.ListBenchmarks, api3.ViewerRole))
	v1.GET("/benchmarks/:benchmark_id", httpserver.AuthorizeHandler(h.GetBenchmark, api3.ViewerRole))
	v1.GET("/benchmarks/:benchmark_id/policies", httpserver.AuthorizeHandler(h.ListPolicies, api3.ViewerRole))
	v1.GET("/benchmarks/policies/:policy_id", httpserver.AuthorizeHandler(h.GetPolicy, api3.ViewerRole))
	v1.GET("/queries/:query_id", httpserver.AuthorizeHandler(h.GetQuery, api3.ViewerRole))

	v1.GET("/assignments", httpserver.AuthorizeHandler(h.ListAssignments, api3.ViewerRole))
	v1.GET("/assignments/benchmark/:benchmark_id", httpserver.AuthorizeHandler(h.ListAssignmentsByBenchmark, api3.ViewerRole))
	v1.GET("/assignments/connection/:connection_id", httpserver.AuthorizeHandler(h.ListAssignmentsByConnection, api3.ViewerRole))
	v1.POST("/assignments/:benchmark_id/connection/:connection_id", httpserver.AuthorizeHandler(h.CreateBenchmarkAssignment, api3.EditorRole))
	v1.DELETE("/assignments/:benchmark_id/connection/:connection_id", httpserver.AuthorizeHandler(h.DeleteBenchmarkAssignment, api3.EditorRole))

	v1.GET("/insight/peer", httpserver.AuthorizeHandler(h.ListPeerInsightGroups, api3.ViewerRole))
	v1.GET("/insight/peer/:peerGroupId", httpserver.AuthorizeHandler(h.GetInsightPeerGroup, api3.ViewerRole))
	v1.GET("/insight", httpserver.AuthorizeHandler(h.ListInsights, api3.ViewerRole))
	v1.GET("/insight/:insightId", httpserver.AuthorizeHandler(h.GetInsight, api3.ViewerRole))

	v1.GET("/benchmarks/summary", httpserver.AuthorizeHandler(h.GetBenchmarksSummary, api3.ViewerRole))
	v1.GET("/benchmark/:benchmark_id/summary", httpserver.AuthorizeHandler(h.GetBenchmarkSummary, api3.ViewerRole))
	v1.GET("/benchmark/:benchmark_id/summary/result/trend", httpserver.AuthorizeHandler(h.GetBenchmarkResultTrend, api3.ViewerRole))
	v1.GET("/benchmark/:benchmark_id/tree", httpserver.AuthorizeHandler(h.GetBenchmarkTree, api3.ViewerRole))

	v1.POST("/findings", httpserver.AuthorizeHandler(h.GetFindings, api3.ViewerRole))
	v1.GET("/findings/:benchmarkId/:field/top/:count", httpserver.AuthorizeHandler(h.GetTopFieldByFindingCount, api3.ViewerRole))
	v1.GET("/findings/metrics", httpserver.AuthorizeHandler(h.GetFindingsMetrics, api3.ViewerRole))

	v1.POST("/alarms/top", httpserver.AuthorizeHandler(h.GetTopFieldByAlarmCount, api3.ViewerRole))
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

// GetFindings godoc
//
//	@Summary	Returns all findings with respect to filters
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		request	body		api.GetFindingsRequest	true	"Request Body"
//	@Success	200		{object}	api.GetFindingsResponse
//	@Router		/compliance/api/v1/findings [post]
func (h *HttpHandler) GetFindings(ctx echo.Context) error {
	var req api.GetFindingsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	lastIdx := (req.Page.No - 1) * req.Page.Size

	var response api.GetFindingsResponse
	var sorts []map[string]interface{}
	for _, sortItem := range req.Sorts {
		item := map[string]interface{}{}
		item[string(sortItem.Field)] = sortItem.Direction
		sorts = append(sorts, item)
	}

	var benchmarkIDs []string
	for _, b := range req.Filters.BenchmarkID {
		bs, err := h.GetBenchmarkTreeIDs(b)
		if err != nil {
			return err
		}

		benchmarkIDs = append(benchmarkIDs, bs...)
	}
	res, err := es.FindingsQuery(h.client, nil, req.Filters.Connector, req.Filters.ResourceID, req.Filters.ConnectionID, benchmarkIDs, req.Filters.PolicyID, req.Filters.Severity,
		sorts, lastIdx, req.Page.Size)
	if err != nil {
		return err
	}

	for _, h := range res.Hits.Hits {
		response.Findings = append(response.Findings, h.Source)
	}
	response.TotalCount = res.Hits.Total.Value
	return ctx.JSON(http.StatusOK, response)
}

// GetTopFieldByFindingCount godoc
//
//	@Summary	Returns all findings with respect to filters
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		benchmarkId	path		string	true	"BenchmarkID"
//	@Param		field		path		string	true	"Field"	Enums(resourceType,serviceName,sourceID,resourceID)
//	@Param		count		path		int		true	"Count"
//	@Success	200			{object}	api.GetTopFieldResponse
//	@Router		/compliance/api/v1/findings/{benchmarkId}/{field}/top/{count} [get]
func (h *HttpHandler) GetTopFieldByFindingCount(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")
	field := ctx.Param("field")
	countStr := ctx.Param("count")
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return err
	}

	benchmarkIDs, err := h.GetBenchmarkTreeIDs(benchmarkID)
	if err != nil {
		return err
	}

	var response api.GetTopFieldResponse
	res, err := es.FindingsTopFieldQuery(h.client, field, nil, nil, nil, nil, benchmarkIDs, nil, nil, count)
	if err != nil {
		return err
	}

	for _, item := range res.Aggregations.FieldFilter.Buckets {
		response.Records = append(response.Records, api.TopFieldRecord{
			Value: item.Key,
			Count: item.DocCount,
		})
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetTopFieldByAlarmCount godoc
//
//	@Summary	Returns all findings with respect to filters
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		request	body		api.GetTopFieldRequest	true	"Request Body"
//	@Success	200		{object}	api.GetTopFieldResponse
//	@Router		/compliance/api/v1/alarms/top [post]
func (h *HttpHandler) GetTopFieldByAlarmCount(ctx echo.Context) error {
	var req api.GetTopFieldRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var response api.GetTopFieldResponse
	res, err := query.AlarmTopFieldQuery(h.client, req.Field, req.Filters.Connector, req.Filters.ResourceTypeID,
		req.Filters.ConnectionID, req.Filters.Status, req.Filters.BenchmarkID, req.Filters.PolicyID,
		req.Filters.Severity, req.Count)
	if err != nil {
		return err
	}

	for _, item := range res.Aggregations.FieldFilter.Buckets {
		response.Records = append(response.Records, api.TopFieldRecord{
			Value: item.Key,
			Count: item.DocCount,
		})
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetFindingsMetrics godoc
//
//	@Summary	Returns findings metrics
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		start	query		int64	false	"Start"
//	@Param		end		query		int64	false	"End"
//	@Success	200		{object}	api.GetFindingsMetricsResponse
//	@Router		/compliance/api/v1/findings/metrics [get]
func (h *HttpHandler) GetFindingsMetrics(ctx echo.Context) error {
	startDateStr := ctx.QueryParam("start")
	endDateStr := ctx.QueryParam("end")

	startDate, err := strconv.ParseInt(startDateStr, 10, 64)
	if err != nil {
		return err
	}

	endDate, err := strconv.ParseInt(endDateStr, 10, 64)
	if err != nil {
		return err
	}

	startDateTo := time.UnixMilli(startDate)
	startDateFrom := startDateTo.Add(-24 * time.Hour)
	metricStart, err := query.GetFindingMetrics(h.client, startDateTo, startDateFrom)
	if err != nil {
		return err
	}

	endDateTo := time.UnixMilli(endDate)
	endDateFrom := startDateTo.Add(-24 * time.Hour)
	metricEnd, err := query.GetFindingMetrics(h.client, endDateTo, endDateFrom)
	if err != nil {
		return err
	}

	if metricEnd == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "metrics not found")
	}
	if metricStart == nil {
		metricStart = &es2.FindingMetrics{}
	}

	var response api.GetFindingsMetricsResponse
	response.TotalFindings = metricEnd.PassedFindingsCount + metricEnd.FailedFindingsCount + metricEnd.UnknownFindingsCount
	response.PassedFindings = metricEnd.PassedFindingsCount
	response.FailedFindings = metricEnd.FailedFindingsCount
	response.UnknownFindings = metricEnd.UnknownFindingsCount

	response.LastTotalFindings = metricStart.PassedFindingsCount + metricStart.FailedFindingsCount + metricStart.UnknownFindingsCount
	response.LastPassedFindings = metricStart.PassedFindingsCount
	response.LastFailedFindings = metricStart.FailedFindingsCount
	response.LastUnknownFindings = metricStart.UnknownFindingsCount
	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmarksSummary godoc
//
//	@Summary	Get benchmark summary
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		start	query		int64	true	"Start"
//	@Param		end		query		int64	true	"End"
//	@Success	200		{object}	api.GetBenchmarksSummaryResponse
//	@Router		/compliance/api/v1/benchmarks/summary [get]
func (h *HttpHandler) GetBenchmarksSummary(ctx echo.Context) error {
	startDateStr := ctx.QueryParam("start")
	endDateStr := ctx.QueryParam("end")
	if startDateStr == "" || endDateStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "start & end query params are required")
	}
	startDate, err := strconv.ParseInt(startDateStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	endDate, err := strconv.ParseInt(endDateStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	_, _ = startDate, endDate

	var response api.GetBenchmarksSummaryResponse
	benchmarks, err := h.db.ListRootBenchmarks()
	if err != nil {
		return err
	}

	totalWorkspaceAssets, err := h.inventoryClient.CountResources(httpclient.FromEchoContext(ctx))
	if err != nil {
		return err
	}

	summ := ShortSummary{}
	for _, b := range benchmarks {
		be := b.ToApi()
		err = b.PopulateConnectors(h.db, &be)
		if err != nil {
			return err
		}

		s, err := GetShortSummary(h.client, h.db, b)
		if err != nil {
			return err
		}

		var totalBenchmarkCoveredAssets int64
		for _, conn := range s.ConnectionIDs {
			count, err := h.inventoryClient.GetAccountsResourceCount(httpclient.FromEchoContext(ctx), source.Nil, &conn)
			if err != nil {
				return err
			}
			totalBenchmarkCoveredAssets += int64(count[0].ResourceCount)
		}

		coverage := 100.0
		if totalWorkspaceAssets > 0 {
			coverage = float64(totalBenchmarkCoveredAssets) / float64(totalWorkspaceAssets) * 100.0
		}

		trend, err := h.BuildBenchmarkResultTrend(b, startDate, endDate)
		if err != nil {
			return err
		}

		var ctrend []api.Datapoint
		for _, v := range trend {
			ctrend = append(ctrend, api.Datapoint{
				Time:  v.Time,
				Value: int64(v.Result.PassedCount),
			})
		}

		response.BenchmarkSummary = append(response.BenchmarkSummary, api.BenchmarkSummary{
			ID:              b.ID,
			Title:           b.Title,
			Description:     b.Description,
			Connectors:      be.Connectors,
			Tags:            be.Tags,
			Enabled:         b.Enabled,
			Result:          s.Result,
			Checks:          s.Checks,
			Coverage:        coverage,
			CompliancyTrend: ctrend,
			PassedResources: int64(len(s.PassedResourceIDs)),
			FailedResources: int64(len(s.FailedResourceIDs)),
		})
		summ.PassedResourceIDs = append(summ.PassedResourceIDs, s.PassedResourceIDs...)
		summ.FailedResourceIDs = append(summ.FailedResourceIDs, s.FailedResourceIDs...)
		summ.ConnectionIDs = append(summ.ConnectionIDs, s.ConnectionIDs...)
	}
	summ.PassedResourceIDs = UniqueArray(summ.PassedResourceIDs, func(t, t2 string) bool {
		return t == t2
	})
	summ.FailedResourceIDs = UniqueArray(summ.FailedResourceIDs, func(t, t2 string) bool {
		return t == t2
	})
	summ.ConnectionIDs = UniqueArray(summ.ConnectionIDs, func(t, t2 string) bool {
		return t == t2
	})

	response.PassedResources = int64(len(summ.PassedResourceIDs))
	response.FailedResources = int64(len(summ.FailedResourceIDs))
	for _, conn := range summ.ConnectionIDs {
		count, err := h.inventoryClient.GetAccountsResourceCount(httpclient.FromEchoContext(ctx), source.Nil, &conn)
		if err != nil {
			return err
		}
		response.TotalAssets += int64(count[0].ResourceCount)
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmarkSummary godoc
//
//	@Summary	Get benchmark summary
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		benchmark_id	path		string	true	"BenchmarkID"
//	@Success	200				{object}	api.BenchmarkSummary
//	@Router		/compliance/api/v1/benchmark/{benchmark_id}/summary [get]
func (h *HttpHandler) GetBenchmarkSummary(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmark_id")

	benchmark, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		return err
	}

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid benchmarkID")
	}

	totalWorkspaceAssets, err := h.inventoryClient.CountResources(httpclient.FromEchoContext(ctx))
	if err != nil {
		return err
	}

	be := benchmark.ToApi()
	err = benchmark.PopulateConnectors(h.db, &be)
	if err != nil {
		return err
	}

	s, err := GetShortSummary(h.client, h.db, *benchmark)
	if err != nil {
		return err
	}

	var totalBenchmarkCoveredAssets int64
	for _, conn := range s.ConnectionIDs {
		count, err := h.inventoryClient.GetAccountsResourceCount(httpclient.FromEchoContext(ctx), source.Nil, &conn)
		if err != nil {
			return err
		}
		totalBenchmarkCoveredAssets += int64(count[0].ResourceCount)
	}

	coverage := 100.0
	if totalWorkspaceAssets > 0 {
		coverage = float64(totalBenchmarkCoveredAssets) / float64(totalWorkspaceAssets) * 100.0
	}
	response := api.BenchmarkSummary{
		ID:              benchmark.ID,
		Title:           benchmark.Title,
		Description:     benchmark.Description,
		Connectors:      be.Connectors,
		Tags:            be.Tags,
		Enabled:         benchmark.Enabled,
		Result:          s.Result,
		Checks:          s.Checks,
		Coverage:        coverage,
		PassedResources: int64(len(s.PassedResourceIDs)),
		FailedResources: int64(len(s.FailedResourceIDs)),
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmarkResultTrend godoc
//
//	@Summary	Get result trend
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		start			query		int64	true	"Start"
//	@Param		end				query		int64	true	"End"
//	@Param		benchmark_id	path		string	true	"BenchmarkID"
//	@Success	200				{object}	api.BenchmarkResultTrend
//	@Router		/compliance/api/v1/benchmark/{benchmark_id}/summary/result/trend [get]
func (h *HttpHandler) GetBenchmarkResultTrend(ctx echo.Context) error {
	startDateStr := ctx.QueryParam("start")
	endDateStr := ctx.QueryParam("end")
	if startDateStr == "" || endDateStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "start & end query params are required")
	}
	startDate, err := strconv.ParseInt(startDateStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	endDate, err := strconv.ParseInt(endDateStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	_, _ = startDate, endDate

	benchmarkID := ctx.Param("benchmark_id")
	benchmark, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		return err
	}
	if benchmark == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid benchmarkID")
	}

	trend, err := h.BuildBenchmarkResultTrend(*benchmark, startDate, endDate)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, api.BenchmarkResultTrend{
		ResultDatapoint: trend,
	})
}

// GetBenchmarkTree godoc
//
//	@Summary	Get benchmark tree
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		benchmark_id	path		string	true	"BenchmarkID"
//	@Param		status			query		string	true	"Status"	Enums(passed,failed,unknown)
//	@Success	200				{object}	api.BenchmarkTree
//	@Router		/compliance/api/v1/benchmark/{benchmark_id}/tree [get]
func (h *HttpHandler) GetBenchmarkTree(ctx echo.Context) error {
	var status []types.PolicyStatus
	benchmarkID := ctx.Param("benchmark_id")
	for k, va := range ctx.QueryParams() {
		if k == "status" {
			for _, v := range va {
				status = append(status, types.PolicyStatus(v))
			}
		}
	}

	benchmark, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		return err
	}

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid benchmarkID")
	}

	response, err := GetBenchmarkTree(h.db, h.client, *benchmark, status)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, response)
}

// CreateBenchmarkAssignment godoc
//
//	@Summary		Create benchmark assignment for inventory service
//	@Description	Returns benchmark assignment which insert
//	@Tags			benchmarks_assignment
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id	path		string	true	"Benchmark ID"
//	@Param			source_id		path		string	true	"Source ID"
//	@Success		200				{object}	api.BenchmarkAssignment
//	@Router			/compliance/api/v1/assignments/{benchmark_id}/connection/{connection_id} [post]
func (h *HttpHandler) CreateBenchmarkAssignment(ctx echo.Context) error {
	connectionID := ctx.Param("connection_id")
	if connectionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "connection id is empty")
	}

	benchmarkId := ctx.Param("benchmark_id")
	if benchmarkId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmark id is empty")
	}

	benchmark, err := h.db.GetBenchmark(benchmarkId)
	if err != nil {
		return err
	}

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark %s not found", benchmarkId))
	}

	src, err := h.schedulerClient.GetSource(httpclient.FromEchoContext(ctx), connectionID)
	if err != nil {
		return err
	}

	for _, policy := range benchmark.Policies {
		if policy.QueryID == nil {
			continue
		}

		q, err := h.db.GetQuery(*policy.QueryID)
		if err != nil {
			return err
		}

		if q == nil {
			return fmt.Errorf("query %s not found", *policy.QueryID)
		}

		if q.Connector != string(src.Type) {
			return echo.NewHTTPError(http.StatusBadRequest, "connector not match")
		}
	}

	assignment := &db.BenchmarkAssignment{
		BenchmarkId:  benchmarkId,
		ConnectionId: connectionID,
		AssignedAt:   time.Now(),
	}
	if err := h.db.AddBenchmarkAssignment(assignment); err != nil {
		ctx.Logger().Errorf("add benchmark assignment: %v", err)
		return err
	}

	return ctx.JSON(http.StatusOK, api.BenchmarkAssignment{
		BenchmarkId:  benchmarkId,
		ConnectionId: connectionID,
		AssignedAt:   assignment.AssignedAt.Unix(),
	})
}

// ListAssignmentsByConnection godoc
//
//	@Summary		Get all benchmark assignments with source id
//	@Description	Returns all benchmark assignments with source id
//	@Tags			benchmarks_assignment
//	@Accept			json
//	@Produce		json
//	@Param			source_id	path		string	true	"Source ID"
//	@Success		200			{object}	[]api.BenchmarkAssignment
//	@Router			/compliance/api/v1/assignments/connection/{connection_id} [get]
func (h *HttpHandler) ListAssignmentsByConnection(ctx echo.Context) error {
	connectionId := ctx.Param("connection_id")
	if connectionId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "connection id is empty")
	}

	dbAssignments, err := h.db.GetBenchmarkAssignmentsBySourceId(connectionId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark assignments for %s not found", connectionId))
		}
		ctx.Logger().Errorf("find benchmark assignments by source %s: %v", connectionId, err)
		return err
	}

	var assignments []api.BenchmarkAssignment
	for _, assignment := range dbAssignments {
		assignments = append(assignments, api.BenchmarkAssignment{
			BenchmarkId:  assignment.BenchmarkId,
			ConnectionId: assignment.ConnectionId,
			AssignedAt:   assignment.AssignedAt.Unix(),
		})
	}

	return ctx.JSON(http.StatusOK, assignments)
}

// ListAssignmentsByBenchmark godoc
//
//	@Summary		Get all benchmark assigned sources with benchmark id
//	@Description	Returns all benchmark assigned sources with benchmark id
//	@Tags			benchmarks_assignment
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id	path		string	true	"Benchmark ID"
//	@Success		200				{object}	[]api.BenchmarkAssignedSource
//	@Router			/compliance/api/v1/assignments/benchmark/{benchmark_id} [get]
func (h *HttpHandler) ListAssignmentsByBenchmark(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmark_id")
	if benchmarkId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmark id is empty")
	}

	benchmark, err := h.db.GetBenchmark(benchmarkId)
	if err != nil {
		return err
	}

	var apiBenchmark api.Benchmark
	err = benchmark.PopulateConnectors(h.db, &apiBenchmark)
	if err != nil {
		return err
	}

	hctx := httpclient.FromEchoContext(ctx)

	var resp []api.BenchmarkAssignedSource
	for _, connector := range apiBenchmark.Connectors {
		connections, err := h.onboardClient.ListSources(hctx, []source.Type{connector})
		if err != nil {
			return err
		}

		for _, connection := range connections {
			ba := api.BenchmarkAssignedSource{
				ConnectionID:   connection.ConnectionID,
				ConnectionName: connection.ConnectionName,
				Connector:      connector,
				Status:         false,
			}
			resp = append(resp, ba)
		}
	}

	dbAssignments, err := h.db.GetBenchmarkAssignmentsByBenchmarkId(benchmarkId)
	if err != nil {
		return err
	}

	for _, assignment := range dbAssignments {
		for idx, r := range resp {
			if r.ConnectionID == assignment.ConnectionId {
				r.Status = true
				resp[idx] = r
			}
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

// ListAssignments godoc
//
//	@Summary		Get all assignments
//	@Description	Returns all assignments
//	@Tags			benchmarks_assignment
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]api.BenchmarkAssignment
//	@Router			/compliance/api/v1/assignments [get]
func (h *HttpHandler) ListAssignments(ctx echo.Context) error {
	dbAssignments, err := h.db.ListBenchmarkAssignments()
	if err != nil {
		return err
	}

	var sources []api.BenchmarkAssignment
	for _, assignment := range dbAssignments {
		ba := api.BenchmarkAssignment{
			BenchmarkId:  assignment.BenchmarkId,
			ConnectionId: assignment.ConnectionId,
			AssignedAt:   assignment.AssignedAt.Unix(),
		}
		sources = append(sources, ba)
	}

	return ctx.JSON(http.StatusOK, sources)
}

// DeleteBenchmarkAssignment godoc
//
//	@Summary		Delete benchmark assignment for inventory service
//	@Description	Delete benchmark assignment with source id and benchmark id
//	@Tags			benchmarks_assignment
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id	path	string	true	"Benchmark ID"
//	@Param			source_id		path	string	true	"Source ID"
//	@Success		200
//	@Router			/compliance/api/v1/assignments/{benchmark_id}/connection/{connection_id} [delete]
func (h *HttpHandler) DeleteBenchmarkAssignment(ctx echo.Context) error {
	connectionId := ctx.Param("connection_id")
	if connectionId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "connection id is empty")
	}
	benchmarkId := ctx.Param("benchmark_id")
	if benchmarkId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmark id is empty")
	}

	if _, err := h.db.GetBenchmarkAssignmentByIds(connectionId, benchmarkId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusFound, "benchmark assignment not found")
		}
		ctx.Logger().Errorf("find benchmark assignment: %v", err)
		return err
	}

	if err := h.db.DeleteBenchmarkAssignmentById(connectionId, benchmarkId); err != nil {
		ctx.Logger().Errorf("delete benchmark assignment: %v", err)
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// ListBenchmarks godoc
//
//	@Summary	List benchmarks
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	[]api.Benchmark
//	@Router		/compliance/api/v1/benchmarks [get]
func (h *HttpHandler) ListBenchmarks(ctx echo.Context) error {
	var response []api.Benchmark

	benchmarks, err := h.db.ListRootBenchmarks()
	if err != nil {
		return err
	}

	for _, b := range benchmarks {
		be := b.ToApi()
		err = b.PopulateConnectors(h.db, &be)
		if err != nil {
			return err
		}
		response = append(response, be)
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmark godoc
//
//	@Summary	Get benchmark
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	api.Benchmark
//	@Router		/compliance/api/v1/benchmarks/{benchmark_id} [get]
func (h *HttpHandler) GetBenchmark(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmark_id")
	benchmark, err := h.db.GetBenchmark(benchmarkId)
	if err != nil {
		return err
	}

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}
	resp := benchmark.ToApi()
	err = benchmark.PopulateConnectors(h.db, &resp)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, resp)
}

// ListPolicies godoc
//
//	@Summary	List policies
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	[]api.Policy
//	@Router		/compliance/api/v1/benchmarks/{benchmark_id}/policies [get]
func (h *HttpHandler) ListPolicies(ctx echo.Context) error {
	var response []api.Policy

	benchmarkId := ctx.Param("benchmark_id")
	b, err := h.db.GetBenchmark(benchmarkId)
	if err != nil {
		return err
	}

	if b == nil {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	var policyIDs []string
	for _, p := range b.Policies {
		policyIDs = append(policyIDs, p.ID)
	}

	policies, err := h.db.GetPolicies(policyIDs)
	if err != nil {
		return err
	}

	for _, p := range policies {
		response = append(response, p.ToApi())
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetPolicy godoc
//
//	@Summary	Get policy
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	api.Policy
//	@Router		/compliance/api/v1/benchmarks/policies/{policy_id} [get]
func (h *HttpHandler) GetPolicy(ctx echo.Context) error {
	policyId := ctx.Param("policy_id")
	policy, err := h.db.GetPolicy(policyId)
	if err != nil {
		return err
	}

	if policy == nil {
		return echo.NewHTTPError(http.StatusNotFound, "policy not found")
	}

	pa := policy.ToApi()
	err = policy.PopulateConnector(h.db, &pa)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, pa)
}

// GetQuery godoc
//
//	@Summary	Get query
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	api.Query
//	@Router		/compliance/api/v1/queries/{query_id} [get]
func (h *HttpHandler) GetQuery(ctx echo.Context) error {
	queryID := ctx.Param("query_id")
	q, err := h.db.GetQuery(queryID)
	if err != nil {
		return err
	}

	if q == nil {
		return echo.NewHTTPError(http.StatusNotFound, "query not found")
	}

	return ctx.JSON(http.StatusOK, q.ToApi())
}

// ListInsights godoc
//
//	@Summary		List insights
//	@Description	Listing insights
//	@Tags			insights
//	@Produce		json
//	@Param			connector	query		source.Type	false	"filter by connector"
//	@Success		200			{object}	[]api.Insight
//	@Router			/compliance/api/v1/insight [get]
func (h *HttpHandler) ListInsights(ctx echo.Context) error {
	connector, _ := source.ParseType(ctx.QueryParam("connector"))

	enabled := true
	insightRows, err := h.db.ListInsightsWithFilters(connector, &enabled)
	if err != nil {
		return err
	}

	var result []api.Insight
	for _, insightRow := range insightRows {
		result = append(result, insightRow.ToApi())
	}
	return ctx.JSON(200, result)
}

// ListPeerInsightGroups godoc
//
//	@Summary		List insights
//	@Description	Listing insights
//	@Tags			insights
//	@Produce		json
//	@Success		200			{object}	[]api.InsightPeerGroup
//	@Param			connector	query		source.Type	false	"filter by connector"
//	@Router			/compliance/api/v1/insight/peer [get]
func (h *HttpHandler) ListPeerInsightGroups(ctx echo.Context) error {
	connector, _ := source.ParseType(ctx.QueryParam("connector"))

	queries, err := h.db.ListInsightsPeerGroups()
	if err != nil {
		return err
	}

	var result []api.InsightPeerGroup
	for _, insightPeerGroup := range queries {
		result = append(result, insightPeerGroup.ToApi())
	}

	if connector != source.Nil {
		for _, insightPeerGroup := range result {
			var filtered []api.Insight
			for _, insight := range insightPeerGroup.Insights {
				if insight.Connector == connector {
					filtered = append(filtered, insight)
				}
			}
			insightPeerGroup.Insights = filtered
		}
	}

	return ctx.JSON(200, result)
}

// GetInsight godoc
//
//	@Summary		Get insight by id
//	@Description	Get insight by id
//	@Tags			insights
//	@Produce		json
//	@Success		200	{object}	api.Insight
//	@Router			/compliance/api/v1/insight/{insightId} [get]
func (h *HttpHandler) GetInsight(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("insightId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	insight, err := h.db.GetInsight(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "insight not found")
		}
		return err
	}

	result := insight.ToApi()

	return ctx.JSON(200, result)
}

// GetInsightPeerGroup godoc
//
//	@Summary		Get insight by id
//	@Description	Get insight by id
//	@Tags			insights
//	@Produce		json
//	@Success		200	{object}	api.InsightPeerGroup
//	@Router			/compliance/api/v1/insight/peer/{peerGroupId} [get]
func (h *HttpHandler) GetInsightPeerGroup(ctx echo.Context) error {
	id, err := strconv.ParseUint(ctx.Param("peerGroupId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	insightPeerGroup, err := h.db.GetInsightsPeerGroup(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "insightPeerGroup not found")
		}
		return err
	}

	res := insightPeerGroup.ToApi()

	// filter out disabled insights
	var filtered []api.Insight
	for _, insight := range res.Insights {
		if insight.Enabled {
			filtered = append(filtered, insight)
		}
	}
	res.Insights = filtered

	return ctx.JSON(200, res)
}
