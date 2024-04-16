package wastage

import (
	"errors"
	aws2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-aws-describer/aws/describer"
	"github.com/kaytu-io/kaytu-aws-describer/aws/model"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/cost"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type API struct {
	tracer  trace.Tracer
	logger  *zap.Logger
	costSvc *cost.Service
}

func New(costSvc *cost.Service, logger *zap.Logger) API {
	return API{
		costSvc: costSvc,
		tracer:  otel.GetTracerProvider().Tracer("wastage.http.sources"),
		logger:  logger.Named("wastage-api"),
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

	cfg, err := aws.GetConfig(ctx, req.Credential.AccessKey, req.Credential.SecretKey, "", "", nil)
	if err != nil {
		return err
	}
	dctx := describer.WithDescribeContext(ctx, describer.DescribeContext{
		AccountID:   req.Credential.AccountID,
		Region:      req.Region,
		KaytuRegion: req.Region,
		Partition:   "",
	})
	cfg.Region = req.Region
	resources, err := describer.GetEC2Instance(dctx, cfg, map[string]string{"id": req.InstanceId})
	if err != nil {
		return err
	}

	if len(resources) == 0 {
		return errors.New("instance not found")
	}
	instance := resources[0].Description.(model.EC2InstanceDescription)

	var volumes []model.EC2VolumeDescription
	for _, bd := range instance.Instance.BlockDeviceMappings {
		res, err := describer.GetEC2Volume(dctx, cfg, map[string]string{"id": *bd.Ebs.VolumeId})
		if err != nil {
			return err
		}

		if len(res) == 0 {
			return errors.New("volume not found")
		}
		volume := res[0].Description.(model.EC2VolumeDescription)
		volumes = append(volumes, volume)
	}

	client := cloudwatch.NewFromConfig(cfg)
	paginator := cloudwatch.NewListMetricsPaginator(client, &cloudwatch.ListMetricsInput{
		Namespace: aws2.String("AWS/EC2"),
		Dimensions: []types.DimensionFilter{
			{
				Name:  aws2.String("InstanceId"),
				Value: instance.Instance.InstanceId,
			},
		},
	})
	startTime := time.Now().Add(-24 * 7 * time.Hour)
	endTime := time.Now()

	metrics := map[string][]types.Datapoint{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, p := range page.Metrics {
			statistics := []types.Statistic{
				types.StatisticAverage,
			}

			// Create input for GetMetricStatistics
			input := &cloudwatch.GetMetricStatisticsInput{
				Namespace:  aws2.String("AWS/EC2"),
				MetricName: p.MetricName,
				Dimensions: []types.Dimension{
					{
						Name:  aws2.String("InstanceId"),
						Value: instance.Instance.InstanceId,
					},
				},
				StartTime:  aws2.Time(startTime),
				EndTime:    aws2.Time(endTime),
				Period:     aws2.Int32(60 * 60), // 1 hour intervals
				Statistics: statistics,
			}

			// Get metric data
			resp, err := client.GetMetricStatistics(ctx, input)
			if err != nil {
				return err
			}

			metrics[*p.MetricName] = resp.Datapoints
		}
	}

	currentCost, err := s.costSvc.GetEC2InstanceCost(req.Region, instance, volumes, metrics)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, entity.EC2InstanceWastageResponse{
		CurrentCost: currentCost,
	})
}

func (s API) Register(g *echo.Group) {
	g.POST("/ec2-instance", httpserver.AuthorizeHandler(s.EC2Instance, api.ViewerRole))
}
