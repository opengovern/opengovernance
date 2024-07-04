package wastage

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/alitto/pond"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	types2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/config"
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
	"golang.org/x/mod/semver"
	"golang.org/x/net/context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type API struct {
	cfg            config.WastageConfig
	tracer         trace.Tracer
	logger         *zap.Logger
	blobClient     *azblob.Client
	blobWorkerPool *pond.WorkerPool
	costSvc        *cost.Service
	usageRepo      repo.UsageV2Repo
	usageV1Repo    repo.UsageRepo
	userRepo       repo.UserRepo
	orgRepo        repo.OrganizationRepo
	recomSvc       *recommendation.Service
	ingestionSvc   *ingestion.Service
}

func New(cfg config.WastageConfig, blobClient *azblob.Client, blobWorkerPool *pond.WorkerPool, costSvc *cost.Service, recomSvc *recommendation.Service, ingestionService *ingestion.Service, usageV1Repo repo.UsageRepo, usageRepo repo.UsageV2Repo, userRepo repo.UserRepo, orgRepo repo.OrganizationRepo, logger *zap.Logger) API {
	return API{
		cfg:            cfg,
		blobClient:     blobClient,
		blobWorkerPool: blobWorkerPool,
		costSvc:        costSvc,
		recomSvc:       recomSvc,
		usageRepo:      usageRepo,
		usageV1Repo:    usageV1Repo,
		userRepo:       userRepo,
		orgRepo:        orgRepo,
		ingestionSvc:   ingestionService,
		tracer:         otel.GetTracerProvider().Tracer("wastage.http.sources"),
		logger:         logger.Named("wastage-api"),
	}
}

func (s API) Register(e *echo.Echo) {
	g := e.Group("/api/v1/wastage")
	g.POST("/configuration", s.Configuration)
	g.POST("/ec2-instance", httpserver.AuthorizeHandler(s.EC2Instance, api.ViewerRole))
	g.POST("/aws-rds", httpserver.AuthorizeHandler(s.AwsRDS, api.ViewerRole))
	g.POST("/aws-rds-cluster", httpserver.AuthorizeHandler(s.AwsRDSCluster, api.ViewerRole))
	i := e.Group("/api/v1/wastage-ingestion")
	i.PUT("/ingest/:service", httpserver.AuthorizeHandler(s.TriggerIngest, api.InternalRole))
	i.GET("/usages/:id", httpserver.AuthorizeHandler(s.GetUsage, api.InternalRole))
	i.GET("/usages/accountID/:endpoint/:accountID", httpserver.AuthorizeHandler(s.GetUsageIDByAccountID, api.InternalRole))
	i.GET("/usages/accountID/:endpoint/:accountID/:groupBy/last", httpserver.AuthorizeHandler(s.GetLastUsageIDByAccountID, api.InternalRole))
	i.PUT("/usages/migrate", s.MigrateUsages)
	i.PUT("/usages/migrate/v2", s.MigrateUsagesV2)
	i.PUT("/usages/fill-rds-costs", s.FillRdsCosts)
	i.POST("/user", httpserver.AuthorizeHandler(s.CreateUser, api.InternalRole))
	i.PUT("/user/:userId", httpserver.AuthorizeHandler(s.UpdateUser, api.InternalRole))
	i.POST("/organization", httpserver.AuthorizeHandler(s.CreateOrganization, api.InternalRole))
	i.PUT("/organization/:organizationId", httpserver.AuthorizeHandler(s.UpdateOrganization, api.InternalRole))
}

