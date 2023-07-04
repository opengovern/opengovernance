package describe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	describe2 "github.com/kaytu-io/kaytu-util/pkg/describe/enums"
	"github.com/lib/pq"

	"github.com/kaytu-io/kaytu-engine/pkg/describe/enums"

	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"

	api3 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/cloudservice"
	complianceapi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	insightapi "github.com/kaytu-io/kaytu-engine/pkg/insight/api"
	summarizerapi "github.com/kaytu-io/kaytu-engine/pkg/summarizer/api"

	"gorm.io/gorm"

	"github.com/google/uuid"
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
	v0 := e.Group("/api/v0") // experimental/debug apis
	v1 := e.Group("/api/v1")

	v0.GET("/describe/trigger", httpserver.AuthorizeHandler(h.TriggerDescribeJob, api3.AdminRole))
	v0.GET("/summarize/trigger", httpserver.AuthorizeHandler(h.TriggerSummarizeJob, api3.AdminRole))
	v0.GET("/insight/trigger", httpserver.AuthorizeHandler(h.TriggerInsightJob, api3.AdminRole))
	v0.GET("/compliance/trigger", httpserver.AuthorizeHandler(h.TriggerComplianceJob, api3.AdminRole))
	v0.GET("/compliance/summarizer/trigger", httpserver.AuthorizeHandler(h.TriggerComplianceSummarizerJob, api3.AdminRole))
	v1.PUT("/benchmark/evaluation/trigger", httpserver.AuthorizeHandler(h.TriggerBenchmarkEvaluation, api3.AdminRole))
	v1.PUT("/insight/evaluation/trigger", httpserver.AuthorizeHandler(h.TriggerInsightEvaluation, api3.AdminRole))
	v1.PUT("/describe/trigger/:connection_id", httpserver.AuthorizeHandler(h.TriggerDescribeJobV1, api3.AdminRole))

	v1.GET("/insight/job/:jobId", httpserver.AuthorizeHandler(h.GetInsightJob, api3.ViewerRole))

	v1.GET("/describe/source/jobs/pending", httpserver.AuthorizeHandler(h.HandleListPendingDescribeSourceJobs, api3.ViewerRole))
	v1.GET("/describe/resource/jobs/pending", httpserver.AuthorizeHandler(h.HandleListPendingDescribeResourceJobs, api3.ViewerRole))
	v1.GET("/summarize/jobs/pending", httpserver.AuthorizeHandler(h.HandleListPendingSummarizeJobs, api3.ViewerRole))
	v1.GET("/insight/jobs/pending", httpserver.AuthorizeHandler(h.HandleListPendingInsightJobs, api3.ViewerRole))

	v1.GET("/sources", httpserver.AuthorizeHandler(h.HandleListSources, api3.ViewerRole))
	v1.GET("/sources/:source_id", httpserver.AuthorizeHandler(h.HandleGetSource, api3.ViewerRole))
	v1.GET("/sources/:source_id/jobs/describe", httpserver.AuthorizeHandler(h.HandleListSourceDescribeJobs, api3.ViewerRole))
	v1.GET("/sources/:source_id/jobs/compliance", httpserver.AuthorizeHandler(h.HandleListSourceComplianceReports, api3.ViewerRole))

	v1.POST("/sources/:source_id/jobs/describe/refresh", httpserver.AuthorizeHandler(h.RunDescribeJobs, api3.EditorRole))
	v1.POST("/sources/:source_id/jobs/compliance/refresh", httpserver.AuthorizeHandler(h.RunComplianceReportJobs, api3.EditorRole))

	v1.GET("/resource_type/:provider", httpserver.AuthorizeHandler(h.GetResourceTypesByProvider, api3.ViewerRole))

	v1.GET("/compliance/report/last/completed", httpserver.AuthorizeHandler(h.HandleGetLastCompletedComplianceReport, api3.ViewerRole))

	v1.GET("/benchmark/evaluations", httpserver.AuthorizeHandler(h.HandleListBenchmarkEvaluations, api3.ViewerRole))

	v1.POST("/describe/resource", httpserver.AuthorizeHandler(h.DescribeSingleResource, api3.AdminRole))

	stacks := v1.Group("/stacks")
	stacks.POST("/benchmark/trigger", httpserver.AuthorizeHandler(h.TriggerStackBenchmark, api3.AdminRole))
	stacks.GET("", httpserver.AuthorizeHandler(h.ListStack, api3.ViewerRole))
	stacks.GET("/:stackId", httpserver.AuthorizeHandler(h.GetStack, api3.ViewerRole))
	stacks.POST("/create", httpserver.AuthorizeHandler(h.CreateStack, api3.AdminRole))
	stacks.DELETE("/:stackId", httpserver.AuthorizeHandler(h.DeleteStack, api3.AdminRole))
	stacks.POST("/:stackId/findings", httpserver.AuthorizeHandler(h.GetStackFindings, api3.ViewerRole))
	stacks.GET("/:stackId/insight", httpserver.AuthorizeHandler(h.GetStackInsight, api3.ViewerRole))
	stacks.GET("/resource", httpserver.AuthorizeHandler(h.ListResourceStack, api3.ViewerRole))
	stacks.POST("/insight/trigger", httpserver.AuthorizeHandler(h.TriggerStackInsight, api3.AdminRole))
	stacks.POST("/describer/trigger", httpserver.AuthorizeHandler(h.TriggerStackDescriber, api3.AdminRole))
}

