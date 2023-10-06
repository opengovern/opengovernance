package describe

import (
	"encoding/json"
	"errors"
	"fmt"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/terraform-package/external/states/statefile"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/lib/pq"
	"github.com/sony/sonyflake"
	"go.uber.org/zap"

	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"

	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	complianceapi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
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
	DB         Database
	Scheduler  *Scheduler
	kubeClient k8sclient.Client
	helmConfig HelmConfig
}

func NewHTTPServer(
	address string,
	db Database,
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

	v1.PUT("/describe/trigger/:connection_id", httpserver.AuthorizeHandler(h.TriggerPerConnectionDescribeJob, apiAuth.AdminRole))
	v1.PUT("/describe/trigger", httpserver.AuthorizeHandler(h.TriggerDescribeJob, apiAuth.InternalRole))
	v1.PUT("/insight/trigger/:insight_id", httpserver.AuthorizeHandler(h.TriggerInsightJob, apiAuth.AdminRole))
	v1.PUT("/compliance/trigger/:benchmark_id", httpserver.AuthorizeHandler(h.TriggerComplianceJob, apiAuth.AdminRole))
	v1.PUT("/analytics/trigger", httpserver.AuthorizeHandler(h.TriggerAnalyticsJob, apiAuth.InternalRole))
	v1.PUT("/summarize/trigger", httpserver.AuthorizeHandler(h.TriggerSummarizeJob, apiAuth.InternalRole))
	v1.GET("/describe/status/:resource_type", httpserver.AuthorizeHandler(h.GetDescribeStatus, apiAuth.InternalRole))
	v1.GET("/describe/connection/status", httpserver.AuthorizeHandler(h.GetConnectionDescribeStatus, apiAuth.InternalRole))
	v1.GET("/describe/pending/connections", httpserver.AuthorizeHandler(h.ListAllPendingConnection, apiAuth.InternalRole))

	stacks := v1.Group("/stacks")
	stacks.GET("", httpserver.AuthorizeHandler(h.ListStack, apiAuth.ViewerRole))
	stacks.GET("/:stackId", httpserver.AuthorizeHandler(h.GetStack, apiAuth.ViewerRole))
	stacks.POST("/create", httpserver.AuthorizeHandler(h.CreateStack, apiAuth.AdminRole))
	stacks.DELETE("/:stackId", httpserver.AuthorizeHandler(h.DeleteStack, apiAuth.AdminRole))
	stacks.POST("/:stackId/findings", httpserver.AuthorizeHandler(h.GetStackFindings, apiAuth.ViewerRole))
	stacks.GET("/:stackId/insight", httpserver.AuthorizeHandler(h.GetStackInsight, apiAuth.ViewerRole))
	stacks.GET("/resource", httpserver.AuthorizeHandler(h.ListResourceStack, apiAuth.ViewerRole))
	stacks.POST("/describer/trigger", httpserver.AuthorizeHandler(h.TriggerStackDescriber, apiAuth.AdminRole))
	stacks.GET("/:stackId/insights", httpserver.AuthorizeHandler(h.ListStackInsights, apiAuth.ViewerRole))
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

	src, err := h.Scheduler.onboardClient.GetSource(&httpclient.Context{UserRole: apiAuth.KaytuAdminRole}, connectionID)
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

	err = h.DB.CreateJobSequencer(&JobSequencer{
		DependencyList:   dependencyIDs,
		DependencySource: string(JobSequencerJobTypeDescribe),
		NextJob:          string(JobSequencerJobTypeAnalytics),
		Status:           JobSequencerWaitingForDependencies,
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

	err := h.Scheduler.CheckWorkspaceResourceLimit()
	if err != nil {
		h.Scheduler.logger.Error("failed to get limits", zap.String("spot", "CheckWorkspaceResourceLimit"), zap.Error(err))
		DescribeJobsCount.WithLabelValues("failure").Inc()
		if err == ErrMaxResourceCountExceeded {
			return ctx.JSON(http.StatusNotAcceptable, api.ErrorResponse{Message: err.Error()})
		}
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: err.Error()})
	}

	connections, err := h.Scheduler.onboardClient.ListSources(&httpclient.Context{UserRole: apiAuth.KaytuAdminRole}, connectors)
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
//	@Success		200
//	@Param			insight_id	path	string	true	"Insight ID"
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
	for _, ins := range insights {
		if ins.ID != uint(insightID) {
			continue
		}

		id := fmt.Sprintf("all:%s", strings.ToLower(string(ins.Connector)))
		err := h.Scheduler.runInsightJob(true, ins, id, id, ins.Connector)
		if err != nil {
			return err
		}
	}
	return ctx.JSON(http.StatusOK, "")
}

