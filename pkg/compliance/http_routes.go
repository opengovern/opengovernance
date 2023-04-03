package compliance

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	es2 "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/query"
	"gitlab.com/keibiengine/keibi-engine/pkg/timewindow"

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
	// finding dashboard
	v1.GET("/findings/:field/top/:count", httpserver.AuthorizeHandler(h.GetTopFieldByFindingCount, api3.ViewerRole))
	v1.GET("/findings/metrics", httpserver.AuthorizeHandler(h.GetFindingsMetrics, api3.ViewerRole))

	v1.POST("/alarms/top", httpserver.AuthorizeHandler(h.GetTopFieldByAlarmCount, api3.ViewerRole))
	v1.GET("/benchmark/:benchmark_id/summary", httpserver.AuthorizeHandler(h.GetBenchmarkSummary, api3.ViewerRole))
	v1.GET("/benchmarks/summary", httpserver.AuthorizeHandler(h.GetBenchmarksSummary, api3.ViewerRole))
	v1.GET("/policy/summary/:benchmark_id", httpserver.AuthorizeHandler(h.GetPolicySummary, api3.ViewerRole))
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

// GetTopFieldByFindingCount godoc
//
//	@Summary	Returns all findings with respect to filters
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	api.GetTopFieldResponse
//	@Router		/compliance/api/v1/findings/{field}/top/{count} [post]
func (h *HttpHandler) GetTopFieldByFindingCount(ctx echo.Context) error {
	field := ctx.QueryParam("field")
	countStr := ctx.QueryParam("count")
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return err
	}

	//var req api.GetTopFieldRequest
	//if err := bindValidate(ctx, &req); err != nil {
	//	return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	//}

	var response api.GetTopFieldResponse
	res, err := es.FindingsTopFieldQuery(h.client, field, nil, nil, nil, nil, nil, nil, nil, count)
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
//	@Param		timeWindow	query		string	false	"Time Window"	Enums(24h,1w,3m,1y,max)
//	@Success	200			{object}	api.GetFindingsMetricsResponse
//	@Router		/compliance/api/v1/findings/metrics [get]
func (h *HttpHandler) GetFindingsMetrics(ctx echo.Context) error {
	tw, err := timewindow.ParseTimeWindow(ctx.QueryParam("timeWindow"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid time window")
	}

	after := time.Now().Add(-1 * tw)
	before := after.Add(24 * time.Hour)

	metric, err := query.GetFindingMetrics(h.client, before, after)
	if err != nil {
		return err
	}

	if metric == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "metrics not found")
	}

	var response api.GetFindingsMetricsResponse
	response.TotalFindings = metric.PassedFindingsCount + metric.FailedFindingsCount + metric.UnknownFindingsCount
	response.PassedFindings = metric.PassedFindingsCount
	response.FailedFindings = metric.FailedFindingsCount
	response.UnknownFindings = metric.UnknownFindingsCount
	return ctx.JSON(http.StatusOK, response)
}

// GetFindingDetails godoc
//
//	@Summary	Returns details of a single finding
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		finding_id	path		string	true	"FindingID"
//	@Success	200			{object}	api.GetFindingDetailsResponse
//	@Router		/compliance/api/v1/findings/{finding_id} [get]
//func (h *HttpHandler) GetFindingDetails(ctx echo.Context) error {
//	findingID := ctx.Param("finding_id")
//	findings, err := es.FindingsQuery(h.client, []string{findingID}, nil, nil, nil, nil,
//		nil, nil, nil, nil, 0, es2.EsFetchPageSize)
//	if err != nil {
//		return err
//	}
//
//	if len(findings.Hits.Hits) == 0 {
//		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
//	}
//
//	finding := findings.Hits.Hits[0].Source
//	p, err := h.db.GetPolicy(finding.PolicyID)
//	if err != nil {
//		return err
//	}
//
//	tags := map[string]string{}
//	for _, t := range p.Tags {
//		tags[t.Key] = t.Value
//	}
//
//	var alarms []api.Alarms
//
//	als, err := query.GetAlarms(h.client, finding.ResourceID, finding.PolicyID)
//	if err != nil {
//		return err
//	}
//
//	for _, a := range als {
//		alarms = append(alarms, api.Alarms{
//			Policy: types.FullPolicy{
//				ID:    p.ID,
//				Title: p.Title,
//			},
//			CreatedAt: a.CreatedAt,
//			Status:    a.Status,
//		})
//	}
//
//	response := api.GetFindingDetailsResponse{
//		Connection: types.FullConnection{
//			ID:           finding.ConnectionID,
//			ProviderID:   finding.ConnectionProviderID,
//			ProviderName: finding.ConnectionProviderName,
//		},
//		Resource: types.FullResource{
//			ID:   finding.ResourceID,
//			Name: finding.ResourceName,
//		},
//		ResourceType: types.FullResourceType{
//			ID:   finding.ResourceType,
//			Name: cloudservice.ResourceTypeName(finding.ResourceType),
//		},
//		State:             finding.Status,
//		CreatedAt:         finding.EvaluatedAt,
//		PolicyTags:        tags,
//		PolicyDescription: p.Description,
//		Reason:            finding.Reason,
//		Alarms:            alarms,
//	}
//	return ctx.JSON(http.StatusOK, response)
//}