// HandleListSources godoc
//
//	@Summary		List Sources
//	@Description	Retrieves list of all of Keibi sources
//	@Security		BearerToken
//	@Tags			schedule
//	@Produce		json
//	@Success		200	{object}	[]api.Source
//	@Router			/schedule/api/v1/sources [get]
func (h HttpServer) HandleListSources(ctx echo.Context) error {
	sources, err := h.DB.ListSources()
	if err != nil {
		ctx.Logger().Errorf("fetching sources: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "internal error"})
	}

	var objs []api.Source
	for _, source := range sources {
		lastDescribeAt := time.Time{}
		lastComplianceReportAt := time.Time{}
		if source.LastDescribedAt.Valid {
			lastDescribeAt = source.LastDescribedAt.Time
		}
		if source.LastComplianceReportAt.Valid {
			lastComplianceReportAt = source.LastComplianceReportAt.Time
		}

		job, err := h.DB.GetLastDescribeSourceJob(source.ID)
		if err != nil {
			ctx.Logger().Errorf("fetching source last describe job %s: %v", source.ID, err)
			return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "fetching source last describe job"})
		}
		lastJobStatus := ""
		if job != nil {
			lastJobStatus = string(job.Status)
		}

		objs = append(objs, api.Source{
			ID:                     source.ID,
			Type:                   source.Type,
			LastDescribedAt:        lastDescribeAt,
			LastComplianceReportAt: lastComplianceReportAt,
			LastDescribeJobStatus:  lastJobStatus,
		})
	}

	return ctx.JSON(http.StatusOK, objs)
}

// HandleGetSource godoc
//
//	@Summary		Get source
//	@Description	Retrieves Keibi source details by id
//	@Security		BearerToken
//	@Description	Getting Keibi source by id
//	@Tags			schedule
//	@Produce		json
//	@Param			source_id	path		string	true	"SourceID"
//	@Success		200			{object}	api.Source
//	@Router			/schedule/api/v1/sources/{source_id} [get]
func (h HttpServer) HandleGetSource(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid source uuid"})
	}
	source, err := h.DB.GetSourceByUUID(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("fetching source %s: %v", sourceID, err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "fetching source"})
	}

	job, err := h.DB.GetLastDescribeSourceJob(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("fetching source last describe job %s: %v", sourceID, err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "fetching source last describe job"})
	}

	lastDescribeAt := time.Time{}
	lastComplianceReportAt := time.Time{}
	if source.LastDescribedAt.Valid {
		lastDescribeAt = source.LastDescribedAt.Time
	}
	if source.LastComplianceReportAt.Valid {
		lastComplianceReportAt = source.LastComplianceReportAt.Time
	}
	lastJobStatus := ""
	if job != nil {
		lastJobStatus = string(job.Status)
	}

	return ctx.JSON(http.StatusOK, api.Source{
		ID:                     source.ID,
		AccountID:              source.AccountID,
		Type:                   source.Type,
		LastDescribedAt:        lastDescribeAt,
		LastComplianceReportAt: lastComplianceReportAt,
		LastDescribeJobStatus:  lastJobStatus,
	})
}

// HandleListPendingDescribeSourceJobs godoc
//
//	@Summary	Listing describe source jobs
//	@Security	BearerToken
//	@Tags		schedule
//	@Produce	json
//	@Success	200	{object}	[]DescribeSourceJob
//	@Router		/schedule/api/v1/describe/source/jobs/pending [get]
func (h HttpServer) HandleListPendingDescribeSourceJobs(ctx echo.Context) error {
	jobs, err := h.DB.ListPendingDescribeSourceJobs()
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, jobs)
}

