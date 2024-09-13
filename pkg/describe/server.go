package describe

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgtype"
	runner2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	apiAuth "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/labstack/echo/v4"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-aws-describer/aws"
	awsDescriberLocal "github.com/kaytu-io/kaytu-aws-describer/local"
	awsSteampipe "github.com/kaytu-io/kaytu-aws-describer/pkg/steampipe"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	analyticsapi "github.com/kaytu-io/kaytu-engine/pkg/analytics/api"
	complianceapi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db"
	model2 "github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	onboardapi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
	"gorm.io/gorm"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type HttpServer struct {
	Address       string
	DB            db.Database
	Scheduler     *Scheduler
	onboardClient onboardClient.OnboardServiceClient
	kubeClient    k8sclient.Client
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
	v1.PUT("/compliance/trigger", httpserver.AuthorizeHandler(h.TriggerConnectionsComplianceJobs, apiAuth.AdminRole))
	v1.PUT("/compliance/trigger/:benchmark_id", httpserver.AuthorizeHandler(h.TriggerConnectionsComplianceJob, apiAuth.AdminRole))
	v1.PUT("/compliance/trigger/:benchmark_id/summary", httpserver.AuthorizeHandler(h.TriggerConnectionsComplianceJobSummary, apiAuth.AdminRole))
	v1.GET("/compliance/re-evaluate/:benchmark_id", httpserver.AuthorizeHandler(h.CheckReEvaluateComplianceJob, apiAuth.AdminRole))
	v1.PUT("/compliance/re-evaluate/:benchmark_id", httpserver.AuthorizeHandler(h.ReEvaluateComplianceJob, apiAuth.AdminRole))
	v1.GET("/compliance/status/:benchmark_id", httpserver.AuthorizeHandler(h.GetComplianceBenchmarkStatus, apiAuth.AdminRole))
	v1.PUT("/analytics/trigger", httpserver.AuthorizeHandler(h.TriggerAnalyticsJob, apiAuth.AdminRole))
	v1.GET("/analytics/job/:job_id", httpserver.AuthorizeHandler(h.GetAnalyticsJob, apiAuth.InternalRole))
	v1.GET("/describe/status/:resource_type", httpserver.AuthorizeHandler(h.GetDescribeStatus, apiAuth.InternalRole))
	v1.GET("/describe/connection/status", httpserver.AuthorizeHandler(h.GetConnectionDescribeStatus, apiAuth.InternalRole))
	v1.GET("/describe/pending/connections", httpserver.AuthorizeHandler(h.ListAllPendingConnection, apiAuth.InternalRole))
	v1.GET("/describe/all/jobs/state", httpserver.AuthorizeHandler(h.GetDescribeAllJobsStatus, apiAuth.InternalRole))

	v1.GET("/discovery/resourcetypes/list", httpserver.AuthorizeHandler(h.GetDiscoveryResourceTypeList, apiAuth.ViewerRole))
	v1.POST("/jobs", httpserver.AuthorizeHandler(h.ListJobs, apiAuth.ViewerRole))
	v1.GET("/jobs/bydate", httpserver.AuthorizeHandler(h.CountJobsByDate, apiAuth.InternalRole))

	v2 := e.Group("/api/v2")
	v2.POST("/jobs/discovery/connections/:connection-id", httpserver.AuthorizeHandler(h.GetDescribeJobsHistory, apiAuth.ViewerRole))
	v2.POST("/jobs/compliance/connections/:connection-id", httpserver.AuthorizeHandler(h.GetComplianceJobsHistory, apiAuth.ViewerRole))
	v2.POST("/jobs/discovery/connections", httpserver.AuthorizeHandler(h.GetDescribeJobsHistory, apiAuth.ViewerRole))
	v2.POST("/jobs/compliance/connections", httpserver.AuthorizeHandler(h.GetComplianceJobsHistory, apiAuth.ViewerRole))
	v2.POST("/compliance/benchmark/:benchmark-id/run", httpserver.AuthorizeHandler(h.RunBenchmarkById, apiAuth.AdminRole))
	v2.POST("/compliance/run", httpserver.AuthorizeHandler(h.RunBenchmark, apiAuth.AdminRole))
	v2.POST("/discovery/run", httpserver.AuthorizeHandler(h.RunDiscovery, apiAuth.AdminRole))
	v2.GET("/job/discovery/:job-id", httpserver.AuthorizeHandler(h.GetDescribeJobStatus, apiAuth.ViewerRole))
	v2.GET("/job/compliance/:job-id", httpserver.AuthorizeHandler(h.GetComplianceJobStatus, apiAuth.ViewerRole))
	v2.GET("/job/analytics/:job-id", httpserver.AuthorizeHandler(h.GetAnalyticsJobStatus, apiAuth.ViewerRole))
	v2.GET("/jobs/discovery", httpserver.AuthorizeHandler(h.ListDescribeJobs, apiAuth.ViewerRole))
	v2.GET("/jobs/compliance", httpserver.AuthorizeHandler(h.ListComplianceJobs, apiAuth.ViewerRole))
	v2.GET("/jobs/analytics", httpserver.AuthorizeHandler(h.ListAnalyticsJobs, apiAuth.ViewerRole))
	v2.PUT("/jobs/cancel", httpserver.AuthorizeHandler(h.CancelJob, apiAuth.AdminRole))
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
//	@Param			force_full		query	bool		false	"Force full discovery"
//	@Param			resource_type	query	[]string	false	"Resource Type"
//	@Param			cost_discovery	query	bool		false	"Cost discovery"
//	@Router			/schedule/api/v1/describe/trigger/{connection_id} [put]
func (h HttpServer) TriggerPerConnectionDescribeJob(ctx echo.Context) error {
	connectionID := ctx.Param("connection_id")
	forceFull := ctx.QueryParam("force_full") == "true"
	costDiscovery := ctx.QueryParam("cost_discovery") == "true"
	costFullDiscovery := ctx.QueryParam("cost_full_discovery") == "true"

	ctx2 := &httpclient.Context{UserRole: apiAuth.InternalRole}
	ctx2.Ctx = ctx.Request().Context()

	var srcs []onboardapi.Connection
	if connectionID == "all" {
		var err error
		srcs, err = h.Scheduler.onboardClient.ListSources(ctx2, nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
	} else {
		src, err := h.Scheduler.onboardClient.GetSource(ctx2, connectionID)
		if err != nil || src == nil {
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			} else {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid connection id")
			}
		}
		srcs = []onboardapi.Connection{*src}
	}

	dependencyIDs := make([]int64, 0)
	for _, src := range srcs {
		resourceTypes := ctx.QueryParams()["resource_type"]

		if resourceTypes == nil {
			if costDiscovery {
				switch src.Connector {
				case source.CloudAWS:
					resourceTypes = []string{"AWS::CostExplorer::ByServiceDaily"}
				case source.CloudAzure:
					resourceTypes = []string{"Microsoft.CostManagement/CostByResourceType"}
				}
			} else {
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
		}

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
			daj, err := h.Scheduler.describe(src, resourceType, false, costFullDiscovery, false)
			if err == ErrJobInProgress {
				return echo.NewHTTPError(http.StatusConflict, err.Error())
			}
			if err != nil {
				return err
			}
			dependencyIDs = append(dependencyIDs, int64(daj.ID))
		}
	}

	err := h.DB.CreateJobSequencer(&model2.JobSequencer{
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

	_, err = h.Scheduler.complianceScheduler.CreateComplianceReportJobs(benchmarkID, lastJob, connectionIDs, true)
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
//	@Param			benchmark_id	query	[]string	true	"Benchmark IDs leave empty for everything"
//	@Param			connection_id	query	[]string	false	"Connection IDs leave empty for default (enabled connections)"
//	@Router			/schedule/api/v1/compliance/trigger [put]
func (h HttpServer) TriggerConnectionsComplianceJobs(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}
	benchmarkIDs := httpserver.QueryArrayParam(ctx, "benchmark_id")

	connectionIDs := httpserver.QueryArrayParam(ctx, "connection_id")

	var benchmarks []complianceapi.Benchmark
	var err error
	if len(benchmarkIDs) == 0 {
		benchmarks, err = h.Scheduler.complianceClient.ListBenchmarks(clientCtx, nil)
		if err != nil {
			return fmt.Errorf("error while getting benchmarks: %v", err)
		}
	} else {
		for _, benchmarkID := range benchmarkIDs {
			benchmark, err := h.Scheduler.complianceClient.GetBenchmark(clientCtx, benchmarkID)
			if err != nil {
				return fmt.Errorf("error while getting benchmarks: %v", err)
			}
			if benchmark == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark %s not found", benchmark.ID))
			}
			benchmarks = append(benchmarks, *benchmark)
		}
	}

	for _, benchmark := range benchmarks {
		lastJob, err := h.Scheduler.db.GetLastComplianceJob(benchmark.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if lastJob != nil && (lastJob.Status == model2.ComplianceJobRunnersInProgress ||
			lastJob.Status == model2.ComplianceJobSummarizerInProgress ||
			lastJob.Status == model2.ComplianceJobCreated) {
			return echo.NewHTTPError(http.StatusConflict, "compliance job is already running")
		}

		_, err = h.Scheduler.complianceScheduler.CreateComplianceReportJobs(benchmark.ID, lastJob, connectionIDs, true)
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
//	@Param			benchmark_id	path	string	true	"Benchmark ID use 'all' for everything"
//	@Router			/schedule/api/v1/compliance/trigger/{benchmark_id}/summary [put]
func (h HttpServer) TriggerConnectionsComplianceJobSummary(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}
	benchmarkID := ctx.Param("benchmark_id")

	var benchmarks []complianceapi.Benchmark
	var err error
	if benchmarkID == "all" {
		benchmarks, err = h.Scheduler.complianceClient.ListBenchmarks(clientCtx, nil)
		if err != nil {
			return fmt.Errorf("error while getting benchmarks: %v", err)
		}
	} else {
		benchmark, err := h.Scheduler.complianceClient.GetBenchmark(clientCtx, benchmarkID)
		if err != nil {
			return fmt.Errorf("error while getting benchmarks: %v", err)
		}
		if benchmark == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark %s not found", benchmark.ID))
		}
		benchmarks = append(benchmarks, *benchmark)
	}

	for _, benchmark := range benchmarks {
		err = h.Scheduler.complianceScheduler.CreateSummarizer(benchmark.ID, nil, model2.ComplianceTriggerTypeManual)
		if err != nil {
			return fmt.Errorf("error while creating compliance job summarizer: %v", err)
		}
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

	h.Scheduler.logger.Info("re-evaluating compliance job", zap.Any("job_parameters", jobParameters))

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
	if err != nil {
		h.Scheduler.logger.Error("failed to create job sequencer", zap.Error(err))
		return err
	}

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

// TriggerAnalyticsJob godoc
//
//	@Summary		TriggerAnalyticsJob
//	@Description	Triggers an analytics job to run immediately
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Router			/schedule/api/v1/analytics/trigger [put]
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

// GetDescribeJobsHistory godoc
//
//	@Summary	Get describe jobs history for give connection
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetDescribeJobsHistoryRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetDescribeJobsHistoryResponse
//	@Router		/schedule/api/v2/jobs/discovery/connections/{connection-id} [post]
func (h HttpServer) GetDescribeJobsHistory(ctx echo.Context) error {
	connectionId := ctx.Param("connection-id")

	var request api.GetDescribeJobsHistoryRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var jobsResults []api.GetDescribeJobsHistoryResponse

	jobs, err := h.DB.ListDescribeJobsByFilters([]string{connectionId}, request.ResourceType,
		request.DiscoveryType, request.JobStatus, request.StartTime, request.EndTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	for _, j := range jobs {
		jobsResults = append(jobsResults, api.GetDescribeJobsHistoryResponse{
			JobId:         j.ID,
			DiscoveryType: string(j.DiscoveryType),
			ResourceType:  j.ResourceType,
			JobStatus:     j.Status,
			DateTime:      j.UpdatedAt,
		})
	}
	if request.SortBy != nil {
		switch strings.ToLower(*request.SortBy) {
		case "id":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		case "datetime":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].DateTime.Before(jobsResults[j].DateTime)
			})
		case "discoverytype":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].DiscoveryType < jobsResults[j].DiscoveryType
			})
		case "resourcetype":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].ResourceType < jobsResults[j].ResourceType
			})
		case "jobstatus":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobStatus < jobsResults[j].JobStatus
			})
		default:
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		}
	} else {
		sort.Slice(jobsResults, func(i, j int) bool {
			return jobsResults[i].JobId < jobsResults[j].JobId
		})
	}
	if request.PerPage != nil {
		if request.Cursor == nil {
			jobsResults = utils.Paginate(1, *request.PerPage, jobsResults)
		} else {
			jobsResults = utils.Paginate(*request.Cursor, *request.PerPage, jobsResults)
		}
	}

	return ctx.JSON(http.StatusOK, jobsResults)
}

