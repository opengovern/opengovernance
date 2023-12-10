package describe

import (
	"encoding/json"
	"errors"
	"fmt"
	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	analyticsDb "github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db"
	model2 "github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	httpserver2 "github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	"github.com/kaytu-io/terraform-package/external/states/statefile"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/sony/sonyflake"
	"go.uber.org/zap"

	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"

	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	complianceapi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	onboardapi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"gorm.io/gorm"

	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/internal"
	"github.com/labstack/echo/v4"
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
	v1.GET("/compliance/status/:benchmark_id", httpserver2.AuthorizeHandler(h.GetComplianceBenchmarkStatus, apiAuth.AdminRole))
	v1.PUT("/analytics/trigger", httpserver2.AuthorizeHandler(h.TriggerAnalyticsJob, apiAuth.InternalRole))
	v1.GET("/analytics/job/:job_id", httpserver2.AuthorizeHandler(h.GetAnalyticsJob, apiAuth.InternalRole))
	v1.GET("/describe/status/:resource_type", httpserver2.AuthorizeHandler(h.GetDescribeStatus, apiAuth.InternalRole))
	v1.GET("/describe/connection/status", httpserver2.AuthorizeHandler(h.GetConnectionDescribeStatus, apiAuth.InternalRole))
	v1.GET("/describe/pending/connections", httpserver2.AuthorizeHandler(h.ListAllPendingConnection, apiAuth.InternalRole))
	v1.GET("/describe/all/jobs/state", httpserver2.AuthorizeHandler(h.GetDescribeAllJobsStatus, apiAuth.InternalRole))

	v1.GET("/discovery/resourcetypes/list", httpserver2.AuthorizeHandler(h.GetDiscoveryResourceTypeList, apiAuth.ViewerRole))
	v1.GET("/discovery/resourcetypes/:resource_type/accounts", httpserver2.AuthorizeHandler(h.GetDiscoveryResourceTypeAccounts, apiAuth.ViewerRole))
	v1.GET("/jobs", httpserver2.AuthorizeHandler(h.ListJobs, apiAuth.ViewerRole))
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
}

// ListJobs godoc
//
//	@Summary	Lists all jobs
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		limit	query	int	false	"Limit"
//	@Param		hours	query	int	false	"Hours"
//	@Produce	json
//	@Success	200	{object}	api.ListJobsResponse
//	@Router		/schedule/api/v1/jobs [get]
func (h HttpServer) ListJobs(ctx echo.Context) error {
	hoursStr := ctx.QueryParam("hours")
	limitStr := ctx.QueryParam("limit")
	typeFilter := ctx.QueryParam("type")
	statusFilter := api.JobStatus(ctx.QueryParam("status"))

	var queryStatusFilter []string
	switch statusFilter {
	case api.JobStatus_Created:
		queryStatusFilter = []string{"CREATED"}
	case api.JobStatus_Queued:
		queryStatusFilter = []string{"QUEUED"}
	case api.JobStatus_InProgress:
		queryStatusFilter = []string{"IN_PROGRESS", "RUNNERS_IN_PROGRESS", "SUMMARIZER_IN_PROGRESS"}
	case api.JobStatus_Successful:
		queryStatusFilter = []string{"COMPLETED", "SUCCESSFUL", "SUCCEEDED"}
	case api.JobStatus_Failure:
		queryStatusFilter = []string{"COMPLETED_WITH_FAILURE", "FAILED"}
	case api.JobStatus_Timeout:
		queryStatusFilter = []string{"TIMEOUT", "TIMEDOUT"}
	default:
		queryStatusFilter = []string{}
	}

	hours := 24
	limit := 500

	if len(hoursStr) > 0 {
		n, err := strconv.Atoi(hoursStr)
		if err != nil {
			return err
		}
		hours = n
	}

	if len(limitStr) > 0 {
		n, err := strconv.Atoi(limitStr)
		if err != nil {
			return err
		}
		limit = n
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

	describeJobs, err := h.DB.ListAllJobs(limit, typeFilter, queryStatusFilter)
	if err != nil {
		return err
	}
	for _, job := range describeJobs {
		var status api.JobStatus
		switch job.Status {
		case "CREATED":
			status = api.JobStatus_Created
		case "QUEUED":
			status = api.JobStatus_Queued
		case "IN_PROGRESS", "RUNNERS_IN_PROGRESS", "SUMMARIZER_IN_PROGRESS":
			status = api.JobStatus_InProgress
		case "COMPLETED", "SUCCESSFUL", "SUCCEEDED":
			status = api.JobStatus_Successful
		case "COMPLETED_WITH_FAILURE", "FAILED":
			status = api.JobStatus_Failure
		case "TIMEOUT", "TIMEDOUT":
			status = api.JobStatus_Timeout
		}

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
			Status:                 status,
			FailureReason:          job.FailureMessage,
		})
	}

	var jobSummaries []api.JobSummary
	summaries, err := h.DB.GetAllJobSummary(hours)
	if err != nil {
		return err
	}
	for _, summary := range summaries {
		var status api.JobStatus
		switch summary.Status {
		case "CREATED":
			status = api.JobStatus_Created
		case "QUEUED":
			status = api.JobStatus_Queued
		case "IN_PROGRESS", "RUNNERS_IN_PROGRESS", "SUMMARIZER_IN_PROGRESS":
			status = api.JobStatus_InProgress
		case "COMPLETED", "SUCCESSFUL", "SUCCEEDED":
			status = api.JobStatus_Successful
		case "COMPLETED_WITH_FAILURE", "FAILED":
			status = api.JobStatus_Failure
		case "TIMEOUT", "TIMEDOUT":
			status = api.JobStatus_Timeout
		}

		jobSummaries = append(jobSummaries, api.JobSummary{
			Type:   api.JobType(summary.JobType),
			Status: status,
			Count:  summary.Count,
		})
	}

	return ctx.JSON(http.StatusOK, api.ListJobsResponse{
		Jobs:      jobs,
		Summaries: jobSummaries,
	})
}

