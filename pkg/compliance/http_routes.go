package compliance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	ConnectionIdParam    = "connectionId"
	ConnectionGroupParam = "connectionGroup"
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
	assignments.GET("/connection", httpserver.AuthorizeHandler(h.ListAssignmentsByConnection, authApi.ViewerRole))
	assignments.POST("/:benchmark_id/connection", httpserver.AuthorizeHandler(h.CreateBenchmarkAssignment, authApi.EditorRole))
	assignments.DELETE("/:benchmark_id/connection", httpserver.AuthorizeHandler(h.DeleteBenchmarkAssignment, authApi.EditorRole))

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

func (h *HttpHandler) getConnectionIdFilterFromParams(ctx echo.Context) ([]string, error) {
	connectionIds := httpserver.QueryArrayParam(ctx, ConnectionIdParam)
	connectionGroup := httpserver.QueryArrayParam(ctx, ConnectionGroupParam)
	if len(connectionIds) == 0 && len(connectionGroup) == 0 {
		return nil, nil
	}

	if len(connectionIds) > 0 && len(connectionGroup) > 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "connectionId and connectionGroup cannot be used together")
	}

	if len(connectionIds) > 0 {
		return connectionIds, nil
	}

	check := make(map[string]bool)
	var connectionIDSChecked []string

	for i := 0; i < len(connectionGroup); i++ {
		connectionGroupObj, err := h.onboardClient.GetConnectionGroup(&httpclient.Context{UserRole: authApi.KaytuAdminRole}, connectionGroup[i])
		if err != nil {
			return nil, err
		}
		if len(connectionGroupObj.ConnectionIds) == 0 {
			return nil, err
		}

		// Check for duplicate connection groups
		for _, entry := range connectionGroupObj.ConnectionIds {
			if _, value := check[entry]; !value {
				check[entry] = true
				connectionIDSChecked = append(connectionIDSChecked, entry)
			}
		}
	}
	connectionIds = connectionIDSChecked

	return connectionIds, nil
}

var tracer = otel.Tracer("new_compliance")

