package compliance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/uuid"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	api "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/db"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/es"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/internal"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer/types"
	"github.com/kaytu-io/kaytu-engine/pkg/demo"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	httpserver2 "github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	insight "github.com/kaytu-io/kaytu-engine/pkg/insight/es"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	kaytuTypes "github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	es2 "github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/labstack/echo/v4"
	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"io"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"net/url"
	"os"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ConnectionIdParam    = "connectionId"
	ConnectionGroupParam = "connectionGroup"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	benchmarks := v1.Group("/benchmarks")

	benchmarks.GET("", httpserver2.AuthorizeHandler(h.ListBenchmarks, authApi.ViewerRole))
	benchmarks.GET("/all", httpserver2.AuthorizeHandler(h.ListAllBenchmarks, authApi.InternalRole))
	benchmarks.GET("/:benchmark_id", httpserver2.AuthorizeHandler(h.GetBenchmark, authApi.ViewerRole))
	benchmarks.GET("/controls/:control_id", httpserver2.AuthorizeHandler(h.GetControl, authApi.ViewerRole))
	benchmarks.GET("/controls", httpserver2.AuthorizeHandler(h.ListControls, authApi.InternalRole))
	benchmarks.GET("/queries", httpserver2.AuthorizeHandler(h.ListQueries, authApi.InternalRole))

	benchmarks.GET("/summary", httpserver2.AuthorizeHandler(h.ListBenchmarksSummary, authApi.ViewerRole))
	benchmarks.GET("/:benchmark_id/summary", httpserver2.AuthorizeHandler(h.GetBenchmarkSummary, authApi.ViewerRole))
	benchmarks.GET("/:benchmark_id/trend", httpserver2.AuthorizeHandler(h.GetBenchmarkTrend, authApi.ViewerRole))
	benchmarks.GET("/:benchmark_id/controls", httpserver2.AuthorizeHandler(h.GetBenchmarkControlsTree, authApi.ViewerRole))
	benchmarks.GET("/:benchmark_id/controls/:controlId", httpserver2.AuthorizeHandler(h.GetBenchmarkControl, authApi.ViewerRole))

	controls := v1.Group("/controls")
	controls.GET("/summary", httpserver2.AuthorizeHandler(h.ListControlsSummary, authApi.ViewerRole))
	controls.GET("/:controlId/summary", httpserver2.AuthorizeHandler(h.GetControlSummary, authApi.ViewerRole))
	controls.GET("/:controlId/trend", httpserver2.AuthorizeHandler(h.GetControlTrend, authApi.ViewerRole))

	queries := v1.Group("/queries")
	queries.GET("/:query_id", httpserver2.AuthorizeHandler(h.GetQuery, authApi.ViewerRole))
	queries.GET("/sync", httpserver2.AuthorizeHandler(h.SyncQueries, authApi.AdminRole))

	assignments := v1.Group("/assignments")
	assignments.GET("/benchmark/:benchmark_id", httpserver2.AuthorizeHandler(h.ListAssignmentsByBenchmark, authApi.ViewerRole))
	assignments.GET("/connection/:connection_id", httpserver2.AuthorizeHandler(h.ListAssignmentsByConnection, authApi.ViewerRole))
	assignments.GET("/resource_collection/:resource_collection_id", httpserver2.AuthorizeHandler(h.ListAssignmentsByResourceCollection, authApi.ViewerRole))
	assignments.POST("/:benchmark_id/connection", httpserver2.AuthorizeHandler(h.CreateBenchmarkAssignment, authApi.EditorRole))
	assignments.DELETE("/:benchmark_id/connection", httpserver2.AuthorizeHandler(h.DeleteBenchmarkAssignment, authApi.EditorRole))

	metadata := v1.Group("/metadata")
	metadata.GET("/tag/compliance", httpserver2.AuthorizeHandler(h.ListComplianceTags, authApi.ViewerRole))
	metadata.GET("/tag/insight", httpserver2.AuthorizeHandler(h.ListInsightTags, authApi.ViewerRole))
	metadata.GET("/insight", httpserver2.AuthorizeHandler(h.ListInsightsMetadata, authApi.ViewerRole))
	metadata.GET("/insight/:insightId", httpserver2.AuthorizeHandler(h.GetInsightMetadata, authApi.ViewerRole))

	insights := v1.Group("/insight")
	insightGroups := insights.Group("/group")
	insightGroups.GET("", httpserver2.AuthorizeHandler(h.ListInsightGroups, authApi.ViewerRole))
	insightGroups.GET("/:insightGroupId", httpserver2.AuthorizeHandler(h.GetInsightGroup, authApi.ViewerRole))
	insightGroups.GET("/:insightGroupId/trend", httpserver2.AuthorizeHandler(h.GetInsightGroupTrend, authApi.ViewerRole))
	insights.GET("", httpserver2.AuthorizeHandler(h.ListInsights, authApi.ViewerRole))
	insights.GET("/:insightId", httpserver2.AuthorizeHandler(h.GetInsight, authApi.ViewerRole))
	insights.GET("/:insightId/trend", httpserver2.AuthorizeHandler(h.GetInsightTrend, authApi.ViewerRole))

	findings := v1.Group("/findings")
	findings.POST("", httpserver2.AuthorizeHandler(h.GetFindings, authApi.ViewerRole))
	findings.POST("/resource", httpserver2.AuthorizeHandler(h.GetSingleResourceFinding, authApi.ViewerRole))
	findings.GET("/single/:id", httpserver2.AuthorizeHandler(h.GetSingleFindingByFindingID, authApi.ViewerRole))
	findings.GET("/events/:id", httpserver2.AuthorizeHandler(h.GetFindingEventsByFindingID, authApi.ViewerRole))
	findings.GET("/count", httpserver2.AuthorizeHandler(h.CountFindings, authApi.ViewerRole))
	findings.POST("/filters", httpserver2.AuthorizeHandler(h.GetFindingFilterValues, authApi.ViewerRole))
	findings.GET("/kpi", httpserver2.AuthorizeHandler(h.GetFindingKPIs, authApi.ViewerRole))
	findings.GET("/top/:field/:count", httpserver2.AuthorizeHandler(h.GetTopFieldByFindingCount, authApi.ViewerRole))
	findings.GET("/:benchmarkId/:field/count", httpserver2.AuthorizeHandler(h.GetFindingsFieldCountByControls, authApi.ViewerRole))
	findings.GET("/:benchmarkId/accounts", httpserver2.AuthorizeHandler(h.GetAccountsFindingsSummary, authApi.ViewerRole))
	findings.GET("/:benchmarkId/services", httpserver2.AuthorizeHandler(h.GetServicesFindingsSummary, authApi.ViewerRole))

	findingEvents := v1.Group("/finding_events")
	findingEvents.POST("", httpserver2.AuthorizeHandler(h.GetFindingEvents, authApi.ViewerRole))
	findingEvents.POST("/filters", httpserver2.AuthorizeHandler(h.GetFindingEventFilterValues, authApi.ViewerRole))
	findingEvents.GET("/single/:id", httpserver2.AuthorizeHandler(h.GetSingleFindingEvent, authApi.ViewerRole))

	resourceFindings := v1.Group("/resource_findings")
	resourceFindings.POST("", httpserver2.AuthorizeHandler(h.ListResourceFindings, authApi.ViewerRole))

	ai := v1.Group("/ai")
	ai.POST("/control/:controlID/remediation", httpserver2.AuthorizeHandler(h.GetControlRemediation, authApi.ViewerRole))
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
	connectionIds := httpserver2.QueryArrayParam(ctx, ConnectionIdParam)
	connectionIds, err := httpserver2.ResolveConnectionIDs(ctx, connectionIds)
	if err != nil {
		return nil, err
	}
	connectionGroup := httpserver2.QueryArrayParam(ctx, ConnectionGroupParam)
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
		connectionGroupObj, err := h.onboardClient.GetConnectionGroup(&httpclient.Context{UserRole: authApi.InternalRole}, connectionGroup[i])
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

	var err error
	req.Filters.ConnectionID, err = httpserver2.ResolveConnectionIDs(ctx, req.Filters.ConnectionID)
	if err != nil {
		return err
	}

	var response api.GetFindingsResponse

	if len(req.Filters.ConformanceStatus) == 0 {
		req.Filters.ConformanceStatus = []api.ConformanceStatus{api.ConformanceStatusFailed}
	}

	esConformanceStatuses := make([]kaytuTypes.ConformanceStatus, 0, len(req.Filters.ConformanceStatus))
	for _, status := range req.Filters.ConformanceStatus {
		esConformanceStatuses = append(esConformanceStatuses, status.GetEsConformanceStatuses()...)
	}

	if len(req.Sort) == 0 {
		req.Sort = []api.FindingsSort{
			{ConformanceStatus: utils.GetPointer(api.SortDirectionDescending)},
		}
	}

	if len(req.AfterSortKey) != 0 {
		expectedLen := len(req.Sort) + 1
		if len(req.AfterSortKey) != expectedLen {
			return echo.NewHTTPError(http.StatusBadRequest, "sort key length should be zero or match a returned sort key from previous response")
		}
	}

	var lastEventFrom, lastEventTo, evaluatedAtFrom, evaluatedAtTo *time.Time
	if req.Filters.LastEvent.From != nil && *req.Filters.LastEvent.From != 0 {
		lastEventFrom = utils.GetPointer(time.Unix(*req.Filters.LastEvent.From, 0))
	}
	if req.Filters.LastEvent.To != nil && *req.Filters.LastEvent.To != 0 {
		lastEventTo = utils.GetPointer(time.Unix(*req.Filters.LastEvent.To, 0))
	}
	if req.Filters.EvaluatedAt.From != nil && *req.Filters.EvaluatedAt.From != 0 {
		evaluatedAtFrom = utils.GetPointer(time.Unix(*req.Filters.EvaluatedAt.From, 0))
	}
	if req.Filters.EvaluatedAt.To != nil && *req.Filters.EvaluatedAt.To != 0 {
		evaluatedAtTo = utils.GetPointer(time.Unix(*req.Filters.EvaluatedAt.To, 0))
	}

	res, totalCount, err := es.FindingsQuery(h.logger, h.client,
		req.Filters.ResourceID, req.Filters.Connector, req.Filters.ConnectionID, req.Filters.NotConnectionID,
		req.Filters.ResourceTypeID,
		req.Filters.BenchmarkID, req.Filters.ControlID, req.Filters.Severity,
		lastEventFrom, lastEventTo,
		evaluatedAtFrom, evaluatedAtTo,
		req.Filters.StateActive, esConformanceStatuses, req.Sort, req.Limit, req.AfterSortKey)
	if err != nil {
		h.logger.Error("failed to get findings", zap.Error(err))
		return err
	}

	allSources, err := h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), nil)
	if err != nil {
		h.logger.Error("failed to get sources", zap.Error(err))
		return err
	}
	allSourcesMap := make(map[string]*onboardApi.Connection)
	for _, src := range allSources {
		src := src
		allSourcesMap[src.ID.String()] = &src
	}

	controls, err := h.db.ListControls()
	if err != nil {
		h.logger.Error("failed to get controls", zap.Error(err))
		return err
	}
	controlsMap := make(map[string]*db.Control)
	for _, control := range controls {
		control := control
		controlsMap[control.ID] = &control
	}

	benchmarks, err := h.db.ListBenchmarksBare()
	if err != nil {
		h.logger.Error("failed to get benchmarks", zap.Error(err))
		return err
	}
	benchmarksMap := make(map[string]*db.Benchmark)
	for _, benchmark := range benchmarks {
		benchmark := benchmark
		benchmarksMap[benchmark.ID] = &benchmark
	}

	resourceTypeMetadata, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx),
		nil, nil, nil, false, nil, 10000, 1)
	if err != nil {
		h.logger.Error("failed to get resource type metadata", zap.Error(err))
		return err
	}
	resourceTypeMetadataMap := make(map[string]*inventoryApi.ResourceType)
	for _, item := range resourceTypeMetadata.ResourceTypes {
		item := item
		resourceTypeMetadataMap[strings.ToLower(item.ResourceType)] = &item
	}

	for _, h := range res {
		finding := api.GetAPIFindingFromESFinding(h.Source)

		for _, parentBenchmark := range h.Source.ParentBenchmarks {
			if benchmark, ok := benchmarksMap[parentBenchmark]; ok {
				finding.ParentBenchmarkNames = append(finding.ParentBenchmarkNames, benchmark.Title)
			}
		}

		if src, ok := allSourcesMap[finding.ConnectionID]; ok {
			finding.ProviderConnectionID = demo.EncodeResponseData(ctx, src.ConnectionID)
			finding.ProviderConnectionName = demo.EncodeResponseData(ctx, src.ConnectionName)
		}

		if control, ok := controlsMap[finding.ControlID]; ok {
			finding.ControlTitle = control.Title
		}

		if rtMetadata, ok := resourceTypeMetadataMap[strings.ToLower(finding.ResourceType)]; ok {
			finding.ResourceTypeName = rtMetadata.ResourceLabel
		}

		finding.SortKey = h.Sort

		response.Findings = append(response.Findings, finding)
	}
	response.TotalCount = totalCount

	kaytuResourceIds := make([]string, 0, len(response.Findings))
	for _, finding := range response.Findings {
		kaytuResourceIds = append(kaytuResourceIds, finding.KaytuResourceID)
	}

	lookupResources, err := es.FetchLookupByResourceIDBatch(h.client, kaytuResourceIds)
	if err != nil {
		h.logger.Error("failed to fetch lookup resources", zap.Error(err))
		return err
	}

	lookupResourcesMap := make(map[string]*es2.LookupResource)
	for _, r := range lookupResources {
		r := r
		lookupResourcesMap[r.ResourceID] = &r
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetFindingEventsByFindingID godoc
//
//	@Summary		Get finding events by finding ID
//	@Description	Retrieving all compliance run finding events with respect to filters.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Finding ID"
//	@Success		200	{object}	api.GetFindingEventsByFindingIDResponse
//	@Router			/compliance/api/v1/findings/events/{id} [get]
func (h *HttpHandler) GetFindingEventsByFindingID(ctx echo.Context) error {
	findingID := ctx.Param("id")

	findingEvents, err := es.FetchFindingEventsByFindingIDs(h.logger, h.client, []string{findingID})
	if err != nil {
		h.logger.Error("failed to fetch finding by id", zap.Error(err))
		return err
	}

	response := api.GetFindingEventsByFindingIDResponse{
		FindingEvents: make([]api.FindingEvent, 0, len(findingEvents)),
	}
	for _, findingEvent := range findingEvents {
		response.FindingEvents = append(response.FindingEvents, api.GetAPIFindingEventFromESFindingEvent(findingEvent))
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetSingleResourceFinding godoc
//
//	@Summary		Get finding
//	@Description	Retrieving a single finding
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			request	body		api.GetSingleResourceFindingRequest	true	"Request Body"
//	@Success		200		{object}	api.GetSingleResourceFindingResponse
//	@Router			/compliance/api/v1/findings/resource [post]
func (h *HttpHandler) GetSingleResourceFinding(ctx echo.Context) error {
	var req api.GetSingleResourceFindingRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	kaytuResourceID := req.KaytuResourceId

	lookupResourceRes, err := es.FetchLookupByResourceIDBatch(h.client, []string{kaytuResourceID})
	if err != nil {
		h.logger.Error("failed to fetch lookup resources", zap.Error(err))
		return err
	}
	if len(lookupResourceRes) == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "resource not found")
	}

	lookupResource := lookupResourceRes[0]

	resource, err := es.FetchResourceByResourceIdAndType(h.client, lookupResource.ResourceID, lookupResource.ResourceType)
	if err != nil {
		h.logger.Error("failed to fetch resource", zap.Error(err))
		return err
	}
	if resource == nil {
		return echo.NewHTTPError(http.StatusNotFound, "resource not found")
	}

	response := api.GetSingleResourceFindingResponse{
		Resource: *resource,
	}

	controlFindings, err := es.FetchFindingsPerControlForResourceId(h.logger, h.client, lookupResource.ResourceID)
	if err != nil {
		h.logger.Error("failed to fetch control findings", zap.Error(err))
		return err
	}

	allSources, err := h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), nil)
	if err != nil {
		h.logger.Error("failed to get sources", zap.Error(err))
		return err
	}
	allSourcesMap := make(map[string]*onboardApi.Connection)
	for _, src := range allSources {
		src := src
		allSourcesMap[src.ID.String()] = &src
	}

	controls, err := h.db.ListControls()
	if err != nil {
		h.logger.Error("failed to get controls", zap.Error(err))
		return err
	}
	controlsMap := make(map[string]*db.Control)
	for _, control := range controls {
		control := control
		controlsMap[control.ID] = &control
	}

	benchmarks, err := h.db.ListBenchmarksBare()
	if err != nil {
		h.logger.Error("failed to get benchmarks", zap.Error(err))
		return err
	}
	benchmarksMap := make(map[string]*db.Benchmark)
	for _, benchmark := range benchmarks {
		benchmark := benchmark
		benchmarksMap[benchmark.ID] = &benchmark
	}

	resourceTypeMetadata, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx),
		nil, nil, nil, false, nil, 10000, 1)
	if err != nil {
		h.logger.Error("failed to get resource type metadata", zap.Error(err))
		return err
	}
	resourceTypeMetadataMap := make(map[string]*inventoryApi.ResourceType)
	for _, item := range resourceTypeMetadata.ResourceTypes {
		item := item
		resourceTypeMetadataMap[strings.ToLower(item.ResourceType)] = &item
	}

	findingsIDs := make([]string, 0, len(controlFindings))
	for _, controlFinding := range controlFindings {
		findingsIDs = append(findingsIDs, controlFinding.EsID)
		controlFinding := controlFinding
		controlFinding.ResourceName = lookupResource.Name
		controlFinding.ResourceLocation = lookupResource.Location
		finding := api.GetAPIFindingFromESFinding(controlFinding)

		for _, parentBenchmark := range finding.ParentBenchmarks {
			if benchmark, ok := benchmarksMap[parentBenchmark]; ok {
				finding.ParentBenchmarkNames = append(finding.ParentBenchmarkNames, benchmark.Title)
			}
		}

		if src, ok := allSourcesMap[finding.ConnectionID]; ok {
			finding.ProviderConnectionID = demo.EncodeResponseData(ctx, src.ConnectionID)
			finding.ProviderConnectionName = demo.EncodeResponseData(ctx, src.ConnectionName)
		}

		if control, ok := controlsMap[finding.ControlID]; ok {
			finding.ControlTitle = control.Title
		}

		if rtMetadata, ok := resourceTypeMetadataMap[strings.ToLower(finding.ResourceType)]; ok {
			finding.ResourceTypeName = rtMetadata.ResourceLabel
		}

		response.ControlFindings = append(response.ControlFindings, finding)
	}

	findingEvents, err := es.FetchFindingEventsByFindingIDs(h.logger, h.client, findingsIDs)
	if err != nil {
		h.logger.Error("failed to fetch finding events", zap.Error(err))
		return err
	}

	response.FindingEvents = make([]api.FindingEvent, 0, len(findingEvents))
	for _, findingEvent := range findingEvents {
		response.FindingEvents = append(response.FindingEvents, api.GetAPIFindingEventFromESFindingEvent(findingEvent))
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetSingleFindingByFindingID
//
//	@Summary		Get single finding by finding ID
//	@Description	Retrieving a single finding by finding ID
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Finding ID"
//	@Success		200	{object}	api.Finding
//	@Router			/compliance/api/v1/findings/single/{id} [get]
func (h *HttpHandler) GetSingleFindingByFindingID(ctx echo.Context) error {
	findingID := ctx.Param("id")

	finding, err := es.FetchFindingByID(h.logger, h.client, findingID)
	if err != nil {
		h.logger.Error("failed to fetch finding by id", zap.Error(err))
		return err
	}
	if finding == nil {
		return echo.NewHTTPError(http.StatusNotFound, "finding not found")
	}

	apiFinding := api.GetAPIFindingFromESFinding(*finding)

	connection, err := h.onboardClient.GetSource(httpclient.FromEchoContext(ctx), finding.ConnectionID)
	if err != nil {
		h.logger.Error("failed to get connection", zap.Error(err), zap.String("connection_id", finding.ConnectionID))
		return err
	}
	apiFinding.ProviderConnectionID = connection.ConnectionID
	apiFinding.ProviderConnectionName = connection.ConnectionName

	if len(finding.ResourceType) > 0 {
		resourceTypeMetadata, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx),
			nil, nil,
			[]string{finding.ResourceType}, false, nil, 10000, 1)
		if err != nil {
			h.logger.Error("failed to get resource type metadata", zap.Error(err))
			return err
		}
		if len(resourceTypeMetadata.ResourceTypes) > 0 {
			apiFinding.ResourceTypeName = resourceTypeMetadata.ResourceTypes[0].ResourceLabel
		}
	}

	control, err := h.db.GetControl(finding.ControlID)
	if err != nil {
		h.logger.Error("failed to get control", zap.Error(err), zap.String("control_id", finding.ControlID))
		return err
	}
	apiFinding.ControlTitle = control.Title

	parentBenchmarks, err := h.db.GetBenchmarksBare(finding.ParentBenchmarks)
	if err != nil {
		h.logger.Error("failed to get parent benchmarks", zap.Error(err), zap.Strings("parent_benchmarks", finding.ParentBenchmarks))
		return err
	}
	parentBenchmarksMap := make(map[string]db.Benchmark)
	for _, benchmark := range parentBenchmarks {
		parentBenchmarksMap[benchmark.ID] = benchmark
	}
	for _, parentBenchmark := range finding.ParentBenchmarks {
		if benchmark, ok := parentBenchmarksMap[parentBenchmark]; ok {
			apiFinding.ParentBenchmarkNames = append(apiFinding.ParentBenchmarkNames, benchmark.Title)
		}
	}

	return ctx.JSON(http.StatusOK, apiFinding)
}