// GetComplianceJobsHistory godoc
//
//	@Summary	Get compliance jobs history for give connection
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetComplianceJobsHistoryRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetComplianceJobsHistoryResponse
//	@Router		/schedule/api/v2/jobs/compliance/connections/{connection-id} [post]
func (h HttpServer) GetComplianceJobsHistory(ctx echo.Context) error {
	var request api.GetComplianceJobsHistoryRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	connectionId := ctx.Param("connection-id")

	jobs, err := h.DB.ListComplianceJobsByFilters([]string{connectionId}, request.BenchmarkId, request.JobStatus, request.StartTime, request.EndTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jobsResults []api.GetComplianceJobsHistoryResponse
	for _, j := range jobs {
		jobsResults = append(jobsResults, api.GetComplianceJobsHistoryResponse{
			JobId:       j.ID,
			BenchmarkId: j.BenchmarkID,
			JobStatus:   j.Status.ToApi(),
			DateTime:    j.UpdatedAt,
		})
	}
	if request.SortBy != nil {
		switch strings.ToLower(*request.SortBy) {
		case "id":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		case "datetime":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].DateTime.Before(jobsResults[j].DateTime)
			})
		case "benchmarkid":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].BenchmarkId < jobsResults[j].BenchmarkId
			})
		case "jobstatus":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobStatus < jobsResults[j].JobStatus
			})
		default:
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		}
	} else {
		sort.Slice(jobsResults, func(i, j int) bool {
			return jobsResults[i].JobId < jobsResults[j].JobId
		})
	}
	if request.PerPage != nil {
		if request.Cursor == nil {
			jobsResults = utils.Paginate(1, *request.PerPage, jobsResults)
		} else {
			jobsResults = utils.Paginate(*request.Cursor, *request.PerPage, jobsResults)
		}
	}

	return ctx.JSON(http.StatusOK, jobsResults)
}

