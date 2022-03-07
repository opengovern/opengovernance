package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSQueueDescription struct {
	Attributes map[string]string
	Tags       map[string]string
}

func SQSQueue(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := sqs.NewFromConfig(cfg)
	paginator := sqs.NewListQueuesPaginator(client, &sqs.ListQueuesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, url := range page.QueueUrls {
			output, err := client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
				QueueUrl: &url,
				AttributeNames: []types.QueueAttributeName{
					types.QueueAttributeNameAll,
				},
			})
			if err != nil {
				return nil, err
			}

			tOutput, err := client.ListQueueTags(ctx, &sqs.ListQueueTagsInput{
				QueueUrl: &url,
			})
			if err != nil {
				return nil, err
			}

			// Add Queue URL since it doesn't exists in the description
			output.Attributes["QueueUrl"] = url

			values = append(values, Resource{
				ARN:  url,
				Name: url,
				Description: SQSQueueDescription{
					Attributes: output.Attributes,
					Tags:       tOutput.Tags,
				},
			})
		}
	}

	return values, nil
}
