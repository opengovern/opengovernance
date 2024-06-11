package wastage

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	envoyAuth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	kaytuGrpc "github.com/kaytu-io/kaytu-util/pkg/grpc"
	pb "github.com/kaytu-io/plugin-kubernetes-internal/plugin/proto/src/golang"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"net"
	"strings"
	"time"
)

type Server struct {
	pb.OptimizationServer

	tracer    trace.Tracer
	logger    *zap.Logger
	usageRepo repo.UsageV2Repo
	recomSvc  *recommendation.Service
}

func NewServer(logger *zap.Logger, usageRepo repo.UsageV2Repo, recomSvc *recommendation.Service) *Server {
	return &Server{
		tracer:    otel.GetTracerProvider().Tracer("wastage.http.sources"),
		logger:    logger.Named("grpc"),
		usageRepo: usageRepo,
		recomSvc:  recomSvc,
	}
}

func Logger(logger *zap.Logger) func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		reqId := uuid.New()

		logger.Info("Request", zap.String("ReqID", reqId.String()))
		startTime := time.Now()
		resp, err := handler(ctx, req)
		elapsed := time.Since(startTime).Seconds()
		if err != nil {
			logger.Error("Request failed", zap.String("ReqID", reqId.String()), zap.Error(err), zap.Float64("latency", elapsed))
		} else {
			logger.Info("Request succeeded", zap.String("ReqID", reqId.String()), zap.Float64("latency", elapsed))
		}

		return resp, err
	}
}

func StartGrpcServer(server *Server, grpcServerAddress string, authGRPCURI string) error {
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
		grpc.MaxRecvMsgSize(128*1024*1024),
		grpc.UnaryInterceptor(kaytuGrpc.CheckGRPCAuthUnaryInterceptorWrapper(authGrpcClient)),
		grpc.ChainUnaryInterceptor(Logger(server.logger)),
	)
	pb.RegisterOptimizationServer(s, server)
	server.logger.Info("server listening at", zap.String("address", lis.Addr().String()))
	utils.EnsureRunGoroutine(func() {
		if err = s.Serve(lis); err != nil {
			server.logger.Error("failed to serve", zap.Error(err))
			panic(err)
		}
	})
	return nil
}

func (s *Server) KubernetesPodOptimization(ctx context.Context, req *pb.KubernetesPodOptimizationRequest) (*pb.KubernetesPodOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp pb.KubernetesPodOptimizationResponse
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

	stats := model.Statistics{
		AccountID:   req.Identification["auth_info_name"],
		OrgEmail:    email,
		ResourceID:  req.Pod.Id,
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
	usage := model.UsageV2{
		ApiEndpoint:    "kubernetes-pod",
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

			// TODO: We don't have cost here. What can we store?

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
	resp = pb.KubernetesPodOptimizationResponse{
		Rightsizing: podRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage

	return &resp, nil
}

func (s *Server) KubernetesDeploymentOptimization(ctx context.Context, req *pb.KubernetesDeploymentOptimizationRequest) (*pb.KubernetesDeploymentOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp pb.KubernetesDeploymentOptimizationResponse
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

	stats := model.Statistics{
		AccountID:   req.Identification["auth_info_name"],
		OrgEmail:    email,
		ResourceID:  req.GetDeployment().GetId(),
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
	usage := model.UsageV2{
		ApiEndpoint:    "kubernetes-deployment",
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

			// TODO: We don't have cost here. What can we store?

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
	resp = pb.KubernetesDeploymentOptimizationResponse{
		Rightsizing: deploymentRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage

	return &resp, nil
}

func (s *Server) KubernetesStatefulsetOptimization(ctx context.Context, req *pb.KubernetesStatefulsetOptimizationRequest) (*pb.KubernetesStatefulsetOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp pb.KubernetesStatefulsetOptimizationResponse
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

	stats := model.Statistics{
		AccountID:   req.Identification["auth_info_name"],
		OrgEmail:    email,
		ResourceID:  req.GetStatefulset().GetId(),
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
	usage := model.UsageV2{
		ApiEndpoint:    "kubernetes-statefulset",
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

			// TODO: We don't have cost here. What can we store?

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
	resp = pb.KubernetesStatefulsetOptimizationResponse{
		Rightsizing: statefulsetRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage

	return &resp, nil
}

func (s *Server) KubernetesDaemonsetOptimization(ctx context.Context, req *pb.KubernetesDaemonsetOptimizationRequest) (*pb.KubernetesDaemonsetOptimizationResponse, error) {
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, "get")
	defer span.End()

	var resp pb.KubernetesDaemonsetOptimizationResponse
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

	stats := model.Statistics{
		AccountID:   req.Identification["auth_info_name"],
		OrgEmail:    email,
		ResourceID:  req.GetDaemonset().GetId(),
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
	usage := model.UsageV2{
		ApiEndpoint:    "kubernetes-daemonset",
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

			// TODO: We don't have cost here. What can we store?

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
	resp = pb.KubernetesDaemonsetOptimizationResponse{
		Rightsizing: daemonsetRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage

	return &resp, nil
}
