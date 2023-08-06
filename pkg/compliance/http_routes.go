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
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	api "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/db"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/es"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/internal"
	insight "github.com/kaytu-io/kaytu-engine/pkg/insight/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	kaytuTypes "github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
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
	benchmarks.GET("", httpserver.AuthorizeHandler(h.ListBenchmarks, authApi.ViewerRole))
	benchmarks.GET("/:benchmark_id", httpserver.AuthorizeHandler(h.GetBenchmark, authApi.ViewerRole))
	benchmarks.GET("/policies/:policy_id", httpserver.AuthorizeHandler(h.GetPolicy, authApi.ViewerRole))
	benchmarks.GET("/summary", httpserver.AuthorizeHandler(h.ListBenchmarksSummary, authApi.ViewerRole))
	benchmarks.GET("/:benchmark_id/summary", httpserver.AuthorizeHandler(h.GetBenchmarkSummary, authApi.ViewerRole))
	benchmarks.GET("/:benchmark_id/trend", httpserver.AuthorizeHandler(h.GetBenchmarkTrend, authApi.ViewerRole))
	benchmarks.GET("/:benchmark_id/tree", httpserver.AuthorizeHandler(h.GetBenchmarkTree, authApi.ViewerRole))

	queries := v1.Group("/queries")
	queries.GET("/:query_id", httpserver.AuthorizeHandler(h.GetQuery, authApi.ViewerRole))
	queries.GET("/sync", httpserver.AuthorizeHandler(h.SyncQueries, authApi.AdminRole))

	assignments := v1.Group("/assignments")
	assignments.GET("/benchmark/:benchmark_id", httpserver.AuthorizeHandler(h.ListAssignmentsByBenchmark, authApi.ViewerRole))
	assignments.GET("/connection/:connection_id", httpserver.AuthorizeHandler(h.ListAssignmentsByConnection, authApi.ViewerRole))
	assignments.POST("/:benchmark_id/connection/:connection_id", httpserver.AuthorizeHandler(h.CreateBenchmarkAssignment, authApi.EditorRole))
	assignments.DELETE("/:benchmark_id/connection/:connection_id", httpserver.AuthorizeHandler(h.DeleteBenchmarkAssignment, authApi.EditorRole))

	metadata := v1.Group("/metadata")
	metadata.GET("/tag/insight", httpserver.AuthorizeHandler(h.ListInsightTags, authApi.ViewerRole))
	metadata.GET("/insight", httpserver.AuthorizeHandler(h.ListInsightsMetadata, authApi.ViewerRole))
	metadata.GET("/insight/:insightId", httpserver.AuthorizeHandler(h.GetInsightMetadata, authApi.ViewerRole))

	insights := v1.Group("/insight")
	insightGroups := insights.Group("/group")
	insightGroups.GET("", httpserver.AuthorizeHandler(h.ListInsightGroups, authApi.ViewerRole))
	insights.GET("", httpserver.AuthorizeHandler(h.ListInsights, authApi.ViewerRole))
	insights.GET("/:insightId", httpserver.AuthorizeHandler(h.GetInsight, authApi.ViewerRole))
	insights.GET("/:insightId/trend", httpserver.AuthorizeHandler(h.GetInsightTrend, authApi.ViewerRole))

	findings := v1.Group("/findings")
	findings.POST("", httpserver.AuthorizeHandler(h.GetFindings, authApi.ViewerRole))
	findings.GET("/:benchmarkId/:field/top/:count", httpserver.AuthorizeHandler(h.GetTopFieldByFindingCount, authApi.ViewerRole))
}

