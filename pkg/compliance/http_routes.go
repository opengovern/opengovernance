package compliance

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/timewindow"

	es2 "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/query"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	api2 "gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"

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

	// finding dashboard
	v1.POST("/findings", h.GetFindings)
	v1.POST("/findings/filters", h.GetFindingFilters)
	v1.POST("/findings/top", h.GetTopFieldByFindingCount)
	v1.GET("/findings/metrics", h.GetFindingsMetrics)
	v1.GET("/findings/:finding_id", h.GetFindingDetails)

	// benchmark dashboard
	v1.GET("/benchmark/:benchmark_id", h.GetBenchmark)
	v1.GET("/benchmarks/summary", h.GetBenchmarksSummary)
	v1.GET("/benchmarks/:benchmark_id/insight", h.GetBenchmarkInsight)
	v1.GET("/policy/summary/:benchmark_id", h.GetPolicySummary)

	// benchmark assignment
	v1.POST("/benchmarks/:benchmark_id/source/:source_id", h.CreateBenchmarkAssignment)
	v1.GET("/benchmarks/source/:source_id", h.GetAllBenchmarkAssignmentsBySourceId)
	v1.GET("/benchmarks/:benchmark_id/sources", h.GetAllBenchmarkAssignedSourcesByBenchmarkId)
	v1.DELETE("/benchmarks/:benchmark_id/source/:source_id", h.DeleteBenchmarkAssignment)
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