// CountFindings godoc
//
//	@Summary		Get findings count
//	@Description	Retrieving all compliance run findings count with respect to filters.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			conformanceStatus	query		[]api.ConformanceStatus	false	"ConformanceStatus to filter by defaults to all conformanceStatus except passed"
//	@Success		200					{object}	api.CountFindingsResponse
//	@Router			/compliance/api/v1/findings/count [get]
func (h *HttpHandler) CountFindings(ctx echo.Context) error {
	conformanceStatuses := api.ParseConformanceStatuses(httpserver2.QueryArrayParam(ctx, "conformanceStatus"))
	if len(conformanceStatuses) == 0 {
		conformanceStatuses = []api.ConformanceStatus{api.ConformanceStatusFailed}
	}

	esConformanceStatuses := make([]kaytuTypes.ConformanceStatus, 0, len(conformanceStatuses))
	for _, status := range conformanceStatuses {
		esConformanceStatuses = append(esConformanceStatuses, status.GetEsConformanceStatuses()...)
	}

	totalCount, err := es.FindingsCount(h.client, esConformanceStatuses)
	if err != nil {
		return err
	}

	response := api.CountFindingsResponse{
		Count: totalCount,
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetFindingFilterValues godoc
//
//	@Summary		Get possible values for finding filters
//	@Description	Retrieving possible values for finding filters.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			request	body		api.FindingFilters	true	"Request Body"
//	@Success		200		{object}	api.FindingFiltersWithMetadata
//	@Router			/compliance/api/v1/findings/filters [post]
func (h *HttpHandler) GetFindingFilterValues(ctx echo.Context) error {
	var req api.FindingFilters
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var err error
	req.ConnectionID, err = httpserver2.ResolveConnectionIDs(ctx, req.ConnectionID)
	if err != nil {
		return err
	}

	if len(req.ConformanceStatus) == 0 {
		req.ConformanceStatus = []api.ConformanceStatus{api.ConformanceStatusFailed}
	}

	esConformanceStatuses := make([]kaytuTypes.ConformanceStatus, 0, len(req.ConformanceStatus))
	for _, status := range req.ConformanceStatus {
		esConformanceStatuses = append(esConformanceStatuses, status.GetEsConformanceStatuses()...)
	}

	resourceTypeMetadata, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx),
		nil, nil, nil, false, nil, 10000, 1)
	if err != nil {
		h.logger.Error("failed to get resource type metadata", zap.Error(err))
		return err
	}
	resourceTypeMetadataMap := make(map[string]*inventoryApi.ResourceType)
	for _, item := range resourceTypeMetadata.ResourceTypes {
		item := item
		resourceTypeMetadataMap[strings.ToLower(item.ResourceType)] = &item
	}

	resourceCollectionMetadata, err := h.inventoryClient.ListResourceCollections(httpclient.FromEchoContext(ctx))
	if err != nil {
		h.logger.Error("failed to get resource collection metadata", zap.Error(err))
		return err
	}
	resourceCollectionMetadataMap := make(map[string]*inventoryApi.ResourceCollection)
	for _, item := range resourceCollectionMetadata {
		item := item
		resourceCollectionMetadataMap[item.ID] = &item
	}

	connectionMetadata, err := h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), nil)
	if err != nil {
		h.logger.Error("failed to get connections", zap.Error(err))
		return err
	}
	connectionMetadataMap := make(map[string]*onboardApi.Connection)
	for _, item := range connectionMetadata {
		item := item
		connectionMetadataMap[item.ID.String()] = &item
	}

	benchmarkMetadata, err := h.db.ListBenchmarksBare()
	if err != nil {
		h.logger.Error("failed to get benchmarks", zap.Error(err))
		return err
	}
	benchmarkMetadataMap := make(map[string]*db.Benchmark)
	for _, item := range benchmarkMetadata {
		item := item
		benchmarkMetadataMap[item.ID] = &item
	}

	controlMetadata, err := h.db.ListControlsBare()
	if err != nil {
		h.logger.Error("failed to get controls", zap.Error(err))
		return err
	}
	controlMetadataMap := make(map[string]*db.Control)
	for _, item := range controlMetadata {
		item := item
		controlMetadataMap[item.ID] = &item
	}

	var lastEventFrom, lastEventTo, evaluatedAtFrom, evaluatedAtTo *time.Time
	if req.LastEvent.From != nil && *req.LastEvent.From != 0 {
		lastEventFrom = utils.GetPointer(time.Unix(*req.LastEvent.From, 0))
	}
	if req.LastEvent.To != nil && *req.LastEvent.To != 0 {
		lastEventTo = utils.GetPointer(time.Unix(*req.LastEvent.To, 0))
	}
	if req.EvaluatedAt.From != nil && *req.EvaluatedAt.From != 0 {
		evaluatedAtFrom = utils.GetPointer(time.Unix(*req.EvaluatedAt.From, 0))
	}
	if req.EvaluatedAt.To != nil && *req.EvaluatedAt.To != 0 {
		evaluatedAtTo = utils.GetPointer(time.Unix(*req.EvaluatedAt.To, 0))
	}

	possibleFilters, err := es.FindingsFiltersQuery(h.logger, h.client,
		req.ResourceID, req.Connector, req.ConnectionID, req.NotConnectionID,
		req.ResourceTypeID,
		req.BenchmarkID, req.ControlID,
		req.Severity,
		lastEventFrom, lastEventTo,
		evaluatedAtFrom, evaluatedAtTo,
		req.StateActive, esConformanceStatuses)
	if err != nil {
		h.logger.Error("failed to get possible filters", zap.Error(err))
		return err
	}
	response := api.FindingFiltersWithMetadata{}
	for _, item := range possibleFilters.Aggregations.BenchmarkIDFilter.Buckets {
		if benchmark, ok := benchmarkMetadataMap[item.Key]; ok {
			response.BenchmarkID = append(response.BenchmarkID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: benchmark.Title,
				Count:       utils.GetPointer(item.DocCount),
			})
		} else {
			response.BenchmarkID = append(response.BenchmarkID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: item.Key,
				Count:       utils.GetPointer(item.DocCount),
			})
		}
	}
	for _, item := range possibleFilters.Aggregations.ControlIDFilter.Buckets {
		if control, ok := controlMetadataMap[item.Key]; ok {
			response.ControlID = append(response.ControlID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: control.Title,
				Count:       utils.GetPointer(item.DocCount),
			})
		} else {
			response.ControlID = append(response.ControlID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: item.Key,
				Count:       utils.GetPointer(item.DocCount),
			})
		}
	}
	if len(possibleFilters.Aggregations.ConnectorFilter.Buckets) > 0 {
		for _, bucket := range possibleFilters.Aggregations.ConnectorFilter.Buckets {
			connector, _ := source.ParseType(bucket.Key)
			response.Connector = append(response.Connector, api.FilterWithMetadata{
				Key:         connector.String(),
				DisplayName: connector.String(),
				Count:       utils.GetPointer(bucket.DocCount),
			})
		}
	}
	for _, item := range possibleFilters.Aggregations.ResourceTypeFilter.Buckets {
		if rtMetadata, ok := resourceTypeMetadataMap[strings.ToLower(item.Key)]; ok {
			response.ResourceTypeID = append(response.ResourceTypeID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: rtMetadata.ResourceLabel,
				Count:       utils.GetPointer(item.DocCount),
			})
		} else if item.Key == "" {
			response.ResourceTypeID = append(response.ResourceTypeID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: "Unknown",
			})
		} else {
			response.ResourceTypeID = append(response.ResourceTypeID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: item.Key,
				Count:       utils.GetPointer(item.DocCount),
			})
		}
	}

	for _, item := range possibleFilters.Aggregations.ConnectionIDFilter.Buckets {
		if connection, ok := connectionMetadataMap[item.Key]; ok {
			response.ConnectionID = append(response.ConnectionID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: connection.ConnectionName,
				Count:       utils.GetPointer(item.DocCount),
			})
		} else {
			response.ConnectionID = append(response.ConnectionID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: item.Key,
				Count:       utils.GetPointer(item.DocCount),
			})
		}
	}

	for _, item := range possibleFilters.Aggregations.ResourceCollectionFilter.Buckets {
		if resourceCollection, ok := resourceCollectionMetadataMap[item.Key]; ok {
			response.ResourceCollection = append(response.ResourceCollection, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: resourceCollection.Name,
				Count:       utils.GetPointer(item.DocCount),
			})
		} else {
			response.ResourceCollection = append(response.ResourceCollection, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: item.Key,
				Count:       utils.GetPointer(item.DocCount),
			})
		}
	}

	for _, item := range possibleFilters.Aggregations.SeverityFilter.Buckets {
		response.Severity = append(response.Severity, api.FilterWithMetadata{
			Key:         item.Key,
			DisplayName: item.Key,
			Count:       utils.GetPointer(item.DocCount),
		})
	}

	for _, item := range possibleFilters.Aggregations.StateActiveFilter.Buckets {
		response.StateActive = append(response.StateActive, api.FilterWithMetadata{
			Key:         item.KeyAsString,
			DisplayName: item.KeyAsString,
			Count:       utils.GetPointer(item.DocCount),
		})
	}

	apiConformanceStatuses := make(map[api.ConformanceStatus]int)
	for _, item := range possibleFilters.Aggregations.ConformanceStatusFilter.Buckets {
		if kaytuTypes.ParseConformanceStatus(item.Key).IsPassed() {
			apiConformanceStatuses[api.ConformanceStatusPassed] += item.DocCount
		} else {
			apiConformanceStatuses[api.ConformanceStatusFailed] += item.DocCount
		}
	}
	for status, count := range apiConformanceStatuses {
		count := count
		response.ConformanceStatus = append(response.ConformanceStatus, api.FilterWithMetadata{
			Key:         string(status),
			DisplayName: string(status),
			Count:       &count,
		})
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetFindingKPIs godoc
//
//	@Summary		Get finding KPIs
//	@Description	Retrieving KPIs for findings.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	api.FindingKPIResponse
//	@Router			/compliance/api/v1/findings/kpi [get]
func (h *HttpHandler) GetFindingKPIs(ctx echo.Context) error {
	kpiRes, err := es.FindingKPIQuery(h.logger, h.client)
	if err != nil {
		h.logger.Error("failed to get finding kpis", zap.Error(err))
		return err
	}
	response := api.FindingKPIResponse{
		FailedFindingsCount:   kpiRes.Hits.Total.Value,
		FailedResourceCount:   kpiRes.Aggregations.ResourceCount.Value,
		FailedControlCount:    kpiRes.Aggregations.ControlCount.Value,
		FailedConnectionCount: kpiRes.Aggregations.ConnectionCount.Value,
	}
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
//	@Param			field				path		string							true	"Field"	Enums(resourceType,connectionID,resourceID,service,controlID)
//	@Param			count				path		int								true	"Count"
//	@Param			connectionId		query		[]string						false	"Connection IDs to filter by (inclusive)"
//	@Param			notConnectionId		query		[]string						false	"Connection IDs to filter by (exclusive)"
//	@Param			connectionGroup		query		[]string						false	"Connection groups to filter by "
//	@Param			connector			query		[]source.Type					false	"Connector type to filter by"
//	@Param			benchmarkId			query		[]string						false	"BenchmarkID"
//	@Param			controlId			query		[]string						false	"ControlID"
//	@Param			severities			query		[]kaytuTypes.FindingSeverity	false	"Severities to filter by defaults to all severities except passed"
//	@Param			conformanceStatus	query		[]api.ConformanceStatus			false	"ConformanceStatus to filter by defaults to all conformanceStatus except passed"
//	@Param			stateActive			query		[]bool							false	"StateActive to filter by defaults to true"
//	@Success		200					{object}	api.GetTopFieldResponse
//	@Router			/compliance/api/v1/findings/top/{field}/{count} [get]
func (h *HttpHandler) GetTopFieldByFindingCount(ctx echo.Context) error {
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
	notConnectionIDs := httpserver2.QueryArrayParam(ctx, "notConnectionId")
	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	benchmarkIDs := httpserver2.QueryArrayParam(ctx, "benchmarkId")
	controlIDs := httpserver2.QueryArrayParam(ctx, "controlId")
	severities := kaytuTypes.ParseFindingSeverities(httpserver2.QueryArrayParam(ctx, "severities"))
	conformanceStatuses := api.ParseConformanceStatuses(httpserver2.QueryArrayParam(ctx, "conformanceStatus"))
	if len(conformanceStatuses) == 0 {
		conformanceStatuses = []api.ConformanceStatus{
			api.ConformanceStatusFailed,
		}
	}

	esConformanceStatuses := make([]kaytuTypes.ConformanceStatus, 0, len(conformanceStatuses))
	for _, status := range conformanceStatuses {
		esConformanceStatuses = append(esConformanceStatuses, status.GetEsConformanceStatuses()...)
	}

	stateActives := []bool{true}
	if stateActiveStr := httpserver2.QueryArrayParam(ctx, "stateActive"); len(stateActiveStr) > 0 {
		stateActives = make([]bool, 0, len(stateActiveStr))
		for _, item := range stateActiveStr {
			stateActive, err := strconv.ParseBool(item)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid stateActive")
			}
			stateActives = append(stateActives, stateActive)
		}
	}

	var response api.GetTopFieldResponse
	topFieldResponse, err := es.FindingsTopFieldQuery(h.logger, h.client, esField, connectors,
		nil, connectionIDs, notConnectionIDs,
		benchmarkIDs, controlIDs, severities, esConformanceStatuses, stateActives, min(10000, esCount))
	if err != nil {
		h.logger.Error("failed to get top field", zap.Error(err))
		return err
	}
	topFieldTotalResponse, err := es.FindingsTopFieldQuery(h.logger, h.client, esField, connectors,
		nil, connectionIDs, notConnectionIDs,
		benchmarkIDs, controlIDs, severities, nil, stateActives, 10000)
	if err != nil {
		h.logger.Error("failed to get top field total", zap.Error(err))
		return err
	}

	switch strings.ToLower(field) {
	case "resourcetype":
		resourceTypeList := make([]string, 0, len(topFieldResponse.Aggregations.FieldFilter.Buckets))
		for _, item := range topFieldResponse.Aggregations.FieldFilter.Buckets {
			if item.Key == "" {
				continue
			}
			resourceTypeList = append(resourceTypeList, item.Key)
		}
		resourceTypeMetadata, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx),
			nil, nil, resourceTypeList, false, nil, 10000, 1)
		if err != nil {
			return err
		}
		resourceTypeMetadataMap := make(map[string]*inventoryApi.ResourceType)
		for _, item := range resourceTypeMetadata.ResourceTypes {
			item := item
			resourceTypeMetadataMap[strings.ToLower(item.ResourceType)] = &item
		}
		resourceTypeCountMap := make(map[string]int)
		for _, item := range topFieldResponse.Aggregations.FieldFilter.Buckets {
			if item.Key == "" {
				item.Key = "Unknown"
			}
			resourceTypeCountMap[item.Key] += item.DocCount
		}
		resourceTypeTotalCountMap := make(map[string]int)
		for _, item := range topFieldTotalResponse.Aggregations.FieldFilter.Buckets {
			if item.Key == "" {
				item.Key = "Unknown"
			}
			resourceTypeTotalCountMap[item.Key] += item.DocCount
		}
		resourceTypeCountList := make([]api.TopFieldRecord, 0, len(resourceTypeCountMap))
		for k, v := range resourceTypeCountMap {
			rt, ok := resourceTypeMetadataMap[strings.ToLower(k)]
			if !ok {
				rt = &inventoryApi.ResourceType{
					ResourceType:  k,
					ResourceLabel: k,
				}
			}
			resourceTypeCountList = append(resourceTypeCountList, api.TopFieldRecord{
				ResourceType: rt,
				Count:        v,
				TotalCount:   resourceTypeTotalCountMap[k],
			})
		}
		sort.Slice(resourceTypeCountList, func(i, j int) bool {
			return resourceTypeCountList[i].Count > resourceTypeCountList[j].Count
		})
		if len(resourceTypeCountList) > count {
			response.Records = resourceTypeCountList[:count]
		} else {
			response.Records = resourceTypeCountList
		}
		response.TotalCount = len(resourceTypeCountList)
	case "service":
		resourceTypeList := make([]string, 0, len(topFieldResponse.Aggregations.FieldFilter.Buckets))
		for _, item := range topFieldResponse.Aggregations.FieldFilter.Buckets {
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
		for _, item := range topFieldResponse.Aggregations.FieldFilter.Buckets {
			if rtMetadata, ok := resourceTypeMetadataMap[strings.ToLower(item.Key)]; ok {
				serviceCountMap[rtMetadata.ServiceName] += item.DocCount
			}
		}
		serviceTotalCountMap := make(map[string]int)
		for _, item := range topFieldTotalResponse.Aggregations.FieldFilter.Buckets {
			if rtMetadata, ok := resourceTypeMetadataMap[strings.ToLower(item.Key)]; ok {
				serviceTotalCountMap[rtMetadata.ServiceName] += item.DocCount
			}
		}
		serviceCountList := make([]api.TopFieldRecord, 0, len(serviceCountMap))
		for k, v := range serviceCountMap {
			k := k
			serviceCountList = append(serviceCountList, api.TopFieldRecord{
				Service:    &k,
				Count:      v,
				TotalCount: serviceTotalCountMap[k],
			})
		}
		sort.Slice(serviceCountList, func(i, j int) bool {
			return serviceCountList[i].Count > serviceCountList[j].Count
		})
		if len(serviceCountList) > count {
			response.Records = serviceCountList[:count]
		} else {
			response.Records = serviceCountList
		}
		response.TotalCount = len(serviceCountList)
	case "connectionid":
		resConnectionIDs := make([]string, 0, len(topFieldTotalResponse.Aggregations.FieldFilter.Buckets))
		for _, item := range topFieldTotalResponse.Aggregations.FieldFilter.Buckets {
			resConnectionIDs = append(resConnectionIDs, item.Key)
		}
		connections, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), resConnectionIDs)
		if err != nil {
			h.logger.Error("failed to get connections", zap.Error(err))
			return err
		}

		recordMap := make(map[string]api.TopFieldRecord)

		for _, item := range topFieldTotalResponse.Aggregations.FieldFilter.Buckets {
			record, ok := recordMap[item.Key]
			if !ok {
				id, err := uuid.Parse(item.Key)
				if err != nil {
					h.logger.Error("failed to parse connection id", zap.Error(err))
					return err
				}
				record = api.TopFieldRecord{
					Connection: &onboardApi.Connection{ID: id},
				}
			}
			record.TotalCount += item.DocCount
			recordMap[item.Key] = record
		}

		for _, item := range topFieldResponse.Aggregations.FieldFilter.Buckets {
			record, ok := recordMap[item.Key]
			if !ok {
				id, err := uuid.Parse(item.Key)
				if err != nil {
					h.logger.Error("failed to parse connection id", zap.Error(err))
					return err
				}
				record = api.TopFieldRecord{
					Connection: &onboardApi.Connection{ID: id},
				}
			}
			record.Count = item.DocCount
			recordMap[item.Key] = record
		}

		for _, connection := range connections {
			connection := connection
			record, ok := recordMap[connection.ID.String()]
			if !ok {
				continue
			}
			record.Connection = &connection
			recordMap[connection.ID.String()] = record
		}

		controlsResult, err := es.FindingsConformanceStatusCountByControlPerConnection(
			h.logger, h.client, connectors, nil, resConnectionIDs, benchmarkIDs, controlIDs, severities, nil)
		if err != nil {
			h.logger.Error("failed to get controls", zap.Error(err))
			return err
		}
		for _, item := range controlsResult.Aggregations.ConnectionGroup.Buckets {
			record, ok := recordMap[item.Key]
			if !ok {
				continue
			}
			if record.ControlCount == nil {
				record.ControlCount = utils.GetPointer(0)
			}
			if record.ControlTotalCount == nil {
				record.ControlTotalCount = utils.GetPointer(0)
			}
			for _, control := range item.ControlCount.Buckets {
				isFailed := false
				for _, conformanceStatus := range control.ConformanceStatuses.Buckets {
					status := kaytuTypes.ParseConformanceStatus(conformanceStatus.Key)
					if !status.IsPassed() && conformanceStatus.DocCount > 0 {
						isFailed = true
						break
					}
				}
				if isFailed {
					record.ControlCount = utils.PAdd(record.ControlCount, utils.GetPointer(1))
				}
				record.ControlTotalCount = utils.PAdd(record.ControlTotalCount, utils.GetPointer(1))
			}
			recordMap[item.Key] = record
		}

		resourcesResult, err := es.GetPerFieldResourceConformanceResult(h.logger, h.client, "connectionID",
			resConnectionIDs, notConnectionIDs, nil, controlIDs, benchmarkIDs, severities, nil)
		if err != nil {
			h.logger.Error("failed to get resourcesResult", zap.Error(err))
			return err
		}

		for connectionId, results := range resourcesResult {
			results := results
			record, ok := recordMap[connectionId]
			if !ok {
				continue
			}
			record.ResourceTotalCount = utils.GetPointer(results.TotalCount)
			for _, conformanceStatus := range conformanceStatuses {
				switch conformanceStatus {
				case api.ConformanceStatusFailed:
					record.ResourceCount = utils.PAdd(record.ResourceCount, &results.AlarmCount)
					record.ResourceCount = utils.PAdd(record.ResourceCount, &results.ErrorCount)
					record.ResourceCount = utils.PAdd(record.ResourceCount, &results.InfoCount)
					record.ResourceCount = utils.PAdd(record.ResourceCount, &results.SkipCount)
				case api.ConformanceStatusPassed:
					record.ResourceCount = utils.PAdd(record.ResourceCount, &results.OkCount)
				}
			}
			recordMap[connectionId] = record
		}

		for _, record := range recordMap {
			response.Records = append(response.Records, record)
		}

		response.TotalCount = topFieldTotalResponse.Aggregations.BucketCount.Value
	case "controlid":
		resControlIDs := make([]string, 0, len(topFieldTotalResponse.Aggregations.FieldFilter.Buckets))
		for _, item := range topFieldTotalResponse.Aggregations.FieldFilter.Buckets {
			resControlIDs = append(resControlIDs, item.Key)
		}
		controls, err := h.db.GetControls(resControlIDs)
		if err != nil {
			h.logger.Error("failed to get controls", zap.Error(err))
			return err
		}

		recordMap := make(map[string]api.TopFieldRecord)

		for _, item := range topFieldTotalResponse.Aggregations.FieldFilter.Buckets {
			record, ok := recordMap[item.Key]
			if !ok {
				record = api.TopFieldRecord{
					Control: &api.Control{ID: item.Key},
				}
			}
			record.TotalCount += item.DocCount
			recordMap[item.Key] = record
		}

		for _, item := range topFieldResponse.Aggregations.FieldFilter.Buckets {
			record, ok := recordMap[item.Key]
			if !ok {
				record = api.TopFieldRecord{
					Control: &api.Control{ID: item.Key},
				}
			}
			record.Count = item.DocCount
			recordMap[item.Key] = record
		}

		for _, control := range controls {
			control := control
			record, ok := recordMap[control.ID]
			if !ok {
				continue
			}
			record.Control = utils.GetPointer(control.ToApi())
			recordMap[control.ID] = record
		}

		resourcesResult, err := es.GetPerFieldResourceConformanceResult(h.logger, h.client, "controlID",
			connectionIDs, notConnectionIDs, nil, resControlIDs, benchmarkIDs, severities, nil)
		if err != nil {
			h.logger.Error("failed to get resourcesResult", zap.Error(err))
			return err
		}

		for controlId, results := range resourcesResult {
			results := results
			record, ok := recordMap[controlId]
			if !ok {
				continue
			}
			record.ResourceTotalCount = utils.GetPointer(results.TotalCount)
			for _, conformanceStatus := range conformanceStatuses {
				switch conformanceStatus {
				case api.ConformanceStatusFailed:
					record.ResourceCount = utils.PAdd(record.ResourceCount, &results.AlarmCount)
					record.ResourceCount = utils.PAdd(record.ResourceCount, &results.ErrorCount)
					record.ResourceCount = utils.PAdd(record.ResourceCount, &results.InfoCount)
					record.ResourceCount = utils.PAdd(record.ResourceCount, &results.SkipCount)
				case api.ConformanceStatusPassed:
					record.ResourceCount = utils.PAdd(record.ResourceCount, &results.OkCount)
				}
			}
			recordMap[controlId] = record
		}

		for _, record := range recordMap {
			response.Records = append(response.Records, record)
		}

		response.TotalCount = topFieldTotalResponse.Aggregations.BucketCount.Value
	default:
		totalCountMap := make(map[string]int)
		for _, item := range topFieldTotalResponse.Aggregations.FieldFilter.Buckets {
			totalCountMap[item.Key] += item.DocCount
		}

		for _, item := range topFieldResponse.Aggregations.FieldFilter.Buckets {
			item := item
			response.Records = append(response.Records, api.TopFieldRecord{
				Field:      &item.Key,
				Count:      item.DocCount,
				TotalCount: totalCountMap[item.Key],
			})
		}
		response.TotalCount = topFieldResponse.Aggregations.BucketCount.Value
	}

	sort.Slice(response.Records, func(i, j int) bool {
		if response.Records[i].Count != response.Records[j].Count {
			return response.Records[i].Count > response.Records[j].Count
		}
		return response.Records[i].TotalCount > response.Records[j].TotalCount
	})
	if len(response.Records) > 0 {
		response.Records = response.Records[:min(len(response.Records), count)]
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetFindingsFieldCountByControls godoc
//
//	@Summary		Get findings field count by controls
//	@Description	Retrieving the number of findings field count by controls.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmarkId			path		string							true	"BenchmarkID"
//	@Param			field				path		string							true	"Field"	Enums(resourceType,connectionID,resourceID,service)
//	@Param			connectionId		query		[]string						false	"Connection IDs to filter by"
//	@Param			connectionGroup		query		[]string						false	"Connection groups to filter by "
//	@Param			resourceCollection	query		[]string						false	"Resource collection IDs to filter by"
//	@Param			connector			query		[]source.Type					false	"Connector type to filter by"
//	@Param			severities			query		[]kaytuTypes.FindingSeverity	false	"Severities to filter by defaults to all severities except passed"
//	@Param			conformanceStatus	query		[]api.ConformanceStatus			false	"ConformanceStatus to filter by defaults to failed"
//	@Success		200					{object}	api.GetTopFieldResponse
//	@Router			/compliance/api/v1/findings/{benchmarkId}/{field}/count [get]
func (h *HttpHandler) GetFindingsFieldCountByControls(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")
	field := ctx.Param("field")
	var esField string
	if field == "resource" {
		esField = "resourceID"
	} else {
		esField = field
	}

	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	severities := kaytuTypes.ParseFindingSeverities(httpserver2.QueryArrayParam(ctx, "severities"))
	conformanceStatuses := api.ParseConformanceStatuses(httpserver2.QueryArrayParam(ctx, "conformanceStatus"))
	if len(conformanceStatuses) == 0 {
		conformanceStatuses = []api.ConformanceStatus{
			api.ConformanceStatusFailed,
		}
	}
	esConformanceStatuses := make([]kaytuTypes.ConformanceStatus, 0, len(conformanceStatuses))
	for _, status := range conformanceStatuses {
		esConformanceStatuses = append(esConformanceStatuses, status.GetEsConformanceStatuses()...)
	}
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

	var response api.GetFieldCountResponse
	res, err := es.FindingsFieldCountByControl(h.logger, h.client, esField, connectors, nil, connectionIDs, benchmarkIDs, nil, severities,
		esConformanceStatuses)
	if err != nil {
		return err
	}
	for _, b := range res.Aggregations.ControlCount.Buckets {
		var fieldCounts []api.TopFieldRecord
		for _, bucketField := range b.ConformanceStatuses.Buckets {
			bucketField := bucketField
			fieldCounts = append(fieldCounts, api.TopFieldRecord{Field: &bucketField.Key, Count: bucketField.FieldCount.Value})
		}
		response.Controls = append(response.Controls, struct {
			ControlName string               `json:"controlName"`
			FieldCounts []api.TopFieldRecord `json:"fieldCounts"`
		}{ControlName: b.Key, FieldCounts: fieldCounts})
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetAccountsFindingsSummary godoc
//
//	@Summary		Get accounts findings summaries
//	@Description	Retrieving list of accounts with their security score and severities findings count
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmarkId		path		string		true	"BenchmarkID"
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param			connectionGroup	query		[]string	false	"Connection groups to filter by "
//	@Success		200				{object}	api.GetAccountsFindingsSummaryResponse
//	@Router			/compliance/api/v1/findings/{benchmarkId}/accounts [get]
func (h *HttpHandler) GetAccountsFindingsSummary(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	var response api.GetAccountsFindingsSummaryResponse
	res, evaluatedAt, err := es.BenchmarkConnectionSummary(h.logger, h.client, benchmarkID)
	if err != nil {
		return err
	}

	if len(connectionIDs) == 0 {
		assignmentsByBenchmarkId, err := h.db.GetBenchmarkAssignmentsByBenchmarkId(benchmarkID)
		if err != nil {
			return err
		}

		for _, assignment := range assignmentsByBenchmarkId {
			if assignment.ConnectionId != nil {
				connectionIDs = append(connectionIDs, *assignment.ConnectionId)
			}
		}
	}

	srcs, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), connectionIDs)
	if err != nil {
		return err
	}

	for _, src := range srcs {
		summary, ok := res[src.ID.String()]
		if !ok {
			summary.Result.SeverityResult = map[kaytuTypes.FindingSeverity]int{}
			summary.Result.QueryResult = map[kaytuTypes.ConformanceStatus]int{}
		}

		account := api.AccountsFindingsSummary{
			AccountName:   src.ConnectionName,
			AccountId:     src.ConnectionID,
			SecurityScore: summary.Result.SecurityScore,
			SeveritiesCount: struct {
				Critical int `json:"critical"`
				High     int `json:"high"`
				Medium   int `json:"medium"`
				Low      int `json:"low"`
				None     int `json:"none"`
			}{
				Critical: summary.Result.SeverityResult[kaytuTypes.FindingSeverityCritical],
				High:     summary.Result.SeverityResult[kaytuTypes.FindingSeverityHigh],
				Medium:   summary.Result.SeverityResult[kaytuTypes.FindingSeverityMedium],
				Low:      summary.Result.SeverityResult[kaytuTypes.FindingSeverityLow],
				None:     summary.Result.SeverityResult[kaytuTypes.FindingSeverityNone],
			},
			ConformanceStatusesCount: struct {
				Passed int `json:"passed"`
				Failed int `json:"failed"`
				Error  int `json:"error"`
				Info   int `json:"info"`
				Skip   int `json:"skip"`
			}{
				Passed: summary.Result.QueryResult[kaytuTypes.ConformanceStatusOK],
				Failed: summary.Result.QueryResult[kaytuTypes.ConformanceStatusALARM],
				Error:  summary.Result.QueryResult[kaytuTypes.ConformanceStatusERROR],
				Info:   summary.Result.QueryResult[kaytuTypes.ConformanceStatusINFO],
				Skip:   summary.Result.QueryResult[kaytuTypes.ConformanceStatusSKIP],
			},
			LastCheckTime: time.Unix(evaluatedAt, 0),
		}

		response.Accounts = append(response.Accounts, account)
	}

	for idx, conn := range response.Accounts {
		conn.AccountId = demo.EncodeResponseData(ctx, conn.AccountId)
		conn.AccountName = demo.EncodeResponseData(ctx, conn.AccountName)
		response.Accounts[idx] = conn
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetServicesFindingsSummary godoc
//
//	@Summary		Get services findings summary
//	@Description	Retrieving list of services with their security score and severities findings count
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmarkId		path		string		true	"BenchmarkID"
//	@Param			connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param			connectionGroup	query		[]string	false	"Connection groups to filter by "
//	@Success		200				{object}	api.GetServicesFindingsSummaryResponse
//	@Router			/compliance/api/v1/findings/{benchmarkId}/services [get]
func (h *HttpHandler) GetServicesFindingsSummary(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmarkId")
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	var response api.GetServicesFindingsSummaryResponse
	resp, err := es.ResourceTypesFindingsSummary(h.logger, h.client, connectionIDs, benchmarkID)
	if err != nil {
		return err
	}

	resourceTypes, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx),
		nil, nil, nil, false, nil, 10000, 1)
	if err != nil {
		h.logger.Error("failed to get resource types metadata", zap.Error(err))
		return err
	}
	resourceTypeMap := make(map[string]inventoryApi.ResourceType)
	for _, rt := range resourceTypes.ResourceTypes {
		resourceTypeMap[strings.ToLower(rt.ResourceType)] = rt
	}

	for _, resourceType := range resp.Aggregations.Summaries.Buckets {
		sevMap := make(map[string]int)
		for _, severity := range resourceType.Severity.Buckets {
			sevMap[severity.Key] = severity.DocCount
		}
		resMap := make(map[string]int)
		for _, controlResult := range resourceType.ConformanceStatus.Buckets {
			resMap[controlResult.Key] = controlResult.DocCount
		}

		securityScore := float64(resMap[string(kaytuTypes.ConformanceStatusOK)]) / float64(resourceType.DocCount) * 100.0

		resourceTypeMetadata := resourceTypeMap[strings.ToLower(resourceType.Key)]
		if resourceTypeMetadata.ResourceType == "" {
			resourceTypeMetadata.ResourceType = resourceType.Key
			if resourceTypeMetadata.ResourceType == "" {
				resourceTypeMetadata.ResourceType = "Unknown"
			}
			resourceTypeMetadata.ResourceLabel = resourceType.Key
			if resourceTypeMetadata.ResourceLabel == "" {
				resourceTypeMetadata.ResourceLabel = "Unknown"
			}
		}
		service := api.ServiceFindingsSummary{
			ServiceName:   resourceTypeMetadata.ResourceType,
			ServiceLabel:  resourceTypeMetadata.ResourceLabel,
			SecurityScore: securityScore,
			SeveritiesCount: struct {
				Critical int `json:"critical"`
				High     int `json:"high"`
				Medium   int `json:"medium"`
				Low      int `json:"low"`
				None     int `json:"none"`
			}{
				Critical: sevMap[string(kaytuTypes.FindingSeverityCritical)],
				High:     sevMap[string(kaytuTypes.FindingSeverityHigh)],
				Medium:   sevMap[string(kaytuTypes.FindingSeverityMedium)],
				Low:      sevMap[string(kaytuTypes.FindingSeverityLow)],
				None:     sevMap[string(kaytuTypes.FindingSeverityNone)],
			},
			ConformanceStatusesCount: struct {
				Passed int `json:"passed"`
				Failed int `json:"failed"`
			}{
				Passed: resMap[string(kaytuTypes.ConformanceStatusOK)],
				Failed: resMap[string(kaytuTypes.ConformanceStatusALARM)] +
					resMap[string(kaytuTypes.ConformanceStatusERROR)] +
					resMap[string(kaytuTypes.ConformanceStatusINFO)] +
					resMap[string(kaytuTypes.ConformanceStatusSKIP)],
			},
		}
		response.Services = append(response.Services, service)
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetFindingEvents godoc
//
//	@Summary		Get finding events
//	@Description	Retrieving all compliance finding events with respect to filters.
//	@Tags			compliance
//	@Security		BearerToken
//	@Accept			json
//	@Produce		json
//	@Param			request	body		api.GetFindingEventsRequest	true	"Request Body"
//	@Success		200		{object}	api.GetFindingEventsResponse
//	@Router			/compliance/api/v1/finding_events [post]
func (h *HttpHandler) GetFindingEvents(ctx echo.Context) error {
	var req api.GetFindingEventsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var err error
	req.Filters.ConnectionID, err = httpserver2.ResolveConnectionIDs(ctx, req.Filters.ConnectionID)
	if err != nil {
		return err
	}

	var response api.GetFindingEventsResponse

	if len(req.Filters.ConformanceStatus) == 0 {
		req.Filters.ConformanceStatus = []api.ConformanceStatus{api.ConformanceStatusFailed}
	}

	esConformanceStatuses := make([]kaytuTypes.ConformanceStatus, 0, len(req.Filters.ConformanceStatus))
	for _, status := range req.Filters.ConformanceStatus {
		esConformanceStatuses = append(esConformanceStatuses, status.GetEsConformanceStatuses()...)
	}

	if len(req.Sort) == 0 {
		req.Sort = []api.FindingEventsSort{
			{ConformanceStatus: utils.GetPointer(api.SortDirectionDescending)},
		}
	}

	if len(req.AfterSortKey) != 0 {
		expectedLen := len(req.Sort) + 1
		if len(req.AfterSortKey) != expectedLen {
			return echo.NewHTTPError(http.StatusBadRequest, "sort key length should be zero or match a returned sort key from previous response")
		}
	}

	var evaluatedAtFrom, evaluatedAtTo *time.Time
	if req.Filters.EvaluatedAt.From != nil && *req.Filters.EvaluatedAt.From != 0 {
		evaluatedAtFrom = utils.GetPointer(time.Unix(*req.Filters.EvaluatedAt.From, 0))
	}
	if req.Filters.EvaluatedAt.To != nil && *req.Filters.EvaluatedAt.To != 0 {
		evaluatedAtTo = utils.GetPointer(time.Unix(*req.Filters.EvaluatedAt.To, 0))
	}

	res, totalCount, err := es.FindingEventsQuery(h.logger, h.client,
		req.Filters.FindingID, req.Filters.KaytuResourceID,
		req.Filters.Connector, req.Filters.ConnectionID, req.Filters.NotConnectionID,
		req.Filters.ResourceType,
		req.Filters.BenchmarkID, req.Filters.ControlID, req.Filters.Severity,
		evaluatedAtFrom, evaluatedAtTo,
		req.Filters.StateActive, esConformanceStatuses, req.Sort, req.Limit, req.AfterSortKey)
	if err != nil {
		h.logger.Error("failed to get findings", zap.Error(err))
		return err
	}

	allSources, err := h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), nil)
	if err != nil {
		h.logger.Error("failed to get sources", zap.Error(err))
		return err
	}
	allConnectionsMap := make(map[string]*onboardApi.Connection)
	for _, src := range allSources {
		src := src
		allConnectionsMap[src.ID.String()] = &src
	}

	resourceTypeMetadata, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx),
		nil, nil, nil, false, nil, 10000, 1)
	if err != nil {
		h.logger.Error("failed to get resource type metadata", zap.Error(err))
		return err
	}
	resourceTypeMetadataMap := make(map[string]*inventoryApi.ResourceType)
	for _, item := range resourceTypeMetadata.ResourceTypes {
		item := item
		resourceTypeMetadataMap[strings.ToLower(item.ResourceType)] = &item
	}

	for _, h := range res {
		findingEvent := api.GetAPIFindingEventFromESFindingEvent(h.Source)
		if rtMetadata, ok := resourceTypeMetadataMap[strings.ToLower(h.Source.ResourceType)]; ok {
			findingEvent.ResourceType = rtMetadata.ResourceLabel
		}
		if connection, ok := allConnectionsMap[h.Source.ConnectionID]; ok {
			findingEvent.ProviderConnectionID = connection.ConnectionID
			findingEvent.ProviderConnectionName = connection.ConnectionName
		}
		findingEvent.SortKey = h.Sort
		response.FindingEvents = append(response.FindingEvents, findingEvent)
	}
	response.TotalCount = totalCount

	return ctx.JSON(http.StatusOK, response)
}

