package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
)

func CloudTrailTrail(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudtrail.NewFromConfig(cfg)
	paginator := cloudtrail.NewListTrailsPaginator(client, &cloudtrail.ListTrailsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		var trails []string
		for _, trail := range page.Trails {
			if trail.TrailARN != nil {
				trails = append(trails, *trail.TrailARN)
			} else if trail.Name != nil {
				trails = append(trails, *trail.Name)
			} else {
				continue
			}
		}

		output, err := client.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{
			IncludeShadowTrails: aws.Bool(false),
			TrailNameList:       trails,
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.TrailList {
			values = append(values, Resource{
				ARN:         *v.TrailARN,
				Description: v,
			})
		}
	}

	return values, nil
}
