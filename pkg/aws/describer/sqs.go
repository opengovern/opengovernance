package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func SQSQueue(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := sqs.NewFromConfig(cfg)
	paginator := sqs.NewListQueuesPaginator(client, &sqs.ListQueuesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, url := range page.QueueUrls {
			// url example: http://sqs.us-west-2.amazonaws.com/123456789012/queueName
			// This prevents Implicit memory aliasing in for loop
			url := url

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

			resource := Resource{
				ARN:  url,
				Name: nameFromArn(url),
				Description: model.SQSQueueDescription{
					Attributes: output.Attributes,
					Tags:       tOutput.Tags,
				},
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
		}
	}

	return values, nil
}
