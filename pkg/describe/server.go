package describe

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/ProtonMail/gopenpgp/v2/helper"
	api3 "gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	complianceapi "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
	insightapi "gitlab.com/keibiengine/keibi-engine/pkg/insight/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	summarizerapi "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/api"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
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

	v1.POST("/jobs/:job_id/creds", httpserver.AuthorizeHandler(h.HandleGetCredsForJob, api3.AdminRole))
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
		DescribeJobsCount.WithLabelValues("failure").Inc()
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: errMsg})
	}
	return ctx.JSON(http.StatusOK, "")
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
//	@Summary		Triggers a compliance summarizer job to run immediately
//	@Tags			describe
//	@Produce		json
//	@Success		200
//	@Router			/schedule/api/v0/compliance/summarizer/trigger [get]
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
//	@Success	200
//	@Param		request	body	api.TriggerBenchmarkEvaluationRequest	true	"Request Body"
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

	scheduleJob, err := h.DB.FetchLastScheduleJob()
	if err != nil {
		return err
	}

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
	}

	return ctx.NoContent(http.StatusOK)
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

// HandleGetCredsForJob godoc
//
//	@Summary	Get credentials for a cloud native job by providing job info
//	@Tags		jobs
//	@Produce	json
//	@Param		request	body		api.GetCredsForJobRequest	true	"Request Body"
//	@Success	200		{object}	api.GetCredsForJobResponse
//	@Router		/schedule/api/v1/jobs/{job_id}/creds [post]
func (h HttpServer) HandleGetCredsForJob(ctx echo.Context) error {
	var req api.GetCredsForJobRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	jobId := ctx.Param("job_id")

	job, err := h.DB.GetCloudNativeDescribeSourceJob(jobId)
	if err != nil {
		return err
	}
	if job == nil || job.SourceJob.SourceID.String() != req.SourceID {
		return echo.NewHTTPError(http.StatusNotFound, "job not found")
	}
	if job.SourceJob.Status != api.DescribeSourceJobInProgress {
		return echo.NewHTTPError(http.StatusBadRequest, "job not in progress")
	}
	describeIntervalHours, err := strconv.ParseInt(DescribeIntervalHours, 10, 64)
	if err != nil {
		describeIntervalHours = 6
	}
	if job.CreatedAt.Add(time.Hour * time.Duration(describeIntervalHours)).Before(time.Now()) {
		return echo.NewHTTPError(http.StatusBadRequest, "job expired")
	}

	// TODO: check if any other job is in progress for this source and return error if so

	src, err := h.DB.GetSourceByUUID(job.SourceJob.SourceID)
	if err != nil {
		return err
	}
	if src == nil {
		return echo.NewHTTPError(http.StatusNotFound, "source not found")
	}

	creds, err := h.Scheduler.vault.Read(src.ConfigRef)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to read creds")
	}
	jsonCreds, err := json.Marshal(creds)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal creds")
	}
	encryptedCreds, err := helper.EncryptMessageArmored(job.CredentialEncryptionPublicKey, string(jsonCreds))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to encrypt creds")
	}
	return ctx.JSON(http.StatusOK, api.GetCredsForJobResponse{
		Credentials: encryptedCreds,
	})
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
