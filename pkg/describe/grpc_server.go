package describe

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/queue"
	"google.golang.org/grpc/metadata"

	"github.com/go-redis/redis/v8"
	azure "github.com/kaytu-io/kaytu-azure-describer/azure/describer"
	azuremodel "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	"github.com/turbot/steampipe-plugin-sdk/v4/grpc/proto"
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/proto/src/golang"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"
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

func (s *GRPCDescribeServer) DeliverAWSResources(ctx context.Context, resources *golang.AWSResources) (*golang.ResponseOK, error) {
	startTime := time.Now().UnixMilli()
	defer func() {
		ResourceBatchProcessLatency.WithLabelValues("aws").Observe(float64(time.Now().UnixMilli() - startTime))
	}()

	var err error
	var resourceJobId uint64

	md, ok := metadata.FromIncomingContext(ctx)
	if ok && md.Get("resource-job-id") != nil {
		resourceJobIdStr := md.Get("resource-job-id")[0]
		resourceJobId, err = strconv.ParseUint(resourceJobIdStr, 10, 64)
		if err != nil {
			StreamFailureCount.WithLabelValues("aws").Inc()
			s.logger.Error("failed to parse resource job id:", zap.Error(err))
			return nil, fmt.Errorf("failed to parse resource job id: %v", err)
		}
		err = s.db.UpdateDescribeResourceJobToInProgress(uint(resourceJobId)) //TODO this is called too much
		if err != nil {
			StreamFailureCount.WithLabelValues("aws").Inc()
			s.logger.Error("failed to update describe resource job status", zap.Error(err), zap.Uint("jobID", uint(resourceJobId)))
		}
	}

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
			Description:   resource.DescriptionJson,
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
		msgs = append(msgs, kafkaResource, lookupResource)
	}

	if err := kafka.DoSend(s.producer, s.topic, 1, msgs, s.logger); err != nil {
		return nil, fmt.Errorf("send to kafka: %w", err)
	}
	return &golang.ResponseOK{}, nil
}

func (s *GRPCDescribeServer) DeliverAzureResources(ctx context.Context, resources *golang.AzureResources) (*golang.ResponseOK, error) {
	return &golang.ResponseOK{}, nil
}

func (s *GRPCDescribeServer) TempDeliverAzureResources(ctx context.Context, resources *golang.AzureResources) (*golang.ResponseOK, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok && md.Get("resource-job-id") != nil {
		resourceJobIdStr := md.Get("resource-job-id")[0]
		resourceJobId, err := strconv.ParseUint(resourceJobIdStr, 10, 64)
		if err != nil {
			StreamFailureCount.WithLabelValues("azure").Inc()
			s.logger.Error("failed to parse resource job id", zap.Error(err), zap.Uint("jobID", uint(resourceJobId)))
			return nil, fmt.Errorf("failed to parse resource job id: %v", err)
		}
		err = s.db.UpdateDescribeResourceJobToInProgress(uint(resourceJobId))
		if err != nil {
			StreamFailureCount.WithLabelValues("azure").Inc()
			s.logger.Error("failed to update describe resource job status", zap.Error(err), zap.Uint("jobID", uint(resourceJobId)))
		}
	}

	for _, resource := range resources.Resources {
		var description interface{}
		err := json.Unmarshal([]byte(resource.DescriptionJson), &description)
		if err != nil {
			ResourcesDescribedCount.WithLabelValues("azure", "failure").Inc()
			s.logger.Error("failed to unmarshal azure resource", zap.Error(err), zap.Uint32("jobID", resource.Job.JobId), zap.String("resourceID", resource.Id))
			return nil, err
		}

		azureResource := azure.Resource{
			ID:             resource.Id,
			Description:    description,
			Name:           resource.Name,
			Type:           resource.Type,
			ResourceGroup:  resource.ResourceGroup,
			Location:       resource.Location,
			SubscriptionID: resource.SubscriptionId,
		}
		err = s.HandleAzureResource(azureResource, resource.Job)
		if err != nil {
			ResourcesDescribedCount.WithLabelValues("azure", "failure").Inc()
			s.logger.Error("failed to handle azure resource", zap.Error(err), zap.Uint32("jobID", resource.Job.JobId), zap.String("resourceID", resource.Id))
			return nil, err
		}
		ResourcesDescribedCount.WithLabelValues("azure", "successful").Inc()
	}
	return &golang.ResponseOK{}, nil
}

