package grpc_server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/alitto/pond"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/wastage/config"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	kubernetesPluginProto "github.com/kaytu-io/plugin-kubernetes-internal/plugin/proto/src/golang"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"strings"
	"time"
)

type kubernetesPluginServer struct {
	kubernetesPluginProto.OptimizationServer

	cfg config.WastageConfig

	tracer trace.Tracer
	logger *zap.Logger

	blobClient     *azblob.Client
	blobWorkerPool *pond.WorkerPool

	usageRepo repo.UsageV2Repo
	recomSvc  *recommendation.Service
}

func newKubernetesPluginServer(logger *zap.Logger, cfg config.WastageConfig, blobClient *azblob.Client, blobWorkerPool *pond.WorkerPool, usageRepo repo.UsageV2Repo, recomSvc *recommendation.Service) *kubernetesPluginServer {
	return &kubernetesPluginServer{
		cfg:            cfg,
		blobClient:     blobClient,
		blobWorkerPool: blobWorkerPool,
		usageRepo:      usageRepo,
		recomSvc:       recomSvc,
		tracer:         otel.GetTracerProvider().Tracer("wastage.grpc.kubernetes"),
		logger:         logger.Named("kubernetes-grpc-server"),
	}
}

func (s *kubernetesPluginServer) KubernetesPodOptimization(ctx context.Context, req *kubernetesPluginProto.KubernetesPodOptimizationRequest) (*kubernetesPluginProto.KubernetesPodOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp kubernetesPluginProto.KubernetesPodOptimizationResponse
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

	email := req.Identification["cluster_name"]
	if !strings.Contains(email, "@") {
		email = email + "@local.temp"
	}

	accountId := req.Identification["auth_info_name"]
	if accountId == "" {
		accountId = req.Identification["cluster_server"]
	}
	stats := model.Statistics{
		AccountID:   accountId,
		OrgEmail:    email,
		ResourceID:  req.Pod.Id,
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
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("kubernetes-pod/%s.json", *requestId), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})

	usage := model.UsageV2{
		ApiEndpoint:    "kubernetes-pod",
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

			if req != nil && req.Pod != nil {
				for _, container := range req.Pod.Containers {
					if container == nil {
						continue
					}
					stats.KubernetesCurrentCPURequest += container.CpuRequest
					stats.KubernetesCurrentMemoryRequest += container.MemoryRequest
				}
			}
			if resp.Rightsizing != nil {
				for _, container := range resp.Rightsizing.ContainerResizing {
					if container != nil && container.Current != nil && container.Recommended != nil {
						stats.KubernetesRecommendedCPURequest += container.Recommended.CpuRequest
						stats.KubernetesRecommendedMemoryRequest += container.Recommended.MemoryRequest

						stats.KubernetesCPURequestSavings += container.Current.CpuRequest - container.Recommended.CpuRequest
						stats.KubernetesMemoryRequestSavings += container.Current.MemoryRequest - container.Recommended.MemoryRequest
					}
				}
			}

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

	podRightSizingRecom, err := s.recomSvc.KubernetesPodRecommendation(*req.Pod, req.Metrics, req.Preferences)
	if err != nil {
		s.logger.Error("failed to get kubernetes pod recommendation", zap.Error(err))
		return nil, err
	}

	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		return nil, err
	}

	// DO NOT change this, resp is used in updating usage
	resp = kubernetesPluginProto.KubernetesPodOptimizationResponse{
		Rightsizing: podRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage

	return &resp, nil
}

func (s *kubernetesPluginServer) KubernetesDeploymentOptimization(ctx context.Context, req *kubernetesPluginProto.KubernetesDeploymentOptimizationRequest) (*kubernetesPluginProto.KubernetesDeploymentOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp kubernetesPluginProto.KubernetesDeploymentOptimizationResponse
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get incoming context")
	}

	userIds := md.Get(httpserver.XKaytuUserIDHeader)
	userId := ""
	if len(userIds) > 0 {
		userId = userIds[0]
	}

	email := req.Identification["cluster_name"]
	if !strings.Contains(email, "@") {
		email = email + "@local.temp"
	}

	accountId := req.Identification["auth_info_name"]
	if accountId == "" {
		accountId = req.Identification["cluster_server"]
	}
	stats := model.Statistics{
		AccountID:   accountId,
		OrgEmail:    email,
		ResourceID:  req.GetDeployment().GetId(),
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
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("kubernetes-deployment/%s.json", *requestId), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})

	usage := model.UsageV2{
		ApiEndpoint:    "kubernetes-deployment",
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

			if req != nil && req.Deployment != nil {
				for _, container := range req.Deployment.Containers {
					if container == nil {
						continue
					}
					stats.KubernetesCurrentCPURequest += container.CpuRequest
					stats.KubernetesCurrentMemoryRequest += container.MemoryRequest
				}
			}
			if resp.Rightsizing != nil {
				for _, container := range resp.Rightsizing.ContainerResizing {
					if container != nil && container.Current != nil && container.Recommended != nil {
						stats.KubernetesRecommendedCPURequest += container.Recommended.CpuRequest
						stats.KubernetesRecommendedMemoryRequest += container.Recommended.MemoryRequest

						stats.KubernetesCPURequestSavings += container.Current.CpuRequest - container.Recommended.CpuRequest
						stats.KubernetesMemoryRequestSavings += container.Current.MemoryRequest - container.Recommended.MemoryRequest
					}
				}
			}

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

	deploymentRightSizingRecom, err := s.recomSvc.KubernetesDeploymentRecommendation(*req.Deployment, req.Metrics, req.Preferences)
	if err != nil {
		s.logger.Error("failed to get kubernetes deployment recommendation", zap.Error(err))
		return nil, err
	}

	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		return nil, err
	}

	// DO NOT change this, resp is used in updating usage
	resp = kubernetesPluginProto.KubernetesDeploymentOptimizationResponse{
		Rightsizing: deploymentRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage

	return &resp, nil
}