// RunBenchmarkById godoc
//
//	@Summary		Triggers compliance job by benchmark id
//	@Description	Triggers a compliance job to run immediately for the given benchmark
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Param			benchmark_id	path	string		true	"Benchmark ID"
//	@Param			request	body	api.RunBenchmarkByIdRequest	true	""
//	@Router			/schedule/api/v1/compliance/benchmark/{benchmark-id}/run [post]
func (h HttpServer) RunBenchmarkById(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	benchmarkID := ctx.Param("benchmark-id")

	var request api.RunBenchmarkByIdRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if len(request.ConnectionInfo) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "please provide at least one connection info")
	}

	var connections []onboardapi.Connection
	for _, info := range request.ConnectionInfo {
		if info.ConnectionId != nil {
			connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, *info.ConnectionId)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			if connection != nil {
				connections = append(connections, *connection)
			}
			continue
		}
		connectionsTmp, err := h.Scheduler.onboardClient.GetSourceByFilters(clientCtx,
			onboardapi.GetSourceByFiltersRequest{
				Connector:         info.Connector,
				ProviderNameRegex: info.ProviderNameRegex,
				ProviderIdRegex:   info.ProviderIdRegex,
			})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		connections = append(connections, connectionsTmp...)
	}

	var connectionInfo []api.IntegrationInfo
	var connectionIDs []string
	for _, c := range connections {
		connectionInfo = append(connectionInfo, api.IntegrationInfo{
			IntegrationTracker: c.ID.String(),
			Integration:        c.Connector.String(),
			IDName:             c.ConnectionName,
			ID:                 c.ConnectionID,
		})
		connectionIDs = append(connectionIDs, c.ID.String())
	}

	benchmark, err := h.Scheduler.complianceClient.GetBenchmark(&httpclient.Context{UserRole: apiAuth.InternalRole}, benchmarkID)
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

	jobId, err := h.Scheduler.complianceScheduler.CreateComplianceReportJobs(benchmarkID, lastJob, connectionIDs, true)
	if err != nil {
		return fmt.Errorf("error while creating compliance job: %v", err)
	}

	return ctx.JSON(http.StatusOK, api.RunBenchmarkResponse{
		JobId:           jobId,
		BenchmarkId:     benchmark.ID,
		IntegrationInfo: connectionInfo,
	})
}