func (s API) Configuration(c echo.Context) error {
	return c.JSON(http.StatusOK, entity.Configuration{
		EC2LazyLoad:        20,
		RDSLazyLoad:        20,
		KubernetesLazyLoad: 10000,
	})
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
func (s API) EC2Instance(echoCtx echo.Context) error {
	ctx := echoCtx.Request().Context()
	start := time.Now()
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(echoCtx.Request().Header))
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var req entity.EC2InstanceWastageRequest
	if err := echoCtx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := echoCtx.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var resp entity.EC2InstanceWastageResponse
	var err error

	stats := model.Statistics{
		AccountID:   req.Identification["account"],
		OrgEmail:    req.Identification["org_m_email"],
		ResourceID:  req.Instance.HashedInstanceId,
		Auth0UserId: httpserver.GetUserID(echoCtx),
	}
	statsOut, _ := json.Marshal(stats)

	fullReqJson, _ := json.Marshal(req)
	metrics := req.Metrics
	volMetrics := req.VolumeMetrics
	req.Metrics = nil
	req.VolumeMetrics = nil
	trimmedReqJson, _ := json.Marshal(req)
	req.Metrics = metrics
	req.VolumeMetrics = volMetrics

	if req.RequestId == nil {
		id := uuid.New().String()
		req.RequestId = &id
	}

	s.blobWorkerPool.Submit(func() {
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("ec2-instance/%s.json", *req.RequestId), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})

	usage := model.UsageV2{
		ApiEndpoint:    "ec2-instance",
		Request:        trimmedReqJson,
		RequestId:      req.RequestId,
		CliVersion:     req.CliVersion,
		Response:       nil,
		FailureMessage: nil,
		Statistics:     statsOut,
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

			recom := entity.RightsizingEC2Instance{}
			if resp.RightSizing.Recommended != nil {
				recom = *resp.RightSizing.Recommended
			}

			instanceCost := resp.RightSizing.Current.Cost
			recomInstanceCost := recom.Cost

			volumeCurrentCost := 0.0
			volumeRecomCost := 0.0
			for _, v := range resp.VolumeRightSizing {
				volumeCurrentCost += v.Current.Cost
				if v.Recommended != nil {
					volumeRecomCost += v.Recommended.Cost
				}
			}

			stats.CurrentCost = instanceCost + volumeCurrentCost
			stats.RecommendedCost = recomInstanceCost + volumeRecomCost
			stats.Savings = (instanceCost + volumeCurrentCost) - (recomInstanceCost + volumeRecomCost)
			stats.EC2InstanceCurrentCost = instanceCost
			stats.EC2InstanceRecommendedCost = recomInstanceCost
			stats.EC2InstanceSavings = instanceCost - recomInstanceCost
			stats.EBSCurrentCost = volumeCurrentCost
			stats.EBSRecommendedCost = volumeRecomCost
			stats.EBSSavings = volumeCurrentCost - volumeRecomCost
			stats.EBSVolumeCount = len(resp.VolumeRightSizing)

			statsOut, _ := json.Marshal(stats)
			usage.Statistics = statsOut
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

	if req.Loading {
		return echoCtx.JSON(http.StatusOK, entity.EC2InstanceWastageResponse{})
	}

	usageAverageType := recommendation.UsageAverageTypeMax
	if req.CliVersion == nil || semver.Compare("v"+*req.CliVersion, "v0.5.2") < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "plugin version is no longer supported - please update to the latest version")
	}

	ok, err := s.checkAccountsLimit(httpserver.GetUserID(echoCtx), req.Identification["org_m_email"], req.Identification["account"])
	if err != nil {
		s.logger.Error("failed to check profile limit", zap.Error(err))
		return err
	}
	if !ok {
		err = s.checkPremiumAndSendErr(echoCtx, req.Identification["org_m_email"], "profile")
		if err != nil {
			return err
		}
	}

	ok, err = s.checkEC2InstanceLimit(httpserver.GetUserID(echoCtx), req.Identification["org_m_email"])
	if err != nil {
		s.logger.Error("failed to check aws ec2 instance limit", zap.Error(err))
		return err
	}
	if !ok {
		err = s.checkPremiumAndSendErr(echoCtx, req.Identification["org_m_email"], "ec2 instance")
		if err != nil {
			return err
		}
	}

	ec2RightSizingRecom, err := s.recomSvc.EC2InstanceRecommendation(ctx, req.Region, req.Instance, req.Volumes, req.Metrics, req.VolumeMetrics, req.Preferences, usageAverageType)
	if err != nil {
		err = fmt.Errorf("failed to get ec2 instance recommendation: %s", err.Error())
		return err
	}

	ebsRightSizingRecoms := make(map[string]entity.EBSVolumeRecommendation)
	for _, vol := range req.Volumes {
		//ok, err := checkEBSVolumeLimit(s.usageRepo, httpserver.GetUserID(c), req.Identification["org_m_email"])
		//if err != nil {
		//	s.logger.Error("failed to check aws ebs volume limit", zap.Error(err))
		//	return err
		//}
		//if !ok {
		//	err = s.checkPremiumAndSendErr(c, req.Identification["org_m_email"], "ebs volume")
		//	if err != nil {
		//		return err
		//	}
		//}
		var ebsRightSizingRecom *entity.EBSVolumeRecommendation
		ebsRightSizingRecom, err = s.recomSvc.EBSVolumeRecommendation(ctx, req.Region, vol, req.VolumeMetrics[vol.HashedVolumeId], req.Preferences, usageAverageType)
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
	return echoCtx.JSON(http.StatusOK, resp)
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
func (s API) AwsRDS(echoCtx echo.Context) error {
	ctx := echoCtx.Request().Context()
	start := time.Now()
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(echoCtx.Request().Header))
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var req entity.AwsRdsWastageRequest
	if err := echoCtx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := echoCtx.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var resp entity.AwsRdsWastageResponse
	var err error

	stats := model.Statistics{
		AccountID:   req.Identification["account"],
		OrgEmail:    req.Identification["org_m_email"],
		ResourceID:  req.Instance.HashedInstanceId,
		Auth0UserId: httpserver.GetUserID(echoCtx),
	}
	statsOut, _ := json.Marshal(stats)

	fullReqJson, _ := json.Marshal(req)
	metrics := req.Metrics
	req.Metrics = nil
	trimmedReqJson, _ := json.Marshal(req)
	req.Metrics = metrics

	if req.RequestId == nil {
		id := uuid.New().String()
		req.RequestId = &id
	}

	s.blobWorkerPool.Submit(func() {
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("aws-rds/%s.json", *req.RequestId), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})
	usage := model.UsageV2{
		ApiEndpoint:    "aws-rds",
		Request:        trimmedReqJson,
		RequestId:      req.RequestId,
		CliVersion:     req.CliVersion,
		Response:       nil,
		FailureMessage: nil,
		Statistics:     statsOut,
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

			recom := entity.RightsizingAwsRds{}
			if resp.RightSizing.Recommended != nil {
				recom = *resp.RightSizing.Recommended
			}
			stats.CurrentCost = resp.RightSizing.Current.Cost
			stats.RecommendedCost = recom.Cost
			stats.Savings = resp.RightSizing.Current.Cost - recom.Cost
			stats.RDSInstanceCurrentCost = resp.RightSizing.Current.Cost
			stats.RDSInstanceRecommendedCost = recom.Cost
			stats.RDSInstanceSavings = resp.RightSizing.Current.Cost - recom.Cost

			statsOut, _ := json.Marshal(stats)
			usage.Statistics = statsOut
		}
		err = s.usageRepo.Update(usage.ID, usage)
		if err != nil {
			s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		}
	}()
	if req.Loading {
		return echoCtx.JSON(http.StatusOK, entity.AwsRdsWastageResponse{})
	}

	usageAverageType := recommendation.UsageAverageTypeMax
	if req.CliVersion == nil || semver.Compare("v"+*req.CliVersion, "v0.5.2") < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "plugin version is no longer supported - please update to the latest version")
	}

	ok, err := s.checkAccountsLimit(httpserver.GetUserID(echoCtx), req.Identification["org_m_email"], req.Identification["account"])
	if err != nil {
		s.logger.Error("failed to check profile limit", zap.Error(err))
		return err
	}
	if !ok {
		err = s.checkPremiumAndSendErr(echoCtx, req.Identification["org_m_email"], "profile")
		if err != nil {
			return err
		}
	}

	ok, err = s.checkRDSInstanceLimit(httpserver.GetUserID(echoCtx), req.Identification["org_m_email"])
	if err != nil {
		s.logger.Error("failed to check aws rds instance limit", zap.Error(err))
		return err
	}
	if !ok {
		err = s.checkPremiumAndSendErr(echoCtx, req.Identification["org_m_email"], "rds instance")
		if err != nil {
			return err
		}
	}

	rdsRightSizingRecom, err := s.recomSvc.AwsRdsRecommendation(ctx, req.Region, req.Instance, req.Metrics, req.Preferences, usageAverageType)
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
		RightSizing: *rdsRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage
	return echoCtx.JSON(http.StatusOK, resp)
}

