package describe

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	describe2 "github.com/kaytu-io/kaytu-util/pkg/describe/enums"
	"github.com/lib/pq"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	api3 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	complianceapi "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
	insightapi "gitlab.com/keibiengine/keibi-engine/pkg/insight/api"
	inventoryapi "gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	summarizerapi "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/api"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/internal"
	inventory "gitlab.com/keibiengine/keibi-engine/pkg/inventory/client"
)

type HttpServer struct {
	Address   string
	DB        Database
	Scheduler *Scheduler
}

func NewHTTPServer(
	address string,
	db Database,
	s *Scheduler,
) *HttpServer {

	return &HttpServer{
		Address:   address,
		DB:        db,
		Scheduler: s,
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
	v1.PUT("/describe/trigger/:connection_id", httpserver.AuthorizeHandler(h.TriggerDescribeJobV1, api3.AdminRole))

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

	v1.POST("/stacks/benchmark/trigger", httpserver.AuthorizeHandler(h.TriggerStackBenchmark, api3.AdminRole))
	v1.GET("/stacks", httpserver.AuthorizeHandler(h.ListStack, api3.ViewerRole))
	v1.GET("/stacks/:stackId", httpserver.AuthorizeHandler(h.GetStack, api3.ViewerRole))
	v1.POST("/stacks/build", httpserver.AuthorizeHandler(h.CreateStack, api3.AdminRole))
	v1.DELETE("/stacks/:stackId", httpserver.AuthorizeHandler(h.DeleteStack, api3.AdminRole))
	v1.GET("/stacks/benchmark/:jobId", httpserver.AuthorizeHandler(h.GetBenchmarkResult, api3.ViewerRole))
	v1.GET("/stacks/:stackId/insights", httpserver.AuthorizeHandler(h.GetInsights, api3.ViewerRole))
}

// HandleListSources godoc
//
//	@Summary		List Sources
//	@Description	Getting all of Keibi sources
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
//	@Summary		Get Source by id
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
		Type:                   source.Type,
		LastDescribedAt:        lastDescribeAt,
		LastComplianceReportAt: lastComplianceReportAt,
		LastDescribeJobStatus:  lastJobStatus,
	})
}

// HandleListPendingDescribeSourceJobs godoc
//
//	@Summary	Listing describe source jobs
//	@Tags		schedule
//	@Produce	json
//	@Success	200	{object}	api.Source
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
//	@Tags		schedule
//	@Produce	json
//	@Success	200	{object}	api.Source
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
//	@Tags		schedule
//	@Produce	json
//	@Success	200	{object}	api.Source
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
//	@Tags		schedule
//	@Produce	json
//	@Success	200	{object}	api.Source
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
//	@Description	List source describe jobs
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
//	@Description	List source compliance reports
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
//	@Tags			schedule
//	@Produce		json
//	@Param			source_id	path	string	true	"SourceID"
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
//	@Tags			schedule
//	@Produce		json
//	@Param			source_id	path	string	true	"SourceID"
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
//	@Tags			describe
//	@Produce		json
//	@Success		200
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
//	@Summary	Triggers a benchmark evaluation job to run immediately
//	@Tags		describe
//	@Produce	json
//	@Param		request	body		api.TriggerBenchmarkEvaluationRequest	true	"Request Body"
//	@Success	200		{object}	[]describe.ComplianceReportJob
//	@Router		/schedule/api/v1/compliance/trigger [put]
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
//	@Summary	Lists all benchmark evaluations
//	@Tags		compliance
//	@Produce	json
//	@Success	200
//	@Param		request	body		api.ListBenchmarkEvaluationsRequest	true	"Request Body"
//	@Success	200		{object}	[]describe.ComplianceReportJob
//	@Router		/schedule/api/v1/benchmark/evaluations [get]
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

// BuildStackFromStatefile godoc
//
//	@Summary		Build a stack
//	@Description	Temporary API for building a stack by giving a terraform statefile directory
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			request	body		api.CreateStackRequest	true	"Request Body"
//	@Success		200		{object}	api.Stack
//	@Router			/schedule/api/v1/stack/create [put]
func (h HttpServer) CreateStack(ctx echo.Context) error {
	var req api.CreateStackRequest
	bindValidate(ctx, &req)
	resources := req.Resources
	if req.Statefile != "" {
		data, err := ioutil.ReadFile(req.Statefile)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		arns, err := internal.GetArns(string(data))
		if err != nil {
			return err
		}
		resources = append(resources, arns...)
	}
	if len(resources) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No resource provided")
	}
	var recordTags []*StackTag
	for key, value := range req.Tags {
		recordTags = append(recordTags, &StackTag{
			Key:   key,
			Value: pq.StringArray(value),
		})
	}
	id := "stack-" + uuid.New().String()
	stackRecord := Stack{
		StackID:   id,
		Resources: pq.StringArray(resources),
		Tags:      recordTags,
	}
	err := h.DB.AddStack(&stackRecord)
	if err != nil {
		return err
	}

	stack := api.Stack{
		StackID:   stackRecord.StackID,
		CreatedAt: stackRecord.CreatedAt,
		UpdatedAt: stackRecord.UpdatedAt,
		Resources: []string(stackRecord.Resources),
		Tags:      trimPrivateTags(stackRecord.GetTagsMap()),
	}
	return ctx.JSON(http.StatusOK, stack)
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

	var evaluations []api.StackEvaluation
	for _, e := range stackRecord.Evaluations {
		evaluations = append(evaluations, api.StackEvaluation{
			BenchmarkID: e.BenchmarkID,
			JobID:       e.JobID,
			CreatedAt:   e.CreatedAt,
		})
	}

	stack := api.Stack{
		StackID:     stackRecord.StackID,
		CreatedAt:   stackRecord.CreatedAt,
		UpdatedAt:   stackRecord.UpdatedAt,
		Resources:   []string(stackRecord.Resources),
		Tags:        trimPrivateTags(stackRecord.GetTagsMap()),
		Evaluations: evaluations,
	}
	return ctx.JSON(http.StatusOK, stack)
}

