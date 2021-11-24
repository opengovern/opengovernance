package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationinsights"
)

func ApplicationInsightsApplication(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := applicationinsights.NewFromConfig(cfg)
	paginator := applicationinsights.NewListApplicationsPaginator(client, &applicationinsights.ListApplicationsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ApplicationInfoList {
			values = append(values, v)
		}
	}

	return values, nil
}
