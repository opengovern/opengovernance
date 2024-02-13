package describe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgtype"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-aws-describer/aws"
	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	complianceapi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db"
	model2 "github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/internal"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	httpserver2 "github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	onboardapi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	es2 "github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/pipeline"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/terraform-package/external/states/statefile"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/sony/sonyflake"
	"go.uber.org/zap"
	"gorm.io/gorm"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type HttpServer struct {
	Address    string
	DB         db.Database
	Scheduler  *Scheduler
	kubeClient k8sclient.Client
	helmConfig HelmConfig
}

func NewHTTPServer(
	address string,
	db db.Database,
	s *Scheduler,
	helmConfig HelmConfig,
) *HttpServer {
	return &HttpServer{
		Address:    address,
		DB:         db,
		Scheduler:  s,
		helmConfig: helmConfig,
	}
}

func (h HttpServer) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.PUT("/describe/trigger/:connection_id", httpserver2.AuthorizeHandler(h.TriggerPerConnectionDescribeJob, apiAuth.AdminRole))
	v1.PUT("/describe/trigger", httpserver2.AuthorizeHandler(h.TriggerDescribeJob, apiAuth.InternalRole))
	v1.PUT("/insight/trigger/:insight_id", httpserver2.AuthorizeHandler(h.TriggerInsightJob, apiAuth.AdminRole))
	v1.PUT("/insight/in_progress/:job_id", httpserver2.AuthorizeHandler(h.InsightJobInProgress, apiAuth.AdminRole))
	v1.GET("/insight/job/:job_id", httpserver2.AuthorizeHandler(h.GetInsightJob, apiAuth.InternalRole))
	v1.GET("/insight/:insight_id/jobs", httpserver2.AuthorizeHandler(h.GetJobsByInsightID, apiAuth.InternalRole))
	v1.PUT("/compliance/trigger/:benchmark_id", httpserver2.AuthorizeHandler(h.TriggerConnectionsComplianceJob, apiAuth.AdminRole))
	v1.PUT("/compliance/re-evaluate/:benchmark_id", httpserver2.AuthorizeHandler(h.ReEvaluateComplianceJob, apiAuth.AdminRole))
	v1.GET("/compliance/status/:benchmark_id", httpserver2.AuthorizeHandler(h.GetComplianceBenchmarkStatus, apiAuth.AdminRole))
	v1.PUT("/analytics/trigger", httpserver2.AuthorizeHandler(h.TriggerAnalyticsJob, apiAuth.InternalRole))
	v1.GET("/analytics/job/:job_id", httpserver2.AuthorizeHandler(h.GetAnalyticsJob, apiAuth.InternalRole))
	v1.GET("/describe/status/:resource_type", httpserver2.AuthorizeHandler(h.GetDescribeStatus, apiAuth.InternalRole))
	v1.GET("/describe/connection/status", httpserver2.AuthorizeHandler(h.GetConnectionDescribeStatus, apiAuth.InternalRole))
	v1.GET("/describe/pending/connections", httpserver2.AuthorizeHandler(h.ListAllPendingConnection, apiAuth.InternalRole))
	v1.GET("/describe/all/jobs/state", httpserver2.AuthorizeHandler(h.GetDescribeAllJobsStatus, apiAuth.InternalRole))

	v1.GET("/discovery/resourcetypes/list", httpserver2.AuthorizeHandler(h.GetDiscoveryResourceTypeList, apiAuth.ViewerRole))
	v1.POST("/jobs", httpserver2.AuthorizeHandler(h.ListJobs, apiAuth.ViewerRole))
	v1.GET("/jobs/bydate", httpserver2.AuthorizeHandler(h.CountJobsByDate, apiAuth.InternalRole))

	stacks := v1.Group("/stacks")
	stacks.GET("", httpserver2.AuthorizeHandler(h.ListStack, apiAuth.ViewerRole))
	stacks.GET("/:stackId", httpserver2.AuthorizeHandler(h.GetStack, apiAuth.ViewerRole))
	stacks.POST("/create", httpserver2.AuthorizeHandler(h.CreateStack, apiAuth.AdminRole))
	stacks.DELETE("/:stackId", httpserver2.AuthorizeHandler(h.DeleteStack, apiAuth.AdminRole))
	stacks.POST("/:stackId/findings", httpserver2.AuthorizeHandler(h.GetStackFindings, apiAuth.ViewerRole))
	stacks.GET("/:stackId/insight", httpserver2.AuthorizeHandler(h.GetStackInsight, apiAuth.ViewerRole))
	stacks.GET("/resource", httpserver2.AuthorizeHandler(h.ListResourceStack, apiAuth.ViewerRole))
	stacks.POST("/describer/trigger", httpserver2.AuthorizeHandler(h.TriggerStackDescriber, apiAuth.AdminRole))
	stacks.GET("/:stackId/insights", httpserver2.AuthorizeHandler(h.ListStackInsights, apiAuth.ViewerRole))

	v1.PUT("/elastic/to/opensearch/migrate", httpserver2.AuthorizeHandler(h.DoOpenSearchMigrate, apiAuth.InternalRole))
}