// HandleListPendingDescribeResourceJobs godoc
//
//	@Summary	Listing describe resource jobs
//	@Security	BearerToken
//	@Tags		schedule
//	@Produce	json
//	@Success	200	{object}	[]DescribeResourceJob
//	@Router		/schedule/api/v1/describe/resource/jobs/pending [get]
func (h HttpServer) HandleListPendingDescribeResourceJobs(ctx echo.Context) error {
	jobs, err := h.DB.ListPendingDescribeResourceJobs()
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, jobs)
}

// HandleListPendingSummarizeJobs godoc
//
//	@Summary	Listing summarize jobs
//	@Security	BearerToken
//	@Tags		schedule
//	@Produce	json
//	@Success	200	{object}	[]SummarizerJob
//	@Router		/schedule/api/v1/summarize/jobs/pending [get]
func (h HttpServer) HandleListPendingSummarizeJobs(ctx echo.Context) error {
	jobs, err := h.DB.ListPendingSummarizeJobs()
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, jobs)
}

// HandleListPendingInsightJobs godoc
//
//	@Summary	Listing insight jobs
//	@Security	BearerToken
//	@Tags		schedule
//	@Produce	json
//	@Success	200	{object}	[]InsightJob
//	@Router		/schedule/api/v1/insight/jobs/pending [get]
func (h HttpServer) HandleListPendingInsightJobs(ctx echo.Context) error {
	jobs, err := h.DB.ListPendingInsightJobs()
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, jobs)
}

// HandleListSourceDescribeJobs godoc
//
//	@Summary		List source describe jobs
//	@Description	Retrieves list of source describe jobs for a source by the given source id
//	@Security		BearerToken
//	@Tags			schedule
//	@Produce		json
//	@Param			source_id	path		string	true	"SourceID"
//	@Success		200			{object}	[]api.DescribeSource
//	@Router			/schedule/api/v1/sources/{source_id}/jobs/describe [get]
func (h HttpServer) HandleListSourceDescribeJobs(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid source uuid"})
	}

	jobs, err := h.DB.ListDescribeSourceJobs(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("fetching describe source jobs: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "internal error"})
	}

	var objs []api.DescribeSource
	for _, job := range jobs {
		var describeResourceJobs []api.DescribeResource
		for _, describeResourceJob := range job.DescribeResourceJobs {
			describeResourceJobs = append(describeResourceJobs, api.DescribeResource{
				ResourceType:   describeResourceJob.ResourceType,
				Status:         describeResourceJob.Status,
				FailureMessage: describeResourceJob.FailureMessage,
			})
		}

		objs = append(objs, api.DescribeSource{
			DescribeResourceJobs: describeResourceJobs,
			Status:               job.Status,
		})
	}

	return ctx.JSON(http.StatusOK, objs)
}

// HandleListSourceComplianceReports godoc
//
//	@Summary		List source compliance reports
//	@Description	Retrieves list of source compliance reports for a source by the given source id
//	@Security		BearerToken
//	@Tags			schedule
//	@Produce		json
//	@Param			source_id	path		string	true	"SourceID"
//	@Param			from		query		int		false	"From Time (TimeRange)"
//	@Param			to			query		int		false	"To Time (TimeRange)"
//	@Success		200			{object}	[]complianceapi.ComplianceReport
//	@Router			/schedule/api/v1/sources/{source_id}/jobs/compliance [get]
func (h HttpServer) HandleListSourceComplianceReports(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid source uuid"})
	}

	from := ctx.QueryParam("from")
	to := ctx.QueryParam("to")

	var jobs []ComplianceReportJob
	if from == "" && to == "" {
		report, err := h.DB.GetLastCompletedSourceComplianceReport(sourceUUID)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: err.Error()})
		}
		if report != nil {
			jobs = append(jobs, *report)
		}
	} else if from == "" || to == "" {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "both from and to must be provided"})
	} else {
		n, err := strconv.ParseInt(from, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: err.Error()})
		}
		fromTime := time.UnixMilli(n)

		n, err = strconv.ParseInt(to, 10, 64)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: err.Error()})
		}
		toTime := time.UnixMilli(n)

		jobs, err = h.DB.ListCompletedComplianceReportByDate(sourceUUID, fromTime, toTime)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: err.Error()})
		}
	}

	var objs []complianceapi.ComplianceReport
	for _, job := range jobs {
		objs = append(objs, complianceapi.ComplianceReport{
			ID:              job.ID,
			UpdatedAt:       job.UpdatedAt,
			ReportCreatedAt: job.ReportCreatedAt,
			Status:          job.Status,
			FailureMessage:  job.FailureMessage,
		})
	}

	return ctx.JSON(http.StatusOK, objs)
}

