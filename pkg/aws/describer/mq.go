package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mq"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func MQBroker(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := mq.NewFromConfig(cfg)
	paginator := mq.NewListBrokersPaginator(client, &mq.ListBrokersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.BrokerSummaries {
			tags, err := client.ListTags(ctx, &mq.ListTagsInput{
				ResourceArn: v.BrokerArn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *v.BrokerArn,
				Name: *v.BrokerName,
				Description: model.MQBrokerDescription{
					Broker: v,
					Tags:   tags.Tags,
				},
			})
		}
	}

	return values, nil
}