// TriggerComplianceJob godoc
//
//	@Summary		Triggers compliance job
//	@Description	Triggers a compliance job to run immediately for the given benchmark
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Param			benchmark_id	path	string	true	"Benchmark ID"
//	@Router			/schedule/api/v1/compliance/trigger/{benchmark_id} [put]
func (h HttpServer) TriggerComplianceJob(ctx echo.Context) error {
	clientCtx := &httpclient.Context{UserRole: apiAuth.InternalRole}
	benchmarkID := ctx.Param("benchmark_id")
	benchmark, err := h.Scheduler.complianceClient.GetBenchmark(clientCtx, benchmarkID)
	if err != nil {
		return fmt.Errorf("error while getting benchmarks: %v", err)
	}

	if benchmark == nil {
		return echo.NewHTTPError(http.StatusNotFound, "benchmark not found")
	}

	var sources []onboardApi.Connection
	assignments, err := h.Scheduler.complianceClient.ListAssignmentsByBenchmark(clientCtx, benchmark.ID)
	if err != nil {
		return fmt.Errorf("error while listing assignments: %v", err)
	}

	for _, ass := range assignments {
		if !ass.Status {
			continue
		}

		src, err := h.Scheduler.onboardClient.GetSource(clientCtx, ass.ConnectionID)
		if err != nil {
			return fmt.Errorf("error while get source: %v", err)
		}

		if !src.IsEnabled() {
			continue
		}
		sources = append(sources, *src)
	}

	var dependencyIDs []int64
	for _, src := range sources {
		crj := newComplianceReportJob(src.ID.String(), src.Connector, benchmark.ID)
		err = h.DB.CreateComplianceReportJob(&crj)
		if err != nil {
			ComplianceSourceJobsCount.WithLabelValues("failure").Inc()
			return fmt.Errorf("error while creating compliance job: %v", err)
		}
		enqueueComplianceReportJobs(h.Scheduler.logger, h.DB, h.Scheduler.complianceReportJobQueue, src, &crj)
		ComplianceSourceJobsCount.WithLabelValues("successful").Inc()
		dependencyIDs = append(dependencyIDs, int64(crj.ID))
	}

	err = h.DB.CreateJobSequencer(&JobSequencer{
		DependencyList:   dependencyIDs,
		DependencySource: string(JobSequencerJobTypeBenchmark),
		NextJob:          string(JobSequencerJobTypeBenchmarkSummarizer),
		Status:           JobSequencerWaitingForDependencies,
	})
	if err != nil {
		return fmt.Errorf("failed to create job sequencer: %v", err)
	}

	return ctx.JSON(http.StatusOK, "")
}

func (h HttpServer) TriggerSummarizeJob(ctx echo.Context) error {
	err := h.Scheduler.scheduleMustSummarizerJob()
	if err != nil {
		errMsg := fmt.Sprintf("error scheduling summarize job: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: errMsg})
	}
	return ctx.JSON(http.StatusOK, "")
}

func (h HttpServer) TriggerAnalyticsJob(ctx echo.Context) error {
	err := h.Scheduler.scheduleAnalyticsJob(nil)
	if err != nil {
		errMsg := fmt.Sprintf("error scheduling summarize job: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: errMsg})
	}
	return ctx.JSON(http.StatusOK, "")
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

	var recordTags []*StackTag
	if len(tags) != 0 {
		for key, value := range tags {
			recordTags = append(recordTags, &StackTag{
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

	stackRecord := Stack{
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
	tagMap := model.TagStringsToTagMap(httpserver.QueryArrayParam(ctx, "tag"))
	accountIds := httpserver.QueryArrayParam(ctx, "accountIds")
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
			Tags:          trimPrivateTags(sr.GetTagsMap()),
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
		Sorts: reqBody.Sorts,
		Page:  reqBody.Page,
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

	insightIds := httpserver.QueryArrayParam(ctx, "insightIds")
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
			Tags:          trimPrivateTags(sr.GetTagsMap()),
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