var awsResourceTypeReg, _ = regexp.Compile("aws::[a-z0-9-_/]+::[a-z0-9-_/]+")
var azureResourceTypeReg, _ = regexp.Compile("microsoft.[a-z0-9-_/]+")

var awsTableReg, _ = regexp.Compile("aws_[a-z0-9_]+")
var azureTableReg, _ = regexp.Compile("azure_[a-z0-9_]+")

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

func extractResourceTypes(query string) []string {
	var result []string

	awsTables := awsTableReg.FindAllString(query, -1)
	azureTables := azureTableReg.FindAllString(query, -1)
	result = append(result, awsTables...)
	result = append(result, azureTables...)

	awsTables = awsResourceTypeReg.FindAllString(query, -1)
	for _, table := range awsTables {
		resourceType := getResourceTypeFromTableName(table, source.CloudAWS)
		if resourceType == "" {
			resourceType = table
		}
		result = append(result, resourceType)
	}

	azureTables = azureResourceTypeReg.FindAllString(query, -1)
	for _, table := range azureTables {
		resourceType := getResourceTypeFromTableName(table, source.CloudAzure)
		if resourceType == "" {
			resourceType = table
		}
		result = append(result, resourceType)
	}

	return result
}

func UniqueArray(arr []string) []string {
	m := map[string]interface{}{}
	for _, item := range arr {
		m[item] = struct{}{}
	}
	var resp []string
	for k, _ := range m {
		resp = append(resp, k)
	}
	return resp
}

