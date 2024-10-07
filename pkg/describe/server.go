package describe

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgtype"
	apiAuth "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/describe/enums"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	runner2 "github.com/kaytu-io/open-governance/pkg/compliance/runner"
	queryrunner "github.com/kaytu-io/open-governance/pkg/inventory/query-runner"
	onboardClient "github.com/kaytu-io/open-governance/pkg/onboard/client"
	"github.com/kaytu-io/open-governance/pkg/utils"
	"github.com/labstack/echo/v4"
	"github.com/sony/sonyflake"
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
	"github.com/kaytu-io/kaytu-util/pkg/source"
	analyticsapi "github.com/kaytu-io/open-governance/pkg/analytics/api"
	complianceapi "github.com/kaytu-io/open-governance/pkg/compliance/api"
	"github.com/kaytu-io/open-governance/pkg/describe/api"
	"github.com/kaytu-io/open-governance/pkg/describe/db"
	model2 "github.com/kaytu-io/open-governance/pkg/describe/db/model"
	"github.com/kaytu-io/open-governance/pkg/describe/es"
	onboardapi "github.com/kaytu-io/open-governance/pkg/onboard/api"
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

	v3 := e.Group("/api/v3")
	v3.POST("/jobs/discovery/connections/:connection_id", httpserver.AuthorizeHandler(h.GetDescribeJobsHistory, apiAuth.ViewerRole))
	v3.POST("/jobs/compliance/connections/:connection_id", httpserver.AuthorizeHandler(h.GetComplianceJobsHistory, apiAuth.ViewerRole))
	v3.POST("/jobs/discovery/connections", httpserver.AuthorizeHandler(h.GetDescribeJobsHistoryByIntegration, apiAuth.ViewerRole))
	v3.POST("/jobs/compliance/connections", httpserver.AuthorizeHandler(h.GetComplianceJobsHistoryByIntegration, apiAuth.ViewerRole))

	v3.POST("/compliance/benchmark/:benchmark_id/run", httpserver.AuthorizeHandler(h.RunBenchmarkById, apiAuth.AdminRole))
	v3.POST("/compliance/run", httpserver.AuthorizeHandler(h.RunBenchmark, apiAuth.AdminRole))
	v3.POST("/discovery/run", httpserver.AuthorizeHandler(h.RunDiscovery, apiAuth.AdminRole))
	v3.POST("/discovery/status", httpserver.AuthorizeHandler(h.GetIntegrationDiscoveryProgress, apiAuth.ViewerRole))

	v3.PUT("/query/:query_id/run", httpserver.AuthorizeHandler(h.RunQuery, apiAuth.AdminRole))
	v3.GET("/job/discovery/:job_id", httpserver.AuthorizeHandler(h.GetDescribeJobStatus, apiAuth.ViewerRole))
	v3.GET("/job/compliance/:job_id", httpserver.AuthorizeHandler(h.GetComplianceJobStatus, apiAuth.ViewerRole))
	v3.GET("/job/analytics/:job_id", httpserver.AuthorizeHandler(h.GetAnalyticsJobStatus, apiAuth.ViewerRole))
	v3.GET("/job/query/:job_id", httpserver.AuthorizeHandler(h.GetAsyncQueryRunJobStatus, apiAuth.ViewerRole))
	v3.POST("/jobs/discovery", httpserver.AuthorizeHandler(h.ListDescribeJobs, apiAuth.ViewerRole))
	v3.POST("/jobs/compliance", httpserver.AuthorizeHandler(h.ListComplianceJobs, apiAuth.ViewerRole))
	v3.POST("/benchmark/:benchmark_id/run-history", httpserver.AuthorizeHandler(h.BenchmarkAuditHistory, apiAuth.ViewerRole))
	v3.GET("/benchmark/run-history/integrations", httpserver.AuthorizeHandler(h.BenchmarkAuditHistoryIntegrations, apiAuth.ViewerRole))
	v3.GET("/jobs/analytics", httpserver.AuthorizeHandler(h.ListAnalyticsJobs, apiAuth.ViewerRole))
	v3.PUT("/jobs/cancel/byid", httpserver.AuthorizeHandler(h.CancelJobById, apiAuth.AdminRole))
	v3.POST("/jobs/cancel", httpserver.AuthorizeHandler(h.CancelJob, apiAuth.AdminRole))
	v3.POST("/jobs", httpserver.AuthorizeHandler(h.ListJobsByType, apiAuth.ViewerRole))
	v3.GET("/jobs/interval", httpserver.AuthorizeHandler(h.ListJobsInterval, apiAuth.ViewerRole))
	v3.GET("/jobs/compliance/summary/jobs", httpserver.AuthorizeHandler(h.GetSummaryJobs, apiAuth.ViewerRole))
	v3.GET("/jobs/history/compliance", httpserver.AuthorizeHandler(h.ListComplianceJobsHistory, apiAuth.ViewerRole))

	v3.PUT("/sample/purge", httpserver.AuthorizeHandler(h.PurgeSampleData, apiAuth.AdminRole))

	v3.GET("/integration/discovery/last-job", httpserver.AuthorizeHandler(h.GetIntegrationLastDiscoveryJob, apiAuth.ViewerRole))
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

	sortOrder := "DESC"
	if request.SortOrder == api.JobSortOrder_ASC {
		sortOrder = "ASC"
	}

	var startTime, endTime *time.Time
	if request.Interval != nil {
		tmpInterval, err := convertInterval(*request.Interval)
		if err != nil {
			h.Scheduler.logger.Error("failed to parse interval", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to parse interval")
		}
		request.Interval = &tmpInterval
	} else if request.To != nil && request.From != nil {
		endTimeTmp := time.Unix(*request.To, 0)
		endTime = &endTimeTmp

		startTimeTmp := time.Unix(*request.From, 0)
		startTime = &startTimeTmp
	}

	describeJobs, err := h.DB.ListAllJobs(request.PageStart, request.PageEnd, request.Interval, startTime, endTime, request.TypeFilters,
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
	summaries, err := h.DB.GetAllJobSummary(request.Interval, startTime, endTime, request.TypeFilters, request.StatusFilter)
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

	switch strings.ToLower(sortOrder) {
	case "asc":
		switch strings.ToLower(string(request.SortBy)) {
		case "id":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].ID < jobs[j].ID
			})
		case "connection_id":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].ConnectionID < jobs[j].ConnectionID
			})
		case "job_type":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].Type < jobs[j].Type
			})
		case "status":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].Status < jobs[j].Status
			})
		case "created_at", "createdat":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
			})
		case "updated_at", "updatedat":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].UpdatedAt.Before(jobs[j].UpdatedAt)
			})
		default:
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].UpdatedAt.Before(jobs[j].UpdatedAt)
			})
		}
	case "desc":
		switch strings.ToLower(string(request.SortBy)) {
		case "id":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].ID > jobs[j].ID
			})
		case "connection_id":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].ConnectionID > jobs[j].ConnectionID
			})
		case "job_type":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].Type > jobs[j].Type
			})
		case "status":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].Status > jobs[j].Status
			})
		case "created_at", "createdat":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
			})
		case "updated_at", "updatedat":
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].UpdatedAt.After(jobs[j].UpdatedAt)
			})
		default:
			sort.Slice(jobs, func(i, j int) bool {
				return jobs[i].UpdatedAt.After(jobs[j].UpdatedAt)
			})
		}
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
	userID := httpserver.GetUserID(ctx)
	if userID == "" {
		userID = "system"
	}

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
			daj, err := h.Scheduler.describe(src, resourceType, false, costFullDiscovery, false, nil, userID)
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
	userID := httpserver.GetUserID(ctx)
	if userID == "" {
		userID = "system"
	}

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
			_, err = h.Scheduler.describe(connection, resourceType, false, false, false, nil, userID)
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
	userID := httpserver.GetUserID(ctx)
	if userID == "" {
		userID = "system"
	}

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

	for _, c := range connectionIDs {
		_, err = h.Scheduler.complianceScheduler.CreateComplianceReportJobs(benchmarkID, lastJob, c, true, userID)
		if err != nil {
			return fmt.Errorf("error while creating compliance job: %v", err)
		}
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
	userID := httpserver.GetUserID(ctx)
	if userID == "" {
		userID = "system"
	}

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

		for _, c := range connectionIDs {
			_, err = h.Scheduler.complianceScheduler.CreateComplianceReportJobs(benchmark.ID, lastJob, c, true, userID)
			if err != nil {
				return fmt.Errorf("error while creating compliance job: %v", err)
			}
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
	userID := httpserver.GetUserID(ctx)
	if userID == "" {
		userID = "system"
	}
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
		daj, err := h.Scheduler.describe(describeJob.Connection, describeJob.ResourceType, false, false, false, nil, userID)
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
//	@Param		request			body	api.GetDescribeJobsHistoryRequest	true	"List jobs request"
//	@Param		connection_id	path	string								true	"Connection ID"
//	@Produce	json
//	@Success	200	{object}	[]api.GetDescribeJobsHistoryResponse
//	@Router		/schedule/api/v3/jobs/discovery/connections/{connection_id} [post]
func (h HttpServer) GetDescribeJobsHistory(ctx echo.Context) error {
	connectionId := ctx.Param("connection_id")

	var request api.GetDescribeJobsHistoryRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var jobsResults []api.GetDescribeJobsHistoryResponse

	jobs, err := h.DB.ListDescribeJobsByFilters(nil, []string{connectionId}, request.ResourceType,
		request.DiscoveryType, request.JobStatus, &request.StartTime, request.EndTime)
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
//	@Param		request			body	api.GetComplianceJobsHistoryRequest	true	"List jobs request"
//	@Param		connection_id	path	string								true	"Connection ID"
//	@Produce	json
//	@Success	200	{object}	[]api.GetComplianceJobsHistoryResponse
//	@Router		/schedule/api/v3/jobs/compliance/connections/{connection_id} [post]
func (h HttpServer) GetComplianceJobsHistory(ctx echo.Context) error {
	var request api.GetComplianceJobsHistoryRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	connectionId := ctx.Param("connection_id")

	jobs, err := h.DB.ListComplianceJobsByFilters([]string{connectionId}, request.BenchmarkId, request.JobStatus, &request.StartTime, request.EndTime)
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
//	@Param			benchmark_id	path		string						true	"Benchmark ID"
//	@Param			request			body		api.RunBenchmarkByIdRequest	true	"Integrations filter"
//	@Success		200				{object}	api.RunBenchmarkResponse
//	@Router			/schedule/api/v3/compliance/benchmark/{benchmark_id}/run [post]
func (h HttpServer) RunBenchmarkById(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}
	userID := httpserver.GetUserID(ctx)
	if userID == "" {
		userID = "system"
	}
	benchmarkID := strings.ToLower(ctx.Param("benchmark_id"))

	var request api.RunBenchmarkByIdRequest
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
		connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
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

	var jobs []api.RunBenchmarkItem
	for _, c := range connectionIDs {
		jobId, err := h.Scheduler.complianceScheduler.CreateComplianceReportJobs(benchmarkID, lastJob, c, true, userID)
		if err != nil {
			return fmt.Errorf("error while creating compliance job: %v", err)
		}
		jobs = append(jobs, api.RunBenchmarkItem{
			JobId:           jobId,
			BenchmarkId:     benchmark.ID,
			IntegrationInfo: connectionInfo,
		})
	}

	return ctx.JSON(http.StatusOK, api.RunBenchmarkResponse{
		Jobs: jobs,
	})
}

// RunBenchmark godoc
//
//	@Summary		Triggers compliance job
//	@Description	Triggers a compliance job to run immediately for the given benchmark
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200		{object}	api.RunBenchmarkResponse
//	@Param			request	body		api.RunBenchmarkRequest	true	"Requst Body"
//	@Router			/schedule/api/v3/compliance/run [post]
func (h HttpServer) RunBenchmark(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	userID := httpserver.GetUserID(ctx)
	if userID == "" {
		userID = "system"
	}

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
		connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
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

	var jobs []api.RunBenchmarkItem
	for _, benchmark := range benchmarks {
		lastJob, err := h.Scheduler.db.GetLastComplianceJob(benchmark.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		for _, c := range connectionIDs {
			jobId, err := h.Scheduler.complianceScheduler.CreateComplianceReportJobs(benchmark.ID, lastJob, c, true, userID)
			if err != nil {
				return fmt.Errorf("error while creating compliance job: %v", err)
			}

			jobs = append(jobs, api.RunBenchmarkItem{
				JobId:           jobId,
				BenchmarkId:     benchmark.ID,
				IntegrationInfo: connectionInfo,
			})
		}
	}

	return ctx.JSON(http.StatusOK, api.RunBenchmarkResponse{
		Jobs: jobs,
	})
}

// RunDiscovery godoc
//
//	@Summary		Run Discovery job
//	@Description	Triggers a discovery job to run immediately for the given resource types and Integrations
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200		{object}	api.RunDiscoveryResponse
//	@Param			request	body		api.RunDiscoveryRequest	true	"Request Body"
//	@Router			/schedule/api/v3/discovery/run [post]
func (h HttpServer) RunDiscovery(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}
	userID := httpserver.GetUserID(ctx)
	if userID == "" {
		userID = "system"
	}

	sf := sonyflake.NewSonyflake(sonyflake.Settings{})
	triggerId, err := sf.NextID()
	if err != nil {
		return err
	}

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
				h.Scheduler.logger.Error("failed to get source", zap.String("source id", *info.IntegrationTracker), zap.Error(err))
				return echo.NewHTTPError(http.StatusBadRequest, "failed to get source")
			}
			if connection != nil {
				connections = append(connections, *connection)
			}
			continue
		}
		connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
			onboardapi.GetSourceByFiltersRequest{
				Connector:         info.Integration,
				ProviderNameRegex: info.IDName,
				ProviderIdRegex:   info.ID,
			})
		if err != nil {
			h.Scheduler.logger.Error("failed to get source", zap.Any("source", info), zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, "failed to get source")
		}
		connections = append(connections, connectionsTmp...)
	}

	var jobs []api.RunDiscoveryJob
	for _, connection := range connections {
		if !connection.IsEnabled() {
			continue
		}
		rtToDescribe := request.ResourceTypes
		discoveryType := model2.DiscoveryType_Fast
		if request.ForceFull {
			discoveryType = model2.DiscoveryType_Full
		}
		integrationDiscovery := &model2.IntegrationDiscovery{
			TriggerID:     uint(triggerId),
			ConnectionID:  connection.ID.String(),
			AccountID:     connection.ConnectionID,
			TriggerType:   enums.DescribeTriggerTypeManual,
			TriggeredBy:   userID,
			DiscoveryType: discoveryType,
			ResourceTypes: rtToDescribe,
		}
		err = h.DB.CreateIntegrationDiscovery(integrationDiscovery)
		if err != nil {
			h.Scheduler.logger.Error("failed to create integration discovery", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create integration discovery")
		}

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
			job, err := h.Scheduler.describe(connection, resourceType, false, false, false, &integrationDiscovery.ID, userID)
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
				if failureReason == "" && err != nil {
					failureReason = err.Error()
				}
			} else {
				jobId = job.ID
				status = string(job.Status)
			}
			jobs = append(jobs, api.RunDiscoveryJob{
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
	return ctx.JSON(http.StatusOK, api.RunDiscoveryResponse{
		Jobs:      jobs,
		TriggerID: uint(triggerId),
	})
}

// GetDescribeJobStatus godoc
//
//	@Summary	Get describe job status by job id
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		job_id	path	string	true	"Job ID"
//	@Produce	json
//	@Success	200	{object}	api.GetDescribeJobStatusResponse
//	@Router		/schedule/api/v3/jobs/discovery/{job_id} [get]
func (h HttpServer) GetDescribeJobStatus(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	jobId := ctx.Param("job_id")

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
//	@Param		job_id	path	string	true	"Job ID"
//	@Produce	json
//	@Success	200	{object}	api.GetComplianceJobStatusResponse
//	@Router		/schedule/api/v3/job/compliance/{job_id} [get]
func (h HttpServer) GetComplianceJobStatus(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	jobIdString := ctx.Param("job_id")
	jobId, err := strconv.ParseUint(jobIdString, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	j, err := h.DB.GetComplianceJobByID(uint(jobId))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var connectionInfo api.IntegrationInfo
	connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, j.ConnectionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	connectionInfo = api.IntegrationInfo{
		IntegrationTracker: connection.ID.String(),
		Integration:        connection.Connector.String(),
		ID:                 connection.ConnectionID,
		IDName:             connection.ConnectionName,
	}

	jobsResult := api.GetComplianceJobStatusResponse{
		JobId:           j.ID,
		IntegrationInfo: connectionInfo,
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
//	@Param		job_id	path	string	true	"Job ID"
//	@Produce	json
//	@Success	200	{object}	api.GetAnalyticsJobStatusResponse
//	@Router		/schedule/api/v3/job/analytics/{job_id} [get]
func (h HttpServer) GetAnalyticsJobStatus(ctx echo.Context) error {

	jobIdString := ctx.Param("job_id")
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

// GetAsyncQueryRunJobStatus godoc
//
//	@Summary	Get async query run job status by job id
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		job_id	path	string	true	"Job ID"
//	@Produce	json
//	@Success	200	{object}	api.GetAsyncQueryRunJobStatusResponse
//	@Router		/schedule/api/v3/job/query/{job_id} [get]
func (h HttpServer) GetAsyncQueryRunJobStatus(ctx echo.Context) error {

	jobIdString := ctx.Param("job_id")
	jobId, err := strconv.ParseUint(jobIdString, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	j, err := h.DB.GetQueryRunnerJob(uint(jobId))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jobsResult := api.GetAsyncQueryRunJobStatusResponse{
		JobId:          j.ID,
		QueryId:        j.QueryId,
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
		CreatedBy:      j.CreatedBy,
		JobStatus:      j.Status,
		FailureMessage: j.FailureMessage,
	}

	return ctx.JSON(http.StatusOK, jobsResult)
}

// ListDescribeJobs godoc
//
//	@Summary	Get describe jobs history for give connection
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.ListDescribeJobsRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetDescribeJobsHistoryResponse
//	@Router		/schedule/api/v3/jobs/discovery [post]
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
		connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
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

	jobs, err := h.DB.ListDescribeJobsByFilters(nil, connectionIDs, request.ResourceType,
		request.DiscoveryType, request.JobStatus, &request.StartTime, request.EndTime)
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
//	@Param		request	body	api.ListComplianceJobsRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetComplianceJobsHistoryResponse
//	@Router		/schedule/api/v3/jobs/compliance [post]
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
		connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
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

	jobs, err := h.DB.ListComplianceJobsByFilters(connectionIDs, request.BenchmarkId, request.JobStatus, &request.StartTime, request.EndTime)
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
		if info, ok := connectionInfo[j.ConnectionID]; ok {
			jobResult.IntegrationInfo = info
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

// BenchmarkAuditHistory godoc
//
//	@Summary	Get compliance jobs history for give connection
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request			body	api.BenchmarkAuditHistoryRequest	true	"List jobs request"
//	@Param		benchmark_id	query	string	true	"Benchmark ID to get the run history for"
//	@Produce	json
//	@Success	200	{object}	api.BenchmarkAuditHistoryResponse
//	@Router		/schedule/api/v3/benchmark/:benchmark_id/run-history [post]
func (h HttpServer) BenchmarkAuditHistory(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	benchmarkID := ctx.Param("benchmark_id")

	var request api.BenchmarkAuditHistoryRequest
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
		connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
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
		connectionIDs = append(connectionIDs, c.ID.String())
	}
	connections2, err := h.Scheduler.onboardClient.ListSources(clientCtx, nil)
	if err != nil {
		h.Scheduler.logger.Error("failed to list connections", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	for _, c := range connections2 {
		connectionInfo[c.ID.String()] = api.IntegrationInfo{
			IntegrationTracker: c.ID.String(),
			Integration:        c.Connector.String(),
			IDName:             c.ConnectionName,
			ID:                 c.ConnectionID,
		}
	}

	var startTime, endTime *time.Time
	if request.Interval != nil {
		startTime, endTime, err = parseTimeInterval(*request.Interval)
	} else {
		startTime = &request.StartTime
		endTime = request.EndTime
	}

	jobs, err := h.DB.ListComplianceJobsByFilters(connectionIDs, []string{benchmarkID}, request.JobStatus, startTime, endTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var items []api.BenchmarkAuditHistoryItem
	for _, j := range jobs {
		item := api.BenchmarkAuditHistoryItem{
			JobId:       j.ID,
			BenchmarkId: j.BenchmarkID,
			JobStatus:   j.Status.ToApi(),
			CreatedAt:   j.CreatedAt,
			UpdatedAt:   j.UpdatedAt,
		}
		if info, ok := connectionInfo[j.ConnectionID]; ok {
			item.IntegrationInfo = info
			item.NumberOfIntegrations = 1
		}

		items = append(items, item)
	}

	totalCount := len(items)
	if request.SortBy != nil {
		switch strings.ToLower(*request.SortBy) {
		case "id":
			sort.Slice(items, func(i, j int) bool {
				return items[i].JobId < items[j].JobId
			})
		case "updated_at", "updatedat":
			sort.Slice(items, func(i, j int) bool {
				return items[i].UpdatedAt.After(items[j].UpdatedAt)
			})
		case "created_at", "createdat":
			sort.Slice(items, func(i, j int) bool {
				return items[i].CreatedAt.After(items[j].CreatedAt)
			})
		case "benchmarkid":
			sort.Slice(items, func(i, j int) bool {
				return items[i].BenchmarkId < items[j].BenchmarkId
			})
		case "jobstatus":
			sort.Slice(items, func(i, j int) bool {
				return items[i].JobStatus < items[j].JobStatus
			})
		default:
			sort.Slice(items, func(i, j int) bool {
				return items[i].UpdatedAt.After(items[j].UpdatedAt)
			})
		}
	} else {
		sort.Slice(items, func(i, j int) bool {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		})
	}
	if request.PerPage != nil {
		if request.Cursor == nil {
			items = utils.Paginate(1, *request.PerPage, items)
		} else {
			items = utils.Paginate(*request.Cursor, *request.PerPage, items)
		}
	}

	return ctx.JSON(http.StatusOK, api.BenchmarkAuditHistoryResponse{
		Items:      items,
		TotalCount: totalCount,
	})
}

// BenchmarkAuditHistoryIntegrations godoc
//
//	@Summary	Get compliance jobs history for give connection possible integrations
//	@Security	BearerToken
//	@Tags		scheduler
//	@Produce	json
//	@Success	200	{object}	[]api.IntegrationInfo
//	@Router		/schedule/api/v3/benchmark/run-history/integrations [get]
func (h HttpServer) BenchmarkAuditHistoryIntegrations(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	connectionIDs, err := h.DB.GetComplianceJobsIntegrations()
	if err != nil {
		h.Scheduler.logger.Error("failed to get compliance jobs integrations", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get compliance jobs integrations")
	}

	connections, err := h.Scheduler.onboardClient.GetSources(clientCtx, connectionIDs)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var integrations []api.IntegrationInfo

	for _, c := range connections {
		integrations = append(integrations, api.IntegrationInfo{
			IntegrationTracker: c.ID.String(),
			Integration:        c.Connector.String(),
			IDName:             c.ConnectionName,
			ID:                 c.ConnectionID,
		})
	}

	return ctx.JSON(http.StatusOK, integrations)
}

// GetIntegrationLastDiscoveryJob godoc
//
//	@Summary	Get Last dicovery job for integration
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request			body	api.GetIntegrationLastDiscoveryJobRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	model2.DescribeConnectionJob
//	@Router		/schedule/api/v3/integration/discovery/last-job [post]
func (h HttpServer) GetIntegrationLastDiscoveryJob(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	var request api.GetIntegrationLastDiscoveryJobRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var connection onboardapi.Connection
	if request.IntegrationInfo.IntegrationTracker != nil {
		connectionTmp, err := h.Scheduler.onboardClient.GetSource(clientCtx, *request.IntegrationInfo.IntegrationTracker)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if connectionTmp != nil {
			connection = *connectionTmp
		}
	} else {
		connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
			onboardapi.GetSourceByFiltersRequest{
				Connector:         request.IntegrationInfo.Integration,
				ProviderNameRegex: request.IntegrationInfo.IDName,
				ProviderIdRegex:   request.IntegrationInfo.ID,
			})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		connection = connectionsTmp[0]
	}

	job, err := h.DB.ListDescribeJobs(connection.ID.String())
	if err != nil {
		h.Scheduler.logger.Error("failed to get describe jobs", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get describe jobs")
	}

	return ctx.JSON(http.StatusOK, job)
}

// ListAnalyticsJobs godoc
//
//	@Summary	Get analytics jobs history for give connection
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		job_status	query	[]string	true	"List of job statuses to filter"
//	@Param		type		query	[]string	true	"List of types to filter"
//	@Param		sort_by		query	string		true	"sort by field"
//	@Param		cursor		query	int			true	"cursor"
//	@Param		per_page	query	int			true	"per page"
//	@Param		startTime	query	int			false	"timestamp for start of the chart in epoch seconds"
//	@Param		endTime		query	int			false	"timestamp for end of the chart in epoch seconds"
//	@Produce	json
//	@Success	200	{object}	[]api.GetAnalyticsJobsHistoryResponse
//	@Router		/schedule/api/v3/jobs/analytics [get]
func (h HttpServer) ListAnalyticsJobs(ctx echo.Context) error {
	types := httpserver.QueryArrayParam(ctx, "types")
	jobStatus := httpserver.QueryArrayParam(ctx, "job_status")
	sortBy := ctx.QueryParam("job_status")

	var cursor, perPage int64
	var err error
	cursorStr := ctx.QueryParam("cursor")
	if cursorStr != "" {
		cursor, err = strconv.ParseInt(cursorStr, 10, 64)
		if err != nil {
			return err
		}
	}
	perPageStr := ctx.QueryParam("per_page")
	if perPageStr != "" {
		perPage, err = strconv.ParseInt(perPageStr, 10, 64)
		if err != nil {
			return err
		}
	}

	var startTime, endTime *time.Time
	if endTimeStr := ctx.QueryParam("timeAt"); endTimeStr != "" {
		endTimeInt, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return err
		}
		endTimeTmp := time.Unix(endTimeInt, 0)
		endTime = &endTimeTmp
	}

	if startTimeStr := ctx.QueryParam("startTime"); startTimeStr != "" {
		startTimeInt, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return err
		}
		startTimeTmp := time.Unix(startTimeInt, 0)
		startTime = &startTimeTmp
	}

	jobs, err := h.DB.ListAnalyticsJobsByFilter(types, jobStatus, startTime, endTime)
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
	if sortBy != "" {
		switch strings.ToLower(sortBy) {
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

	if perPage != 0 {
		if cursor == 0 {
			jobsResults = utils.Paginate(1, perPage, jobsResults)
		} else {
			jobsResults = utils.Paginate(cursor, perPage, jobsResults)
		}
	}

	return ctx.JSON(http.StatusOK, jobsResults)
}

// GetDescribeJobsHistoryByIntegration godoc
//
//	@Summary	Get describe jobs history for give connection
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetDescribeJobsHistoryByIntegrationRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetDescribeJobsHistoryResponse
//	@Router		/schedule/api/v3/jobs/discovery/connections [post]
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
		connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
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
		jobs, err := h.DB.ListDescribeJobsByFilters(nil, []string{c.IntegrationTracker}, request.ResourceType,
			request.DiscoveryType, request.JobStatus, &request.StartTime, request.EndTime)
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
//	@Param		request	body	api.GetComplianceJobsHistoryByIntegrationRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	[]api.GetComplianceJobsHistoryResponse
//	@Router		/schedule/api/v3/jobs/compliance/connections [post]
func (h HttpServer) GetComplianceJobsHistoryByIntegration(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	var request api.GetComplianceJobsHistoryByIntegrationRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	var connections onboardapi.Connection
	if request.IntegrationInfo.IntegrationTracker != nil {
		connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, *request.IntegrationInfo.IntegrationTracker)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if connection != nil {
			connections = *connection
		}
	} else {
		connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
			onboardapi.GetSourceByFiltersRequest{
				Connector:         request.IntegrationInfo.Integration,
				ProviderNameRegex: request.IntegrationInfo.IDName,
				ProviderIdRegex:   request.IntegrationInfo.ID,
			})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		connections = connectionsTmp[0]
	}

	connectionInfo := make(map[string]api.IntegrationInfo)
	connectionInfo[connections.ID.String()] = api.IntegrationInfo{
		IntegrationTracker: connections.ID.String(),
		Integration:        connections.Connector.String(),
		IDName:             connections.ConnectionName,
		ID:                 connections.ConnectionID,
	}

	var jobsResults []api.GetComplianceJobsHistoryResponse
	for _, c := range connectionInfo {
		jobs, err := h.DB.ListComplianceJobsByFilters([]string{c.IntegrationTracker}, request.BenchmarkId, request.JobStatus, &request.StartTime, request.EndTime)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		for _, j := range jobs {
			var jobIntegrations api.IntegrationInfo
			if info, ok := connectionInfo[j.ConnectionID]; ok {
				jobIntegrations = info
			} else {
				connection, err := h.Scheduler.onboardClient.GetSource(clientCtx, j.ConnectionID)
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
					connectionInfo[j.ConnectionID] = info
					jobIntegrations = info
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

// CancelJobById godoc
//
//	@Summary	Cancel job by given job type and job ID
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		job_id		query	string	true	"Job ID to Cancel"
//	@Param		job_type	query	string	true	"Job Type"
//	@Produce	json
//	@Success	200
//	@Router		/schedule/api/v3/jobs/cancel/byid [put]
func (h HttpServer) CancelJobById(ctx echo.Context) error {
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

// CancelJob godoc
//
//	@Summary	Cancel job by given job type and job ID
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.CancelJobRequest	true	"Request Body"
//	@Produce	json
//	@Success	200	{object}	[]api.CancelJobResponse
//	@Router		/schedule/api/v3/jobs/cancel [post]
func (h HttpServer) CancelJob(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	var request api.CancelJobRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if strings.ToLower(request.JobType) != "compliance" && strings.ToLower(request.JobType) != "discovery" &&
		strings.ToLower(request.JobType) != "analytics" && strings.ToLower(request.JobType) != "query" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job type")
	}

	if strings.ToLower(request.JobType) == "query" && strings.ToLower(request.Selector) != "job_id" {
		return echo.NewHTTPError(http.StatusBadRequest, "only jobId is acceptable for query run")
	}

	var jobIDs []string
	var results []api.CancelJobResponse

	switch strings.ToLower(request.Selector) {
	case "job_id":
		jobIDs = request.JobId
	case "integration_info":
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
			connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
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

		connectionIDsMap := make(map[string]bool)
		for _, c := range connections {
			connectionIDsMap[c.ID.String()] = true
		}
		var connectionIDs []string
		switch strings.ToLower(request.JobType) {
		case "compliance":
			jobs, err := h.DB.ListPendingComplianceJobsByConnectionID(connectionIDs)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			for _, j := range jobs {
				jobIDs = append(jobIDs, strconv.Itoa(int(j.ID)))
			}
		case "discovery":
			jobs, err := h.DB.ListPendingDescribeJobsByFilters(connectionIDs, nil, nil, nil, nil, nil)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			for _, j := range jobs {
				jobIDs = append(jobIDs, strconv.Itoa(int(j.ID)))
			}
		case "analytics":
			jobs, err := h.DB.ListPendingAnalyticsJobs()
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			for _, j := range jobs {
				jobIDs = append(jobIDs, strconv.Itoa(int(j.ID)))
			}
		}
	case "status":
		for _, status := range request.Status {
			switch strings.ToLower(request.JobType) {
			case "compliance":
				if model2.ComplianceJobStatus(strings.ToUpper(status)) != model2.ComplianceJobCreated &&
					model2.ComplianceJobStatus(strings.ToUpper(status)) != model2.ComplianceJobRunnersInProgress {
					continue
				}
				jobs, err := h.DB.ListComplianceJobsByStatus(model2.ComplianceJobStatus(strings.ToUpper(status)))
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				for _, j := range jobs {
					jobIDs = append(jobIDs, strconv.Itoa(int(j.ID)))
				}
			case "discovery":
				if api.DescribeResourceJobStatus(strings.ToUpper(status)) != api.DescribeResourceJobCreated &&
					api.DescribeResourceJobStatus(strings.ToUpper(status)) != api.DescribeResourceJobQueued {
					continue
				}
				jobs, err := h.DB.ListDescribeJobsByStatus(api.DescribeResourceJobStatus(strings.ToUpper(status)))
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				for _, j := range jobs {
					jobIDs = append(jobIDs, strconv.Itoa(int(j.ID)))
				}
			case "analytics":
				if strings.ToUpper(status) != "CREATED" {
					continue
				}
				jobs, err := h.DB.ListAnalyticsJobsByFilter(nil, []string{strings.ToUpper(status)}, nil, nil)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				for _, j := range jobs {
					jobIDs = append(jobIDs, strconv.Itoa(int(j.ID)))
				}
			}
		}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid selector. valid values: job_id, integration, status")
	}

	for _, jobIdStr := range jobIDs {
		var failureReason string
		var canceled bool
		switch strings.ToLower(request.JobType) {
		case "compliance":
			jobId, err := strconv.ParseUint(jobIdStr, 10, 64)
			if err != nil {
				failureReason = "invalid job id"
				break
			}
			complianceJob, err := h.DB.GetComplianceJobByID(uint(jobId))
			if err != nil {
				failureReason = err.Error()
				break
			}
			if complianceJob == nil {
				failureReason = "job not found"
				break
			}
			if complianceJob.Status == model2.ComplianceJobCreated {
				err = h.DB.UpdateComplianceJob(uint(jobId), model2.ComplianceJobCanceled, "")
				if err != nil {
					failureReason = err.Error()
					break
				}
				canceled = true
				break
			} else if complianceJob.Status == model2.ComplianceJobSucceeded || complianceJob.Status == model2.ComplianceJobFailed ||
				complianceJob.Status == model2.ComplianceJobTimeOut || complianceJob.Status == model2.ComplianceJobCanceled {
				failureReason = "job is already finished"
				break
			} else if complianceJob.Status == model2.ComplianceJobSummarizerInProgress || complianceJob.Status == model2.ComplianceJobSinkInProgress {
				failureReason = "job is already in progress, unable to cancel"
				break
			}
			runners, err := h.DB.ListComplianceJobRunnersWithID(uint(jobId))
			if err != nil {
				failureReason = err.Error()
				break
			}
			if len(runners) == 0 {
				err = h.DB.UpdateComplianceJob(uint(jobId), model2.ComplianceJobCanceled, "")
				if err != nil {
					failureReason = err.Error()
					break
				}
				canceled = true
				break
			} else {
				allInProgress := true
				for _, r := range runners {
					if r.Status == runner2.ComplianceRunnerCreated {
						allInProgress = false
						err = h.DB.UpdateRunnerJob(r.ID, runner2.ComplianceRunnerCanceled, r.StartedAt, nil, "")
						if err != nil {
							failureReason = err.Error()
							break
						}
					} else if r.Status == runner2.ComplianceRunnerQueued {
						allInProgress = false
						err = h.Scheduler.jq.DeleteMessage(ctx.Request().Context(), runner2.StreamName, r.NatsSequenceNumber)
						if err != nil {
							failureReason = err.Error()
							break
						}
						err = h.DB.UpdateRunnerJob(r.ID, runner2.ComplianceRunnerCanceled, r.StartedAt, nil, "")
						if err != nil {
							failureReason = err.Error()
							break
						}
					}
				}
				if allInProgress {
					failureReason = "job is already in progress, unable to cancel"
					break
				} else {
					err = h.DB.UpdateComplianceJob(uint(jobId), model2.ComplianceJobCanceled, "")
					if err != nil {
						failureReason = err.Error()
						break
					}
					canceled = true
					break
				}
			}
		case "discovery":
			job, err := h.DB.GetDescribeJobById(jobIdStr)
			if err != nil {
				failureReason = "invalid job id"
				break
			}
			if job == nil {
				failureReason = "job not found"
				break
			}
			if job.Status == api.DescribeResourceJobCreated {
				err = h.DB.UpdateDescribeConnectionJobStatus(job.ID, api.DescribeResourceJobCanceled, "", "", 0, 0)
				if err != nil {
					failureReason = err.Error()
					break
				}
				canceled = true
				break
			} else if job.Status == api.DescribeResourceJobCanceled || job.Status == api.DescribeResourceJobFailed ||
				job.Status == api.DescribeResourceJobSucceeded || job.Status == api.DescribeResourceJobTimeout {
				failureReason = "job is already finished"
				break
			} else if job.Status == api.DescribeResourceJobQueued {
				err = h.Scheduler.jq.DeleteMessage(ctx.Request().Context(), awsDescriberLocal.StreamName, job.NatsSequenceNumber)
				if err != nil {
					failureReason = err.Error()
					break
				}
				err = h.DB.UpdateDescribeConnectionJobStatus(job.ID, api.DescribeResourceJobCanceled, "", "", 0, 0)
				if err != nil {
					failureReason = err.Error()
					break
				}
				canceled = true
				break
			} else {
				failureReason = "job is already in progress, unable to cancel"
				break
			}
		case "analytics":
			jobId, err := strconv.ParseUint(jobIdStr, 10, 64)
			if err != nil {
				failureReason = "invalid job id"
				break
			}
			job, err := h.DB.GetAnalyticsJobByID(uint(jobId))
			if err != nil {
				failureReason = err.Error()
				break
			}
			if job == nil {
				failureReason = "job not found"
				break
			}
			if job.Status == analyticsapi.JobCreated {
				job.Status = analyticsapi.JobCanceled
				err = h.DB.UpdateAnalyticsJobStatus(*job)
				if err != nil {
					failureReason = err.Error()
					break
				}
				canceled = true
				break
			} else if job.Status == analyticsapi.JobInProgress {
				failureReason = "job is already in progress, unable to cancel"
				break
			} else {
				failureReason = "job is already finished"
				break
			}
		case "query":
			jobId, err := strconv.ParseUint(jobIdStr, 10, 64)
			if err != nil {
				failureReason = "invalid job id"
				break
			}
			job, err := h.DB.GetQueryRunnerJob(uint(jobId))
			if err != nil {
				failureReason = err.Error()
				break
			}
			if job == nil {
				failureReason = "job not found"
				break
			}
			if job.Status == queryrunner.QueryRunnerCreated {
				err = h.DB.UpdateQueryRunnerJobStatus(job.ID, queryrunner.QueryRunnerCanceled, "")
				if err != nil {
					failureReason = err.Error()
					break
				}
			} else if job.Status == queryrunner.QueryRunnerInProgress {
				failureReason = "job is already in progress, unable to cancel"
				break
			} else if job.Status == queryrunner.QueryRunnerQueued {
				err = h.Scheduler.jq.DeleteMessage(ctx.Request().Context(), queryrunner.StreamName, job.NatsSequenceNumber)
				if err != nil {
					failureReason = err.Error()
					break
				}
				err = h.DB.UpdateQueryRunnerJobStatus(job.ID, queryrunner.QueryRunnerCanceled, "")
				if err != nil {
					failureReason = err.Error()
					break
				}
			} else {
				failureReason = "job is already finished"
				break
			}

		default:
			failureReason = "invalid job type"
			break
		}
		results = append(results, api.CancelJobResponse{
			JobId:    jobIdStr,
			JobType:  strings.ToLower(request.JobType),
			Canceled: canceled,
			Reason:   failureReason,
		})
	}

	return ctx.JSON(http.StatusOK, results)
}

// ListJobsByType godoc
//
//	@Summary	List jobs by job type and filters
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.ListJobsByTypeRequest	true	"Request Body"
//	@Produce	json
//	@Success	200	{object}	[]api.ListJobsByTypeResponse
//	@Router		/schedule/api/v3/jobs [post]
func (h HttpServer) ListJobsByType(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	var request api.ListJobsByTypeRequest
	if err := ctx.Bind(&request); err != nil {
		ctx.Logger().Errorf("bind the request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if strings.ToLower(request.JobType) != "compliance" && strings.ToLower(request.JobType) != "discovery" &&
		strings.ToLower(request.JobType) != "analytics" && strings.ToLower(request.JobType) != "query_run" && strings.ToLower(request.JobType) != "queryrun" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job type")
	}

	sortBy := "id"
	switch request.SortBy {
	case api.JobSort_ByConnectionID, api.JobSort_ByJobID, api.JobSort_ByJobType, api.JobSort_ByStatus, api.JobSort_ByCreatedAt,
		api.JobSort_ByUpdatedAt:
		sortBy = string(request.SortBy)
	}

	if strings.ToLower(request.JobType) == "query" && strings.ToLower(request.Selector) != "job_id" {
		return echo.NewHTTPError(http.StatusBadRequest, "only jobId is acceptable for query run")
	}

	var items []api.ListJobsByTypeItem

	var err error
	switch strings.ToLower(request.Selector) {
	case "job_id":
		switch strings.ToLower(request.JobType) {
		case "query":
			var jobs []model2.QueryRunnerJob
			if len(request.JobId) == 0 {
				jobs, err = h.DB.ListQueryRunnerJobs()
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
			} else {
				jobs, err = h.DB.ListQueryRunnerJobsById(request.JobId)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
			}

			for _, j := range jobs {
				items = append(items, api.ListJobsByTypeItem{
					JobId:     strconv.Itoa(int(j.ID)),
					JobType:   strings.ToLower(request.JobType),
					JobStatus: string(j.Status),
					CreatedAt: j.CreatedAt,
					UpdatedAt: j.UpdatedAt,
				})
			}
		case "compliance":
			jobs, err := h.DB.ListComplianceJobsByIds(request.JobId)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			for _, j := range jobs {
				items = append(items, api.ListJobsByTypeItem{
					JobId:     strconv.Itoa(int(j.ID)),
					JobType:   strings.ToLower(request.JobType),
					JobStatus: string(j.Status),
					CreatedAt: j.CreatedAt,
					UpdatedAt: j.UpdatedAt,
				})
			}
		case "discovery":
			jobs, err := h.DB.ListDescribeJobsByIds(request.JobId)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			for _, j := range jobs {
				items = append(items, api.ListJobsByTypeItem{
					JobId:     strconv.Itoa(int(j.ID)),
					JobType:   strings.ToLower(request.JobType),
					JobStatus: string(j.Status),
					CreatedAt: j.CreatedAt,
					UpdatedAt: j.UpdatedAt,
				})
			}
		case "analytics":
			jobs, err := h.DB.ListAnalyticsJobsByIds(request.JobId)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			for _, j := range jobs {
				items = append(items, api.ListJobsByTypeItem{
					JobId:     strconv.Itoa(int(j.ID)),
					JobType:   strings.ToLower(request.JobType),
					JobStatus: string(j.Status),
					CreatedAt: j.CreatedAt,
					UpdatedAt: j.UpdatedAt,
				})
			}
		}
	case "integration_info":
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
			connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
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

		connectionIDsMap := make(map[string]bool)
		for _, c := range connections {
			connectionIDsMap[c.ID.String()] = true
		}
		var connectionIDs []string
		switch strings.ToLower(request.JobType) {
		case "compliance":
			jobs, err := h.DB.ListComplianceJobsByConnectionID(connectionIDs)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			for _, j := range jobs {
				items = append(items, api.ListJobsByTypeItem{
					JobId:     strconv.Itoa(int(j.ID)),
					JobType:   strings.ToLower(request.JobType),
					JobStatus: string(j.Status),
					CreatedAt: j.CreatedAt,
					UpdatedAt: j.UpdatedAt,
				})
			}
		case "discovery":
			jobs, err := h.DB.ListDescribeJobsByFilters(nil, connectionIDs, nil, nil, nil, nil, nil)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			for _, j := range jobs {
				items = append(items, api.ListJobsByTypeItem{
					JobId:     strconv.Itoa(int(j.ID)),
					JobType:   strings.ToLower(request.JobType),
					JobStatus: string(j.Status),
					CreatedAt: j.CreatedAt,
					UpdatedAt: j.UpdatedAt,
				})
			}
		case "analytics":
			jobs, err := h.DB.ListAnalyticsJobs()
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			for _, j := range jobs {
				items = append(items, api.ListJobsByTypeItem{
					JobId:     strconv.Itoa(int(j.ID)),
					JobType:   strings.ToLower(request.JobType),
					JobStatus: string(j.Status),
					CreatedAt: j.CreatedAt,
					UpdatedAt: j.UpdatedAt,
				})
			}
		}
	case "status":
		for _, status := range request.Status {
			switch strings.ToLower(request.JobType) {
			case "compliance":
				jobs, err := h.DB.ListComplianceJobsByStatus(model2.ComplianceJobStatus(strings.ToUpper(status)))
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				for _, j := range jobs {
					items = append(items, api.ListJobsByTypeItem{
						JobId:     strconv.Itoa(int(j.ID)),
						JobType:   strings.ToLower(request.JobType),
						JobStatus: string(j.Status),
						CreatedAt: j.CreatedAt,
						UpdatedAt: j.UpdatedAt,
					})
				}
			case "discovery":
				jobs, err := h.DB.ListDescribeJobsByStatus(api.DescribeResourceJobStatus(strings.ToUpper(status)))
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				for _, j := range jobs {
					items = append(items, api.ListJobsByTypeItem{
						JobId:     strconv.Itoa(int(j.ID)),
						JobType:   strings.ToLower(request.JobType),
						JobStatus: string(j.Status),
						CreatedAt: j.CreatedAt,
						UpdatedAt: j.UpdatedAt,
					})
				}
			case "analytics":
				jobs, err := h.DB.ListAnalyticsJobsByFilter(nil, []string{strings.ToUpper(status)}, nil, nil)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				for _, j := range jobs {
					items = append(items, api.ListJobsByTypeItem{
						JobId:     strconv.Itoa(int(j.ID)),
						JobType:   strings.ToLower(request.JobType),
						JobStatus: string(j.Status),
						CreatedAt: j.CreatedAt,
						UpdatedAt: j.UpdatedAt,
					})
				}
			}
		}
	case "benchmark":
		jobs, err := h.DB.ListComplianceJobsByBenchmarkID(request.Benchmark)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		for _, j := range jobs {
			items = append(items, api.ListJobsByTypeItem{
				JobId:     strconv.Itoa(int(j.ID)),
				JobType:   strings.ToLower(request.JobType),
				JobStatus: string(j.Status),
				CreatedAt: j.CreatedAt,
				UpdatedAt: j.UpdatedAt,
			})
		}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid selector. valid values: job_id, integration_info, status, benchmark")
	}

	switch sortBy {
	case api.JobSort_ByJobID:
		sort.Slice(items, func(i, j int) bool {
			return items[i].JobId < items[j].JobId
		})
	case api.JobSort_ByCreatedAt:
		sort.Slice(items, func(i, j int) bool {
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		})
	case api.JobSort_ByUpdatedAt:
		sort.Slice(items, func(i, j int) bool {
			return items[i].UpdatedAt.Before(items[j].UpdatedAt)
		})
	case api.JobSort_ByStatus:
		sort.Slice(items, func(i, j int) bool {
			return items[i].JobStatus < items[j].JobStatus
		})
	}

	if request.SortOrder == api.JobSortOrder_DESC {
		sort.Slice(items, func(i, j int) bool {
			return i > j
		})
	}

	totalCount := len(items)

	if request.PerPage != nil {
		if request.Cursor == nil {
			items = utils.Paginate(1, *request.PerPage, items)
		} else {
			items = utils.Paginate(*request.Cursor, *request.PerPage, items)
		}
	}

	response := api.ListJobsByTypeResponse{
		Items:      items,
		TotalCount: totalCount,
	}

	return ctx.JSON(http.StatusOK, response)
}

// ListJobsInterval godoc
//
//	@Summary	List jobs by job type and filters
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		job_type		query	string	true	"Job Type"
//	@Param		interval		query	string	true	"Time Interval to filter by"
//	@Param		trigger_type	query	string	true	"Trigger Type: (all(default), manual, system)"
//	@Param		created_by		query	string	true	"Created By User ID"
//	@Param		cursor			query	int		true	"cursor"
//	@Param		per_page		query	int		true	"per page"
//	@Produce	json
//	@Success	200	{object}	[]api.ListJobsByTypeItem
//	@Router		/schedule/api/v3/jobs/interval [get]
func (h HttpServer) ListJobsInterval(ctx echo.Context) error {
	jobType := ctx.QueryParam("job_type")
	interval := ctx.QueryParam("interval")
	triggerType := ctx.QueryParam("trigger_type")
	createdBy := ctx.QueryParam("created_by")

	var cursor, perPage int64
	var err error
	cursorStr := ctx.QueryParam("cursor")
	if cursorStr != "" {
		cursor, err = strconv.ParseInt(cursorStr, 10, 64)
		if err != nil {
			return err
		}
	}
	perPageStr := ctx.QueryParam("per_page")
	if perPageStr != "" {
		perPage, err = strconv.ParseInt(perPageStr, 10, 64)
		if err != nil {
			return err
		}
	}

	convertedInterval, err := convertInterval(interval)
	if err != nil {
		h.Scheduler.logger.Error("invalid interval", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid interval")
	}

	if strings.ToLower(jobType) != "compliance" && strings.ToLower(jobType) != "discovery" &&
		strings.ToLower(jobType) != "analytics" && strings.ToLower(jobType) != "query" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job type")
	}

	var items []api.ListJobsByTypeItem

	switch strings.ToLower(jobType) {
	case "compliance":
		jobs, err := h.DB.ListComplianceJobsForInterval(convertedInterval, triggerType, createdBy)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		for _, j := range jobs {
			items = append(items, api.ListJobsByTypeItem{
				JobId:     strconv.Itoa(int(j.ID)),
				JobType:   strings.ToLower(jobType),
				JobStatus: string(j.Status),
				CreatedAt: j.CreatedAt,
				UpdatedAt: j.UpdatedAt,
			})
		}
	case "discovery":
		jobs, err := h.DB.ListDescribeJobsForInterval(convertedInterval, triggerType, createdBy)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		for _, j := range jobs {
			items = append(items, api.ListJobsByTypeItem{
				JobId:     strconv.Itoa(int(j.ID)),
				JobType:   strings.ToLower(jobType),
				JobStatus: string(j.Status),
				CreatedAt: j.CreatedAt,
				UpdatedAt: j.UpdatedAt,
			})
		}
	case "analytics":
		jobs, err := h.DB.ListAnalyticsJobsForInterval(convertedInterval)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		for _, j := range jobs {
			items = append(items, api.ListJobsByTypeItem{
				JobId:     strconv.Itoa(int(j.ID)),
				JobType:   strings.ToLower(jobType),
				JobStatus: string(j.Status),
				CreatedAt: j.CreatedAt,
				UpdatedAt: j.UpdatedAt,
			})
		}
	case "query":
		jobs, err := h.DB.ListQueryRunnerJobForInterval(convertedInterval, triggerType, createdBy)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		for _, j := range jobs {
			items = append(items, api.ListJobsByTypeItem{
				JobId:     strconv.Itoa(int(j.ID)),
				JobType:   strings.ToLower(jobType),
				JobStatus: string(j.Status),
				CreatedAt: j.CreatedAt,
				UpdatedAt: j.UpdatedAt,
			})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].JobId > items[j].JobId
	})

	totalCount := len(items)
	if perPage != 0 {
		if cursor == 0 {
			items = utils.Paginate(1, perPage, items)
		} else {
			items = utils.Paginate(cursor, perPage, items)
		}
	}

	return ctx.JSON(http.StatusOK, api.ListJobsIntervalResponse{
		TotalCount: totalCount,
		Items:      items,
	})
}

// GetSummaryJobs godoc
//
//	@Summary	List jobs by job type and filters
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		job_ids		query	[]string	true	"Compliance Job ID"
//	@Produce	json
//	@Success	200	{object}	[]string
//	@Router		/schedule/api/v3/jobs/compliance/summary/jobs [get]
func (h HttpServer) GetSummaryJobs(ctx echo.Context) error {
	jobIds := httpserver.QueryArrayParam(ctx, "job_ids")

	jobs, err := h.DB.ListSummaryJobs(jobIds)
	if err != nil {
		h.Scheduler.logger.Error("could not get summary jobs", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "could not get summary jobs")
	}

	var summaryJobIDs []string
	for _, j := range jobs {
		summaryJobIDs = append(summaryJobIDs, strconv.Itoa(int(j.ID)))
	}

	return ctx.JSON(http.StatusOK, summaryJobIDs)
}

func convertInterval(input string) (string, error) {
	if input == "" {
		return "", nil
	}
	// Define regex to match shorthand formats like 90m, 50s, 1h
	re := regexp.MustCompile(`^(\d+)([smhd])$`)

	// Check if the input matches the shorthand format
	matches := re.FindStringSubmatch(input)
	if len(matches) == 3 {
		number := matches[1]
		unit := matches[2]

		// Map shorthand units to full words
		unitMap := map[string]string{
			"s": "seconds",
			"m": "minutes",
			"h": "hours",
			"d": "days",
		}

		// Convert the shorthand unit to full word
		if fullUnit, ok := unitMap[unit]; ok {
			return fmt.Sprintf("%s %s", number, fullUnit), nil
		}
	}

	// If the input doesn't match shorthand, assume it's already in the correct format
	if strings.Contains(input, "minute") || strings.Contains(input, "second") || strings.Contains(input, "hour") || strings.Contains(input, "day") {
		return input, nil
	}

	// If the input is invalid, return an error
	return "", fmt.Errorf("invalid interval format: %s", input)
}

// RunQuery godoc
//
//	@Summary	List jobs by job type and filters
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		query_id	path	string	true	"Query ID"
//	@Produce	json
//	@Success	200	{object}	api.RunQueryResponse
//	@Router		/schedule/api/v3/query/{query_id}/run [put]
func (h *HttpServer) RunQuery(ctx echo.Context) error {
	queryId := ctx.Param("query_id")

	userID := httpserver.GetUserID(ctx)
	if userID == "" {
		userID = "system"
	}

	job := &model2.QueryRunnerJob{
		QueryId:        queryId,
		Status:         queryrunner.QueryRunnerCreated,
		CreatedBy:      userID,
		FailureMessage: "",
		RetryCount:     0,
	}
	jobId, err := h.DB.CreateQueryRunnerJob(job)
	if err != nil {
		h.Scheduler.logger.Error("failed to create query runner job", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create query runner job")
	}

	response := api.RunQueryResponse{
		ID:        jobId,
		QueryId:   queryId,
		CreatedAt: job.CreatedAt,
		CreatedBy: userID,
		Status:    job.Status,
	}
	return ctx.JSON(http.StatusOK, response)
}

// PurgeSampleData godoc
//
//	@Summary		List all workspaces with owner id
//	@Description	Returns all workspaces with owner id
//	@Security		BearerToken
//	@Tags			workspace
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/schedule/api/v3/sample/purge [put]
func (s *HttpServer) PurgeSampleData(c echo.Context) error {
	err := s.DB.CleanupAllDescribeConnectionJobs()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete describe connection jobs")
	}
	err = s.DB.CleanupAllQueryRunnerJobs()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete query runner jobs")
	}
	err = s.DB.CleanupAllComplianceJobs()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete compliance jobs")
	}
	err = s.DB.CleanupAllComplianceSummarizerJobs()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete compliance summarizer jobs")
	}
	err = s.DB.CleanupAllComplianceRunners()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete compliance runners")
	}
	err = s.DB.CleanupAllCheckupJobs()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete checkup jobs")
	}
	err = s.DB.CleanupAllAnalyticsJobs()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete analytics jobs")
	}

	return c.NoContent(http.StatusOK)
}

// GetIntegrationDiscoveryProgress godoc
//
//	@Summary	Get Integration discovery progress (number of jobs in different states)
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		request	body	api.GetIntegrationDiscoveryProgressRequest	true	"List jobs request"
//	@Produce	json
//	@Success	200	{object}	api.GetIntegrationDiscoveryProgressResponse
//	@Router		/schedule/api/v3/discovery/status [post]
func (h HttpServer) GetIntegrationDiscoveryProgress(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	var request api.GetIntegrationDiscoveryProgressRequest
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
		connectionsTmp, err := h.Scheduler.onboardClient.ListSourcesByFilters(clientCtx,
			onboardapi.GetSourceByFiltersRequest{
				Connector:         info.Integration,
				ProviderNameRegex: info.IDName,
				ProviderIdRegex:   info.ID,
			})
		if err != nil {
			h.Scheduler.logger.Error("failed to list connections", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		connections = append(connections, connectionsTmp...)
	}
	var err error
	if len(connections) == 0 {
		connections, err = h.Scheduler.onboardClient.ListSources(clientCtx, nil)
		if err != nil {
			h.Scheduler.logger.Error("failed to list connections", zap.Error(err))
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
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

	integrationDiscoveries, err := h.DB.ListIntegrationDiscovery(request.TriggerID, connectionIDs)
	if err != nil {
		h.Scheduler.logger.Error("cannot find integration discoveries", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot find integration discoveries")
	}
	var integrationDiscoveriesIds []string
	for _, i := range integrationDiscoveries {
		integrationDiscoveriesIds = append(integrationDiscoveriesIds, strconv.Itoa(int(i.ID)))
	}

	jobs, err := h.DB.ListDescribeJobsByFilters(integrationDiscoveriesIds, nil, nil,
		nil, nil, nil, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	triggerIdProgressBreakdown := &api.DiscoveryProgressStatusBreakdown{}
	triggerIdProgressSummary := &api.DiscoveryProgressStatusSummary{}
	integrationsDiscoveryProgressStatus := make(map[string]api.IntegrationDiscoveryProgressStatus)
	for _, j := range jobs {
		if _, ok := integrationsDiscoveryProgressStatus[j.ConnectionID]; !ok {
			integrationsDiscoveryProgressStatus[j.ConnectionID] = api.IntegrationDiscoveryProgressStatus{
				Integration:             connectionInfo[j.ConnectionID],
				ProgressStatusBreakdown: &api.DiscoveryProgressStatusBreakdown{},
				ProgressStatusSummary:   &api.DiscoveryProgressStatusSummary{},
			}
		}
		switch j.Status {
		case api.DescribeResourceJobCreated:
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.CreatedCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.CreatedCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount + 1
			triggerIdProgressBreakdown.CreatedCount = triggerIdProgressBreakdown.CreatedCount + 1
			triggerIdProgressSummary.TotalCount = triggerIdProgressSummary.TotalCount + 1
		case api.DescribeResourceJobQueued:
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.QueuedCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.QueuedCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount + 1
			triggerIdProgressBreakdown.QueuedCount = triggerIdProgressBreakdown.QueuedCount + 1
			triggerIdProgressSummary.TotalCount = triggerIdProgressSummary.TotalCount + 1
		case api.DescribeResourceJobInProgress:
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.InProgressCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.InProgressCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount + 1
			triggerIdProgressBreakdown.InProgressCount = triggerIdProgressBreakdown.InProgressCount + 1
			triggerIdProgressSummary.TotalCount = triggerIdProgressSummary.TotalCount + 1
		case api.DescribeResourceJobOldResourceDeletion:
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.OldResourceDeletionCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.OldResourceDeletionCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount + 1
			triggerIdProgressBreakdown.OldResourceDeletionCount = triggerIdProgressBreakdown.OldResourceDeletionCount + 1
			triggerIdProgressSummary.TotalCount = triggerIdProgressSummary.TotalCount + 1
		case api.DescribeResourceJobTimeout:
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.TimeoutCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.TimeoutCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.ProcessedCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.ProcessedCount + 1
			triggerIdProgressBreakdown.TimeoutCount = triggerIdProgressBreakdown.TimeoutCount + 1
			triggerIdProgressSummary.TotalCount = triggerIdProgressSummary.TotalCount + 1
			triggerIdProgressSummary.ProcessedCount = triggerIdProgressSummary.ProcessedCount + 1
		case api.DescribeResourceJobFailed:
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.FailedCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.FailedCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.ProcessedCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.ProcessedCount + 1
			triggerIdProgressBreakdown.FailedCount = triggerIdProgressBreakdown.FailedCount + 1
			triggerIdProgressSummary.TotalCount = triggerIdProgressSummary.TotalCount + 1
			triggerIdProgressSummary.ProcessedCount = triggerIdProgressSummary.ProcessedCount + 1
		case api.DescribeResourceJobSucceeded:
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.SucceededCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.SucceededCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.ProcessedCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.ProcessedCount + 1
			triggerIdProgressBreakdown.SucceededCount = triggerIdProgressBreakdown.SucceededCount + 1
			triggerIdProgressSummary.TotalCount = triggerIdProgressSummary.TotalCount + 1
			triggerIdProgressSummary.ProcessedCount = triggerIdProgressSummary.ProcessedCount + 1
		case api.DescribeResourceJobRemovingResources:
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.RemovingResourcesCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.RemovingResourcesCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount + 1
			triggerIdProgressBreakdown.RemovingResourcesCount = triggerIdProgressBreakdown.RemovingResourcesCount + 1
			triggerIdProgressSummary.TotalCount = triggerIdProgressSummary.TotalCount + 1
		case api.DescribeResourceJobCanceled:
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.CanceledCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusBreakdown.CanceledCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.TotalCount + 1
			integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.ProcessedCount = integrationsDiscoveryProgressStatus[j.ConnectionID].ProgressStatusSummary.ProcessedCount + 1
			triggerIdProgressBreakdown.CanceledCount = triggerIdProgressBreakdown.CanceledCount + 1
			triggerIdProgressSummary.TotalCount = triggerIdProgressSummary.TotalCount + 1
			triggerIdProgressSummary.ProcessedCount = triggerIdProgressSummary.ProcessedCount + 1
		}
	}

	var integrationsDiscoveryProgressStatusResult []api.IntegrationDiscoveryProgressStatus
	for _, v := range integrationsDiscoveryProgressStatus {
		integrationsDiscoveryProgressStatusResult = append(integrationsDiscoveryProgressStatusResult, v)
	}

	response := api.GetIntegrationDiscoveryProgressResponse{
		IntegrationProgress:        integrationsDiscoveryProgressStatusResult,
		TriggerIdProgressBreakdown: triggerIdProgressBreakdown,
		TriggerIdProgressSummary:   triggerIdProgressSummary,
	}

	return ctx.JSON(http.StatusOK, response)
}

// ListComplianceJobsHistory godoc
//
//	@Summary	List jobs by job type and filters
//	@Security	BearerToken
//	@Tags		scheduler
//	@Param		interval		query	string		true	"Time Interval to filter by"
//	@Param		trigger_type	query	string		true	"Trigger Type: (all(default), manual, system)"
//	@Param		created_by		query	string		true	"Created By User ID"
//	@Param		benchmark_ids	query	[]string	true	"Created By User ID"
//	@Param		cursor			query	int			true	"cursor"
//	@Param		per_page		query	int			true	"per page"
//	@Produce	json
//	@Success	200	{object}	api.ListComplianceJobsHistoryResponse
//	@Router		/schedule/api/v3/jobs/history/compliance [get]
func (h HttpServer) ListComplianceJobsHistory(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}

	interval := ctx.QueryParam("interval")
	triggerType := ctx.QueryParam("trigger_type")
	createdBy := ctx.QueryParam("created_by")
	benchmarkIDs := httpserver.QueryArrayParam(ctx, "benchmark_ids")

	var cursor, perPage int64
	var err error
	cursorStr := ctx.QueryParam("cursor")
	if cursorStr != "" {
		cursor, err = strconv.ParseInt(cursorStr, 10, 64)
		if err != nil {
			return err
		}
	}
	perPageStr := ctx.QueryParam("per_page")
	if perPageStr != "" {
		perPage, err = strconv.ParseInt(perPageStr, 10, 64)
		if err != nil {
			return err
		}
	}

	convertedInterval, err := convertInterval(interval)
	if err != nil {
		h.Scheduler.logger.Error("invalid interval", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid interval")
	}

	var items []api.ListComplianceJobsHistoryItem
	connectionIdsMap := make(map[string]bool)

	jobs, err := h.DB.ListComplianceJobsWithSummaryJob(convertedInterval, triggerType, createdBy, benchmarkIDs)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	for _, j := range jobs {
		var integrations []api.IntegrationInfo
		for _, c := range j.ConnectionIDs {
			connectionIdsMap[c] = true
			integrations = append(integrations, api.IntegrationInfo{
				IntegrationTracker: c,
			})
		}
		items = append(items, api.ListComplianceJobsHistoryItem{
			BenchmarkId:    j.BenchmarkID,
			JobId:          strconv.Itoa(int(j.ID)),
			SummarizerJobs: j.SummarizerJobs,
			TriggerType:    string(j.TriggerType),
			CreatedBy:      j.CreatedBy,
			JobStatus:      string(j.Status),
			CreatedAt:      j.CreatedAt,
			UpdatedAt:      j.UpdatedAt,
		})
	}
	var connectionIds []string
	for k, _ := range connectionIdsMap {
		connectionIds = append(connectionIds, k)
	}

	connections, err := h.Scheduler.onboardClient.GetSources(clientCtx, connectionIds)
	if err != nil {
		h.Scheduler.logger.Error("failed to get connections", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get integrations info")
	}
	integrationsMap := make(map[string]api.IntegrationInfo)
	for _, c := range connections {
		integrationsMap[c.ID.String()] = api.IntegrationInfo{
			Integration:        c.Connector.String(),
			ID:                 c.ConnectionID,
			IDName:             c.ConnectionName,
			IntegrationTracker: c.ID.String(),
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].JobId > items[j].JobId
	})

	totalCount := len(items)
	if perPage != 0 {
		if cursor == 0 {
			items = utils.Paginate(1, perPage, items)
		} else {
			items = utils.Paginate(cursor, perPage, items)
		}
	}

	for i, j := range items {
		for ii, integration := range j.Integrations {
			if integrationData, ok := integrationsMap[integration.IntegrationTracker]; ok {
				items[i].Integrations[ii] = integrationData
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].JobId > items[j].JobId
	})
	return ctx.JSON(http.StatusOK, api.ListComplianceJobsHistoryResponse{
		TotalCount: totalCount,
		Items:      items,
	})
}

func parseTimeInterval(intervalStr string) (*time.Time, *time.Time, error) {
	// Define regex patterns to extract the time components
	patterns := map[string]*regexp.Regexp{
		"days":    regexp.MustCompile(`(\d+)\s*days?`),
		"hours":   regexp.MustCompile(`(\d+)\s*hours?`),
		"minutes": regexp.MustCompile(`(\d+)\s*minutes?`),
		"seconds": regexp.MustCompile(`(\d+)\s*seconds?`),
	}

	// Variables to store the extracted values
	days, hours, minutes, seconds := 0, 0, 0, 0

	// Extract and convert the values from the string
	for key, pattern := range patterns {
		match := pattern.FindStringSubmatch(intervalStr)
		if len(match) > 1 {
			value, err := strconv.Atoi(match[1])
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing %s: %v", key, err)
			}
			switch key {
			case "days":
				days = value
			case "hours":
				hours = value
			case "minutes":
				minutes = value
			case "seconds":
				seconds = value
			}
		}
	}

	// Calculate total duration based on extracted values
	duration := time.Duration(days)*24*time.Hour +
		time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second

	// Calculate endTime as now and startTime by subtracting the duration
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	return &startTime, &endTime, nil
}
