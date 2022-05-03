package describe

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"

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
	InventorySummaryIndex  = "inventory_summary"
	SourceResourcesSummary = "source_resources_summary"
	describeJobTimeout     = 5 * time.Minute
	cleanupJobTimeout      = 5 * time.Minute
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

		if err := kafka.DoSendToKafka(producer, topic, msgs, logger); err != nil {
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
func doDescribe(ctx context.Context, job DescribeJob, config map[string]interface{}, logger *zap.Logger) ([]kafka.KafkaMessage, error) {
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

func doDescribeAWS(ctx context.Context, job DescribeJob, config map[string]interface{}) ([]kafka.KafkaMessage, error) {
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

	var msgs []kafka.KafkaMessage
	locationDistribution := map[string]int{}
	for _, resources := range output.Resources {
		for _, resource := range resources {
			if resource.Description == nil {
				continue
			}

			msgs = append(msgs, kafka.KafkaResource{
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

			msgs = append(msgs, kafka.KafkaLookupResource{
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

	trend := kafka.KafkaSourceResourcesSummary{
		SourceID:      job.SourceID,
		SourceType:    job.SourceType,
		SourceJobID:   job.JobID,
		DescribedAt:   job.DescribedAt,
		ResourceCount: len(output.Resources),
		ReportType:    kafka.ResourceSummaryTypeResourceGrowthTrend,
	}
	msgs = append(msgs, trend)

	last := kafka.KafkaSourceResourcesLastSummary{
		trend,
	}
	last.ReportType = kafka.ResourceSummaryTypeLastSummary
	msgs = append(msgs, last)

	locDistribution := kafka.KafkaLocationDistributionResource{
		SourceID:             job.SourceID,
		SourceType:           job.SourceType,
		SourceJobID:          job.JobID,
		LocationDistribution: locationDistribution,
		ReportType:           kafka.ResourceSummaryTypeLocationDistribution,
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

func doDescribeAzure(ctx context.Context, job DescribeJob, config map[string]interface{}) ([]kafka.KafkaMessage, error) {
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

	var msgs []kafka.KafkaMessage
	locationDistribution := map[string]int{}
	for _, resource := range output.Resources {
		if resource.Description == nil {
			continue
		}

		msgs = append(msgs, kafka.KafkaResource{
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

		msgs = append(msgs, kafka.KafkaLookupResource{
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
	trend := kafka.KafkaSourceResourcesSummary{
		SourceID:      job.SourceID,
		SourceType:    job.SourceType,
		SourceJobID:   job.JobID,
		DescribedAt:   job.DescribedAt,
		ResourceCount: len(output.Resources),
	}
	msgs = append(msgs, trend)

	last := kafka.KafkaSourceResourcesLastSummary{
		trend,
	}
	last.ReportType = kafka.ResourceSummaryTypeLastSummary
	msgs = append(msgs, last)

	locDistribution := kafka.KafkaLocationDistributionResource{
		SourceID:             job.SourceID,
		SourceType:           job.SourceType,
		SourceJobID:          job.JobID,
		LocationDistribution: locationDistribution,
	}
	msgs = append(msgs, locDistribution)

	return msgs, nil
}

func ResourceTypeToESIndex(t string) string {
	t = stopWordsRe.ReplaceAllString(t, "_")
	return strings.ToLower(t)
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
