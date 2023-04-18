package describe

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	awsmodel "gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
	azuremodel "gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	trace2 "gitlab.com/keibiengine/keibi-engine/pkg/trace"
	"go.opentelemetry.io/otel"

	"github.com/turbot/steampipe-plugin-sdk/v4/grpc/proto"

	"github.com/go-redis/redis/v8"

	"gitlab.com/keibiengine/keibi-engine/pkg/steampipe"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"

	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/go-errors/errors"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
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

var DoDescribeCleanupJobsCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: "keibi",
	Subsystem: "describe_cleanup_worker",
	Name:      "do_describe_cleanup_jobs_total",
	Help:      "Count of done describe cleanup jobs in describe-worker service",
}, []string{"resource_type", "status"})

var DoDescribeCleanupJobsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "keibi",
	Subsystem: "describe_cleanup_worker",
	Name:      "do_describe_cleanup_jobs_duration_seconds",
	Help:      "Duration of done describe cleanup jobs in describe-worker service",
	Buckets:   []float64{5, 60, 300, 600, 1800, 3600, 7200, 36000},
}, []string{"resource_type", "status"})

const (
	InventorySummaryIndex = "inventory_summary"
	describeJobTimeout    = 3 * 60 * time.Minute
	cleanupJobTimeout     = 5 * time.Minute
)

var stopWordsRe = regexp.MustCompile(`\W+`)

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
	ConfigReg     string
	TriggerType   enums.DescribeTriggerType
	RetryCounter  uint
}

type DescribeJobResult struct {
	JobID       uint
	ParentJobID uint
	Status      api.DescribeResourceJobStatus
	Error       string
	DescribeJob DescribeJob
}

type DescribeConnectionJob struct {
	JobID        uint            // DescribeSourceJob ID
	ResourceJobs map[uint]string // DescribeResourceJob ID -> ResourceType
	SourceID     string
	AccountID    string
	DescribedAt  int64
	SourceType   api.SourceType
	ConfigReg    string
	TriggerType  enums.DescribeTriggerType
}

type DescribeConnectionJobResult struct {
	JobID  uint
	Result map[uint]DescribeJobResult // DescribeResourceJob ID -> DescribeJobResult
}

func (j DescribeConnectionJob) Do(ictx context.Context, vlt vault.SourceConfig, rdb *redis.Client, producer sarama.SyncProducer, topic string, logger *zap.Logger) (r DescribeConnectionJobResult) {
	if j.TriggerType == "" {
		j.TriggerType = enums.DescribeTriggerTypeScheduled
	}
	workerCount, err := strconv.Atoi(AccountConcurrentDescribe)
	if err != nil {
		fmt.Println("Invalid worker count:", AccountConcurrentDescribe, err)
	}

	if workerCount < 1 {
		workerCount = 1
	}

	workChannel := make(chan DescribeJob, workerCount*2)
	resultChannel := make(chan DescribeJobResult, len(j.ResourceJobs)*2)
	doneChannel := make(chan struct{}, workerCount*2)
	defer close(doneChannel)
	defer close(resultChannel)
	defer close(workChannel)

	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			for {
				select {
				case <-doneChannel:
					wg.Done()
					return
				case job := <-workChannel:
					res := job.Do(ictx, vlt, rdb, producer, topic, logger)

					resultChannel <- res
				}
			}
		}()
	}

	for id, resourceType := range j.ResourceJobs {
		workChannel <- DescribeJob{
			JobID:         id,
			ScheduleJobID: j.ScheduleJobID,
			ParentJobID:   j.JobID,
			ResourceType:  resourceType,
			SourceID:      j.SourceID,
			AccountID:     j.AccountID,
			DescribedAt:   j.DescribedAt,
			SourceType:    j.SourceType,
			ConfigReg:     j.ConfigReg,
			TriggerType:   j.TriggerType,
			RetryCounter:  0,
		}
	}

	result := map[uint]DescribeJobResult{}
	for range j.ResourceJobs {
		res := <-resultChannel
		result[res.JobID] = res
	}

	for i := 0; i < workerCount; i++ {
		doneChannel <- struct{}{}
	}
	wg.Wait()

	return DescribeConnectionJobResult{
		JobID:  j.JobID,
		Result: result,
	}
}