// GetFindings godoc
//
//	@Summary		Get findings
//	@Description	Retrieving all compliance run findings with respect to filters.
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
	// tracer :
	output1, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmarkTreeIDs(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmarkTreeIDs(loop)")

	for _, b := range req.Filters.BenchmarkID {
		// tracer :
		output2, span2 := tracer.Start(output1, "new_GetBenchmarkTreeIDs", trace.WithSpanKind(trace.SpanKindServer))
		span2.SetName("new_GetBenchmarkTreeIDs")
		bs, err := h.GetBenchmarkTreeIDs(output2, b)
		if err != nil {
			span2.RecordError(err)
			span2.SetStatus(codes.Error, err.Error())
			return err
		}
		span2.AddEvent("information", trace.WithAttributes(
			attribute.String("benchmark id", b),
		))
		span2.End()

		benchmarkIDs = append(benchmarkIDs, bs...)
	}
	span1.End()

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
//	@Description	Retrieving the top field by finding count.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmarkId		path		string							true	"BenchmarkID"
//	@Param			field			path		string							true	"Field"	Enums(resourceType,connectionID,resourceID,service)
//	@Param			count			path		int								true	"Count"
//	@Param			connectionId	query		[]string						false	"Connection IDs to filter by"
//	@Param			connectionGroup	query		[]string						false	"Connection groups to filter by "
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

	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	connectors := source.ParseTypes(ctx.QueryParams()["connector"])
	severities := kaytuTypes.ParseFindingSeverities(ctx.QueryParams()["severities"])
	//tracer :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmarkTreeIDs", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmarkTreeIDs")

	benchmarkIDs, err := h.GetBenchmarkTreeIDs(ctx.Request().Context(), benchmarkID)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("benchmark id", benchmarkID),
	))
	span1.End()

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
//	@Description	Retrieving a summary of all benchmarks and their associated checks and results within a specified time interval.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			connectionGroup	query		[]string		false	"Connection groups to filter by "
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			timeAt			query		int				false	"timestamp for values in epoch seconds"
//	@Success		200				{object}	api.GetBenchmarksSummaryResponse
//	@Router			/compliance/api/v1/benchmarks/summary [get]
func (h *HttpHandler) ListBenchmarksSummary(ctx echo.Context) error {
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
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

	// tracer :
	outputS, span2 := tracer.Start(ctx.Request().Context(), "new_ListRootBenchmarks", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_ListRootBenchmarks")

	benchmarks, err := h.db.ListRootBenchmarks()
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return err
	}
	span2.End()

	benchmarkIDs := make([]string, 0, len(benchmarks))
	for _, b := range benchmarks {
		benchmarkIDs = append(benchmarkIDs, b.ID)
	}

	summariesAtTime, err := es.FetchBenchmarkSummariesAtTime(h.logger, h.client, benchmarkIDs, connectors, connectionIDs, timeAt)
	if err != nil {
		h.logger.Error("failed to fetch benchmark summaries", zap.Error(err))
		return err
	}
	// tracer :
	output3, span3 := tracer.Start(outputS, "new_PopulateConnectors(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span3.SetName("new_PopulateConnectors(loop)")

	for _, b := range benchmarks {
		be := b.ToApi()
		//tracer :
		outputs4, span4 := tracer.Start(output3, "new_PopulateConnectors", trace.WithSpanKind(trace.SpanKindServer))
		span4.SetName("new_PopulateConnectors")

		err = b.PopulateConnectors(outputs4, h.db, &be)
		if err != nil {
			span4.RecordError(err)
			span4.SetStatus(codes.Error, err.Error())
			return err
		}
		span4.End()
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
	span3.End()
	return ctx.JSON(http.StatusOK, response)
}

// GetBenchmarkSummary godoc
//
//	@Summary		Get benchmark summary
//	@Description	Retrieving a summary of a benchmark and its associated checks and results.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id	path		string			true	"Benchmark ID"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			connectionGroup	query		[]string		false	"Connection groups to filter by "
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			timeAt			query		int				false	"timestamp for values in epoch seconds"
//	@Success		200				{object}	api.BenchmarkEvaluationSummary
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id}/summary [get]
func (h *HttpHandler) GetBenchmarkSummary(ctx echo.Context) error {
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
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
	// tracer :
	output1, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmark")

	benchmark, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("benchmark ID", benchmark.ID),
	))
	span1.End()

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid benchmarkID")
	}
	be := benchmark.ToApi()
	//tracer :
	outputS, span2 := tracer.Start(output1, "new_PopulateConnectors", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_PopulateConnectors")

	err = benchmark.PopulateConnectors(outputS, h.db, &be)
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return err
	}
	span2.End()

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
//	@Description	Retrieving the benchmark tree, including all of its child benchmarks.
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
	// tracer :
	output1, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmark")

	benchmark, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("benchmark ID", benchmark.ID),
	))
	span1.End()

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid benchmarkID")
	}
	outputS2, span2 := tracer.Start(output1, "new_GetBenchmarkTree")

	response, err := GetBenchmarkTree(outputS2, h.db, h.client, *benchmark, status)
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return err
	}
	span2.End()
	return ctx.JSON(http.StatusOK, response)
}

