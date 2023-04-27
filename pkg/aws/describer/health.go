package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/health"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func HealthEvent(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := health.NewFromConfig(cfg)
	paginator := health.NewDescribeEventsPaginator(client, &health.DescribeEventsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, event := range page.Events {
			resource := Resource{
				ARN: *event.Arn,
				Description: model.HealthEventDescription{
					Event: event,
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