// ListJobs godoc
//
//	@Summary	Lists all jobs
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.ListJobsRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	api.ListJobsResponse
//	@Router		/schedule/api/v1/jobs [post]
func (h HttpServer) ListJobs(ctx echo.Context) error {
	var request api.ListJobsRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var jobs []api.Job

	srcs, err := h.Scheduler.onboardClient.ListSources(httpclient.FromEchoContext(ctx), nil)
	if err != nil {
		return err
	}

	insights, err := h.Scheduler.complianceClient.ListInsights(httpclient.FromEchoContext(ctx))
	if err != nil {
		return err
	}

	benchmarks, err := h.Scheduler.complianceClient.ListBenchmarks(httpclient.FromEchoContext(ctx))
	if err != nil {
		return err
	}

	sortBy := "id"
	switch request.SortBy {
	case api.JobSort_ByConnectionID, api.JobSort_ByJobID, api.JobSort_ByJobType, api.JobSort_ByStatus:
		sortBy = string(request.SortBy)
	}

	sortOrder := "ASC"
	if request.SortOrder == api.JobSortOrder_DESC {
		sortOrder = "DESC"
	}

	describeJobs, err := h.DB.ListAllJobs(request.PageStart, request.PageEnd, request.Hours, request.TypeFilters,
		request.StatusFilter, sortBy, sortOrder)
	if err != nil {
		return err
	}
	for _, job := range describeJobs {
		var jobSRC onboardapi.Connection
		for _, src := range srcs {
			if src.ID.String() == job.ConnectionID {
				jobSRC = src
			}
		}

		if job.JobType == "insight" {
			for _, ins := range insights {
				if fmt.Sprintf("%v", ins.ID) == job.Title {
					job.Title = ins.ShortTitle
				}
			}
		}

		if job.JobType == "compliance" {
			for _, benchmark := range benchmarks {
				if fmt.Sprintf("%v", benchmark.ID) == job.Title {
					job.Title = benchmark.Title
				}
			}
		}

		jobs = append(jobs, api.Job{
			ID:                     job.ID,
			CreatedAt:              job.CreatedAt,
			UpdatedAt:              job.UpdatedAt,
			Type:                   api.JobType(job.JobType),
			ConnectionID:           job.ConnectionID,
			ConnectionProviderID:   jobSRC.ConnectionID,
			ConnectionProviderName: jobSRC.ConnectionName,
			Title:                  job.Title,
			Status:                 job.Status,
			FailureReason:          job.FailureMessage,
		})
	}

	var jobSummaries []api.JobSummary
	summaries, err := h.DB.GetAllJobSummary(request.Hours, request.TypeFilters, request.StatusFilter)
	if err != nil {
		return err
	}
	for _, summary := range summaries {
		jobSummaries = append(jobSummaries, api.JobSummary{
			Type:   api.JobType(summary.JobType),
			Status: summary.Status,
			Count:  summary.Count,
		})
	}

	return ctx.JSON(http.StatusOK, api.ListJobsResponse{
		Jobs:      jobs,
		Summaries: jobSummaries,
	})
}

var (
	awsResourceTypeReg, _   = regexp.Compile("aws::[a-z0-9-_/]+::[a-z0-9-_/]+")
	azureResourceTypeReg, _ = regexp.Compile("microsoft.[a-z0-9-_/]+")
)

var (
	awsTableReg, _   = regexp.Compile("aws_[a-z0-9_]+")
	azureTableReg, _ = regexp.Compile("azure_[a-z0-9_]+")
)

func getResourceTypeFromTableName(tableName string, queryConnector source.Type) string {
	switch queryConnector {
	case source.CloudAWS:
		return awsSteampipe.ExtractResourceType(tableName)
	case source.CloudAzure:
		return azureSteampipe.ExtractResourceType(tableName)
	default:
		resourceType := awsSteampipe.ExtractResourceType(tableName)
		if resourceType == "" {
			resourceType = azureSteampipe.ExtractResourceType(tableName)
		}
		return resourceType
	}
}

func extractResourceTypes(query string, connector source.Type) []string {
	var result []string

	if connector == source.CloudAWS {
		awsTables := awsResourceTypeReg.FindAllString(query, -1)
		result = append(result, awsTables...)

		awsTables = awsTableReg.FindAllString(query, -1)
		for _, table := range awsTables {
			resourceType := getResourceTypeFromTableName(table, source.CloudAWS)
			if resourceType == "" {
				resourceType = table
			}
			result = append(result, resourceType)
		}
	}

	if connector == source.CloudAzure {
		azureTables := azureTableReg.FindAllString(query, -1)
		for _, table := range azureTables {
			resourceType := getResourceTypeFromTableName(table, source.CloudAzure)
			if resourceType == "" {
				resourceType = table
			}
			result = append(result, resourceType)
		}

		azureTables = azureResourceTypeReg.FindAllString(query, -1)
		result = append(result, azureTables...)
	}

	return result
}

func UniqueArray(arr []string) []string {
	m := map[string]interface{}{}
	for _, item := range arr {
		m[item] = struct{}{}
	}
	var resp []string
	for k := range m {
		resp = append(resp, k)
	}
	return resp
}

// GetDiscoveryResourceTypeList godoc
//
//	@Summary	List all resource types that will be discovered
//	@Security	BearerToken
//	@Tags		scheduler
//	@Produce	json
//	@Success	200	{object}	api.ListDiscoveryResourceTypes
//	@Router		/schedule/api/v1/discovery/resourcetypes/list [get]
func (h HttpServer) GetDiscoveryResourceTypeList(ctx echo.Context) error {
	result, err := h.Scheduler.ListDiscoveryResourceTypes()
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}

func (h HttpServer) CountJobsByDate(ctx echo.Context) error {
	startDate, err := strconv.ParseInt(ctx.QueryParam("startDate"), 10, 64)
	if err != nil {
		return err
	}
	endDate, err := strconv.ParseInt(ctx.QueryParam("endDate"), 10, 64)
	if err != nil {
		return err
	}
	includeCostStr := ctx.QueryParam("include_cost")

	var count int64
	switch api.JobType(ctx.QueryParam("jobType")) {
	case api.JobType_Discovery:
		var includeCost *bool
		if len(includeCostStr) > 0 {
			v, err := strconv.ParseBool(includeCostStr)
			if err != nil {
				return err
			}

			includeCost = &v
		}
		count, err = h.DB.CountDescribeJobsByDate(includeCost, time.UnixMilli(startDate), time.UnixMilli(endDate))
	case api.JobType_Analytics:
		count, err = h.DB.CountAnalyticsJobsByDate(time.UnixMilli(startDate), time.UnixMilli(endDate))
	case api.JobType_Compliance:
		count, err = h.DB.CountComplianceJobsByDate(time.UnixMilli(startDate), time.UnixMilli(endDate))
	case api.JobType_Insight:
		count, err = h.DB.CountInsightJobsByDate(time.UnixMilli(startDate), time.UnixMilli(endDate))
	default:
		return errors.New("invalid job type")
	}
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, count)
}