func GetBenchmarkTree(ctx context.Context, db db.Database, client kaytu.Client, b db.Benchmark, status []kaytuTypes.PolicyStatus) (api.BenchmarkTree, error) {
	tree := api.BenchmarkTree{
		ID:       b.ID,
		Title:    b.Title,
		Children: nil,
		Policies: nil,
	}
	// tracer :
	output1, span1 := tracer.Start(ctx, "new_GetBenchmark(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmark(loop)")

	for _, child := range b.Children {
		// tracer :
		_, span2 := tracer.Start(output1, "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindServer))
		span2.SetName("new_GetBenchmark")

		childObj, err := db.GetBenchmark(child.ID)
		if err != nil {
			span2.RecordError(err)
			span2.SetStatus(codes.Error, err.Error())
			return tree, err
		}
		span2.SetAttributes(
			attribute.String("benchmark ID", childObj.ID),
		)
		span2.End()

		childTree, err := GetBenchmarkTree(ctx, db, client, *childObj, status)
		if err != nil {
			return tree, err
		}

		tree.Children = append(tree.Children, childTree)
	}
	span1.End()

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
//	@Description	Retrieving a trend of a benchmark result and checks.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id	path		string			true	"Benchmark ID"
//	@Param			connectionId	query		[]string		false	"Connection IDs to filter by"
//	@Param			connectionGroup	query		[]string		false	"Connection groups to filter by "
//	@Param			connector		query		[]source.Type	false	"Connector type to filter by"
//	@Param			startTime		query		int				false	"timestamp for start of the chart in epoch seconds"
//	@Param			endTime			query		int				false	"timestamp for end of the chart in epoch seconds"
//	@Success		200				{object}	[]api.BenchmarkTrendDatapoint
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id}/trend [get]
func (h *HttpHandler) GetBenchmarkTrend(ctx echo.Context) error {
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
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
	// tracer :
	output1, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmark")
	span1.SetName("new_GetBenchmark")

	benchmark, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("benchmark ID", benchmark.ID),
	))
	span1.End()

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid benchmarkID")
	}

	be := benchmark.ToApi()
	// tracer :
	outputS2, span2 := tracer.Start(output1, "new_PopulateConnectors", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_PopulateConnectors")

	err = benchmark.PopulateConnectors(outputS2, h.db, &be)
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return err
	}
	span2.End()

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
//	@Summary		Create benchmark assignment
//	@Description	Creating a benchmark assignment for a connection.
//	@Security		BearerToken
//	@Tags			benchmarks_assignment
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id		path		string	true	"Benchmark ID"
//	@Param			connection_id		query		string	false	"Connection ID or 'all' for everything"
//	@Param			connection_group	query		string	false	"Connection group "
//	@Success		200					{object}	[]api.BenchmarkAssignment
//	@Router			/compliance/api/v1/assignments/{benchmark_id}/connection/{connection_id} [post]
func (h *HttpHandler) CreateBenchmarkAssignment(ctx echo.Context) error {
	connectionID, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	benchmarkId := ctx.Param("benchmark_id")
	if benchmarkId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmark id is empty")
	}
	// trace :
	outputS1, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmark")

	benchmark, err := h.db.GetBenchmark(benchmarkId)

	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("benchmark Id", benchmark.ID),
	))
	span1.End()

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark %s not found", benchmarkId))
	}

	connectorType := source.Nil
	// trace :
	outputS2, span2 := tracer.Start(outputS1, "new_GetQuery(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_GetQuery(loop)")

	for _, policy := range benchmark.Policies {
		if policy.QueryID == nil {
			continue
		}
		//trace :
		_, span3 := tracer.Start(outputS2, "new_GetQuery", trace.WithSpanKind(trace.SpanKindServer))
		span3.SetName("new_GetQuery")

		q, err := h.db.GetQuery(*policy.QueryID)
		if err != nil {
			span3.RecordError(err)
			span3.SetStatus(codes.Error, err.Error())
			return err
		}
		span3.SetAttributes(
			attribute.String("query ID", q.ID),
		)
		span3.End()

		if q == nil {
			return fmt.Errorf("query %s not found", *policy.QueryID)
		}

		if t, _ := source.ParseType(q.Connector); t != source.Nil {
			connectorType = t
			break
		}
	}
	span2.End()

	connections := make([]onboardApi.Connection, 0)
	if len(connectionID) == 1 {
		if strings.ToLower(connectionID[0]) == "all" {
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
			src, err := h.onboardClient.GetSource(httpclient.FromEchoContext(ctx), connectionID[0])
			if err != nil {
				return err
			}
			connections = append(connections, *src)
		}
	} else {
		for _, connectionId := range connectionID {
			src, err := h.onboardClient.GetSource(httpclient.FromEchoContext(ctx), connectionId)
			if err != nil {
				return err
			}
			connections = append(connections, *src)
		}
	}

	result := make([]api.BenchmarkAssignment, 0, len(connections))
	// trace :
	output4, span4 := tracer.Start(outputS1, "new_AddBenchmarkAssignment(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span4.SetName("new_AddBenchmarkAssignment(loop)")

	for _, src := range connections {
		assignment := &db.BenchmarkAssignment{
			BenchmarkId:  benchmarkId,
			ConnectionId: src.ConnectionID,
			AssignedAt:   time.Now(),
		}
		//trace :
		_, span5 := tracer.Start(output4, "new_AddBenchmarkAssignment", trace.WithSpanKind(trace.SpanKindServer))
		span5.SetName("new_AddBenchmarkAssignment")

		if err := h.db.AddBenchmarkAssignment(assignment); err != nil {
			span5.RecordError(err)
			span5.SetStatus(codes.Error, err.Error())
			ctx.Logger().Errorf("add benchmark assignment: %v", err)
			return err
		}
		span5.SetAttributes(
			attribute.String("Benchmark ID", assignment.BenchmarkId),
		)
		span5.End()

		for _, connectionId := range connectionID {
			result = append(result, api.BenchmarkAssignment{
				BenchmarkId:  benchmarkId,
				ConnectionId: connectionId,
				AssignedAt:   assignment.AssignedAt,
			})
		}
	}
	span4.End()
	return ctx.JSON(http.StatusOK, result)
}

