package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/fms"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func FMSPolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := fms.NewFromConfig(cfg)
	paginator := fms.NewListPoliciesPaginator(client, &fms.ListPoliciesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.PolicyList {
			tags, err := client.ListTagsForResource(ctx, &fms.ListTagsForResourceInput{
				ResourceArn: v.PolicyArn,
			})
			if err != nil {
				return nil, err
			}
			values = append(values, Resource{
				ARN:  *v.PolicyArn,
				Name: *v.PolicyName,
				Description: model.FMSPolicyDescription{
					Policy: v,
					Tags:   tags.TagList,
				},
			})
		}
	}

	return values, nil
}