// TriggerPerConnectionDescribeJob godoc
//
//	@Summary		Triggers describer
//	@Description	Triggers a describe job to run immediately for the given connection
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Param			connection_id	path	string		true	"Connection ID"
//	@Param			resource_type	query	[]string	false	"Resource Type"
//	@Router			/schedule/api/v1/describe/trigger/{connection_id} [put]
func (h HttpServer) TriggerPerConnectionDescribeJob(ctx echo.Context) error {
	connectionID := ctx.Param("connection_id")
	forceFull := ctx.QueryParam("force_full") == "true"
	costFullDiscovery := ctx.QueryParam("cost_full_discovery") == "true"

	src, err := h.Scheduler.onboardClient.GetSource(&httpclient.Context{UserRole: apiAuth.InternalRole}, connectionID)
	if err != nil || src == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connection id")
	}

	resourceTypes := ctx.QueryParams()["resource_type"]

	if resourceTypes == nil {
		switch src.Connector {
		case source.CloudAWS:
			if forceFull {
				resourceTypes = aws.ListResourceTypes()
			} else {
				resourceTypes = aws.ListFastDiscoveryResourceTypes()
			}
		case source.CloudAzure:
			if forceFull {
				resourceTypes = azure.ListResourceTypes()
			} else {
				resourceTypes = azure.ListFastDiscoveryResourceTypes()
			}
		}
	}

	dependencyIDs := make([]int64, 0)
	for _, resourceType := range resourceTypes {
		switch src.Connector {
		case source.CloudAWS:
			if _, err := aws.GetResourceType(resourceType); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid resource type: %s", resourceType))
			}
		case source.CloudAzure:
			if _, err := azure.GetResourceType(resourceType); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid resource type: %s", resourceType))
			}
		}
		if !src.GetSupportedResourceTypeMap()[strings.ToLower(resourceType)] {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid resource type for connection: %s", resourceType))
		}
		daj, err := h.Scheduler.describe(*src, resourceType, false, costFullDiscovery)
		if err == ErrJobInProgress {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		if err != nil {
			return err
		}
		dependencyIDs = append(dependencyIDs, int64(daj.ID))
	}

	err = h.DB.CreateJobSequencer(&model2.JobSequencer{
		DependencyList:   dependencyIDs,
		DependencySource: model2.JobSequencerJobTypeDescribe,
		NextJob:          model2.JobSequencerJobTypeAnalytics,
		Status:           model2.JobSequencerWaitingForDependencies,
	})
	if err != nil {
		return fmt.Errorf("failed to create job sequencer: %v", err)
	}

	return ctx.NoContent(http.StatusOK)
}

func (h HttpServer) TriggerDescribeJob(ctx echo.Context) error {
	resourceTypes := httpserver2.QueryArrayParam(ctx, "resource_type")
	connectors := source.ParseTypes(httpserver2.QueryArrayParam(ctx, "connector"))
	forceFull := ctx.QueryParam("force_full") == "true"

	//err := h.Scheduler.CheckWorkspaceResourceLimit()
	//if err != nil {
	//	h.Scheduler.logger.Error("failed to get limits", zap.String("spot", "CheckWorkspaceResourceLimit"), zap.Error(err))
	//	DescribeJobsCount.WithLabelValues("failure").Inc()
	//	if err == ErrMaxResourceCountExceeded {
	//		return ctx.JSON(http.StatusNotAcceptable, api.ErrorResponse{Message: err.Error()})
	//	}
	//	return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: err.Error()})
	//}
	//
	connections, err := h.Scheduler.onboardClient.ListSources(&httpclient.Context{UserRole: apiAuth.InternalRole}, connectors)
	if err != nil {
		h.Scheduler.logger.Error("failed to get list of sources", zap.String("spot", "ListSources"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: err.Error()})
	}
	for _, connection := range connections {
		if !connection.IsEnabled() {
			continue
		}
		rtToDescribe := resourceTypes

		if len(rtToDescribe) == 0 {
			switch connection.Connector {
			case source.CloudAWS:
				if forceFull {
					rtToDescribe = aws.ListResourceTypes()
				} else {
					rtToDescribe = aws.ListFastDiscoveryResourceTypes()
				}
			case source.CloudAzure:
				if forceFull {
					rtToDescribe = azure.ListResourceTypes()
				} else {
					rtToDescribe = azure.ListFastDiscoveryResourceTypes()
				}
			}
		}

		for _, resourceType := range rtToDescribe {
			switch connection.Connector {
			case source.CloudAWS:
				if _, err := aws.GetResourceType(resourceType); err != nil {
					continue
				}
			case source.CloudAzure:
				if _, err := azure.GetResourceType(resourceType); err != nil {
					continue
				}
			}
			if !connection.GetSupportedResourceTypeMap()[strings.ToLower(resourceType)] {
				continue
			}
			_, err = h.Scheduler.describe(connection, resourceType, false, false)
			if err != nil {
				h.Scheduler.logger.Error("failed to describe connection", zap.String("connection_id", connection.ID.String()), zap.Error(err))
			}
		}
	}
	return ctx.JSON(http.StatusOK, "")
}