// AwsRDSCluster godoc
//
//	@Summary		List wastage in AWS RDS Cluster
//	@Description	List wastage in AWS RDS Cluster
//	@Security		BearerToken
//	@Tags			wastage
//	@Produce		json
//	@Param			request	body		entity.AwsClusterWastageRequest	true	"Request"
//	@Success		200		{object}	entity.AwsClusterWastageResponse
//	@Router			/wastage/api/v1/wastage/aws-rds-cluster [post]
func (s API) AwsRDSCluster(echoCtx echo.Context) error {
	ctx := echoCtx.Request().Context()
	start := time.Now()
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(echoCtx.Request().Header))
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var req entity.AwsClusterWastageRequest
	if err := echoCtx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := echoCtx.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var resp entity.AwsClusterWastageResponse
	var err error

	stats := model.Statistics{
		AccountID:   req.Identification["account"],
		OrgEmail:    req.Identification["org_m_email"],
		ResourceID:  req.Cluster.HashedClusterId,
		Auth0UserId: httpserver.GetUserID(echoCtx),
	}
	statsOut, _ := json.Marshal(stats)

	fullReqJson, _ := json.Marshal(req)
	metrics := req.Metrics
	req.Metrics = nil
	trimmedReqJson, _ := json.Marshal(req)
	req.Metrics = metrics

	if req.RequestId == nil {
		id := uuid.New().String()
		req.RequestId = &id
	}

	s.blobWorkerPool.Submit(func() {
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("aws-rds-cluster/%s.json", *req.RequestId), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})
	usage := model.UsageV2{
		ApiEndpoint:    "aws-rds-cluster",
		Request:        trimmedReqJson,
		RequestId:      req.RequestId,
		CliVersion:     req.CliVersion,
		Response:       nil,
		FailureMessage: nil,
		Statistics:     statsOut,
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

			recom := entity.RightsizingAwsRds{}
			for _, instance := range resp.RightSizing {
				recom.Region = instance.Recommended.Region
				recom.InstanceType = instance.Recommended.InstanceType
				recom.Engine = instance.Recommended.Engine
				recom.EngineVersion = instance.Recommended.EngineVersion
				recom.ClusterType = instance.Recommended.ClusterType
				recom.VCPU += instance.Recommended.VCPU
				recom.MemoryGb += instance.Recommended.MemoryGb
				recom.StorageType = instance.Recommended.StorageType
				recom.StorageSize = instance.Recommended.StorageSize
				recom.StorageIops = instance.Recommended.StorageIops
				recom.StorageThroughput = instance.Recommended.StorageThroughput

				recom.Cost += instance.Recommended.Cost
				recom.ComputeCost += instance.Recommended.ComputeCost
				recom.StorageCost += instance.Recommended.StorageCost

				stats.CurrentCost += instance.Current.Cost
				stats.RDSInstanceCurrentCost += instance.Current.Cost
			}
			stats.Savings = stats.CurrentCost - recom.Cost
			stats.RDSInstanceSavings = stats.CurrentCost - recom.Cost
			stats.RecommendedCost = recom.Cost
			stats.RDSInstanceRecommendedCost = recom.Cost

			statsOut, _ := json.Marshal(stats)
			usage.Statistics = statsOut
		}
		err = s.usageRepo.Update(usage.ID, usage)
		if err != nil {
			s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		}
	}()
	if req.Loading {
		return echoCtx.JSON(http.StatusOK, entity.AwsClusterWastageResponse{})
	}

	usageAverageType := recommendation.UsageAverageTypeMax
	if req.CliVersion == nil || semver.Compare("v"+*req.CliVersion, "v0.5.2") < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "plugin version is no longer supported - please update to the latest version")
	}

	resp = entity.AwsClusterWastageResponse{
		RightSizing: make(map[string]entity.AwsRdsRightsizingRecommendation),
	}

	ok, err := s.checkAccountsLimit(httpserver.GetUserID(echoCtx), req.Identification["org_m_email"], req.Identification["account"])
	if err != nil {
		s.logger.Error("failed to check profile limit", zap.Error(err))
		return err
	}
	if !ok {
		err = s.checkPremiumAndSendErr(echoCtx, req.Identification["org_m_email"], "profile")
		if err != nil {
			return err
		}
	}

	ok, err = s.checkRDSClusterLimit(httpserver.GetUserID(echoCtx), req.Identification["org_m_email"])
	if err != nil {
		s.logger.Error("failed to check aws rds cluster limit", zap.Error(err))
		return err
	}
	if !ok {
		err = s.checkPremiumAndSendErr(echoCtx, req.Identification["org_m_email"], "rds cluster")
		if err != nil {
			return err
		}
	}

	var aggregatedInstance *entity.AwsRds
	var aggregatedMetrics map[string][]types.Datapoint
	for _, instance := range req.Instances {
		instance := instance
		rdsRightSizingRecom, err2 := s.recomSvc.AwsRdsRecommendation(ctx, req.Region, instance, req.Metrics[instance.HashedInstanceId], req.Preferences, usageAverageType)
		if err2 != nil {
			s.logger.Error("failed to get aws rds recommendation", zap.Error(err))
			err = err2
			return err
		}
		resp.RightSizing[instance.HashedInstanceId] = *rdsRightSizingRecom
		if aggregatedInstance == nil {
			aggregatedInstance = &instance
		}
		if aggregatedMetrics == nil {
			aggregatedMetrics = req.Metrics[instance.HashedInstanceId]
		} else {
			for key, value := range req.Metrics[instance.HashedInstanceId] {
				switch key {
				case "FreeableMemory", "FreeStorageSpace":
					aggregatedMetrics[key] = recommendation.MergeDatapoints(aggregatedMetrics[key], value, func(aa, bb float64) float64 { return math.Min(aa, bb) })
				default:
					aggregatedMetrics[key] = recommendation.MergeDatapoints(aggregatedMetrics[key], value, func(aa, bb float64) float64 { return math.Max(aa, bb) })
				}
			}
		}
	}
	if aggregatedInstance == nil {
		return echoCtx.JSON(http.StatusBadRequest, "no instances found in the request")
	}
	rdsClusterRightSizingRecom, err := s.recomSvc.AwsRdsRecommendation(ctx, req.Region, *aggregatedInstance, aggregatedMetrics, req.Preferences, usageAverageType)
	if err != nil {
		s.logger.Error("failed to get aws rds recommendation", zap.Error(err))
		return err
	}

	if !strings.Contains(strings.ToLower(req.Cluster.Engine), "aurora") {
		for k, instance := range resp.RightSizing {
			instance := instance
			instance.Recommended = rdsClusterRightSizingRecom.Recommended
			instance.Description = rdsClusterRightSizingRecom.Description
			resp.RightSizing[k] = instance
		}
	} else {
		// TODO Handle aurora storage somehow
	}

	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
	}

	return echoCtx.JSON(http.StatusOK, resp)
}

