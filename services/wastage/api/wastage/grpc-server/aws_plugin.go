package grpc_server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/alitto/pond"
	types2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/wastage/limit"
	"github.com/kaytu-io/kaytu-engine/services/wastage/config"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	aws "github.com/kaytu-io/plugin-aws/plugin/proto/src/golang"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/mod/semver"
	"google.golang.org/grpc/metadata"
	"math"
	"net/http"
	"strings"
	"time"
)

type awsPluginServer struct {
	aws.OptimizationServer

	cfg config.WastageConfig

	tracer trace.Tracer
	logger *zap.Logger

	blobClient     *azblob.Client
	blobWorkerPool *pond.WorkerPool

	usageRepo repo.UsageV2Repo
	recomSvc  *recommendation.Service

	limitService *limit.Service
}

func newAwsPluginServer(logger *zap.Logger, cfg config.WastageConfig, blobClient *azblob.Client, blobWorkerPool *pond.WorkerPool,
	usageRepo repo.UsageV2Repo, recomSvc *recommendation.Service, limitService *limit.Service) *awsPluginServer {

	return &awsPluginServer{
		cfg:            cfg,
		tracer:         otel.GetTracerProvider().Tracer("wastage.http.sources"),
		logger:         logger.Named("grpc"),
		blobClient:     blobClient,
		blobWorkerPool: blobWorkerPool,
		usageRepo:      usageRepo,
		recomSvc:       recomSvc,
		limitService:   limitService,
	}
}

