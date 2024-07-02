package grpc_server

import (
	"context"
	"crypto/tls"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/alitto/pond"
	envoyAuth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/wastage/config"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	kaytuGrpc "github.com/kaytu-io/kaytu-util/pkg/grpc"
	kubernetesPluginProto "github.com/kaytu-io/plugin-kubernetes-internal/plugin/proto/src/golang"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"time"
)

type Server struct {
	logger                 *zap.Logger
	kubernetesPluginServer *kubernetesPluginServer
}

func NewServer(logger *zap.Logger, cfg config.WastageConfig, blobClient *azblob.Client, blobWorkerPool *pond.WorkerPool, usageRepo repo.UsageV2Repo, recomSvc *recommendation.Service) *Server {
	kuberServer := newKubernetesPluginServer(logger, cfg, blobClient, blobWorkerPool, usageRepo, recomSvc)

	svr := Server{
		logger:                 logger,
		kubernetesPluginServer: kuberServer,
	}
	return &svr
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
		grpc.MaxRecvMsgSize(256*1024*1024),
		grpc.UnaryInterceptor(kaytuGrpc.CheckGRPCAuthUnaryInterceptorWrapper(authGrpcClient)),
		grpc.ChainUnaryInterceptor(Logger(server.logger)),
	)
	kubernetesPluginProto.RegisterOptimizationServer(s, server.kubernetesPluginServer)
	server.logger.Info("server listening at", zap.String("address", lis.Addr().String()))
	utils.EnsureRunGoroutine(func() {
		if err = s.Serve(lis); err != nil {
			server.logger.Error("failed to serve", zap.Error(err))
			panic(err)
		}
	})
	return nil
}