// GetFindingEventFilterValues godoc
//
//	@Summary		Get possible values for finding event filters
//	@Description	Retrieving possible values for finding event filters.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			request	body		api.FindingEventFilters	true	"Request Body"
//	@Success		200		{object}	api.FindingEventFiltersWithMetadata
//	@Router			/compliance/api/v1/finding_events/filters [post]
func (h *HttpHandler) GetFindingEventFilterValues(ctx echo.Context) error {
	var req api.FindingEventFilters
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var err error
	req.ConnectionID, err = httpserver2.ResolveConnectionIDs(ctx, req.ConnectionID)
	if err != nil {
		return err
	}

	if len(req.ConformanceStatus) == 0 {
		req.ConformanceStatus = []api.ConformanceStatus{api.ConformanceStatusFailed}
	}

	esConformanceStatuses := make([]kaytuTypes.ConformanceStatus, 0, len(req.ConformanceStatus))
	for _, status := range req.ConformanceStatus {
		esConformanceStatuses = append(esConformanceStatuses, status.GetEsConformanceStatuses()...)
	}

	var evaluatedAtFrom, evaluatedAtTo *time.Time
	if req.EvaluatedAt.From != nil && *req.EvaluatedAt.From != 0 {
		evaluatedAtFrom = utils.GetPointer(time.Unix(*req.EvaluatedAt.From, 0))
	}
	if req.EvaluatedAt.To != nil && *req.EvaluatedAt.To != 0 {
		evaluatedAtTo = utils.GetPointer(time.Unix(*req.EvaluatedAt.To, 0))
	}

	resourceTypeMetadata, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx),
		nil, nil, nil, false, nil, 10000, 1)
	if err != nil {
		h.logger.Error("failed to get resource type metadata", zap.Error(err))
		return err
	}
	resourceTypeMetadataMap := make(map[string]*inventoryApi.ResourceType)
	for _, item := range resourceTypeMetadata.ResourceTypes {
		item := item
		resourceTypeMetadataMap[strings.ToLower(item.ResourceType)] = &item
	}

	resourceCollectionMetadata, err := h.inventoryClient.ListResourceCollections(httpclient.FromEchoContext(ctx))
	if err != nil {
		h.logger.Error("failed to get resource collection metadata", zap.Error(err))
		return err
	}
	resourceCollectionMetadataMap := make(map[string]*inventoryApi.ResourceCollection)
	for _, item := range resourceCollectionMetadata {
		item := item
		resourceCollectionMetadataMap[item.ID] = &item
	}

	connectionMetadata, err := h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), nil)
	if err != nil {
		h.logger.Error("failed to get connections", zap.Error(err))
		return err
	}
	connectionMetadataMap := make(map[string]*onboardApi.Connection)
	for _, item := range connectionMetadata {
		item := item
		connectionMetadataMap[item.ID.String()] = &item
	}

	benchmarkMetadata, err := h.db.ListBenchmarksBare()
	if err != nil {
		h.logger.Error("failed to get benchmarks", zap.Error(err))
		return err
	}
	benchmarkMetadataMap := make(map[string]*db.Benchmark)
	for _, item := range benchmarkMetadata {
		item := item
		benchmarkMetadataMap[item.ID] = &item
	}

	controlMetadata, err := h.db.ListControlsBare()
	if err != nil {
		h.logger.Error("failed to get controls", zap.Error(err))
		return err
	}
	controlMetadataMap := make(map[string]*db.Control)
	for _, item := range controlMetadata {
		item := item
		controlMetadataMap[item.ID] = &item
	}

	possibleFilters, err := es.FindingEventsFiltersQuery(h.logger, h.client,
		req.FindingID, req.KaytuResourceID, req.Connector, req.ConnectionID, req.NotConnectionID,
		req.ResourceType,
		req.BenchmarkID, req.ControlID,
		req.Severity,
		evaluatedAtFrom, evaluatedAtTo,
		req.StateActive, esConformanceStatuses)
	if err != nil {
		h.logger.Error("failed to get possible filters", zap.Error(err))
		return err
	}
	response := api.FindingFiltersWithMetadata{}
	for _, item := range possibleFilters.Aggregations.BenchmarkIDFilter.Buckets {
		if benchmark, ok := benchmarkMetadataMap[item.Key]; ok {
			response.BenchmarkID = append(response.BenchmarkID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: benchmark.Title,
				Count:       utils.GetPointer(item.DocCount),
			})
		} else {
			response.BenchmarkID = append(response.BenchmarkID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: item.Key,
				Count:       utils.GetPointer(item.DocCount),
			})
		}
	}
	for _, item := range possibleFilters.Aggregations.ControlIDFilter.Buckets {
		if control, ok := controlMetadataMap[item.Key]; ok {
			response.ControlID = append(response.ControlID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: control.Title,
				Count:       utils.GetPointer(item.DocCount),
			})
		} else {
			response.ControlID = append(response.ControlID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: item.Key,
				Count:       utils.GetPointer(item.DocCount),
			})
		}
	}
	if len(possibleFilters.Aggregations.ConnectorFilter.Buckets) > 0 {
		for _, bucket := range possibleFilters.Aggregations.ConnectorFilter.Buckets {
			connector, _ := source.ParseType(bucket.Key)
			response.Connector = append(response.Connector, api.FilterWithMetadata{
				Key:         connector.String(),
				DisplayName: connector.String(),
				Count:       utils.GetPointer(bucket.DocCount),
			})
		}
	}
	for _, item := range possibleFilters.Aggregations.ResourceTypeFilter.Buckets {
		if rtMetadata, ok := resourceTypeMetadataMap[strings.ToLower(item.Key)]; ok {
			response.ResourceTypeID = append(response.ResourceTypeID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: rtMetadata.ResourceLabel,
				Count:       utils.GetPointer(item.DocCount),
			})
		} else if item.Key == "" {
			response.ResourceTypeID = append(response.ResourceTypeID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: "Unknown",
			})
		} else {
			response.ResourceTypeID = append(response.ResourceTypeID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: item.Key,
				Count:       utils.GetPointer(item.DocCount),
			})
		}
	}

	for _, item := range possibleFilters.Aggregations.ConnectionIDFilter.Buckets {
		if connection, ok := connectionMetadataMap[item.Key]; ok {
			response.ConnectionID = append(response.ConnectionID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: connection.ConnectionName,
				Count:       utils.GetPointer(item.DocCount),
			})
		} else {
			response.ConnectionID = append(response.ConnectionID, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: item.Key,
				Count:       utils.GetPointer(item.DocCount),
			})
		}
	}

	for _, item := range possibleFilters.Aggregations.ResourceCollectionFilter.Buckets {
		if resourceCollection, ok := resourceCollectionMetadataMap[item.Key]; ok {
			response.ResourceCollection = append(response.ResourceCollection, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: resourceCollection.Name,
				Count:       utils.GetPointer(item.DocCount),
			})
		} else {
			response.ResourceCollection = append(response.ResourceCollection, api.FilterWithMetadata{
				Key:         item.Key,
				DisplayName: item.Key,
				Count:       utils.GetPointer(item.DocCount),
			})
		}
	}

	for _, item := range possibleFilters.Aggregations.SeverityFilter.Buckets {
		response.Severity = append(response.Severity, api.FilterWithMetadata{
			Key:         item.Key,
			DisplayName: item.Key,
			Count:       utils.GetPointer(item.DocCount),
		})
	}

	for _, item := range possibleFilters.Aggregations.StateActiveFilter.Buckets {
		response.StateActive = append(response.StateActive, api.FilterWithMetadata{
			Key:         item.KeyAsString,
			DisplayName: item.KeyAsString,
			Count:       utils.GetPointer(item.DocCount),
		})
	}

	apiConformanceStatuses := make(map[api.ConformanceStatus]int)
	for _, item := range possibleFilters.Aggregations.ConformanceStatusFilter.Buckets {
		if kaytuTypes.ParseConformanceStatus(item.Key).IsPassed() {
			apiConformanceStatuses[api.ConformanceStatusPassed] += item.DocCount
		} else {
			apiConformanceStatuses[api.ConformanceStatusFailed] += item.DocCount
		}
	}
	for status, count := range apiConformanceStatuses {
		count := count
		response.ConformanceStatus = append(response.ConformanceStatus, api.FilterWithMetadata{
			Key:         string(status),
			DisplayName: string(status),
			Count:       &count,
		})
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetSingleFindingEvent
//
//	@Summary		Get single finding event
//	@Description	Retrieving single finding event
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			findingID	path		string	true	"FindingID"
//	@Success		200			{object}	api.FindingEvent
//	@Router			/compliance/api/v1/finding_events/single/{id} [get]
func (h *HttpHandler) GetSingleFindingEvent(ctx echo.Context) error {
	findingEventID := ctx.Param("id")

	findingEvent, err := es.FetchFindingEventByID(h.logger, h.client, findingEventID)
	if err != nil {
		h.logger.Error("failed to fetch findingEvent by id", zap.Error(err))
		return err
	}
	if findingEvent == nil {
		return echo.NewHTTPError(http.StatusNotFound, "findingEvent not found")
	}

	apiFindingEvent := api.GetAPIFindingEventFromESFindingEvent(*findingEvent)

	connection, err := h.onboardClient.GetSource(httpclient.FromEchoContext(ctx), findingEvent.ConnectionID)
	if err != nil {
		h.logger.Error("failed to get connection", zap.Error(err), zap.String("connection_id", findingEvent.ConnectionID))
		return err
	}
	apiFindingEvent.ProviderConnectionID = connection.ConnectionID
	apiFindingEvent.ProviderConnectionName = connection.ConnectionName

	if len(findingEvent.ResourceType) > 0 {
		resourceTypeMetadata, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx),
			nil, nil,
			[]string{findingEvent.ResourceType}, false, nil, 10000, 1)
		if err != nil {
			h.logger.Error("failed to get resource type metadata", zap.Error(err))
			return err
		}
		if len(resourceTypeMetadata.ResourceTypes) > 0 {
			apiFindingEvent.ResourceTypeName = resourceTypeMetadata.ResourceTypes[0].ResourceLabel
		}
	}

	return ctx.JSON(http.StatusOK, apiFindingEvent)
}