// RunComplianceReportJobs godoc
//
//	@Summary		Run compliance report jobs
//	@Description	Run compliance report jobs
//	@Security		BearerToken
//	@Tags			schedule
//	@Produce		json
//	@Param			source_id	path	string	true	"SourceID"
//	@Success		200
//	@Router			/schedule/api/v1/sources/{source_id}/jobs/compliance/refresh [post]
func (h HttpServer) RunComplianceReportJobs(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid source uuid"})
	}

	err = h.DB.UpdateSourceNextComplianceReportToNow(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("update source next compliance report run: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "internal error"})
	}

	return ctx.String(http.StatusOK, "")
}

// HandleGetLastCompletedComplianceReport godoc
//
//	@Summary	Get last completed compliance report
//	@Security	BearerToken
//	@Tags		schedule
//	@Produce	json
//	@Success	200	{object}	int
//	@Router		/schedule/api/v1/compliance/report/last/completed [get]
func (h HttpServer) HandleGetLastCompletedComplianceReport(ctx echo.Context) error {
	id, err := h.DB.GetLastCompletedComplianceReportID()
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, id)
}

// RunDescribeJobs godoc
//
//	@Summary		Run describe jobs
//	@Description	Run describe jobs
//	@Security		BearerToken
//	@Tags			schedule
//	@Produce		json
//	@Param			source_id	path	string	true	"SourceID"
//	@Success		200
//	@Router			/schedule/api/v1/sources/{source_id}/jobs/describe/refresh [post]
func (h HttpServer) RunDescribeJobs(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid source uuid"})
	}

	err = h.DB.UpdateSourceNextDescribeAtToNow(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("update source next describe run: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "internal error"})
	}

	return ctx.String(http.StatusOK, "")
}

// GetResourceTypesByProvider godoc
//
//	@Summary		get resource type by provider
//	@Description	get resource type by provider
//	@Security		BearerToken
//	@Tags			schedule
//	@Produce		json
//	@Param			provider	path		string	true	"Provider"	Enums(aws,azure)
//	@Success		200			{object}	[]api.ResourceTypeDetail
//	@Router			/schedule/api/v1/resource_type/{provider} [get]
func (h HttpServer) GetResourceTypesByProvider(ctx echo.Context) error {
	provider := ctx.Param("provider")

	var resourceTypes []api.ResourceTypeDetail

	if provider == "azure" || provider == "all" {
		for _, resourceType := range azure.ListResourceTypes() {
			resourceTypes = append(resourceTypes, api.ResourceTypeDetail{
				ResourceTypeARN:  resourceType,
				ResourceTypeName: cloudservice.ResourceTypeName(resourceType),
			})
		}
	}
	if provider == "aws" || provider == "all" {
		for _, resourceType := range aws.ListResourceTypes() {
			resourceTypes = append(resourceTypes, api.ResourceTypeDetail{
				ResourceTypeARN:  resourceType,
				ResourceTypeName: cloudservice.ResourceTypeName(resourceType),
			})
		}
	}

	return ctx.JSON(http.StatusOK, resourceTypes)
}

