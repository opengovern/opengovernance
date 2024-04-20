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
		s.logger.Error("failed to get ec2 instance cost", zap.Error(err))
		return err
	}

	currentVolumeCosts := make(map[string]float64)
	for _, vol := range req.Volumes {
		volumeCost, err := s.costSvc.GetEBSVolumeCost(req.Region, vol, req.VolumeMetrics[*vol.VolumeId])
		if err != nil {
			s.logger.Error("failed to get ebs volume cost", zap.Error(err))
			return err
		}
		currentVolumeCosts[*vol.VolumeId] = volumeCost
	}

	ec2RightSizingRecom, err := s.recomSvc.EC2InstanceRecommendation(req.Region, req.Instance, req.Volumes, req.Metrics, req.Preferences)
	if err != nil {
		return err
	}

	ebsRightSizingRecoms := make(map[string]*recommendation.EbsVolumeRecommendation)
	for _, vol := range req.Volumes {
		ebsRightSizingRecom, err := s.recomSvc.EBSVolumeRecommendation(req.Region, vol, req.VolumeMetrics[*vol.VolumeId], req.Preferences)
		if err != nil {
			return err
		}
		if ebsRightSizingRecom == nil {
			continue
		}
		ebsRightSizingRecoms[*vol.VolumeId] = ebsRightSizingRecom
	}
	newVolumes := make([]types2.Volume, 0)
	for _, vol := range ebsRightSizingRecoms {
		newVolumes = append(newVolumes, vol.NewVolume)
	}
	ec2RightSizingRecom.NewVolumes = newVolumes

	costAfterRightSizing, err := s.costSvc.GetEC2InstanceCost(req.Region, ec2RightSizingRecom.NewInstance, ec2RightSizingRecom.NewVolumes, req.Metrics)
	if err != nil {
		return err
	}
	ebsTotalSavings := make(map[string]float64)
	ebsCostAfterRightSizing := make(map[string]float64)
	for _, vol := range ec2RightSizingRecom.NewVolumes {
		volumeCost, err := s.costSvc.GetEBSVolumeCost(req.Region, vol, req.VolumeMetrics[*vol.VolumeId])
		if err != nil {
			s.logger.Error("failed to get ebs volume cost", zap.Error(err))
			return err
		}
		ebsCostAfterRightSizing[*vol.VolumeId] = volumeCost
		ebsTotalSavings[*vol.VolumeId] = currentVolumeCosts[*vol.VolumeId] - volumeCost
	}

	var rightSizingRecomResp *entity.RightSizingRecommendation
	if ec2RightSizingRecom != nil {
		rightSizingRecomResp = &entity.RightSizingRecommendation{
			TargetInstanceType:        ec2RightSizingRecom.NewInstanceType.InstanceType,
			Saving:                    resp.CurrentCost - costAfterRightSizing,
			CurrentCost:               resp.CurrentCost,
			TargetCost:                costAfterRightSizing,
			AvgCPUUsage:               ec2RightSizingRecom.AvgCPUUsage,
			TargetCores:               ec2RightSizingRecom.NewInstanceType.VCPUStr,
			AvgNetworkBandwidth:       ec2RightSizingRecom.AvgNetworkBandwidth,
			TargetNetworkPerformance:  ec2RightSizingRecom.NewInstanceType.NetworkPerformance,
			CurrentNetworkPerformance: ec2RightSizingRecom.CurrentInstanceType.NetworkPerformance,
			CurrentMemory:             ec2RightSizingRecom.CurrentInstanceType.Memory,
			TargetMemory:              ec2RightSizingRecom.NewInstanceType.Memory,
			VolumesCurrentSizes:       make(map[string]int32),
			VolumesTargetSizes:        make(map[string]int32),
			VolumesCurrentTypes:       make(map[string]types2.VolumeType),
			VolumesTargetTypes:        make(map[string]types2.VolumeType),
			VolumesCurrentIOPS:        make(map[string]int32),
			VolumesTargetIOPS:         make(map[string]int32),
			VolumesCurrentThroughput:  make(map[string]int32),
			VolumesTargetThroughput:   make(map[string]int32),
			VolumesCurrentCosts:       make(map[string]float64),
			VolumesTargetCosts:        make(map[string]float64),
		}
	}

	for k, v := range ebsRightSizingRecoms {
		if rightSizingRecomResp == nil {
			rightSizingRecomResp = &entity.RightSizingRecommendation{
				VolumesCurrentSizes:      make(map[string]int32),
				VolumesTargetSizes:       make(map[string]int32),
				VolumesCurrentTypes:      make(map[string]types2.VolumeType),
				VolumesTargetTypes:       make(map[string]types2.VolumeType),
				VolumesCurrentIOPS:       make(map[string]int32),
				VolumesTargetIOPS:        make(map[string]int32),
				VolumesCurrentThroughput: make(map[string]int32),
				VolumesTargetThroughput:  make(map[string]int32),
				VolumesCurrentCosts:      make(map[string]float64),
				VolumesTargetCosts:       make(map[string]float64),
			}
		}
		rightSizingRecomResp.VolumesCurrentCosts[k] = currentVolumeCosts[k]
		rightSizingRecomResp.VolumesTargetCosts[k] = ebsCostAfterRightSizing[k]
		rightSizingRecomResp.VolumesCurrentSizes[k] = v.CurrentSize
		rightSizingRecomResp.VolumesTargetSizes[k] = v.NewSize
		rightSizingRecomResp.VolumesCurrentTypes[k] = v.CurrentVolumeType
		rightSizingRecomResp.VolumesTargetTypes[k] = v.NewVolumeType
		if v.CurrentProvisionedIOPS != nil {
			rightSizingRecomResp.VolumesCurrentIOPS[k] = *v.CurrentProvisionedIOPS
		}
		if v.NewProvisionedIOPS != nil {
			rightSizingRecomResp.VolumesTargetIOPS[k] = *v.NewProvisionedIOPS
		}
		if v.CurrentProvisionedThroughput != nil {
			rightSizingRecomResp.VolumesCurrentThroughput[k] = *v.CurrentProvisionedThroughput
		}
		if v.NewProvisionedThroughput != nil {
			rightSizingRecomResp.VolumesTargetThroughput[k] = *v.NewProvisionedThroughput
		}
	}

	resp.TotalSavings += resp.CurrentCost - costAfterRightSizing
	resp.RightSizing = rightSizingRecomResp
	resp.EbsTotalSavings = ebsTotalSavings
	return c.JSON(http.StatusOK, resp)
}

func (s API) Register(g *echo.Group) {
	g.POST("/ec2-instance", httpserver.AuthorizeHandler(s.EC2Instance, api.ViewerRole))
}
