package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

type CloudFrontDistributionDescription struct {
	Distribution *types.Distribution
	ETag         *string
	Tags         []types.Tag
}

func CloudFrontDistribution(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudfront.NewFromConfig(cfg)
	paginator := cloudfront.NewListDistributionsPaginator(client, &cloudfront.ListDistributionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.DistributionList.Items {
			tags, err := client.ListTagsForResource(ctx, &cloudfront.ListTagsForResourceInput{
				Resource: item.ARN,
			})
			if err != nil {
				return nil, err
			}

			distribution, err := client.GetDistribution(ctx, &cloudfront.GetDistributionInput{
				Id: item.Id,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *item.ARN,
				Name: *item.DomainName,
				Description: CloudFrontDistributionDescription{
					Distribution: distribution.Distribution,
					ETag:         distribution.ETag,
					Tags:         tags.Tags.Items,
				},
			})
		}
	}

	return values, nil
}