// RunBenchmark godoc
//
//	@Summary		Triggers compliance job
//	@Description	Triggers a compliance job to run immediately for the given benchmark
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200 {object} []api.RunBenchmarkResponse
//	@Param			request	body	api.RunBenchmarkRequest		true	""
//	@Router			/schedule/api/v1/compliance/benchmark/run [post]
func (h HttpServer) RunBenchmark(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	var request api.RunBenchmarkRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if len(request.IntegrationInfo) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "please provide at least one connection info")
	}

	var connections []onboardapi.Connection
	for _, info := range request.IntegrationInfo {
		if info.IntegrationTracker != nil {
			connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, *info.IntegrationTracker)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			if connection != nil {
				connections = append(connections, *connection)
			}
			continue
		}
		connectionsTmp, err := h.Scheduler.onboardClient.GetSourceByFilters(clientCtx,
			onboardapi.GetSourceByFiltersRequest{
				Connector:         info.Integration,
				ProviderNameRegex: info.IDName,
				ProviderIdRegex:   info.ID,
			})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		connections = append(connections, connectionsTmp...)
	}

	var connectionInfo []api.IntegrationInfo
	var connectionIDs []string
	for _, c := range connections {
		connectionInfo = append(connectionInfo, api.IntegrationInfo{
			IntegrationTracker: c.ID.String(),
			Integration:        c.Connector.String(),
			IDName:             c.ConnectionName,
			ID:                 c.ConnectionID,
		})
		connectionIDs = append(connectionIDs, c.ID.String())
	}

	var benchmarks []complianceapi.Benchmark
	var err error
	if len(request.BenchmarkIds) == 0 {
		benchmarks, err = h.Scheduler.complianceClient.ListBenchmarks(clientCtx, nil)
		if err != nil {
			return fmt.Errorf("error while getting benchmarks: %v", err)
		}
	} else {
		for _, benchmarkID := range request.BenchmarkIds {
			benchmark, err := h.Scheduler.complianceClient.GetBenchmark(clientCtx, benchmarkID)
			if err != nil {
				return fmt.Errorf("error while getting benchmarks: %v", err)
			}
			if benchmark == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("benchmark %s not found", benchmarkID))
			}
			benchmarks = append(benchmarks, *benchmark)
		}
	}

	var jobs []api.RunBenchmarkResponse
	for _, benchmark := range benchmarks {
		lastJob, err := h.Scheduler.db.GetLastComplianceJob(benchmark.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		jobId, err := h.Scheduler.complianceScheduler.CreateComplianceReportJobs(benchmark.ID, lastJob, connectionIDs, true)
		if err != nil {
			return fmt.Errorf("error while creating compliance job: %v", err)
		}

		jobs = append(jobs, api.RunBenchmarkResponse{
			JobId:           jobId,
			BenchmarkId:     benchmark.ID,
			IntegrationInfo: connectionInfo,
		})
	}

	return ctx.JSON(http.StatusOK, jobs)
}

