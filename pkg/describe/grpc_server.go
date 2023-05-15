package describe

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"strconv"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"google.golang.org/grpc/metadata"

	"github.com/go-redis/redis/v8"
	"github.com/kaytu-io/kaytu-util/proto/src/golang"
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
)

type GRPCDescribeServer struct {
	db                     Database
	rdb                    *redis.Client
	producer               sarama.SyncProducer
	topic                  string
	logger                 *zap.Logger
	describeJobResultQueue queue.Interface

	golang.DescribeServiceServer
}

func NewDescribeServer(db Database, rdb *redis.Client, producer sarama.SyncProducer, topic string, describeJobResultQueue queue.Interface, logger *zap.Logger) *GRPCDescribeServer {
	return &GRPCDescribeServer{
		db:                     db,
		rdb:                    rdb,
		producer:               producer,
		topic:                  topic,
		describeJobResultQueue: describeJobResultQueue,
		logger:                 logger,
	}
}

func (s *GRPCDescribeServer) UpdateInProgress(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok && md.Get("resource-job-id") != nil {
		resourceJobIdStr := md.Get("resource-job-id")[0]
		resourceJobId, err := strconv.ParseUint(resourceJobIdStr, 10, 64)
		if err != nil {
			StreamFailureCount.WithLabelValues("aws").Inc()
			s.logger.Error("failed to parse resource job id:", zap.Error(err))
			return fmt.Errorf("failed to parse resource job id: %v", err)
		}
		err = s.db.UpdateDescribeResourceJobToInProgress(uint(resourceJobId)) //TODO this is called too much
		if err != nil {
			StreamFailureCount.WithLabelValues("aws").Inc()
			s.logger.Error("failed to update describe resource job status", zap.Error(err), zap.Uint("jobID", uint(resourceJobId)))
		}
	}
	return nil
}

func (s *GRPCDescribeServer) DeliverAWSResources(ctx context.Context, resources *golang.AWSResources) (*golang.ResponseOK, error) {
	startTime := time.Now().UnixMilli()
	defer func() {
		ResourceBatchProcessLatency.WithLabelValues("aws").Observe(float64(time.Now().UnixMilli() - startTime))
	}()

	//TODO-Saleh expensive operation on psql
	//if err := s.UpdateInProgress(ctx); err != nil {
	//	return nil, err
	//}

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
			Tags:          resource.Tags,
		}
		msgs = append(msgs, kafkaResource)
		msgs = append(msgs, lookupResource)
		ResourcesDescribedCount.WithLabelValues("aws", "successful").Inc()
	}

	if err := kafka.DoSend(s.producer, s.topic, 0, msgs, s.logger); err != nil {
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

	//TODO-Saleh expensive operation on psql
	//if err := s.UpdateInProgress(ctx); err != nil {
	//	return nil, err
	//}
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
			Tags:          resource.Tags,
		}
		msgs = append(msgs, kafkaResource)
		msgs = append(msgs, lookupResource)
		ResourcesDescribedCount.WithLabelValues("azure", "successful").Inc()
	}

	if err := kafka.DoSend(s.producer, s.topic, 1, msgs, s.logger); err != nil {
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
		DescribedResourceIDs: nil, // req.DescribedResourceIds,
	})
	return &golang.ResponseOK{}, err
}