// ListResourceFindings godoc
//
//	@Summary		List resource findings
//	@Description	Retrieving list of resource findings
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			request	body		api.ListResourceFindingsRequest	true	"Request"
//	@Success		200		{object}	api.ListResourceFindingsResponse
//	@Router			/compliance/api/v1/resource_findings [post]
func (h *HttpHandler) ListResourceFindings(ctx echo.Context) error {
	var req api.ListResourceFindingsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var err error
	req.Filters.ConnectionID, err = httpserver2.ResolveConnectionIDs(ctx, req.Filters.ConnectionID)
	if err != nil {
		return err
	}

	if len(req.AfterSortKey) != 0 {
		expectedLen := len(req.Sort) + 1
		if len(req.AfterSortKey) != expectedLen {
			return echo.NewHTTPError(http.StatusBadRequest, "sort key length should be zero or match a returned sort key from previous response")
		}
	}

	var evaluatedAtFrom, evaluatedAtTo *time.Time
	if req.Filters.EvaluatedAt.From != nil && *req.Filters.EvaluatedAt.From != 0 {
		evaluatedAtFrom = utils.GetPointer(time.Unix(*req.Filters.EvaluatedAt.From, 0))
	}
	if req.Filters.EvaluatedAt.To != nil && *req.Filters.EvaluatedAt.To != 0 {
		evaluatedAtTo = utils.GetPointer(time.Unix(*req.Filters.EvaluatedAt.To, 0))
	}

	connections, err := h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), nil)
	if err != nil {
		h.logger.Error("failed to get connections", zap.Error(err))
		return err
	}
	connectionMap := make(map[string]*onboardApi.Connection)
	for _, connection := range connections {
		connection := connection
		connectionMap[connection.ID.String()] = &connection
	}

	resourceTypes, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx), nil, nil, nil, false, nil, 10000, 1)
	if err != nil {
		h.logger.Error("failed to get resource types metadata", zap.Error(err))
		return err
	}
	resourceTypeMap := make(map[string]*inventoryApi.ResourceType)
	for _, rt := range resourceTypes.ResourceTypes {
		rt := rt
		resourceTypeMap[strings.ToLower(rt.ResourceType)] = &rt
	}

	if len(req.Filters.ConformanceStatus) == 0 {
		req.Filters.ConformanceStatus = []api.ConformanceStatus{
			api.ConformanceStatusFailed,
		}
	}

	esConformanceStatuses := make([]kaytuTypes.ConformanceStatus, 0, len(req.Filters.ConformanceStatus))
	for _, status := range req.Filters.ConformanceStatus {
		esConformanceStatuses = append(esConformanceStatuses, status.GetEsConformanceStatuses()...)
	}

	resourceFindings, totalCount, err := es.ResourceFindingsQuery(h.logger, h.client,
		req.Filters.Connector, req.Filters.ConnectionID, req.Filters.NotConnectionID,
		req.Filters.ResourceCollection, req.Filters.ResourceTypeID,
		req.Filters.BenchmarkID, req.Filters.ControlID, req.Filters.Severity,
		evaluatedAtFrom, evaluatedAtTo,
		esConformanceStatuses, req.Sort, req.Limit, req.AfterSortKey)
	if err != nil {
		h.logger.Error("failed to get resource findings", zap.Error(err))
		return err
	}

	response := api.ListResourceFindingsResponse{
		TotalCount:       int(totalCount),
		ResourceFindings: nil,
	}

	for _, resourceFinding := range resourceFindings {
		apiRf := api.GetAPIResourceFinding(resourceFinding.Source)
		if connection, ok := connectionMap[apiRf.ConnectionID]; ok {
			apiRf.ProviderConnectionID = connection.ConnectionID
			apiRf.ProviderConnectionName = connection.ConnectionName
		}
		if resourceType, ok := resourceTypeMap[strings.ToLower(apiRf.ResourceType)]; ok {
			apiRf.ResourceTypeLabel = resourceType.ResourceLabel
		}
		apiRf.SortKey = resourceFinding.Sort
		response.ResourceFindings = append(response.ResourceFindings, apiRf)
	}

	return ctx.JSON(http.StatusOK, response)
}

// GetControlRemediation godoc
//
//	@Summary	Get control remediation using AI
//	@Security	BearerToken
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		controlID	path		string	true	"ControlID"
//	@Success	200			{object}	api.BenchmarkRemediation
//	@Router		/compliance/api/v1/ai/control/{controlID}/remediation [post]
func (h *HttpHandler) GetControlRemediation(ctx echo.Context) error {
	controlID := ctx.Param("controlID")

	control, err := h.db.GetControl(controlID)
	if err != nil {
		return err
	}

	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You will be provided with a problem on AWS, and your task is to create a numbered list of how to fix it using AWS console.",
			},
		},
	}

	req.Messages = append(req.Messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: control.Title,
	})

	resp, err := h.openAIClient.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, api.BenchmarkRemediation{Remediation: resp.Choices[0].Message.Content})
}

