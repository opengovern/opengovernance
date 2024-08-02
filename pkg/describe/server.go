package describe

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgtype"
	runner2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	apiAuth "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/labstack/echo/v4"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-aws-describer/aws"
	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	complianceapi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db"
	model2 "github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	onboardapi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
	"gorm.io/gorm"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type HttpServer struct {
	Address    string
	DB         db.Database
	Scheduler  *Scheduler
	kubeClient k8sclient.Client
}

func NewHTTPServer(
	address string,
	db db.Database,
	s *Scheduler,
) *HttpServer {
	return &HttpServer{
		Address:   address,
		DB:        db,
		Scheduler: s,
	}
}

func (h HttpServer) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.PUT("/describe/trigger/:connection_id", httpserver.AuthorizeHandler(h.TriggerPerConnectionDescribeJob, apiAuth.AdminRole))
	v1.PUT("/describe/trigger", httpserver.AuthorizeHandler(h.TriggerDescribeJob, apiAuth.InternalRole))
	v1.PUT("/insight/trigger/:insight_id", httpserver.AuthorizeHandler(h.TriggerInsightJob, apiAuth.AdminRole))
	v1.PUT("/insight/in_progress/:job_id", httpserver.AuthorizeHandler(h.InsightJobInProgress, apiAuth.AdminRole))
	v1.GET("/insight/job/:job_id", httpserver.AuthorizeHandler(h.GetInsightJob, apiAuth.InternalRole))
	v1.GET("/insight/:insight_id/jobs", httpserver.AuthorizeHandler(h.GetJobsByInsightID, apiAuth.InternalRole))
	v1.PUT("/compliance/trigger", httpserver.AuthorizeHandler(h.TriggerConnectionsComplianceJobs, apiAuth.AdminRole))
	v1.PUT("/compliance/trigger/:benchmark_id", httpserver.AuthorizeHandler(h.TriggerConnectionsComplianceJob, apiAuth.AdminRole))
	v1.PUT("/compliance/trigger/:benchmark_id/summary", httpserver.AuthorizeHandler(h.TriggerConnectionsComplianceJobSummary, apiAuth.AdminRole))
	v1.GET("/compliance/re-evaluate/:benchmark_id", httpserver.AuthorizeHandler(h.CheckReEvaluateComplianceJob, apiAuth.AdminRole))
	v1.PUT("/compliance/re-evaluate/:benchmark_id", httpserver.AuthorizeHandler(h.ReEvaluateComplianceJob, apiAuth.AdminRole))
	v1.GET("/compliance/status/:benchmark_id", httpserver.AuthorizeHandler(h.GetComplianceBenchmarkStatus, apiAuth.AdminRole))
	v1.PUT("/analytics/trigger", httpserver.AuthorizeHandler(h.TriggerAnalyticsJob, apiAuth.InternalRole))
	v1.GET("/analytics/job/:job_id", httpserver.AuthorizeHandler(h.GetAnalyticsJob, apiAuth.InternalRole))
	v1.GET("/describe/status/:resource_type", httpserver.AuthorizeHandler(h.GetDescribeStatus, apiAuth.InternalRole))
	v1.GET("/describe/connection/status", httpserver.AuthorizeHandler(h.GetConnectionDescribeStatus, apiAuth.InternalRole))
	v1.GET("/describe/pending/connections", httpserver.AuthorizeHandler(h.ListAllPendingConnection, apiAuth.InternalRole))
	v1.GET("/describe/all/jobs/state", httpserver.AuthorizeHandler(h.GetDescribeAllJobsStatus, apiAuth.InternalRole))

	v1.GET("/discovery/resourcetypes/list", httpserver.AuthorizeHandler(h.GetDiscoveryResourceTypeList, apiAuth.ViewerRole))
	v1.POST("/jobs", httpserver.AuthorizeHandler(h.ListJobs, apiAuth.ViewerRole))
	v1.GET("/jobs/bydate", httpserver.AuthorizeHandler(h.CountJobsByDate, apiAuth.InternalRole))
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

	benchmarks, err := h.Scheduler.complianceClient.ListBenchmarks(httpclient.FromEchoContext(ctx), nil)
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