func (h *HttpHandler) ListAssignmentsByConnection(ctx echo.Context) error {
	connectionID, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	var dbAssignments [][]db.BenchmarkAssignment
	if len(connectionID) == 1 {
		// trace :
		_, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmarkAssignmentsBySourceId", trace.WithSpanKind(trace.SpanKindServer))
		span1.SetName("new_GetBenchmarkAssignmentsBySourceId")

		dbAssignments[0], err = h.db.GetBenchmarkAssignmentsBySourceId(connectionID[0])
		if err != nil {
			span1.RecordError(err)
			span1.SetStatus(codes.Error, err.Error())
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark assignments for %s not found", connectionID[0]))
			}
			ctx.Logger().Errorf("find benchmark assignments by source %s: %v", connectionID[0], err)
			return err
		}
		span1.AddEvent("information", trace.WithAttributes(
			attribute.String("connection ID", connectionID[0]),
		))
		span1.End()
	} else {
		// trace :
		outputS2, span2 := tracer.Start(ctx.Request().Context(), "new_GetBenchmarkAssignmentsBySourceId(loop)", trace.WithSpanKind(trace.SpanKindServer))
		span2.SetName("new_GetBenchmarkAssignmentsBySourceId(loop)")

		for _, connectionId := range connectionID {
			// trace :
			_, span1 := tracer.Start(outputS2, "new_GetBenchmarkAssignmentsBySourceId", trace.WithSpanKind(trace.SpanKindServer))
			span1.SetName("new_GetBenchmarkAssignmentsBySourceId")

			dbAssignmentsCG, err := h.db.GetBenchmarkAssignmentsBySourceId(connectionId)
			if err != nil {
				span1.RecordError(err)
				span1.SetStatus(codes.Error, err.Error())
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark assignments for %s not found", connectionId))
				}
				ctx.Logger().Errorf("find benchmark assignments by source %s: %v", connectionId, err)
				return err
			}
			dbAssignments = append(dbAssignments, dbAssignmentsCG)

			span1.AddEvent("information", trace.WithAttributes(
				attribute.String("connection ID", connectionId),
			))
			span1.End()
		}
		span2.End()
	}

	var assignments []api.BenchmarkAssignment
	for _, assignmentsArray := range dbAssignments {
		for _, assignment := range assignmentsArray {
			assignments = append(assignments, api.BenchmarkAssignment{
				BenchmarkId:  assignment.BenchmarkId,
				ConnectionId: assignment.ConnectionId,
				AssignedAt:   assignment.AssignedAt,
			})
		}
	}

	return ctx.JSON(http.StatusOK, assignments)
}