// TriggerDescribeJob godoc
//
//	@Summary		Triggers a describe job to run immediately
//	@Description	Triggers a describe job to run immediately
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Router			/schedule/api/v0/describe/trigger [get]
func (h HttpServer) TriggerDescribeJob(ctx echo.Context) error {
	scheduleJob, err := h.DB.FetchLastScheduleJob()
	if err != nil {
		errMsg := fmt.Sprintf("error fetching last schedule job: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: errMsg})
	}
	if scheduleJob.Status == summarizerapi.SummarizerJobInProgress {
		return ctx.JSON(http.StatusConflict, api.ErrorResponse{Message: "schedule job in progress"})
	}
	job := ScheduleJob{
		Model:          gorm.Model{},
		Status:         summarizerapi.SummarizerJobInProgress,
		FailureMessage: "",
	}
	err = h.DB.AddScheduleJob(&job)
	if err != nil {
		errMsg := fmt.Sprintf("error adding schedule job: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: errMsg})
	}
	return ctx.JSON(http.StatusOK, "")
}

// TriggerDescribeJobV1 godoc
//
//	@Summary		Triggers a describe job to run immediately
//	@Description	Triggers a describe job to run immediately
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Param			connection_id	path	string	true	"Connection ID"
//	@Router			/schedule/api/v1/describe/trigger/{connection_id} [put]
func (h HttpServer) TriggerDescribeJobV1(ctx echo.Context) error {
	connectionID := ctx.Param("connection_id")

	src, err := h.DB.GetSourceByID(connectionID)
	if err != nil || src == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connection id")
	}

	err = h.Scheduler.describeConnection(*src, false)
	if err != nil {
		return err
	}
	return ctx.NoContent(http.StatusOK)
}

// TriggerSummarizeJob godoc
//
//	@Summary		Triggers a summarize job to run immediately
//	@Description	Triggers a summarize job to run immediately
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Router			/schedule/api/v0/summarize/trigger [get]
func (h HttpServer) TriggerSummarizeJob(ctx echo.Context) error {
	err := h.Scheduler.scheduleMustSummarizerJob(nil)
	if err != nil {
		errMsg := fmt.Sprintf("error scheduling summarize job: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: errMsg})
	}
	return ctx.JSON(http.StatusOK, "")
}

// TriggerInsightJob godoc
//
//	@Summary		Triggers an insight job to run immediately
//	@Description	Triggers an insight job to run immediately
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Router			/schedule/api/v0/insight/trigger [get]
func (h HttpServer) TriggerInsightJob(ctx echo.Context) error {
	insightJob, err := h.DB.FetchLastInsightJob()
	if err != nil {
		return err
	}
	if insightJob.Status == insightapi.InsightJobInProgress {
		return ctx.JSON(http.StatusConflict, api.ErrorResponse{Message: "insight job in progress"})
	}
	h.Scheduler.scheduleInsightJob(true)
	return ctx.JSON(http.StatusOK, "")
}

// TriggerComplianceJob godoc
//
//	@Summary		Triggers a compliance job to run immediately
//	@Description	Triggers a compliance job to run immediately
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Router			/schedule/api/v0/compliance/trigger [get]
func (h HttpServer) TriggerComplianceJob(ctx echo.Context) error {
	scheduleJob, err := h.DB.FetchLastCompletedScheduleJob()
	if err != nil {
		return err
	}

	_, err = h.Scheduler.RunComplianceReport(scheduleJob)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// TriggerComplianceSummarizerJob godoc
//
//	@Summary	Triggers a compliance summarizer job to run immediately
//	@Security	BearerToken
//	@Tags		describe
//	@Produce	json
//	@Success	200
//	@Router		/schedule/api/v0/compliance/summarizer/trigger [get]
func (h HttpServer) TriggerComplianceSummarizerJob(ctx echo.Context) error {
	scheduleJob, err := h.DB.FetchLastScheduleJob()
	if err != nil {
		return err
	}

	err = h.Scheduler.scheduleComplianceSummarizerJob(scheduleJob.ID)
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusOK)
}

// TriggerBenchmarkEvaluation godoc
//
//	@Summary		Trigger benchmark evaluation
//	@Description	Triggers a benchmark evaluation job to run immediately
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Param			request	body		api.TriggerBenchmarkEvaluationRequest	true	"Request Body"
//	@Success		200		{object}	[]ComplianceReportJob
//	@Router			/schedule/api/v1/benchmark/evaluation/trigger [put]
func (h HttpServer) TriggerBenchmarkEvaluation(ctx echo.Context) error {
	var req api.TriggerBenchmarkEvaluationRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var connectionIDs []string
	if req.ConnectionID != nil {
		connectionIDs = append(connectionIDs, *req.ConnectionID)
	}
	if len(req.ResourceIDs) > 0 {
		//TODO
		// figure out connection ids and add them
	}
	//TODO
	// which schedule job best fits for this ?

	job := ScheduleJob{
		Model:          gorm.Model{},
		Status:         summarizerapi.SummarizerJobInProgress,
		FailureMessage: "",
	}
	err := h.DB.AddScheduleJob(&job)
	if err != nil {
		errMsg := fmt.Sprintf("error adding schedule job: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: errMsg})
	}

	scheduleJob, err := h.DB.FetchLastScheduleJob()
	if err != nil {
		return err
	}
	var complianceJobs []ComplianceReportJob
	for _, connectionID := range connectionIDs {
		src, err := h.DB.GetSourceByID(connectionID)
		if err != nil {
			return err
		}

		crj := newComplianceReportJob(connectionID, source.Type(src.Type), req.BenchmarkID, scheduleJob.ID)

		err = h.DB.CreateComplianceReportJob(&crj)
		if err != nil {
			return err
		}

		if src == nil {
			return errors.New("failed to find connection")
		}

		enqueueComplianceReportJobs(h.Scheduler.logger, h.DB, h.Scheduler.complianceReportJobQueue, *src, &crj, scheduleJob)

		err = h.DB.UpdateSourceReportGenerated(connectionID, h.Scheduler.complianceIntervalHours)
		if err != nil {
			return err
		}
		complianceJobs = append(complianceJobs, crj)
	}

	return ctx.JSON(http.StatusOK, complianceJobs)
}

// HandleListBenchmarkEvaluations godoc
//
//	@Summary		Lists benchmark evaluations
//
//	@Description	This API allows users to retrieve details of all benchmark evaluation jobs based on specified filters. The API returns information such as evaluation time, benchmark ID, connection ID, connector name, status, and failure message.
//
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Param			request	body		api.ListBenchmarkEvaluationsRequest	true	"Request Body"
//	@Success		200		{object}	[]ComplianceReportJob
//	@Router			/schedule/api/v1/benchmark/evaluations [get]
func (h HttpServer) HandleListBenchmarkEvaluations(ctx echo.Context) error {
	var req api.ListBenchmarkEvaluationsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	cp, err := h.DB.ListComplianceReportsWithFilter(req.EvaluatedAtAfter, req.EvaluatedAtBefore, req.ConnectionID, req.Connector, req.BenchmarkID)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, cp)
}

// DescribeSingleResource godoc
//
//	@Summary	Describe single resource
//	@Security	BearerToken
//	@Tags		describe
//	@Produce	json
//	@Success	200
//	@Param		request	body		api.DescribeSingleResourceRequest	true	"Request Body"
//	@Success	200		{object}	aws.Resources
//	@Router		/schedule/api/v1/describe/resource [post]
func (h HttpServer) DescribeSingleResource(ctx echo.Context) error {
	var req api.DescribeSingleResourceRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	switch req.Provider {
	case source.CloudAWS:
		resources, err := aws.GetSingleResource(
			context.Background(),
			req.ResourceType,
			describe2.DescribeTriggerType(enums.DescribeTriggerTypeManual),
			req.AccountID,
			nil,
			req.AccessKey,
			req.SecretKey,
			"",
			"",
			false,
			req.AdditionalFields,
		)
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, *resources)

	}
	return echo.NewHTTPError(http.StatusNotImplemented, "provider not implemented")
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
//	@Param			terrafromFile	formData	file		false	"File to upload"
//	@Param			tag				formData	string		false	"Tags Map[string][]string"
//	@Param			resources		formData	[]string	false	"Additional Resources"
//	@Param			config			formData	string		false	"Config json structure"
//	@Success		200				{object}	api.Stack
//	@Router			/schedule/api/v1/stacks/create [post]
func (h HttpServer) CreateStack(ctx echo.Context) error {
	var tags map[string][]string
	tagsData := ctx.FormValue("tag")
	if tagsData != "" {
		json.Unmarshal([]byte(tagsData), &tags)
	}

	var resources []string
	// resourcesData := ctx.FormValue("resources")
	// if resourcesData != "" {
	// 	json.Unmarshal([]byte(resourcesData), &resources)
	// }

	file, err := ctx.FormFile("terrafromFile")
	if err != nil {
		if err.Error() != "http: no such file" {
			return err
		}
	}
	if file == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "No resource provided")
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	data, err := ioutil.ReadAll(src)
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
	resources = append(resources, arns...)

	var recordTags []*StackTag
	if len(tags) != 0 {
		for key, value := range tags {
			recordTags = append(recordTags, &StackTag{
				Key:   key,
				Value: pq.StringArray(value),
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

	terraformResourceTypes, err := internal.GetTypes(string(data))
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

	configStr := ctx.FormValue("config")
	if configStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide the credentials")
	}

	accs, err := internal.ParseAccountsFromArns(resources)
	if err != nil {
		return err
	}
	id := uuid.New()

	stackRecord := Stack{
		StackID:       "stack-" + id.String(),
		Resources:     pq.StringArray(resources),
		Tags:          recordTags,
		AccountIDs:    accs,
		ResourceTypes: pq.StringArray(resourceTypes),
		Status:        api.StackStatusCreated,
	}
	err = h.DB.AddStack(&stackRecord)
	if err != nil {
		return err
	}

	err = h.Scheduler.storeStackCredentials(stackRecord.ToApi(), configStr) // should be removed after describing
	if err != nil {
		return err
	}
	stackSource := Source{
		ID:        id,
		AccountID: accs[0],
		Type:      provider,
		ConfigRef: "",
	}
	if err != nil {
		return err
	}
	err = h.DB.CreateSource(&stackSource)

	return ctx.JSON(http.StatusOK, stackRecord.ToApi())
}

// GetStack godoc
//
//	@Summary		Get a Stack
//	@Description	Get a stack details by ID
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
			AccountIDs:    sr.AccountIDs,
		}
		stacks = append(stacks, stack)
	}
	return ctx.JSON(http.StatusOK, stacks)
}

// DeleteStack godoc
//
//	@Summary		Delete a Stack
//	@Description	Delete a stack by ID
//	@Security		BearerToken
//	@Tags			stack
//	@Cli			delete-stack
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

// TriggerStackBenchmark godoc
//
//	@Summary		Trigger Stack benchmark
//	@Description	Trigger defined benchmarks for a stack and save in the history
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			request	body		api.StackBenchmarkRequest	true	"Request Body"
//	@Success		200		{object}	[]ComplianceReportJob
//	@Router			/schedule/api/v1/stacks/benchmark/trigger [post]
func (h HttpServer) TriggerStackBenchmark(ctx echo.Context) error { // Retired
	var req api.StackBenchmarkRequest
	err := bindValidate(ctx, &req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	stackRecord, err := h.DB.GetStack(req.StackID)
	if err != nil {
		return err
	}
	if stackRecord.StackID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "stack not found")
	}

	job := ScheduleJob{
		Model:          gorm.Model{},
		Status:         summarizerapi.SummarizerJobInProgress,
		FailureMessage: "",
	}
	err = h.DB.AddScheduleJob(&job)
	if err != nil {
		errMsg := fmt.Sprintf("error adding schedule job: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: errMsg})
	}
	connectionId := stackRecord.StackID[6:]
	scheduleJob, err := h.DB.FetchLastScheduleJob()
	if err != nil {
		return err
	}
	var complianceJobs []ComplianceReportJob
	for _, benchmarkID := range req.Benchmarks {
		src, err := h.DB.GetSourceByID(connectionId)
		if err != nil {
			return err
		}

		crj := newComplianceReportJob(connectionId, source.Type(src.Type), benchmarkID, scheduleJob.ID)

		err = h.DB.CreateComplianceReportJob(&crj)
		if err != nil {
			return err
		}

		if src == nil {
			return errors.New("failed to find connection")
		}

		enqueueComplianceReportJobs(h.Scheduler.logger, h.DB, h.Scheduler.complianceReportJobQueue, *src, &crj, scheduleJob)

		err = h.DB.UpdateSourceReportGenerated(connectionId, h.Scheduler.complianceIntervalHours)
		if err != nil {
			return err
		}
		evaluation := StackEvaluation{
			EvaluatorID: benchmarkID,
			Type:        api.EvaluationTypeBenchmark,
			StackID:     stackRecord.StackID,
			JobID:       crj.ID,
		}
		err = h.DB.AddEvaluation(&evaluation)
		if err != nil {
			return err
		}
		complianceJobs = append(complianceJobs, crj)
	}

	return ctx.JSON(http.StatusOK, complianceJobs)
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
	connectionId := stackRecord.StackID[6:]

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
//	@Param			insightId	query		string	true	"InsightID"
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
	connectionId := stackRecord.StackID[6:]

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
			StackID:    sr.StackID,
			CreatedAt:  sr.CreatedAt,
			UpdatedAt:  sr.UpdatedAt,
			Resources:  []string(sr.Resources),
			Tags:       trimPrivateTags(sr.GetTagsMap()),
			AccountIDs: sr.AccountIDs,
		}
		stacks = append(stacks, stack)
	}
	return ctx.JSON(http.StatusOK, stacks)
}

