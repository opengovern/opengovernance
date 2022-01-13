package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type SNSSubscriptionDescription struct {
	Subscription types.Subscription
	Attributes   map[string]string
}

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
			output, err := client.GetSubscriptionAttributes(ctx, &sns.GetSubscriptionAttributesInput{
				SubscriptionArn: v.SubscriptionArn,
			})
			if err != nil {
				if !isErr(err, "NotFound") {
					return nil, err
				}

				output = &sns.GetSubscriptionAttributesOutput{}
			}

			values = append(values, Resource{
				ARN: *v.SubscriptionArn,
				Description: SNSSubscriptionDescription{
					Subscription: v,
					Attributes:   output.Attributes,
				},
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