// ListAssignmentsByBenchmark godoc
//
//	@Summary		Get benchmark assigned sources
//	@Description	Retrieving all benchmark assigned sources with benchmark id
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
	// trace :
	outputS, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmark")

	benchmark, err := h.db.GetBenchmark(benchmarkId)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("benchmark ID", benchmark.ID),
	))
	span1.End()

	var apiBenchmark api.Benchmark
	// tracer :
	outputS2, span2 := tracer.Start(outputS, "new_PopulateConnectors", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_PopulateConnectors")

	err = benchmark.PopulateConnectors(outputS2, h.db, &apiBenchmark)
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return err
	}
	span2.End()

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
	// trace :
	_, span3 := tracer.Start(outputS, "new_GetBenchmarkAssignmentsByBenchmarkId", trace.WithSpanKind(trace.SpanKindServer))
	span3.SetName("new_GetBenchmarkAssignmentsByBenchmarkId")

	dbAssignments, err := h.db.GetBenchmarkAssignmentsByBenchmarkId(benchmarkId)
	if err != nil {
		span3.RecordError(err)
		span3.SetStatus(codes.Error, err.Error())
		return err
	}
	span3.AddEvent("information", trace.WithAttributes(
		attribute.String("benchmark ID", benchmarkId),
	))
	span3.End()

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
//	@Summary		Delete benchmark assignment
//	@Description	Delete benchmark assignment with source id and benchmark id
//	@Security		BearerToken
//	@Tags			benchmarks_assignment
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id		path	string	true	"Benchmark ID"
//	@Param			connection_id		query	string	false	"Connection ID or 'all' for everything"
//	@Param			connection_group	query	string	false	"Connection Group "
//	@Success		200
//	@Router			/compliance/api/v1/assignments/{benchmark_id}/connection/{connection_id} [delete]
func (h *HttpHandler) DeleteBenchmarkAssignment(ctx echo.Context) error {
	connectionID, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	benchmarkId := ctx.Param("benchmark_id")
	if benchmarkId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmark id is empty")
	}

	if len(connectionID) == 1 {
		if strings.ToLower(connectionID[0]) == "all" {
			//trace :
			_, span1 := tracer.Start(ctx.Request().Context(), "new_DeleteBenchmarkAssignmentByBenchmarkId", trace.WithSpanKind(trace.SpanKindServer))
			span1.SetName("new_DeleteBenchmarkAssignmentByBenchmarkId")

			if err := h.db.DeleteBenchmarkAssignmentByBenchmarkId(benchmarkId); err != nil {
				span1.RecordError(err)
				span1.SetStatus(codes.Error, err.Error())
				h.logger.Error("delete benchmark assignment by benchmark id", zap.Error(err))
				return err
			}
			span1.AddEvent("information", trace.WithAttributes(
				attribute.String("benchmark ID", benchmarkId),
			))
			span1.End()
		} else {
			// trace :
			outputS1, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmarkAssignmentByIds", trace.WithSpanKind(trace.SpanKindServer))
			span1.SetName("new_GetBenchmarkAssignmentByIds")

			if _, err := h.db.GetBenchmarkAssignmentByIds(connectionID[0], benchmarkId); err != nil {
				span1.RecordError(err)
				span1.SetStatus(codes.Error, err.Error())
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return echo.NewHTTPError(http.StatusFound, "benchmark assignment not found")
				}
				ctx.Logger().Errorf("find benchmark assignment: %v", err)
				return err
			}
			span1.AddEvent("information", trace.WithAttributes(
				attribute.String("benchmark ID", benchmarkId),
			))
			span1.End()

			// trace :
			_, span2 := tracer.Start(outputS1, "new_DeleteBenchmarkAssignmentByIds", trace.WithSpanKind(trace.SpanKindServer))
			span2.SetName("new_DeleteBenchmarkAssignmentByIds")

			if err := h.db.DeleteBenchmarkAssignmentByIds(connectionID[0], benchmarkId); err != nil {
				span2.RecordError(err)
				span2.SetStatus(codes.Error, err.Error())
				ctx.Logger().Errorf("delete benchmark assignment: %v", err)
				return err
			}
			span2.AddEvent("information", trace.WithAttributes(
				attribute.String("benchmark ID", benchmarkId),
			))
			span2.End()
		}
	} else {
		// tracer :
		outputS5, span5 := tracer.Start(ctx.Request().Context(), "new_GetBenchmarkAssignmentByIds(loop)", trace.WithSpanKind(trace.SpanKindServer))
		span5.SetName("new_GetBenchmarkAssignmentByIds(loop)")

		for _, connectionId := range connectionID {
			// trace :
			outputS3, span3 := tracer.Start(outputS5, "new_GetBenchmarkAssignmentByIds", trace.WithSpanKind(trace.SpanKindServer))
			span3.SetName("new_GetBenchmarkAssignmentByIds")

			if _, err := h.db.GetBenchmarkAssignmentByIds(connectionId, benchmarkId); err != nil {
				span3.RecordError(err)
				span3.SetStatus(codes.Error, err.Error())
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return echo.NewHTTPError(http.StatusFound, "benchmark assignment not found")
				}
				ctx.Logger().Errorf("find benchmark assignment: %v", err)
				return err
			}
			span3.AddEvent("information", trace.WithAttributes(
				attribute.String("benchmark ID", benchmarkId),
			))
			span3.End()

			// trace :
			_, span4 := tracer.Start(outputS3, "new_DeleteBenchmarkAssignmentByIds", trace.WithSpanKind(trace.SpanKindServer))
			span4.SetName("new_DeleteBenchmarkAssignmentByIds")

			if err := h.db.DeleteBenchmarkAssignmentByIds(connectionId, benchmarkId); err != nil {
				span4.RecordError(err)
				span4.SetStatus(codes.Error, err.Error())
				ctx.Logger().Errorf("delete benchmark assignment: %v", err)
				return err
			}
			span4.AddEvent("information", trace.WithAttributes(
				attribute.String("benchmark ID", benchmarkId),
			))
			span4.End()
		}
		span5.End()
	}
	return ctx.NoContent(http.StatusOK)
}

