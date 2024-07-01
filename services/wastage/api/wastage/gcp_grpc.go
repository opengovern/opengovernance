package wastage

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/alitto/pond"
	envoyAuth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/wastage/config"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	kaytuGrpc "github.com/kaytu-io/kaytu-util/pkg/grpc"
	gcp "github.com/kaytu-io/plugin-gcp/plugin/proto/src/golang"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"net"
	"time"
)

type GcpServer struct {
	gcp.OptimizationServer

	cfg config.WastageConfig

	tracer trace.Tracer
	logger *zap.Logger

	blobClient     *azblob.Client
	blobWorkerPool *pond.WorkerPool

	usageRepo repo.UsageV2Repo
	recomSvc  *recommendation.Service
}

func NewGcpServer(logger *zap.Logger, cfg config.WastageConfig, blobClient *azblob.Client, blobWorkerPool *pond.WorkerPool, usageRepo repo.UsageV2Repo, recomSvc *recommendation.Service) *GcpServer {
	return &GcpServer{
		cfg:            cfg,
		tracer:         otel.GetTracerProvider().Tracer("wastage.http.sources"),
		logger:         logger.Named("grpc"),
		blobClient:     blobClient,
		blobWorkerPool: blobWorkerPool,
		usageRepo:      usageRepo,
		recomSvc:       recomSvc,
	}
}

func StartGcpGrpcServer(server *GcpServer, grpcServerAddress string, authGRPCURI string) error {
	lis, err := net.Listen("tcp", grpcServerAddress)
	if err != nil {
		server.logger.Error("failed to listen", zap.Error(err))
		return err
	}
	authGRPCConn, err := grpc.NewClient(authGRPCURI, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	if err != nil {
		server.logger.Error("failed to dial", zap.Error(err))
		return err
	}
	authGrpcClient := envoyAuth.NewAuthorizationClient(authGRPCConn)

	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(256*1024*1024),
		grpc.UnaryInterceptor(kaytuGrpc.CheckGRPCAuthUnaryInterceptorWrapper(authGrpcClient)),
		grpc.ChainUnaryInterceptor(Logger(server.logger)),
	)
	gcp.RegisterOptimizationServer(s, server)
	server.logger.Info("server listening at", zap.String("address", lis.Addr().String()))
	utils.EnsureRunGoroutine(func() {
		if err = s.Serve(lis); err != nil {
			server.logger.Error("failed to serve", zap.Error(err))
			panic(err)
		}
	})
	return nil
}

func (s *GcpServer) GCPComputeOptimization(ctx context.Context, req *gcp.GCPComputeOptimizationRequest) (*gcp.GCPComputeOptimizationResponse, error) {
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
	req.Metrics = nil
	disksMetrics := req.DisksMetrics
	req.DisksMetrics = nil
	trimmedReqJson, _ := json.Marshal(req)
	req.Metrics = metrics
	req.DisksMetrics = disksMetrics
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
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("kubernetes-pod/%s.json", *requestId), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
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
			if resp.Rightsizing.Recommended != nil {
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
