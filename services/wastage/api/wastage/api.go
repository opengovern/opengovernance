package wastage

import (
	"encoding/json"
	types2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/cost"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/wastage/ingestion"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type API struct {
	tracer       trace.Tracer
	logger       *zap.Logger
	costSvc      *cost.Service
	usageRepo    repo.UsageV2Repo
	recomSvc     *recommendation.Service
	ingestionSvc *ingestion.Service
}

func New(costSvc *cost.Service, recomSvc *recommendation.Service, ingestionService *ingestion.Service, usageRepo repo.UsageV2Repo, logger *zap.Logger) API {
	return API{
		costSvc:      costSvc,
		recomSvc:     recomSvc,
		usageRepo:    usageRepo,
		ingestionSvc: ingestionService,
		tracer:       otel.GetTracerProvider().Tracer("wastage.http.sources"),
		logger:       logger.Named("wastage-api"),
	}
}

func (s API) Register(e *echo.Echo) {
	g := e.Group("/api/v1/wastage")
	g.POST("/ec2-instance", s.EC2Instance)
	g.POST("/aws-rds", s.AwsRDS)
	i := e.Group("/api/v1/wastage-ingestion")
	i.PUT("/ingest/:service", httpserver.AuthorizeHandler(s.TriggerIngest, api.InternalRole))
}

// EC2Instance godoc
//
//	@Summary		List wastage in EC2 Instances
//	@Description	List wastage in EC2 Instances
//	@Security		BearerToken
//	@Tags			wastage
//	@Produce		json
//	@Param			request	body		entity.EC2InstanceWastageRequest	true	"Request"
//	@Success		200		{object}	entity.EC2InstanceWastageResponse
//	@Router			/wastage/api/v1/wastage/ec2-instance [post]
func (s API) EC2Instance(c echo.Context) error {
	start := time.Now()
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var req entity.EC2InstanceWastageRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var resp entity.EC2InstanceWastageResponse
	var err error

	reqJson, _ := json.Marshal(req)
	usage := model.UsageV2{
		ApiEndpoint:    "ec2-instance",
		Request:        reqJson,
		RequestId:      req.RequestId,
		CliVersion:     req.CliVersion,
		Response:       nil,
		FailureMessage: nil,
	}
	err = s.usageRepo.Create(&usage)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			fmsg := err.Error()
			usage.FailureMessage = &fmsg
		} else {
			usage.Response, _ = json.Marshal(resp)
			id := uuid.New()
			responseId := id.String()
			usage.ResponseId = &responseId
		}
		err = s.usageRepo.Update(usage.ID, usage)
		if err != nil {
			s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		}
	}()

	if req.Instance.State != types2.InstanceStateNameRunning {
		err = echo.NewHTTPError(http.StatusBadRequest, "instance is not running")
		return err
	}

	ec2RightSizingRecom, err := s.recomSvc.EC2InstanceRecommendation(req.Region, req.Instance, req.Volumes, req.Metrics, req.VolumeMetrics, req.Preferences)
	if err != nil {
		return err
	}

	ebsRightSizingRecoms := make(map[string]entity.EBSVolumeRecommendation)
	for _, vol := range req.Volumes {
		var ebsRightSizingRecom *entity.EBSVolumeRecommendation
		ebsRightSizingRecom, err = s.recomSvc.EBSVolumeRecommendation(req.Region, vol, req.VolumeMetrics[vol.HashedVolumeId], req.Preferences)
		if err != nil {
			return err
		}
		ebsRightSizingRecoms[vol.HashedVolumeId] = *ebsRightSizingRecom
	}
	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
	}

	// DO NOT change this, resp is used in updating usage
	resp = entity.EC2InstanceWastageResponse{
		RightSizing:       *ec2RightSizingRecom,
		VolumeRightSizing: ebsRightSizingRecoms,
	}
	// DO NOT change this, resp is used in updating usage
	return c.JSON(http.StatusOK, resp)
}

// AwsRDS godoc
//
//	@Summary		List wastage in AWS RDS
//	@Description	List wastage in AWS RDS
//	@Security		BearerToken
//	@Tags			wastage
//	@Produce		json
//	@Param			request	body		entity.AwsRdsWastageRequest	true	"Request"
//	@Success		200		{object}	entity.AwsRdsWastageResponse
//	@Router			/wastage/api/v1/wastage/aws-rds [post]
func (s API) AwsRDS(c echo.Context) error {
	start := time.Now()
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var req entity.AwsRdsWastageRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var resp entity.AwsRdsWastageResponse
	var err error

	reqJson, _ := json.Marshal(req)
	usage := model.UsageV2{
		ApiEndpoint:    "aws-rds",
		Request:        reqJson,
		RequestId:      req.RequestId,
		CliVersion:     req.CliVersion,
		Response:       nil,
		FailureMessage: nil,
	}
	err = s.usageRepo.Create(&usage)
	if err != nil {
		s.logger.Error("failed to create usage", zap.Error(err))
		return err
	}

	defer func() {
		if err != nil {
			fmsg := err.Error()
			usage.FailureMessage = &fmsg
		} else {
			usage.Response, _ = json.Marshal(resp)
			id := uuid.New()
			responseId := id.String()
			usage.ResponseId = &responseId
		}
		err = s.usageRepo.Update(usage.ID, usage)
		if err != nil {
			s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		}
	}()

	ec2RightSizingRecom, err := s.recomSvc.AwsRdsRecommendation(req.Region, req.Instance, req.Metrics, req.Preferences)
	if err != nil {
		s.logger.Error("failed to get aws rds recommendation", zap.Error(err))
		return err
	}

	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
	}

	// DO NOT change this, resp is used in updating usage
	resp = entity.AwsRdsWastageResponse{
		RightSizing: *ec2RightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage
	return c.JSON(http.StatusOK, resp)
}

// TriggerIngest godoc
//
//	@Summary		Trigger Ingest for the requested service
//	@Description	Trigger Ingest for the requested service
//	@Security		BearerToken
//	@Tags			wastage
//	@Produce		json
//	@Param			service		path	string		true	"service"
//	@Success		200
//	@Router			/wastage/api/v1/wastage/ingest/{service} [post]
func (s API) TriggerIngest(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	service := c.Param("service")

	switch service {
	case "aws-ec2-instance":
		err := s.ingestionSvc.DataAgeRepo.Delete("AWS::EC2::Instance")
		if err != nil {
			s.logger.Error("failed to delete data age", zap.Error(err), zap.String("service", service))
			return err
		}
		s.logger.Info("deleted data age for AWS::EC2::Instance ingestion will be triggered soon")
	case "aws-rds":
		err := s.ingestionSvc.DataAgeRepo.Delete("AWS::RDS::Instance")
		if err != nil {
			s.logger.Error("failed to delete data age", zap.Error(err), zap.String("service", service))
			return err
		}
		s.logger.Info("deleted data age for AWS::RDS::Instance ingestion will be triggered soon")
	}

	return c.NoContent(http.StatusOK)
}
