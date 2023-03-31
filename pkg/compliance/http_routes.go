package compliance

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	api3 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/db"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"

	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
	"gorm.io/gorm"

	"github.com/google/uuid"
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

	v1.POST("/findings", httpserver.AuthorizeHandler(h.GetFindings, api3.ViewerRole))
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

	res, err := es.FindingsQuery(h.client, nil, req.Filters.Connector, req.Filters.ResourceID, req.Filters.ConnectionID, req.Filters.BenchmarkID, req.Filters.PolicyID, req.Filters.Severity,
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark %s not found", benchmarkId))
		}
		return err
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

	dbAssignments, err := h.db.GetBenchmarkAssignmentsByBenchmarkId(benchmarkId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark assignments for %s not found", benchmarkId))
		}
		ctx.Logger().Errorf("find benchmark assignments by benchmark %s: %v", benchmarkId, err)
		return err
	}

	var sourceIds []string
	for _, assignment := range dbAssignments {
		sourceIds = append(sourceIds, assignment.ConnectionId)
	}
	srcs, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), sourceIds)

	var sources []api.BenchmarkAssignedSource
	for _, assignment := range dbAssignments {
		srcUUID, err := uuid.Parse(assignment.ConnectionId)
		if err != nil {
			return err
		}

		ba := api.BenchmarkAssignedSource{
			Connection: types.FullConnection{
				ID: srcUUID,
			},
			AssignedAt: assignment.AssignedAt.Unix(),
		}
		for _, src := range srcs {
			if src.ID.String() == assignment.ConnectionId {
				ba.Connection.ProviderID = src.ConnectionID
				ba.Connection.ProviderName = src.ConnectionName
			}
		}
		sources = append(sources, ba)
	}

	return ctx.JSON(http.StatusOK, sources)
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

	benchmarks, err := h.db.ListBenchmarks()
	if err != nil {
		return err
	}

	for _, b := range benchmarks {
		response = append(response, b.ToApi())
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
	return ctx.JSON(http.StatusOK, benchmark.ToApi())
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
	return ctx.JSON(http.StatusOK, policy.ToApi())
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