func (s *kubernetesPluginServer) KubernetesStatefulsetOptimization(ctx context.Context, req *kubernetesPluginProto.KubernetesStatefulsetOptimizationRequest) (*kubernetesPluginProto.KubernetesStatefulsetOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp kubernetesPluginProto.KubernetesStatefulsetOptimizationResponse
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get incoming context")
	}

	userIds := md.Get(httpserver.XKaytuUserIDHeader)
	userId := ""
	if len(userIds) > 0 {
		userId = userIds[0]
	}

	email := req.Identification["cluster_name"]
	if !strings.Contains(email, "@") {
		email = email + "@local.temp"
	}

	accountId := req.Identification["auth_info_name"]
	if accountId == "" {
		accountId = req.Identification["cluster_server"]
	}
	stats := model.Statistics{
		AccountID:   accountId,
		OrgEmail:    email,
		ResourceID:  req.GetStatefulset().GetId(),
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
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("kubernetes-statefulset/%s.json", *requestId), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})

	usage := model.UsageV2{
		ApiEndpoint:    "kubernetes-statefulset",
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

			if req != nil && req.Statefulset != nil {
				for _, container := range req.Statefulset.Containers {
					if container == nil {
						continue
					}
					stats.KubernetesCurrentCPURequest += container.CpuRequest
					stats.KubernetesCurrentMemoryRequest += container.MemoryRequest
				}
			}
			if resp.Rightsizing != nil {
				for _, container := range resp.Rightsizing.ContainerResizing {
					if container != nil && container.Current != nil && container.Recommended != nil {
						stats.KubernetesRecommendedCPURequest += container.Recommended.CpuRequest
						stats.KubernetesRecommendedMemoryRequest += container.Recommended.MemoryRequest

						stats.KubernetesCPURequestSavings += container.Current.CpuRequest - container.Recommended.CpuRequest
						stats.KubernetesMemoryRequestSavings += container.Current.MemoryRequest - container.Recommended.MemoryRequest
					}
				}
			}

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

	statefulsetRightSizingRecom, err := s.recomSvc.KubernetesStatefulsetRecommendation(*req.Statefulset, req.Metrics, req.Preferences)
	if err != nil {
		s.logger.Error("failed to get kubernetes statefulset recommendation", zap.Error(err))
		return nil, err
	}

	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		return nil, err
	}

	// DO NOT change this, resp is used in updating usage
	resp = kubernetesPluginProto.KubernetesStatefulsetOptimizationResponse{
		Rightsizing: statefulsetRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage

	return &resp, nil
}

func (s *kubernetesPluginServer) KubernetesDaemonsetOptimization(ctx context.Context, req *kubernetesPluginProto.KubernetesDaemonsetOptimizationRequest) (*kubernetesPluginProto.KubernetesDaemonsetOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp kubernetesPluginProto.KubernetesDaemonsetOptimizationResponse
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get incoming context")
	}

	userIds := md.Get(httpserver.XKaytuUserIDHeader)
	userId := ""
	if len(userIds) > 0 {
		userId = userIds[0]
	}

	email := req.Identification["cluster_name"]
	if !strings.Contains(email, "@") {
		email = email + "@local.temp"
	}

	accountId := req.Identification["auth_info_name"]
	if accountId == "" {
		accountId = req.Identification["cluster_server"]
	}
	stats := model.Statistics{
		AccountID:   accountId,
		OrgEmail:    email,
		ResourceID:  req.GetDaemonset().GetId(),
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
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("kubernetes-daemonset/%s.json", *requestId), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})

	usage := model.UsageV2{
		ApiEndpoint:    "kubernetes-daemonset",
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

			if req != nil && req.Daemonset != nil {
				for _, container := range req.Daemonset.Containers {
					if container == nil {
						continue
					}
					stats.KubernetesCurrentCPURequest += container.CpuRequest
					stats.KubernetesCurrentMemoryRequest += container.MemoryRequest
				}
			}
			if resp.Rightsizing != nil {
				for _, container := range resp.Rightsizing.ContainerResizing {
					if container != nil && container.Current != nil && container.Recommended != nil {
						stats.KubernetesRecommendedCPURequest += container.Recommended.CpuRequest
						stats.KubernetesRecommendedMemoryRequest += container.Recommended.MemoryRequest

						stats.KubernetesCPURequestSavings += container.Current.CpuRequest - container.Recommended.CpuRequest
						stats.KubernetesMemoryRequestSavings += container.Current.MemoryRequest - container.Recommended.MemoryRequest
					}
				}
			}

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

	daemonsetRightSizingRecom, err := s.recomSvc.KubernetesDaemonsetRecommendation(*req.Daemonset, req.Metrics, req.Preferences)
	if err != nil {
		s.logger.Error("failed to get kubernetes daemonset recommendation", zap.Error(err))
		return nil, err
	}

	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		return nil, err
	}

	// DO NOT change this, resp is used in updating usage
	resp = kubernetesPluginProto.KubernetesDaemonsetOptimizationResponse{
		Rightsizing: daemonsetRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage

	return &resp, nil
}