// ListStack godoc
//
//	@Summary		List Stack
//	@Description	Get list of stacks
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			tag				query		string		false	"Key-Value tags in key=value format to filter by"
//	@Success		200	{object}	[]api.Stack
//	@Router			/schedule/api/v1/stacks [get]
func (h HttpServer) ListStack(ctx echo.Context) error {
	tagMap := internal.TagStringsToTagMap(ctx.QueryParams()["tag"])
	stacksRecord, err := h.DB.ListStacks(tagMap)
	if err != nil {
		return err
	}
	var stacks []api.Stack
	for _, sr := range stacksRecord {

		stack := api.Stack{
			StackID:   sr.StackID,
			CreatedAt: sr.CreatedAt,
			UpdatedAt: sr.UpdatedAt,
			Resources: []string(sr.Resources),
			Tags:      trimPrivateTags(sr.GetTagsMap()),
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

// TriggerBenchmark godoc
//
//	@Summary		Trigger benchmarks
//	@Description	Trigger defined benchmarks for a stack
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			request	body		api.EvaluateStack	true	"Request Body"
//	@Success		200		{object}	[]ComplianceReportJob
//	@Router			/schedule/api/v1/stacks/benchmark/trigger [post]
func (h HttpServer) TriggerStackBenchmark(ctx echo.Context) error {
	var req api.EvaluateStack
	bindValidate(ctx, &req)

	stackRecord, err := h.DB.GetStack(req.StackID)
	if err != nil {
		return err
	}
	if stackRecord.StackID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "stack not found")
	}

	stack := api.Stack{
		StackID:   stackRecord.StackID,
		CreatedAt: stackRecord.CreatedAt,
		UpdatedAt: stackRecord.UpdatedAt,
		Resources: []string(stackRecord.Resources),
		Tags:      trimPrivateTags(stackRecord.GetTagsMap()),
	}
	accs, err := internal.ParseAccountsFromArns(stack.Resources)
	if err != nil {
		return err
	}
	var connectionIDs []string
	for _, acc := range accs {
		source, err := h.Scheduler.onboardClient.GetSourcesByAccount(httpclient.FromEchoContext(ctx), acc)
		if err != nil {
			return err
		}
		connectionIDs = append(connectionIDs, source.ID.String())
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

	scheduleJob, err := h.DB.FetchLastScheduleJob()
	if err != nil {
		return err
	}
	var complianceJobs []ComplianceReportJob
	for _, benchmarkID := range req.Benchmarks {
		for _, connectionID := range connectionIDs {
			src, err := h.DB.GetSourceByID(connectionID)
			if err != nil {
				return err
			}

			crj := newComplianceReportJob(connectionID, source.Type(src.Type), benchmarkID, scheduleJob.ID)

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
			evaluation := StackEvaluation{
				BenchmarkID: benchmarkID,
				StackID:     stack.StackID,
				JobID:       job.ID,
			}
			err = h.DB.AddEvaluation(&evaluation)
			if err != nil {
				return err
			}
			complianceJobs = append(complianceJobs, crj)
		}
	}
	return ctx.JSON(http.StatusOK, complianceJobs)
}

// GetBenchmarkResult godoc
//
//	@Summary		Get Benchmark Result
//	@Description	Get a benchmark result by jobId
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			jobId	path		string	true	"JobID"
//	@Success		200		{object}	es.Finding
//	@Router			/schedule/api/v1/stacks/benchmark/{jobId} [get]
func (h HttpServer) GetBenchmarkResult(ctx echo.Context) error {
	jobIdstring := ctx.Param("jobId")
	jobId, err := strconv.ParseUint(jobIdstring, 10, 32)
	if err != nil {
		return err
	}
	evaluation, err := h.DB.GetEvaluation(uint(jobId))
	if err != nil {
		return err
	}
	stackRecord, err := h.DB.GetStack(evaluation.StackID)
	if err != nil {
		return err
	}
	if stackRecord.StackID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "stack not found")
	}
	accs, err := internal.ParseAccountsFromArns([]string(stackRecord.Resources))
	if err != nil {
		return err
	}
	var conns []string
	for _, acc := range accs {
		source, err := h.Scheduler.onboardClient.GetSourcesByAccount(httpclient.FromEchoContext(ctx), acc)
		if err != nil {
			return err
		}
		conns = append(conns, source.ID.String())
	}
	if err != nil {
		return err
	}
	findings, err := h.Scheduler.complianceClient.GetFindings(httpclient.FromEchoContext(ctx), conns, evaluation.BenchmarkID, []string(stackRecord.Resources))
	if err != nil {
		return err
	}
	var result es.Finding
	for _, f := range findings.Findings {
		if f.ComplianceJobID == evaluation.JobID {
			result = f
			break
		}
	}
	return ctx.JSON(http.StatusOK, result)
}

// GetInsights godoc
//
//	@Summary		Get Insights Result
//	@Description	Get a benchmark result by jobId
//	@Security		BearerToken
//	@Tags			stack
//	@Accept			json
//	@Produce		json
//	@Param			time	query		int		false	"unix seconds for the time to get the insight result for"
//	@Param			stackId	path		string	true	"StackID"
//	@Success		200		{object}	[]inventoryapi.InsightPeerGroup
//	@Router			/schedule/api/v1/stacks/{stackId}/insights [get]
func (h HttpServer) GetInsights(ctx echo.Context) error {
	stackId := ctx.Param("stackId")
	timeStr := ctx.QueryParam("time")
	stackRecord, err := h.DB.GetStack(stackId)
	if err != nil {
		return err
	}
	if stackRecord.StackID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "stack not found")
	}
	accs, err := internal.ParseAccountsFromArns([]string(stackRecord.Resources))
	if err != nil {
		return err
	}
	var conns []string
	for _, acc := range accs {
		source, err := h.Scheduler.onboardClient.GetSourcesByAccount(httpclient.FromEchoContext(ctx), acc)
		if err != nil {
			return err
		}
		conns = append(conns, source.ID.String())
	}
	if err != nil {
		return err
	}
	inventoryClient := inventory.NewInventoryServiceClient(h.Address + "/inventory")
	var result []inventoryapi.InsightPeerGroup
	result, err = inventoryClient.ListInsights(httpclient.FromEchoContext(ctx), conns, timeStr)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}
