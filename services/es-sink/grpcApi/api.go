package grpcApi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	envoyAuth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/es-sink/service"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	kaytuGrpc "github.com/kaytu-io/kaytu-util/pkg/grpc"
	"github.com/kaytu-io/kaytu-util/proto/src/golang"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
)

type GRPCSinkServer struct {
	golang.EsSinkServiceServer
	logger        *zap.Logger
	esSinkService *service.EsSinkService
	grpcServer    *grpc.Server
	addr          string
}

func NewGRPCSinkServer(logger *zap.Logger, esSinkService *service.EsSinkService, authGrpcUri string, addr string) (*GRPCSinkServer, error) {
	authGRPCConn, err := grpc.Dial(authGrpcUri, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	if err != nil {
		logger.Error("failed to connect to auth grpc server", zap.Error(err))
		return nil, err
	}
	authClient := envoyAuth.NewAuthorizationClient(authGRPCConn)

	server := GRPCSinkServer{
		logger:        logger,
		esSinkService: esSinkService,
		addr:          addr,
	}

	server.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(128*1024*1024),
		grpc.UnaryInterceptor(kaytuGrpc.CheckGRPCAuthUnaryInterceptorWrapper(authClient)),
		grpc.StreamInterceptor(kaytuGrpc.CheckGRPCAuthStreamInterceptorWrapper(authClient)),
	)
	golang.RegisterEsSinkServiceServer(server.grpcServer, &server)

	return &server, nil
}

func (s *GRPCSinkServer) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.logger.Fatal("failed to listen on grpc port", zap.Error(err))
	}
	utils.EnsureRunGoroutine(func() {
		err := s.grpcServer.Serve(lis)
		if err != nil {
			s.logger.Fatal("failed to serve grpc server", zap.Error(err))
		}
	})
	return nil
}

func (s *GRPCSinkServer) Stop() {
	s.grpcServer.GracefulStop()
}

func (s *GRPCSinkServer) Ingest(ctx context.Context, req *golang.IngestRequest) (*golang.ResponseOK, error) {
	docs := make([]es.DocBase, 0, len(req.Docs))
	for _, doc := range req.Docs {
		var d es.DocBase
		err := json.Unmarshal(doc.Value, &d)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	if _, err := s.esSinkService.Ingest(ctx, docs); err != nil {
		s.logger.Error("failed to ingest data", zap.Error(err))
		return nil, err
	}
	return &golang.ResponseOK{}, nil
}