func addToControlSeverityResult(controlSeverityResult api.BenchmarkControlsSeverityStatus, control *db.Control, controlResult types.ControlResult) api.BenchmarkControlsSeverityStatus {
	if control == nil {
		control = &db.Control{
			Severity: kaytuTypes.FindingSeverityNone,
		}
	}
	switch control.Severity {
	case kaytuTypes.FindingSeverityCritical:
		controlSeverityResult.Total.TotalCount++
		controlSeverityResult.Critical.TotalCount++
		if controlResult.Passed {
			controlSeverityResult.Total.PassedCount++
			controlSeverityResult.Critical.PassedCount++
		}
	case kaytuTypes.FindingSeverityHigh:
		controlSeverityResult.Total.TotalCount++
		controlSeverityResult.High.TotalCount++
		if controlResult.Passed {
			controlSeverityResult.Total.PassedCount++
			controlSeverityResult.High.PassedCount++
		}
	case kaytuTypes.FindingSeverityMedium:
		controlSeverityResult.Total.TotalCount++
		controlSeverityResult.Medium.TotalCount++
		if controlResult.Passed {
			controlSeverityResult.Total.PassedCount++
			controlSeverityResult.Medium.PassedCount++
		}
	case kaytuTypes.FindingSeverityLow:
		controlSeverityResult.Total.TotalCount++
		controlSeverityResult.Low.TotalCount++
		if controlResult.Passed {
			controlSeverityResult.Total.PassedCount++
			controlSeverityResult.Low.PassedCount++
		}
	case kaytuTypes.FindingSeverityNone, "":
		controlSeverityResult.Total.TotalCount++
		controlSeverityResult.None.TotalCount++
		if controlResult.Passed {
			controlSeverityResult.Total.PassedCount++
			controlSeverityResult.None.PassedCount++
		}
	}
	return controlSeverityResult
}

// ListBenchmarksSummary godoc
//
//	@Summary		List benchmarks summaries
//	@Description	Retrieving a summary of all benchmarks and their associated checks and results within a specified time interval.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			connectionId		query		[]string		false	"Connection IDs to filter by"
//	@Param			connectionGroup		query		[]string		false	"Connection groups to filter by "
//	@Param			resourceCollection	query		[]string		false	"Resource collection IDs to filter by"
//	@Param			connector			query		[]source.Type	false	"Connector type to filter by"
//	@Param			tag					query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			timeAt				query		int				false	"timestamp for values in epoch seconds"
//	@Param			topAccountCount		query		int				false	"Top account count"	default(3)
//	@Success		200					{object}	api.GetBenchmarksSummaryResponse
//	@Router			/compliance/api/v1/benchmarks/summary [get]
func (h *HttpHandler) ListBenchmarksSummary(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver2.QueryArrayParam(ctx, "tag"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	if len(connectionIDs) > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "too many connection IDs")
	}

	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
	timeAt := time.Now()
	if timeAtStr := ctx.QueryParam("timeAt"); timeAtStr != "" {
		timeAtInt, err := strconv.ParseInt(timeAtStr, 10, 64)
		if err != nil {
			return err
		}
		timeAt = time.Unix(timeAtInt, 0)
	}
	topAccountCount := 3
	if topAccountCountStr := ctx.QueryParam("topAccountCount"); topAccountCountStr != "" {
		count, err := strconv.ParseInt(topAccountCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid topAccountCount")
		}
		topAccountCount = int(count)
	}

	var response api.GetBenchmarksSummaryResponse

	// tracer :
	outputS, span2 := tracer.Start(ctx.Request().Context(), "new_ListRootBenchmarks", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_ListRootBenchmarks")

	benchmarks, err := h.db.ListRootBenchmarks(tagMap)
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return err
	}
	span2.End()

	controls, err := h.db.ListControlsBare()
	if err != nil {
		h.logger.Error("failed to get controls", zap.Error(err))
		return err
	}
	controlsMap := make(map[string]*db.Control)
	for _, control := range controls {
		control := control
		controlsMap[strings.ToLower(control.ID)] = &control
	}

	benchmarkIDs := make([]string, 0, len(benchmarks))
	for _, b := range benchmarks {
		benchmarkIDs = append(benchmarkIDs, b.ID)
	}

	summariesAtTime, err := es.ListBenchmarkSummariesAtTime(h.logger, h.client, benchmarkIDs, connectionIDs, resourceCollections, timeAt)
	if err != nil {
		h.logger.Error("failed to fetch benchmark summaries", zap.Error(err))
		return err
	}
	// tracer :
	_, span3 := tracer.Start(outputS, "new_PopulateConnectors(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span3.SetName("new_PopulateConnectors(loop)")

	passedResourcesResult, err := es.GetPerBenchmarkResourceSeverityResult(h.logger, h.client, benchmarkIDs, connectionIDs, resourceCollections, nil, []kaytuTypes.ConformanceStatus{
		kaytuTypes.ConformanceStatusOK,
	})
	if err != nil {
		h.logger.Error("failed to fetch per benchmark resource severity result for passed", zap.Error(err))
		return err
	}

	allResourcesResult, err := es.GetPerBenchmarkResourceSeverityResult(h.logger, h.client, benchmarkIDs, connectionIDs, resourceCollections, nil, nil)
	if err != nil {
		h.logger.Error("failed to fetch per benchmark resource severity result for all", zap.Error(err))
		return err
	}

	for _, b := range benchmarks {
		be := b.ToApi()
		if len(connectors) > 0 && !utils.IncludesAny(be.Connectors, connectors) {
			continue
		}

		summaryAtTime := summariesAtTime[b.ID]
		csResult := api.ConformanceStatusSummary{}
		sResult := kaytuTypes.SeverityResult{}
		controlSeverityResult := api.BenchmarkControlsSeverityStatus{}

		if len(connectionIDs) > 0 {
			for _, connectionID := range connectionIDs {
				csResult.AddESConformanceStatusMap(summaryAtTime.Connections.Connections[connectionID].Result.QueryResult)
				sResult.AddResultMap(summaryAtTime.Connections.Connections[connectionID].Result.SeverityResult)
				response.TotalConformanceStatusSummary.AddESConformanceStatusMap(summaryAtTime.Connections.Connections[connectionID].Result.QueryResult)
				response.TotalChecks.AddResultMap(summaryAtTime.Connections.Connections[connectionID].Result.SeverityResult)
				for controlId, controlResult := range summaryAtTime.Connections.Connections[connectionID].Controls {
					control := controlsMap[strings.ToLower(controlId)]
					controlSeverityResult = addToControlSeverityResult(controlSeverityResult, control, controlResult)
				}
			}
		} else if len(resourceCollections) > 0 {
			for _, resourceCollection := range resourceCollections {
				csResult.AddESConformanceStatusMap(summaryAtTime.ResourceCollections[resourceCollection].BenchmarkResult.Result.QueryResult)
				sResult.AddResultMap(summaryAtTime.ResourceCollections[resourceCollection].BenchmarkResult.Result.SeverityResult)
				response.TotalConformanceStatusSummary.AddESConformanceStatusMap(summaryAtTime.ResourceCollections[resourceCollection].BenchmarkResult.Result.QueryResult)
				response.TotalChecks.AddResultMap(summaryAtTime.ResourceCollections[resourceCollection].BenchmarkResult.Result.SeverityResult)
				for controlId, controlResult := range summaryAtTime.ResourceCollections[resourceCollection].BenchmarkResult.Controls {
					control := controlsMap[strings.ToLower(controlId)]
					controlSeverityResult = addToControlSeverityResult(controlSeverityResult, control, controlResult)
				}
			}
		} else {
			csResult.AddESConformanceStatusMap(summaryAtTime.Connections.BenchmarkResult.Result.QueryResult)
			sResult.AddResultMap(summaryAtTime.Connections.BenchmarkResult.Result.SeverityResult)
			response.TotalConformanceStatusSummary.AddESConformanceStatusMap(summaryAtTime.Connections.BenchmarkResult.Result.QueryResult)
			response.TotalChecks.AddResultMap(summaryAtTime.Connections.BenchmarkResult.Result.SeverityResult)
			for controlId, controlResult := range summaryAtTime.Connections.BenchmarkResult.Controls {
				control := controlsMap[strings.ToLower(controlId)]
				controlSeverityResult = addToControlSeverityResult(controlSeverityResult, control, controlResult)
			}
		}

		topConnections := make([]api.TopFieldRecord, 0, topAccountCount)
		if topAccountCount > 0 {
			topFieldResponse, err := es.FindingsTopFieldQuery(h.logger, h.client, "connectionID", connectors, nil, connectionIDs, nil, []string{b.ID}, nil, nil, []kaytuTypes.ConformanceStatus{
				kaytuTypes.ConformanceStatusALARM,
				kaytuTypes.ConformanceStatusERROR,
				kaytuTypes.ConformanceStatusINFO,
				kaytuTypes.ConformanceStatusSKIP,
			}, []bool{true}, topAccountCount)
			if err != nil {
				h.logger.Error("failed to fetch findings top field", zap.Error(err))
				return err
			}
			topFieldTotalResponse, err := es.FindingsTopFieldQuery(h.logger, h.client, "connectionID", connectors, nil, connectionIDs, nil, []string{b.ID}, nil, nil, []kaytuTypes.ConformanceStatus{
				kaytuTypes.ConformanceStatusALARM,
				kaytuTypes.ConformanceStatusERROR,
				kaytuTypes.ConformanceStatusINFO,
				kaytuTypes.ConformanceStatusSKIP,
				kaytuTypes.ConformanceStatusOK,
			}, []bool{true}, topAccountCount)
			if err != nil {
				h.logger.Error("failed to fetch findings top field total", zap.Error(err))
				return err
			}
			totalCountMap := make(map[string]int)
			for _, item := range topFieldTotalResponse.Aggregations.FieldFilter.Buckets {
				totalCountMap[item.Key] += item.DocCount
			}

			resConnectionIDs := make([]string, 0, len(topFieldResponse.Aggregations.FieldFilter.Buckets))
			for _, item := range topFieldResponse.Aggregations.FieldFilter.Buckets {
				resConnectionIDs = append(resConnectionIDs, item.Key)
			}
			if len(resConnectionIDs) > 0 {
				connections, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), resConnectionIDs)
				if err != nil {
					h.logger.Error("failed to get connections", zap.Error(err))
					return err
				}
				connectionMap := make(map[string]*onboardApi.Connection)
				for _, connection := range connections {
					connection := connection
					connectionMap[connection.ID.String()] = &connection
				}

				for _, item := range topFieldResponse.Aggregations.FieldFilter.Buckets {
					topConnections = append(topConnections, api.TopFieldRecord{
						Connection: connectionMap[item.Key],
						Count:      item.DocCount,
						TotalCount: totalCountMap[item.Key],
					})
				}
			}
		}

		resourcesSeverityResult := api.BenchmarkResourcesSeverityStatus{}
		allResources := allResourcesResult[b.ID]
		resourcesSeverityResult.Total.TotalCount = allResources.TotalCount
		resourcesSeverityResult.Critical.TotalCount = allResources.CriticalCount
		resourcesSeverityResult.High.TotalCount = allResources.HighCount
		resourcesSeverityResult.Medium.TotalCount = allResources.MediumCount
		resourcesSeverityResult.Low.TotalCount = allResources.LowCount
		resourcesSeverityResult.None.TotalCount = allResources.NoneCount
		passedResource := passedResourcesResult[b.ID]
		resourcesSeverityResult.Total.PassedCount = passedResource.TotalCount
		resourcesSeverityResult.Critical.PassedCount = passedResource.CriticalCount
		resourcesSeverityResult.High.PassedCount = passedResource.HighCount
		resourcesSeverityResult.Medium.PassedCount = passedResource.MediumCount
		resourcesSeverityResult.Low.PassedCount = passedResource.LowCount
		resourcesSeverityResult.None.PassedCount = passedResource.NoneCount

		response.BenchmarkSummary = append(response.BenchmarkSummary, api.BenchmarkEvaluationSummary{
			Benchmark:                be,
			ConformanceStatusSummary: csResult,
			Checks:                   sResult,
			ControlsSeverityStatus:   controlSeverityResult,
			ResourcesSeverityStatus:  resourcesSeverityResult,
			EvaluatedAt:              utils.GetPointer(time.Unix(summaryAtTime.EvaluatedAtEpoch, 0)),
			LastJobStatus:            "",
			TopConnections:           topConnections,
		})
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
//	@Param			benchmark_id		path		string			true	"Benchmark ID"
//	@Param			connectionId		query		[]string		false	"Connection IDs to filter by"
//	@Param			connectionGroup		query		[]string		false	"Connection groups to filter by "
//	@Param			resourceCollection	query		[]string		false	"Resource collection IDs to filter by"
//	@Param			connector			query		[]source.Type	false	"Connector type to filter by"
//	@Param			timeAt				query		int				false	"timestamp for values in epoch seconds"
//	@Param			topAccountCount		query		int				false	"Top account count"	default(3)
//	@Success		200					{object}	api.BenchmarkEvaluationSummary
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id}/summary [get]
func (h *HttpHandler) GetBenchmarkSummary(ctx echo.Context) error {
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	if len(connectionIDs) > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "too many connection IDs")
	}
	topAccountCount := 3
	if topAccountCountStr := ctx.QueryParam("topAccountCount"); topAccountCountStr != "" {
		count, err := strconv.ParseInt(topAccountCountStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid topAccountCount")
		}
		topAccountCount = int(count)
	}

	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
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
	_, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindServer))
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

	if len(connectors) > 0 && !utils.IncludesAny(be.Connectors, connectors) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connector")
	}

	controls, err := h.db.ListControlsByBenchmarkID(benchmarkID)
	if err != nil {
		h.logger.Error("failed to get controls", zap.Error(err))
		return err
	}
	controlsMap := make(map[string]*db.Control)
	for _, control := range controls {
		control := control
		controlsMap[strings.ToLower(control.ID)] = &control
	}

	summariesAtTime, err := es.ListBenchmarkSummariesAtTime(h.logger, h.client, []string{benchmarkID}, connectionIDs, resourceCollections, timeAt)
	if err != nil {
		return err
	}

	passedResourcesResult, err := es.GetPerBenchmarkResourceSeverityResult(h.logger, h.client, []string{benchmarkID}, connectionIDs, resourceCollections, nil, []kaytuTypes.ConformanceStatus{
		kaytuTypes.ConformanceStatusOK,
	})
	if err != nil {
		h.logger.Error("failed to fetch per benchmark resource severity result for passed", zap.Error(err))
		return err
	}

	allResourcesResult, err := es.GetPerBenchmarkResourceSeverityResult(h.logger, h.client, []string{benchmarkID}, connectionIDs, resourceCollections, nil, nil)
	if err != nil {
		h.logger.Error("failed to fetch per benchmark resource severity result for all", zap.Error(err))
		return err
	}

	summaryAtTime := summariesAtTime[benchmarkID]

	csResult := api.ConformanceStatusSummary{}
	sResult := kaytuTypes.SeverityResult{}
	controlSeverityResult := api.BenchmarkControlsSeverityStatus{}
	if len(connectionIDs) > 0 {
		for _, connectionID := range connectionIDs {
			csResult.AddESConformanceStatusMap(summaryAtTime.Connections.Connections[connectionID].Result.QueryResult)
			sResult.AddResultMap(summaryAtTime.Connections.Connections[connectionID].Result.SeverityResult)
			for controlId, controlResult := range summaryAtTime.Connections.Connections[connectionID].Controls {
				control := controlsMap[strings.ToLower(controlId)]
				controlSeverityResult = addToControlSeverityResult(controlSeverityResult, control, controlResult)
			}
		}
	} else if len(resourceCollections) > 0 {
		for _, resourceCollection := range resourceCollections {
			csResult.AddESConformanceStatusMap(summaryAtTime.ResourceCollections[resourceCollection].BenchmarkResult.Result.QueryResult)
			sResult.AddResultMap(summaryAtTime.ResourceCollections[resourceCollection].BenchmarkResult.Result.SeverityResult)
			for controlId, controlResult := range summaryAtTime.ResourceCollections[resourceCollection].BenchmarkResult.Controls {
				control := controlsMap[strings.ToLower(controlId)]
				controlSeverityResult = addToControlSeverityResult(controlSeverityResult, control, controlResult)
			}
		}
	} else {
		csResult.AddESConformanceStatusMap(summaryAtTime.Connections.BenchmarkResult.Result.QueryResult)
		sResult.AddResultMap(summaryAtTime.Connections.BenchmarkResult.Result.SeverityResult)
		for controlId, controlResult := range summaryAtTime.Connections.BenchmarkResult.Controls {
			control := controlsMap[strings.ToLower(controlId)]
			controlSeverityResult = addToControlSeverityResult(controlSeverityResult, control, controlResult)
		}
	}

	lastJob, err := h.schedulerClient.GetLatestComplianceJobForBenchmark(httpclient.FromEchoContext(ctx), benchmarkID)
	if err != nil {
		h.logger.Error("failed to get latest compliance job for benchmark", zap.Error(err), zap.String("benchmarkID", benchmarkID))
		return err
	}

	var lastJobStatus string
	if lastJob != nil {
		lastJobStatus = string(lastJob.Status)
	}

	topConnections := make([]api.TopFieldRecord, 0, topAccountCount)
	if topAccountCount > 0 {
		res, err := es.FindingsTopFieldQuery(h.logger, h.client, "connectionID", connectors, nil, connectionIDs, nil, []string{benchmark.ID}, nil, nil, []kaytuTypes.ConformanceStatus{
			kaytuTypes.ConformanceStatusALARM,
			kaytuTypes.ConformanceStatusERROR,
			kaytuTypes.ConformanceStatusINFO,
			kaytuTypes.ConformanceStatusSKIP,
		}, []bool{true}, topAccountCount)
		if err != nil {
			h.logger.Error("failed to fetch findings top field", zap.Error(err))
			return err
		}

		topFieldTotalResponse, err := es.FindingsTopFieldQuery(h.logger, h.client, "connectionID", connectors, nil, connectionIDs, nil, []string{benchmark.ID}, nil, nil, []kaytuTypes.ConformanceStatus{
			kaytuTypes.ConformanceStatusALARM,
			kaytuTypes.ConformanceStatusERROR,
			kaytuTypes.ConformanceStatusINFO,
			kaytuTypes.ConformanceStatusSKIP,
		}, []bool{true}, topAccountCount)
		if err != nil {
			h.logger.Error("failed to fetch findings top field total", zap.Error(err))
			return err
		}
		totalCountMap := make(map[string]int)
		for _, item := range topFieldTotalResponse.Aggregations.FieldFilter.Buckets {
			totalCountMap[item.Key] += item.DocCount
		}

		resConnectionIDs := make([]string, 0, len(res.Aggregations.FieldFilter.Buckets))
		for _, item := range res.Aggregations.FieldFilter.Buckets {
			resConnectionIDs = append(resConnectionIDs, item.Key)
		}
		if len(resConnectionIDs) > 0 {
			connections, err := h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), resConnectionIDs)
			if err != nil {
				h.logger.Error("failed to get connections", zap.Error(err))
				return err
			}
			connectionMap := make(map[string]*onboardApi.Connection)
			for _, connection := range connections {
				connection := connection
				connectionMap[connection.ID.String()] = &connection
			}

			for _, item := range res.Aggregations.FieldFilter.Buckets {
				topConnections = append(topConnections, api.TopFieldRecord{
					Connection: connectionMap[item.Key],
					Count:      item.DocCount,
					TotalCount: totalCountMap[item.Key],
				})
			}
		}
	}

	resourcesSeverityResult := api.BenchmarkResourcesSeverityStatus{}
	allResources := allResourcesResult[benchmarkID]
	resourcesSeverityResult.Total.TotalCount = allResources.TotalCount
	resourcesSeverityResult.Critical.TotalCount = allResources.CriticalCount
	resourcesSeverityResult.High.TotalCount = allResources.HighCount
	resourcesSeverityResult.Medium.TotalCount = allResources.MediumCount
	resourcesSeverityResult.Low.TotalCount = allResources.LowCount
	resourcesSeverityResult.None.TotalCount = allResources.NoneCount
	passedResource := passedResourcesResult[benchmarkID]
	resourcesSeverityResult.Total.PassedCount = passedResource.TotalCount
	resourcesSeverityResult.Critical.PassedCount = passedResource.CriticalCount
	resourcesSeverityResult.High.PassedCount = passedResource.HighCount
	resourcesSeverityResult.Medium.PassedCount = passedResource.MediumCount
	resourcesSeverityResult.Low.PassedCount = passedResource.LowCount
	resourcesSeverityResult.None.PassedCount = passedResource.NoneCount

	response := api.BenchmarkEvaluationSummary{
		Benchmark:                be,
		ConformanceStatusSummary: csResult,
		Checks:                   sResult,
		ControlsSeverityStatus:   controlSeverityResult,
		ResourcesSeverityStatus:  resourcesSeverityResult,
		EvaluatedAt:              utils.GetPointer(time.Unix(summaryAtTime.EvaluatedAtEpoch, 0)),
		LastJobStatus:            lastJobStatus,
	}

	return ctx.JSON(http.StatusOK, response)
}