// TriggerInsightEvaluation godoc
//
//	@Summary		Trigger insight evaluation
//	@Description	Trigger an insight evaluation to run immediately with given details
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Param			request	body		api.TriggerInsightEvaluationRequest	true	"Request Body"
//	@Success		200		{object}	[]InsightJob
//	@Router			/schedule/api/v1/insight/evaluation/trigger [put]
func (h HttpServer) TriggerInsightEvaluation(ctx echo.Context) error {
	var req api.TriggerInsightEvaluationRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var connectionIDs []string
	if req.ConnectionID != nil {
		connectionIDs = append(connectionIDs, *req.ConnectionID)
	}
	if len(req.ResourceIDs) > 0 {
		//TODO
		// figure out connection ids and add them
	}
	//TODO
	// which schedule job best fits for this ?

	insight, err := h.Scheduler.complianceClient.GetInsightMetadataById(httpclient.FromEchoContext(ctx), req.InsightID)
	if err != nil {
		return err
	}

	var insightJobs []InsightJob
	for _, srcID := range connectionIDs {
		src, err := h.DB.GetSourceByID(srcID)
		if err != nil {
			return err
		}
		job := newInsightJob(*insight, string(src.Type), srcID, src.AccountID, "")
		err = h.Scheduler.db.AddInsightJob(&job)
		if err != nil {
			return err
		}

		err = enqueueInsightJobs(h.Scheduler.insightJobQueue, job, *insight)
		if err != nil {
			job.Status = insightapi.InsightJobFailed
			job.FailureMessage = "Failed to enqueue InsightJob"
			h.Scheduler.db.UpdateInsightJobStatus(job)
		}
		insightJobs = append(insightJobs, job)
	}
	return ctx.JSON(http.StatusOK, insightJobs)
}