// RunDiscovery godoc
//
//	@Summary		Run Discovery job
//	@Description	Triggers a discovery job to run immediately for the given resource types
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200 {object} []api.RunDiscoveryResponse
//	@Param			request	body	api.RunBenchmarkRequest		true	""
//	@Router			/schedule/api/v1/compliance/discovery/run [post]
func (h HttpServer) RunDiscovery(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	var request api.RunDiscoveryRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if len(request.IntegrationInfo) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "please provide at least one connection info")
	}

	var connections []onboardapi.Connection
	for _, info := range request.IntegrationInfo {
		if info.IntegrationTracker != nil {
			connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, *info.IntegrationTracker)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			if connection != nil {
				connections = append(connections, *connection)
			}
			continue
		}
		connectionsTmp, err := h.Scheduler.onboardClient.GetSourceByFilters(clientCtx,
			onboardapi.GetSourceByFiltersRequest{
				Connector:         info.Integration,
				ProviderNameRegex: info.IDName,
				ProviderIdRegex:   info.ID,
			})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		connections = append(connections, connectionsTmp...)
	}

	var jobs []api.RunDiscoveryResponse
	for _, connection := range connections {
		if !connection.IsEnabled() {
			continue
		}
		rtToDescribe := request.ResourceTypes

		if len(rtToDescribe) == 0 {
			switch connection.Connector {
			case source.CloudAWS:
				if request.ForceFull {
					rtToDescribe = aws.ListResourceTypes()
				} else {
					rtToDescribe = aws.ListFastDiscoveryResourceTypes()
				}
			case source.CloudAzure:
				if request.ForceFull {
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

			var status, failureReason string
			job, err := h.Scheduler.describe(connection, resourceType, false, false, false)
			if err != nil {
				if err.Error() == "job already in progress" {
					tmpJob, err := h.Scheduler.db.GetLastDescribeConnectionJob(connection.ID.String(), resourceType)
					if err != nil {
						h.Scheduler.logger.Error("failed to get last describe job", zap.String("resource_type", resourceType), zap.String("connection_id", connection.ID.String()), zap.Error(err))
					}
					h.Scheduler.logger.Error("failed to describe connection", zap.String("connection_id", connection.ID.String()), zap.Error(err))
					status = "FAILED"
					failureReason = fmt.Sprintf("job already in progress: %v", tmpJob.ID)
				} else {
					failureReason = err.Error()
				}
			}

			var jobId uint
			if job == nil {
				status = "FAILED"
			} else {
				jobId = job.ID
				status = string(job.Status)
			}
			jobs = append(jobs, api.RunDiscoveryResponse{
				JobId:         jobId,
				ResourceType:  resourceType,
				Status:        status,
				FailureReason: failureReason,
				IntegrationInfo: api.IntegrationInfo{
					IntegrationTracker: connection.ID.String(),
					Integration:        connection.Connector.String(),
					ID:                 connection.ConnectionID,
					IDName:             connection.ConnectionName,
				},
			})
		}
	}
	return ctx.JSON(http.StatusOK, jobs)
}

// GetDescribeJobStatus godoc
//
//	@Summary	Get describe job status by job id
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetDescribeJobsHistoryRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	api.GetDescribeJobStatusResponse
//	@Router		/schedule/api/v2/jobs/discovery/{job-id} [post]
func (h HttpServer) GetDescribeJobStatus(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	jobId := ctx.Param("job-id")

	j, err := h.DB.GetDescribeJobById(jobId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, j.ConnectionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	jobsResult := api.GetDescribeJobStatusResponse{
		JobId: j.ID,
		IntegrationInfo: api.IntegrationInfo{
			IntegrationTracker: connection.ID.String(),
			Integration:        connection.Connector.String(),
			ID:                 connection.ConnectionID,
			IDName:             connection.ConnectionName,
		},
		DiscoveryType: string(j.DiscoveryType),
		ResourceType:  j.ResourceType,
		JobStatus:     string(j.Status),
		CreatedAt:     j.CreatedAt,
		UpdatedAt:     j.UpdatedAt,
	}

	return ctx.JSON(http.StatusOK, jobsResult)
}

// GetComplianceJobStatus godoc
//
//	@Summary	Get compliance job status by job id
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetDescribeJobsHistoryRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetDescribeJobsHistoryResponse
//	@Router		/schedule/api/v2/jobs/discovery/connections/{connection-id} [post]
func (h HttpServer) GetComplianceJobStatus(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	jobIdString := ctx.Param("job-id")
	jobId, err := strconv.ParseUint(jobIdString, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	j, err := h.DB.GetComplianceJobByID(uint(jobId))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var connectionInfos []api.IntegrationInfo
	for _, cid := range j.ConnectionIDs {
		connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, cid)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		connectionInfos = append(connectionInfos, api.IntegrationInfo{
			IntegrationTracker: connection.ID.String(),
			Integration:        connection.Connector.String(),
			ID:                 connection.ConnectionID,
			IDName:             connection.ConnectionName,
		})
	}

	jobsResult := api.GetComplianceJobStatusResponse{
		JobId:           j.ID,
		IntegrationInfo: connectionInfos,
		BenchmarkId:     j.BenchmarkID,
		JobStatus:       string(j.Status),
		CreatedAt:       j.CreatedAt,
		UpdatedAt:       j.UpdatedAt,
	}

	return ctx.JSON(http.StatusOK, jobsResult)
}

// GetAnalyticsJobStatus godoc
//
//	@Summary	Get analytics job status by job id
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetDescribeJobsHistoryRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetDescribeJobsHistoryResponse
//	@Router		/schedule/api/v2/jobs/discovery/connections/{connection-id} [post]
func (h HttpServer) GetAnalyticsJobStatus(ctx echo.Context) error {

	jobIdString := ctx.Param("job-id")
	jobId, err := strconv.ParseUint(jobIdString, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	j, err := h.DB.GetAnalyticsJobByID(uint(jobId))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jobsResult := api.GetAnalyticsJobStatusResponse{
		JobId:     j.ID,
		JobStatus: string(j.Status),
		CreatedAt: j.CreatedAt,
		UpdatedAt: j.UpdatedAt,
	}

	return ctx.JSON(http.StatusOK, jobsResult)
}

// ListDescribeJobs godoc
//
//	@Summary	Get describe jobs history for give connection
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetDescribeJobsHistoryRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetDescribeJobsHistoryResponse
//	@Router		/schedule/api/v2/jobs/discovery/connections/{connection-id} [post]
func (h HttpServer) ListDescribeJobs(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	var request api.ListDescribeJobsRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var connections []onboardapi.Connection
	for _, info := range request.IntegrationInfo {
		if info.IntegrationTracker != nil {
			connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, *info.IntegrationTracker)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			if connection != nil {
				connections = append(connections, *connection)
			}
			continue
		}
		connectionsTmp, err := h.Scheduler.onboardClient.GetSourceByFilters(clientCtx,
			onboardapi.GetSourceByFiltersRequest{
				Connector:         info.Integration,
				ProviderNameRegex: info.IDName,
				ProviderIdRegex:   info.ID,
			})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		connections = append(connections, connectionsTmp...)
	}

	connectionInfo := make(map[string]api.IntegrationInfo)
	var connectionIDs []string
	for _, c := range connections {
		connectionInfo[c.ID.String()] = api.IntegrationInfo{
			IntegrationTracker: c.ID.String(),
			Integration:        c.Connector.String(),
			IDName:             c.ConnectionName,
			ID:                 c.ConnectionID,
		}
		connectionIDs = append(connectionIDs, c.ID.String())
	}

	var jobsResults []api.GetDescribeJobsHistoryResponse

	jobs, err := h.DB.ListDescribeJobsByFilters(connectionIDs, request.ResourceType,
		request.DiscoveryType, request.JobStatus, request.StartTime, request.EndTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	for _, j := range jobs {
		jobResult := api.GetDescribeJobsHistoryResponse{
			JobId:         j.ID,
			DiscoveryType: string(j.DiscoveryType),
			ResourceType:  j.ResourceType,
			JobStatus:     j.Status,
			DateTime:      j.UpdatedAt,
		}
		if info, ok := connectionInfo[j.ConnectionID]; ok {
			jobResult.IntegrationInfo = &info
		}
		jobsResults = append(jobsResults, jobResult)
	}
	if request.SortBy != nil {
		switch strings.ToLower(*request.SortBy) {
		case "id":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		case "datetime":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].DateTime.Before(jobsResults[j].DateTime)
			})
		case "discoverytype":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].DiscoveryType < jobsResults[j].DiscoveryType
			})
		case "resourcetype":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].ResourceType < jobsResults[j].ResourceType
			})
		case "jobstatus":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobStatus < jobsResults[j].JobStatus
			})
		default:
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		}
	} else {
		sort.Slice(jobsResults, func(i, j int) bool {
			return jobsResults[i].JobId < jobsResults[j].JobId
		})
	}
	if request.PerPage != nil {
		if request.Cursor == nil {
			jobsResults = utils.Paginate(1, *request.PerPage, jobsResults)
		} else {
			jobsResults = utils.Paginate(*request.Cursor, *request.PerPage, jobsResults)
		}
	}

	return ctx.JSON(http.StatusOK, jobsResults)
}