// GetBenchmarkInsight godoc
//
//	@Summary	Returns insight of a specific benchmark
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		benchmark_id	path		string	true	"BenchmarkID"
//	@Success	200				{object}	api.GetBenchmarkInsightResponse
//	@Router		/benchmarks/{benchmark_id}/insight [get]
//func (h *HttpHandler) GetBenchmarkInsight(ctx echo.Context) error {
//	benchmarkID := ctx.Param("benchmark_id")
//	findings, err := es.FindingsQuery(h.client, nil, nil, nil, nil, nil, []string{benchmarkID},
//		nil, nil, nil, 0, es2.EsFetchPageSize)
//	if err != nil {
//		return err
//	}
//	var response api.GetBenchmarkInsightResponse
//
//	categoryMap := map[string]int64{}
//	resourceTypeMap := map[string]int64{}
//	accountMap := map[string]int64{}
//	severityMap := map[string]int64{}
//	for _, f := range findings.Hits.Hits {
//		categoryMap[f.Source.Category]++
//		resourceTypeMap[f.Source.ResourceType]++
//		accountMap[f.Source.SourceID.String()]++
//		severityMap[f.Source.PolicySeverity]++
//	}
//
//	for k, v := range categoryMap {
//		response.TopCategory = append(response.TopCategory, api.InsightRecord{
//			Name:  k,
//			Value: v,
//		})
//	}
//
//	for k, v := range resourceTypeMap {
//		response.TopCategory = append(response.TopCategory, api.InsightRecord{
//			Name:  k,
//			Value: v,
//		})
//	}
//
//	for k, v := range accountMap {
//		name := ""
//		for _, f := range findings.Hits.Hits {
//			if f.Source.SourceID.String() == k {
//				name = f.Source.ConnectionProviderName
//				break
//			}
//		}
//		response.TopCategory = append(response.TopCategory, api.InsightRecord{
//			Name:  name,
//			Value: v,
//		})
//	}
//
//	for k, v := range severityMap {
//		response.TopCategory = append(response.TopCategory, api.InsightRecord{
//			Name:  k,
//			Value: v,
//		})
//	}
//
//	return ctx.JSON(http.StatusOK, response)
//}