// TriggerInsightEvaluation godoc
//
//	@Summary		Trigger stack insight evaluation
//	@Description	Trigger an insight evaluation to run immediately on a stack with given details
//	@Security		BearerToken
//	@Tags			describe
//	@Produce		json
//	@Param			request	body		api.StackInsightRequest	true	"Request Body"
//	@Success		200		{object}	[]api.InsightJob
//	@Router			/schedule/api/v1/stacks/insight/trigger [post]
func (h HttpServer) TriggerStackInsight(ctx echo.Context) error { // Retired
	var req api.StackInsightRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	stackRecord, err := h.DB.GetStack(req.StackID)
	if err != nil {
		return err
	}
	if stackRecord.StackID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "stack not found")
	}
	stackId := stackRecord.StackID[6:]

	src, err := h.DB.GetSourceByID(stackId)
	if err != nil {
		return err
	}

	var insightJobs []api.InsightJob

	for _, insightId := range req.Insights {
		insight, err := h.Scheduler.complianceClient.GetInsightMetadataById(httpclient.FromEchoContext(ctx), insightId)
		if err != nil {
			return err
		}

		job := newInsightJob(*insight, string(src.Type), src.ID.String(), src.AccountID, "")
		err = h.Scheduler.db.AddInsightJob(&job)
		if err != nil {
			return err
		}

		err = enqueueInsightJobs(h.Scheduler.insightJobQueue, job, *insight)
		if err != nil {
			job.Status = insightapi.InsightJobFailed
			job.FailureMessage = "Failed to enqueue InsightJob"
			h.Scheduler.db.UpdateInsightJobStatus(job)
		}
		evaluation := StackEvaluation{
			EvaluatorID: strconv.FormatUint(uint64(insightId), 10),
			Type:        api.EvaluationTypeInsight,
			StackID:     stackRecord.StackID,
			JobID:       job.ID,
		}
		err = h.DB.AddEvaluation(&evaluation)
		if err != nil {
			return err
		}
		insightJobs = append(insightJobs, api.InsightJob{
			ID:             job.ID,
			InsightID:      job.InsightID,
			SourceID:       job.SourceID,
			AccountID:      job.AccountID,
			SourceType:     job.SourceType,
			Status:         job.Status,
			FailureMessage: job.FailureMessage,
			CreatedAt:      job.CreatedAt,
			UpdatedAt:      job.UpdatedAt,
		})
	}
	return ctx.JSON(http.StatusOK, insightJobs)
}