// TriggerInsightJob godoc
//
//	@Summary		Triggers insight job
//	@Description	Triggers a insight job to run immediately for the given insight
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200			{object}	[]uint
//	@Param			insight_id	path		uint	true	"Insight ID"
//	@Router			/schedule/api/v1/insight/trigger/{insight_id} [put]
func (h HttpServer) TriggerInsightJob(ctx echo.Context) error {
	insightID, err := strconv.ParseUint(ctx.Param("insight_id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid insight_id")
	}

	insights, err := h.Scheduler.complianceClient.ListInsightsMetadata(&httpclient.Context{UserRole: apiAuth.ViewerRole}, nil)
	if err != nil {
		return err
	}

	var jobIDs []uint
	for _, ins := range insights {
		if ins.ID != uint(insightID) {
			continue
		}

		id := fmt.Sprintf("all:%s", strings.ToLower(string(ins.Connector)))
		jobID, err := h.Scheduler.runInsightJob(true, ins, id, id, ins.Connector, nil)
		if err != nil {
			return err
		}
		jobIDs = append(jobIDs, jobID)
	}
	return ctx.JSON(http.StatusOK, jobIDs)
}

func (h HttpServer) InsightJobInProgress(ctx echo.Context) error {
	jobID, err := strconv.ParseUint(ctx.Param("job_id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid insight_id")
	}

	err = h.Scheduler.db.UpdateInsightJob(uint(jobID), api2.InsightJobInProgress, "")
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// TriggerConnectionsComplianceJob godoc
//
//	@Summary		Triggers compliance job
//	@Description	Triggers a compliance job to run immediately for the given benchmark
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Param			benchmark_id	path	string		true	"Benchmark ID"
//	@Param			connection_id	query	[]string	false	"Connection ID"
//	@Router			/schedule/api/v1/compliance/trigger/{benchmark_id} [put]
func (h HttpServer) TriggerConnectionsComplianceJob(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}
	benchmarkID := ctx.Param("benchmark_id")
	benchmark, err := h.Scheduler.complianceClient.GetBenchmark(clientCtx, benchmarkID)
	if err != nil {
		return fmt.Errorf("error while getting benchmarks: %v", err)
	}

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	connectionIDs := httpserver2.QueryArrayParam(ctx, "connection_id")

	lastJob, err := h.Scheduler.db.GetLastComplianceJob(benchmark.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if lastJob != nil && (lastJob.Status == model2.ComplianceJobRunnersInProgress ||
		lastJob.Status == model2.ComplianceJobSummarizerInProgress ||
		lastJob.Status == model2.ComplianceJobCreated) {
		return echo.NewHTTPError(http.StatusConflict, "compliance job is already running")
	}

	_, err = h.Scheduler.complianceScheduler.CreateComplianceReportJobs(benchmarkID, lastJob, connectionIDs)
	if err != nil {
		return fmt.Errorf("error while creating compliance job: %v", err)
	}
	return ctx.JSON(http.StatusOK, "")
}

// ReEvaluateComplianceJob godoc
//
//	@Summary		Re-evaluates compliance job
//	@Description	Triggers a discovery job to run immediately for the given connection then triggers compliance job
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Param			benchmark_id	path	string		true	"Benchmark ID"
//	@Param			connection_id	query	[]string	true	"Connection ID"
//	@Param			control_id		query	[]string	false	"Control ID"
//	@Router			/schedule/api/v1/compliance/re-evaluate/{benchmark_id} [put]
func (h HttpServer) ReEvaluateComplianceJob(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmark_id")
	connectionIDs := httpserver2.QueryArrayParam(ctx, "connection_id")
	controlIDs := httpserver2.QueryArrayParam(ctx, "control_id")

	var controls []complianceapi.Control
	if len(controlIDs) == 0 {
		benchmark, err := h.Scheduler.complianceClient.GetBenchmark(&httpclient.Context{UserRole: apiAuth.InternalRole}, benchmarkID)
		if err != nil {
			h.Scheduler.logger.Error("failed to get benchmark", zap.Error(err))
			return err
		}
		controlIDs = make([]string, 0, len(benchmark.Controls))
		for _, control := range benchmark.Controls {
			controlIDs = append(controlIDs, control)
		}
	}
	controls, err := h.Scheduler.complianceClient.ListControl(&httpclient.Context{UserRole: apiAuth.InternalRole}, controlIDs)
	if err != nil {
		h.Scheduler.logger.Error("failed to get controls", zap.Error(err))
		return err
	}
	if len(controls) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid control_id")
	}

	requiredTables := make(map[string]bool)
	for _, control := range controls {
		for _, table := range control.Query.ListOfTables {
			requiredTables[table] = true
		}
	}
	requiredResourceTypes := make([]string, 0, len(requiredTables))
	for table := range requiredTables {
		for _, provider := range source.List {
			resourceType := getResourceTypeFromTableName(table, provider)
			if resourceType != "" {
				requiredResourceTypes = append(requiredResourceTypes, resourceType)
				break
			}
		}
	}

	connections, err := h.Scheduler.onboardClient.GetSources(&httpclient.Context{UserRole: apiAuth.InternalRole}, connectionIDs)
	if err != nil {
		h.Scheduler.logger.Error("failed to get connections", zap.Error(err))
		return err
	}
	dependencyIDs := make([]int64, 0)
	for _, connection := range connections {
		if !connection.IsEnabled() {
			continue
		}
		for _, resourceType := range requiredResourceTypes {
			daj, err := h.Scheduler.describe(connection, resourceType, false, false)
			if err != nil {
				h.Scheduler.logger.Error("failed to describe connection", zap.String("connection_id", connection.ID.String()), zap.Error(err))
			}
			dependencyIDs = append(dependencyIDs, int64(daj.ID))
		}
	}

	jobParameters := model2.JobSequencerJobTypeBenchmarkRunnerParameters{
		BenchmarkID:   benchmarkID,
		ControlIDs:    controlIDs,
		ConnectionIDs: connectionIDs,
	}
	jobParametersJSON, err := json.Marshal(jobParameters)
	if err != nil {
		h.Scheduler.logger.Error("failed to marshal job parameters", zap.Error(err))
		return err
	}

	jp := pgtype.JSONB{}
	err = jp.Set(jobParametersJSON)
	if err != nil {
		h.Scheduler.logger.Error("failed to set job parameters", zap.Error(err))
		return err
	}

	err = h.DB.CreateJobSequencer(&model2.JobSequencer{
		DependencyList:    dependencyIDs,
		DependencySource:  model2.JobSequencerJobTypeDescribe,
		NextJob:           model2.JobSequencerJobTypeBenchmarkRunner,
		NextJobParameters: &jp,
		Status:            model2.JobSequencerWaitingForDependencies,
	})

	return ctx.NoContent(http.StatusNotImplemented)
}