func (h *HttpHandler) populateBenchmarkControlSummary(benchmarkMap map[string]*db.Benchmark, controlSummaryMap map[string]api.ControlSummary, benchmarkId string) (*api.BenchmarkControlSummary, error) {
	benchmark, ok := benchmarkMap[benchmarkId]
	if !ok {
		return nil, errors.New("benchmark not found")
	}

	result := api.BenchmarkControlSummary{
		Benchmark: benchmark.ToApi(),
	}

	for _, control := range benchmark.Controls {
		controlSummary, ok := controlSummaryMap[control.ID]
		if !ok {
			continue
		}
		result.Controls = append(result.Controls, controlSummary)
	}

	for _, child := range benchmark.Children {
		childResult, err := h.populateBenchmarkControlSummary(benchmarkMap, controlSummaryMap, child.ID)
		if err != nil {
			return nil, err
		}
		result.Children = append(result.Children, *childResult)
	}

	sort.Slice(result.Controls, func(i, j int) bool {
		return result.Controls[i].Control.Title < result.Controls[j].Control.Title
	})

	sort.Slice(result.Children, func(i, j int) bool {
		return result.Children[i].Benchmark.Title < result.Children[j].Benchmark.Title
	})

	return &result, nil
}

// GetBenchmarkControlsTree godoc
//
//	@Summary	Get benchmark controls
//	@Security	BearerToken
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		benchmark_id	path		string		true	"Benchmark ID"
//	@Param		connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param		connectionGroup	query		[]string	false	"Connection groups to filter by "//	@Success	200	{object}	[]api.ControlSummary
//	@Success	200				{object}	api.BenchmarkControlSummary
//	@Router		/compliance/api/v1/benchmarks/{benchmark_id}/controls [get]
func (h *HttpHandler) GetBenchmarkControlsTree(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmark_id")

	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		h.logger.Error("failed to get connection IDs", zap.Error(err))
		return err
	}

	controlsMap := make(map[string]api.Control)
	err = h.populateControlsMap(benchmarkID, controlsMap)
	if err != nil {
		return err
	}

	controlResult, evaluatedAt, err := es.BenchmarkControlSummary(h.logger, h.client, benchmarkID, connectionIDs)
	if err != nil {
		return err
	}

	queryIDs := make([]string, 0, len(controlsMap))
	for _, control := range controlsMap {
		if control.Query == nil {
			continue
		}
		queryIDs = append(queryIDs, control.Query.ID)
	}

	queries, err := h.db.GetQueriesIdAndConnector(queryIDs)
	if err != nil {
		h.logger.Error("failed to fetch queries", zap.Error(err))
		return err
	}
	queryMap := make(map[string]db.Query)
	for _, query := range queries {
		queryMap[query.ID] = query
	}

	controlSummaryMap := make(map[string]api.ControlSummary)
	for _, control := range controlsMap {
		if control.Query != nil {
			if query, ok := queryMap[control.Query.ID]; ok {
				control.Connector, _ = source.ParseType(query.Connector)
			}
		}
		result, ok := controlResult[control.ID]
		if !ok {
			result = types.ControlResult{Passed: true}
		}
		controlSummaryMap[control.ID] = api.ControlSummary{
			Control:               control,
			Passed:                result.Passed,
			FailedResourcesCount:  result.FailedResourcesCount,
			TotalResourcesCount:   result.TotalResourcesCount,
			FailedConnectionCount: result.FailedConnectionCount,
			TotalConnectionCount:  result.TotalConnectionCount,
			EvaluatedAt:           evaluatedAt,
		}
	}

	allBenchmarks, err := h.db.ListBenchmarks()
	if err != nil {
		h.logger.Error("failed to get benchmarks", zap.Error(err))
		return err
	}
	allBenchmarksMap := make(map[string]*db.Benchmark)
	for _, b := range allBenchmarks {
		b := b
		allBenchmarksMap[b.ID] = &b
	}

	benchmarkControlSummary, err := h.populateBenchmarkControlSummary(allBenchmarksMap, controlSummaryMap, benchmarkID)
	if err != nil {
		h.logger.Error("failed to populate benchmark control summary", zap.Error(err))
		return err
	}

	return ctx.JSON(http.StatusOK, benchmarkControlSummary)
}

// GetBenchmarkControl godoc
//
//	@Summary	Get benchmark controls
//	@Security	BearerToken
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		benchmark_id	path		string		true	"Benchmark ID"
//	@Param		controlId		path		string		true	"Control ID"
//	@Param		connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param		connectionGroup	query		[]string	false	"Connection groups to filter by "
//	@Success	200				{object}	api.ControlSummary
//	@Router		/compliance/api/v1/benchmarks/{benchmark_id}/controls/{controlId} [get]
func (h *HttpHandler) GetBenchmarkControl(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmark_id")
	if benchmarkID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmarkID cannot be empty")
	}
	controlID := ctx.Param("controlId")
	if controlID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "controlID cannot be empty")
	}

	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		h.logger.Error("failed to get connection IDs", zap.Error(err))
		return err
	}

	controlSummary, err := h.getControlSummary(controlID, &benchmarkID, connectionIDs)
	if err != nil {
		h.logger.Error("failed to get control summary", zap.Error(err))
		return err
	}

	return ctx.JSON(http.StatusOK, controlSummary)
}

func (h *HttpHandler) populateControlsMap(benchmarkID string, baseControlsMap map[string]api.Control) error {
	benchmark, err := h.db.GetBenchmark(benchmarkID)
	if err != nil {
		return err
	}

	if baseControlsMap == nil {
		return errors.New("baseControlsMap cannot be nil")
	}

	for _, child := range benchmark.Children {
		err := h.populateControlsMap(child.ID, baseControlsMap)
		if err != nil {
			return err
		}
	}

	missingControls := make([]string, 0)
	for _, control := range benchmark.Controls {
		if _, ok := baseControlsMap[control.ID]; !ok {
			missingControls = append(missingControls, control.ID)
		}
	}
	if len(missingControls) > 0 {
		controls, err := h.db.GetControls(missingControls)
		if err != nil {
			h.logger.Error("failed to get controls", zap.Error(err))
			return err
		}
		for _, control := range controls {
			v := control.ToApi()
			v.Connector = benchmark.Connector
			baseControlsMap[control.ID] = v
		}
	}

	return nil
}

// GetBenchmarkTrend godoc
//
//	@Summary		Get benchmark trend
//	@Description	Retrieving a trend of a benchmark result and checks.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Param			benchmark_id		path		string			true	"Benchmark ID"
//	@Param			connectionId		query		[]string		false	"Connection IDs to filter by"
//	@Param			connectionGroup		query		[]string		false	"Connection groups to filter by "
//	@Param			resourceCollection	query		[]string		false	"Resource collection IDs to filter by"
//	@Param			connector			query		[]source.Type	false	"Connector type to filter by"
//	@Param			startTime			query		int				false	"timestamp for start of the chart in epoch seconds"
//	@Param			endTime				query		int				false	"timestamp for end of the chart in epoch seconds"
//	@Success		200					{object}	[]api.BenchmarkTrendDatapoint
//	@Router			/compliance/api/v1/benchmarks/{benchmark_id}/trend [get]
func (h *HttpHandler) GetBenchmarkTrend(ctx echo.Context) error {
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	if len(connectionIDs) > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "too many connection IDs")
	}
	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
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
	_, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmark")
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

	evaluationAcrossTime, err := es.FetchBenchmarkSummaryTrend(h.logger, h.client,
		[]string{benchmarkID}, connectionIDs, resourceCollections, startTime, endTime)
	if err != nil {
		return err
	}

	controls, err := h.db.ListControlsByBenchmarkID(benchmarkID)
	if err != nil {
		h.logger.Error("failed to get controls", zap.Error(err))
		return err
	}
	controlsMap := make(map[string]*db.Control)
	for _, control := range controls {
		control := control
		controlsMap[strings.ToLower(control.ID)] = &control
	}

	var response []api.BenchmarkTrendDatapoint
	for _, datapoint := range evaluationAcrossTime[benchmarkID] {
		apiDataPoint := api.BenchmarkTrendDatapoint{
			Timestamp:                time.Unix(datapoint.DateEpoch, 0),
			ConformanceStatusSummary: api.ConformanceStatusSummary{},
			Checks:                   kaytuTypes.SeverityResult{},
			ControlsSeverityStatus:   api.BenchmarkControlsSeverityStatus{},
		}
		apiDataPoint.ConformanceStatusSummary.AddESConformanceStatusMap(datapoint.QueryResult)
		apiDataPoint.Checks.AddResultMap(datapoint.SeverityResult)
		for controlId, controlResult := range datapoint.Controls {
			control := controlsMap[strings.ToLower(controlId)]
			apiDataPoint.ControlsSeverityStatus = addToControlSeverityResult(apiDataPoint.ControlsSeverityStatus, control, controlResult)
		}

		response = append(response, apiDataPoint)
	}

	sort.Slice(response, func(i, j int) bool {
		return response[i].Timestamp.Before(response[j].Timestamp)
	})

	return ctx.JSON(http.StatusOK, response)
}

// ListControlsSummary godoc
//
//	@Summary	List controls summaries
//	@Security	BearerToken
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		controlId		query		[]string	false	"Control IDs to filter by"
//	@Param		connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param		connectionGroup	query		[]string	false	"Connection groups to filter by "
//	@Success	200				{object}	[]api.ControlSummary
//	@Router		/compliance/api/v1/controls/summary [get]
func (h *HttpHandler) ListControlsSummary(ctx echo.Context) error {
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		h.logger.Error("failed to get connection IDs", zap.Error(err))
		return err
	}

	controlIds := httpserver2.QueryArrayParam(ctx, "controlId")
	controls, err := h.db.GetControls(controlIds)
	if err != nil {
		h.logger.Error("failed to fetch controls", zap.Error(err))
		return err
	}
	controlIds = make([]string, 0, len(controls))
	for _, control := range controls {
		controlIds = append(controlIds, control.ID)
	}

	benchmarks, err := h.db.ListDistinctRootBenchmarksFromControlIds(controlIds)
	if err != nil {
		h.logger.Error("failed to fetch benchmarks", zap.Error(err))
		return err
	}
	benchmarkIds := make([]string, 0, len(benchmarks))
	for _, benchmark := range benchmarks {
		benchmarkIds = append(benchmarkIds, benchmark.ID)
	}

	controlResults, evaluatedAts, err := es.BenchmarksControlSummary(h.logger, h.client, benchmarkIds, connectionIDs)
	if err != nil {
		h.logger.Error("failed to fetch control results", zap.Error(err))
		return err
	}

	resourceTypes, err := h.inventoryClient.ListResourceTypesMetadata(httpclient.FromEchoContext(ctx),
		nil, nil, nil, false, nil, 10000, 1)
	if err != nil {
		h.logger.Error("failed to get resource types metadata", zap.Error(err))
		return err
	}
	resourceTypeMap := make(map[string]*inventoryApi.ResourceType)
	for _, rt := range resourceTypes.ResourceTypes {
		rt := rt
		resourceTypeMap[strings.ToLower(rt.ResourceType)] = &rt
	}

	results := make([]api.ControlSummary, 0, len(controls))
	for _, control := range controls {
		apiControl := control.ToApi()
		var resourceType *inventoryApi.ResourceType
		if control.QueryID != nil {
			query, err := h.db.GetQuery(*control.QueryID)
			if err != nil {
				h.logger.Error("failed to fetch query", zap.Error(err), zap.String("queryID", *control.QueryID), zap.String("controlID", control.ID))
				return err
			}
			apiControl.Connector, _ = source.ParseType(query.Connector)
			apiControl.Query = utils.GetPointer(query.ToApi())
			if query.PrimaryTable != nil {
				rtName := runner.GetResourceTypeFromTableName(*query.PrimaryTable, source.Type(query.Connector))
				resourceType = resourceTypeMap[strings.ToLower(rtName)]
			}
		}

		result, ok := controlResults[control.ID]
		if !ok {
			result = types.ControlResult{Passed: true}
		}
		evaluatedAt, ok := evaluatedAts[control.ID]
		if !ok {
			evaluatedAt = -1
		}

		controlSummary := api.ControlSummary{
			Control:               apiControl,
			ResourceType:          resourceType,
			Passed:                result.Passed,
			FailedResourcesCount:  result.FailedResourcesCount,
			TotalResourcesCount:   result.TotalResourcesCount,
			FailedConnectionCount: result.FailedConnectionCount,
			TotalConnectionCount:  result.TotalConnectionCount,
			EvaluatedAt:           evaluatedAt,
		}
		results = append(results, controlSummary)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].EvaluatedAt == -1 {
			return false
		}
		if results[j].EvaluatedAt == -1 {
			return true
		}
		return results[i].FailedResourcesCount > results[j].FailedResourcesCount
	})

	return ctx.JSON(http.StatusOK, results)
}

// GetControlSummary godoc
//
//	@Summary	Get control summary
//	@Security	BearerToken
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		controlId		path		string		true	"Control ID"
//	@Param		connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param		connectionGroup	query		[]string	false	"Connection groups to filter by "
//	@Success	200				{object}	api.ControlSummary
//	@Router		/compliance/api/v1/controls/{controlId}/summary [get]
func (h *HttpHandler) GetControlSummary(ctx echo.Context) error {
	controlID := ctx.Param("controlId")
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	controlSummary, err := h.getControlSummary(controlID, nil, connectionIDs)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, controlSummary)
}

func (h *HttpHandler) getControlSummary(controlID string, benchmarkID *string, connectionIDs []string) (*api.ControlSummary, error) {
	control, err := h.db.GetControl(controlID)
	if err != nil {
		h.logger.Error("failed to fetch control", zap.Error(err), zap.String("controlID", controlID), zap.Stringp("benchmarkID", benchmarkID))
		return nil, err
	}
	if control == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("control %s not found", controlID))
	}
	apiControl := control.ToApi()
	if benchmarkID != nil {
		benchmark, err := h.db.GetBenchmarkBare(*benchmarkID)
		if err != nil {
			h.logger.Error("failed to fetch benchmark", zap.Error(err), zap.Stringp("benchmarkID", benchmarkID))
			return nil, err
		}
		apiControl.Connector = benchmark.Connector
	}

	resourceTypes, err := h.inventoryClient.ListResourceTypesMetadata(&httpclient.Context{UserRole: authApi.InternalRole},
		nil, nil, nil, false, nil, 10000, 1)
	if err != nil {
		h.logger.Error("failed to get resource types metadata", zap.Error(err))
		return nil, err
	}
	resourceTypeMap := make(map[string]*inventoryApi.ResourceType)
	for _, rt := range resourceTypes.ResourceTypes {
		rt := rt
		resourceTypeMap[strings.ToLower(rt.ResourceType)] = &rt
	}

	var resourceType *inventoryApi.ResourceType
	if control.QueryID != nil {
		query, err := h.db.GetQuery(*control.QueryID)
		if err != nil {
			h.logger.Error("failed to fetch query", zap.Error(err), zap.String("queryID", *control.QueryID), zap.Stringp("benchmarkID", benchmarkID))
			return nil, err
		}
		apiControl.Connector, _ = source.ParseType(query.Connector)
		apiControl.Query = utils.GetPointer(query.ToApi())
		if query.PrimaryTable != nil {
			rtName := runner.GetResourceTypeFromTableName(*query.PrimaryTable, source.Type(query.Connector))
			resourceType = resourceTypeMap[strings.ToLower(rtName)]
		}
	}

	benchmarks, err := h.db.ListDistinctRootBenchmarksFromControlIds([]string{controlID})
	if err != nil {
		h.logger.Error("failed to fetch benchmarks", zap.Error(err), zap.String("controlID", controlID))
		return nil, err
	}
	benchmarkIds := make([]string, 0, len(benchmarks))
	apiBenchmarks := make([]api.Benchmark, 0, len(benchmarks))
	for _, benchmark := range benchmarks {
		benchmarkIds = append(benchmarkIds, benchmark.ID)
		apiBenchmarks = append(apiBenchmarks, benchmark.ToApi())
	}

	var evaluatedAt int64
	var result types.ControlResult
	if benchmarkID != nil {
		controlResult, evAt, err := es.BenchmarkControlSummary(h.logger, h.client, *benchmarkID, connectionIDs)
		if err != nil {
			h.logger.Error("failed to fetch control result", zap.Error(err), zap.String("controlID", controlID), zap.Stringp("benchmarkID", benchmarkID))
			return nil, err
		}
		var ok bool
		result, ok = controlResult[control.ID]
		if !ok {
			result = types.ControlResult{Passed: true}
		}
		evaluatedAt = evAt
	} else {
		controlResult, evaluatedAts, err := es.BenchmarksControlSummary(h.logger, h.client, benchmarkIds, connectionIDs)
		if err != nil {
			h.logger.Error("failed to fetch control result", zap.Error(err), zap.String("controlID", controlID), zap.Stringp("benchmarkID", benchmarkID))
		}
		var ok bool
		result, ok = controlResult[control.ID]
		if !ok {
			result = types.ControlResult{Passed: true}
		}
		evaluatedAt, ok = evaluatedAts[control.ID]
		if !ok {
			evaluatedAt = -1
		}
	}

	controlSummary := api.ControlSummary{
		Control:               apiControl,
		ResourceType:          resourceType,
		Benchmarks:            apiBenchmarks,
		Passed:                result.Passed,
		FailedResourcesCount:  result.FailedResourcesCount,
		TotalResourcesCount:   result.TotalResourcesCount,
		FailedConnectionCount: result.FailedConnectionCount,
		TotalConnectionCount:  result.TotalConnectionCount,
		EvaluatedAt:           evaluatedAt,
	}

	return &controlSummary, nil
}