func bindValidate(ctx echo.Context, i any) error {
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
	var sorts []map[string]any
	for _, sortItem := range req.Sorts {
		item := map[string]any{}
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
	res, err := es.FindingsQuery(
		h.client, req.Filters.ResourceID, req.Filters.Connector, req.Filters.ConnectionID,
		benchmarkIDs, req.Filters.PolicyID, req.Filters.Severity,
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
//	@Param			benchmarkId		path		string							true	"BenchmarkID"
//	@Param			field			path		string							true	"Field"	Enums(resourceType,connectionID,resourceID,service)
//	@Param			count			path		int								true	"Count"
//	@Param			connectionId	query		[]string						false	"Connection IDs to filter by"
//	@Param			connector		query		[]source.Type					false	"Connector type to filter by"
//	@Param			severities		query		[]kaytuTypes.FindingSeverity	false	"Severities to filter by"
//	@Success		200				{object}	api.GetTopFieldResponse
//	@Router			/compliance/api/v1/findings/{benchmarkId}/{field}/top/{count} [get]
func (h *HttpHandler) GetTopFieldByFindingCount(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")
	field := ctx.Param("field")
	esField := field
	countStr := ctx.Param("count")
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return err
	}
	esCount := count

	if field == "service" {
		esField = "resourceType"
		esCount = 10000
	}

	connectionIDs := ctx.QueryParams()["connectionId"]
	connectors := source.ParseTypes(ctx.QueryParams()["connector"])
	severities := kaytuTypes.ParseFindingSeverities(ctx.QueryParams()["severities"])

	benchmarkIDs, err := h.GetBenchmarkTreeIDs(benchmarkID)
	if err != nil {
		return err
	}

	var response api.GetTopFieldResponse
	res, err := es.FindingsTopFieldQuery(
		h.logger, h.client, esField,
		connectors, nil, connectionIDs,
		benchmarkIDs, nil, severities, esCount)
	if err != nil {
		return err
	}

	switch field {
	case "service":
		resourceTypeList := make([]string, 0, len(res.Aggregations.FieldFilter.Buckets))
		for _, item := range res.Aggregations.FieldFilter.Buckets {
			resourceTypeList = append(resourceTypeList, item.Key)
		}
		resourceTypeMetadata, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx),
			nil, nil, resourceTypeList, false, nil, 10000, 1)
		if err != nil {
			return err
		}
		resourceTypeMetadataMap := make(map[string]inventoryApi.ResourceType)
		for _, item := range resourceTypeMetadata.ResourceTypes {
			resourceTypeMetadataMap[strings.ToLower(item.ResourceType)] = item
		}
		serviceCountMap := make(map[string]int)
		for _, item := range res.Aggregations.FieldFilter.Buckets {
			if rtMetadata, ok := resourceTypeMetadataMap[strings.ToLower(item.Key)]; ok {
				serviceCountMap[rtMetadata.ServiceName] += item.DocCount
			}
		}
		serviceCountList := make([]api.TopFieldRecord, 0, len(serviceCountMap))
		for k, v := range serviceCountMap {
			serviceCountList = append(serviceCountList, api.TopFieldRecord{
				Value: k,
				Count: v,
			})
		}
		sort.Slice(serviceCountList, func(i, j int) bool {
			return serviceCountList[i].Count > serviceCountList[j].Count
		})
		response.Records = serviceCountList[:count]
		response.TotalCount = len(serviceCountList)
	default:
		for _, item := range res.Aggregations.FieldFilter.Buckets {
			response.Records = append(response.Records, api.TopFieldRecord{
				Value: item.Key,
				Count: item.DocCount,
			})
		}
		response.TotalCount = res.Aggregations.BucketCount.Value
	}

	return ctx.JSON(http.StatusOK, response)
}