func (h HttpServer) GetComplianceBenchmarkStatus(ctx echo.Context) error {
	benchmarkId := ctx.Param("benchmark_id")
	lastComplianceJob, err := h.Scheduler.db.GetLastComplianceJob(benchmarkId)
	if err != nil {
		h.Scheduler.logger.Error("failed to get compliance job", zap.String("benchmark_id", benchmarkId), zap.Error(err))
		return err
	}
	if lastComplianceJob == nil {
		return ctx.JSON(http.StatusOK, nil)
	}
	return ctx.JSON(http.StatusOK, lastComplianceJob.ToApi())
}

func (h HttpServer) GetAnalyticsJob(ctx echo.Context) error {
	jobIDstr := ctx.Param("job_id")
	jobID, err := strconv.ParseInt(jobIDstr, 10, 64)
	if err != nil {
		return err
	}

	job, err := h.Scheduler.db.GetAnalyticsJobByID(uint(jobID))
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, job)
}

func (h HttpServer) GetInsightJob(ctx echo.Context) error {
	jobIDstr := ctx.Param("job_id")
	jobID, err := strconv.ParseInt(jobIDstr, 10, 64)
	if err != nil {
		return err
	}

	job, err := h.Scheduler.db.GetInsightJobById(uint(jobID))
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, job)
}

func (h HttpServer) GetJobsByInsightID(ctx echo.Context) error {
	insightIDstr := ctx.Param("insight_id")
	insightID, err := strconv.ParseInt(insightIDstr, 10, 64)
	if err != nil {
		return err
	}

	jobs, err := h.Scheduler.db.GetInsightJobByInsightId(uint(insightID))
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, jobs)
}

func (h HttpServer) TriggerAnalyticsJob(ctx echo.Context) error {
	jobID, err := h.Scheduler.scheduleAnalyticsJob(model2.AnalyticsJobTypeNormal)
	if err != nil {
		errMsg := fmt.Sprintf("error scheduling summarize job: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: errMsg})
	}
	return ctx.JSON(http.StatusOK, jobID)
}

func (h HttpServer) GetDescribeStatus(ctx echo.Context) error {
	resourceType := ctx.Param("resource_type")

	status, err := h.DB.GetDescribeStatus(resourceType)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, status)
}

// GetConnectionDescribeStatus godoc
//
//	@Summary	Get connection describe status
//	@Security	BearerToken
//	@Tags		describe
//	@Produce	json
//	@Success	200
//	@Param		connection_id	query	string	true	"Connection ID"
//	@Router		/schedule/api/v1/describe/connection/status [put]
func (h HttpServer) GetConnectionDescribeStatus(ctx echo.Context) error {
	connectionID := ctx.QueryParam("connection_id")

	status, err := h.DB.GetConnectionDescribeStatus(connectionID)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, status)
}

func (h HttpServer) ListAllPendingConnection(ctx echo.Context) error {
	status, err := h.DB.ListAllPendingConnection()
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, status)
}

func (h HttpServer) GetDescribeAllJobsStatus(ctx echo.Context) error {
	count, _, err := h.DB.CountJobsAndResources()
	if err != nil {
		return err
	}

	if count == nil || *count == 0 {
		return ctx.JSON(http.StatusOK, api.DescribeAllJobsStatusNoJobToRun)
	}

	pendingDiscoveryTypes, err := h.DB.ListAllFirstTryPendingConnection()
	if err != nil {
		return err
	}

	for _, dt := range pendingDiscoveryTypes {
		if dt == string(model2.DiscoveryType_Cost) || dt == string(model2.DiscoveryType_Fast) {
			return ctx.JSON(http.StatusOK, api.DescribeAllJobsStatusJobsRunning)
		}
	}

	succeededJobs, err := h.DB.ListAllSuccessfulDescribeJobs()
	if err != nil {
		return err
	}

	publishedJobs := 0
	totalJobs := 0
	for _, job := range succeededJobs {
		totalJobs++

		if job.DescribedResourceCount > 0 {
			resourceCount, err := es.GetInventoryCountResponse(h.Scheduler.es, strings.ToLower(job.ResourceType))
			if err != nil {
				return err
			}

			if resourceCount > 0 {
				publishedJobs++
			}
		} else {
			publishedJobs++
		}
	}

	h.Scheduler.logger.Info("job count",
		zap.Int("publishedJobs", publishedJobs),
		zap.Int("totalJobs", totalJobs),
	)
	if publishedJobs == totalJobs {
		return ctx.JSON(http.StatusOK, api.DescribeAllJobsStatusResourcesPublished)
	}

	job, err := h.DB.GetLastSuccessfulDescribeJob()
	if err != nil {
		return err
	}

	if job != nil &&
		job.UpdatedAt.Before(time.Now().Add(-5*time.Minute)) {
		return ctx.JSON(http.StatusOK, api.DescribeAllJobsStatusResourcesPublished)
	}

	return ctx.JSON(http.StatusOK, api.DescribeAllJobsStatusJobsFinished)
}

type MigratorResponse struct {
	Hits  MigratorHits `json:"hits"`
	PitID string
}
type MigratorHits struct {
	Total kaytu.SearchTotal `json:"total"`
	Hits  []MigratorHit     `json:"hits"`
}
type MigratorHit struct {
	ID      string        `json:"_id"`
	Score   float64       `json:"_score"`
	Index   string        `json:"_index"`
	Type    string        `json:"_type"`
	Version int64         `json:"_version,omitempty"`
	Source  MigrateSource `json:"_source"`
	Sort    []any         `json:"sort"`
}

