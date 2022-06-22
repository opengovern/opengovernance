package describe

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	complianceapi "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report/api"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
)

type HttpServer struct {
	Address string
	DB      Database
}

func NewHTTPServer(
	address string,
	db Database,
) *HttpServer {

	return &HttpServer{
		Address: address,
		DB:      db,
	}
}

func (s *HttpServer) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.GET("/sources", s.HandleListSources)
	v1.GET("/sources/:source_id", s.HandleGetSource)
	v1.GET("/sources/:source_id/jobs/describe", s.HandleListSourceDescribeJobs)
	v1.GET("/sources/:source_id/jobs/compliance", s.HandleListSourceComplianceReports)

	v1.POST("/sources/:source_id/jobs/describe/refresh", s.RunDescribeJobs)
	v1.POST("/sources/:source_id/jobs/compliance/refresh", s.RunComplianceReportJobs)

	v1.GET("/resource_type/:provider", s.GetResourceTypesByProvider)

	v1.GET("/compliance/report/last/completed", s.HandleGetLastCompletedComplianceReport)

	v1.GET("/insight", s.ListInsights)
	v1.PUT("/insight", s.CreateInsight)
	v1.DELETE("/insight/:id", s.DeleteInsight)
}

// HandleListSources godoc
// @Summary      List Sources
// @Description  Getting all of Keibi sources
// @Tags         schedule
// @Produce      json
// @Success      200  {object}  []api.Source
// @Router       /schedule/api/v1/sources [get]
func (s HttpServer) HandleListSources(ctx echo.Context) error {
	sources, err := s.DB.ListSources()
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

		job, err := s.DB.GetLastDescribeSourceJob(source.ID)
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
// @Summary      Get Source by id
// @Description  Getting Keibi source by id
// @Tags         schedule
// @Produce      json
// @Param        source_id  path      string  true  "SourceID"
// @Success      200        {object}  api.Source
// @Router       /schedule/api/v1/sources/{source_id} [get]
func (s HttpServer) HandleGetSource(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid source uuid"})
	}
	source, err := s.DB.GetSourceByUUID(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("fetching source %s: %v", sourceID, err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "fetching source"})
	}

	job, err := s.DB.GetLastDescribeSourceJob(sourceUUID)
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

// HandleListSourceDescribeJobs godoc
// @Summary      List source describe jobs
// @Description  List source describe jobs
// @Tags         schedule
// @Produce      json
// @Param        source_id  path      string  true  "SourceID"
// @Success      200        {object}  []api.DescribeSource
// @Router       /schedule/api/v1/sources/{source_id}/jobs/describe [get]
func (s HttpServer) HandleListSourceDescribeJobs(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid source uuid"})
	}

	jobs, err := s.DB.ListDescribeSourceJobs(sourceUUID)
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
// @Summary      List source compliance reports
// @Description  List source compliance reports
// @Tags         schedule
// @Produce      json
// @Param        source_id  path      string  true   "SourceID"
// @Param        from       query     int     false  "From Time (TimeRange)"
// @Param        to         query     int     false  "To Time (TimeRange)"
// @Success      200        {object}  []complianceapi.ComplianceReport
// @Router       /schedule/api/v1/sources/{source_id}/jobs/compliance [get]
func (s HttpServer) HandleListSourceComplianceReports(ctx echo.Context) error {
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
		report, err := s.DB.GetLastCompletedSourceComplianceReport(sourceUUID)
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

		jobs, err = s.DB.ListCompletedComplianceReportByDate(sourceUUID, fromTime, toTime)
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
// @Summary      Run compliance report jobs
// @Description  Run compliance report jobs
// @Tags         schedule
// @Produce      json
// @Param        source_id  path  string  true  "SourceID"
// @Router       /schedule/api/v1/sources/{source_id}/jobs/compliance/refresh [post]
func (s HttpServer) RunComplianceReportJobs(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid source uuid"})
	}

	err = s.DB.UpdateSourceNextComplianceReportToNow(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("update source next compliance report run: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "internal error"})
	}

	return ctx.String(http.StatusOK, "")
}

// HandleGetLastCompletedComplianceReport godoc
// @Summary  Get last completed compliance report
// @Tags         schedule
// @Produce      json
// @Success  200  {object}  int
// @Router   /schedule/api/v1/compliance/report/last/completed [get]
func (s HttpServer) HandleGetLastCompletedComplianceReport(ctx echo.Context) error {
	id, err := s.DB.GetLastCompletedComplianceReportID()
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, id)
}

// RunDescribeJobs godoc
// @Summary      Run describe jobs
// @Description  Run describe jobs
// @Tags         schedule
// @Produce      json
// @Param        source_id  path  string  true  "SourceID"
// @Router       /schedule/api/v1/sources/{source_id}/jobs/describe/refresh [post]
func (s HttpServer) RunDescribeJobs(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid source uuid"})
	}

	err = s.DB.UpdateSourceNextDescribeAtToNow(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("update source next describe run: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "internal error"})
	}

	return ctx.String(http.StatusOK, "")
}

// GetResourceTypesByProvider godoc
// @Summary      get resource type by provider
// @Description  get resource type by provider
// @Tags         schedule
// @Produce      json
// @Param        provider  path      string  true  "Provider"  Enums(aws,azure)
// @Success      200       {object}  []api.ResourceTypeDetail
// @Router       /schedule/api/v1/resource_type/{provider} [get]
func (s HttpServer) GetResourceTypesByProvider(ctx echo.Context) error {
	provider := ctx.Param("provider")

	var resourceTypes []api.ResourceTypeDetail

	if provider == "azure" || provider == "all" {
		for _, resourceType := range azure.ListResourceTypes() {
			resourceTypes = append(resourceTypes, api.ResourceTypeDetail{
				ResourceTypeARN:  resourceType,
				ResourceTypeName: cloudservice.ServiceNameByResourceType(resourceType),
			})
		}
	}
	if provider == "aws" || provider == "all" {
		for _, resourceType := range aws.ListResourceTypes() {
			resourceTypes = append(resourceTypes, api.ResourceTypeDetail{
				ResourceTypeARN:  resourceType,
				ResourceTypeName: cloudservice.ServiceNameByResourceType(resourceType),
			})
		}
	}

	return ctx.JSON(http.StatusOK, resourceTypes)
}

// CreateInsight godoc
// @Summary  Create a new insight
// @Tags     insights
// @Produce  json
// @Param    request  body      api.CreateInsightRequest  true  "Request Body"
// @Success  200      {object}  uint
// @Router   /schedule/api/v1/insight [put]
func (h *HttpServer) CreateInsight(ctx echo.Context) error {
	var req api.CreateInsightRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var labels []InsightLabel
	for _, value := range req.Labels {
		labels = append(labels, InsightLabel{
			Value: value,
		})
	}
	ins := Insight{
		Description: req.Description,
		Query:       req.Query,
		Labels:      labels,
	}
	err := h.DB.AddInsight(&ins)
	if err != nil {
		return err
	}
	return ctx.JSON(200, ins.ID)
}

// DeleteInsight godoc
// @Summary  Delete an insight
// @Tags     insights
// @Produce  json
// @Param    request  body  uint  true  "Request Body"
// @Success  200
// @Router   /schedule/api/v1/insight/{id} [delete]
func (h *HttpServer) DeleteInsight(ctx echo.Context) error {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		fmt.Println(err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	err = h.DB.DeleteInsight(uint(id))
	if err != nil {
		return err
	}
	return ctx.NoContent(200)
}

// ListInsights godoc
// @Summary      List insights
// @Description  Listing insights
// @Tags         insights
// @Produce      json
// @Param        request  body      api.ListInsightsRequest  true  "Request Body"
// @Success      200      {object}  []api.Insight
// @Router       /schedule/api/v1/insight [get]
func (h *HttpServer) ListInsights(ctx echo.Context) error {
	var req api.ListInsightsRequest
	if err := bindValidate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var search *string
	if len(req.DescriptionFilter) > 0 {
		search = &req.DescriptionFilter
	}

	queries, err := h.DB.ListInsightsWithFilters(search, req.Labels)
	if err != nil {
		return err
	}

	var result []api.Insight
	for _, item := range queries {
		var labels []string
		for _, i := range item.Labels {
			labels = append(labels, i.Value)
		}
		result = append(result, api.Insight{
			ID:          item.Model.ID,
			Description: item.Description,
			Query:       item.Query,
			Labels:      labels,
		})
	}
	return ctx.JSON(200, result)
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