func (h HttpServer) extractBenchmarkResourceTypes(ctx *httpclient.Context, benchmarkID string) ([]string, error) {
	benchmark, err := h.Scheduler.complianceClient.GetBenchmark(ctx, benchmarkID)
	if err != nil {
		return nil, err
	}

	var response []string
	for _, child := range benchmark.Children {
		rts, err := h.extractBenchmarkResourceTypes(ctx, child)
		if err != nil {
			return nil, err
		}
		response = append(response, rts...)
	}

	for _, controlID := range benchmark.Controls {
		control, err := h.Scheduler.complianceClient.GetControl(ctx, controlID)
		if err != nil {
			return nil, err
		}

		if control.ManualVerification || control.QueryID == nil {
			continue
		}

		query, err := h.Scheduler.complianceClient.GetQuery(ctx, *control.QueryID)
		if err != nil {
			return nil, err
		}

		response = append(response, extractResourceTypes(query.QueryToExecute)...)
	}

	return response, nil
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
	assetMetrics, err := h.Scheduler.inventoryClient.ListAnalyticsMetrics(httpclient.FromEchoContext(ctx), analyticsDb.MetricTypeAssets)
	if err != nil {
		return err
	}

	spendMetrics, err := h.Scheduler.inventoryClient.ListAnalyticsMetrics(httpclient.FromEchoContext(ctx), analyticsDb.MetricTypeSpend)
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

	var resourceTypes []string
	for _, metric := range append(assetMetrics, spendMetrics...) {
		rts := extractResourceTypes(metric.Query)
		resourceTypes = append(resourceTypes, rts...)
	}

	for _, ins := range insights {
		rts := extractResourceTypes(ins.Query.QueryToExecute)
		resourceTypes = append(resourceTypes, rts...)
	}

	for _, bench := range benchmarks {
		rts, err := h.extractBenchmarkResourceTypes(httpclient.FromEchoContext(ctx), bench.ID)
		if err != nil {
			return err
		}

		rts = UniqueArray(rts)
		resourceTypes = append(resourceTypes, rts...)
	}

	var result api.ListDiscoveryResourceTypes
	awsResourceTypes, azureResourceTypes := aws.ListResourceTypes(), azure.ListResourceTypes()
	for _, resourceType := range resourceTypes {
		found := false
		resourceType = strings.ToLower(resourceType)
		if strings.HasPrefix(resourceType, "aws") {
			for _, awsResourceType := range awsResourceTypes {
				if strings.ToLower(awsResourceType) == resourceType {
					found = true
					resourceType = awsResourceType
					break
				}
			}
			result.AWSResourceTypes = append(result.AWSResourceTypes, resourceType)
		} else if strings.HasPrefix(resourceType, "microsoft") {
			for _, azureResourceType := range azureResourceTypes {
				if strings.ToLower(azureResourceType) == resourceType {
					found = true
					resourceType = azureResourceType
					break
				}
			}
			result.AzureResourceTypes = append(result.AzureResourceTypes, resourceType)
		} else if strings.HasPrefix(resourceType, "azure") {
			result.AzureResourceTypes = append(result.AzureResourceTypes, resourceType)
		} else {
			return errors.New("invalid resource type:" + resourceType)
		}

		if !found {
			h.Scheduler.logger.Error("resource type " + resourceType + " not found!")
		}
	}
	result.AzureResourceTypes = append(result.AzureResourceTypes, "Microsoft.CostManagement/CostByResourceType")
	result.AWSResourceTypes = append(result.AWSResourceTypes, "AWS::CostExplorer::ByServiceDaily")

	result.AWSResourceTypes = UniqueArray(result.AWSResourceTypes)
	result.AzureResourceTypes = UniqueArray(result.AzureResourceTypes)

	return ctx.JSON(http.StatusOK, result)
}

// GetDiscoveryResourceTypeAccounts godoc
//
//	@Summary	List all cloud accounts which will have the resource type enabled in discovery
//	@Security	BearerToken
//	@Tags		scheduler
//	@Produce	json
//	@Success	200	{object}	api.ListJobsResponse
//	@Router		/schedule/api/v1/discovery/resourcetypes/{resource_type}/accounts [get]
func (h HttpServer) GetDiscoveryResourceTypeAccounts(ctx echo.Context) error {
	return ctx.NoContent(http.StatusOK)
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
		DependencySource: string(model2.JobSequencerJobTypeDescribe),
		NextJob:          string(model2.JobSequencerJobTypeAnalytics),
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

	err := h.Scheduler.CheckWorkspaceResourceLimit()
	if err != nil {
		h.Scheduler.logger.Error("failed to get limits", zap.String("spot", "CheckWorkspaceResourceLimit"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		if err == ErrMaxResourceCountExceeded {
			return ctx.JSON(http.StatusNotAcceptable, api.ErrorResponse{Message: err.Error()})
		}
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: err.Error()})
	}

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
//	@Param			benchmark_id	path	string	true	"Benchmark ID"
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

	lastJob, err := h.Scheduler.db.GetLastComplianceJob(benchmark.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if lastJob != nil && (lastJob.Status == model2.ComplianceJobRunnersInProgress ||
		lastJob.Status == model2.ComplianceJobSummarizerInProgress ||
		lastJob.Status == model2.ComplianceJobCreated) {
		return echo.NewHTTPError(http.StatusConflict, "compliance job is already running")
	}

	_, err = h.Scheduler.complianceScheduler.CreateComplianceReportJobs(benchmarkID)
	if err != nil {
		return fmt.Errorf("error while creating compliance job: %v", err)
	}
	return ctx.JSON(http.StatusOK, "")
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