func (j DescribeConnectionJob) CloudTimeout() (r DescribeConnectionJobResult) {
	describeConnectionJobResult := DescribeConnectionJobResult{
		JobID:  j.JobID,
		Result: map[uint]DescribeJobResult{},
	}
	for id, resourceType := range j.ResourceJobs {
		dj := DescribeJob{
			JobID:         id,
			ScheduleJobID: j.ScheduleJobID,
			ParentJobID:   j.JobID,
			ResourceType:  resourceType,
			SourceID:      j.SourceID,
			AccountID:     j.AccountID,
			DescribedAt:   j.DescribedAt,
			SourceType:    j.SourceType,
			ConfigReg:     j.ConfigReg,
			TriggerType:   j.TriggerType,
			RetryCounter:  0,
		}
		describeConnectionJobResult.Result[id] = DescribeJobResult{
			JobID:       dj.JobID,
			ParentJobID: dj.ParentJobID,
			Status:      api.DescribeResourceJobCloudTimeout,
			Error:       "Cloud job timed out",
			DescribeJob: dj,
		}
	}
	return describeConnectionJobResult
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
func (j DescribeJob) Do(ictx context.Context, vlt vault.SourceConfig, rdb *redis.Client, producer sarama.SyncProducer, topic string, logger *zap.Logger) (r DescribeJobResult) {
	logger.Info("Starting DescribeJob", zap.Uint("jobID", j.JobID), zap.Uint("scheduleJobID", j.ScheduleJobID), zap.Uint("parentJobID", j.ParentJobID), zap.String("resourceType", j.ResourceType), zap.String("sourceID", j.SourceID), zap.String("accountID", j.AccountID), zap.Int64("describedAt", j.DescribedAt), zap.String("sourceType", string(j.SourceType)), zap.String("configReg", j.ConfigReg), zap.String("triggerType", string(j.TriggerType)), zap.Uint("retryCounter", j.RetryCounter))
	ctx, span := otel.Tracer(trace2.DescribeWorkerTrace).Start(ictx, "Do")
	defer span.End()

	startTime := time.Now().Unix()
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("paniced with error:", err)
			fmt.Println(errors.Wrap(err, 2).ErrorStack())
			DoDescribeJobsDuration.WithLabelValues(string(j.SourceType), j.ResourceType, "failure").Observe(float64(time.Now().Unix() - startTime))
			DoDescribeJobsCount.WithLabelValues(string(j.SourceType), j.ResourceType, "failure").Inc()
			r = DescribeJobResult{
				JobID:       j.JobID,
				ParentJobID: j.ParentJobID,
				Status:      api.DescribeResourceJobFailed,
				Error:       fmt.Sprintf("paniced: %s", err),
				DescribeJob: j,
			}
		}
	}()

	// Assume it succeeded unless it fails somewhere
	var (
		status         = api.DescribeResourceJobSucceeded
		firstErr error = nil
	)

	fail := func(err error) {
		DoDescribeJobsDuration.WithLabelValues(string(j.SourceType), j.ResourceType, "failure").Observe(float64(time.Now().Unix() - startTime))
		DoDescribeJobsCount.WithLabelValues(string(j.SourceType), j.ResourceType, "failure").Inc()
		status = api.DescribeResourceJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}

	ctx, cancel := context.WithTimeout(ctx, describeJobTimeout)
	defer cancel()

	config, err := vlt.Read(j.ConfigReg)
	if err != nil {
		fail(fmt.Errorf("resource source config: %w", err))
	} else if config == nil {
		fail(fmt.Errorf("config is null! path is: %s", j.ConfigReg))
	} else {
		msgs, err := doDescribe(ctx, rdb, j, config, logger)
		if err != nil {
			// Don't return here. In certain cases, such as AWS, resources might be
			// available for some regions while there was failures in other regions.
			// Instead, continue to write whatever you can to kafka.
			fail(fmt.Errorf("describe resources: %w", err))
		}

		if len(msgs) > 0 {
			if err := kafka.DoSend(producer, topic, msgs, logger); err != nil {
				fail(fmt.Errorf("send to kafka: %w", err))
			} else {
				status = api.DescribeResourceJobSucceeded
			}
		}
	}

	errMsg := ""
	if firstErr != nil {
		errMsg = firstErr.Error()
	}
	if status == api.DescribeResourceJobSucceeded {
		DoDescribeJobsDuration.WithLabelValues(string(j.SourceType), j.ResourceType, "successful").Observe(float64(time.Now().Unix() - startTime))
		DoDescribeJobsCount.WithLabelValues(string(j.SourceType), j.ResourceType, "successful").Inc()
	}

	return DescribeJobResult{
		JobID:       j.JobID,
		ParentJobID: j.ParentJobID,
		Status:      status,
		Error:       errMsg,
		DescribeJob: j,
	}
}

// doDescribe describes the sources, e.g. AWS, Azure and returns the responses.
func doDescribe(ctx context.Context, rdb *redis.Client, job DescribeJob, config map[string]interface{}, logger *zap.Logger) ([]kafka.Doc, error) {
	logger.Info(fmt.Sprintf("Proccessing Job: ID[%d] ParentJobID[%d] RosourceType[%s]\n", job.JobID, job.ParentJobID, job.ResourceType))

	switch job.SourceType {
	case api.SourceCloudAWS:
		return doDescribeAWS(ctx, rdb, job, config, logger)
	case api.SourceCloudAzure:
		return doDescribeAzure(ctx, rdb, job, config, logger)
	default:
		return nil, fmt.Errorf("invalid SourceType: %s", job.SourceType)
	}
}

