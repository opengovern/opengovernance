package describe

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/aws/describer"
	azureDescriber "gitlab.com/keibiengine/keibi-engine/pkg/azure/describer"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/proto/src/golang"
	"google.golang.org/grpc"

	awsmodel "gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
	azuremodel "gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/turbot/steampipe-plugin-sdk/v4/grpc/proto"

	"github.com/go-redis/redis/v8"

	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"

	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"

	"github.com/go-errors/errors"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"go.uber.org/zap"
)

const MAX_INT64 = 9223372036854775807

var DoDescribeJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "describe_worker",
	Name:      "do_describe_jobs_total",
	Help:      "Count of done describe jobs in describe-worker service",
}, []string{"provider", "resource_type", "status"})

var DoDescribeJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "keibi",
	Subsystem: "describe_worker",
	Name:      "do_describe_jobs_duration_seconds",
	Help:      "Duration of done describe jobs in describe-worker service",
	Buckets:   []float64{5, 60, 300, 600, 1800, 3600, 7200, 36000},
}, []string{"provider", "resource_type", "status"})

const (
	InventorySummaryIndex = "inventory_summary"
	describeJobTimeout    = 3 * 60 * time.Minute
)

type AWSAccountConfig struct {
	AccountID     string   `json:"accountId"`
	Regions       []string `json:"regions"`
	SecretKey     string   `json:"secretKey"`
	AccessKey     string   `json:"accessKey"`
	SessionToken  string   `json:"sessionToken"`
	AssumeRoleARN string   `json:"assumeRoleARN"`
}

func AWSAccountConfigFromMap(m map[string]interface{}) (AWSAccountConfig, error) {
	mj, err := json.Marshal(m)
	if err != nil {
		return AWSAccountConfig{}, err
	}

	var c AWSAccountConfig
	err = json.Unmarshal(mj, &c)
	if err != nil {
		return AWSAccountConfig{}, err
	}

	return c, nil
}

type AzureSubscriptionConfig struct {
	SubscriptionID  string `json:"subscriptionId"`
	TenantID        string `json:"tenantId"`
	ObjectID        string `json:"objectId"`
	SecretID        string `json:"secretId"`
	ClientID        string `json:"clientId"`
	ClientSecret    string `json:"clientSecret"`
	CertificatePath string `json:"certificatePath"`
	CertificatePass string `json:"certificatePass"`
	Username        string `json:"username"`
	Password        string `json:"password"`
}

func AzureSubscriptionConfigFromMap(m map[string]interface{}) (AzureSubscriptionConfig, error) {
	mj, err := json.Marshal(m)
	if err != nil {
		return AzureSubscriptionConfig{}, err
	}

	var c AzureSubscriptionConfig
	err = json.Unmarshal(mj, &c)
	if err != nil {
		return AzureSubscriptionConfig{}, err
	}

	return c, nil
}

type DescribeJob struct {
	JobID         uint // DescribeResourceJob ID
	ScheduleJobID uint
	ParentJobID   uint // DescribeSourceJob ID
	ResourceType  string
	SourceID      string
	AccountID     string
	DescribedAt   int64
	SourceType    api.SourceType
	CipherText    string
	TriggerType   enums.DescribeTriggerType
	RetryCounter  uint
}

type DescribeJobResult struct {
	JobID                uint
	ParentJobID          uint
	Status               api.DescribeResourceJobStatus
	Error                string
	DescribeJob          DescribeJob
	DescribedResourceIDs []string
}