// GetControlTrend godoc
//
//	@Summary	Get control trend
//	@Security	BearerToken
//	@Tags		compliance
//	@Accept		json
//	@Produce	json
//	@Param		controlId		path		string		true	"Control ID"
//	@Param		connectionId	query		[]string	false	"Connection IDs to filter by"
//	@Param		connectionGroup	query		[]string	false	"Connection groups to filter by "
//	@Param		startTime		query		int			false	"timestamp for start of the chart in epoch seconds"
//	@Param		endTime			query		int			false	"timestamp for end of the chart in epoch seconds"
//	@Param		granularity		query		string		false	"granularity of the chart"	Enums(daily,monthly)	Default(daily)
//	@Success	200				{object}	[]api.ControlTrendDatapoint
//	@Router		/compliance/api/v1/controls/{controlId}/trend [get]
func (h *HttpHandler) GetControlTrend(ctx echo.Context) error {
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	if len(connectionIDs) > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "too many connection IDs")
	}
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

	controlID := ctx.Param("controlId")
	control, err := h.db.GetControl(controlID)
	if err != nil {
		h.logger.Error("failed to fetch control", zap.Error(err), zap.String("controlID", controlID))
		return err
	}
	benchmarks, err := h.db.ListDistinctRootBenchmarksFromControlIds([]string{controlID})
	if err != nil {
		h.logger.Error("failed to fetch benchmarks", zap.Error(err), zap.String("controlID", controlID))
		return err
	}
	benchmarkIds := make([]string, 0, len(benchmarks))
	for _, benchmark := range benchmarks {
		benchmarkIds = append(benchmarkIds, benchmark.ID)
	}

	stepDuration := 24 * time.Hour
	if granularity := ctx.QueryParam("granularity"); granularity == "monthly" {
		stepDuration = 30 * 24 * time.Hour
	}

	dataPoints, err := es.FetchBenchmarkSummaryTrendByConnectionIDPerControl(h.logger, h.client,
		benchmarkIds, []string{controlID}, connectionIDs, startTime, endTime, stepDuration)
	if err != nil {
		h.logger.Error("failed to fetch control result", zap.Error(err), zap.String("controlID", controlID))
		return err
	}

	var response []api.ControlTrendDatapoint
	for _, datapoint := range dataPoints[control.ID] {
		response = append(response, api.ControlTrendDatapoint{
			Timestamp:             int(datapoint.DateEpoch),
			FailedResourcesCount:  datapoint.FailedResourcesCount,
			TotalResourcesCount:   datapoint.TotalResourcesCount,
			FailedConnectionCount: datapoint.FailedConnectionCount,
			TotalConnectionCount:  datapoint.TotalConnectionCount,
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
//	@Param			benchmark_id		path		string		true	"Benchmark ID"
//	@Param			auto_assign			query		bool		false	"Auto enable benchmark for connections"
//	@Param			connectionId		query		[]string	false	"Connection ID or 'all' for everything"
//	@Param			connectionGroup		query		[]string	false	"Connection group"
//	@Param			resourceCollection	query		[]string	false	"Resource collection"
//	@Success		200					{object}	[]api.BenchmarkAssignment
//	@Router			/compliance/api/v1/assignments/{benchmark_id}/connection [post]
func (h *HttpHandler) CreateBenchmarkAssignment(ctx echo.Context) error {
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}

	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
	if len(connectionIDs) > 0 && len(resourceCollections) > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot specify both connection and resource collection")
	}

	autoAssignStr := ctx.QueryParam("auto_assign")

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

	//connectorType := source.Nil
	// trace :
	//outputS2, span2 := tracer.Start(outputS1, "new_GetQuery(loop)", trace.WithSpanKind(trace.SpanKindServer))
	//span2.SetName("new_GetQuery(loop)")

	ca := benchmark.ToApi()
	switch {
	case len(autoAssignStr) > 0:
		autoAssign, err := strconv.ParseBool(autoAssignStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid auto_enable value")
		}
		err = h.db.SetBenchmarkAutoAssign(benchmarkId, autoAssign)
		if err != nil {
			h.logger.Error("failed to set auto assign", zap.Error(err))
			return err
		}
		return ctx.JSON(http.StatusOK, []api.BenchmarkAssignment{})
	case len(connectionIDs) > 0:
		connections := make([]onboardApi.Connection, 0)
		if len(connectionIDs) == 1 && strings.ToLower(connectionIDs[0]) == "all" {
			srcs, err := h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), ca.Connectors)
			if err != nil {
				return err
			}
			for _, src := range srcs {
				if src.IsEnabled() {
					connections = append(connections, src)
				}
			}
		} else {
			connections, err = h.onboardClient.GetSources(httpclient.FromEchoContext(ctx), connectionIDs)
			if err != nil {
				return err
			}
		}

		result := make([]api.BenchmarkAssignment, 0, len(connections))
		// trace :
		output4, span4 := tracer.Start(outputS1, "new_AddBenchmarkAssignment(loop)", trace.WithSpanKind(trace.SpanKindServer))
		span4.SetName("new_AddBenchmarkAssignment(loop)")

		for _, src := range connections {
			assignment := &db.BenchmarkAssignment{
				BenchmarkId:  benchmarkId,
				ConnectionId: utils.GetPointer(src.ID.String()),
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

			for _, connectionId := range connectionIDs {
				result = append(result, api.BenchmarkAssignment{
					BenchmarkId:  benchmarkId,
					ConnectionId: utils.GetPointer(connectionId),
					AssignedAt:   assignment.AssignedAt,
				})
			}
		}
		span4.End()
		return ctx.JSON(http.StatusOK, result)
	case len(resourceCollections) > 0:
		result := make([]api.BenchmarkAssignment, 0, len(resourceCollections))

		for _, resourceCollection := range resourceCollections {
			resourceCollection := resourceCollection
			assignment := &db.BenchmarkAssignment{
				BenchmarkId:        benchmarkId,
				ResourceCollection: &resourceCollection,
				AssignedAt:         time.Now(),
			}
			// trace :
			_, span6 := tracer.Start(outputS1, "new_AddBenchmarkAssignment", trace.WithSpanKind(trace.SpanKindServer))
			span6.SetName("new_AddBenchmarkAssignment")

			if err := h.db.AddBenchmarkAssignment(assignment); err != nil {
				span6.RecordError(err)
				span6.SetStatus(codes.Error, err.Error())
				ctx.Logger().Errorf("add benchmark assignment: %v", err)
				return err
			}
			span6.SetAttributes(
				attribute.String("benchmark ID", assignment.BenchmarkId),
			)
			span6.End()

			result = append(result, api.BenchmarkAssignment{
				BenchmarkId:          benchmarkId,
				ResourceCollectionId: assignment.ResourceCollection,
				AssignedAt:           assignment.AssignedAt,
			})
		}
		return ctx.JSON(http.StatusOK, result)
	}
	return echo.NewHTTPError(http.StatusBadRequest, "auto assign, connection or resource collection is required")
}

// ListAssignmentsByConnection godoc
//
//	@Summary		Get list of benchmark assignments for a connection
//	@Description	Retrieving all benchmark assigned to a connection with connection id
//	@Security		BearerToken
//	@Tags			benchmarks_assignment
//	@Accept			json
//	@Produce		json
//	@Param			connection_id	path		string	true	"Connection ID"
//	@Success		200				{object}	[]api.AssignedBenchmark
//	@Router			/compliance/api/v1/assignments/connection/{connection_id} [get]
func (h *HttpHandler) ListAssignmentsByConnection(ctx echo.Context) error {
	connectionId := ctx.Param("connection_id")
	if connectionId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "connection id is empty")
	}

	if err := httpserver2.CheckAccessToConnectionID(ctx, connectionId); err != nil {
		return err
	}

	outputS2, span2 := tracer.Start(ctx.Request().Context(), "new_GetBenchmarkAssignmentsBySourceId(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_GetBenchmarkAssignmentsBySourceId(loop)")

	_, span1 := tracer.Start(outputS2, "new_GetBenchmarkAssignmentsBySourceId", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmarkAssignmentsBySourceId")

	dbAssignments, err := h.db.GetBenchmarkAssignmentsByConnectionId(connectionId)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark assignments for %s not found", connectionId))
		}
		ctx.Logger().Errorf("find benchmark assignments by source %s: %v", connectionId, err)
		return err
	}

	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("connection ID", connectionId),
	))
	span1.End()

	benchmarks, err := h.db.ListRootBenchmarks(nil)
	if err != nil {
		return err
	}

	src, err := h.onboardClient.GetSource(httpclient.FromEchoContext(ctx), connectionId)
	if err != nil {
		return err
	}

	result := make([]api.AssignedBenchmark, 0, len(dbAssignments))
	for _, benchmark := range benchmarks {
		apiBenchmark := benchmark.ToApi()
		if !utils.Includes(apiBenchmark.Connectors, src.Connector) {
			continue
		}
		res := api.AssignedBenchmark{
			Benchmark: benchmark.ToApi(),
			Status:    false,
		}
		if benchmark.AutoAssign && src.IsEnabled() {
			res.Status = true
		} else {
			for _, assignment := range dbAssignments {
				if assignment.ConnectionId != nil && *assignment.ConnectionId == src.ID.String() && assignment.BenchmarkId == benchmark.ID {
					res.Status = true
					break
				}
			}
		}
		result = append(result, res)
	}

	return ctx.JSON(http.StatusOK, result)
}