// ListComplianceJobs godoc
//
//	@Summary	Get compliance jobs history for give connection
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetComplianceJobsHistoryRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetComplianceJobsHistoryResponse
//	@Router		/schedule/api/v1/jobs [post]
func (h HttpServer) ListComplianceJobs(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	var request api.ListComplianceJobsRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var connections []onboardapi.Connection
	for _, info := range request.IntegrationInfo {
		if info.IntegrationTracker != nil {
			connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, *info.IntegrationTracker)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			if connection != nil {
				connections = append(connections, *connection)
			}
			continue
		}
		connectionsTmp, err := h.Scheduler.onboardClient.GetSourceByFilters(clientCtx,
			onboardapi.GetSourceByFiltersRequest{
				Connector:         info.Integration,
				ProviderNameRegex: info.IDName,
				ProviderIdRegex:   info.ID,
			})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		connections = append(connections, connectionsTmp...)
	}

	connectionInfo := make(map[string]api.IntegrationInfo)
	var connectionIDs []string
	for _, c := range connections {
		connectionInfo[c.ID.String()] = api.IntegrationInfo{
			IntegrationTracker: c.ID.String(),
			Integration:        c.Connector.String(),
			IDName:             c.ConnectionName,
			ID:                 c.ConnectionID,
		}
		connectionIDs = append(connectionIDs, c.ID.String())
	}

	jobs, err := h.DB.ListComplianceJobsByFilters(connectionIDs, request.BenchmarkId, request.JobStatus, request.StartTime, request.EndTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jobsResults []api.GetComplianceJobsHistoryResponse
	for _, j := range jobs {
		jobResult := api.GetComplianceJobsHistoryResponse{
			JobId:       j.ID,
			BenchmarkId: j.BenchmarkID,
			JobStatus:   j.Status.ToApi(),
			DateTime:    j.UpdatedAt,
		}
		for _, c := range j.ConnectionIDs {
			if info, ok := connectionInfo[c]; ok {
				jobResult.IntegrationInfo = append(jobResult.IntegrationInfo, info)
			}
		}
		jobsResults = append(jobsResults, jobResult)
	}
	if request.SortBy != nil {
		switch strings.ToLower(*request.SortBy) {
		case "id":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		case "datetime":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].DateTime.Before(jobsResults[j].DateTime)
			})
		case "benchmarkid":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].BenchmarkId < jobsResults[j].BenchmarkId
			})
		case "jobstatus":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobStatus < jobsResults[j].JobStatus
			})
		default:
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		}
	} else {
		sort.Slice(jobsResults, func(i, j int) bool {
			return jobsResults[i].JobId < jobsResults[j].JobId
		})
	}
	if request.PerPage != nil {
		if request.Cursor == nil {
			jobsResults = utils.Paginate(1, *request.PerPage, jobsResults)
		} else {
			jobsResults = utils.Paginate(*request.Cursor, *request.PerPage, jobsResults)
		}
	}

	return ctx.JSON(http.StatusOK, jobsResults)
}

// ListAnalyticsJobs godoc
//
//	@Summary	Get analytics jobs history for give connection
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetAnalyticsJobsHistoryRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetAnalyticsJobsHistoryResponse
//	@Router		/schedule/api/v1/jobs/analytics [post]
func (h HttpServer) ListAnalyticsJobs(ctx echo.Context) error {
	var request api.GetAnalyticsJobsHistoryRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	jobs, err := h.DB.ListAnalyticsJobsByFilter(request.Type, request.JobStatus, request.StartTime, request.EndTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jobsResults []api.GetAnalyticsJobsHistoryResponse
	for _, j := range jobs {
		jobsResults = append(jobsResults, api.GetAnalyticsJobsHistoryResponse{
			JobId:     j.ID,
			Type:      string(j.Type),
			JobStatus: j.Status,
			DateTime:  j.UpdatedAt,
		})
	}
	if request.SortBy != nil {
		switch strings.ToLower(*request.SortBy) {
		case "id":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		case "datetime":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].DateTime.Before(jobsResults[j].DateTime)
			})
		case "type":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].Type < jobsResults[j].Type
			})
		case "jobstatus":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobStatus < jobsResults[j].JobStatus
			})
		default:
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		}
	} else {
		sort.Slice(jobsResults, func(i, j int) bool {
			return jobsResults[i].JobId < jobsResults[j].JobId
		})
	}

	if request.PerPage != nil {
		if request.Cursor == nil {
			jobsResults = utils.Paginate(1, *request.PerPage, jobsResults)
		} else {
			jobsResults = utils.Paginate(*request.Cursor, *request.PerPage, jobsResults)
		}
	}

	return ctx.JSON(http.StatusOK, jobsResults)
}

