package describe

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/elastic/go-elasticsearch/v7"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
)

const (
	esIndexHeader          = "elasticsearch_index"
	InventorySummaryIndex  = "inventory_summary"
	SourceResourcesSummary = "source_resources_summary"
	describeJobTimeout     = 5 * time.Minute
	cleanupJobTimeout      = 5 * time.Minute
)

type AccountReportType string

const (
	AccountReportTypeResourceGrowthTrend  = "resource_growth_trend"
	AccountReportTypeLocationDistribution = "location_distribution"
	AccountReportTypeLastSummary          = "last_summary"
	AccountReportTypeCompliancyTrend      = "compliancy_trend"
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
	JobID        uint // DescribeResourceJob ID
	ParentJobID  uint // DescribeSourceJob ID
	ResourceType string
	SourceID     string
	DescribedAt  int64
	SourceType   api.SourceType
	ConfigReg    string
}

type DescribeJobResult struct {
	JobID       uint
	ParentJobID uint
	Status      api.DescribeResourceJobStatus
	Error       string
}

// Do will perform the job which includes the following tasks:
//
//    1. Describing resources from the cloud providee based on the job definition.
//    2. Send the described resources to Kafka to be consumed by other systems.
//
// There are a variety of things that could go wrong in the process. This method will
// do its best to complete the task even if some errors occur along the way. However,
// if any error occurs, The JobResult will indicate that through the Status and Error
// will be set to the first error that occured.
func (j DescribeJob) Do(vlt vault.SourceConfig, producer sarama.SyncProducer, topic string, logger *zap.Logger) (r DescribeJobResult) {
	defer func() {
		if err := recover(); err != nil {
			r = DescribeJobResult{
				JobID:       j.JobID,
				ParentJobID: j.ParentJobID,
				Status:      api.DescribeResourceJobFailed,
				Error:       fmt.Sprintf("paniced: %s", err),
			}
		}
	}()

	// Assume it succeeded unless it fails somewhere
	var (
		status         = api.DescribeResourceJobSucceeded
		firstErr error = nil
	)

	fail := func(err error) {
		status = api.DescribeResourceJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), describeJobTimeout)
	defer cancel()

	config, err := vlt.Read(j.ConfigReg)
	if err != nil {
		fail(fmt.Errorf("resource source config: %w", err))
	} else {
		msgs, err := doDescribe(ctx, j, config, logger)
		if err != nil {
			// Don't return here. In certain cases, such as AWS, resources might be
			// available for some regions while there was failures in other regions.
			// Instead, continue to write whatever you can to kafka.
			fail(fmt.Errorf("describe resources: %w", err))
		}

		if err := doSendToKafka(producer, topic, msgs, logger); err != nil {
			fail(fmt.Errorf("send to kafka: %w", err))
		}
	}

	errMsg := ""
	if firstErr != nil {
		errMsg = firstErr.Error()
	}

	return DescribeJobResult{
		JobID:       j.JobID,
		ParentJobID: j.ParentJobID,
		Status:      status,
		Error:       errMsg,
	}
}

// doDescribe describes the sources, e.g. AWS, Azure and returns the responses.
func doDescribe(ctx context.Context, job DescribeJob, config map[string]interface{}, logger *zap.Logger) ([]KafkaMessage, error) {
	logger.Info(fmt.Sprintf("Proccessing Job: ID[%d] ParentJobID[%d] RosourceType[%s]\n", job.JobID, job.ParentJobID, job.ResourceType))

	switch job.SourceType {
	case api.SourceCloudAWS:
		return doDescribeAWS(ctx, job, config)
	case api.SourceCloudAzure:
		return doDescribeAzure(ctx, job, config)
	default:
		return nil, fmt.Errorf("invalid SourceType: %s", job.SourceType)
	}
}