// GetBenchmarksSummary godoc
//
//	@Summary	Get benchmark summary
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	api.GetBenchmarksSummaryResponse
//	@Router		/compliance/api/v1/benchmarks/summary [get]
func (h *HttpHandler) GetBenchmarksSummary(ctx echo.Context) error {
	var response api.GetBenchmarksSummaryResponse
	res, err := query.ListBenchmarkSummaries(h.client, nil)
	if err != nil {
		return err
	}

	benchmarks, err := h.db.ListBenchmarks()
	if err != nil {
		return err
	}

	for _, b := range benchmarks {
		var e es2.BenchmarkSummary
		for _, esb := range res {
			if b.ID == esb.BenchmarkID {
				e = esb
			}
		}
		bs := BuildBenchmarkSummary(e, b)
		assignments, err := h.db.GetBenchmarkAssignmentsByBenchmarkId(b.ID)
		if err != nil {
			return err
		}
		for _, conn := range assignments {
			srcId := conn.ConnectionId
			count, err := h.inventoryClient.GetAccountsResourceCount(httpclient.FromEchoContext(ctx), source.Nil, &srcId)
			if err != nil {
				return err
			}
			if len(count) == 0 {
				return errors.New("invalid assignment sourceId")
			}
			bs.TotalConnectionResources += int64(count[0].ResourceCount)
		}
		bs.AssignedConnectionsCount = int64(len(assignments))
		response.ShortSummary.Passed += bs.ShortSummary.Passed
		response.ShortSummary.Failed += bs.ShortSummary.Failed
		response.TotalAssets += bs.TotalConnectionResources
		response.Benchmarks = append(response.Benchmarks, bs)
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmarkSummary godoc
//
//	@Summary	Get benchmark summary
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Success	200	{object}	types.ComplianceResultSummary
//	@Router		/compliance/api/v1/benchmark/:benchmark_id/summary [get]
func (h *HttpHandler) GetBenchmarkSummary(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmark_id")
	row, err := query.ListBenchmarkSummaries(h.client, &benchmarkID)
	if err != nil {
		return err
	}

	if len(row) < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid benchmarkID")
	}

	response := types.ComplianceResultSummary{}
	for _, policy := range row[0].Policies {
		for _, resource := range policy.Resources {
			switch resource.Result {
			case types.ComplianceResultOK:
				response.OkCount++
			case types.ComplianceResultSKIP:
				response.SkipCount++
			case types.ComplianceResultINFO:
				response.InfoCount++
			case types.ComplianceResultERROR:
				response.ErrorCount++
			case types.ComplianceResultALARM:
				response.AlarmCount++
			}
		}
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetPolicySummary godoc
//
//	@Summary	Get benchmark summary
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		benchmarkID	path		string	true	"BenchmarkID"
//	@Success	200			{object}	api.GetFindingsResponse
//	@Router		/compliance/api/v1/policy/summary/{benchmark_id} [get]
func (h *HttpHandler) GetPolicySummary(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmark_id")
	if len(benchmarkID) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "BenchmarkID is required")
	}

	benchmark, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		return err
	}

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid benchmarkID")
	}

	var response api.GetPoliciesSummaryResponse
	res, err := query.ListBenchmarkSummaries(h.client, &benchmarkID)
	if err != nil {
		return err
	}

	response.BenchmarkTitle = benchmark.Title
	response.BenchmarkDescription = benchmark.Description
	response.Enabled = benchmark.Enabled
	response.Tags = make(map[string]string)
	for _, tag := range benchmark.Tags {
		response.Tags[tag.Key] = tag.Value
	}

	for _, p := range benchmark.Policies {
		policyStatus := types.PolicyStatusPASSED
		for _, r := range res {
			for _, pr := range r.Policies {
				if pr.PolicyID != p.ID {
					continue
				}

				for _, resource := range pr.Resources {
					switch resource.Result {
					case types.ComplianceResultOK:
						response.ComplianceSummary.OkCount++
					case types.ComplianceResultALARM:
						policyStatus = types.PolicyStatusFAILED
						response.ComplianceSummary.AlarmCount++
					case types.ComplianceResultERROR:
						policyStatus = types.PolicyStatusFAILED
						response.ComplianceSummary.ErrorCount++
					case types.ComplianceResultINFO:
						if policyStatus == types.PolicyStatusPASSED {
							policyStatus = types.PolicyStatusUNKNOWN
						}
						response.ComplianceSummary.InfoCount++
					case types.ComplianceResultSKIP:
						if policyStatus == types.PolicyStatusPASSED {
							policyStatus = types.PolicyStatusUNKNOWN
						}
						response.ComplianceSummary.SkipCount++
					}
				}
			}
		}

		//TODO
		ps := api.PolicySummary{
			Title: p.Title,
			//Category:    p.Category,
			//Subcategory: p.SubCategory,
			Severity:  p.Severity,
			Status:    policyStatus,
			CreatedAt: p.CreatedAt.UnixMilli(),
		}
		response.PolicySummary = append(response.PolicySummary, ps)
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
		hasParent := false
		for _, parent := range benchmarks {
			for _, child := range parent.Children {
				if child.ID == b.ID {
					hasParent = true
				}
			}
		}

		if !hasParent {
			be := b.ToApi()
			err = b.PopulateConnectors(h.db, &be)
			if err != nil {
				return err
			}
			response = append(response, be)
		}
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