// ListAssignmentsByResourceCollection godoc
//
//	@Summary		Get list of benchmark assignments for a resource collection
//	@Description	Retrieving all benchmark assigned to a resource collection with resource collection id
//	@Security		BearerToken
//	@Tags			benchmarks_assignment
//	@Accept			json
//	@Produce		json
//	@Param			resource_collection_id	path		string	true	"Resource collection ID"
//	@Success		200						{object}	[]api.AssignedBenchmark
//	@Router			/compliance/api/v1/assignments/resource_collection/{resource_collection_id} [get]
func (h *HttpHandler) ListAssignmentsByResourceCollection(ctx echo.Context) error {
	resourceCollectionId := ctx.Param("resource_collection_id")
	if resourceCollectionId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "connection id is empty")
	}

	outputS2, span2 := tracer.Start(ctx.Request().Context(), "new_GetBenchmarkAssignmentsBySourceId(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_GetBenchmarkAssignmentsBySourceId(loop)")

	_, span1 := tracer.Start(outputS2, "new_GetBenchmarkAssignmentsBySourceId", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmarkAssignmentsBySourceId")

	dbAssignments, err := h.db.GetBenchmarkAssignmentsByResourceCollectionId(resourceCollectionId)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark assignments for %s not found", resourceCollectionId))
		}
		h.logger.Error("find benchmark assignments by resource collection", zap.Error(err), zap.String("resourceCollectionId", resourceCollectionId))
		return err
	}

	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("connection ID", resourceCollectionId),
	))
	span1.End()

	benchmarks, err := h.db.ListRootBenchmarks(nil)
	if err != nil {
		return err
	}

	result := make([]api.AssignedBenchmark, 0, len(dbAssignments))
	for _, benchmark := range benchmarks {
		res := api.AssignedBenchmark{
			Benchmark: benchmark.ToApi(),
			Status:    false,
		}
		for _, assignment := range dbAssignments {
			if assignment.ResourceCollection != nil && *assignment.ResourceCollection == resourceCollectionId && assignment.BenchmarkId == benchmark.ID {
				res.Status = true
				break
			}
		}
		result = append(result, res)
	}

	return ctx.JSON(http.StatusOK, result)
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
//	@Success		200				{object}	api.BenchmarkAssignedEntities
//	@Router			/compliance/api/v1/assignments/benchmark/{benchmark_id} [get]
func (h *HttpHandler) ListAssignmentsByBenchmark(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmark_id")
	if benchmarkId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmark id is empty")
	}
	// trace :
	outputS, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetBenchmark")

	benchmark, err := h.db.GetBenchmarkBare(benchmarkId)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("benchmark ID", benchmark.ID),
	))
	span1.End()

	hctx := httpclient.FromEchoContext(ctx)

	var assignedConnections []api.BenchmarkAssignedConnection

	connections, err := h.onboardClient.ListSources(hctx, []source.Type{benchmark.Connector})
	if err != nil {
		return err
	}

	for _, connection := range connections {
		if !connection.IsEnabled() {
			continue
		}
		ba := api.BenchmarkAssignedConnection{
			ConnectionID:           connection.ID.String(),
			ProviderConnectionID:   connection.ConnectionID,
			ProviderConnectionName: connection.ConnectionName,
			Connector:              benchmark.Connector,
			Status:                 false,
		}
		assignedConnections = append(assignedConnections, ba)
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

	if benchmark.AutoAssign {
		for idx, r := range assignedConnections {
			r.Status = true
			assignedConnections[idx] = r
		}
	}

	for _, assignment := range dbAssignments {
		if assignment.ConnectionId != nil && !benchmark.AutoAssign {
			for idx, r := range assignedConnections {
				if r.ConnectionID == *assignment.ConnectionId {
					r.Status = true
					assignedConnections[idx] = r
				}
			}
		}
	}

	resp := api.BenchmarkAssignedEntities{}

	for _, item := range assignedConnections {
		if httpserver2.CheckAccessToConnectionID(ctx, item.ConnectionID) != nil {
			continue
		}
		resp.Connections = append(resp.Connections, item)
	}

	for idx, conn := range resp.Connections {
		conn.ProviderConnectionID = demo.EncodeResponseData(ctx, conn.ProviderConnectionID)
		conn.ProviderConnectionName = demo.EncodeResponseData(ctx, conn.ProviderConnectionName)
		resp.Connections[idx] = conn
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
//	@Param			benchmark_id		path	string		true	"Benchmark ID"
//	@Param			connectionId		query	[]string	false	"Connection ID or 'all' for everything"
//	@Param			connectionGroup		query	[]string	false	"Connection Group "
//	@Param			resourceCollection	query	[]string	false	"Resource Collection"
//	@Success		200
//	@Router			/compliance/api/v1/assignments/{benchmark_id}/connection [delete]
func (h *HttpHandler) DeleteBenchmarkAssignment(ctx echo.Context) error {
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
	if len(connectionIDs) > 0 && len(resourceCollections) > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot specify both connection and resource collection")
	}

	benchmarkId := ctx.Param("benchmark_id")
	if benchmarkId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "benchmark id is empty")
	}

	switch {
	case len(connectionIDs) > 0:
		if len(connectionIDs) == 1 && strings.ToLower(connectionIDs[0]) == "all" {
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
			// tracer :
			outputS5, span5 := tracer.Start(ctx.Request().Context(), "new_GetBenchmarkAssignmentByIds(loop)", trace.WithSpanKind(trace.SpanKindServer))
			span5.SetName("new_GetBenchmarkAssignmentByIds(loop)")

			for _, connectionId := range connectionIDs {
				// trace :
				outputS3, span3 := tracer.Start(outputS5, "new_GetBenchmarkAssignmentByIds", trace.WithSpanKind(trace.SpanKindServer))
				span3.SetName("new_GetBenchmarkAssignmentByIds")

				if _, err := h.db.GetBenchmarkAssignmentByIds(benchmarkId, utils.GetPointer(connectionId), nil); err != nil {
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

				if err := h.db.DeleteBenchmarkAssignmentByIds(benchmarkId, utils.GetPointer(connectionId), nil); err != nil {
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
	case len(resourceCollections) > 0:
		// tracer :
		outputS6, span6 := tracer.Start(ctx.Request().Context(), "new_GetBenchmarkAssignmentByIds(loop)", trace.WithSpanKind(trace.SpanKindServer))
		span6.SetName("new_GetBenchmarkAssignmentByIds(loop)")

		for _, resourceCollection := range resourceCollections {
			// trace :
			resourceCollection := resourceCollection
			outputS4, span4 := tracer.Start(outputS6, "new_GetBenchmarkAssignmentByIds", trace.WithSpanKind(trace.SpanKindServer))
			span4.SetName("new_GetBenchmarkAssignmentByIds")

			if _, err := h.db.GetBenchmarkAssignmentByIds(benchmarkId, nil, &resourceCollection); err != nil {
				span4.RecordError(err)
				span4.SetStatus(codes.Error, err.Error())
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return echo.NewHTTPError(http.StatusFound, "benchmark assignment not found")
				}
				ctx.Logger().Errorf("find benchmark assignment: %v", err)
				return err
			}
			span4.AddEvent("information", trace.WithAttributes(
				attribute.String("benchmark ID", benchmarkId),
			))
			span4.End()

			// trace :
			_, span5 := tracer.Start(outputS4, "new_DeleteBenchmarkAssignmentByIds", trace.WithSpanKind(trace.SpanKindServer))
			span5.SetName("new_DeleteBenchmarkAssignmentByIds")

			if err := h.db.DeleteBenchmarkAssignmentByIds(benchmarkId, nil, &resourceCollection); err != nil {
				span5.RecordError(err)
				span5.SetStatus(codes.Error, err.Error())
				ctx.Logger().Errorf("delete benchmark assignment: %v", err)
				return err
			}
			span5.AddEvent("information", trace.WithAttributes(
				attribute.String("benchmark ID", benchmarkId),
			))
			span5.End()
		}
		span6.End()
		return ctx.NoContent(http.StatusOK)
	}
	return echo.NewHTTPError(http.StatusBadRequest, "connection or resource collection is required")
}

func (h *HttpHandler) ListBenchmarks(ctx echo.Context) error {
	var response []api.Benchmark
	// trace :
	output1, span1 := tracer.Start(ctx.Request().Context(), "new_ListRootBenchmarks", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListRootBenchmarks")

	benchmarks, err := h.db.ListRootBenchmarks(nil)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	// tracer :
	_, span2 := tracer.Start(output1, "new_PopulateConnectors(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_PopulateConnectors(loop)")

	for _, b := range benchmarks {
		response = append(response, b.ToApi())
	}
	span2.End()

	return ctx.JSON(http.StatusOK, response)
}

func (h *HttpHandler) ListAllBenchmarks(ctx echo.Context) error {
	isBare := true
	if bare := ctx.QueryParam("bare"); bare != "" {
		var err error
		isBare, err = strconv.ParseBool(bare)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid bare value")
		}
	}

	var response []api.Benchmark
	// trace :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_ListRootBenchmarks", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListBenchmarks")
	var benchmarks []db.Benchmark
	var err error
	if isBare {
		benchmarks, err = h.db.ListBenchmarksBare()
	} else {
		benchmarks, err = h.db.ListBenchmarks()
	}
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	for _, b := range benchmarks {
		response = append(response, b.ToApi())
	}

	return ctx.JSON(http.StatusOK, response)
}

func (h *HttpHandler) GetBenchmark(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmark_id")
	// trace :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_GetBenchmark", trace.WithSpanKind(trace.SpanKindServer))
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

	return ctx.JSON(http.StatusOK, benchmark.ToApi())
}

func (h *HttpHandler) getBenchmarkControls(ctx context.Context, benchmarkID string) ([]db.Control, error) {
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

	var controlIDs []string
	for _, p := range b.Controls {
		controlIDs = append(controlIDs, p.ID)
	}
	//trace :
	_, span2 := tracer.Start(outputS, "new_GetControls", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_GetControls")

	controls, err := h.db.GetControls(controlIDs)
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span2.End()

	//tracer :
	output3, span3 := tracer.Start(outputS, "new_getBenchmarkControls(loop)", trace.WithSpanKind(trace.SpanKindServer))
	span3.SetName("new_getBenchmarkControls(loop)")

	for _, child := range b.Children {
		// tracer :
		_, span4 := tracer.Start(output3, "new_getBenchmarkControls", trace.WithSpanKind(trace.SpanKindServer))
		span4.SetName("new_getBenchmarkControls")

		childControls, err := h.getBenchmarkControls(ctx, child.ID)
		if err != nil {
			span4.RecordError(err)
			span4.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		span4.SetAttributes(
			attribute.String("benchmark ID", child.ID),
		)
		span4.End()

		controls = append(controls, childControls...)
	}
	span3.End()

	return controls, nil
}

func (h *HttpHandler) GetControl(ctx echo.Context) error {
	controlId := ctx.Param("control_id")
	// trace :
	outputS, span1 := tracer.Start(ctx.Request().Context(), "new_GetControl", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetControl")

	control, err := h.db.GetControl(controlId)
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("control ID", controlId),
	))
	span1.End()

	if control == nil {
		return echo.NewHTTPError(http.StatusNotFound, "control not found")
	}

	pa := control.ToApi()
	// trace :
	outputS2, span2 := tracer.Start(outputS, "new_PopulateConnector", trace.WithSpanKind(trace.SpanKindServer))
	span2.SetName("new_PopulateConnector")

	err = control.PopulateConnector(outputS2, h.db, &pa)
	if err != nil {
		span2.RecordError(err)
		span2.SetStatus(codes.Error, err.Error())
		return err
	}
	span2.End()
	return ctx.JSON(http.StatusOK, pa)
}

func (h *HttpHandler) ListControls(ctx echo.Context) error {
	controls, err := h.db.ListControls()
	if err != nil {
		return err
	}

	var resp []api.Control
	for _, control := range controls {
		pa := control.ToApi()
		resp = append(resp, pa)
	}
	return ctx.JSON(http.StatusOK, resp)
}

func (h *HttpHandler) ListQueries(ctx echo.Context) error {
	queries, err := h.db.ListQueries()
	if err != nil {
		return err
	}

	var resp []api.Query
	for _, query := range queries {
		pa := query.ToApi()
		resp = append(resp, pa)
	}
	return ctx.JSON(http.StatusOK, resp)
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
//	@Param			configzGitURL	query	string	false	"Git URL"
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/compliance/api/v1/queries/sync [get]
func (h *HttpHandler) SyncQueries(ctx echo.Context) error {
	enabled, err := h.metadataClient.GetConfigMetadata(httpclient.FromEchoContext(ctx), models.MetadataKeyCustomizationEnabled)
	if err != nil {
		h.logger.Error("get config metadata", zap.Error(err))
		return err
	}

	if !enabled.GetValue().(bool) {
		return echo.NewHTTPError(http.StatusForbidden, "customization is not allowed")
	}

	configzGitURL := ctx.QueryParam("configzGitURL")
	if configzGitURL != "" {
		// validate url
		_, err := url.ParseRequestURI(configzGitURL)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid url")
		}

		err = h.metadataClient.SetConfigMetadata(httpclient.FromEchoContext(ctx), models.MetadataKeyAnalyticsGitURL, configzGitURL)
		if err != nil {
			h.logger.Error("set config metadata", zap.Error(err))
			return err
		}
	}

	currentNamespace, ok := os.LookupEnv("CURRENT_NAMESPACE")
	if !ok {
		return errors.New("current namespace lookup failed")
	}

	var migratorJob v1.Job
	err = h.kubeClient.Get(context.Background(), k8sclient.ObjectKey{
		Namespace: currentNamespace,
		Name:      "migrator-job",
	}, &migratorJob)
	if err != nil {
		return err
	}

	err = h.kubeClient.Delete(context.Background(), &migratorJob)
	if err != nil {
		return err
	}

	for {
		err = h.kubeClient.Get(context.Background(), k8sclient.ObjectKey{
			Namespace: currentNamespace,
			Name:      "migrator-job",
		}, &migratorJob)
		if err != nil {
			if k8sclient.IgnoreNotFound(err) == nil {
				break
			}
			return err
		}

		time.Sleep(1 * time.Second)
	}

	migratorJob.ObjectMeta = metav1.ObjectMeta{
		Name:      "migrator-job",
		Namespace: currentNamespace,
		Annotations: map[string]string{
			"helm.sh/hook":        "post-install,post-upgrade",
			"helm.sh/hook-weight": "0",
		},
	}
	migratorJob.Spec.Selector = nil
	migratorJob.Spec.Template.ObjectMeta = metav1.ObjectMeta{}
	migratorJob.Status = v1.JobStatus{}

	err = h.kubeClient.Create(context.Background(), &migratorJob)
	if err != nil {
		return err
	}

	//err := h.syncJobsQueue.Publish([]byte{})
	//if err != nil {
	//	h.logger.Error("publish sync jobs", zap.Error(err))
	//	return err
	//}
	return ctx.JSON(http.StatusOK, struct{}{})
}

// ListComplianceTags godoc
//
//	@Summary		List compliance tag keys
//	@Description	Retrieving a list of compliance tag keys with their possible values.
//	@Security		BearerToken
//	@Tags			compliance
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]string
//	@Router			/compliance/api/v1/metadata/tag/compliance [get]
func (h *HttpHandler) ListComplianceTags(ctx echo.Context) error {
	// trace :
	_, span1 := tracer.Start(ctx.Request().Context(), "new_ListComplianceTagKeysWithPossibleValues", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_ListComplianceTagKeysWithPossibleValues")

	tags, err := h.db.ListComplianceTagKeysWithPossibleValues()
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	tags = model.TrimPrivateTags(tags)
	return ctx.JSON(http.StatusOK, tags)
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
	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
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
//	@Param			tag					query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			connector			query		[]source.Type	false	"filter insights by connector"
//	@Param			connectionId		query		[]string		false	"filter the result by source id"
//	@Param			connectionGroup		query		[]string		false	"filter the result by connection group "
//	@Param			resourceCollection	query		[]string		false	"Resource collection IDs to filter by"
//	@Param			startTime			query		int				false	"unix seconds for the start time of the trend"
//	@Param			endTime				query		int				false	"unix seconds for the end time of the trend"
//	@Success		200					{object}	[]api.Insight
//	@Router			/compliance/api/v1/insight [get]
func (h *HttpHandler) ListInsights(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver2.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
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

	insightIdToResults, err := h.inventoryClient.ListInsightResults(httpclient.FromEchoContext(ctx), connectors, connectionIDs, resourceCollections, insightIDsList, &endTime)
	if err != nil {
		return err
	}

	oldInsightIdToResults, err := h.inventoryClient.ListInsightResults(httpclient.FromEchoContext(ctx), connectors, connectionIDs, resourceCollections, insightIDsList, &startTime)
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

func (h *HttpHandler) getInsightApiRes(ctx echo.Context, insightRow *db.Insight, connectionIDs, resourceCollections []string, startTime, endTime time.Time) (*api.Insight, error) {
	insightResults, err := h.inventoryClient.GetInsightResult(httpclient.FromEchoContext(ctx), connectionIDs, resourceCollections, insightRow.ID, &endTime)
	if err != nil {
		return nil, err
	}

	oldInsightResults, err := h.inventoryClient.GetInsightResult(httpclient.FromEchoContext(ctx), connectionIDs, resourceCollections, insightRow.ID, &startTime)
	if err != nil {
		h.logger.Warn("failed to get old insight results", zap.Error(err))
		oldInsightResults = make([]insight.InsightResource, 0)
	}

	connections, err := h.onboardClient.ListSources(httpclient.FromEchoContext(ctx), []source.Type{insightRow.Connector})
	if err != nil {
		return nil, err
	}
	connectionToNameMap := make(map[string]string)
	for _, connection := range connections {
		connectionToNameMap[connection.ID.String()] = connection.ConnectionName
	}

	enabledConnections := make(map[string]bool)
	for _, connectionID := range connectionIDs {
		enabledConnections[connectionID] = true
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
			return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		objectBuffer, err := io.ReadAll(getObjectOutput.Body)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		var steampipeResults steampipe.Result
		err = json.Unmarshal(objectBuffer, &steampipeResults)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		// Add account name
		steampipeResults.Headers = append(steampipeResults.Headers, "account_name")
		for colIdx, header := range steampipeResults.Headers {
			if strings.ToLower(header) != "kaytu_account_id" {
				continue
			}
			for rowIdx, row := range steampipeResults.Data {
				if len(row) <= colIdx {
					continue
				}
				if row[colIdx] == nil {
					continue
				}
				if accountID, ok := row[colIdx].(string); ok {
					if accountName, ok := connectionToNameMap[accountID]; ok {
						steampipeResults.Data[rowIdx] = append(steampipeResults.Data[rowIdx], accountName)
					} else {
						steampipeResults.Data[rowIdx] = append(steampipeResults.Data[rowIdx], "null")
					}
				}
			}
		}

		steampipeFilteredResults := steampipeResults
		if len(connectionIDs) > 0 {
			steampipeFilteredResults.Data = make([][]any, 0)
			for colIdx, header := range steampipeResults.Headers {
				if strings.ToLower(header) != "kaytu_account_id" {
					continue
				}
				for _, row := range steampipeResults.Data {
					if len(row) <= colIdx {
						continue
					}
					if row[colIdx] == nil {
						continue
					}
					if accountID, ok := row[colIdx].(string); ok {
						if _, ok := enabledConnections[accountID]; ok {
							localRow := row
							steampipeFilteredResults.Data = append(steampipeFilteredResults.Data, localRow)
						}
					}
				}
			}
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
				Headers: steampipeFilteredResults.Headers,
				Rows:    steampipeFilteredResults.Data,
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

	return &apiRes, nil
}

// GetInsight godoc
//
//	@Summary		Get insight
//	@Description	Retrieving the specified insight with ID. Provides details of the insight, including results during the specified time period for the specified connection.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			insightId			path		string		true	"Insight ID"
//	@Param			connectionId		query		[]string	false	"filter the result by source id"
//	@param			connectionGroup		query		[]string	false	"filter the result by connection group"
//	@Param			resourceCollection	query		[]string	false	"Resource collection IDs to filter by"
//	@Param			startTime			query		int			false	"unix seconds for the start time of the trend"
//	@Param			endTime				query		int			false	"unix seconds for the end time of the trend"
//	@Success		200					{object}	api.Insight
//	@Router			/compliance/api/v1/insight/{insightId} [get]
func (h *HttpHandler) GetInsight(ctx echo.Context) error {
	insightId, err := strconv.ParseUint(ctx.Param("insightId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")

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

	apiRes, err := h.getInsightApiRes(ctx, insightRow, connectionIDs, resourceCollections, startTime, endTime)
	if err != nil {
		return err
	}

	return ctx.JSON(200, apiRes)
}

func (h *HttpHandler) getInsightTrendApiRes(ctx echo.Context, insightRow *db.Insight, connectionIDs, resourceCollections []string, startTime, endTime *time.Time, datapointCount *int) ([]api.InsightTrendDatapoint, error) {
	timeAtToInsightResults, err := h.inventoryClient.GetInsightTrendResults(httpclient.FromEchoContext(ctx), connectionIDs, resourceCollections, insightRow.ID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	result := make([]api.InsightTrendDatapoint, 0, len(timeAtToInsightResults))
	for timeAt, insightResults := range timeAtToInsightResults {
		datapoint := api.InsightTrendDatapoint{
			Timestamp:       timeAt,
			ConnectionCount: 0,
			Value:           0,
		}
		for _, insightResult := range insightResults {
			datapoint.Value += int(insightResult.Result)
			datapoint.ConnectionCount = max(datapoint.ConnectionCount, len(insightResult.IncludedConnections))
		}
		result = append(result, datapoint)
	}

	if datapointCount != nil {
		result = internal.DownSampleInsightTrendDatapoints(result, *datapointCount)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp < result[j].Timestamp
	})

	return result, nil
}

// GetInsightTrend godoc
//
//	@Summary		Get insight trend
//	@Description	Retrieving insight results datapoints for a specified connection during a specified time period.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			insightId			path		string		true	"Insight ID"
//	@Param			connectionId		query		[]string	false	"filter the result by source id"
//	@param			connectionGroup		query		[]string	false	"filter the result by connection group"
//	@Param			resourceCollection	query		[]string	false	"Resource collection IDs to filter by"
//	@Param			startTime			query		int			false	"unix seconds for the start time of the trend"
//	@Param			endTime				query		int			false	"unix seconds for the end time of the trend"
//	@Param			datapointCount		query		int			false	"number of datapoints to return"
//	@Success		200					{object}	[]api.InsightTrendDatapoint
//	@Router			/compliance/api/v1/insight/{insightId}/trend [get]
func (h *HttpHandler) GetInsightTrend(ctx echo.Context) error {
	insightId, err := strconv.ParseUint(ctx.Param("insightId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
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

	result, err := h.getInsightTrendApiRes(ctx, insightRow, connectionIDs, resourceCollections, startTime, endTime, datapointCount)
	if err != nil {
		return err
	}

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
//	@Param			tag					query		[]string		false	"Key-Value tags in key=value format to filter by"
//	@Param			connector			query		[]source.Type	false	"filter insights by connector"
//	@Param			connectionId		query		[]string		false	"filter the result by source id"
//	@param			connectionGroup		query		[]string		false	"filter the result by connection group"
//	@Param			resourceCollection	query		[]string		false	"Resource collection IDs to filter by"
//	@Param			startTime			query		int				false	"unix seconds for the start time of the trend"
//	@Param			endTime				query		int				false	"unix seconds for the end time of the trend"
//	@Success		200					{object}	[]api.InsightGroup
//	@Router			/compliance/api/v1/insight/group [get]
func (h *HttpHandler) ListInsightGroups(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver2.QueryArrayParam(ctx, "tag"))
	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")

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

	insightIdToResults, err := h.inventoryClient.ListInsightResults(httpclient.FromEchoContext(ctx), nil, connectionIDs, resourceCollections, insightIDsList, &endTime)
	if err != nil {
		return err
	}

	oldInsightIdToResults, err := h.inventoryClient.ListInsightResults(httpclient.FromEchoContext(ctx), nil, connectionIDs, resourceCollections, insightIDsList, &startTime)
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
//	@Description	Retrieving the specified insight group with ID.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			insightGroupId		path		string		true	"Insight Group ID"
//	@Param			connectionId		query		[]string	false	"filter the result by source id"
//	@param			connectionGroup		query		[]string	false	"filter the result by connection group"
//	@Param			resourceCollection	query		[]string	false	"Resource collection IDs to filter by"
//	@Param			startTime			query		int			false	"unix seconds for the start time of the trend"
//	@Param			endTime				query		int			false	"unix seconds for the end time of the trend"
//	@Success		200					{object}	api.InsightGroup
//	@Router			/compliance/api/v1/insight/group/{insightGroupId} [get]
func (h *HttpHandler) GetInsightGroup(ctx echo.Context) error {
	insightGroupId, err := strconv.ParseUint(ctx.Param("insightGroupId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
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

	_, span1 := tracer.Start(ctx.Request().Context(), "new_ListInsightGroups", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetInsightGroup")

	insightGroupRow, err := h.db.GetInsightGroup(uint(insightGroupId))
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	apiRes := insightGroupRow.ToApi()
	apiRes.Insights = make([]api.Insight, 0, len(insightGroupRow.Insights))
	for _, insightRow := range insightGroupRow.Insights {
		localInsightRow := insightRow
		insightApiRes, err := h.getInsightApiRes(ctx, &localInsightRow, connectionIDs, resourceCollections, startTime, endTime)
		if err != nil {
			h.logger.Error("failed to get insight api res", zap.Error(err),
				zap.Uint("insight id", insightRow.ID), zap.Uint("insight group id", uint(insightGroupId)))
			return err
		}
		apiRes.TotalResultValue = utils.PAdd(apiRes.TotalResultValue, insightApiRes.TotalResultValue)
		apiRes.OldTotalResultValue = utils.PAdd(apiRes.OldTotalResultValue, insightApiRes.OldTotalResultValue)
		if apiRes.FirstOldResultDate == nil ||
			insightApiRes.FirstOldResultDate != nil && apiRes.FirstOldResultDate.After(*insightApiRes.FirstOldResultDate) {
			apiRes.FirstOldResultDate = insightApiRes.FirstOldResultDate
		}
		apiRes.Insights = append(apiRes.Insights, *insightApiRes)
	}
	if apiRes.FirstOldResultDate != nil && apiRes.FirstOldResultDate.After(startTime) {
		apiRes.OldTotalResultValue = nil
	}

	return ctx.JSON(200, apiRes)
}

// GetInsightGroupTrend godoc
//
//	@Summary		Get insight group trend
//	@Description	Retrieving insight group results datapoints for a specified connection during a specified time period.
//	@Description	Returns "all:provider" job results if connectionId is not defined.
//	@Security		BearerToken
//	@Tags			insights
//	@Produce		json
//	@Param			insightGroupId		path		string		true	"Insight Group ID"
//	@Param			connectionId		query		[]string	false	"filter the result by source id"
//	@param			connectionGroup		query		[]string	false	"filter the result by connection group"
//	@Param			resourceCollection	query		[]string	false	"Resource collection IDs to filter by"
//	@Param			startTime			query		int			false	"unix seconds for the start time of the trend"
//	@Param			endTime				query		int			false	"unix seconds for the end time of the trend"
//	@Param			datapointCount		query		int			false	"number of datapoints to return"
//	@Success		200					{object}	[]api.InsightTrendDatapoint
//	@Router			/compliance/api/v1/insight/group/{insightGroupId}/trend [get]
func (h *HttpHandler) GetInsightGroupTrend(ctx echo.Context) error {
	insightGroupId, err := strconv.ParseUint(ctx.Param("insightGroupId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	connectionIDs, err := h.getConnectionIdFilterFromParams(ctx)
	if err != nil {
		return err
	}
	resourceCollections := httpserver2.QueryArrayParam(ctx, "resourceCollection")
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

	_, span1 := tracer.Start(ctx.Request().Context(), "new_ListInsightGroups", trace.WithSpanKind(trace.SpanKindServer))
	span1.SetName("new_GetInsightGroups")

	insightGroupRow, err := h.db.GetInsightGroup(uint(insightGroupId))
	if err != nil {
		span1.RecordError(err)
		span1.SetStatus(codes.Error, err.Error())
		return err
	}
	span1.End()

	dateToResultMap := make(map[int]api.InsightTrendDatapoint)
	for _, insightRow := range insightGroupRow.Insights {
		localInsightRow := insightRow
		insightTrendApiRes, err := h.getInsightTrendApiRes(ctx, &localInsightRow, connectionIDs, resourceCollections, startTime, endTime, datapointCount)
		if err != nil {
			h.logger.Error("failed to get insight trend api res", zap.Error(err),
				zap.Uint("insight id", insightRow.ID), zap.Uint("insight group id", uint(insightGroupId)))
			return err
		}
		for _, insightTrendDatapoint := range insightTrendApiRes {
			if v, ok := dateToResultMap[insightTrendDatapoint.Timestamp]; !ok {
				dateToResultMap[insightTrendDatapoint.Timestamp] = insightTrendDatapoint
			} else {
				v.Value += insightTrendDatapoint.Value
				dateToResultMap[insightTrendDatapoint.Timestamp] = v
			}
		}
	}

	result := make([]api.InsightTrendDatapoint, 0, len(dateToResultMap))
	for _, insightTrendDatapoint := range dateToResultMap {
		result = append(result, insightTrendDatapoint)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp < result[j].Timestamp
	})

	return ctx.JSON(200, result)
}
