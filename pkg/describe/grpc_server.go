package describe

import (
	"context"
	"net/http"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/gogo/googleapis/google/rpc"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/config"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/enums"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	kaytuTrace "github.com/kaytu-io/kaytu-util/pkg/trace"
	"github.com/kaytu-io/kaytu-util/proto/src/golang"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GRPCDescribeServer struct {
	db                        db.Database
	producer                  *confluent_kafka.Producer
	conf                      config.SchedulerConfig
	topic                     string
	logger                    *zap.Logger
	DoProcessReceivedMessages bool
	authGrpcClient            envoyauth.AuthorizationClient

	golang.DescribeServiceServer
}

func NewDescribeServer(db db.Database, producer *confluent_kafka.Producer, topic string, authGrpcClient envoyauth.AuthorizationClient, logger *zap.Logger, conf config.SchedulerConfig) *GRPCDescribeServer {
	return &GRPCDescribeServer{
		db:                        db,
		producer:                  producer,
		topic:                     topic,
		logger:                    logger,
		DoProcessReceivedMessages: true,
		authGrpcClient:            authGrpcClient,
		conf:                      conf,
	}
}

func (s *GRPCDescribeServer) checkGRPCAuth(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "missing metadata")
	}

	mdHeaders := make(map[string]string)
	for k, v := range md {
		if len(v) > 0 {
			mdHeaders[k] = v[0]
		}
	}

	result, err := s.authGrpcClient.Check(ctx, &envoyauth.CheckRequest{
		Attributes: &envoyauth.AttributeContext{
			Request: &envoyauth.AttributeContext_Request{
				Http: &envoyauth.AttributeContext_HttpRequest{
					Headers: mdHeaders,
				},
			},
		},
	})
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	if result.GetStatus() == nil || result.GetStatus().GetCode() != int32(rpc.OK) {
		return status.Errorf(codes.Unauthenticated, http.StatusText(http.StatusUnauthorized))
	}

	return nil
}

func (s *GRPCDescribeServer) grpcUnaryAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if err := s.checkGRPCAuth(ctx); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func (s *GRPCDescribeServer) grpcStreamAuthInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := s.checkGRPCAuth(ss.Context()); err != nil {
		return err
	}
	return handler(srv, ss)
}

func (s *GRPCDescribeServer) SetInProgress(ctx context.Context, req *golang.SetInProgressRequest) (*golang.ResponseOK, error) {
	s.logger.Info("changing job to in progress", zap.Uint("jobId", uint(req.JobId)))
	err := s.db.UpdateDescribeConnectionJobToInProgress(uint(req.JobId)) // TODO this is called too much
	if err != nil {
		return nil, err
	}
	return &golang.ResponseOK{}, nil
}

func (s *GRPCDescribeServer) DeliverResult(ctx context.Context, req *golang.DeliverResultRequest) (*golang.ResponseOK, error) {
	ResultsDeliveredCount.WithLabelValues(req.DescribeJob.SourceType).Inc()
	result := DescribeJobResult{
		JobID:       uint(req.JobId),
		ParentJobID: uint(req.ParentJobId),
		Status:      api.DescribeResourceJobStatus(req.Status),
		Error:       req.Error,
		ErrorCode:   req.ErrorCode,
		DescribeJob: DescribeJob{
			JobID:         uint(req.DescribeJob.JobId),
			ScheduleJobID: uint(req.DescribeJob.ScheduleJobId),
			ParentJobID:   uint(req.DescribeJob.ParentJobId),
			ResourceType:  req.DescribeJob.ResourceType,
			SourceID:      req.DescribeJob.SourceId,
			AccountID:     req.DescribeJob.AccountId,
			DescribedAt:   req.DescribeJob.DescribedAt,
			SourceType:    source.Type(req.DescribeJob.SourceType),
			CipherText:    req.DescribeJob.ConfigReg,
			TriggerType:   enums.DescribeTriggerType(req.DescribeJob.TriggerType),
			RetryCounter:  uint(req.DescribeJob.RetryCounter),
		},
		DescribedResourceIDs: req.DescribedResourceIds,
	}

	s.logger.Info("Result delivered",
		zap.Uint("jobID", result.JobID),
		zap.String("status", string(result.Status)),
	)

	ctx, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx, kaytuTrace.GetCurrentFuncName())
	defer span.End()

	var docs []kafka.Doc
	docs = append(docs, result)

	err := kafka.DoSend(s.producer, "kaytu-describe-results-queue", -1, docs, s.logger, nil)
	// err := s.describeJobResultQueue.Publish(result)
	if err != nil {
		s.logger.Error("Failed to publish into rabbitMQ",
			zap.Uint("jobID", result.JobID),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info("Publish finished",
		zap.Uint("jobID", result.JobID),
		zap.String("status", string(result.Status)),
	)
	return &golang.ResponseOK{}, nil
}