// ListBenchmarksSummary godoc
//
//	@Summary		List benchmarks summaries
//	@Description	This API enables users to retrieve a summary of all benchmarks and their associated checks and results within a specified time interval. Users can use this API to obtain an overview of all benchmarks, including their names, descriptions, and other relevant information, as well as the checks and their corresponding results within the specified time period.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			timeAt			query		int				false	"timestamp for values in epoch seconds"
//	@Success		200				{object}	api.GetBenchmarksSummaryResponse
//	@Router			/compliance/api/v1/benchmarks/summary [get]
func (h *HttpHandler) ListBenchmarksSummary(ctx echo.Context) error {
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "too many connection IDs")
	}
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	timeAt := time.Now()
	if timeAtStr := ctx.QueryParam("timeAt"); timeAtStr != "" {
		timeAtInt, err := strconv.ParseInt(timeAtStr, 10, 64)
		if err != nil {
			return err
		}
		timeAt = time.Unix(timeAtInt, 0)
	}
	var response api.GetBenchmarksSummaryResponse
	benchmarks, err := h.db.ListRootBenchmarks()
	if err != nil {
		return err
	}

	benchmarkIDs := make([]string, 0, len(benchmarks))
	for _, b := range benchmarks {
		benchmarkIDs = append(benchmarkIDs, b.ID)
	}

	summariesAtTime, err := es.FetchBenchmarkSummariesAtTime(h.logger, h.client, benchmarkIDs, connectors, connectionIDs, timeAt)
	if err != nil {
		h.logger.Error("failed to fetch benchmark summaries", zap.Error(err))
		return err
	}

	for _, b := range benchmarks {
		be := b.ToApi()
		err = b.PopulateConnectors(h.db, &be)
		if err != nil {
			return err
		}

		if len(connectors) > 0 && !utils.IncludesAny(be.Connectors, connectors) {
			continue
		}

		summaryAtTime := summariesAtTime[b.ID]
		response.BenchmarkSummary = append(response.BenchmarkSummary, api.BenchmarkEvaluationSummary{
			ID:          b.ID,
			Title:       b.Title,
			Description: b.Description,
			Connectors:  be.Connectors,
			Tags:        be.Tags,
			Enabled:     b.Enabled,
			Result:      summaryAtTime.ComplianceResultSummary,
			Checks:      summaryAtTime.SeverityResult,
			EvaluatedAt: summaryAtTime.EvaluatedAt,
		})

		response.TotalResult.AddComplianceResultSummary(summaryAtTime.ComplianceResultSummary)
		response.TotalChecks.AddSeverityResult(summaryAtTime.SeverityResult)
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
//	@Param			benchmark_id	path		string			true	"Benchmark ID"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			timeAt			query		int				false	"timestamp for values in epoch seconds"
//	@Success		200				{object}	api.BenchmarkEvaluationSummary
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id}/summary [get]
func (h *HttpHandler) GetBenchmarkSummary(ctx echo.Context) error {
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "too many connection IDs")
	}
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	timeAt := time.Now()
	if timeAtStr := ctx.QueryParam("timeAt"); timeAtStr != "" {
		timeAtInt, err := strconv.ParseInt(timeAtStr, 10, 64)
		if err != nil {
			return err
		}
		timeAt = time.Unix(timeAtInt, 0)
	}
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
	if len(connectors) > 0 && !utils.IncludesAny(be.Connectors, connectors) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connector")
	}

	summariesAtTime, err := es.FetchBenchmarkSummariesAtTime(h.logger, h.client, []string{benchmarkID}, connectors, connectionIDs, timeAt)
	if err != nil {
		return err
	}

	summaryAtTime := summariesAtTime[benchmarkID]
	response := api.BenchmarkEvaluationSummary{
		ID:          benchmark.ID,
		Title:       benchmark.Title,
		Description: benchmark.Description,
		Connectors:  be.Connectors,
		Tags:        be.Tags,
		Enabled:     benchmark.Enabled,
		Result:      summaryAtTime.ComplianceResultSummary,
		Checks:      summaryAtTime.SeverityResult,
		EvaluatedAt: summaryAtTime.EvaluatedAt,
	}
	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmarkTree godoc
