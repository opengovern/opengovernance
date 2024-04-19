package wastage

import (
	"encoding/json"
	types2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/cost"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
)

type API struct {
	tracer    trace.Tracer
	logger    *zap.Logger
	costSvc   *cost.Service
	usageRepo repo.UsageRepo
	recomSvc  *recommendation.Service
}

func New(costSvc *cost.Service, recomSvc *recommendation.Service, usageRepo repo.UsageRepo, logger *zap.Logger) API {
	return API{
		costSvc:   costSvc,
		recomSvc:  recomSvc,
		usageRepo: usageRepo,
		tracer:    otel.GetTracerProvider().Tracer("wastage.http.sources"),
		logger:    logger.Named("wastage-api"),
	}
}

// EC2Instance godoc
//
//	@Summary		List wastage in EC2 Instances
//	@Description	List wastage in EC2 Instances
//	@Security		BearerToken
//	@Tags			wastage
//	@Produce		json
//	@Param			request			body		entity.EC2InstanceWastageRequest	true	"Request"
//	@Success		200				{object}	entity.EC2InstanceWastageResponse
//	@Router			/wastage/api/v1/wastage/ec2-instance [post]
func (s API) EC2Instance(c echo.Context) error {
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
	usage := model.Usage{
		Request:  reqJson,
		Response: nil,
	}
	err = s.usageRepo.Create(&usage)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			usage.Response, _ = json.Marshal(err)
		} else {
			usage.Response, _ = json.Marshal(resp)
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

	resp.CurrentCost, err = s.costSvc.GetEC2InstanceCost(req.Region, req.Instance, req.Volumes, req.Metrics)
	if err != nil {
		return err
	}

	rightSizingRecom, err := s.recomSvc.EC2InstanceRecommendation(req.Region, req.Instance, req.Volumes, req.Metrics, req.Preferences)
	if err != nil {
		return err
	}

	costAfterRightSizing, err := s.costSvc.GetEC2InstanceCost(req.Region, rightSizingRecom.NewInstance, rightSizingRecom.NewVolumes, req.Metrics)
	if err != nil {
		return err
	}

	var rightSizingRecomResp *entity.RightSizingRecommendation
	if rightSizingRecom != nil {
		rightSizingRecomResp = &entity.RightSizingRecommendation{
			TargetInstanceType:        rightSizingRecom.NewInstanceType.InstanceType,
			Saving:                    resp.CurrentCost - costAfterRightSizing,
			CurrentCost:               resp.CurrentCost,
			TargetCost:                costAfterRightSizing,
			AvgCPUUsage:               rightSizingRecom.AvgCPUUsage,
			TargetCores:               rightSizingRecom.NewInstanceType.VCPUStr,
			AvgNetworkBandwidth:       rightSizingRecom.AvgNetworkBandwidth,
			TargetNetworkPerformance:  rightSizingRecom.NewInstanceType.NetworkPerformance,
			CurrentNetworkPerformance: rightSizingRecom.CurrentInstanceType.NetworkPerformance,
			CurrentMemory:             rightSizingRecom.CurrentInstanceType.Memory,
			TargetMemory:              rightSizingRecom.NewInstanceType.Memory,
		}
	}

	resp.TotalSavings += resp.CurrentCost - costAfterRightSizing
	resp.RightSizing = rightSizingRecomResp
	return c.JSON(http.StatusOK, resp)
}

func (s API) Register(g *echo.Group) {
	g.POST("/ec2-instance", httpserver.AuthorizeHandler(s.EC2Instance, api.ViewerRole))
}