type MigrateSource map[string]any

func (m MigrateSource) KeysAndIndex() ([]string, string) {
	return nil, ""
}

func (h HttpServer) DoOpenSearchMigrate(ctx echo.Context) error {
	indexesToMigrate := []string{
		"analytics_connection_summary",
		"analytics_connector_summary",
		"analytics_spend_connection_summary",
		"analytics_spend_connector_summary",
		"rc_analytics_connection_summary",
		"rc_analytics_connector_summary",
		"insights",
		"benchmark_summary",
	}

	ingestionPipeUrl := ctx.QueryParam("ingestion_pipeline_url")

	for _, indexToMigrate := range indexesToMigrate {
		paginator, err := kaytu.NewPaginator(h.Scheduler.es.ES(), indexToMigrate, nil, nil)
		if err != nil {
			return err
		}

		ctx := context.Background()
		for {
			if paginator.Done() {
				break
			}

			h.Scheduler.logger.Info("migration: next page", zap.String("index", indexToMigrate))
			var res MigratorResponse
			err = paginator.SearchWithLog(ctx, &res, true)
			if err != nil {
				return err
			}

			var items []es2.Doc
			for _, hit := range res.Hits.Hits {
				item := hit.Source
				item["es_id"] = hit.ID
				item["es_index"] = indexToMigrate
				items = append(items, hit.Source)
			}

			h.Scheduler.logger.Info("migration: piping data", zap.String("index", indexToMigrate), zap.Int("count", len(items)))

			for startPageIdx := 0; startPageIdx < len(items); startPageIdx += 100 {
				msgsToSend := items[startPageIdx:min(startPageIdx+100, len(items))]
				err := pipeline.SendToPipeline(ingestionPipeUrl, msgsToSend)
				if err != nil {
					return err
				}
			}

			hits := int64(len(res.Hits.Hits))
			if hits > 0 {
				paginator.UpdateState(hits, res.Hits.Hits[hits-1].Sort, res.PitID)
			} else {
				paginator.UpdateState(hits, nil, "")
			}
		}

		err = paginator.Deallocate(ctx)
		if err != nil {
			return err
		}
	}

	return ctx.NoContent(http.StatusOK)
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

// CreateStack godoc
//
//	@Summary		Create stack
//	@Description	Create a stack by giving terraform statefile and additional resources
//	@Description	Config structure for azure: {tenantId: string, objectId: string, secretId: string, clientId: string, clientSecret:string}
//	@Description	Config structure for aws: {accessKey: string, secretKey: string}
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			stateFile			formData	file	false	"ُTerraform StateFile full path"
//	@Param			tag					formData	string	false	"Tags Map[string][]string"
//	@Param			config				formData	string	true	"Config json structure"
//	@Param			remoteStateConfig	formData	string	false	"Config json structure for terraform remote state backend"
//	@Success		200					{object}	api.Stack
//	@Router			/schedule/api/v1/stacks/create [post]
func (h HttpServer) CreateStack(ctx echo.Context) error {
	var tags map[string][]string
	tagsData := ctx.FormValue("tag")
	if tagsData != "" {
		json.Unmarshal([]byte(tagsData), &tags)
	}

	file, err := ctx.FormFile("stateFile")
	if err != nil {
		if err.Error() != "http: no such file" {
			return err
		}
	}
	if file != nil {
		src, err := file.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(src)
		if err != nil {
			return err
		}
		if string(data) == "{}" {
			file = nil
		}
		err = src.Close()
		if err != nil {
			return err
		}
	}
	stateConfig := ctx.FormValue("remoteStateConfig")
	if file == nil && stateConfig == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "No state file or remote backend provided")
	}
	configStr := ctx.FormValue("config")
	if configStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide the credentials")
	}

	var resources []string
	var terraformResourceTypes []string
	if file != nil {
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		data, err := io.ReadAll(src)
		if err != nil {
			return err
		}
		if !strings.HasSuffix(file.Filename, ".tfstate") {
			echo.NewHTTPError(http.StatusBadRequest, "File must have a .tfstate suffix")
		}
		arns, err := internal.GetArns(string(data))
		if err != nil {
			return err
		}
		terraformResourceTypes, err = internal.GetTypes(string(data))
		if err != nil {
			return err
		}
		resources = append(resources, arns...)
	} else {
		var conf internal.Config
		err = json.Unmarshal([]byte(stateConfig), &conf)
		if err != nil {
			echo.NewHTTPError(http.StatusBadRequest, "Error unmarshaling config json")
		}
		if conf.Type == "s3" {
			err = internal.ConfigureAWSAccount(configStr)
		} else if conf.Type == "azurem" {
			err = internal.ConfigureAzureAccount(configStr)
		}
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Could not parse state backend configs")
		}
		state := internal.GetRemoteState(conf)
		resources = statefile.GetArnsFromStateFile(state)
		terraformResourceTypes = statefile.GetResourcesTypesFromState(state)

		err = internal.RestartCredentials()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Could not reset configs")
		}
	}

	var recordTags []*model2.StackTag
	if len(tags) != 0 {
		for key, value := range tags {
			recordTags = append(recordTags, &model2.StackTag{
				Key:   key,
				Value: value,
			})
		}
	}

	var provider source.Type
	for _, resource := range resources {
		if strings.Contains(resource, "aws") {
			provider = source.CloudAWS
		} else if strings.Contains(resource, "subscriptions") {
			provider = source.CloudAzure
		}
	}

	terraformResourceTypes = removeDuplicates(terraformResourceTypes)
	if err != nil {
		return err
	}
	var resourceTypes []string
	if provider == source.CloudAWS {
		for _, trt := range terraformResourceTypes {
			rt := aws.GetResourceTypeByTerraform(trt)
			if rt != "" {
				resourceTypes = append(resourceTypes, rt)
			}
		}
	} else if provider == source.CloudAzure {
		for _, trt := range terraformResourceTypes {
			rt := azure.GetResourceTypeByTerraform(trt)
			if rt != "" {
				resourceTypes = append(resourceTypes, rt)
			}
		}
	}

	accs, err := internal.ParseAccountsFromArns(resources)
	if err != nil {
		return err
	}
	sf := sonyflake.NewSonyflake(sonyflake.Settings{})
	id, err := sf.NextID()
	if err != nil {
		return err
	}

	stackRecord := model2.Stack{
		StackID:       fmt.Sprintf("stack-%d", id),
		Resources:     pq.StringArray(resources),
		Tags:          recordTags,
		AccountIDs:    accs,
		ResourceTypes: pq.StringArray(resourceTypes),
		SourceType:    provider,
		Status:        api.StackStatusPending,
	}
	err = h.DB.AddStack(&stackRecord)
	if err != nil {
		return err
	}

	err = h.Scheduler.storeStackCredentials(stackRecord.ToApi(), configStr) // should be removed after describing
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, stackRecord.ToApi())
}

