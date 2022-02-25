package describe

import (
	"database/sql"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	cr "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	"net/http"
	"time"
)

type ApiSource struct {
	ID                     uuid.UUID    `json:"id"`
	Type                   SourceType   `json:"type"`
	LastDescribedAt        sql.NullTime `json:"last_described_at"`
	LastComplianceReportAt sql.NullTime `json:"last_compliance_report_at"`
}

type ApiDescribeSource struct {
	DescribeResourceJobs []ApiDescribeResource   `json:"describe_resource_jobs"`
	Status               DescribeSourceJobStatus `json:"status"`
}

type ApiDescribeResource struct {
	ResourceType   string                    `json:"resource_type"`
	Status         DescribeResourceJobStatus `json:"status"`
	FailureMessage string                    `json:"failure_message"`
}

type ApiComplianceReport struct {
	Status         cr.ComplianceReportJobStatus `json:"status"`
	S3ResultURL    string                       `json:"s3_result_url"`
	FailureMessage string                       `json:"failure_message"`
}

type ErrorResponse struct {
	Message string
}

type HttpServer struct {
	Address string
	DB      Database
}

func NewHTTPServer(address string, db Database) *HttpServer {
	return &HttpServer{
		Address: address,
		DB:      db,
	}
}

func (s *HttpServer) Initialize() error {
	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))
	e.GET("/sources", s.HandleListSources)
	e.GET("/sources/:source_id/jobs/describe", s.HandleListSourceDescribeJobs)
	e.GET("/sources/:source_id/jobs/compliance", s.HandleListSourceComplianceReports)
	e.GET("/sources/:source_id/jobs/describe/refresh", s.RunDescribeJobs)
	e.GET("/sources/:source_id/jobs/compliance/refresh", s.RunComplianceReportJobs)

	e.GET("/sources/:source_id/policy/:policy_id", s.AssignSourceToPolicy)
	return e.Start(s.Address)
}

func (s *HttpServer) HandleListSources(ctx echo.Context) error {
	sources, err := s.DB.ListSources()
	if err != nil {
		ctx.Logger().Errorf("fetching sources: %v", err)
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal error"})
	}

	var objs []ApiSource
	for _, source := range sources {
		objs = append(objs, ApiSource{
			ID:                     source.ID,
			Type:                   source.Type,
			LastDescribedAt:        source.LastDescribedAt,
			LastComplianceReportAt: source.LastComplianceReportAt,
		})
	}

	return ctx.JSON(http.StatusOK, objs)
}

func (s *HttpServer) HandleListSourceDescribeJobs(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid source uuid"})
	}

	jobs, err := s.DB.ListDescribeSourceJobs(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("fetching describe source jobs: %v", err)
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal error"})
	}

	var objs []ApiDescribeSource
	for _, job := range jobs {
		var describeResourceJobs []ApiDescribeResource
		for _, describeResourceJob := range job.DescribeResourceJobs {
			describeResourceJobs = append(describeResourceJobs, ApiDescribeResource{
				ResourceType:   describeResourceJob.ResourceType,
				Status:         describeResourceJob.Status,
				FailureMessage: describeResourceJob.FailureMessage,
			})
		}

		objs = append(objs, ApiDescribeSource{
			DescribeResourceJobs: describeResourceJobs,
			Status:               job.Status,
		})
	}

	return ctx.JSON(http.StatusOK, objs)
}

func (s *HttpServer) HandleListSourceComplianceReports(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid source uuid"})
	}

	jobs, err := s.DB.ListComplianceReports(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("fetching compliance reports: %v", err)
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal error"})
	}

	var objs []ApiComplianceReport
	for _, job := range jobs {
		objs = append(objs, ApiComplianceReport{
			Status:         job.Status,
			S3ResultURL:    job.S3ResultURL,
			FailureMessage: job.FailureMessage,
		})
	}

	return ctx.JSON(http.StatusOK, objs)
}

func (s *HttpServer) RunComplianceReportJobs(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid source uuid"})
	}

	err = s.DB.UpdateSourceNextComplianceReportToNow(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("update source next compliance report run: %v", err)
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal error"})
	}

	return ctx.String(http.StatusOK, "")
}

func (s *HttpServer) RunDescribeJobs(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid source uuid"})
	}

	err = s.DB.UpdateSourceNextDescribeAtToNow(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("update source next describe run: %v", err)
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal error"})
	}

	return ctx.String(http.StatusOK, "")
}

func (s *HttpServer) AssignSourceToPolicy(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing source uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid source uuid"})
	}

	policyID := ctx.Param("policy_id")
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		ctx.Logger().Errorf("parsing policy uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid policy uuid"})
	}

	//TODO-Saleh check whether assigned policy exists in policy engine

	err = s.DB.CreateAssignment(&Assignment{
		SourceID:  sourceUUID,
		PolicyID:  policyUUID,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		ctx.Logger().Errorf("assigning policy to source: %v", err)
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: "failed to assign policy"})
	}

	return ctx.String(http.StatusOK, "")
}