func (s *awsPluginServer) EC2InstanceOptimization(ctx context.Context, req *aws.EC2InstanceOptimizationRequest) (*aws.EC2InstanceOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp aws.EC2InstanceOptimizationResponse
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get incoming context")
	}

	userIds := md.Get(httpserver.XKaytuUserIDHeader)
	userId := ""
	if len(userIds) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	userId = userIds[0]

	stats := model.Statistics{
		AccountID:   req.Identification["account"],
		OrgEmail:    req.Identification["org_m_email"],
		ResourceID:  req.Instance.HashedInstanceId,
		Auth0UserId: userId,
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

	var requestId *string
	var cliVersion *string
	if req.RequestId != nil {
		requestId = &req.RequestId.Value
	}
	if req.CliVersion != nil {
		cliVersion = &req.CliVersion.Value
	}

	if requestId == nil {
		id := uuid.New().String()
		requestId = &id
	}

	s.blobWorkerPool.Submit(func() {
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("ec2-instance/%s.json", req.RequestId.String()), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})

	usage := model.UsageV2{
		ApiEndpoint:    "ec2-instance",
		Request:        trimmedReqJson,
		RequestId:      requestId,
		CliVersion:     cliVersion,
		Response:       nil,
		FailureMessage: nil,
		Statistics:     statsOut,
	}
	err = s.usageRepo.Create(&usage)
	if err != nil {
		return nil, err
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

			recom := aws.RightsizingEC2Instance{}
			if resp.RightSizing != nil && resp.RightSizing.Recommended != nil {
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

	if req.Instance.State != string(types2.InstanceStateNameRunning) {
		err = echo.NewHTTPError(http.StatusBadRequest, "instance is not running")
		return nil, err
	}

	if req.Loading {
		return nil, nil
	}

	usageAverageType := recommendation.UsageAverageTypeMax
	if req.CliVersion == nil || semver.Compare("v"+req.CliVersion.GetValue(), "v0.5.2") < 0 {
		return nil, fmt.Errorf("plugin version is no longer supported - please update to the latest version")
	}

	//ok, err = s.limitService.CheckAccountsLimit(ctx, userId, req.Identification["org_m_email"], req.Identification["account"])
	//if err != nil {
	//	s.logger.Error("failed to check profile limit", zap.Error(err))
	//	return nil, err
	//}
	//if !ok {
	//err = s.limitService.CheckPremiumAndSendErr(ctx, userId, req.Identification["org_m_email"], "profile")
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	//
	//ok, err = s.limitService.CheckEC2InstanceLimit(ctx, userId, req.Identification["org_m_email"])
	//if err != nil {
	//	s.logger.Error("failed to check aws ec2 instance limit", zap.Error(err))
	//	return nil, err
	//}
	//if !ok {
	//	err = s.limitService.CheckPremiumAndSendErr(ctx, userId, req.Identification["org_m_email"], "ec2 instance")
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	ec2RightSizingRecom, err := s.recomSvc.EC2InstanceRecommendationGrpc(ctx, req.Region, req.Instance, req.Volumes, req.Metrics, req.VolumeMetrics, req.Preferences, usageAverageType)
	if err != nil {
		err = fmt.Errorf("failed to get ec2 instance recommendation: %s", err.Error())
		return nil, err
	}

	ebsRightSizingRecoms := make(map[string]*aws.EBSVolumeRecommendation)
	for _, vol := range req.Volumes {
		var ebsRightSizingRecom *aws.EBSVolumeRecommendation
		ebsRightSizingRecom, err = s.recomSvc.EBSVolumeRecommendationGrpc(ctx, req.Region, vol, req.VolumeMetrics[vol.HashedVolumeId], req.Preferences, usageAverageType)
		if err != nil {
			err = fmt.Errorf("failed to get ebs volume %s recommendation: %s", vol.HashedVolumeId, err.Error())
			return nil, err
		}
		ebsRightSizingRecoms[vol.HashedVolumeId] = ebsRightSizingRecom
	}
	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
	}

	// DO NOT change this, resp is used in updating usage
	resp = aws.EC2InstanceOptimizationResponse{
		RightSizing:       ec2RightSizingRecom,
		VolumeRightSizing: ebsRightSizingRecoms,
	}
	// DO NOT change this, resp is used in updating usage
	return &resp, nil
}

func (s *awsPluginServer) RDSInstanceOptimization(ctx context.Context, req *aws.RDSInstanceOptimizationRequest) (*aws.RDSInstanceOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get incoming context")
	}

	userIds := md.Get(httpserver.XKaytuUserIDHeader)
	userId := ""
	if len(userIds) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	userId = userIds[0]

	var resp aws.RDSInstanceOptimizationResponse
	var err error

	stats := model.Statistics{
		AccountID:   req.Identification["account"],
		OrgEmail:    req.Identification["org_m_email"],
		ResourceID:  req.Instance.HashedInstanceId,
		Auth0UserId: userId,
	}
	statsOut, _ := json.Marshal(stats)

	fullReqJson, _ := json.Marshal(req)
	metrics := req.Metrics
	req.Metrics = nil
	trimmedReqJson, _ := json.Marshal(req)
	req.Metrics = metrics

	var requestId *string
	var cliVersion *string
	if req.RequestId != nil {
		requestId = &req.RequestId.Value
	}
	if req.CliVersion != nil {
		cliVersion = &req.CliVersion.Value
	}

	if requestId == nil {
		id := uuid.New().String()
		requestId = &id
	}

	s.blobWorkerPool.Submit(func() {
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("aws-rds/%s.json", req.RequestId.String()), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})
	usage := model.UsageV2{
		ApiEndpoint:    "aws-rds",
		Request:        trimmedReqJson,
		RequestId:      requestId,
		CliVersion:     cliVersion,
		Response:       nil,
		FailureMessage: nil,
		Statistics:     statsOut,
	}
	err = s.usageRepo.Create(&usage)
	if err != nil {
		s.logger.Error("failed to create usage", zap.Error(err))
		return nil, err
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

			recom := aws.RightsizingAwsRds{}
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
		return nil, nil
	}

	usageAverageType := recommendation.UsageAverageTypeMax
	if req.CliVersion == nil || semver.Compare("v"+req.CliVersion.GetValue(), "v0.5.2") < 0 {
		return nil, fmt.Errorf("plugin version is no longer supported - please update to the latest version")
	}

	ok, err = s.limitService.CheckAccountsLimit(ctx, userId, req.Identification["org_m_email"], req.Identification["account"])
	if err != nil {
		s.logger.Error("failed to check profile limit", zap.Error(err))
		return nil, err
	}
	if !ok {
		err = s.limitService.CheckPremiumAndSendErr(ctx, userId, req.Identification["org_m_email"], "profile")
		if err != nil {
			return nil, err
		}
	}

	ok, err = s.limitService.CheckRDSInstanceLimit(ctx, userId, req.Identification["org_m_email"])
	if err != nil {
		s.logger.Error("failed to check aws rds instance limit", zap.Error(err))
		return nil, err
	}
	if !ok {
		err = s.limitService.CheckPremiumAndSendErr(ctx, userId, req.Identification["org_m_email"], "rds instance")
		if err != nil {
			return nil, err
		}
	}

	rdsRightSizingRecom, err := s.recomSvc.AwsRdsRecommendationGrpc(ctx, req.Region, req.Instance, req.Metrics, req.Preferences, usageAverageType)
	if err != nil {
		s.logger.Error("failed to get aws rds recommendation", zap.Error(err))
		return nil, err
	}

	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
	}

	// DO NOT change this, resp is used in updating usage
	resp = aws.RDSInstanceOptimizationResponse{
		RightSizing: rdsRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage
	return &resp, nil
}

