package wastage

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/model"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	pb "github.com/kaytu-io/plugin-kubernetes/plugin/proto/src/golang"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"time"
)

type Server struct {
	pb.OptimizationServer

	stream pb.Optimization_KubernetesPodOptimizationServer

	logger    *zap.Logger
	usageRepo repo.UsageV2Repo
	recomSvc  *recommendation.Service
}

func NewServer(logger *zap.Logger, usageRepo repo.UsageV2Repo, recomSvc *recommendation.Service) *Server {
	return &Server{
		logger:    logger,
		usageRepo: usageRepo,
		recomSvc:  recomSvc,
	}
}

func StartGrpcServer(server *Server, grpcServerAddress string) {
	lis, err := net.Listen("tcp", grpcServerAddress)
	if err != nil {
		server.logger.Error("failed to listen", zap.Error(err))
	}
	s := grpc.NewServer()
	pb.RegisterOptimizationServer(s, server)
	server.logger.Info("server listening at", zap.String("address", lis.Addr().String()))
	go func() {
		if err := s.Serve(lis); err != nil {
			server.logger.Error("failed to serve", zap.Error(err))
		}
	}()
}

func (s *Server) KubernetesPodOptimization(stream pb.Optimization_KubernetesPodOptimizationServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		s.stream = stream
		err = s.kubernetesPodOptimizationHandler(req)
		if err != nil {
			return err
		}
	}
}

func (s *Server) kubernetesPodOptimizationHandler(req *pb.KubernetesPodOptimizationRequest) error {
	start := time.Now()

	var resp pb.KubernetesPodOptimizationResponse
	var err error

	stats := model.Statistics{
		AccountID:   req.Identification["account"],
		OrgEmail:    req.Identification["org_m_email"],
		ResourceID:  req.Pod.Id,
		Auth0UserId: "", // TODO
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
		return nil
	}

	rdsRightSizingRecom, err := s.recomSvc.KubernetesPodRecommendation(*req.Pod, req.Metrics, req.Preferences)
	if err != nil {
		s.logger.Error("failed to get aws rds recommendation", zap.Error(err))
		return err
	}

	elapsed := time.Since(start).Seconds()
	usage.Latency = &elapsed
	err = s.usageRepo.Update(usage.ID, usage)
	if err != nil {
		s.logger.Error("failed to update usage", zap.Error(err), zap.Any("usage", usage))
		return err
	}

	// DO NOT change this, resp is used in updating usage
	resp = pb.KubernetesPodOptimizationResponse{
		Rightsizing: rdsRightSizingRecom,
	}
	// DO NOT change this, resp is used in updating usage

	err = s.stream.Send(&resp)
	if err != nil {
		s.logger.Error("failed to send response", zap.Error(err))
		return err
	}
	return nil
}