// GetDescribeJobsHistoryByIntegration godoc
//
//	@Summary	Get describe jobs history for give connection
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetDescribeJobsHistoryRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetDescribeJobsHistoryResponse
//	@Router		/schedule/api/v2/jobs/discovery/connections [post]
func (h HttpServer) GetDescribeJobsHistoryByIntegration(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	var request api.GetDescribeJobsHistoryByIntegrationRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var connections []onboardapi.Connection
	for _, info := range request.IntegrationInfo {
		if info.IntegrationTracker != nil {
			connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, *info.IntegrationTracker)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			if connection != nil {
				connections = append(connections, *connection)
			}
			continue
		}
		connectionsTmp, err := h.Scheduler.onboardClient.GetSourceByFilters(clientCtx,
			onboardapi.GetSourceByFiltersRequest{
				Connector:         info.Integration,
				ProviderNameRegex: info.IDName,
				ProviderIdRegex:   info.ID,
			})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		connections = append(connections, connectionsTmp...)
	}

	connectionInfo := make(map[string]api.IntegrationInfo)
	for _, c := range connections {
		connectionInfo[c.ID.String()] = api.IntegrationInfo{
			IntegrationTracker: c.ID.String(),
			Integration:        c.Connector.String(),
			IDName:             c.ConnectionName,
			ID:                 c.ConnectionID,
		}
	}

	var jobsResults []api.GetDescribeJobsHistoryResponse

	for _, c := range connectionInfo {
		jobs, err := h.DB.ListDescribeJobsByFilters([]string{c.IntegrationTracker}, request.ResourceType,
			request.DiscoveryType, request.JobStatus, request.StartTime, request.EndTime)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		for _, j := range jobs {
			jobsResults = append(jobsResults, api.GetDescribeJobsHistoryResponse{
				JobId:           j.ID,
				DiscoveryType:   string(j.DiscoveryType),
				ResourceType:    j.ResourceType,
				JobStatus:       j.Status,
				DateTime:        j.UpdatedAt,
				IntegrationInfo: &c,
			})
		}
	}

	if request.SortBy != nil {
		switch strings.ToLower(*request.SortBy) {
		case "id":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		case "datetime":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].DateTime.Before(jobsResults[j].DateTime)
			})
		case "discoverytype":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].DiscoveryType < jobsResults[j].DiscoveryType
			})
		case "resourcetype":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].ResourceType < jobsResults[j].ResourceType
			})
		case "jobstatus":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobStatus < jobsResults[j].JobStatus
			})
		default:
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		}
	} else {
		sort.Slice(jobsResults, func(i, j int) bool {
			return jobsResults[i].JobId < jobsResults[j].JobId
		})
	}
	if request.PerPage != nil {
		if request.Cursor == nil {
			jobsResults = utils.Paginate(1, *request.PerPage, jobsResults)
		} else {
			jobsResults = utils.Paginate(*request.Cursor, *request.PerPage, jobsResults)
		}
	}

	return ctx.JSON(http.StatusOK, jobsResults)
}

// GetComplianceJobsHistoryByIntegration godoc
//
//	@Summary	Get compliance jobs history for give connection
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetComplianceJobsHistoryRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetComplianceJobsHistoryResponse
//	@Router		/schedule/api/v2/jobs/compliance/connections [post]
func (h HttpServer) GetComplianceJobsHistoryByIntegration(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	var request api.GetComplianceJobsHistoryByIntegrationRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var connections []onboardapi.Connection
	for _, info := range request.IntegrationInfo {
		if info.IntegrationTracker != nil {
			connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, *info.IntegrationTracker)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			if connection != nil {
				connections = append(connections, *connection)
			}
			continue
		}
		connectionsTmp, err := h.Scheduler.onboardClient.GetSourceByFilters(clientCtx,
			onboardapi.GetSourceByFiltersRequest{
				Connector:         info.Integration,
				ProviderNameRegex: info.IDName,
				ProviderIdRegex:   info.ID,
			})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		connections = append(connections, connectionsTmp...)
	}

	connectionInfo := make(map[string]api.IntegrationInfo)
	for _, c := range connections {
		connectionInfo[c.ID.String()] = api.IntegrationInfo{
			IntegrationTracker: c.ID.String(),
			Integration:        c.Connector.String(),
			IDName:             c.ConnectionName,
			ID:                 c.ConnectionID,
		}
	}

	var jobsResults []api.GetComplianceJobsHistoryResponse
	for _, c := range connectionInfo {
		jobs, err := h.DB.ListComplianceJobsByFilters([]string{c.IntegrationTracker}, request.BenchmarkId, request.JobStatus, request.StartTime, request.EndTime)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		for _, j := range jobs {
			var jobIntegrations []api.IntegrationInfo
			for _, cid := range j.ConnectionIDs {
				if info, ok := connectionInfo[cid]; ok {
					jobIntegrations = append(jobIntegrations, info)
				} else {
					connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, cid)
					if err != nil {
						return echo.NewHTTPError(http.StatusBadRequest, err.Error())
					}
					if connection != nil {
						info = api.IntegrationInfo{
							IntegrationTracker: connection.ID.String(),
							Integration:        connection.Connector.String(),
							IDName:             connection.ConnectionName,
							ID:                 connection.ConnectionID,
						}
						connectionInfo[cid] = info
						jobIntegrations = append(jobIntegrations, info)
					}
				}
			}

			jobsResults = append(jobsResults, api.GetComplianceJobsHistoryResponse{
				JobId:           j.ID,
				BenchmarkId:     j.BenchmarkID,
				JobStatus:       j.Status.ToApi(),
				DateTime:        j.UpdatedAt,
				IntegrationInfo: jobIntegrations,
			})
		}
	}

	if request.SortBy != nil {
		switch strings.ToLower(*request.SortBy) {
		case "id":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		case "datetime":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].DateTime.Before(jobsResults[j].DateTime)
			})
		case "benchmarkid":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].BenchmarkId < jobsResults[j].BenchmarkId
			})
		case "jobstatus":
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobStatus < jobsResults[j].JobStatus
			})
		default:
			sort.Slice(jobsResults, func(i, j int) bool {
				return jobsResults[i].JobId < jobsResults[j].JobId
			})
		}
	} else {
		sort.Slice(jobsResults, func(i, j int) bool {
			return jobsResults[i].JobId < jobsResults[j].JobId
		})
	}
	if request.PerPage != nil {
		if request.Cursor == nil {
			jobsResults = utils.Paginate(1, *request.PerPage, jobsResults)
		} else {
			jobsResults = utils.Paginate(*request.Cursor, *request.PerPage, jobsResults)
		}
	}

	return ctx.JSON(http.StatusOK, jobsResults)
}

