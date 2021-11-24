package describe

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitlab.com/anil94/golang-aws-inventory/pkg/aws"
	"gitlab.com/anil94/golang-aws-inventory/pkg/azure"
	"gopkg.in/Shopify/sarama.v1"
)

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

	resources, err := doDescribe(j)
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
func doDescribe(job Job) ([]map[string]interface{}, error) {
	fmt.Printf("Proccessing Job: ID[%d] ParentJobID[%d] RosourceType[%s]\n", job.JobID, job.ParentJobID, job.ResourceType)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var inventory []map[string]interface{}
	switch job.SourceType {
	case SourceCloudAWS:
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

		for region, resources := range output.Resources {
			for _, resource := range resources {
				if resource.Description == nil {
					continue
				}

				inventory = append(inventory, map[string]interface{}{
					"ID":                         resource.UniqueID(),
					output.Metadata.ResourceType: resource.Description,
					"SourceType":                 SourceCloudAWS,
					"ResourceType":               output.Metadata.ResourceType,
					"AccountId":                  output.Metadata.AccountId,
					"Region":                     region,
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
	case SourceCloudAzure:
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

		for _, resource := range output.Resources {
			if resource.Description == nil {
				continue
			}

			inventory = append(inventory, map[string]interface{}{
				"ID":                         resource.UniqueID(),
				output.Metadata.ResourceType: resource.Description,
				"SourceType":                 SourceCloudAzure,
				"ResourceType":               output.Metadata.ResourceType,
				"SubscriptionIds":            strings.Join(output.Metadata.SubscriptionIds, ","),
			})
		}
	}

	return inventory, nil
}

func doSendToKafka(producer sarama.SyncProducer, topic string, resources []map[string]interface{}) error {
	var msgs []*sarama.ProducerMessage
	for _, v := range resources {
		value, err := json.Marshal(v)
		if err != nil {
			return err
		}

		id := v["ID"].(string)
		h := sha256.New()
		h.Write([]byte(id))

		msgs = append(msgs, &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(fmt.Sprintf("%x", h.Sum(nil))),
			Value: sarama.ByteEncoder(value),
		})
	}

	if len(msgs) == 0 {
		return nil
	}

	if err := producer.SendMessages(msgs); err != nil {
		if errs, ok := err.(sarama.ProducerErrors); ok {
			for _, e := range errs {
				fmt.Printf("failed to persist resource in kafka: %s\nMessage: %v\n", e.Error(), e.Msg)
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