func getResourceTypeFromTableNameLower(tableName string, queryConnector source.Type) string {
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

func getResourceTypeFromTableName(tableName string, queryConnector source.Type) string {
	switch queryConnector {
	case source.CloudAWS:
		rt := awsSteampipe.GetResourceTypeByTableName(tableName)
		if rt != "" {
			for k, _ := range awsSteampipe.AWSDescriptionMap {
				if strings.ToLower(k) == strings.ToLower(rt) {
					return k
				}
			}
		}
	case source.CloudAzure:
		rt := azureSteampipe.GetResourceTypeByTableName(tableName)
		if rt != "" {
			for k, _ := range azureSteampipe.AzureDescriptionMap {
				if strings.ToLower(k) == strings.ToLower(rt) {
					return k
				}
			}
		}
	default:
		rt := awsSteampipe.GetResourceTypeByTableName(tableName)
		if rt != "" {
			for k, _ := range awsSteampipe.AWSDescriptionMap {
				if strings.ToLower(k) == strings.ToLower(rt) {
					return k
				}
			}
		}
		rt = azureSteampipe.GetResourceTypeByTableName(tableName)
		if rt != "" {
			for k, _ := range azureSteampipe.AzureDescriptionMap {
				if strings.ToLower(k) == strings.ToLower(rt) {
					return k
				}
			}
		}
	}
	return ""
}

func extractResourceTypes(query string, connectors []source.Type) []string {
	var result []string

	for _, connector := range connectors {
		if connector == source.CloudAWS {
			awsTables := awsResourceTypeReg.FindAllString(query, -1)
			result = append(result, awsTables...)

			awsTables = awsTableReg.FindAllString(query, -1)
			for _, table := range awsTables {
				resourceType := getResourceTypeFromTableNameLower(table, source.CloudAWS)
				if resourceType == "" {
					resourceType = table
				}
				result = append(result, resourceType)
			}
		}

		if connector == source.CloudAzure {
			azureTables := azureTableReg.FindAllString(query, -1)
			for _, table := range azureTables {
				resourceType := getResourceTypeFromTableNameLower(table, source.CloudAzure)
				if resourceType == "" {
					resourceType = table
				}
				result = append(result, resourceType)
			}

			azureTables = azureResourceTypeReg.FindAllString(query, -1)
			result = append(result, azureTables...)
		}
	}

	return result
}

func UniqueArray[T any](arr []T) []T {
	m := make(map[string]T)
	for _, item := range arr {
		// hash the item
		hash := sha1.New()
		hash.Write([]byte(fmt.Sprintf("%v", item)))
		hashResult := hash.Sum(nil)
		m[fmt.Sprintf("%x", hashResult)] = item
	}
	var resp []T
	for _, v := range m {
		resp = append(resp, v)
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

	ctx2 := &httpclient.Context{UserRole: apiAuth.InternalRole}
	ctx2.Ctx = ctx.Request().Context()
	src, err := h.Scheduler.onboardClient.GetSource(ctx2, connectionID)
	if err != nil || src == nil {
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		} else {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid connection id")
		}
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
		daj, err := h.Scheduler.describe(*src, resourceType, false, costFullDiscovery, false)
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
	resourceTypes := httpserver.QueryArrayParam(ctx, "resource_type")
	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))
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
			_, err = h.Scheduler.describe(connection, resourceType, false, false, false)
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
		jobID, err := h.Scheduler.runInsightJob(ctx.Request().Context(), true, ins, id, id, ins.Connector, nil)
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

	connectionIDs := httpserver.QueryArrayParam(ctx, "connection_id")

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

// TriggerConnectionsComplianceJobs godoc
//
//	@Summary		Triggers compliance job
//	@Description	Triggers a compliance job to run immediately for the given benchmark
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Param			benchmark_id	query	[]string	true	"Benchmark ID"
//	@Param			connection_id	query	[]string	false	"Connection ID"
//	@Router			/schedule/api/v1/compliance/trigger [put]
func (h HttpServer) TriggerConnectionsComplianceJobs(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}
	benchmarkIDs := httpserver.QueryArrayParam(ctx, "benchmark_id")

	connectionIDs := httpserver.QueryArrayParam(ctx, "connection_id")

	for _, benchmarkID := range benchmarkIDs {
		benchmark, err := h.Scheduler.complianceClient.GetBenchmark(clientCtx, benchmarkID)
		if err != nil {
			return fmt.Errorf("error while getting benchmarks: %v", err)
		}

		if benchmark == nil {
			return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
		}

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
	}
	return ctx.JSON(http.StatusOK, "")
}

// TriggerConnectionsComplianceJobSummary godoc
//
//	@Summary		Triggers compliance job
//	@Description	Triggers a compliance job to run immediately for the given benchmark
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Param			benchmark_id	path	string	true	"Benchmark ID"
//	@Router			/schedule/api/v1/compliance/trigger/{benchmark_id}/summary [put]
func (h HttpServer) TriggerConnectionsComplianceJobSummary(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}
	benchmarkID := ctx.Param("benchmark_id")
	benchmark, err := h.Scheduler.complianceClient.GetBenchmark(clientCtx, benchmarkID)
	if err != nil {
		return fmt.Errorf("error while getting benchmarks: %v", err)
	}

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	err = h.Scheduler.complianceScheduler.CreateSummarizer(benchmarkID, nil)
	if err != nil {
		return fmt.Errorf("error while creating compliance job summarizer: %v", err)
	}
	return ctx.JSON(http.StatusOK, "")
}