func (h *HttpHandler) ListBenchmarks(ctx echo.Context) error {
	var response []api.Benchmark
	// trace :
	output1, span1 := tracer.Start(ctx.Request().Context(), "new_ListRootBenchmarks", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListRootBenchmarks")

	benchmarks, err := h.db.ListRootBenchmarks()
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	// tracer :
	output2, span2 := tracer.Start(output1, "new_PopulateConnectors(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_PopulateConnectors(loop)")

	for _, b := range benchmarks {
		be := b.ToApi()
		// tracer :
		outputS3, span3 := tracer.Start(output2, "new_PopulateConnectors", trace.WithSpanKind(trace.SpanKindServer))
		span3.SetName("new_PopulateConnectors")

		err = b.PopulateConnectors(outputS3, h.db, &be)
		if err != nil {
			span3.RecordError(err)
			span3.SetStatus(codes.Error, err.Error())
			return err
		}
		span3.SetAttributes(
			attribute.String("ID", be.ID),
		)
		span3.End()

		response = append(response, be)
	}
	span2.End()

	return ctx.JSON(http.StatusOK, response)
}

func (h *HttpHandler) GetBenchmark(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmark_id")
	// trace :
	output1, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmark")

	benchmark, err := h.db.GetBenchmark(benchmarkId)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("benchmark ID", benchmark.ID),
	))
	span1.End()

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}
	resp := benchmark.ToApi()

	outputS2, span2 := tracer.Start(output1, "new_PopulateConnectors", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_PopulateConnectors")

	err = benchmark.PopulateConnectors(outputS2, h.db, &resp)
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return err
	}
	span2.End()

	return ctx.JSON(http.StatusOK, resp)
}

func (h *HttpHandler) getBenchmarkPolicies(ctx context.Context, benchmarkID string) ([]db.Policy, error) {
	//trace :
	outputS, span1 := tracer.Start(ctx, "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmark")

	b, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("benchmark ID", b.ID),
	))
	span1.End()

	if b == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	var policyIDs []string
	for _, p := range b.Policies {
		policyIDs = append(policyIDs, p.ID)
	}
	//trace :
	_, span2 := tracer.Start(outputS, "new_GetPolicies", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_GetPolicies")

	policies, err := h.db.GetPolicies(policyIDs)
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span2.End()

	//tracer :
	output3, span3 := tracer.Start(outputS, "new_getBenchmarkPolicies(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span3.SetName("new_getBenchmarkPolicies(loop)")

	for _, child := range b.Children {
		// tracer :
		_, span4 := tracer.Start(output3, "new_getBenchmarkPolicies", trace.WithSpanKind(trace.SpanKindServer))
		span4.SetName("new_getBenchmarkPolicies")

		childPolicies, err := h.getBenchmarkPolicies(ctx, child.ID)
		if err != nil {
			span4.RecordError(err)
			span4.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span4.SetAttributes(
			attribute.String("benchmark ID", child.ID),
		)
		span4.End()

		policies = append(policies, childPolicies...)
	}
	span3.End()

	return policies, nil
}

func (h *HttpHandler) GetPolicy(ctx echo.Context) error {
	policyId := ctx.Param("policy_id")
	// trace :
	outputS, span1 := tracer.Start(ctx.Request().Context(), "new_GetPolicy", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetPolicy")

	policy, err := h.db.GetPolicy(policyId)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("policy ID", policyId),
	))
	span1.End()

	if policy == nil {
		return echo.NewHTTPError(http.StatusNotFound, "policy not found")
	}

	pa := policy.ToApi()
	// trace :
	outputS2, span2 := tracer.Start(outputS, "new_PopulateConnector", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_PopulateConnector")

	err = policy.PopulateConnector(outputS2, h.db, &pa)
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return err
	}
	span2.End()
	return ctx.JSON(http.StatusOK, pa)
}

func (h *HttpHandler) GetQuery(ctx echo.Context) error {
	queryID := ctx.Param("query_id")
	// trace :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_GetQuery", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetQuery")

	q, err := h.db.GetQuery(queryID)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("query ID", queryID),
	))
	span1.End()

	if q == nil {
		return echo.NewHTTPError(http.StatusNotFound, "query not found")
	}

	return ctx.JSON(http.StatusOK, q.ToApi())
}

