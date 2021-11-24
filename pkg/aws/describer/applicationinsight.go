package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationinsights"
)

func ApplicationInsightsApplication(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := applicationinsights.NewFromConfig(cfg)
	paginator := applicationinsights.NewListApplicationsPaginator(client, &applicationinsights.ListApplicationsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ApplicationInfoList {
			values = append(values, Resource{
				ID:          *v.ResourceGroupName,
				Description: v,
			})
		}
	}

	return values, nil
}