func (s *GRPCDescribeServer) HandleAzureResource(resource azure.Resource, job *golang.DescribeJob) error {
	ctx := context.Background()

	var msgs []kafka.Doc
	var remaining int64 = MAX_INT64

	if s.rdb != nil {
		currentResourceLimitRemaining, err := s.rdb.Get(ctx, RedisKeyWorkspaceResourceRemaining).Result()
		if err != nil {
			return fmt.Errorf("redisGet: %v", err.Error())
		}
		remaining, err = strconv.ParseInt(currentResourceLimitRemaining, 10, 64)
		if remaining <= 0 {
			return fmt.Errorf("workspace has reached its max resources limit")
		}

		_, err = s.rdb.DecrBy(ctx, RedisKeyWorkspaceResourceRemaining, 1).Result()
		if err != nil {
			return fmt.Errorf("failed to decrement workspace resource limit: %v", err.Error())
		}
		remaining--
	}

	if resource.Description == nil {
		return nil
	}

	resource.Location = fixAzureLocation(resource.Location)

	azureMetadata := azuremodel.Metadata{
		ID:               resource.ID,
		Name:             resource.Name,
		SubscriptionID:   job.AccountId,
		Location:         resource.Location,
		CloudEnvironment: "AzurePublicCloud",
		ResourceType:     strings.ToLower(resource.Type),
		SourceID:         job.SourceId,
	}
	azureMetadataBytes, err := json.Marshal(azureMetadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %v", err.Error())
	}

	metadata := make(map[string]string)
	err = json.Unmarshal(azureMetadataBytes, &metadata)
	if err != nil {
		return fmt.Errorf("unmarshal metadata: %v", err.Error())
	}

	kafkaResource := es.Resource{
		ID:            resource.UniqueID(),
		Name:          resource.Name,
		ResourceGroup: resource.ResourceGroup,
		Location:      resource.Location,
		SourceType:    source.CloudAzure,
		ResourceType:  strings.ToLower(job.ResourceType),
		ResourceJobID: uint(job.JobId),
		SourceJobID:   uint(job.ParentJobId),
		SourceID:      job.SourceId,
		ScheduleJobID: uint(job.ScheduleJobId),
		CreatedAt:     job.DescribedAt,
		Description:   resource.Description,
		Metadata:      metadata,
	}
	lookupResource := es.LookupResource{
		ResourceID:    resource.UniqueID(),
		Name:          resource.Name,
		SourceType:    source.CloudAzure,
		ResourceType:  strings.ToLower(job.ResourceType),
		ResourceGroup: resource.ResourceGroup,
		ServiceName:   cloudservice.ServiceNameByResourceType(job.ResourceType),
		Category:      cloudservice.CategoryByResourceType(job.ResourceType),
		Location:      resource.Location,
		SourceID:      job.SourceId,
		ScheduleJobID: uint(job.ScheduleJobId),
		ResourceJobID: uint(job.JobId),
		SourceJobID:   uint(job.ParentJobId),
		CreatedAt:     job.DescribedAt,
		IsCommon:      cloudservice.IsCommonByResourceType(job.ResourceType),
	}
	pluginTableName := steampipe.ExtractTableName(job.ResourceType)
	desc, err := steampipe.ConvertToDescription(job.ResourceType, kafkaResource)
	if err != nil {
		return fmt.Errorf("convertToDescription: %v", err.Error())
	}
	pluginProvider := steampipe.ExtractPlugin(job.ResourceType)
	var cells map[string]*proto.Column
	if pluginProvider == steampipe.SteampipePluginAzure {
		cells, err = steampipe.AzureDescriptionToRecord(desc, pluginTableName)
		if err != nil {
			return fmt.Errorf("azureDescriptionToRecord: %v", err.Error())
		}
	} else {
		cells, err = steampipe.AzureADDescriptionToRecord(desc, pluginTableName)
		if err != nil {
			return fmt.Errorf("azureADDescriptionToRecord: %v", err.Error())
		}
	}
	for name, v := range cells {
		if name == "title" || name == "name" {
			kafkaResource.Metadata["name"] = v.GetStringValue()
		}
	}

	tags, err := steampipe.ExtractTags(job.ResourceType, kafkaResource)
	if err != nil {
		tags = map[string]string{}
		return fmt.Errorf("failed to build tags for service: %v", err.Error())
	}
	lookupResource.Tags = tags

	if s.rdb != nil {
		for key, value := range tags {
			key = strings.TrimSpace(key)
			_, err = s.rdb.SAdd(context.Background(), "tag-"+key, value).Result()
			if err != nil {
				return fmt.Errorf("failed to push tag into redis: %v", err.Error())
			}
			_, err = s.rdb.Expire(context.Background(), "tag-"+key, 12*time.Hour).Result() //TODO-Saleh set time based on describe interval
			if err != nil {
				return fmt.Errorf("failed to set tag expire into redis: %v", err.Error())
			}
		}
	}

	msgs = append(msgs, kafkaResource)
	msgs = append(msgs, lookupResource)

	if err := kafka.DoSend(s.producer, s.topic, 0, msgs, s.logger); err != nil {
		return fmt.Errorf("send to kafka: %w", err)
	}
	return nil
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