// SyncQueries godoc
//
//	@Summary		Sync queries
//
//	@Description	Syncs queries with the git backend.
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
//	@Description	Retrieving a list of insights tag keys with their possible values.
//	@Security		BearerToken
//	@Tags			insights
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]string
//	@Router			/compliance/api/v1/metadata/tag/insight [get]
func (h *HttpHandler) ListInsightTags(ctx echo.Context) error {
	// trace :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_ListInsightTagKeysWithPossibleValues", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListInsightTagKeysWithPossibleValues")

	tags, err := h.db.ListInsightTagKeysWithPossibleValues()
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	tags = model.TrimPrivateTags(tags)
	return ctx.JSON(http.StatusOK, tags)
}

func (h *HttpHandler) ListInsightsMetadata(ctx echo.Context) error {
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	enabled := true
	// trace :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_ListInsightsWithFilters", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListInsightsWithFilters")

	insightRows, err := h.db.ListInsightsWithFilters(nil, connectors, &enabled, nil)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	var result []api.Insight
	for _, insightRow := range insightRows {
		result = append(result, insightRow.ToApi())
	}
	return ctx.JSON(200, result)
}

// GetInsightMetadata godoc
//
//	@Summary		Get insight metadata
//	@Description	Retrieving insight metadata by id
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
	// trace :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_GetInsight", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetInsight")

	insight, err := h.db.GetInsight(uint(id))
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "insight not found")
		}
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("query ID", insight.QueryID),
	))
	span1.End()

	result := insight.ToApi()

	return ctx.JSON(200, result)
}

// ListInsights godoc
//
//	@Summary		List insights
//	@Description	Retrieving list of insights based on specified filters. Provides details of insights, including results during the specified time period for the specified connection.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			tag				query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			connector		query		[]source.Type	false	"filter insights by connector"
//	@Param			connectionId	query		[]string		false	"filter the result by source id"
//	@Param			connectionGroup	query		[]string		false	"filter the result by connection group "
//	@Param			startTime		query		int				false	"unix seconds for the start time of the trend"
//	@Param			endTime			query		int				false	"unix seconds for the end time of the trend"
//	@Success		200				{object}	[]api.Insight
//	@Router			/compliance/api/v1/insight [get]
func (h *HttpHandler) ListInsights(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
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
	// trace :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_ListInsightsWithFilters", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListInsightsWithFilters")

	enabled := true
	insightRows, err := h.db.ListInsightsWithFilters(nil, connectors, &enabled, tagMap)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

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
//	@Description	Retrieving the specified insight with ID. Provides details of the insight, including results during the specified time period for the specified connection.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			insightId		path		string		true	"Insight ID"
//	@Param			connectionId	query		[]string	false	"filter the result by source id"
//	@param			connectionGroup	query		[]string	false	"filter the result by connection group"
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
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

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
	// trace :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_GetInsight", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetInsight")

	insightRow, err := h.db.GetInsight(uint(insightId))
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

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
//	@Description	Retrieving insight results datapoints for a specified connection during a specified time period.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			insightId		path		string		true	"Insight ID"
//	@Param			connectionId	query		[]string	false	"filter the result by source id"
//	@param			connectionGroup	query		[]string	false	"filter the result by connection group"
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
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
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
	// trace :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_GetInsight", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetInsight")

	insightRow, err := h.db.GetInsight(uint(insightId))
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("query ID", insightRow.QueryID),
	))
	span1.End()

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
//	@Description	Retrieving list of insight groups based on specified filters. The API provides details of insights, including results during the specified time period for the specified connection.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Accept			json
//	@Produce		json
//	@Param			tag				query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			connector		query		[]source.Type	false	"filter insights by connector"
//	@Param			connectionId	query		[]string		false	"filter the result by source id"
//	@param			connectionGroup	query		[]string		false	"filter the result by connection group"
//	@Param			startTime		query		int				false	"unix seconds for the start time of the trend"
//	@Param			endTime			query		int				false	"unix seconds for the end time of the trend"
//	@Success		200				{object}	[]api.InsightGroup
//	@Router			/compliance/api/v1/insight/group [get]
func (h *HttpHandler) ListInsightGroups(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

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

	// trace :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_ListInsightGroups", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListInsightGroups")

	insightGroupRows, err := h.db.ListInsightGroups(connectors, tagMap)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

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
