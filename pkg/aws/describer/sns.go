package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
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
				ARN:  *v.SubscriptionArn,
				Name: nameFromArn(*v.SubscriptionArn),
				Description: model.SNSSubscriptionDescription{
					Subscription: v,
					Attributes:   output.Attributes,
				},
			})
		}
	}

	return values, nil
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
				ARN:  *v.TopicArn,
				Name: nameFromArn(*v.TopicArn),
				Description: model.SNSTopicDescription{
					Attributes: output.Attributes,
					Tags:       tOutput.Tags,
				},
			})
		}
	}

	return values, nil
}
