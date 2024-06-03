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
	pb "github.com/kaytu-io/plugin-kubernetes/plugin/proto/src/golang"
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

		logger.Info("Request", zap.String("ReqID", reqId.String()), zap.Any("request", req))
		resp, err := handler(ctx, req)
		if err != nil {
			logger.Error("Request failed", zap.String("ReqID", reqId.String()), zap.Error(err))
		} else {
			logger.Info("Response", zap.String("ReqID", reqId.String()), zap.Any("response", resp))
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
		grpc.UnaryInterceptor(kaytuGrpc.CheckGRPCAuthUnaryInterceptorWrapper(authGrpcClient)),
		grpc.UnaryInterceptor(Logger(server.logger)),
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

	userId := md.Get(httpserver.XKaytuUserIDHeader)
	if len(userId) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	email := req.Identification["cluster_name"]
	if !strings.Contains(email, "@") {
		email = email + "@local.temp"
	}

	stats := model.Statistics{
		AccountID:   req.Identification["auth_info_name"],
		OrgEmail:    email,
		ResourceID:  req.Pod.Id,
		Auth0UserId: userId[0],
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

	rdsRightSizingRecom, err := s.recomSvc.KubernetesPodRecommendation(*req.Pod, req.Metrics, req.Preferences)
	if err != nil {
		s.logger.Error("failed to get aws rds recommendation", zap.Error(err))
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
		Rightsizing: rdsRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage

	return &resp, nil
}