func (s API) TriggerIngest(echoCtx echo.Context) error {
	ctx := echoCtx.Request().Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(echoCtx.Request().Header))
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	service := echoCtx.Param("service")

	s.logger.Info(fmt.Sprintf("Ingester is going to be triggered for %s", service))

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
	case "gcp-compute-instance":
		err := s.ingestionSvc.DataAgeRepo.Delete("GCPComputeEngine")
		if err != nil {
			s.logger.Error("failed to delete data age", zap.Error(err), zap.String("service", service))
			return err
		}
		s.logger.Info("deleted data age for GCPComputeEngine ingestion will be triggered soon")
	default:
		s.logger.Error(fmt.Sprintf("Service %s not supported", service))
	}

	s.logger.Info(fmt.Sprintf("Ingester triggered for %s", service))

	return echoCtx.NoContent(http.StatusOK)
}

func (s API) MigrateUsages(echoCtx echo.Context) error {
	go func() {
		ctx := context.Background()
		s.logger.Info("Usage table migration started")

		for true {
			usage, err := s.usageV1Repo.GetRandomNotMoved()
			if err != nil {
				s.logger.Error("error while getting usage_v1 usages list", zap.Error(err))
				break
			}
			if usage == nil {
				break
			}
			if usage.Endpoint == "aws-rds" {
				var requestBody entity.AwsRdsWastageRequest
				err = json.Unmarshal(usage.Request, &requestBody)
				if err != nil {
					s.logger.Error("failed to unmarshal request body", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
				requestId := fmt.Sprintf("usage_v1_%v", usage.ID)
				cliVersion := "unknown"
				requestBody.RequestId = &requestId
				requestBody.CliVersion = &cliVersion

				url := "https://api.kaytu.io/kaytu/wastage/api/v1/wastage/aws-rds"

				payload, err := json.Marshal(requestBody)
				if err != nil {
					s.logger.Error("failed to marshal request body", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}

				if _, err := httpclient.DoRequest(ctx, http.MethodPost, url, httpclient.FromEchoContext(echoCtx).ToHeaders(), payload, nil); err != nil {
					s.logger.Error("failed to rerun request", zap.Any("usage_id", usage.ID), zap.Error(err))
				}

				usage.Moved = true
				err = s.usageV1Repo.Update(usage.ID, *usage)
				if err != nil {
					s.logger.Error("failed to update usage moved flag", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
			} else {
				var requestBody entity.EC2InstanceWastageRequest
				err = json.Unmarshal(usage.Request, &requestBody)
				if err != nil {
					s.logger.Error("failed to unmarshal request body", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
				requestId := fmt.Sprintf("usage_v1_%v", usage.ID)
				cliVersion := "unknown"
				requestBody.RequestId = &requestId
				requestBody.CliVersion = &cliVersion

				url := "https://api.kaytu.io/kaytu/wastage/api/v1/wastage/ec2-instance"

				payload, err := json.Marshal(requestBody)
				if err != nil {
					s.logger.Error("failed to marshal request body", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}

				if _, err := httpclient.DoRequest(ctx, http.MethodPost, url, httpclient.FromEchoContext(echoCtx).ToHeaders(), payload, nil); err != nil {
					s.logger.Error("failed to rerun request", zap.Any("usage_id", usage.ID), zap.Error(err))
				}

				usage.Moved = true
				err = s.usageV1Repo.Update(usage.ID, *usage)
				if err != nil {
					s.logger.Error("failed to update usage moved flag", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
			}
		}

	}()

	return echoCtx.NoContent(http.StatusOK)
}

func (s API) MigrateUsagesV2(c echo.Context) error {
	go func() {
		//ctx := context.Background()
		s.logger.Info("Usage table migration started")

		for {
			usage, err := s.usageRepo.GetRandomNullStatistics()
			if err != nil {
				s.logger.Error("error while getting null statistic usages list", zap.Error(err))
				break
			}
			if usage == nil {
				break
			}
			if usage.ApiEndpoint == "aws-rds" {
				var requestBody entity.AwsRdsWastageRequest
				var responseBody entity.AwsRdsWastageResponse
				err = json.Unmarshal(usage.Request, &requestBody)
				if err != nil {
					s.logger.Error("failed to unmarshal request body", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
				stats := model.Statistics{
					AccountID:  requestBody.Identification["account"],
					OrgEmail:   requestBody.Identification["org_m_email"],
					ResourceID: requestBody.Instance.HashedInstanceId,
				}

				err = json.Unmarshal(usage.Response, &responseBody)
				if err == nil {
					recom := entity.RightsizingAwsRds{}
					if responseBody.RightSizing.Recommended != nil {
						recom = *responseBody.RightSizing.Recommended
					}
					stats.CurrentCost = responseBody.RightSizing.Current.Cost
					stats.RecommendedCost = recom.Cost
					stats.Savings = responseBody.RightSizing.Current.Cost - recom.Cost
					stats.RDSInstanceCurrentCost = responseBody.RightSizing.Current.Cost
					stats.RDSInstanceRecommendedCost = recom.Cost
					stats.RDSInstanceSavings = responseBody.RightSizing.Current.Cost - recom.Cost
				}

				out, err := json.Marshal(stats)
				if err != nil {
					s.logger.Error("failed to marshal stats", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
				usage.Statistics = out

				err = s.usageRepo.Update(usage.ID, *usage)
				if err != nil {
					s.logger.Error("failed to update usage moved flag", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
			} else {
				var requestBody entity.EC2InstanceWastageRequest
				var responseBody entity.EC2InstanceWastageResponse
				err = json.Unmarshal(usage.Request, &requestBody)
				if err != nil {
					s.logger.Error("failed to unmarshal request body", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
				stats := model.Statistics{
					AccountID:  requestBody.Identification["account"],
					OrgEmail:   requestBody.Identification["org_m_email"],
					ResourceID: requestBody.Instance.HashedInstanceId,
				}

				err = json.Unmarshal(usage.Response, &responseBody)
				if err == nil {
					recom := entity.RightsizingEC2Instance{}
					if responseBody.RightSizing.Recommended != nil {
						recom = *responseBody.RightSizing.Recommended
					}

					instanceCost := responseBody.RightSizing.Current.Cost
					recomInstanceCost := recom.Cost

					volumeCurrentCost := 0.0
					volumeRecomCost := 0.0
					for _, v := range responseBody.VolumeRightSizing {
						volumeCurrentCost += v.Current.Cost
						if v.Recommended != nil {
							volumeRecomCost += v.Recommended.Cost
						}
					}

					stats.CurrentCost = instanceCost + volumeCurrentCost
					stats.RecommendedCost = recomInstanceCost + volumeRecomCost
					stats.Savings = (instanceCost + volumeCurrentCost) - (recomInstanceCost + volumeRecomCost)
					stats.EC2InstanceCurrentCost = instanceCost
					stats.EC2InstanceRecommendedCost = recomInstanceCost
					stats.EC2InstanceSavings = instanceCost - recomInstanceCost
					stats.EBSCurrentCost = volumeCurrentCost
					stats.EBSRecommendedCost = volumeRecomCost
					stats.EBSSavings = volumeCurrentCost - volumeRecomCost
					stats.EBSVolumeCount = len(responseBody.VolumeRightSizing)
				}

				out, err := json.Marshal(stats)
				if err != nil {
					s.logger.Error("failed to marshal stats", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
				usage.Statistics = out

				err = s.usageRepo.Update(usage.ID, *usage)
				if err != nil {
					s.logger.Error("failed to update usage moved flag", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
			}
		}

	}()

	return c.NoContent(http.StatusOK)
}

func (s API) GetUsageIDByAccountID(echoCtx echo.Context) error {
	accountId := echoCtx.Param("accountID")
	endpoint := echoCtx.Param("endpoint")

	usage, err := s.usageRepo.GetByAccountID(endpoint, accountId)
	if err != nil {
		return err
	}

	return echoCtx.JSON(http.StatusOK, usage)
}

func (s API) GetLastUsageIDByAccountID(echoCtx echo.Context) error {
	accountId := echoCtx.Param("accountID")
	endpoint := echoCtx.Param("endpoint")
	groupByType := echoCtx.Param("groupBy")

	usage, err := s.usageRepo.GetLastByAccountID(endpoint, accountId, groupByType)
	if err != nil {
		return err
	}

	return echoCtx.JSON(http.StatusOK, usage)
}

func (s API) GetUsage(echoCtx echo.Context) error {
	idStr := echoCtx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}

	usage, err := s.usageRepo.Get(uint(id))
	if err != nil {
		return err
	}

	return echoCtx.JSON(http.StatusOK, usage)
}

func (s API) FillRdsCosts(echoCtx echo.Context) error {
	go func() {
		//ctx := context.Background()
		s.logger.Info("Filling RDS costs started")

		for {
			usage, err := s.usageRepo.GetCostZero()
			if err != nil {
				s.logger.Error("error while getting null statistic usages list", zap.Error(err))
				break
			}
			if usage == nil {
				break
			}
			if usage.ApiEndpoint == "aws-rds" {
				var responseBody entity.AwsRdsWastageResponse
				err = json.Unmarshal(usage.Response, &responseBody)
				if err == nil {
					responseBody.RightSizing.Current.Cost = responseBody.RightSizing.Current.ComputeCost + responseBody.RightSizing.Current.StorageCost
					responseBody.RightSizing.Recommended.Cost = responseBody.RightSizing.Recommended.ComputeCost + responseBody.RightSizing.Recommended.StorageCost
				}

				out, err := json.Marshal(responseBody)
				if err != nil {
					s.logger.Error("failed to marshal stats", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
				usage.Response = out

				err = s.usageRepo.Update(usage.ID, *usage)
				if err != nil {
					s.logger.Error("failed to update usage moved flag", zap.Any("usage_id", usage.ID), zap.Error(err))
					continue
				}
			}
		}

	}()

	return echoCtx.NoContent(http.StatusOK)
}

func (s API) checkPremiumAndSendErr(echoCtx echo.Context, orgEmail string, service string) error {
	user, err := s.userRepo.Get(httpserver.GetUserID(echoCtx))
	if err != nil {
		s.logger.Error("failed to get user", zap.Error(err))
		return err
	}
	if user != nil && user.PremiumUntil != nil {
		if time.Now().Before(*user.PremiumUntil) {
			return nil
		}
	}

	if orgEmail != "" && strings.Contains(orgEmail, "@") {
		org := strings.Split(orgEmail, "@")
		if org[1] != "" {
			orgName := strings.Split(orgEmail, "@")
			org, err := s.orgRepo.Get(orgName[1])
			if err != nil {
				s.logger.Error("failed to get organization", zap.Error(err))
				return err
			}
			if org != nil && org.PremiumUntil != nil {
				if time.Now().Before(*org.PremiumUntil) {
					return nil
				}
			}
		}
	}

	err = fmt.Errorf("reached the %s limit for both user and organization", service)
	s.logger.Error(err.Error(), zap.String("auth0UserId", httpserver.GetUserID(echoCtx)), zap.String("orgEmail", orgEmail))
	return nil
}

func (s API) CreateUser(echoCtx echo.Context) error {
	var user entity.User
	err := echoCtx.Bind(&user)
	if err != nil {
		return err
	}

	err = s.userRepo.Create(user.ToModel())
	if err != nil {
		return err
	}

	return echoCtx.JSON(http.StatusCreated, user)
}

func (s API) UpdateUser(echoCtx echo.Context) error {
	idString := echoCtx.Param("userId")
	if idString == "" {
		return errors.New("userId is required")
	}

	premiumUntil, err := strconv.ParseInt(echoCtx.QueryParam("premiumUntil"), 10, 64)
	if err != nil {
		return err
	}

	premiumUntilTime := time.UnixMilli(premiumUntil)
	user := model.User{
		UserId:       idString,
		PremiumUntil: &premiumUntilTime,
	}
	err = s.userRepo.Update(idString, &user)
	if err != nil {
		return err
	}
	return echoCtx.JSON(http.StatusOK, user)
}

func (s API) CreateOrganization(echoCtx echo.Context) error {
	var org entity.Organization
	err := echoCtx.Bind(&org)
	if err != nil {
		return err
	}

	err = s.orgRepo.Create(org.ToModel())
	if err != nil {
		return err
	}

	return echoCtx.JSON(http.StatusCreated, org)
}

func (s API) UpdateOrganization(echoCtx echo.Context) error {
	idString := echoCtx.Param("organizationId")
	if idString == "" {
		return errors.New("organizationId is required")
	}

	premiumUntil, err := strconv.ParseInt(echoCtx.QueryParam("premiumUntil"), 10, 64)
	if err != nil {
		return err
	}

	premiumUntilTime := time.UnixMilli(premiumUntil)
	org := model.Organization{
		OrganizationId: idString,
		PremiumUntil:   &premiumUntilTime,
	}
	err = s.orgRepo.Update(idString, &org)
	if err != nil {
		return err
	}
	return echoCtx.JSON(http.StatusOK, org)
}
