package describe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	envoyauth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/go-redis/redis/v8"
	"github.com/gogo/googleapis/google/rpc"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/proto/src/golang"
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GRPCDescribeServer struct {
	db                        Database
	rdb                       *redis.Client
	producer                  *confluent_kafka.Producer
	topic                     string
	logger                    *zap.Logger
	describeJobResultQueue    queue.Interface
	DoProcessReceivedMessages bool
	authGrpcClient            envoyauth.AuthorizationClient

	golang.DescribeServiceServer
}

func NewDescribeServer(db Database, rdb *redis.Client, producer *confluent_kafka.Producer, topic string, describeJobResultQueue queue.Interface, authGrpcClient envoyauth.AuthorizationClient, logger *zap.Logger) *GRPCDescribeServer {
	return &GRPCDescribeServer{
		db:                        db,
		rdb:                       rdb,
		producer:                  producer,
		topic:                     topic,
		describeJobResultQueue:    describeJobResultQueue,
		logger:                    logger,
		DoProcessReceivedMessages: true,
		authGrpcClient:            authGrpcClient,
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
	err := s.db.UpdateDescribeResourceJobToInProgress(uint(req.JobId)) //TODO this is called too much
	if err != nil {
		return nil, err
	}
	return &golang.ResponseOK{}, nil
}

func (s *GRPCDescribeServer) DeliverAWSResources(ctx context.Context, resources *golang.AWSResources) (*golang.ResponseOK, error) {
	startTime := time.Now().UnixMilli()
	defer func() {
		ResourceBatchProcessLatency.WithLabelValues("aws").Observe(float64(time.Now().UnixMilli() - startTime))
	}()

	var msgs []kafka.Doc
	for _, resource := range resources.GetResources() {
		var description any
		err := json.Unmarshal([]byte(resource.DescriptionJson), &description)
		if err != nil {
			ResourcesDescribedCount.WithLabelValues("aws", "failure").Inc()
			s.logger.Error("failed to parse resource description json", zap.Error(err), zap.Uint32("jobID", resource.Job.JobId), zap.String("resourceID", resource.Id))
			return nil, err
		}
		kafkaResource := es.Resource{
			ID:            resource.UniqueId,
			ARN:           resource.Arn,
			Name:          resource.Name,
			SourceType:    source.CloudAWS,
			ResourceType:  strings.ToLower(resource.Job.ResourceType),
			ResourceGroup: "",
			Location:      resource.Region,
			SourceID:      resource.Job.SourceId,
			ResourceJobID: uint(resource.Job.JobId),
			SourceJobID:   uint(resource.Job.ParentJobId),
			ScheduleJobID: uint(resource.Job.ScheduleJobId),
			CreatedAt:     resource.Job.DescribedAt,
			Description:   description,
			Metadata:      resource.Metadata,
		}
		//kmsg, _ := json.Marshal(kafkaResource)
		//keys, _ := kafkaResource.KeysAndIndex()
		//id := kafka.HashOf(keys...)
		//s.logger.Warn(fmt.Sprintf("sending resource id=%s : %s", id, string(kmsg)))

		var tags []es.Tag
		for k, v := range resource.Tags {
			tags = append(tags, es.Tag{
				Key:   k,
				Value: v,
			})
		}
		lookupResource := es.LookupResource{
			ResourceID:    resource.UniqueId,
			Name:          resource.Name,
			SourceType:    source.CloudAWS,
			ResourceType:  strings.ToLower(resource.Job.ResourceType),
			ServiceName:   cloudservice.ServiceNameByResourceType(resource.Job.ResourceType),
			Category:      cloudservice.CategoryByResourceType(resource.Job.ResourceType),
			ResourceGroup: "",
			Location:      resource.Region,
			SourceID:      resource.Job.SourceId,
			ResourceJobID: uint(resource.Job.JobId),
			SourceJobID:   uint(resource.Job.ParentJobId),
			ScheduleJobID: uint(resource.Job.ScheduleJobId),
			CreatedAt:     resource.Job.DescribedAt,
			IsCommon:      cloudservice.IsCommonByResourceType(resource.Job.ResourceType),
			Tags:          tags,
		}
		//kmsg, _ = json.Marshal(lookupResource)
		//keys, _ = lookupResource.KeysAndIndex()
		//id = kafka.HashOf(keys...)
		//s.logger.Warn(fmt.Sprintf("sending lookup id=%s : %s", id, string(kmsg)))

		msgs = append(msgs, kafkaResource)
		msgs = append(msgs, lookupResource)
		ResourcesDescribedCount.WithLabelValues("aws", "successful").Inc()
	}

	if !s.DoProcessReceivedMessages {
		return &golang.ResponseOK{}, nil
	}
	if err := kafka.DoSend(s.producer, s.topic, -1, msgs, s.logger); err != nil {
		StreamFailureCount.WithLabelValues("aws").Inc()
		return nil, fmt.Errorf("send to kafka: %w", err)
	}
	return &golang.ResponseOK{}, nil
}

func (s *GRPCDescribeServer) DeliverAzureResources(ctx context.Context, resources *golang.AzureResources) (*golang.ResponseOK, error) {
	startTime := time.Now().UnixMilli()
	defer func() {
		ResourceBatchProcessLatency.WithLabelValues("azure").Observe(float64(time.Now().UnixMilli() - startTime))
	}()

	var msgs []kafka.Doc
	for _, resource := range resources.GetResources() {
		var description any
		err := json.Unmarshal([]byte(resource.DescriptionJson), &description)
		if err != nil {
			ResourcesDescribedCount.WithLabelValues("azure", "failure").Inc()
			s.logger.Error("failed to parse resource description json", zap.Error(err), zap.Uint32("jobID", resource.Job.JobId), zap.String("resourceID", resource.Id))
			return nil, err
		}

		kafkaResource := es.Resource{
			ID:            resource.UniqueId,
			ARN:           "",
			Description:   description,
			SourceType:    source.CloudAzure,
			ResourceType:  strings.ToLower(resource.Job.ResourceType),
			ResourceJobID: uint(resource.Job.JobId),
			SourceID:      resource.Job.SourceId,
			SourceJobID:   uint(resource.Job.ParentJobId),
			Metadata:      resource.Metadata,
			Name:          resource.Name,
			ResourceGroup: resource.ResourceGroup,
			Location:      resource.Location,
			ScheduleJobID: uint(resource.Job.ScheduleJobId),
			CreatedAt:     resource.Job.DescribedAt,
		}
		//kmsg, _ := json.Marshal(kafkaResource)
		//keys, _ := kafkaResource.KeysAndIndex()
		//id := kafka.HashOf(keys...)
		//s.logger.Warn(fmt.Sprintf("sending resource id=%s : %s", id, string(kmsg)))

		var tags []es.Tag
		for k, v := range resource.Tags {
			tags = append(tags, es.Tag{
				Key:   k,
				Value: v,
			})
		}
		lookupResource := es.LookupResource{
			ResourceID:    resource.UniqueId,
			Name:          resource.Name,
			SourceType:    source.CloudAzure,
			ResourceType:  strings.ToLower(resource.Job.ResourceType),
			ServiceName:   cloudservice.ServiceNameByResourceType(resource.Job.ResourceType),
			Category:      cloudservice.CategoryByResourceType(resource.Job.ResourceType),
			ResourceGroup: resource.ResourceGroup,
			Location:      resource.Location,
			SourceID:      resource.Job.SourceId,
			ResourceJobID: uint(resource.Job.JobId),
			SourceJobID:   uint(resource.Job.ParentJobId),
			ScheduleJobID: uint(resource.Job.ScheduleJobId),
			CreatedAt:     resource.Job.DescribedAt,
			IsCommon:      cloudservice.IsCommonByResourceType(resource.Job.ResourceType),
			Tags:          tags,
		}
		//kmsg, _ = json.Marshal(lookupResource)
		//keys, _ = lookupResource.KeysAndIndex()
		//id = kafka.HashOf(keys...)
		//s.logger.Warn(fmt.Sprintf("sending lookup id=%s : %s", id, string(kmsg)))

		msgs = append(msgs, kafkaResource)
		msgs = append(msgs, lookupResource)
		ResourcesDescribedCount.WithLabelValues("azure", "successful").Inc()
	}

	if err := kafka.DoSend(s.producer, s.topic, -1, msgs, s.logger); err != nil {
		StreamFailureCount.WithLabelValues("azure").Inc()
		return nil, fmt.Errorf("send to kafka: %w", err)
	}
	return &golang.ResponseOK{}, nil
}

func (s *GRPCDescribeServer) DeliverResult(ctx context.Context, req *golang.DeliverResultRequest) (*golang.ResponseOK, error) {
	ResultsDeliveredCount.WithLabelValues(req.DescribeJob.SourceType).Inc()

	err := s.describeJobResultQueue.Publish(DescribeJobResult{
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
	})
	return &golang.ResponseOK{}, err
}
