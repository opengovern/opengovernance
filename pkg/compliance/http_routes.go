package compliance

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/labstack/echo/v4"
	api3 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/internal"
	insight "gitlab.com/keibiengine/keibi-engine/pkg/insight/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	es2 "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/query"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

	metadata := v1.Group("/metadata")
	metadata.GET("/tag/insight", httpserver.AuthorizeHandler(h.ListInsightTags, api3.ViewerRole))
	metadata.GET("/tag/insight/:key", httpserver.AuthorizeHandler(h.GetInsightTag, api3.ViewerRole))
	metadata.GET("/insight", httpserver.AuthorizeHandler(h.ListInsightsMetadata, api3.ViewerRole))
	metadata.GET("/insight/:insightId", httpserver.AuthorizeHandler(h.GetInsightMetadata, api3.ViewerRole))

	v1.GET("/insight", httpserver.AuthorizeHandler(h.ListInsights, api3.ViewerRole))
	v1.GET("/insight/:insightId", httpserver.AuthorizeHandler(h.GetInsight, api3.ViewerRole))
	v1.GET("/insight/:insightId/trend", httpserver.AuthorizeHandler(h.GetInsightTrend, api3.ViewerRole))

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
//	@Param			connection_id	path		string	true	"Connection ID"
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
//	@Param			connection_id	path		string	true	"Connection ID"
//	@Success		200				{object}	[]api.BenchmarkAssignment
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
//	@Param			connection_id	path	string	true	"Connection ID"
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
//	@Success	200				{object}	api.Benchmark
//	@Param		benchmark_id	path		string	true	"BenchmarkID"
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
//	@Success	200				{object}	[]api.Policy
//	@Param		benchmark_id	path		string	true	"BenchmarkID"
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
//	@Param		policy_id	path		string	true	"PolicyID"
//	@Success	200			{object}	api.Policy
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
//	@Success	200			{object}	api.Query
//	@Param		query_id	path		string	true	"QueryID"
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

// ListInsightsMetadata godoc
//
//	@Summary		List insight metadata
//	@Description	Listing insight metadata
//	@Tags			insights
//	@Produce		json
//	@Param			connector	query		[]source.Type	false	"filter by connector"
//	@Success		200			{object}	[]api.Insight
//	@Router			/compliance/api/v1/metadata/insight [get]
func (h *HttpHandler) ListInsightsMetadata(ctx echo.Context) error {
	connectors := source.ParseTypes(ctx.QueryParams()["connector"])
	enabled := true
	insightRows, err := h.db.ListInsightsWithFilters(connectors, &enabled, nil)
	if err != nil {
		return err
	}

	var result []api.Insight
	for _, insightRow := range insightRows {
		result = append(result, insightRow.ToApi())
	}
	return ctx.JSON(200, result)
}

// GetInsightMetadata godoc
//
//	@Summary		Get insight metadata by id
//	@Description	Get insight metadata by id
//	@Tags			insights
//	@Produce		json
//	@Param			insightId	path		string	true	"InsightID"
//	@Success		200			{object}	api.Insight
//	@Router			/compliance/api/v1/insight/{insightId} [get]
func (h *HttpHandler) GetInsightMetadata(ctx echo.Context) error {
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

// ListInsights godoc
//
//	@Summary		List insight with result
//	@Description	Listing insight with result
//	@Tags			insights
//	@Produce		json
//	@Param			tag				query		string			false	"Key-Value tags in key=value format to filter by"
//	@Param			connector		query		[]source.Type	false	"filter insights by connector"
//	@Param			connectionId	query		[]string		false	"filter the result by source id"
//	@Param			startTime		query		int				false	"unix seconds for the start time of the trend"
//	@Param			endTime			query		int				false	"unix seconds for the end time of the trend"
//	@Success		200				{object}	[]api.Insight
//	@Router			/compliance/api/v1/insight [get]
func (h *HttpHandler) ListInsights(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(ctx.QueryParams()["tag"])
	connectors := source.ParseTypes(ctx.QueryParams()["connector"])
	connectionIDs := ctx.QueryParams()["connectionId"]
	endTime := time.Now()
	if ctx.QueryParam("endTime") != "" {
		t, err := strconv.ParseInt(ctx.QueryParam("endTime"), 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		endTime = time.Unix(t, 0)
	}
	startTime := endTime.Add(-1 * 7 * 24 * time.Hour)
	if ctx.QueryParam("startTime") != "" {
		t, err := strconv.ParseInt(ctx.QueryParam("startTime"), 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		startTime = time.Unix(t, 0)
	}

	enabled := true
	insightRows, err := h.db.ListInsightsWithFilters(connectors, &enabled, tagMap)
	if err != nil {
		return err
	}

	insightIDsList := make([]uint, 0, len(insightRows))
	for _, insightRow := range insightRows {
		insightIDsList = append(insightIDsList, insightRow.ID)
	}

	insightIdToResults, err := h.inventoryClient.ListInsightResults(httpclient.FromEchoContext(ctx), connectors, connectionIDs, insightIDsList, &endTime)
	if err != nil {
		return err
	}

	oldInsightIdToResults, err := h.inventoryClient.ListInsightResults(httpclient.FromEchoContext(ctx), connectors, connectionIDs, insightIDsList, &startTime)
	if err != nil {
		h.logger.Warn("failed to get old insight results", zap.Error(err))
		oldInsightIdToResults = make(map[uint][]insight.InsightResource)
	}

	var result []api.Insight
	for _, insightRow := range insightRows {
		apiRes := insightRow.ToApi()
		totalOldResultValue := int64(0)
		if insightResults, ok := insightIdToResults[insightRow.ID]; ok {
			for _, insightResult := range insightResults {
				apiRes.Results = append(apiRes.Results, api.InsightResult{
					JobID:        insightResult.JobID,
					InsightID:    insightRow.ID,
					ConnectionID: insightResult.SourceID,
					ExecutedAt:   time.UnixMilli(insightResult.ExecutedAt),
					Result:       insightResult.Result,
					Locations:    insightResult.Locations,
				})
				apiRes.TotalResultValue = utils.PAdd(apiRes.TotalResultValue, &insightResult.Result)
			}
		}
		if oldInsightResults, ok := oldInsightIdToResults[insightRow.ID]; ok {
			for _, oldInsightResult := range oldInsightResults {
				totalOldResultValue += oldInsightResult.Result
			}
		}
		if totalOldResultValue != 0 && apiRes.TotalResultValue != nil {
			apiRes.TotalResultValueChangePercent = utils.GetPointer((float64(*apiRes.TotalResultValue-totalOldResultValue) / float64(totalOldResultValue)) * 100)
		}
		result = append(result, apiRes)
	}
	return ctx.JSON(200, result)
}

// GetInsight godoc
//
//	@Summary		Get insight with result by id
//	@Description	Get insight with result by id
//	@Tags			insights
//	@Produce		json
//	@Param			insightId		path		string		true	"InsightID"
//	@Param			connectionId	query		[]string	false	"filter the result by source id"
//	@Param			startTime		query		int			false	"unix seconds for the start time of the trend"
//	@Param			endTime			query		int			false	"unix seconds for the end time of the trend"
//	@Success		200				{object}	api.Insight
//	@Router			/compliance/api/v1/insight/{insightId} [get]
func (h *HttpHandler) GetInsight(ctx echo.Context) error {
	insightIdStr := ctx.Param("insightId")
	insightId, err := strconv.ParseUint(insightIdStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	connectionIDs := ctx.QueryParams()["connectionId"]
	endTime := time.Now()
	if ctx.QueryParam("endTime") != "" {
		t, err := strconv.ParseInt(ctx.QueryParam("endTime"), 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		endTime = time.Unix(t, 0)
	}
	startTime := endTime.Add(-1 * 7 * 24 * time.Hour)
	if ctx.QueryParam("startTime") != "" {
		t, err := strconv.ParseInt(ctx.QueryParam("startTime"), 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		startTime = time.Unix(t, 0)
	}

	insightRow, err := h.db.GetInsight(uint(insightId))
	if err != nil {
		return err
	}

	insightResults, err := h.inventoryClient.GetInsightResult(httpclient.FromEchoContext(ctx), connectionIDs, insightRow.ID, &endTime)
	if err != nil {
		return err
	}

	oldInsightResults, err := h.inventoryClient.GetInsightResult(httpclient.FromEchoContext(ctx), connectionIDs, insightRow.ID, &startTime)
	if err != nil {
		h.logger.Warn("failed to get old insight results", zap.Error(err))
		oldInsightResults = make([]insight.InsightResource, 0)
	}

	apiRes := insightRow.ToApi()
	totalOldResultValue := int64(0)
	for _, insightResult := range insightResults {
		connections := make([]api.InsightConnection, 0, len(insightResult.IncludedConnections))
		for _, connection := range insightResult.IncludedConnections {
			connections = append(connections, api.InsightConnection{
				ConnectionID: connection.ConnectionID,
				OriginalID:   connection.OriginalID,
			})
		}

		bucket, key, err := utils.ParseHTTPSubpathS3URIToBucketAndKey(insightResult.S3Location)
		getObjectOutput, err := h.s3Client.GetObject(ctx.Request().Context(), &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		objectBuffer, err := io.ReadAll(getObjectOutput.Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		var steampipeResults steampipe.Result
		err = json.Unmarshal(objectBuffer, &steampipeResults)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		apiRes.Results = append(apiRes.Results, api.InsightResult{
			JobID:        insightResult.JobID,
			InsightID:    insightRow.ID,
			ConnectionID: insightResult.SourceID,
			ExecutedAt:   time.UnixMilli(insightResult.ExecutedAt),
			Result:       insightResult.Result,
			Locations:    insightResult.Locations,
			Connections:  connections,
			Details: &api.InsightDetail{
				Headers: steampipeResults.Headers,
				Rows:    steampipeResults.Data,
			},
		})
		apiRes.TotalResultValue = utils.PAdd(apiRes.TotalResultValue, &insightResult.Result)
	}
	for _, oldInsightResult := range oldInsightResults {
		totalOldResultValue += oldInsightResult.Result
	}
	if totalOldResultValue != 0 && apiRes.TotalResultValue != nil {
		apiRes.TotalResultValueChangePercent = utils.GetPointer((float64(*apiRes.TotalResultValue-totalOldResultValue) / float64(totalOldResultValue)) * 100)
	}

	return ctx.JSON(200, apiRes)
}

// GetInsightTrend godoc
//
//	@Summary		Get insight trend with result by id
//	@Description	Get insight trend with result by id
//	@Tags			insights
//	@Produce		json
//	@Param			insightId		path		string		true	"InsightID"
//	@Param			connectionId	query		[]string	false	"filter the result by source id"
//	@Param			startTime		query		int			false	"unix seconds for the start time of the trend"
//	@Param			endTime			query		int			false	"unix seconds for the end time of the trend"
//	@Param			datapointCount	query		int			false	"number of datapoints to return"
//	@Success		200				{object}	[]api.InsightTrendDatapoint
//	@Router			/compliance/api/v1/insight/{insightId}/trend [get]
func (h *HttpHandler) GetInsightTrend(ctx echo.Context) error {
	insightIdStr := ctx.Param("insightId")
	insightId, err := strconv.ParseUint(insightIdStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	connectionIDs := ctx.QueryParams()["connectionId"]
	var startTime *time.Time
	if ctx.QueryParam("startTime") != "" {
		t, err := strconv.ParseInt(ctx.QueryParam("startTime"), 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		tt := time.Unix(t, 0)
		startTime = &tt
	}
	var endTime *time.Time
	if ctx.QueryParam("endTime") != "" {
		t, err := strconv.ParseInt(ctx.QueryParam("endTime"), 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		tt := time.Unix(t, 0)
		endTime = &tt
	}
	var datapointCount *int
	if ctx.QueryParam("datapointCount") != "" {
		t, err := strconv.ParseInt(ctx.QueryParam("datapointCount"), 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid datapointCount")
		}
		tt := int(t)
		datapointCount = &tt
	}

	insightRow, err := h.db.GetInsight(uint(insightId))
	if err != nil {
		return err
	}

	timeAtToInsightResults, err := h.inventoryClient.GetInsightTrendResults(httpclient.FromEchoContext(ctx), connectionIDs, insightRow.ID, startTime, endTime)
	if err != nil {
		return err
	}

	result := make([]api.InsightTrendDatapoint, 0, len(timeAtToInsightResults))
	for timeAt, insightResults := range timeAtToInsightResults {
		datapoint := api.InsightTrendDatapoint{
			Timestamp: timeAt,
			Value:     0,
		}
		for _, insightResult := range insightResults {
			datapoint.Value += int(insightResult.Result)
		}
		result = append(result, datapoint)
	}

	if datapointCount != nil {
		result = internal.DownSampleInsightTrendDatapoints(result, *datapointCount)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp < result[j].Timestamp
	})

	return ctx.JSON(200, result)
}

// ListInsightTags godoc
//
//	@Summary	Return list of the keys with possible values for filtering insights
//	@Security	BearerToken
//	@Tags		insights
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	map[string][]string
//	@Router		/compliance/api/v1/metadata/tag/insight [get]
func (h *HttpHandler) ListInsightTags(ctx echo.Context) error {
	tags, err := h.db.ListInsightTagKeysWithPossibleValues()
	if err != nil {
		return err
	}
	tags = model.TrimPrivateTags(tags)
	return ctx.JSON(http.StatusOK, tags)
}

// GetInsightTag godoc
//
//	@Summary	Return list of the possible values for filtering insights with specified key
//	@Security	BearerToken
//	@Tags		insights
//	@Accept		json
//	@Produce	json
//	@Param		key	path		string	true	"Tag key"
//	@Success	200	{object}	[]string
//	@Router		/compliance/api/v1/metadata/tag/insight/{key} [get]
func (h *HttpHandler) GetInsightTag(ctx echo.Context) error {
	tagKey := ctx.Param("key")
	if tagKey == "" || strings.HasPrefix(tagKey, model.KaytuPrivateTagPrefix) {
		return echo.NewHTTPError(http.StatusBadRequest, "tag key is invalid")
	}

	tags, err := h.db.GetInsightTagTagPossibleValues(tagKey)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, tags)
}