// CancelJob godoc
//
//	@Summary	Cancel job by given job type and job ID
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		job_id			query	string		true	"Job ID"
//	@Param		job_type		query	string		true	"Job Type"
//	@Produce	json
//	@Success	200	{object}	[]api.GetComplianceJobsHistoryResponse
//	@Router		/schedule/api/v2/jobs/cancel [post]
func (h HttpServer) CancelJob(ctx echo.Context) error {
	jobIdStr := ctx.QueryParam("job_id")
	jobType := strings.ToLower(ctx.QueryParam("job_type"))

	switch jobType {
	case "compliance":
		jobId, err := strconv.ParseUint(jobIdStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
		}
		complianceJob, err := h.DB.GetComplianceJobByID(uint(jobId))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if complianceJob == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "job not found")
		}
		if complianceJob.Status == model2.ComplianceJobCreated {
			err = h.DB.UpdateComplianceJob(uint(jobId), model2.ComplianceJobCanceled, "")
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			return ctx.NoContent(http.StatusOK)
		} else if complianceJob.Status == model2.ComplianceJobSucceeded || complianceJob.Status == model2.ComplianceJobFailed ||
			complianceJob.Status == model2.ComplianceJobTimeOut || complianceJob.Status == model2.ComplianceJobCanceled {
			return echo.NewHTTPError(http.StatusOK, "job is already finished")
		} else if complianceJob.Status == model2.ComplianceJobSummarizerInProgress || complianceJob.Status == model2.ComplianceJobSinkInProgress {
			return echo.NewHTTPError(http.StatusOK, "job is already in progress, unable to cancel")
		}
		runners, err := h.DB.ListComplianceJobRunnersWithID(uint(jobId))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if len(runners) == 0 {
			err = h.DB.UpdateComplianceJob(uint(jobId), model2.ComplianceJobCanceled, "")
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			return ctx.NoContent(http.StatusOK)
		} else {
			allInProgress := true
			for _, r := range runners {
				if r.Status == runner2.ComplianceRunnerCreated {
					allInProgress = false
					err = h.DB.UpdateRunnerJob(r.ID, runner2.ComplianceRunnerCanceled, r.StartedAt, nil, "")
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
					}
				} else if r.Status == runner2.ComplianceRunnerQueued {
					allInProgress = false
					err = h.Scheduler.jq.DeleteMessage(ctx.Request().Context(), runner2.StreamName, r.NatsSequenceNumber)
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
					}
					err = h.DB.UpdateRunnerJob(r.ID, runner2.ComplianceRunnerCanceled, r.StartedAt, nil, "")
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
					}
				}
			}
			if allInProgress {
				return echo.NewHTTPError(http.StatusOK, "job is already in progress, unable to cancel")
			} else {
				err = h.DB.UpdateComplianceJob(uint(jobId), model2.ComplianceJobCanceled, "")
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				return ctx.NoContent(http.StatusOK)
			}
		}
	case "discovery":
		job, err := h.DB.GetDescribeJobById(jobIdStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
		}
		if job == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "job not found")
		}
		if job.Status == api.DescribeResourceJobCreated {
			err = h.DB.UpdateDescribeConnectionJobStatus(job.ID, api.DescribeResourceJobCanceled, "", "", 0, 0)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			return ctx.NoContent(http.StatusOK)
		} else if job.Status == api.DescribeResourceJobCanceled || job.Status == api.DescribeResourceJobFailed ||
			job.Status == api.DescribeResourceJobSucceeded || job.Status == api.DescribeResourceJobTimeout {
			return echo.NewHTTPError(http.StatusOK, "job is already finished")
		} else if job.Status == api.DescribeResourceJobQueued {
			err = h.Scheduler.jq.DeleteMessage(ctx.Request().Context(), awsDescriberLocal.StreamName, job.NatsSequenceNumber)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			err = h.DB.UpdateDescribeConnectionJobStatus(job.ID, api.DescribeResourceJobCanceled, "", "", 0, 0)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			return ctx.NoContent(http.StatusOK)
		} else {
			return echo.NewHTTPError(http.StatusOK, "job is already in progress, unable to cancel")
		}
	case "analytics":
		jobId, err := strconv.ParseUint(jobIdStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
		}
		job, err := h.DB.GetAnalyticsJobByID(uint(jobId))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if job == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "job not found")
		}
		if job.Status == analyticsapi.JobCreated {
			job.Status = analyticsapi.JobCanceled
			err = h.DB.UpdateAnalyticsJobStatus(*job)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
		} else if job.Status == analyticsapi.JobInProgress {
			return echo.NewHTTPError(http.StatusOK, "job is already in progress, unable to cancel")
		} else {
			return echo.NewHTTPError(http.StatusOK, "job is already finished")
		}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job type")
	}
	return echo.NewHTTPError(http.StatusOK, "nothing done")
}
