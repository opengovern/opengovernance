package grpc_server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/alitto/pond"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"github.com/kaytu-io/open-governance/pkg/utils"
	"github.com/kaytu-io/open-governance/services/wastage/config"
	"github.com/kaytu-io/open-governance/services/wastage/db/model"
	"github.com/kaytu-io/open-governance/services/wastage/db/repo"
	"github.com/kaytu-io/open-governance/services/wastage/recommendation"
	gcp "github.com/kaytu-io/plugin-gcp/plugin/proto/src/golang/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"time"
)

type gcpPluginServer struct {
	gcp.OptimizationServer

	cfg config.WastageConfig

	tracer trace.Tracer
	logger *zap.Logger

	blobClient     *azblob.Client
	blobWorkerPool *pond.WorkerPool

	usageRepo repo.UsageV2Repo
	recomSvc  *recommendation.Service
}

func newGcpPluginServer(logger *zap.Logger, cfg config.WastageConfig, blobClient *azblob.Client, blobWorkerPool *pond.WorkerPool, usageRepo repo.UsageV2Repo, recomSvc *recommendation.Service) *gcpPluginServer {
	return &gcpPluginServer{
		cfg:            cfg,
		tracer:         otel.GetTracerProvider().Tracer("wastage.http.sources"),
		logger:         logger.Named("grpc"),
		blobClient:     blobClient,
		blobWorkerPool: blobWorkerPool,
		usageRepo:      usageRepo,
		recomSvc:       recomSvc,
	}
}

func (s *gcpPluginServer) GCPComputeOptimization(ctx context.Context, req *gcp.GCPComputeOptimizationRequest) (*gcp.GCPComputeOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp gcp.GCPComputeOptimizationResponse
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
		AccountID:   "",
		OrgEmail:    "",
		ResourceID:  req.Instance.Id,
		Auth0UserId: userId,
	}
	statsOut, _ := json.Marshal(stats)

	fullReqJson, _ := json.Marshal(req)
	metrics := req.Metrics
	diskMetrics := req.DisksMetrics
	req.Metrics = nil
	req.DisksMetrics = nil
	trimmedReqJson, _ := json.Marshal(req)
	req.Metrics = metrics
	req.DisksMetrics = diskMetrics

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
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("gcp-compute-instance/%s.json", *requestId), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})

	usage := model.UsageV2{
		ApiEndpoint:    "gcp-compute-instance",
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

			recom := gcp.RightsizingGcpComputeInstance{}
			if resp.Rightsizing != nil && resp.Rightsizing.Recommended != nil {
				recom = *resp.Rightsizing.Recommended
			}
			stats.CurrentCost = resp.Rightsizing.Current.Cost
			stats.RecommendedCost = recom.Cost
			stats.Savings = resp.Rightsizing.Current.Cost - recom.Cost
			stats.GCPComputeInstanceCurrentCost = resp.Rightsizing.Current.Cost
			stats.GCPComputeInstanceRecommendedCost = recom.Cost
			stats.GCPComputeInstanceSavings = resp.Rightsizing.Current.Cost - recom.Cost

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

	rightSizingRecom, currentMachine, recomMachine, err := s.recomSvc.GCPComputeInstanceRecommendation(ctx, *req.Instance, req.Metrics, req.Preferences)
	if err != nil {
		s.logger.Error("failed to get gcp compute instance recommendation", zap.Error(err))
		return nil, err
	}

	diskRightSizingRecoms := make(map[string]*gcp.GcpComputeDiskRecommendation)
	for _, disk := range req.Disks {
		var diskRightSizingRecom *gcp.GcpComputeDiskRecommendation
		diskRightSizingRecom, err = s.recomSvc.GCPComputeDiskRecommendation(ctx, *disk, currentMachine, recomMachine, *req.DisksMetrics[disk.Id], req.Preferences)
		if err != nil {
			err = fmt.Errorf("failed to get GCP Compute Disk %s recommendation: %s", disk.Id, err.Error())
			return nil, err
		}
		diskRightSizingRecoms[disk.Id] = diskRightSizingRecom
	}

	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
	}

	// DO NOT change this, resp is used in updating usage
	resp = gcp.GCPComputeOptimizationResponse{
		Rightsizing:        rightSizingRecom,
		VolumesRightsizing: diskRightSizingRecoms,
	}
	// DO NOT change this, resp is used in updating usage

	return &resp, nil
}
