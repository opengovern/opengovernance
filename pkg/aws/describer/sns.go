package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

func SNSSubscription(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := sns.NewFromConfig(cfg)
	paginator := sns.NewListSubscriptionsPaginator(client, &sns.ListSubscriptionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Subscriptions {
			values = append(values, Resource{
				ARN:         *v.SubscriptionArn,
				Description: v,
			})
		}
	}

	return values, nil
}

type SNSTopicDescription struct {
	Attributes map[string]string
	Tags       []types.Tag
}

func SNSTopic(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := sns.NewFromConfig(cfg)
	paginator := sns.NewListTopicsPaginator(client, &sns.ListTopicsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Topics {
			output, err := client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
				TopicArn: v.TopicArn,
			})
			if err != nil {
				return nil, err
			}

			tOutput, err := client.ListTagsForResource(ctx, &sns.ListTagsForResourceInput{
				ResourceArn: v.TopicArn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN: *v.TopicArn,
				Description: SNSTopicDescription{
					Attributes: output.Attributes,
					Tags:       tOutput.Tags,
				},
			})
		}
	}

	return values, nil
}