// Do will perform the job which includes the following tasks:
//
//  1. Describing resources from the cloud providee based on the job definition.
//  2. Send the described resources to Kafka to be consumed by other systems.
//
// There are a variety of things that could go wrong in the process. This method will
// do its best to complete the task even if some errors occur along the way. However,
// if any error occurs, The JobResult will indicate that through the Status and Error
// will be set to the first error that occured.
func (j DescribeJob) Do(ctx context.Context, vlt *vault.KMSVaultSourceConfig, keyARN string, rdb *redis.Client, logger *zap.Logger, describeDeliverEndpoint *string) {
	logger.Info("Starting DescribeJob", zap.Uint("jobID", j.JobID), zap.Uint("scheduleJobID", j.ScheduleJobID), zap.Uint("parentJobID", j.ParentJobID), zap.String("resourceType", j.ResourceType), zap.String("sourceID", j.SourceID), zap.String("accountID", j.AccountID), zap.Int64("describedAt", j.DescribedAt), zap.String("sourceType", string(j.SourceType)), zap.String("configReg", j.CipherText), zap.String("triggerType", string(j.TriggerType)), zap.Uint("retryCounter", j.RetryCounter))

	startTime := time.Now().Unix()
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("paniced with error:", err)
			fmt.Println(errors.Wrap(err, 2).ErrorStack())
		}
	}()

	// Assume it succeeded unless it fails somewhere
	var (
		status               = api.DescribeResourceJobSucceeded
		firstErr    error    = nil
		resourceIDs []string = nil
	)

	fail := func(err error) {
		status = api.DescribeResourceJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}

	ctx, cancel := context.WithTimeout(ctx, describeJobTimeout)
	defer cancel()

	if conn, err := grpc.Dial(*describeDeliverEndpoint); err == nil {
		defer conn.Close()
		client := golang.NewDescribeServiceClient(conn)

		if config, err := vlt.Decrypt(j.CipherText, keyARN); err == nil {
			_, resourceIDs, err = doDescribe(ctx, rdb, j, config, logger, &client)
			if err != nil {
				// Don't return here. In certain cases, such as AWS, resources might be
				// available for some regions while there was failures in other regions.
				// Instead, continue to write whatever you can to kafka.
				fail(fmt.Errorf("describe resources: %w", err))
			}
		} else if config == nil {
			fail(fmt.Errorf("config is null! path is: %s", j.CipherText))
		} else {
			fail(fmt.Errorf("resource source config: %w", err))
		}

		errMsg := ""
		if firstErr != nil {
			errMsg = firstErr.Error()
		}
		if status == api.DescribeResourceJobSucceeded {
			DoDescribeJobsDuration.WithLabelValues(string(j.SourceType), j.ResourceType, "successful").Observe(float64(time.Now().Unix() - startTime))
			DoDescribeJobsCount.WithLabelValues(string(j.SourceType), j.ResourceType, "successful").Inc()
		}

		_, err := client.DeliverResult(ctx, &golang.DeliverResultRequest{
			JobId:       uint32(j.JobID),
			ParentJobId: uint32(j.ParentJobID),
			Status:      string(status),
			Error:       errMsg,
			DescribeJob: &golang.DescribeJob{
				JobId:         uint32(j.JobID),
				ScheduleJobId: uint32(j.ScheduleJobID),
				ParentJobId:   uint32(j.ParentJobID),
				ResourceType:  j.ResourceType,
				SourceId:      j.SourceID,
				AccountId:     j.AccountID,
				DescribedAt:   j.DescribedAt,
				SourceType:    string(j.SourceType),
				ConfigReg:     j.CipherText,
				TriggerType:   string(j.TriggerType),
				RetryCounter:  uint32(j.RetryCounter),
			},
			DescribedResourceIds: resourceIDs,
		})
		if err != nil {
			fail(fmt.Errorf("DeliverResult: %v", err))
		}
	} else {
		fail(fmt.Errorf("grpc: %v", err))
	}
}

// doDescribe describes the sources, e.g. AWS, Azure and returns the responses.
func doDescribe(ctx context.Context, rdb *redis.Client, job DescribeJob, config map[string]interface{}, logger *zap.Logger, client *golang.DescribeServiceClient) ([]kafka.Doc, []string, error) {
	logger.Info(fmt.Sprintf("Proccessing Job: ID[%d] ParentJobID[%d] RosourceType[%s]\n", job.JobID, job.ParentJobID, job.ResourceType))

	switch job.SourceType {
	case api.SourceCloudAWS:
		return doDescribeAWS(ctx, rdb, job, config, logger, client)
	case api.SourceCloudAzure:
		return doDescribeAzure(ctx, rdb, job, config, logger, client)
	default:
		return nil, nil, fmt.Errorf("invalid SourceType: %s", job.SourceType)
	}
}