// GetStack godoc
//
//	@Summary		Get Stack
//	@Description	Get stack details by ID
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			stackId	path		string	true	"StackID"
//	@Success		200		{object}	api.Stack
//	@Router			/schedule/api/v1/stacks/{stackId} [get]
func (h HttpServer) GetStack(ctx echo.Context) error {
	stackId := ctx.Param("stackId")
	stackRecord, err := h.DB.GetStack(stackId)
	if err != nil {
		return err
	}

	if stackRecord.StackID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "stack not found")
	}

	return ctx.JSON(http.StatusOK, stackRecord.ToApi())
}

// ListStack godoc
//
//	@Summary		List Stacks
//	@Description	Get list of stacks
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			tag			query		[]string	false	"Key-Value tags in key=value format to filter by"
//	@Param			accountIds	query		[]string	false	"Account IDs to filter by"
//	@Success		200			{object}	[]api.Stack
//	@Router			/schedule/api/v1/stacks [get]
func (h HttpServer) ListStack(ctx echo.Context) error {
	tagMap := model.TagStringsToTagMap(httpserver2.QueryArrayParam(ctx, "tag"))
	accountIds := httpserver2.QueryArrayParam(ctx, "accountIds")
	stacksRecord, err := h.DB.ListStacks(tagMap, accountIds)
	if err != nil {
		return err
	}
	var stacks []api.Stack
	for _, sr := range stacksRecord {

		stack := api.Stack{
			StackID:       sr.StackID,
			CreatedAt:     sr.CreatedAt,
			UpdatedAt:     sr.UpdatedAt,
			Resources:     []string(sr.Resources),
			ResourceTypes: []string(sr.ResourceTypes),
			Tags:          model.TrimPrivateTags(sr.GetTagsMap()),
			Status:        sr.Status,
			SourceType:    sr.SourceType,
			AccountIDs:    sr.AccountIDs,
		}
		stacks = append(stacks, stack)
	}
	return ctx.JSON(http.StatusOK, stacks)
}

// DeleteStack godoc
//
//	@Summary		Delete Stack
//	@Description	Delete a stack by ID
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			stackId	path	string	true	"StackID"
//	@Success		200
//	@Router			/schedule/api/v1/stacks/{stackId} [delete]
func (h HttpServer) DeleteStack(ctx echo.Context) error {
	stackId := ctx.Param("stackId")
	err := h.DB.DeleteStack(stackId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, "stack not found")
		} else {
			return err
		}
	}
	return ctx.NoContent(http.StatusOK)
}

// GetStackFindings godoc
//
//	@Summary		Get Stack Findings
//	@Description	Get all findings for a stack
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			stackId	path		string					true	"StackId"
//	@Param			request	body		api.GetStackFindings	true	"Request Body"
//	@Success		200		{object}	complianceapi.GetFindingsResponse
//	@Router			/schedule/api/v1/stacks/{stackId}/findings [post]
func (h HttpServer) GetStackFindings(ctx echo.Context) error {
	stackId := ctx.Param("stackId")
	var reqBody api.GetStackFindings
	bindValidate(ctx, &reqBody)
	stackRecord, err := h.DB.GetStack(stackId)
	if err != nil {
		return err
	}
	if stackRecord.StackID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "stack not found")
	}
	connectionId := stackRecord.StackID

	req := complianceapi.GetFindingsRequest{
		Filters: complianceapi.FindingFilters{
			ConnectionID: []string{connectionId},
			BenchmarkID:  reqBody.BenchmarkIDs,
			ResourceID:   []string(stackRecord.Resources),
		},
	}

	findings, err := h.Scheduler.complianceClient.GetFindings(httpclient.FromEchoContext(ctx), req)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, findings)
}

// GetStackInsight godoc
//
//	@Summary		Get Stack Insight
//	@Description	Get Insight results for a stack in the given time period
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			insightId	query		int		true	"InsightID"
//	@Param			startTime	query		int		false	"unix seconds for the start time of the trend"
//	@Param			endTime		query		int		false	"unix seconds for the end time of the trend"
//	@Param			stackId		path		string	true	"StackID"
//	@Success		200			{object}	complianceapi.Insight
//	@Router			/schedule/api/v1/stacks/{stackId}/insight [get]
func (h HttpServer) GetStackInsight(ctx echo.Context) error {
	stackId := ctx.Param("stackId")
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
	insightId := ctx.QueryParam("insightId")
	stackRecord, err := h.DB.GetStack(stackId)
	if err != nil {
		return err
	}
	if stackRecord.StackID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "stack not found")
	}
	connectionId := stackRecord.StackID

	insight, err := h.Scheduler.complianceClient.GetInsight(httpclient.FromEchoContext(ctx), insightId, []string{connectionId}, &startTime, &endTime)
	if err != nil {
		return err
	}
	var totalResaults int64
	var filteredResults []complianceapi.InsightResult
	for _, result := range insight.Results {
		var headerIndex int
		for i, header := range result.Details.Headers {
			if header == "kaytu_resource_id" {
				headerIndex = i
			}
		}
		var count int64
		var filteredRows [][]interface{}
		for _, row := range result.Details.Rows {
			for _, resourceId := range []string(stackRecord.Resources) {
				if row[headerIndex] == resourceId {
					filteredRows = append(filteredRows, row)
					count++
					break
				}
			}
		}
		if count > 0 {
			result.Details = &complianceapi.InsightDetail{
				Headers: result.Details.Headers,
				Rows:    filteredRows,
			}
			result.Result = count
			filteredResults = append(filteredResults, result)
			totalResaults = totalResaults + count
		}
	}
	insight.Results = filteredResults
	insight.TotalResultValue = &totalResaults
	return ctx.JSON(http.StatusOK, insight)
}

