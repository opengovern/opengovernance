package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func SQSQueue(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := sqs.NewFromConfig(cfg)
	paginator := sqs.NewListQueuesPaginator(client, &sqs.ListQueuesInput{})

	var values []interface{}
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

			values = append(values, output.Attributes)
		}
	}

	return values, nil
}

// OMIT: Part of SQSQueue
// func SQSQueuePolicy(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }
