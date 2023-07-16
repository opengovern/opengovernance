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
	api3 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/db"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/es"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/internal"
	insight "github.com/kaytu-io/kaytu-engine/pkg/insight/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	benchmarks := v1.Group("/benchmarks")
	benchmarks.GET("", httpserver.AuthorizeHandler(h.ListBenchmarks, api3.ViewerRole))
	benchmarks.GET("/:benchmark_id", httpserver.AuthorizeHandler(h.GetBenchmark, api3.ViewerRole))
	benchmarks.GET("/:benchmark_id/policies", httpserver.AuthorizeHandler(h.ListPolicies, api3.ViewerRole))
	benchmarks.GET("/policies/:policy_id", httpserver.AuthorizeHandler(h.GetPolicy, api3.ViewerRole))
	benchmarks.GET("/summary", httpserver.AuthorizeHandler(h.GetBenchmarksSummary, api3.ViewerRole))
	benchmarks.GET("/:benchmark_id/summary", httpserver.AuthorizeHandler(h.GetBenchmarkSummary, api3.ViewerRole))
	benchmarks.GET("/:benchmark_id/summary/result/trend", httpserver.AuthorizeHandler(h.GetBenchmarkResultTrend, api3.ViewerRole))
	benchmarks.GET("/:benchmark_id/tree", httpserver.AuthorizeHandler(h.GetBenchmarkTree, api3.ViewerRole))

	queries := v1.Group("/queries")
	queries.GET("/:query_id", httpserver.AuthorizeHandler(h.GetQuery, api3.ViewerRole))
	queries.GET("/sync", httpserver.AuthorizeHandler(h.SyncQueries, api3.AdminRole))

	assignments := v1.Group("/assignments")
	assignments.GET("", httpserver.AuthorizeHandler(h.ListAssignments, api3.ViewerRole))
	assignments.GET("/benchmark/:benchmark_id", httpserver.AuthorizeHandler(h.ListAssignmentsByBenchmark, api3.ViewerRole))
	assignments.GET("/connection/:connection_id", httpserver.AuthorizeHandler(h.ListAssignmentsByConnection, api3.ViewerRole))
	assignments.POST("/:benchmark_id/connection/:connection_id", httpserver.AuthorizeHandler(h.CreateBenchmarkAssignment, api3.EditorRole))
	assignments.DELETE("/:benchmark_id/connection/:connection_id", httpserver.AuthorizeHandler(h.DeleteBenchmarkAssignment, api3.EditorRole))

	metadata := v1.Group("/metadata")
	metadata.GET("/tag/insight", httpserver.AuthorizeHandler(h.ListInsightTags, api3.ViewerRole))
	metadata.GET("/tag/insight/:key", httpserver.AuthorizeHandler(h.GetInsightTag, api3.ViewerRole))
	metadata.GET("/insight", httpserver.AuthorizeHandler(h.ListInsightsMetadata, api3.ViewerRole))
	metadata.GET("/insight/:insightId", httpserver.AuthorizeHandler(h.GetInsightMetadata, api3.ViewerRole))

	insights := v1.Group("/insight")
	insightGroups := insights.Group("/group")
	insightGroups.GET("", httpserver.AuthorizeHandler(h.ListInsightGroups, api3.ViewerRole))
	insightGroups.GET("/:insightGroupId", httpserver.AuthorizeHandler(h.GetInsightGroup, api3.ViewerRole))
	insightGroups.GET("/:insightGroupId/trend", httpserver.AuthorizeHandler(h.GetInsightGroupTrend, api3.ViewerRole))
	insights.GET("", httpserver.AuthorizeHandler(h.ListInsights, api3.ViewerRole))
	insights.GET("/:insightId", httpserver.AuthorizeHandler(h.GetInsight, api3.ViewerRole))
	insights.GET("/:insightId/trend", httpserver.AuthorizeHandler(h.GetInsightTrend, api3.ViewerRole))

	findings := v1.Group("/findings")
	findings.POST("", httpserver.AuthorizeHandler(h.GetFindings, api3.ViewerRole))
	findings.GET("/:benchmarkId/:field/top/:count", httpserver.AuthorizeHandler(h.GetTopFieldByFindingCount, api3.ViewerRole))
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
//	@Summary		Get findings
//	@Description	This API enables users to retrieve all compliance run findings with respect to filters. Users can use this API to obtain a list of all compliance run findings that match specific filters, such as compliance run ID, resource ID, results, and other relevant parameters.
//	@Tags			compliance
//	@Security		BearerToken
//	@Accept			json
//	@Produce		json
//	@Param			request	body		api.GetFindingsRequest	true	"Request Body"
//	@Success		200		{object}	api.GetFindingsResponse
//	@Router			/compliance/api/v1/findings [post]
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
//	@Summary		Get top field by finding count
//	@Description	This API enables users to retrieve the top field by finding count.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmarkId	path		string	true	"BenchmarkID"
//	@Param			field		path		string	true	"Field"	Enums(resourceType,serviceName,sourceID,resourceID)
//	@Param			count		path		int		true	"Count"
//	@Success		200			{object}	api.GetTopFieldResponse
//	@Router			/compliance/api/v1/findings/{benchmarkId}/{field}/top/{count} [get]
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

// GetBenchmarksSummary godoc
//
//	@Summary		List benchmarks summaries
//	@Description	This API enables users to retrieve a summary of all benchmarks and their associated checks and results within a specified time interval. Users can use this API to obtain an overview of all benchmarks, including their names, descriptions, and other relevant information, as well as the checks and their corresponding results within the specified time period.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Success		200				{object}	api.GetBenchmarksSummaryResponse
//	@Router			/compliance/api/v1/benchmarks/summary [get]
func (h *HttpHandler) GetBenchmarksSummary(ctx echo.Context) error {
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	var response api.GetBenchmarksSummaryResponse
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

		s, err := getBenchmarkEvaluationSummary(h.client, h.db, b, connectionIDs)
		if err != nil {
			return err
		}

		response.BenchmarkSummary = append(response.BenchmarkSummary, api.BenchmarkEvaluationSummary{
			ID:          b.ID,
			Title:       b.Title,
			Description: b.Description,
			Connectors:  be.Connectors,
			Tags:        be.Tags,
			Enabled:     b.Enabled,
			Result:      s.Result,
			Checks:      s.Checks,
		})

		response.TotalResult.OkCount += s.Result.OkCount
		response.TotalResult.AlarmCount += s.Result.AlarmCount
		response.TotalResult.InfoCount += s.Result.InfoCount
		response.TotalResult.SkipCount += s.Result.SkipCount
		response.TotalResult.ErrorCount += s.Result.ErrorCount

		response.TotalChecks.UnknownCount += s.Checks.UnknownCount
		response.TotalChecks.PassedCount += s.Checks.PassedCount
		response.TotalChecks.LowCount += s.Checks.LowCount
		response.TotalChecks.MediumCount += s.Checks.MediumCount
		response.TotalChecks.HighCount += s.Checks.HighCount
		response.TotalChecks.CriticalCount += s.Checks.CriticalCount
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmarkSummary godoc
//
//	@Summary		Get benchmark summary
//	@Description	This API enables users to retrieve a summary of a benchmark and its associated checks and results. Users can use this API to obtain an overview of the benchmark, including its name, description, and other relevant information, as well as the checks and their corresponding results.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id	path		string		true	"Benchmark ID"
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Success		200				{object}	api.BenchmarkEvaluationSummary
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id}/summary [get]
func (h *HttpHandler) GetBenchmarkSummary(ctx echo.Context) error {
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	benchmarkID := ctx.Param("benchmark_id")

	benchmark, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		return err
	}

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid benchmarkID")
	}

	be := benchmark.ToApi()
	err = benchmark.PopulateConnectors(h.db, &be)
	if err != nil {
		return err
	}

	s, err := getBenchmarkEvaluationSummary(h.client, h.db, *benchmark, connectionIDs)
	if err != nil {
		return err
	}

	response := api.BenchmarkEvaluationSummary{
		ID:          benchmark.ID,
		Title:       benchmark.Title,
		Description: benchmark.Description,
		Connectors:  be.Connectors,
		Tags:        be.Tags,
		Enabled:     benchmark.Enabled,
		Result:      s.Result,
		Checks:      s.Checks,
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmarkResultTrend godoc
//
//	@Summary		Get compliance result trend
//	@Description	This API allows users to retrieve datapoints of compliance severities over a specified time period, enabling users to keep track of and monitor changes in compliance.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			start			query		int64	true	"Start time"
//	@Param			end				query		int64	true	"End time"
//	@Param			benchmark_id	path		string	true	"Benchmark ID"
//	@Success		200				{object}	api.BenchmarkResultTrend
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id}/summary/result/trend [get]
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
//	@Summary		Get benchmark tree
//	@Description	This API retrieves the benchmark tree, including all of its child benchmarks. Users can use this API to obtain a comprehensive overview of the benchmarks within a particular category or hierarchy.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id	path		string		true	"Benchmark ID"
//	@Param			status			query		[]string	true	"Status"	Enums(passed,failed,unknown)
//	@Success		200				{object}	api.BenchmarkTree
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id}/tree [get]
func (h *HttpHandler) GetBenchmarkTree(ctx echo.Context) error {
	var status []types.PolicyStatus
	benchmarkID := ctx.Param("benchmark_id")
	for k, va := range ctx.QueryParams() {
		if k == "status" || k == "status[]" {
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
//	@Security		BearerToken
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
//	@Security		BearerToken
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
//	@Security		BearerToken
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
//	@Security		BearerToken
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
//	@Security		BearerToken
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
//	@Summary		List benchmarks
//	@Description	This API returns a comprehensive list of all available benchmarks. Users can use this API to obtain an overview of the entire set of benchmarks and their corresponding details, such as their names, descriptions, and IDs.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]api.Benchmark
//	@Router			/compliance/api/v1/benchmarks [get]
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
//	@Summary		Get benchmark
//	@Description	This API enables users to retrieve benchmark details by specifying the benchmark ID. Users can use this API to obtain specific details about a particular benchmark, such as its name, description, and other relevant information.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Success		200				{object}	api.Benchmark
//	@Param			benchmark_id	path		string	true	"BenchmarkID"
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id} [get]
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
//	@Summary		List Benchmark Policies
//	@Description	This API returns a list of all policies associated with a specific benchmark. Users can use this API to obtain a comprehensive overview of the policies related to a particular benchmark and their corresponding details, such as their names, descriptions, and IDs.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Success		200				{object}	[]api.Policy
//	@Param			benchmark_id	path		string	true	"Benchmark ID"
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id}/policies [get]
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
//	@Summary		Get policy
//	@Description	This API enables users to retrieve policy details by specifying the policy ID. Users can use this API to obtain specific details about a particular policy, such as its title, description, and other relevant information.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			policy_id	path		string	true	"Policy ID"
//	@Success		200			{object}	api.Policy
//	@Router			/compliance/api/v1/benchmarks/policies/{policy_id} [get]
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
//	@Summary		Get query
//
//	@Description	This API enables users to retrieve query details by specifying the query ID.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Success		200			{object}	api.Query
//	@Param			query_id	path		string	true	"Query ID"
//	@Router			/compliance/api/v1/queries/{query_id} [get]
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

// SyncQueries godoc
//
//	@Summary		Sync queries
//
//	@Description	This API syncs queries with the git backend.
//
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/compliance/api/v1/queries/sync [get]
func (h *HttpHandler) SyncQueries(ctx echo.Context) error {
	err := h.syncJobsQueue.Publish([]byte{})
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, struct{}{})
}

// ListInsightTags godoc
//
//	@Summary		List insights tag keys
//	@Description	This API allows users to retrieve a list of insights tag keys with their possible values.
//	@Security		BearerToken
//	@Tags			insights
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]string
//	@Router			/compliance/api/v1/metadata/tag/insight [get]
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
//	@Summary		Get insights tag key
//	@Description	This API allows users to retrieve an insights tag key with the possible values for it.
//	@Security		BearerToken
//	@Tags			insights
//	@Accept			json
//	@Produce		json
//	@Param			key	path		string	true	"Tag key"
//	@Success		200	{object}	[]string
//	@Router			/compliance/api/v1/metadata/tag/insight/{key} [get]
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

// ListInsightsMetadata godoc
//
//	@Summary		List insights metadata
//	@Description	Retrieves all insights metadata.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			connector	query		[]source.Type	false	"filter by connector"
//	@Success		200			{object}	[]api.Insight
//	@Router			/compliance/api/v1/metadata/insight [get]
func (h *HttpHandler) ListInsightsMetadata(ctx echo.Context) error {
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	enabled := true
	insightRows, err := h.db.ListInsightsWithFilters(nil, connectors, &enabled, nil)
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
//	@Summary		Get insight metadata
//	@Description	Get insight metadata by id
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			insightId	path		string	true	"Insight ID"
//	@Success		200			{object}	api.Insight
//	@Router			/compliance/api/v1/metadata/insight/{insightId} [get]
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
//	@Summary		List insights
//	@Description	This API returns a list of insights based on specified filters. The API provides details of insights, including results during the specified time period for the specified connection.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			tag				query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			connector		query		[]source.Type	false	"filter insights by connector"
//	@Param			connectionId	query		[]string		false	"filter the result by source id"
//	@Param			startTime		query		int				false	"unix seconds for the start time of the trend"
//	@Param			endTime			query		int				false	"unix seconds for the end time of the trend"
//	@Success		200				{object}	[]api.Insight
//	@Router			/compliance/api/v1/insight [get]
func (h *HttpHandler) ListInsights(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
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
	insightRows, err := h.db.ListInsightsWithFilters(nil, connectors, &enabled, tagMap)
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
				localOldInsightResult := oldInsightResult.Result
				apiRes.OldTotalResultValue = utils.PAdd(apiRes.OldTotalResultValue, &localOldInsightResult)
				if apiRes.FirstOldResultDate == nil || apiRes.FirstOldResultDate.After(time.UnixMilli(oldInsightResult.ExecutedAt)) {
					apiRes.FirstOldResultDate = utils.GetPointer(time.UnixMilli(oldInsightResult.ExecutedAt))
				}
			}
		}
		if apiRes.FirstOldResultDate != nil && apiRes.FirstOldResultDate.After(startTime) {
			apiRes.OldTotalResultValue = nil
		}
		result = append(result, apiRes)
	}
	return ctx.JSON(200, result)
}

// GetInsight godoc
//
//	@Summary		Get insight
//	@Description	This API returns the specified insight with ID. The API provides details of the insight, including results during the specified time period for the specified connection.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			insightId		path		string		true	"Insight ID"
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
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
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
		localOldInsightResult := oldInsightResult.Result
		apiRes.OldTotalResultValue = utils.PAdd(apiRes.OldTotalResultValue, &localOldInsightResult)
		if apiRes.FirstOldResultDate == nil || apiRes.FirstOldResultDate.After(time.UnixMilli(oldInsightResult.ExecutedAt)) {
			apiRes.FirstOldResultDate = utils.GetPointer(time.UnixMilli(oldInsightResult.ExecutedAt))
		}
	}
	if apiRes.FirstOldResultDate != nil && apiRes.FirstOldResultDate.After(startTime) {
		apiRes.OldTotalResultValue = nil
	}

	return ctx.JSON(200, apiRes)
}

// GetInsightTrend godoc
//
//	@Summary		Get insight trend
//	@Description	This API allows users to retrieve insight results datapoints for a specified connection during a specified time period.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			insightId		path		string		true	"Insight ID"
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
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
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

// ListInsightGroups godoc
//
//	@Summary		List insight groups
//	@Description	This API returns a list of insight groups based on specified filters. The API provides details of insights, including results during the specified time period for the specified connection.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Accept			json
//	@Produce		json
//	@Param			tag				query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			connector		query		[]source.Type	false	"filter insights by connector"
//	@Param			connectionId	query		[]string		false	"filter the result by source id"
//	@Param			startTime		query		int				false	"unix seconds for the start time of the trend"
//	@Param			endTime			query		int				false	"unix seconds for the end time of the trend"
//	@Success		200				{object}	[]api.InsightGroup
//	@Router			/compliance/api/v1/insight/group [get]
func (h *HttpHandler) ListInsightGroups(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	endTime := time.Now()
	if ctx.QueryParam("endTime") != "" {
		t, err := strconv.ParseInt(ctx.QueryParam("endTime"), 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		endTime = time.Unix(t, 0)
	}
	startTime := endTime.AddDate(0, 0, -7)
	if ctx.QueryParam("startTime") != "" {
		t, err := strconv.ParseInt(ctx.QueryParam("startTime"), 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid time")
		}
		startTime = time.Unix(t, 0)
	}

	insightGroupRows, err := h.db.ListInsightGroups(connectors, tagMap)
	if err != nil {
		return err
	}

	if len(insightGroupRows) == 0 {
		return ctx.JSON(200, []api.InsightGroup{})
	}

	insightIDMap := make(map[uint]bool)
	for _, insightGroupRow := range insightGroupRows {
		for _, insightRow := range insightGroupRow.Insights {
			insightIDMap[insightRow.ID] = true
		}
	}
	insightIDsList := make([]uint, 0, len(insightIDMap))
	for insightID := range insightIDMap {
		insightIDsList = append(insightIDsList, insightID)
	}

	insightIdToResults, err := h.inventoryClient.ListInsightResults(httpclient.FromEchoContext(ctx), nil, connectionIDs, insightIDsList, &endTime)
	if err != nil {
		return err
	}

	oldInsightIdToResults, err := h.inventoryClient.ListInsightResults(httpclient.FromEchoContext(ctx), nil, connectionIDs, insightIDsList, &startTime)
	if err != nil {
		h.logger.Warn("failed to get old insight results", zap.Error(err))
		oldInsightIdToResults = make(map[uint][]insight.InsightResource)
	}

	var result []api.InsightGroup
	for _, insightGroupRow := range insightGroupRows {
		apiRes := insightGroupRow.ToApi()
		apiRes.Insights = make([]api.Insight, 0, len(insightGroupRow.Insights))
		for _, insightRow := range insightGroupRow.Insights {
			insightApiRes := insightRow.ToApi()
			if insightResults, ok := insightIdToResults[insightRow.ID]; ok {
				for _, insightResult := range insightResults {
					insightApiRes.Results = append(insightApiRes.Results, api.InsightResult{
						JobID:        insightResult.JobID,
						InsightID:    insightRow.ID,
						ConnectionID: insightResult.SourceID,
						ExecutedAt:   time.UnixMilli(insightResult.ExecutedAt),
						Result:       insightResult.Result,
						Locations:    insightResult.Locations,
					})
					insightApiRes.TotalResultValue = utils.PAdd(insightApiRes.TotalResultValue, &insightResult.Result)
				}
			}
			if oldInsightResults, ok := oldInsightIdToResults[insightRow.ID]; ok {
				for _, oldInsightResult := range oldInsightResults {
					localOldInsightResult := oldInsightResult.Result
					insightApiRes.OldTotalResultValue = utils.PAdd(insightApiRes.OldTotalResultValue, &localOldInsightResult)
					if insightApiRes.FirstOldResultDate == nil || insightApiRes.FirstOldResultDate.After(time.UnixMilli(oldInsightResult.ExecutedAt)) {
						insightApiRes.FirstOldResultDate = utils.GetPointer(time.UnixMilli(oldInsightResult.ExecutedAt))
					}
				}
			}
			if insightApiRes.FirstOldResultDate != nil && insightApiRes.FirstOldResultDate.After(startTime) {
				insightApiRes.OldTotalResultValue = nil
			}

			apiRes.TotalResultValue = utils.PAdd(apiRes.TotalResultValue, insightApiRes.TotalResultValue)
			apiRes.OldTotalResultValue = utils.PAdd(apiRes.OldTotalResultValue, insightApiRes.OldTotalResultValue)
			if apiRes.FirstOldResultDate == nil || insightApiRes.FirstOldResultDate != nil && apiRes.FirstOldResultDate.After(*insightApiRes.FirstOldResultDate) {
				apiRes.FirstOldResultDate = insightApiRes.FirstOldResultDate
			}
			apiRes.Insights = append(apiRes.Insights, insightApiRes)
		}
		if apiRes.FirstOldResultDate != nil && apiRes.FirstOldResultDate.After(startTime) {
			apiRes.OldTotalResultValue = nil
		}
		result = append(result, apiRes)
	}

	return ctx.JSON(200, result)
}

// GetInsightGroup godoc
//
//	@Summary		Get insight group
//	@Description	This API returns the specified insight group with ID. The API provides details of the insight, including results during the specified time period for the specified connection.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Accept			json
//	@Produce		json
//	@Param			insightGroupId	path		string		true	"Insight Group ID"
//	@Param			connectionId	query		[]string	false	"filter the result by source id"
//	@Param			startTime		query		int			false	"unix seconds for the start time of the trend"
//	@Param			endTime			query		int			false	"unix seconds for the end time of the trend"
//	@Success		200				{object}	api.InsightGroup
//	@Router			/compliance/api/v1/insight/group/{insightGroupId} [get]
func (h *HttpHandler) GetInsightGroup(ctx echo.Context) error {
	insightGroupIdStr := ctx.Param("insightGroupId")
	insightGroupId, err := strconv.ParseUint(insightGroupIdStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
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

	insightGroupRow, err := h.db.GetInsightGroup(uint(insightGroupId))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "insight group not found")
		}
		return err
	}

	insightIDList := make([]uint, 0, len(insightGroupRow.Insights))
	for _, insightRow := range insightGroupRow.Insights {
		insightIDList = append(insightIDList, insightRow.ID)
	}
	insightResultsMap, err := h.inventoryClient.ListInsightResults(httpclient.FromEchoContext(ctx), nil, connectionIDs, insightIDList, &endTime)
	if err != nil {
		return err
	}

	oldInsightResultsMap, err := h.inventoryClient.ListInsightResults(httpclient.FromEchoContext(ctx), nil, connectionIDs, insightIDList, &startTime)
	if err != nil {
		h.logger.Warn("failed to get old insight results", zap.Error(err))
		oldInsightResultsMap = make(map[uint][]insight.InsightResource)
	}

	apiRes := insightGroupRow.ToApi()
	apiRes.Insights = make([]api.Insight, 0, len(insightGroupRow.Insights))
	for _, insightRow := range insightGroupRow.Insights {
		insightApiRes := insightRow.ToApi()
		if insightResults, ok := insightResultsMap[insightRow.ID]; ok {
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

				insightApiRes.Results = append(insightApiRes.Results, api.InsightResult{
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
				insightApiRes.TotalResultValue = utils.PAdd(insightApiRes.TotalResultValue, &insightResult.Result)
			}
		}
		if oldInsightResults, ok := oldInsightResultsMap[insightRow.ID]; ok {
			for _, oldInsightResult := range oldInsightResults {
				localOldInsightResult := oldInsightResult.Result
				insightApiRes.OldTotalResultValue = utils.PAdd(insightApiRes.OldTotalResultValue, &localOldInsightResult)
				if insightApiRes.FirstOldResultDate == nil || insightApiRes.FirstOldResultDate.After(time.UnixMilli(oldInsightResult.ExecutedAt)) {
					insightApiRes.FirstOldResultDate = utils.GetPointer(time.UnixMilli(oldInsightResult.ExecutedAt))
				}
			}
		}
		if insightApiRes.FirstOldResultDate != nil && insightApiRes.FirstOldResultDate.After(startTime) {
			insightApiRes.OldTotalResultValue = nil
		}

		apiRes.TotalResultValue = utils.PAdd(apiRes.TotalResultValue, insightApiRes.TotalResultValue)
		apiRes.OldTotalResultValue = utils.PAdd(apiRes.OldTotalResultValue, insightApiRes.OldTotalResultValue)
		if apiRes.FirstOldResultDate == nil || insightApiRes.FirstOldResultDate != nil && apiRes.FirstOldResultDate.After(*insightApiRes.FirstOldResultDate) {
			apiRes.FirstOldResultDate = insightApiRes.FirstOldResultDate
		}
		apiRes.Insights = append(apiRes.Insights, insightApiRes)
	}
	if apiRes.FirstOldResultDate != nil && apiRes.FirstOldResultDate.After(startTime) {
		apiRes.OldTotalResultValue = nil
	}

	return ctx.JSON(200, apiRes)
}

// GetInsightGroupTrend godoc
//
//	@Summary		Get insight group trend
//	@Description	This API allows users to retrieve insight group results datapoints for a specified connection during a specified time period.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			insightGroupId	path		string		true	"Insight ID"
//	@Param			connectionId	query		[]string	false	"filter the result by source id"
//	@Param			startTime		query		int			false	"unix seconds for the start time of the trend"
//	@Param			endTime			query		int			false	"unix seconds for the end time of the trend"
//	@Param			datapointCount	query		int			false	"number of datapoints to return"
//	@Success		200				{object}	api.InsightGroupTrendResponse
//	@Router			/compliance/api/v1/insight/group/{insightGroupId}/trend [get]
func (h *HttpHandler) GetInsightGroupTrend(ctx echo.Context) error {
	insightGroupIdStr := ctx.Param("insightGroupId")
	insightGroupId, err := strconv.ParseUint(insightGroupIdStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
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

	insightGroupRow, err := h.db.GetInsightGroup(uint(insightGroupId))
	if err != nil {
		return err
	}

	result := api.InsightGroupTrendResponse{
		TrendPerInsight: make(map[uint][]api.InsightTrendDatapoint),
	}
	trendMap := make(map[int]int)
	for _, insightRow := range insightGroupRow.Insights {
		timeAtToInsightResults, err := h.inventoryClient.GetInsightTrendResults(httpclient.FromEchoContext(ctx), connectionIDs, insightRow.ID, startTime, endTime)
		if err != nil {
			return err
		}
		perInsightResult := make([]api.InsightTrendDatapoint, 0, len(timeAtToInsightResults))
		for timeAt, insightResults := range timeAtToInsightResults {
			datapoint := api.InsightTrendDatapoint{
				Timestamp: timeAt,
				Value:     0,
			}
			for _, insightResult := range insightResults {
				datapoint.Value += int(insightResult.Result)
			}
			perInsightResult = append(perInsightResult, datapoint)
		}

		if datapointCount != nil {
			perInsightResult = internal.DownSampleInsightTrendDatapoints(perInsightResult, *datapointCount)
		}
		for _, datapoint := range perInsightResult {
			trendMap[datapoint.Timestamp] += datapoint.Value
		}

		sort.Slice(perInsightResult, func(i, j int) bool {
			return perInsightResult[i].Timestamp < perInsightResult[j].Timestamp
		})
		result.TrendPerInsight[insightRow.ID] = perInsightResult
	}
	for timestamp, value := range trendMap {
		result.Trend = append(result.Trend, api.InsightTrendDatapoint{
			Timestamp: timestamp,
			Value:     value,
		})
	}
	sort.Slice(result.Trend, func(i, j int) bool {
		return result.Trend[i].Timestamp < result.Trend[j].Timestamp
	})
	if datapointCount != nil {
		result.Trend = internal.DownSampleInsightTrendDatapoints(result.Trend, *datapointCount)
	}

	return ctx.JSON(200, result)
}