// ListStackInsights godoc
//
//	@Summary		List Stack Insights
//	@Description	Get all Insights results with the given filters
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			insightIds	query		[]int	false	"Insight IDs to filter with. If empty, then all insights are returned"
//	@Param			startTime	query		int		false	"unix seconds for the start time of the trend"
//	@Param			endTime		query		int		false	"unix seconds for the end time of the trend"
//	@Param			stackId		path		string	true	"Stack ID"
//	@Success		200			{object}	[]complianceapi.Insight
//	@Router			/schedule/api/v1/stacks/{stackId}/insights [get]
func (h HttpServer) ListStackInsights(ctx echo.Context) error {
	stackId := ctx.Param("stackId")
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

	stackRecord, err := h.DB.GetStack(stackId)
	if err != nil {
		return err
	}
	if stackRecord.StackID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "stack not found")
	}
	connectionId := stackRecord.StackID

	insightIds := httpserver2.QueryArrayParam(ctx, "insightIds")
	if len(insightIds) == 0 {
		insightIds = []string{}
		insights, err := h.Scheduler.complianceClient.ListInsightsMetadata(httpclient.FromEchoContext(ctx), []source.Type{stackRecord.SourceType})
		if err != nil {
			return err
		}
		for _, insight := range insights {
			insightIds = append(insightIds, strconv.FormatUint(uint64(insight.ID), 10))
		}
	}

	var insights []complianceapi.Insight
	for _, insightId := range insightIds {
		insight, err := h.Scheduler.complianceClient.GetInsight(httpclient.FromEchoContext(ctx), insightId, []string{connectionId}, &startTime, &endTime)
		if err != nil {
			if strings.Contains(err.Error(), "no data for insight found") {
				continue
			} else {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("error for insight %s: %s", insightId, err.Error()))
			}
		}
		var totalResaults int64
		var filteredResults []complianceapi.InsightResult
		for _, result := range insight.Results {
			var headerIndex int
			for i, header := range result.Details.Headers {
				if header == "kaytu_resource_id" {
					headerIndex = i
				}
			}
			var count int64
			var filteredRows [][]interface{}
			for _, row := range result.Details.Rows {
				for _, resourceId := range []string(stackRecord.Resources) {
					if row[headerIndex] == resourceId {
						filteredRows = append(filteredRows, row)
						count++
						break
					}
				}
			}
			if count > 0 {
				result.Details = &complianceapi.InsightDetail{
					Headers: result.Details.Headers,
					Rows:    filteredRows,
				}
				result.Result = count
				filteredResults = append(filteredResults, result)
				totalResaults = totalResaults + count
			}
		}
		insight.Results = filteredResults
		insight.TotalResultValue = &totalResaults
		if totalResaults > 0 {
			insights = append(insights, insight)
		}
	}
	return ctx.JSON(http.StatusOK, insights)
}

// ListResourceStack godoc
//
//	@Summary		List Resource Stacks
//	@Description	Get list of all stacks containing a resource
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			resourceId	query		string	true	"Resource ID"
//	@Success		200			{object}	[]api.Stack
//	@Router			/schedule/api/v1/stacks/resource [get]
func (h HttpServer) ListResourceStack(ctx echo.Context) error {
	resourceId := ctx.QueryParam("resourceId")
	stacksRecord, err := h.DB.GetResourceStacks(resourceId)
	if err != nil {
		return err
	}
	var stacks []api.Stack
	for _, sr := range stacksRecord {

		stack := api.Stack{
			StackID:       sr.StackID,
			CreatedAt:     sr.CreatedAt,
			UpdatedAt:     sr.UpdatedAt,
			Resources:     []string(sr.Resources),
			Tags:          model.TrimPrivateTags(sr.GetTagsMap()),
			AccountIDs:    sr.AccountIDs,
			Status:        sr.Status,
			SourceType:    sr.SourceType,
			ResourceTypes: sr.ResourceTypes,
		}
		stacks = append(stacks, stack)
	}
	return ctx.JSON(http.StatusOK, stacks)
}

func (h HttpServer) TriggerStackDescriber(ctx echo.Context) error { // Retired
	var req api.DescribeStackRequest

	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	stackRecord, err := h.DB.GetStack(req.StackID)
	if err != nil {
		return err
	}
	stack := stackRecord.ToApi()
	configStr, err := json.Marshal(req.Config)
	if err != nil {
		return err
	}
	err = h.Scheduler.storeStackCredentials(stack, string(configStr))
	if err != nil {
		return err
	}
	err = h.Scheduler.triggerStackDescriberJob(stack)
	if err != nil {
		return err
	}
	return ctx.NoContent(http.StatusOK)
}

// function to remove duplicate values
func removeDuplicates(s []string) []string {
	bucket := make(map[string]bool)
	var result []string
	for _, str := range s {
		if _, ok := bucket[str]; !ok {
			bucket[str] = true
			result = append(result, str)
		}
	}
	return result
}