func doDescribeAWS(ctx context.Context, rdb *redis.Client, job DescribeJob, config map[string]interface{}, logger *zap.Logger) ([]kafka.Doc, error) {
	creds, err := AWSAccountConfigFromMap(config)
	if err != nil {
		return nil, fmt.Errorf("aws account credentials: %w", err)
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
	)
	if err != nil {
		return nil, fmt.Errorf("AWS: %w", err)
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

	return msgs, err
}

func doDescribeAzure(ctx context.Context, rdb *redis.Client, job DescribeJob, config map[string]interface{}, logger *zap.Logger) ([]kafka.Doc, error) {
	logger.Warn("starting to describe azure subscription", zap.String("resourceType", job.ResourceType), zap.Uint("jobID", job.JobID))
	creds, err := AzureSubscriptionConfigFromMap(config)
	if err != nil {
		return nil, fmt.Errorf("azure subscription credentials: %w", err)
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
	)
	if err != nil {
		return nil, fmt.Errorf("azure: %w", err)
	}
	logger.Warn("got the resources, finding summaries", zap.String("resourceType", job.ResourceType), zap.Uint("jobID", job.JobID))

	var msgs []kafka.Doc
	var errs []string
	var remaining int64 = MAX_INT64

	if rdb != nil {
		currentResourceLimitRemaining, err := rdb.Get(ctx, RedisKeyWorkspaceResourceRemaining).Result()
		if err != nil {
			return nil, fmt.Errorf("redisGet: %v", err.Error())
		}
		remaining, err = strconv.ParseInt(currentResourceLimitRemaining, 10, 64)
		if remaining <= 0 {
			return nil, fmt.Errorf("workspace has reached its max resources limit")
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
	return msgs, err
}

func ResourceTypeToESIndex(t string) string {
	t = stopWordsRe.ReplaceAllString(t, "_")
	return strings.ToLower(t)
}

type DescribeCleanupJobType string

const (
	DescribeCleanupJobTypeInclusiveDelete DescribeCleanupJobType = "inclusive_delete"
	DescribeCleanupJobTypeExclusiveDelete DescribeCleanupJobType = "exclusive_delete"
)

type DescribeCleanupJob struct {
	JobType      DescribeCleanupJobType `json:"job_type"`
	ResourceType string                 `json:"resource_type"`
	JobIDs       []uint                 `json:"job_id"` // DescribeResourceJob ID
}

func (j DescribeCleanupJob) Do(esClient *elasticsearch.Client) error {
	startTime := time.Now().Unix()
	ctx, cancel := context.WithTimeout(context.Background(), cleanupJobTimeout)
	defer cancel()

	rIndex := ResourceTypeToESIndex(j.ResourceType)

	if j.JobIDs == nil || len(j.JobIDs) == 0 {
		return nil
	}

	var query map[string]any
	switch j.JobType {
	case DescribeCleanupJobTypeInclusiveDelete:
		fmt.Printf("Cleaning resources with resource_job_id of %v from index %s inclusivly\n", j.JobIDs, rIndex)
		query = map[string]any{
			"query": map[string]any{
				"bool": map[string]any{
					"filter": []any{
						map[string]any{
							"terms": map[string]any{
								"resource_job_id": j.JobIDs,
							},
						},
						map[string]any{
							"term": map[string]any{
								"resource_type": strings.ToLower(j.ResourceType),
							},
						},
					},
				},
			},
		}
	case DescribeCleanupJobTypeExclusiveDelete:
		fmt.Printf("Cleaning resources with resource_job_id of %v from index %s exclusivly\n", j.JobIDs, rIndex)
		query = map[string]any{
			"query": map[string]any{
				"bool": map[string]any{
					"must_not": []any{
						map[string]any{
							"terms": map[string]any{
								"resource_job_id": j.JobIDs,
							},
						},
					},
					"filter": []any{
						map[string]any{
							"term": map[string]any{
								"resource_type": strings.ToLower(j.ResourceType),
							},
						},
					},
				},
			},
		}
	}

	// Delete the resources from both inventory_summary and resource specific index
	indices := []string{
		rIndex,
		InventorySummaryIndex,
	}

	resp, err := keibi.DeleteByQuery(ctx, esClient, indices, query,
		esClient.DeleteByQuery.WithRefresh(true),
		esClient.DeleteByQuery.WithConflicts("proceed"),
	)
	if err != nil {
		DoDescribeCleanupJobsDuration.WithLabelValues(j.ResourceType, "failure").Observe(float64(time.Now().Unix() - startTime))
		DoDescribeCleanupJobsCount.WithLabelValues(j.ResourceType, "failure").Inc()
		return err
	}

	if len(resp.Failures) != 0 {
		body, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		DoDescribeCleanupJobsDuration.WithLabelValues(j.ResourceType, "failure").Observe(float64(time.Now().Unix() - startTime))
		DoDescribeCleanupJobsCount.WithLabelValues(j.ResourceType, "failure").Inc()
		fmt.Printf("Failed to delete %d resources of type %s with error: %s\n", resp.Deleted, j.ResourceType, string(body))
		return fmt.Errorf("elasticsearch: delete by query: %s", string(body))
	}

	fmt.Printf("Successfully delete %d resources of type %s\n", resp.Deleted, j.ResourceType)
	DoDescribeCleanupJobsDuration.WithLabelValues(j.ResourceType, "successful").Observe(float64(time.Now().Unix() - startTime))
	DoDescribeCleanupJobsCount.WithLabelValues(j.ResourceType, "successful").Inc()
	return nil
}