// GetInsightJob godoc
//
//	@Summary		Get an Insight Job
//	@Description	Get an Insight Job details by ID
//	@Security		BearerToken
//	@Tags			describe
//	@Accept			json
//	@Produce		json
//	@Param			jobId	path		string	true	"Job Id"
//	@Success		200		{object}	api.InsightJob
//	@Router			/schedule/api/v1/insight/job/{jobId} [get]
func (h HttpServer) GetInsightJob(ctx echo.Context) error {
	jobIdstring := ctx.Param("jobId")
	jobId, err := strconv.ParseUint(jobIdstring, 10, 64)
	if err != nil {
		return err
	}
	job, err := h.DB.GetInsightJobById(uint(jobId))
	result := api.InsightJob{
		ID:             job.ID,
		InsightID:      job.InsightID,
		SourceID:       job.SourceID,
		AccountID:      job.AccountID,
		SourceType:     job.SourceType,
		Status:         job.Status,
		FailureMessage: job.FailureMessage,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
	}
	return ctx.JSON(http.StatusOK, result)
}

// TriggerStackDescriber godoc
//
//	@Summary		Trigger Stack Describer
//	@Description	Describe stack resources. This is needed before triggering insights and benchmarks
//	@Description	Config structure for azure: {tenantId: string, objectId: string, secretId: string, clientId: string, clientSecret:string}
//	@Description	Config structure for aws: {accessKey: string, secretKey: string}
//	@Security		BearerToken
//	@Tags			stack
//	@Produce		json
//	@Success		200
//	@Param			req	body	api.DescribeStackRequest	true	"request"
//	@Router			/schedule/api/v1/stacks/describer/trigger [post]
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
