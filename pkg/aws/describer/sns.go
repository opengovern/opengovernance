package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func SNSSubscription(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := sns.NewFromConfig(cfg)
	paginator := sns.NewListSubscriptionsPaginator(client, &sns.ListSubscriptionsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Subscriptions {
			values = append(values, v)
		}
	}

	return values, nil
}

func SNSTopic(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := sns.NewFromConfig(cfg)
	paginator := sns.NewListTopicsPaginator(client, &sns.ListTopicsInput{})

	var values []interface{}
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

			values = append(values, output.Attributes)
		}
	}

	return values, nil
}

// OMIT: Part of SNSTopic
// func SNSTopicPolicy(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }
