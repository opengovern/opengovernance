package wastage

import (
	"encoding/json"
	"fmt"
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
	usageV1Repo  repo.UsageRepo
	recomSvc     *recommendation.Service
	ingestionSvc *ingestion.Service
}

func New(costSvc *cost.Service, recomSvc *recommendation.Service, ingestionService *ingestion.Service, usageV1Repo repo.UsageRepo, usageRepo repo.UsageV2Repo, logger *zap.Logger) API {
	return API{
		costSvc:      costSvc,
		recomSvc:     recomSvc,
		usageRepo:    usageRepo,
		usageV1Repo:  usageV1Repo,
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
	i.PUT("/usages/migrate", s.MigrateUsages)
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
		err = fmt.Errorf("failed to get ec2 instance recommendation: %s", err.Error())
		return err
	}

	ebsRightSizingRecoms := make(map[string]entity.EBSVolumeRecommendation)
	for _, vol := range req.Volumes {
		var ebsRightSizingRecom *entity.EBSVolumeRecommendation
		ebsRightSizingRecom, err = s.recomSvc.EBSVolumeRecommendation(req.Region, vol, req.VolumeMetrics[vol.HashedVolumeId], req.Preferences)
		if err != nil {
			err = fmt.Errorf("failed to get ebs volume %s recommendation: %s", vol.HashedVolumeId, err.Error())
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
//	@Router			/wastage/api/v1/wastage-ingestion/ingest/{service} [post]
func (s API) TriggerIngest(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	service := c.Param("service")
	dataAge, err := s.ingestionSvc.DataAgeRepo.List()
	if err != nil {
		return err
	}

	var ec2InstanceData *model.DataAge
	var rdsData *model.DataAge
	for _, data := range dataAge {
		data := data
		switch data.DataType {
		case "AWS::EC2::Instance":
			ec2InstanceData = &data
		case "AWS::RDS::Instance":
			rdsData = &data
		}
	}
	go func() {
		switch service {
		case "aws-ec2-instance":
			s.logger.Info("Ingestion for EC2 started")
			err := s.ingestionSvc.IngestEc2Instances(ctx)
			if err != nil {
				s.logger.Error(err.Error())
			}
			if ec2InstanceData == nil {
				err = s.ingestionSvc.DataAgeRepo.Create(&model.DataAge{
					DataType:  "AWS::EC2::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					s.logger.Error(err.Error())
				}
			} else {
				err = s.ingestionSvc.DataAgeRepo.Update("AWS::EC2::Instance", model.DataAge{
					DataType:  "AWS::EC2::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					s.logger.Error(err.Error())
				}
			}
		case "aws-rds":
			s.logger.Info("Ingestion for RDS started")
			err = s.ingestionSvc.IngestRDS()
			if err != nil {
				s.logger.Error(err.Error())
			}
			if rdsData == nil {
				err = s.ingestionSvc.DataAgeRepo.Create(&model.DataAge{
					DataType:  "AWS::RDS::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					s.logger.Error(err.Error())
				}
			} else {
				err = s.ingestionSvc.DataAgeRepo.Update("AWS::RDS::Instance", model.DataAge{
					DataType:  "AWS::RDS::Instance",
					UpdatedAt: time.Now(),
				})
				if err != nil {
					s.logger.Error(err.Error())
				}
			}
		}
	}()

	return c.NoContent(http.StatusOK)
}

// MigrateUsages godoc
//
//	@Summary		Migrate all usages from v1 to v2 and recall and get the response for each again
//	@Description	Migrate all usages from v1 to v2 and recall and get the response for each again
//	@Security		BearerToken
//	@Tags			wastage
//	@Produce		json
//	@Success		200
//	@Router			/wastage/api/v1/wastage-ingestion/usages/migrate [post]
func (s API) MigrateUsages(c echo.Context) error {
	go func() {
		s.logger.Info("Usage table migration started")

		for true {
			u, err := s.usageV1Repo.GetRandomNotMoved()
			if err != nil {
				s.logger.Error("error while getting usage_v1 usages list", zap.Error(err))
				break
			}
			if u == nil {
				break
			}
			start := time.Now()
			var req entity.EC2InstanceWastageRequest
			err = u.Request.Scan(&req)
			requestId := fmt.Sprintf("usage_v1_%v", u.ID)
			cliVersion := "unknown"
			req.RequestId = &requestId
			req.CliVersion = &cliVersion

			usage := model.UsageV2{
				ApiEndpoint:    "ec2-instance",
				Request:        u.Request,
				RequestId:      &requestId,
				CliVersion:     &cliVersion,
				Response:       nil,
				FailureMessage: nil,
			}
			err = s.usageRepo.Create(&usage)
			if err != nil {
				s.logger.Error("error while putting usage in database",
					zap.Any("usage_id", u.ID),
					zap.Any("usage", usage),
					zap.Error(err))

				u.Moved = true
				err = s.usageV1Repo.Update(usage.ID, *u)
				if err != nil {
					s.logger.Error("failed to update usage moved flag", zap.Any("usage_id", u.ID), zap.Error(err))
					continue
				}
				continue
			}

			if req.Instance.State != types2.InstanceStateNameRunning {
				err = echo.NewHTTPError(http.StatusBadRequest, "instance is not running")
				s.logger.Error("request failed", zap.Any("usage_v1_id", u.ID), zap.Any("usage", usage), zap.Error(err))
				fmsg := err.Error()
				usage.FailureMessage = &fmsg
				err = s.usageRepo.Update(usage.ID, usage)
				if err != nil {
					s.logger.Error("failed to update usage", zap.Any("usage_v1_id", u.ID), zap.Error(err), zap.Any("usage", usage))
				}
				u.Moved = true
				err = s.usageV1Repo.Update(usage.ID, *u)
				if err != nil {
					s.logger.Error("failed to update usage moved flag", zap.Any("usage_id", u.ID), zap.Error(err))
					continue
				}
				continue
			}

			ec2RightSizingRecom, err := s.recomSvc.EC2InstanceRecommendation(req.Region, req.Instance, req.Volumes, req.Metrics, req.VolumeMetrics, req.Preferences)
			if err != nil {
				s.logger.Error("request failed", zap.Any("usage_v1_id", u.ID), zap.Any("usage", usage), zap.Error(err))
				fmsg := err.Error()
				usage.FailureMessage = &fmsg
				err = s.usageRepo.Update(usage.ID, usage)
				if err != nil {
					s.logger.Error("failed to update usage", zap.Any("usage_v1_id", u.ID), zap.Error(err), zap.Any("usage", usage))
				}
				u.Moved = true
				err = s.usageV1Repo.Update(usage.ID, *u)
				if err != nil {
					s.logger.Error("failed to update usage moved flag", zap.Any("usage_id", u.ID), zap.Error(err))
					continue
				}
				continue
			}

			ebsRightSizingRecoms := make(map[string]entity.EBSVolumeRecommendation)
			for _, vol := range req.Volumes {
				var ebsRightSizingRecom *entity.EBSVolumeRecommendation
				ebsRightSizingRecom, err = s.recomSvc.EBSVolumeRecommendation(req.Region, vol, req.VolumeMetrics[vol.HashedVolumeId], req.Preferences)
				if err != nil {
					s.logger.Error("request failed", zap.Any("usage_v1_id", u.ID), zap.Any("usage", usage), zap.Error(err))
					fmsg := err.Error()
					usage.FailureMessage = &fmsg
					err = s.usageRepo.Update(usage.ID, usage)
					if err != nil {
						s.logger.Error("failed to update usage", zap.Any("usage_v1_id", u.ID), zap.Error(err), zap.Any("usage", usage))
					}
					u.Moved = true
					err = s.usageV1Repo.Update(usage.ID, *u)
					if err != nil {
						s.logger.Error("failed to update usage moved flag", zap.Any("usage_id", u.ID), zap.Error(err))
						continue
					}
					continue
				}
				ebsRightSizingRecoms[vol.HashedVolumeId] = *ebsRightSizingRecom
			}
			elapsed := time.Since(start).Seconds()
			usage.Latency = &elapsed

			// DO NOT change this, resp is used in updating usage
			resp := entity.EC2InstanceWastageResponse{
				RightSizing:       *ec2RightSizingRecom,
				VolumeRightSizing: ebsRightSizingRecoms,
			}
			// DO NOT change this, resp is used in updating usage

			usage.Response, _ = json.Marshal(resp)
			id := uuid.New()
			responseId := id.String()
			usage.ResponseId = &responseId
			err = s.usageRepo.Update(usage.ID, usage)
			if err != nil {
				s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage_v1_id", u.ID), zap.Any("usage", usage))
			}

			u.Moved = true
			err = s.usageV1Repo.Update(usage.ID, *u)
			if err != nil {
				s.logger.Error("failed to update usage moved flag", zap.Any("usage_id", u.ID), zap.Error(err))
				continue
			}

		}

	}()

	return c.NoContent(http.StatusOK)
}
