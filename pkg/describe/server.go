package describe

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
)

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

	v1 := e.Group("/api/v1")

	v1.GET("/sources", s.HandleListSources)
	v1.GET("/sources/:source_id/jobs/describe", s.HandleListSourceDescribeJobs)
	v1.GET("/sources/:source_id/jobs/compliance", s.HandleListSourceComplianceReports)

	v1.POST("/sources/:source_id/jobs/describe/refresh", s.RunDescribeJobs)
	v1.POST("/sources/:source_id/jobs/compliance/refresh", s.RunComplianceReportJobs)

	v1.PUT("/sources/:source_id/policy/:policy_id", s.AssignPolicyToSource)

	v1.GET("/resource_type/:provider", s.GetResourceTypesByProvider)

	return e.Start(s.Address)
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

		objs = append(objs, api.Source{
			ID:                     source.ID,
			Type:                   source.Type,
			LastDescribedAt:        lastDescribeAt,
			LastComplianceReportAt: lastComplianceReportAt,
		})
	}

	return ctx.JSON(http.StatusOK, objs)
}

// HandleListSourceDescribeJobs godoc
// @Summary      List source describe jobs
// @Description  List source describe jobs
// @Tags         schedule
// @Produce      json
// @Param        source_id   path      string  true  "SourceID"
// @Success      200  {object}  []api.DescribeSource
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
// @Param        source_id   path      string  true  "SourceID"
// @Success      200  {object}  []api.ComplianceReport
// @Router       /schedule/api/v1/sources/{source_id}/jobs/compliance [get]
func (s HttpServer) HandleListSourceComplianceReports(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid source uuid"})
	}

	jobs, err := s.DB.ListComplianceReports(sourceUUID)
	if err != nil {
		ctx.Logger().Errorf("fetching compliance reports: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "internal error"})
	}

	var objs []api.ComplianceReport
	for _, job := range jobs {
		objs = append(objs, api.ComplianceReport{
			Status:         job.Status,
			S3ResultURL:    job.S3ResultURL,
			FailureMessage: job.FailureMessage,
		})
	}

	return ctx.JSON(http.StatusOK, objs)
}

// RunComplianceReportJobs godoc
// @Summary      Run compliance report jobs
// @Description  Run compliance report jobs
// @Tags         schedule
// @Produce      json
// @Param        source_id   path      string  true  "SourceID"
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

// RunDescribeJobs godoc
// @Summary      Run describe jobs
// @Description  Run describe jobs
// @Tags         schedule
// @Produce      json
// @Param        source_id   path      string  true  "SourceID"
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

// AssignPolicyToSource godoc
// @Summary      Assign source to policy
// @Description  Assign source to policy
// @Tags         schedule
// @Produce      json
// @Param        source_id   path      string  true  "SourceID"
// @Router       /schedule/api/v1/sources/{source_id}/policy/{policy_id} [get]
func (s HttpServer) AssignPolicyToSource(ctx echo.Context) error {
	sourceID := ctx.Param("source_id")
	sourceUUID, err := uuid.Parse(sourceID)
	if err != nil {
		ctx.Logger().Errorf("parsing source uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid source uuid"})
	}

	policyID := ctx.Param("policy_id")
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		ctx.Logger().Errorf("parsing policy uuid: %v", err)
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "invalid policy uuid"})
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
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "failed to assign policy"})
	}

	return ctx.String(http.StatusOK, "")
}

// GetResourceTypesByProvider godoc
// @Summary      get resource type by provider
// @Description  get resource type by provider
// @Tags         schedule
// @Produce      json
// @Param        provider   path      string  true  "Provider" Enums(aws,azure)
// @Success      200  {object}  []string
// @Router       /schedule/api/v1/resource_type/{provider} [get]
func (s HttpServer) GetResourceTypesByProvider(ctx echo.Context) error {
	provider := ctx.Param("provider")

	var resourceTypes []string

	if provider == "azure" || provider == "all" {
		resourceTypes = append(resourceTypes, azure.ListResourceTypes()...)
	}
	if provider == "aws" || provider == "all" {
		resourceTypes = append(resourceTypes, aws.ListResourceTypes()...)
	}

	return ctx.JSON(http.StatusOK, resourceTypes)
}
