package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func EventBridgeBus(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := eventbridge.NewFromConfig(cfg)

	input := eventbridge.ListEventBusesInput{Limit: aws.Int32(100)}

	var values []Resource
	for {
		response, err := client.ListEventBuses(ctx, &input)
		if err != nil {
			return nil, err
		}
		for _, bus := range response.EventBuses {
			tagsOutput, err := client.ListTagsForResource(ctx, &eventbridge.ListTagsForResourceInput{
				ResourceARN: bus.Arn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *bus.Arn,
				Name: *bus.Name,
				Description: model.EventBridgeBusDescription{
					Bus:  bus,
					Tags: tagsOutput.Tags,
				},
			})
		}
		if response.NextToken == nil {
			break
		}
		input.NextToken = response.NextToken
	}

	return values, nil
}