func doDescribeAWS(ctx context.Context, rdb *redis.Client, job DescribeJob, config map[string]interface{},
	logger *zap.Logger, client *golang.DescribeServiceClient) ([]kafka.Doc, []string, error) {

	var resourceIDs []string
	creds, err := AWSAccountConfigFromMap(config)
	if err != nil {
		return nil, nil, fmt.Errorf("aws account credentials: %w", err)
	}

	var clientStream *describer.StreamSender
	if client != nil {
		stream, err := (*client).DeliverAWSResources(context.Background())
		if err != nil {
			return nil, nil, err
		}

		f := func(resource describer.Resource) error {
			descriptionJSON, err := json.Marshal(resource.Description)
			if err != nil {
				return err
			}

			return stream.Send(&golang.AWSResource{
				Arn:             resource.ARN,
				Id:              resource.ID,
				Name:            resource.Name,
				Account:         resource.Account,
				Region:          resource.Region,
				Partition:       resource.Partition,
				Type:            resource.Type,
				DescriptionJson: string(descriptionJSON),
				Job: &golang.DescribeJob{
					JobId:         uint32(job.JobID),
					ScheduleJobId: uint32(job.ScheduleJobID),
					ParentJobId:   uint32(job.ParentJobID),
					ResourceType:  job.ResourceType,
					SourceId:      job.SourceID,
					AccountId:     job.AccountID,
					DescribedAt:   job.DescribedAt,
					SourceType:    string(job.SourceType),
					ConfigReg:     job.CipherText,
					TriggerType:   string(job.TriggerType),
					RetryCounter:  uint32(job.RetryCounter),
				},
			})
		}
		clientStream = (*describer.StreamSender)(&f)
	}

	output, err := aws.GetResources(
		ctx,
		job.ResourceType,
		job.TriggerType,
		creds.AccountID,
		creds.Regions,
		creds.AccessKey,
		creds.SecretKey,
		creds.SessionToken,
		creds.AssumeRoleARN,
		false,
		clientStream,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("AWS: %w", err)
	}

	var errs []string
	for region, err := range output.Errors {
		if err != "" {
			errs = append(errs, fmt.Sprintf("region (%s): %s", region, err))
		}
	}

	var msgs []kafka.Doc

	for _, resources := range output.Resources {
		var remaining int64 = MAX_INT64
		if rdb != nil {
			currentResourceLimitRemaining, err := rdb.Get(ctx, RedisKeyWorkspaceResourceRemaining).Result()
			if err != nil {
				errs = append(errs, fmt.Sprintf("redisGet: %v", err.Error()))
				continue
			}
			remaining, err = strconv.ParseInt(currentResourceLimitRemaining, 10, 64)
			if remaining <= 0 {
				errs = append(errs, fmt.Sprintf("workspace has reached its max resources limit"))
				continue
			}
			_, err = rdb.DecrBy(ctx, RedisKeyWorkspaceResourceRemaining, int64(len(resources))).Result()
			if err != nil {
				errs = append(errs, fmt.Sprintf("redisDecr: %v", err.Error()))
				continue
			}
		}
		for _, resource := range resources {
			if resource.Description == nil {
				continue
			}
			if rdb != nil {
				if remaining <= 0 {
					errs = append(errs, fmt.Sprintf("workspace has reached its max resources limit"))
					break
				}
				remaining--
			}

			awsMetadata := awsmodel.Metadata{
				Name:         resource.Name,
				AccountID:    resource.Account,
				SourceID:     job.SourceID,
				Region:       resource.Region,
				Partition:    resource.Name,
				ResourceType: strings.ToLower(resource.Type),
			}
			awsMetadataBytes, err := json.Marshal(awsMetadata)
			if err != nil {
				errs = append(errs, fmt.Sprintf("marshal metadata: %v", err.Error()))
				continue
			}
			metadata := make(map[string]string)
			err = json.Unmarshal(awsMetadataBytes, &metadata)
			if err != nil {
				errs = append(errs, fmt.Sprintf("unmarshal metadata: %v", err.Error()))
				continue
			}

			kafkaResource := es.Resource{
				ID:            resource.UniqueID(),
				Name:          resource.Name,
				SourceType:    source.CloudAWS,
				ResourceType:  strings.ToLower(job.ResourceType),
				ResourceGroup: "",
				Location:      resource.Region,
				SourceID:      job.SourceID,
				ResourceJobID: job.JobID,
				SourceJobID:   job.ParentJobID,
				ScheduleJobID: job.ScheduleJobID,
				CreatedAt:     job.DescribedAt,
				Description:   resource.Description,
				Metadata:      metadata,
			}
			lookupResource := es.LookupResource{
				ResourceID:    resource.UniqueID(),
				Name:          resource.Name,
				SourceType:    source.CloudAWS,
				ResourceType:  strings.ToLower(job.ResourceType),
				ServiceName:   cloudservice.ServiceNameByResourceType(job.ResourceType),
				Category:      cloudservice.CategoryByResourceType(job.ResourceType),
				ResourceGroup: "",
				Location:      resource.Region,
				SourceID:      job.SourceID,
				ResourceJobID: job.JobID,
				SourceJobID:   job.ParentJobID,
				ScheduleJobID: job.ScheduleJobID,
				CreatedAt:     job.DescribedAt,
				IsCommon:      cloudservice.IsCommonByResourceType(job.ResourceType),
			}
			resourceIDs = append(resourceIDs, resource.UniqueID())
			pluginTableName := steampipe.ExtractTableName(job.ResourceType)
			desc, err := steampipe.ConvertToDescription(job.ResourceType, kafkaResource)
			if err != nil {
				errs = append(errs, fmt.Sprintf("convertToDescription: %v", err.Error()))
				continue
			}
			cells, err := steampipe.AWSDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				errs = append(errs, fmt.Sprintf("awsdescriptionToRecord: %v", err.Error()))
				continue
			}
			for name, v := range cells {
				if name == "title" || name == "name" {
					kafkaResource.Metadata["name"] = v.GetStringValue()
				}
			}

			tags, err := steampipe.ExtractTags(job.ResourceType, kafkaResource)
			if err != nil {
				errs = append(errs, fmt.Sprintf("failed to build tags for service: %v", err.Error()))
				tags = map[string]string{}
			}
			lookupResource.Tags = tags
			if rdb != nil {
				for key, value := range tags {
					key = strings.TrimSpace(key)
					_, err = rdb.SAdd(context.Background(), "tag-"+key, value).Result()
					if err != nil {
						errs = append(errs, fmt.Sprintf("failed to push tag into redis: %v", err.Error()))
						continue
					}

					_, err = rdb.Expire(context.Background(), "tag-"+key, 12*time.Hour).Result() //TODO-Saleh set time based on describe interval
					if err != nil {
						errs = append(errs, fmt.Sprintf("failed to set tag expire into redis: %v", err.Error()))
						continue
					}
				}
			}

			msgs = append(msgs, kafkaResource)
			msgs = append(msgs, lookupResource)
		}
	}

	logger.Info(fmt.Sprintf("job[%d] parent[%d] resourceType[%s]\n",
		job.JobID, job.ParentJobID, job.ResourceType))

	// For AWS resources, since they are queries independently per region,
	// if there is an error in some regions, return those errors. For the regions
	// with no error, return the list of resources.
	if len(errs) > 0 {
		err = fmt.Errorf("AWS: [%s]", strings.Join(errs, ","))
	} else {
		err = nil
	}

	return msgs, resourceIDs, err
}