func (s *awsPluginServer) RDSClusterOptimization(ctx context.Context, req *aws.RDSClusterOptimizationRequest) (*aws.RDSClusterOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp aws.RDSClusterOptimizationResponse
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get incoming context")
	}

	userIds := md.Get(httpserver.XKaytuUserIDHeader)
	userId := ""
	if len(userIds) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	userId = userIds[0]

	stats := model.Statistics{
		AccountID:   req.Identification["account"],
		OrgEmail:    req.Identification["org_m_email"],
		ResourceID:  req.Cluster.HashedClusterId,
		Auth0UserId: userId,
	}
	statsOut, _ := json.Marshal(stats)

	fullReqJson, _ := json.Marshal(req)
	metrics := req.Metrics
	req.Metrics = nil
	trimmedReqJson, _ := json.Marshal(req)
	req.Metrics = metrics

	var requestId *string
	var cliVersion *string
	if req.RequestId != nil {
		requestId = &req.RequestId.Value
	}
	if req.CliVersion != nil {
		cliVersion = &req.CliVersion.Value
	}

	if requestId == nil {
		id := uuid.New().String()
		requestId = &id
	}

	s.blobWorkerPool.Submit(func() {
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("aws-rds-cluster/%s.json", req.RequestId.GetValue()), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})
	usage := model.UsageV2{
		ApiEndpoint:    "aws-rds-cluster",
		Request:        trimmedReqJson,
		RequestId:      requestId,
		CliVersion:     cliVersion,
		Response:       nil,
		FailureMessage: nil,
		Statistics:     statsOut,
	}
	err = s.usageRepo.Create(&usage)
	if err != nil {
		s.logger.Error("failed to create usage", zap.Error(err))
		return nil, err
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
				recom.ClusterType = entity.AwsRdsClusterType(instance.Recommended.ClusterType)
				recom.VCPU += instance.Recommended.Vcpu
				recom.MemoryGb += instance.Recommended.MemoryGb
				recom.StorageType = recommendation.WrappedToString(instance.Recommended.StorageType)
				recom.StorageSize = recommendation.WrappedToInt32(instance.Recommended.StorageSize)
				recom.StorageIops = recommendation.WrappedToInt32(instance.Recommended.StorageIops)
				recom.StorageThroughput = recommendation.WrappedToFloat64(instance.Recommended.StorageThroughput)

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
		return nil, nil
	}

	usageAverageType := recommendation.UsageAverageTypeMax
	if req.CliVersion == nil || semver.Compare("v"+req.CliVersion.GetValue(), "v0.5.2") < 0 {
		return nil, fmt.Errorf("plugin version is no longer supported - please update to the latest version")
	}

	resp = aws.RDSClusterOptimizationResponse{
		RightSizing: make(map[string]*aws.RDSInstanceRightSizingRecommendation),
	}

	ok, err = s.limitService.CheckAccountsLimit(ctx, userId, req.Identification["org_m_email"], req.Identification["account"])
	if err != nil {
		s.logger.Error("failed to check profile limit", zap.Error(err))
		return nil, err
	}
	if !ok {
		err = s.limitService.CheckPremiumAndSendErr(ctx, userId, req.Identification["org_m_email"], "profile")
		if err != nil {
			return nil, err
		}
	}

	ok, err = s.limitService.CheckRDSClusterLimit(ctx, userId, req.Identification["org_m_email"])
	if err != nil {
		s.logger.Error("failed to check aws rds cluster limit", zap.Error(err))
		return nil, err
	}
	if !ok {
		err = s.limitService.CheckPremiumAndSendErr(ctx, userId, req.Identification["org_m_email"], "rds cluster")
		if err != nil {
			return nil, err
		}
	}

	var aggregatedInstance *aws.RDSInstance
	var aggregatedMetrics map[string]*aws.Metric
	for _, instance := range req.Instances {
		instance := instance
		rdsRightSizingRecom, err2 := s.recomSvc.AwsRdsRecommendationGrpc(ctx, req.Region, instance, req.Metrics[instance.HashedInstanceId].Metrics, req.Preferences, usageAverageType)
		if err2 != nil {
			s.logger.Error("failed to get aws rds recommendation", zap.Error(err))
			err = err2
			return nil, err
		}
		resp.RightSizing[instance.HashedInstanceId] = rdsRightSizingRecom
		if aggregatedInstance == nil {
			aggregatedInstance = instance
		}
		if aggregatedMetrics == nil {
			aggregatedMetrics = req.Metrics[instance.HashedInstanceId].Metrics
		} else {
			for key, value := range req.Metrics[instance.HashedInstanceId].Metrics {
				switch key {
				case "FreeableMemory", "FreeStorageSpace":
					aggregatedMetrics[key] = &aws.Metric{
						Metric: recommendation.MergeGrpcDatapoints(aggregatedMetrics[key].Metric, value.Metric, func(aa, bb float64) float64 { return math.Min(aa, bb) }),
					}
				default:
					aggregatedMetrics[key] = &aws.Metric{
						Metric: recommendation.MergeGrpcDatapoints(aggregatedMetrics[key].Metric, value.Metric, func(aa, bb float64) float64 { return math.Max(aa, bb) }),
					}
				}
			}
		}
	}
	if aggregatedInstance == nil {
		return nil, fmt.Errorf("no instances found in the request")
	}
	rdsClusterRightSizingRecom, err := s.recomSvc.AwsRdsRecommendationGrpc(ctx, req.Region, aggregatedInstance, aggregatedMetrics, req.Preferences, usageAverageType)
	if err != nil {
		s.logger.Error("failed to get aws rds recommendation", zap.Error(err))
		return nil, err
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

	return &resp, nil
}