//
//	@Summary		Get benchmark tree
//	@Description	This API retrieves the benchmark tree, including all of its child benchmarks. Users can use this API to obtain a comprehensive overview of the benchmarks within a particular category or hierarchy.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id	path		string	true	"Benchmark ID"
//	@Success		200				{object}	api.BenchmarkTree
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id}/tree [get]
func (h *HttpHandler) GetBenchmarkTree(ctx echo.Context) error {
	var status []kaytuTypes.PolicyStatus
	benchmarkID := ctx.Param("benchmark_id")
	for k, va := range ctx.QueryParams() {
		if k == "status" || k == "status[]" {
			for _, v := range va {
				status = append(status, kaytuTypes.PolicyStatus(v))
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

func GetBenchmarkTree(db db.Database, client keibi.Client, b db.Benchmark, status []kaytuTypes.PolicyStatus) (api.BenchmarkTree, error) {
	tree := api.BenchmarkTree{
		ID:       b.ID,
		Title:    b.Title,
		Children: nil,
		Policies: nil,
	}
	for _, child := range b.Children {
		childObj, err := db.GetBenchmark(child.ID)
		if err != nil {
			return tree, err
		}

		childTree, err := GetBenchmarkTree(db, client, *childObj, status)
		if err != nil {
			return tree, err
		}

		tree.Children = append(tree.Children, childTree)
	}

	res, err := es.ListBenchmarkSummaries(client, &b.ID)
	if err != nil {
		return tree, err
	}

	for _, policy := range b.Policies {
		pt := api.PolicyTree{
			ID:          policy.ID,
			Title:       policy.Title,
			Severity:    policy.Severity,
			Status:      kaytuTypes.PolicyStatusPASSED,
			LastChecked: 0,
		}

		for _, bs := range res {
			for _, ps := range bs.Policies {
				if ps.PolicyID == policy.ID {
					pt.LastChecked = bs.EvaluatedAt
					pt.Status = kaytuTypes.PolicyStatusPASSED
					if ps.TotalResult.AlarmCount > 0 || ps.TotalResult.ErrorCount > 0 {
						pt.Status = kaytuTypes.PolicyStatusFAILED
					} else if ps.TotalResult.InfoCount > 0 || ps.TotalResult.SkipCount > 0 {
						pt.Status = kaytuTypes.PolicyStatusUNKNOWN
					}
				}
			}
		}
		if len(status) > 0 {
			contains := false
			for _, s := range status {
				if s == pt.Status {
					contains = true
				}
			}

			if !contains {
				continue
			}
		}
		tree.Policies = append(tree.Policies, pt)
	}

	return tree, nil
}

// GetBenchmarkTrend godoc
//
//	@Summary		Get benchmark trend
//	@Description	This API enables users to retrieve a trend of a benchmark result and checks
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id	path		string			true	"Benchmark ID"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			startTime		query		int				false	"timestamp for start of the chart in epoch seconds"
//	@Param			endTime			query		int				false	"timestamp for end of the chart in epoch seconds"
//	@Success		200				{object}	[]api.BenchmarkTrendDatapoint
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id}/trend [get]
func (h *HttpHandler) GetBenchmarkTrend(ctx echo.Context) error {
	connectionIDs := httpserver.QueryArrayParam(ctx, "connectionId")
	if len(connectionIDs) > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "too many connection IDs")
	}
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	endTime := time.Now()
	if endTimeStr := ctx.QueryParam("timeAt"); endTimeStr != "" {
		endTimeInt, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return err
		}
		endTime = time.Unix(endTimeInt, 0)
	}
	startTime := endTime.AddDate(0, 0, -7)
	if startTimeStr := ctx.QueryParam("startTime"); startTimeStr != "" {
		startTimeInt, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return err
		}
		startTime = time.Unix(startTimeInt, 0)
	}
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
	if len(connectors) > 0 && !utils.IncludesAny(be.Connectors, connectors) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connector")
	}

	datapointCount := int(endTime.Sub(startTime).Hours() / 24)
	if datapointCount > 30 {
		datapointCount = 30
	}
	if datapointCount < 1 {
		datapointCount = 1
	}

	evaluationAcrossTime, err := es.FetchBenchmarkSummaryTrend(h.logger, h.client, []string{benchmarkID}, connectors, connectionIDs, startTime, endTime, datapointCount)
	if err != nil {
		return err
	}

	response := make([]api.BenchmarkTrendDatapoint, 0, datapointCount)
	for timeKey, datapoint := range evaluationAcrossTime[benchmarkID] {
		response = append(response, api.BenchmarkTrendDatapoint{
			Timestamp: timeKey,
			Result:    datapoint.ComplianceResultSummary,
			Checks:    datapoint.SeverityResult,
		})
	}

	sort.Slice(response, func(i, j int) bool {
		return response[i].Timestamp < response[j].Timestamp
	})

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
//	@Param			connection_id	path		string	true	"Connection ID or 'all' for everything"
//	@Success		200				{object}	[]api.BenchmarkAssignment
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

	connectorType := source.Nil
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

		if t, _ := source.ParseType(q.Connector); t != source.Nil {
			connectorType = t
			break
		}
	}

	connections := make([]onboardApi.Connection, 0)
	if strings.ToLower(connectionID) == "all" {
		srcs, err := h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), nil)
		if err != nil {
			return err
		}
		for _, src := range srcs {
			if src.Connector == connectorType &&
				(src.LifecycleState == onboardApi.ConnectionLifecycleStateOnboard || src.LifecycleState == onboardApi.ConnectionLifecycleStateInProgress) {
				connections = append(connections, src)
			}
		}
	} else {
		src, err := h.onboardClient.GetSource(httpclient.FromEchoContext(ctx), connectionID)
		if err != nil {
			return err
		}
		connections = append(connections, *src)
	}

	result := make([]api.BenchmarkAssignment, 0, len(connections))
	for _, src := range connections {
		assignment := &db.BenchmarkAssignment{
			BenchmarkId:  benchmarkId,
			ConnectionId: src.ConnectionID,
			AssignedAt:   time.Now(),
		}
		if err := h.db.AddBenchmarkAssignment(assignment); err != nil {
			ctx.Logger().Errorf("add benchmark assignment: %v", err)
			return err
		}
		result = append(result, api.BenchmarkAssignment{
			BenchmarkId:  benchmarkId,
			ConnectionId: connectionID,
			AssignedAt:   assignment.AssignedAt,
		})
	}

	return ctx.JSON(http.StatusOK, result)
}

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
			AssignedAt:   assignment.AssignedAt,
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
				ConnectionID:           connection.ID.String(),
				ProviderConnectionID:   connection.ConnectionID,
				ProviderConnectionName: connection.ConnectionName,
				Connector:              connector,
				Status:                 false,
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