func doDescribeAzure(ctx context.Context, rdb *redis.Client, job DescribeJob, config map[string]interface{},
	logger *zap.Logger, client *golang.DescribeServiceClient) ([]kafka.Doc, []string, error) {
	var clientStream *azureDescriber.StreamSender
	if client != nil {
		stream, err := (*client).DeliverAzureResources(context.Background())
		if err != nil {
			return nil, nil, err
		}

		f := func(resource azureDescriber.Resource) error {
			descriptionJSON, err := json.Marshal(resource.Description)
			if err != nil {
				return err
			}

			return stream.Send(&golang.AzureResource{
				Id:              resource.ID,
				Name:            resource.Name,
				Type:            resource.Type,
				ResourceGroup:   resource.ResourceGroup,
				Location:        resource.Location,
				SubscriptionId:  resource.SubscriptionID,
				DescriptionJson: string(descriptionJSON),
				Job: &golang.DescribeJob{
					JobId:         uint32(job.JobID),
					ScheduleJobId: uint32(job.ScheduleJobID),
					ParentJobId:   uint32(job.ParentJobID),
					ResourceType:  job.ResourceType,
					SourceId:      job.SourceID,
					AccountId:     job.AccountID,
					DescribedAt:   job.DescribedAt,
					SourceType:    string(job.SourceType),
					ConfigReg:     job.CipherText,
					TriggerType:   string(job.TriggerType),
					RetryCounter:  uint32(job.RetryCounter),
				},
			})
		}
		clientStream = (*azureDescriber.StreamSender)(&f)
	}

	var resourceIDs []string

	logger.Warn("starting to describe azure subscription", zap.String("resourceType", job.ResourceType), zap.Uint("jobID", job.JobID))
	creds, err := AzureSubscriptionConfigFromMap(config)
	if err != nil {
		return nil, nil, fmt.Errorf("azure subscription credentials: %w", err)
	}

	subscriptionId := job.AccountID
	if len(subscriptionId) == 0 {
		subscriptionId = creds.SubscriptionID
	}

	logger.Warn("getting resources", zap.String("resourceType", job.ResourceType), zap.Uint("jobID", job.JobID))
	output, err := azure.GetResources(
		ctx,
		job.ResourceType,
		job.TriggerType,
		[]string{subscriptionId},
		azure.AuthConfig{
			TenantID:            creds.TenantID,
			ClientID:            creds.ClientID,
			ClientSecret:        creds.ClientSecret,
			CertificatePath:     creds.CertificatePath,
			CertificatePassword: creds.CertificatePass,
			Username:            creds.Username,
			Password:            creds.Password,
		},
		string(azure.AuthEnv),
		"",
		clientStream,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("azure: %w", err)
	}
	logger.Warn("got the resources, finding summaries", zap.String("resourceType", job.ResourceType), zap.Uint("jobID", job.JobID))

	var msgs []kafka.Doc
	var errs []string
	var remaining int64 = MAX_INT64

	if rdb != nil {
		currentResourceLimitRemaining, err := rdb.Get(ctx, RedisKeyWorkspaceResourceRemaining).Result()
		if err != nil {
			return nil, nil, fmt.Errorf("redisGet: %v", err.Error())
		}
		remaining, err = strconv.ParseInt(currentResourceLimitRemaining, 10, 64)
		if remaining <= 0 {
			return nil, nil, fmt.Errorf("workspace has reached its max resources limit")
		}

		_, err = rdb.DecrBy(ctx, RedisKeyWorkspaceResourceRemaining, int64(len(output.Resources))).Result()
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to decrement workspace resource limit: %v", err.Error()))
		}
	}
	for idx, resource := range output.Resources {
		if rdb != nil {
			if remaining <= 0 {
				errs = append(errs, fmt.Sprintf("workspace has reached its max resources limit"))
				break
			}
			remaining--
		}

		if resource.Description == nil {
			continue
		}

		output.Resources[idx].Location = fixAzureLocation(resource.Location)

		azureMetadata := azuremodel.Metadata{
			ID:               resource.ID,
			Name:             resource.Name,
			SubscriptionID:   strings.Join(output.Metadata.SubscriptionIds, ","),
			Location:         resource.Location,
			CloudEnvironment: output.Metadata.CloudEnvironment,
			ResourceType:     strings.ToLower(resource.Type),
			SourceID:         job.SourceID,
		}
		azureMetadataBytes, err := json.Marshal(azureMetadata)
		if err != nil {
			errs = append(errs, fmt.Sprintf("marshal metadata: %v", err.Error()))
			continue
		}
		metadata := make(map[string]string)
		err = json.Unmarshal(azureMetadataBytes, &metadata)
		if err != nil {
			errs = append(errs, fmt.Sprintf("unmarshal metadata: %v", err.Error()))
			continue
		}

		kafkaResource := es.Resource{
			ID:            resource.UniqueID(),
			Name:          resource.Name,
			ResourceGroup: resource.ResourceGroup,
			Location:      resource.Location,
			SourceType:    source.CloudAzure,
			ResourceType:  strings.ToLower(output.Metadata.ResourceType),
			ResourceJobID: job.JobID,
			SourceJobID:   job.ParentJobID,
			SourceID:      job.SourceID,
			ScheduleJobID: job.ScheduleJobID,
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
			SourceID:      job.SourceID,
			ScheduleJobID: job.ScheduleJobID,
			ResourceJobID: job.JobID,
			SourceJobID:   job.ParentJobID,
			CreatedAt:     job.DescribedAt,
			IsCommon:      cloudservice.IsCommonByResourceType(job.ResourceType),
		}
		resourceIDs = append(resourceIDs, resource.UniqueID())
		pluginTableName := steampipe.ExtractTableName(job.ResourceType)
		desc, err := steampipe.ConvertToDescription(job.ResourceType, kafkaResource)
		if err != nil {
			errs = append(errs, fmt.Sprintf("convertToDescription: %v", err.Error()))
			continue
		}
		pluginProvider := steampipe.ExtractPlugin(job.ResourceType)
		var cells map[string]*proto.Column
		if pluginProvider == steampipe.SteampipePluginAzure {
			cells, err = steampipe.AzureDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				errs = append(errs, fmt.Sprintf("azureDescriptionToRecord: %v", err.Error()))
				continue
			}
		} else {
			cells, err = steampipe.AzureADDescriptionToRecord(desc, pluginTableName)
			if err != nil {
				errs = append(errs, fmt.Sprintf("azureADDescriptionToRecord: %v", err.Error()))
				continue
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
			errs = append(errs, fmt.Sprintf("failed to build tags for service: %v", err.Error()))
		}
		lookupResource.Tags = tags

		if rdb != nil {
			for key, value := range tags {
				key = strings.TrimSpace(key)
				_, err = rdb.SAdd(context.Background(), "tag-"+key, value).Result()
				if err != nil {
					errs = append(errs, fmt.Sprintf("failed to push tag into redis: %v", err.Error()))
					continue
				}
				_, err = rdb.Expire(context.Background(), "tag-"+key, 12*time.Hour).Result() //TODO-Saleh set time based on describe interval
				if err != nil {
					errs = append(errs, fmt.Sprintf("failed to set tag expire into redis: %v", err.Error()))
					continue
				}
			}
		}

		msgs = append(msgs, kafkaResource)
		msgs = append(msgs, lookupResource)
	}
	logger.Warn("finished describing azure", zap.String("resourceType", job.ResourceType), zap.Uint("jobID", job.JobID))

	if len(errs) > 0 {
		err = fmt.Errorf("AWS: [%s]", strings.Join(errs, ","))
	} else {
		err = nil
	}
	return msgs, resourceIDs, err
}
