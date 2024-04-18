package wastage

import (
	types2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/cost"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
)

type API struct {
	tracer   trace.Tracer
	logger   *zap.Logger
	costSvc  *cost.Service
	recomSvc *recommendation.Service
}

func New(costSvc *cost.Service, recomSvc *recommendation.Service, logger *zap.Logger) API {
	return API{
		costSvc:  costSvc,
		recomSvc: recomSvc,
		tracer:   otel.GetTracerProvider().Tracer("wastage.http.sources"),
		logger:   logger.Named("wastage-api"),
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

	if req.Instance.State.Name != types2.InstanceStateNameRunning {
		return echo.NewHTTPError(http.StatusBadRequest, "instance is not running")
	}

	currentCost, err := s.costSvc.GetEC2InstanceCost(req.Region, req.Instance, req.Volumes, req.Metrics)
	if err != nil {
		return err
	}

	rightSizingRecom, err := s.recomSvc.EC2InstanceRecommendation(req.Region, req.Instance, req.Volumes, req.Metrics)
	if err != nil {
		return err
	}

	totalSavings := float64(0)
	costAfterRightSizing, err := s.costSvc.GetEC2InstanceCost(req.Region, rightSizingRecom.NewInstance, rightSizingRecom.NewVolumes, req.Metrics)
	if err != nil {
		return err
	}

	totalSavings += currentCost - costAfterRightSizing
	return c.JSON(http.StatusOK, entity.EC2InstanceWastageResponse{
		CurrentCost:  currentCost,
		TotalSavings: totalSavings,
		RightSizing: entity.RightSizingRecommendation{
			Saving:             currentCost - costAfterRightSizing,
			TargetInstanceType: rightSizingRecom.NewInstanceType.InstanceType,
		},
	})
}

func (s API) Register(g *echo.Group) {
	g.POST("/ec2-instance", httpserver.AuthorizeHandler(s.EC2Instance, api.ViewerRole))
}