func doDescribeAWS(ctx context.Context, job DescribeJob, config map[string]interface{}) ([]KafkaMessage, error) {
	creds, err := AWSAccountConfigFromMap(config)
	if err != nil {
		return nil, fmt.Errorf("aws account credentials: %w", err)
	}

	output, err := aws.GetResources(
		ctx,
		job.ResourceType,
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

	var msgs []KafkaMessage
	locationDistribution := map[string]int{}
	for _, resources := range output.Resources {
		for _, resource := range resources {
			if resource.Description == nil {
				continue
			}

			msgs = append(msgs, KafkaResource{
				ID:            resource.UniqueID(),
				Description:   resource.Description,
				SourceType:    api.SourceCloudAWS,
				ResourceType:  job.ResourceType,
				ResourceJobID: job.JobID,
				SourceJobID:   job.ParentJobID,
				SourceID:      job.SourceID,
				Metadata: map[string]string{
					"name":       resource.Name,
					"partition":  resource.Partition,
					"region":     resource.Region,
					"account_id": resource.Account,
				},
			})

			msgs = append(msgs, KafkaLookupResource{
				ResourceID:    resource.UniqueID(),
				Name:          resource.Name,
				SourceType:    api.SourceCloudAWS,
				ResourceType:  job.ResourceType,
				ResourceGroup: "",
				Location:      resource.Region,
				SourceID:      job.SourceID,
				ResourceJobID: job.JobID,
				SourceJobID:   job.ParentJobID,
				CreatedAt:     job.DescribedAt,
			})
			region := strings.TrimSpace(resource.Region)
			if region != "" {
				locationDistribution[region]++
			}
		}
	}

	trend := KafkaSourceResourcesSummary{
		SourceID:      job.SourceID,
		SourceType:    job.SourceType,
		SourceJobID:   job.JobID,
		DescribedAt:   job.DescribedAt,
		ResourceCount: len(output.Resources),
		ReportType:    AccountReportTypeResourceGrowthTrend,
	}
	msgs = append(msgs, trend)

	last := KafkaSourceResourcesLastSummary{
		trend,
	}
	last.ReportType = AccountReportTypeLastSummary
	msgs = append(msgs, last)

	locDistribution := KafkaLocationDistributionResource{
		SourceID:             job.SourceID,
		SourceType:           job.SourceType,
		SourceJobID:          job.JobID,
		LocationDistribution: locationDistribution,
		ReportType:           AccountReportTypeLocationDistribution,
	}
	msgs = append(msgs, locDistribution)

	// For AWS resources, since they are queries independently per region,
	// if there is an error in some regions, return those errors. For the regions
	// with no error, return the list of resources.
	if len(errs) > 0 {
		err = fmt.Errorf("AWS: [%s]", strings.Join(errs, ","))
	} else {
		err = nil
	}

	return msgs, nil
}

func doDescribeAzure(ctx context.Context, job DescribeJob, config map[string]interface{}) ([]KafkaMessage, error) {
	creds, err := AzureSubscriptionConfigFromMap(config)
	if err != nil {
		return nil, fmt.Errorf("aure subscription credentials: %w", err)
	}

	output, err := azure.GetResources(
		ctx,
		job.ResourceType,
		[]string{creds.SubscriptionID},
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
		return nil, fmt.Errorf("Azure: %w", err)
	}

	var msgs []KafkaMessage
	locationDistribution := map[string]int{}
	for _, resource := range output.Resources {
		if resource.Description == nil {
			continue
		}

		msgs = append(msgs, KafkaResource{
			ID:            resource.UniqueID(),
			Description:   resource.Description,
			SourceType:    api.SourceCloudAzure,
			ResourceType:  output.Metadata.ResourceType,
			ResourceJobID: job.JobID,
			SourceJobID:   job.ParentJobID,
			SourceID:      job.SourceID,
			Metadata: map[string]string{
				"id":                resource.ID,
				"name":              resource.Name,
				"subscription_id":   strings.Join(output.Metadata.SubscriptionIds, ","),
				"location":          resource.Location,
				"cloud_environment": output.Metadata.CloudEnvironment,
			},
		})

		msgs = append(msgs, KafkaLookupResource{
			ResourceID:    resource.UniqueID(),
			Name:          resource.Name,
			SourceType:    api.SourceCloudAzure,
			ResourceType:  job.ResourceType,
			ResourceGroup: resource.ResourceGroup,
			Location:      resource.Location,
			SourceID:      job.SourceID,
			ResourceJobID: job.JobID,
			SourceJobID:   job.ParentJobID,
			CreatedAt:     job.DescribedAt,
		})
		location := strings.TrimSpace(resource.Location)
		if location != "" {
			locationDistribution[location]++
		}
	}
	trend := KafkaSourceResourcesSummary{
		SourceID:      job.SourceID,
		SourceType:    job.SourceType,
		SourceJobID:   job.JobID,
		DescribedAt:   job.DescribedAt,
		ResourceCount: len(output.Resources),
	}
	msgs = append(msgs, trend)

	last := KafkaSourceResourcesLastSummary{
		trend,
	}
	last.ReportType = AccountReportTypeLastSummary
	msgs = append(msgs, last)

	locDistribution := KafkaLocationDistributionResource{
		SourceID:             job.SourceID,
		SourceType:           job.SourceType,
		SourceJobID:          job.JobID,
		LocationDistribution: locationDistribution,
	}
	msgs = append(msgs, locDistribution)

	return msgs, nil
}

type KafkaMessage interface {
	AsProducerMessage() (*sarama.ProducerMessage, error)
	MessageID() string
}

type KafkaResource struct {
	// ID is the globally unique ID of the resource.
	ID string `json:"id"`
	// Description is the description of the resource based on the describe call.
	Description interface{} `json:"description"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType api.SourceType `json:"source_type"`
	// ResourceType is the type of the resource.
	ResourceType string `json:"resource_type"`
	// ResourceJobID is the DescribeResourceJob ID that described this resource
	ResourceJobID uint `json:"resource_job_id"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// SourceID is the Source ID that the resource belongs to
	SourceID string `json:"source_id"`
	// Metadata is arbitrary data associated with each resource
	Metadata map[string]string `json:"metadata"`
}

func (r KafkaResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(r.ID))

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(fmt.Sprintf("%x", h.Sum(nil))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(ResourceTypeToESIndex(r.ResourceType)),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}
func (r KafkaResource) MessageID() string {
	return r.ID
}

type KafkaLookupResource struct {
	// ResourceID is the globally unique ID of the resource.
	ResourceID string `json:"resource_id"`
	// Name is the name of the resource.
	Name string `json:"name"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType api.SourceType `json:"source_type"`
	// ResourceType is the type of the resource.
	ResourceType string `json:"resource_type"`
	// ResourceGroup is the group of resource (Azure only)
	ResourceGroup string `json:"resource_group"`
	// Location is location/region of the resource
	Location string `json:"location"`
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// ResourceJobID is the DescribeResourceJob ID that described this resource
	ResourceJobID uint `json:"resource_job_id"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// CreatedAt is when the DescribeSourceJob is created
	CreatedAt int64 `json:"created_at"`
}

func (r KafkaLookupResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(r.ResourceID))
	h.Write([]byte(r.SourceType))

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(fmt.Sprintf("%x", h.Sum(nil))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(InventorySummaryIndex),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}
func (r KafkaLookupResource) MessageID() string {
	return r.ResourceID
}

type KafkaSourceResourcesSummary struct {
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType api.SourceType `json:"source_type"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// DescribedAt is when the DescribeSourceJob is created
	DescribedAt int64 `json:"described_at"`
	// ResourceCount is total of resources for specified account
	ResourceCount int `json:"resource_count"`
	// ReportType of document
	ReportType AccountReportType `json:"report_type"`
}

func (r KafkaSourceResourcesSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(uuid.New().String()),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummary),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}
func (r KafkaSourceResourcesSummary) MessageID() string {
	return r.SourceID
}

type KafkaSourceResourcesLastSummary struct {
	KafkaSourceResourcesSummary
}

func (r KafkaSourceResourcesLastSummary) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(r.SourceID))
	h.Write([]byte(r.ReportType))

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(fmt.Sprintf("%x", h.Sum(nil))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummary),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}

type KafkaLocationDistributionResource struct {
	// SourceID is aws account id or azure subscription id
	SourceID string `json:"source_id"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType api.SourceType `json:"source_type"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
	// LocationDistribution is total of resources per location for specified account
	LocationDistribution map[string]int `json:"location_distribution"`
	// ReportType of document
	ReportType AccountReportType `json:"report_type"`
}

func (r KafkaLocationDistributionResource) AsProducerMessage() (*sarama.ProducerMessage, error) {
	value, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(r.SourceID))
	h.Write([]byte(r.ReportType))

	return &sarama.ProducerMessage{
		Key: sarama.StringEncoder(fmt.Sprintf("%x", h.Sum(nil))),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte(esIndexHeader),
				Value: []byte(SourceResourcesSummary),
			},
		},
		Value: sarama.ByteEncoder(value),
	}, nil
}
func (r KafkaLocationDistributionResource) MessageID() string {
	return r.SourceID
}

func ResourceTypeToESIndex(t string) string {
	t = stopWordsRe.ReplaceAllString(t, "_")
	return strings.ToLower(t)
}

func doSendToKafka(producer sarama.SyncProducer, topic string, kafkaMsgs []KafkaMessage, logger *zap.Logger) error {
	var msgs []*sarama.ProducerMessage
	for _, v := range kafkaMsgs {
		msg, err := v.AsProducerMessage()
		if err != nil {
			logger.Error("Failed calling AsProducerMessage", zap.Error(fmt.Errorf("Failed to convert msg[%s] to Kafka ProducerMessage, ignoring...", v.MessageID())))
			continue
		}

		// Override the topic
		msg.Topic = topic

		msgs = append(msgs, msg)
	}

	if len(msgs) == 0 {
		return nil
	}

	if err := producer.SendMessages(msgs); err != nil {
		if errs, ok := err.(sarama.ProducerErrors); ok {
			for _, e := range errs {
				logger.Error("Falied calling SendMessages", zap.Error(fmt.Errorf("Failed to persist resource[%s] in kafka topic[%s]: %s\nMessage: %v\n", e.Msg.Key, e.Msg.Topic, e.Error(), e.Msg)))
			}
		}

		return err
	}

	return nil
}

type DescribeCleanupJob struct {
	JobID        uint // DescribeResourceJob ID
	ResourceType string
}

func (j DescribeCleanupJob) Do(esClient *elasticsearch.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupJobTimeout)
	defer cancel()

	rIndex := ResourceTypeToESIndex(j.ResourceType)
	fmt.Printf("Cleaning resources with resource_job_id of %d from index %s\n", j.JobID, rIndex)

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"resource_job_id": j.JobID,
			},
		},
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
		return err
	}

	if len(resp.Failures) != 0 {
		body, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		return fmt.Errorf("elasticsearch: delete by query: %s", string(body))
	}

	fmt.Printf("Successfully delete %d resources of type %s\n", resp.Deleted, j.ResourceType)
	return nil
}
