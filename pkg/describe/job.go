package describe

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gopkg.in/Shopify/sarama.v1"
)

const (
	esIndexHeader = "elasticsearch_index"
	jobTimeout    = 5 * time.Minute
)

var stopWordsRe = regexp.MustCompile(`\W+`)

type AWSAccountCredentials struct {
	AccountId     string
	Regions       []string
	SecretKey     string
	AccessKey     string
	SessionToken  string
	AssumeRoleARN string
}

type AzureSubscriptionCredentials struct {
	SubscriptionId  string
	TenantID        string
	ClientID        string
	ClientSecret    string
	CertificatePath string
	CertificatePass string
	Username        string
	Password        string
}

func IsCredentialsValid(creds []byte, cloud SourceType) bool {
	switch cloud {
	case SourceCloudAWS:
		var v AWSAccountCredentials
		if err := json.Unmarshal(creds, &v); err != nil {
			return false
		}

		return true
	case SourceCloudAzure:
		var v AzureSubscriptionCredentials
		if err := json.Unmarshal(creds, &v); err != nil {
			return false
		}

		return true
	default:
		panic(fmt.Errorf("unsupported cloudtype: %s", cloud))
	}
}

type Job struct {
	JobID               uint // DescribeResourceJob ID
	ParentJobID         uint // DescribeSourceJob ID
	ResourceType        string
	SourceType          SourceType
	DescribeCredentials []byte
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
func (j Job) Do(producer sarama.SyncProducer, topic string) (r JobResult) {
	defer func() {
		if err := recover(); err != nil {
			r = JobResult{
				JobID:       j.JobID,
				ParentJobID: j.ParentJobID,
				Status:      DescribeResourceJobFailed,
				Error:       fmt.Sprintf("paniced: %s", err),
			}
		}
	}()

	// Assume it succeeded unless it fails somewhere
	var (
		status         = DescribeResourceJobSucceeded
		firstErr error = nil
	)

	fail := func(err error) {
		status = DescribeResourceJobFailed
		if firstErr == nil {
			firstErr = err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	resources, err := doDescribe(ctx, j)
	if err != nil {
		// Don't return here. In certain cases, such as AWS, resources might be
		// available for some regions while there was failures in other regions.
		// Instead, continue to write whatever you can to kafka.
		fail(fmt.Errorf("describe resources: %w", err))
	}

	if err := doSendToKafka(producer, topic, resources); err != nil {
		fail(fmt.Errorf("send to kafka: %w", err))
	}

	errMsg := ""
	if firstErr != nil {
		errMsg = firstErr.Error()
	}

	return JobResult{
		JobID:       j.JobID,
		ParentJobID: j.ParentJobID,
		Status:      status,
		Error:       errMsg,
	}
}

// doDescribe describes the sources, e.g. AWS, Azure and returns the responses.
func doDescribe(ctx context.Context, job Job) ([]KafkaResource, error) {
	fmt.Printf("Proccessing Job: ID[%d] ParentJobID[%d] RosourceType[%s]\n", job.JobID, job.ParentJobID, job.ResourceType)

	switch job.SourceType {
	case SourceCloudAWS:
		return doDescribeAWS(ctx, job)
	case SourceCloudAzure:
		return doDescribeAzure(ctx, job)
	default:
		return nil, fmt.Errorf("invalid SourceType: %s", job.SourceType)
	}
}

func doDescribeAWS(ctx context.Context, job Job) ([]KafkaResource, error) {
	var creds AWSAccountCredentials
	if err := json.Unmarshal(job.DescribeCredentials, &creds); err != nil {
		return nil, fmt.Errorf("aws account credentials: %w", err)
	}

	output, err := aws.GetResources(
		ctx,
		job.ResourceType,
		creds.AccountId,
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

	var inventory []KafkaResource
	for region, resources := range output.Resources {
		for _, resource := range resources {
			if resource.Description == nil {
				continue
			}

			inventory = append(inventory, KafkaResource{
				ID:            resource.UniqueID(),
				Description:   resource.Description,
				SourceType:    SourceCloudAWS,
				ResourceType:  output.Metadata.ResourceType,
				ResourceJobID: job.JobID,
				SourceJobID:   job.ParentJobID,
				Metadata: map[string]string{
					"region":     region,
					"account_id": output.Metadata.AccountId,
				},
			})
		}
	}

	// For AWS resources, since they are queries independently per region,
	// if there is an error in some regions, return those errors. For the regions
	// with no error, return the list of resources.
	if len(errs) > 0 {
		err = fmt.Errorf("AWS: [%s]", strings.Join(errs, ","))
	} else {
		err = nil
	}

	return inventory, err
}

func doDescribeAzure(ctx context.Context, job Job) ([]KafkaResource, error) {
	var creds AzureSubscriptionCredentials
	if err := json.Unmarshal(job.DescribeCredentials, &creds); err != nil {
		return nil, fmt.Errorf("azure subscription credentials: %w", err)
	}

	output, err := azure.GetResources(
		ctx,
		job.ResourceType,
		[]string{creds.SubscriptionId},
		creds.TenantID,
		creds.ClientID,
		creds.ClientSecret,
		creds.CertificatePath,
		creds.CertificatePass,
		creds.Username,
		creds.Password,
		string(azure.AuthEnv),
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("Azure: %w", err)
	}

	var inventory []KafkaResource
	for _, resource := range output.Resources {
		if resource.Description == nil {
			continue
		}

		inventory = append(inventory, KafkaResource{
			ID:            resource.UniqueID(),
			Description:   resource.Description,
			SourceType:    SourceCloudAzure,
			ResourceType:  output.Metadata.ResourceType,
			ResourceJobID: job.JobID,
			SourceJobID:   job.ParentJobID,
			Metadata: map[string]string{
				"subscription_id": strings.Join(output.Metadata.SubscriptionIds, ","),
			},
		})
	}

	return inventory, nil
}

type KafkaResource struct {
	// ID is the globally unique ID of the resource.
	ID string `json:"id"`
	// Description is the description of the resource based on the describe call.
	Description interface{} `json:"description"`
	// SourceType is the type of the source of the resource, i.e. AWS Cloud, Azure Cloud.
	SourceType SourceType `json:"source_type"`
	// ResourceType is the type of the resource.
	ResourceType string `json:"resource_type"`
	// ResourceJobID is the DescribeResourceJob ID that described this resource
	ResourceJobID uint `json:"resource_job_id"`
	// SourceJobID is the DescribeSourceJob ID that the ResourceJobID was created for
	SourceJobID uint `json:"source_job_id"`
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

func ResourceTypeToESIndex(t string) string {
	t = stopWordsRe.ReplaceAllString(t, "_")
	return strings.ToLower(t)
}

func doSendToKafka(producer sarama.SyncProducer, topic string, resources []KafkaResource) error {
	var msgs []*sarama.ProducerMessage
	for _, v := range resources {
		msg, err := v.AsProducerMessage()
		if err != nil {
			fmt.Printf("Failed to convert resource[%s] to Kafka ProducerMessage, ignoring...", v.ID)
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
				fmt.Printf("Failed to persist resource[%s] in kafka topic[%s]: %s\nMessage: %v\n", e.Msg.Key, e.Msg.Topic, e.Error(), e.Msg)
			}
		}

		return err
	}

	return nil
}

type JobResult struct {
	JobID       uint
	ParentJobID uint
	Status      DescribeResourceJobStatus
	Error       string
}