// DeleteBenchmarkAssignment godoc
//
//	@Summary		Delete benchmark assignment for inventory service
//	@Description	Delete benchmark assignment with source id and benchmark id
//	@Security		BearerToken
//	@Tags			benchmarks_assignment
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id	path	string	true	"Benchmark ID"
//	@Param			connection_id	path	string	true	"Connection ID or 'all' for everything"
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
	if strings.ToLower(connectionId) == "all" {
		if err := h.db.DeleteBenchmarkAssignmentByBenchmarkId(benchmarkId); err != nil {
			h.logger.Error("delete benchmark assignment by benchmark id", zap.Error(err))
			return err
		}
	} else {
		if _, err := h.db.GetBenchmarkAssignmentByIds(connectionId, benchmarkId); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return echo.NewHTTPError(http.StatusFound, "benchmark assignment not found")
			}
			ctx.Logger().Errorf("find benchmark assignment: %v", err)
			return err
		}

		if err := h.db.DeleteBenchmarkAssignmentByIds(connectionId, benchmarkId); err != nil {
			ctx.Logger().Errorf("delete benchmark assignment: %v", err)
			return err
		}
	}

	return ctx.NoContent(http.StatusOK)
}

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

func (h *HttpHandler) getBenchmarkPolicies(benchmarkID string) ([]db.Policy, error) {
	b, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		return nil, err
	}

	if b == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	var policyIDs []string
	for _, p := range b.Policies {
		policyIDs = append(policyIDs, p.ID)
	}

	policies, err := h.db.GetPolicies(policyIDs)
	if err != nil {
		return nil, err
	}

	for _, child := range b.Children {
		childPolicies, err := h.getBenchmarkPolicies(child.ID)
		if err != nil {
			return nil, err
		}
		policies = append(policies, childPolicies...)
	}

	return policies, nil
}

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