func (s *kubernetesPluginServer) KubernetesJobOptimization(ctx context.Context, req *kubernetesPluginProto.KubernetesJobOptimizationRequest) (*kubernetesPluginProto.KubernetesJobOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp kubernetesPluginProto.KubernetesJobOptimizationResponse
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get incoming context")
	}

	userIds := md.Get(httpserver.XKaytuUserIDHeader)
	userId := ""
	if len(userIds) > 0 {
		userId = userIds[0]
	}

	email := req.Identification["cluster_name"]
	if !strings.Contains(email, "@") {
		email = email + "@local.temp"
	}

	accountId := req.Identification["auth_info_name"]
	if accountId == "" {
		accountId = req.Identification["cluster_server"]
	}
	stats := model.Statistics{
		AccountID:   accountId,
		OrgEmail:    email,
		ResourceID:  req.GetJob().GetId(),
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
		_, err = s.blobClient.UploadBuffer(context.Background(), s.cfg.AzBlob.Container, fmt.Sprintf("kubernetes-job/%s.json", *requestId), fullReqJson, &azblob.UploadBufferOptions{AccessTier: utils.GetPointer(blob.AccessTierCold)})
		if err != nil {
			s.logger.Error("failed to upload usage to blob storage", zap.Error(err))
		}
	})

	usage := model.UsageV2{
		ApiEndpoint:    "kubernetes-job",
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

			if req != nil && req.Job != nil {
				for _, container := range req.Job.Containers {
					if container == nil {
						continue
					}
					stats.KubernetesCurrentCPURequest += container.CpuRequest
					stats.KubernetesCurrentMemoryRequest += container.MemoryRequest
				}
			}
			if resp.Rightsizing != nil {
				for _, container := range resp.Rightsizing.ContainerResizing {
					if container != nil && container.Current != nil && container.Recommended != nil {
						stats.KubernetesRecommendedCPURequest += container.Recommended.CpuRequest
						stats.KubernetesRecommendedMemoryRequest += container.Recommended.MemoryRequest

						stats.KubernetesCPURequestSavings += container.Current.CpuRequest - container.Recommended.CpuRequest
						stats.KubernetesMemoryRequestSavings += container.Current.MemoryRequest - container.Recommended.MemoryRequest
					}
				}
			}

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

	jobRightSizingRecom, err := s.recomSvc.KubernetesJobRecommendation(*req.Job, req.Metrics, req.Preferences)
	if err != nil {
		s.logger.Error("failed to get kubernetes daemonset recommendation", zap.Error(err))
		return nil, err
	}

	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		return nil, err
	}

	// DO NOT change this, resp is used in updating usage
	resp = kubernetesPluginProto.KubernetesJobOptimizationResponse{
		Rightsizing: jobRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage

	return &resp, nil
}

func (s *kubernetesPluginServer) KubernetesNodeGetCost(ctx context.Context, req *kubernetesPluginProto.KubernetesNodeGetCostRequest) (*kubernetesPluginProto.KubernetesNodeGetCostResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp kubernetesPluginProto.KubernetesNodeGetCostResponse
	var err error

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get incoming context")
	}

	userIds := md.Get(httpserver.XKaytuUserIDHeader)
	userId := ""
	if len(userIds) > 0 {
		userId = userIds[0]
	}

	email := req.Identification["cluster_name"]
	if !strings.Contains(email, "@") {
		email = email + "@local.temp"
	}

	accountId := req.Identification["auth_info_name"]
	if accountId == "" {
		accountId = req.Identification["cluster_server"]
	}
	stats := model.Statistics{
		AccountID:   accountId,
		OrgEmail:    email,
		ResourceID:  req.GetNode().GetId(),
		Auth0UserId: userId,
	}
	statsOut, _ := json.Marshal(stats)

	reqJson, _ := json.Marshal(req)
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

	usage := model.UsageV2{
		ApiEndpoint:    "kubernetes-node",
		Request:        reqJson,
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
		}
		err = s.usageRepo.Update(usage.ID, usage)
		if err != nil {
			s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		}
	}()

	// TODO add cost calc

	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		return nil, err
	}

	// DO NOT change this, resp is used in updating usage
	resp = kubernetesPluginProto.KubernetesNodeGetCostResponse{
		Cost: nil, // TODO
	}

	return &resp, nil
}
