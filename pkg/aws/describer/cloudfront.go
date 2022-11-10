package describer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

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
				Name: *item.Id,
				Description: model.CloudFrontDistributionDescription{
					Distribution: distribution.Distribution,
					ETag:         distribution.ETag,
					Tags:         tags.Tags.Items,
				},
			})
		}
	}

	return values, nil
}

func CloudFrontOriginAccessControl(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := cloudfront.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.ListOriginAccessControls(ctx, &cloudfront.ListOriginAccessControlsInput{
			Marker:   prevToken,
			MaxItems: aws.Int32(100),
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.OriginAccessControlList.Items {
			arn := fmt.Sprintf("arn:%s:cloudfront::%s:origin-access-control/%s", describeCtx.Partition, describeCtx.AccountID, *v.Id) //TODO: this is fake ARN, find out the real one's format
			tags, err := client.ListTagsForResource(ctx, &cloudfront.ListTagsForResourceInput{
				Resource: &arn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  arn,
				Name: *v.Id,
				Description: model.CloudFrontOriginAccessControlDescription{
					OriginAccessControl: v,
					Tags:                tags.Tags.Items,
				},
			})
		}
		return output.OriginAccessControlList.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}