type ReEvaluateDescribeJob struct {
	Connection   onboardapi.Connection
	ResourceType string
}

func (h HttpServer) getReEvaluateParams(benchmarkID string, connectionIDs, controlIDs []string) (*model2.JobSequencerJobTypeBenchmarkRunnerParameters, []ReEvaluateDescribeJob, error) {
	var controls []complianceapi.Control
	var err error
	if len(controlIDs) == 0 {
		controlIDs, err = h.getBenchmarkChildrenControls(benchmarkID)
		if err != nil {
			return nil, nil, err
		}
	}
	controls, err = h.Scheduler.complianceClient.ListControl(&httpclient.Context{UserRole: apiAuth.InternalRole}, controlIDs, nil)
	if err != nil {
		h.Scheduler.logger.Error("failed to get controls", zap.Error(err))
		return nil, nil, err
	}
	if len(controls) == 0 {
		return nil, nil, echo.NewHTTPError(http.StatusBadRequest, "invalid control_id")
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
	if len(requiredResourceTypes) == 0 {
		return nil, nil, echo.NewHTTPError(http.StatusNotFound, "no resource type found for controls")
	}

	connections, err := h.Scheduler.onboardClient.GetSources(&httpclient.Context{UserRole: apiAuth.InternalRole}, connectionIDs)
	if err != nil {
		h.Scheduler.logger.Error("failed to get connections", zap.Error(err))
		return nil, nil, err
	}
	var describeJobs []ReEvaluateDescribeJob
	for _, connection := range connections {
		if !connection.IsEnabled() {
			continue
		}
		for _, resourceType := range requiredResourceTypes {
			describeJobs = append(describeJobs, ReEvaluateDescribeJob{
				Connection:   connection,
				ResourceType: resourceType,
			})
		}
	}

	return &model2.JobSequencerJobTypeBenchmarkRunnerParameters{
		BenchmarkID:   benchmarkID,
		ControlIDs:    controlIDs,
		ConnectionIDs: connectionIDs,
	}, describeJobs, nil
}

func (h HttpServer) getBenchmarkChildrenControls(benchmarkID string) ([]string, error) {
	benchmark, err := h.Scheduler.complianceClient.GetBenchmark(&httpclient.Context{UserRole: apiAuth.InternalRole}, benchmarkID)
	if err != nil {
		h.Scheduler.logger.Error("failed to get benchmark", zap.Error(err))
		return nil, err
	}
	if benchmark == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	var controlIDs []string
	for _, control := range benchmark.Controls {
		controlIDs = append(controlIDs, control)
	}
	for _, childBenchmarkID := range benchmark.Children {
		childControlIDs, err := h.getBenchmarkChildrenControls(childBenchmarkID)
		if err != nil {
			return nil, err
		}
		controlIDs = append(controlIDs, childControlIDs...)
	}
	return controlIDs, nil
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
	connectionIDs := httpserver.QueryArrayParam(ctx, "connection_id")
	if len(connectionIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "connection_id is required")
	}
	controlIDs := httpserver.QueryArrayParam(ctx, "control_id")

	jobParameters, describeJobs, err := h.getReEvaluateParams(benchmarkID, connectionIDs, controlIDs)
	if err != nil {
		return err
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

	var dependencyIDs []int64
	for _, describeJob := range describeJobs {
		daj, err := h.Scheduler.describe(describeJob.Connection, describeJob.ResourceType, false, false, false)
		if err != nil {
			h.Scheduler.logger.Error("failed to describe connection", zap.String("connection_id", describeJob.Connection.ID.String()), zap.Error(err))
			continue
		}
		dependencyIDs = append(dependencyIDs, int64(daj.ID))
	}

	err = h.DB.CreateJobSequencer(&model2.JobSequencer{
		DependencyList:    dependencyIDs,
		DependencySource:  model2.JobSequencerJobTypeDescribe,
		NextJob:           model2.JobSequencerJobTypeBenchmarkRunner,
		NextJobParameters: &jp,
		Status:            model2.JobSequencerWaitingForDependencies,
	})

	return ctx.NoContent(http.StatusOK)
}

// CheckReEvaluateComplianceJob godoc
//
//	@Summary		Get re-evaluates compliance job
//	@Description	Get re-evaluate job for the given connection and control
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Param			benchmark_id	path		string		true	"Benchmark ID"
//	@Param			connection_id	query		[]string	true	"Connection ID"
//	@Param			control_id		query		[]string	false	"Control ID"
//	@Success		200				{object}	api.JobSeqCheckResponse
//	@Router			/schedule/api/v1/compliance/re-evaluate/{benchmark_id} [get]
func (h HttpServer) CheckReEvaluateComplianceJob(ctx echo.Context) error {
	benchmarkID := ctx.Param("benchmark_id")
	connectionIDs := httpserver.QueryArrayParam(ctx, "connection_id")
	if len(connectionIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "connection_id is required")
	}
	controlIDs := httpserver.QueryArrayParam(ctx, "control_id")

	jobParameters, describeJobs, err := h.getReEvaluateParams(benchmarkID, connectionIDs, controlIDs)
	if err != nil {
		return err
	}

	var dependencyIDs []int64
	for _, describeJob := range describeJobs {
		daj, err := h.Scheduler.db.GetLastDescribeConnectionJob(describeJob.Connection.ID.String(), describeJob.ResourceType)
		if err != nil {
			h.Scheduler.logger.Error("failed to describe connection", zap.String("connection_id", describeJob.Connection.ID.String()), zap.Error(err))
			continue
		}
		dependencyIDs = append(dependencyIDs, int64(daj.ID))
	}

	jobs, err := h.Scheduler.db.ListJobSequencersOfTypeOfToday(model2.JobSequencerJobTypeDescribe, model2.JobSequencerJobTypeBenchmarkRunner)
	if err != nil {
		return err
	}

	var theJob *model2.JobSequencer
	for _, job := range jobs {
		var params model2.JobSequencerJobTypeBenchmarkRunnerParameters
		err := json.Unmarshal(job.NextJobParameters.Bytes, &params)
		if err != nil {
			h.Scheduler.logger.Error("failed to unmarshal job parameters", zap.Error(err))
			return err
		}

		fmt.Println(">>>", job)
		fmt.Println("<<<", params, dependencyIDs)
		fmt.Println("----", params.BenchmarkID, jobParameters.BenchmarkID)
		fmt.Println("----", params.ConnectionIDs, jobParameters.ConnectionIDs)
		fmt.Println("----", params.ControlIDs, jobParameters.ControlIDs)
		fmt.Println("----", job.DependencyList, dependencyIDs)

		if params.BenchmarkID == jobParameters.BenchmarkID &&
			utils.IncludesAll(params.ConnectionIDs, jobParameters.ConnectionIDs) &&
			utils.IncludesAll(params.ControlIDs, jobParameters.ControlIDs) &&
			utils.IncludesAll(job.DependencyList, dependencyIDs) {
			theJob = &job
			break
		}
	}

	if theJob == nil || theJob.Status == model2.JobSequencerFailed {
		fmt.Println("job not found/failed", theJob)
		return ctx.JSON(http.StatusOK, api.JobSeqCheckResponse{
			IsRunning: false,
		})
	}

	if theJob.Status == model2.JobSequencerWaitingForDependencies {
		fmt.Println("job waiting", theJob)
		return ctx.JSON(http.StatusOK, api.JobSeqCheckResponse{
			IsRunning: true,
		})
	}

	var nid []int64
	for _, m := range strings.Split(theJob.NextJobIDs, ",") {
		i, _ := strconv.ParseInt(m, 10, 64)
		nid = append(nid, i)
	}
	runnerJobs, err := h.Scheduler.db.ListRunnersWithID(nid)
	if err != nil {
		return err
	}
	for _, runner := range runnerJobs {
		if runner.Status != runner2.ComplianceRunnerSucceeded &&
			runner.Status != runner2.ComplianceRunnerFailed &&
			runner.Status != runner2.ComplianceRunnerTimeOut {
			fmt.Println("+++ job status", runner.Status)

			return ctx.JSON(http.StatusOK, api.JobSeqCheckResponse{
				IsRunning: true,
			})
		}
	}

	fmt.Println("job finished", theJob)
	return ctx.JSON(http.StatusOK, api.JobSeqCheckResponse{
		IsRunning: false,
	})
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
	jobID, err := h.Scheduler.scheduleAnalyticsJob(model2.AnalyticsJobTypeNormal, ctx.Request().Context())
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
			resourceCount, err := es.GetInventoryCountResponse(ctx.Request().Context(), h.Scheduler.es, strings.ToLower(job.ResourceType))
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

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}

	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
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