// GetFindingFilters godoc
// @Summary Returns all findings with respect to filters
// @Tags    compliance
// @Accept  json
// @Produce json
// @Param   request body     api.GetFindingsRequest true "Request Body"
// @Success 200     {object} api.GetFindingsFiltersResponse
// @Router  /compliance/api/v1/findings/filters [post]
func (h *HttpHandler) GetFindingFilters(ctx echo.Context) error {
	var req api.GetFindingsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var response api.GetFindingsFiltersResponse
	res, err := es.FindingsFiltersQuery(h.client, req.Filters.Provider, req.Filters.ResourceTypeID, req.Filters.ConnectionID,
		req.Filters.FindingStatus, req.Filters.BenchmarkID, req.Filters.PolicyID, req.Filters.Severity)
	if err != nil {
		return err
	}

	var benchmarkIDs []string
	for _, item := range res.Aggregations.BenchmarkIDFilter.Buckets {
		benchmarkIDs = append(benchmarkIDs, item.Key)
	}
	idTitles, err := h.db.GetBenchmarksTitle(benchmarkIDs)
	if err != nil {
		return err
	}

	for _, item := range res.Aggregations.BenchmarkIDFilter.Buckets {
		response.Filters.Benchmarks = append(response.Filters.Benchmarks, types.FullBenchmark{
			ID:    item.Key,
			Title: idTitles[item.Key],
		})
	}

	var policyIds []string
	for _, item := range res.Aggregations.PolicyIDFilter.Buckets {
		policyIds = append(policyIds, item.Key)
	}
	idTitles, err = h.db.GetPoliciesTitle(policyIds)
	if err != nil {
		return err
	}

	for _, item := range res.Aggregations.PolicyIDFilter.Buckets {
		response.Filters.Policies = append(response.Filters.Policies, types.FullPolicy{
			ID:    item.Key,
			Title: idTitles[item.Key],
		})
	}
	for _, item := range res.Aggregations.ResourceTypeFilter.Buckets {
		response.Filters.ResourceType = append(response.Filters.ResourceType, types.FullResourceType{
			ID:   item.Key,
			Name: cloudservice.ResourceTypeName(item.Key),
		})
	}
	for _, item := range res.Aggregations.SeverityFilter.Buckets {
		response.Filters.Severity = append(response.Filters.Severity, item.Key)
	}

	var connectionIds []string
	for _, item := range res.Aggregations.SourceIDFilter.Buckets {
		connectionIds = append(connectionIds, item.Key)
	}
	sources, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), connectionIds)
	if err != nil {
		return err
	}

	for _, item := range res.Aggregations.SourceIDFilter.Buckets {
		v, err := uuid.Parse(item.Key)
		if err != nil {
			continue
		}

		var src api2.Source
		for _, s := range sources {
			if s.ID == v {
				src = s
			}
		}
		response.Filters.Connections = append(response.Filters.Connections, types.FullConnection{
			ID:           v,
			ProviderID:   src.ConnectionID,
			ProviderName: src.ConnectionName,
		})
	}
	for _, item := range res.Aggregations.SourceTypeFilter.Buckets {
		response.Filters.Provider = append(response.Filters.Provider, source.Type(item.Key))
	}
	for _, item := range res.Aggregations.StatusFilter.Buckets {
		response.Filters.FindingStatus = append(response.Filters.FindingStatus, types.ComplianceResult(item.Key))
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetTopFieldByFindingCount godoc
// @Summary Returns all findings with respect to filters
// @Tags    compliance
// @Accept  json
// @Produce json
// @Param   request body     api.GetTopFieldRequest true "Request Body"
// @Success 200     {object} api.GetTopFieldResponse
// @Router  /compliance/api/v1/findings/top [post]
func (h *HttpHandler) GetTopFieldByFindingCount(ctx echo.Context) error {
	var req api.GetTopFieldRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var response api.GetTopFieldResponse
	res, err := es.FindingsTopFieldQuery(h.client, req.Field, req.Filters.Provider, req.Filters.ResourceTypeID,
		req.Filters.ConnectionID, req.Filters.FindingStatus, req.Filters.BenchmarkID, req.Filters.PolicyID,
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

// GetTopFieldByAlarmCount godoc
// @Summary Returns all findings with respect to filters
// @Tags    compliance
// @Accept  json
// @Produce json
// @Param   request body     api.GetTopFieldRequest true "Request Body"
// @Success 200     {object} api.GetTopFieldResponse
// @Router  /compliance/api/v1/alarms/top [post]
func (h *HttpHandler) GetTopFieldByAlarmCount(ctx echo.Context) error {
	var req api.GetTopFieldRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var response api.GetTopFieldResponse
	res, err := query.AlarmTopFieldQuery(h.client, req.Field, req.Filters.Provider, req.Filters.ResourceTypeID,
		req.Filters.ConnectionID, req.Filters.FindingStatus, req.Filters.BenchmarkID, req.Filters.PolicyID,
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

// GetFindings godoc
// @Summary Returns all findings with respect to filters
// @Tags    compliance
// @Accept  json
// @Produce json
// @Param   request body     api.GetFindingsRequest true "Request Body"
// @Success 200     {object} api.GetFindingsResponse
// @Router  /compliance/api/v1/findings [post]
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

	res, err := es.FindingsQuery(h.client, nil, req.Filters.Provider, req.Filters.ResourceTypeID, req.Filters.ConnectionID,
		req.Filters.FindingStatus, req.Filters.BenchmarkID, req.Filters.PolicyID, req.Filters.Severity,
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

// GetFindingsMetrics godoc
// @Summary Returns findings metrics
// @Tags    compliance
// @Accept  json
// @Produce json
// @Param   timeWindow query    string false "Time Window" Enums(24h,1w,3m,1y,max)
// @Success 200        {object} api.GetFindingsMetricsResponse
// @Router  /compliance/api/v1/findings/metrics [get]
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
// @Summary Returns details of a single finding
// @Tags    compliance
// @Accept  json
// @Produce json
// @Param   finding_id path     string true "FindingID"
// @Success 200        {object} api.GetFindingDetailsResponse
// @Router  /compliance/api/v1/findings/{finding_id} [get]
func (h *HttpHandler) GetFindingDetails(ctx echo.Context) error {
	findingID := ctx.Param("finding_id")
	findings, err := es.FindingsQuery(h.client, []string{findingID}, nil, nil, nil, nil,
		nil, nil, nil, nil, 0, es2.EsFetchPageSize)
	if err != nil {
		return err
	}

	if len(findings.Hits.Hits) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	finding := findings.Hits.Hits[0].Source
	p, err := h.db.GetPolicy(finding.PolicyID)
	if err != nil {
		return err
	}

	tags := map[string]string{}
	for _, t := range p.Tags {
		tags[t.Key] = t.Value
	}

	var alarms []api.Alarms

	als, err := query.GetAlarms(h.client, finding.ResourceID, finding.PolicyID)
	if err != nil {
		return err
	}

	for _, a := range als {
		alarms = append(alarms, api.Alarms{
			Policy: types.FullPolicy{
				ID:    p.ID,
				Title: p.Title,
			},
			CreatedAt: a.CreatedAt,
			Status:    a.Status,
		})
	}

	response := api.GetFindingDetailsResponse{
		Connection: types.FullConnection{
			ID:           finding.SourceID,
			ProviderID:   finding.ConnectionProviderID,
			ProviderName: finding.ConnectionProviderName,
		},
		Resource: types.FullResource{
			ID:   finding.ResourceID,
			Name: finding.ResourceName,
		},
		ResourceType: types.FullResourceType{
			ID:   finding.ResourceType,
			Name: cloudservice.ResourceTypeName(finding.ResourceType),
		},
		State:             finding.Status,
		CreatedAt:         finding.EvaluatedAt,
		PolicyTags:        tags,
		PolicyDescription: p.Description,
		Reason:            finding.Reason,
		Alarms:            alarms,
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmarkInsight godoc
// @Summary Returns insight of a specific benchmark
// @Tags    compliance
// @Accept  json
// @Produce json
// @Param   benchmark_id path     string true "BenchmarkID"
// @Success 200          {object} api.GetBenchmarkInsightResponse
// @Router  /benchmarks/{benchmark_id}/insight [get]
func (h *HttpHandler) GetBenchmarkInsight(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmark_id")
	findings, err := es.FindingsQuery(h.client, nil, nil, nil, nil, nil, []string{benchmarkID},
		nil, nil, nil, 0, es2.EsFetchPageSize)
	if err != nil {
		return err
	}
	var response api.GetBenchmarkInsightResponse

	categoryMap := map[string]int64{}
	resourceTypeMap := map[string]int64{}
	accountMap := map[string]int64{}
	severityMap := map[string]int64{}
	for _, f := range findings.Hits.Hits {
		categoryMap[f.Source.Category]++
		resourceTypeMap[f.Source.ResourceType]++
		accountMap[f.Source.SourceID.String()]++
		severityMap[f.Source.PolicySeverity]++
	}

	for k, v := range categoryMap {
		response.TopCategory = append(response.TopCategory, api.InsightRecord{
			Name:  k,
			Value: v,
		})
	}

	for k, v := range resourceTypeMap {
		response.TopCategory = append(response.TopCategory, api.InsightRecord{
			Name:  k,
			Value: v,
		})
	}

	for k, v := range accountMap {
		name := ""
		for _, f := range findings.Hits.Hits {
			if f.Source.SourceID.String() == k {
				name = f.Source.ConnectionProviderName
				break
			}
		}
		response.TopCategory = append(response.TopCategory, api.InsightRecord{
			Name:  name,
			Value: v,
		})
	}

	for k, v := range severityMap {
		response.TopCategory = append(response.TopCategory, api.InsightRecord{
			Name:  k,
			Value: v,
		})
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmarksSummary godoc
// @Summary Get benchmark summary
// @Tags    compliance
// @Accept  json
// @Produce json
// @Success 200 {object} api.GetBenchmarksSummaryResponse
// @Router  /compliance/api/v1/benchmarks/summary [get]
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
			srcId := conn.SourceId.String()
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

// GetBenchmark godoc
// @Summary Get benchmark summary
// @Tags    compliance
// @Accept  json
// @Produce json
// @Router  /compliance/api/v1/benchmark/:benchmark_id [get]
func (h *HttpHandler) GetBenchmark(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmark_id")
	benchmark, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		return err
	}

	response := api.Benchmark{
		ID:          benchmark.ID,
		Title:       benchmark.Title,
		Description: benchmark.Description,
		Provider:    benchmark.Provider,
		Enabled:     benchmark.Enabled,
		Tags:        make(map[string]string),
		Policies:    nil,
	}
	for _, tag := range benchmark.Tags {
		response.Tags[tag.Key] = tag.Value
	}

	for _, p := range benchmark.Policies {
		policy := api.Policy{
			ID:                    p.ID,
			Title:                 p.Title,
			Description:           p.Description,
			Tags:                  make(map[string]string),
			Provider:              p.Provider,
			Category:              p.Category,
			SubCategory:           p.SubCategory,
			Section:               p.Section,
			Severity:              p.Severity,
			ManualVerification:    p.ManualVerification,
			ManualRemedation:      p.ManualRemedation,
			CommandLineRemedation: p.CommandLineRemedation,
			QueryToRun:            p.QueryToRun,
			KeibiManaged:          p.KeibiManaged,
		}
		for _, tag := range p.Tags {
			policy.Tags[tag.Key] = tag.Value
		}
		response.Policies = append(response.Policies, policy)
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetPolicySummary godoc
// @Summary Get benchmark summary
// @Tags    compliance
// @Accept  json
// @Produce json
// @Param   benchmarkID path     string true "BenchmarkID"
// @Success 200         {object} api.GetFindingsResponse
// @Router  /compliance/api/v1/policy/summary/{benchmark_id} [get]
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

		ps := api.PolicySummary{
			Title:       p.Title,
			Category:    p.Category,
			Subcategory: p.SubCategory,
			Severity:    p.Severity,
			Status:      policyStatus,
			CreatedAt:   p.CreatedAt.UnixMilli(),
		}
		response.PolicySummary = append(response.PolicySummary, ps)
	}
	return ctx.JSON(http.StatusOK, response)
}

// CreateBenchmarkAssignment godoc
// @Summary     Create benchmark assignment for inventory service
// @Description Returns benchmark assignment which insert
// @Tags        benchmarks_assignment
// @Accept      json
// @Produce     json
// @Param       benchmark_id path     string true "Benchmark ID"
// @Param       source_id    path     string true "Source ID"
// @Success     200          {object} api.BenchmarkAssignment
// @Router      /compliance/api/v1/benchmarks/{benchmark_id}/source/{source_id} [post]
func (h *HttpHandler) CreateBenchmarkAssignment(ctx echo.Context) error {
	sourceId := ctx.Param("source_id")
	if sourceId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source id is empty")
	}
	sourceUUID, err := uuid.Parse(sourceId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
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
		ctx.Logger().Errorf("find benchmark assignment: %v", err)
		return err
	}

	src, err := h.schedulerClient.GetSource(httpclient.FromEchoContext(ctx), sourceUUID.String())
	if err != nil {
		ctx.Logger().Errorf(fmt.Sprintf("request source: %v", err))
		return err
	}
	if benchmark.Provider != string(src.Type) {
		return echo.NewHTTPError(http.StatusBadRequest, "source type not match")
	}

	assignment := &BenchmarkAssignment{
		BenchmarkId: benchmarkId,
		SourceId:    sourceUUID,
		AssignedAt:  time.Now(),
	}
	if err := h.db.AddBenchmarkAssignment(assignment); err != nil {
		ctx.Logger().Errorf("add benchmark assignment: %v", err)
		return err
	}

	return ctx.JSON(http.StatusOK, api.BenchmarkAssignment{
		BenchmarkId: benchmarkId,
		SourceId:    sourceUUID.String(),
		AssignedAt:  assignment.AssignedAt.Unix(),
	})
}

// GetAllBenchmarkAssignmentsBySourceId godoc
// @Summary     Get all benchmark assignments with source id
// @Description Returns all benchmark assignments with source id
// @Tags        benchmarks_assignment
// @Accept      json
// @Produce     json
// @Param       source_id path     string true "Source ID"
// @Success     200       {object} []api.BenchmarkAssignment
// @Router      /compliance/api/v1/benchmarks/source/{source_id} [get]
func (h *HttpHandler) GetAllBenchmarkAssignmentsBySourceId(ctx echo.Context) error {
	sourceId := ctx.Param("source_id")
	if sourceId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source id is empty")
	}
	sourceUUID, err := uuid.Parse(sourceId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}

	dbAssignments, err := h.db.GetBenchmarkAssignmentsBySourceId(sourceUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark assignments for %s not found", sourceId))
		}
		ctx.Logger().Errorf("find benchmark assignments by source %s: %v", sourceId, err)
		return err
	}

	var assignments []api.BenchmarkAssignment
	for _, assignment := range dbAssignments {
		assignments = append(assignments, api.BenchmarkAssignment{
			BenchmarkId: assignment.BenchmarkId,
			SourceId:    assignment.SourceId.String(),
			AssignedAt:  assignment.AssignedAt.Unix(),
		})
	}

	return ctx.JSON(http.StatusOK, assignments)
}

// GetAllBenchmarkAssignedSourcesByBenchmarkId godoc
// @Summary     Get all benchmark assigned sources with benchmark id
// @Description Returns all benchmark assigned sources with benchmark id
// @Tags        benchmarks_assignment
// @Accept      json
// @Produce     json
// @Param       benchmark_id path     string true "Benchmark ID"
// @Success     200          {object} []api.BenchmarkAssignedSource
// @Router      /compliance/api/v1/benchmarks/{benchmark_id}/sources [get]
func (h *HttpHandler) GetAllBenchmarkAssignedSourcesByBenchmarkId(ctx echo.Context) error {
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
		sourceIds = append(sourceIds, assignment.SourceId.String())
	}
	srcs, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), sourceIds)

	var sources []api.BenchmarkAssignedSource
	for _, assignment := range dbAssignments {
		ba := api.BenchmarkAssignedSource{
			Connection: types.FullConnection{
				ID: assignment.SourceId,
			},
			AssignedAt: assignment.AssignedAt.Unix(),
		}
		for _, src := range srcs {
			if src.ID == assignment.SourceId {
				ba.Connection.ProviderID = src.ConnectionID
				ba.Connection.ProviderName = src.ConnectionName
			}
		}
		sources = append(sources, ba)
	}

	return ctx.JSON(http.StatusOK, sources)
}

// DeleteBenchmarkAssignment godoc
// @Summary     Delete benchmark assignment for inventory service
// @Description Delete benchmark assignment with source id and benchmark id
// @Tags        benchmarks_assignment
// @Accept      json
// @Produce     json
// @Param       benchmark_id path string true "Benchmark ID"
// @Param       source_id    path string true "Source ID"
// @Success     200
// @Router      /compliance/api/v1/benchmarks/{benchmark_id}/source/{source_id} [delete]
func (h *HttpHandler) DeleteBenchmarkAssignment(ctx echo.Context) error {
	sourceId := ctx.Param("source_id")
	if sourceId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source id is empty")
	}
	sourceUUID, err := uuid.Parse(sourceId)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid source uuid")
	}
	benchmarkId := ctx.Param("benchmark_id")
	if benchmarkId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmark id is empty")
	}

	if _, err := h.db.GetBenchmarkAssignmentByIds(sourceUUID, benchmarkId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusFound, "benchmark assignment not found")
		}
		ctx.Logger().Errorf("find benchmark assignment: %v", err)
		return err
	}

	if err := h.db.DeleteBenchmarkAssignmentById(sourceUUID, benchmarkId); err != nil {
		ctx.Logger().Errorf("delete benchmark assignment: %v", err)
		return err
	}

	return ctx.JSON(http.StatusOK, nil)
}
