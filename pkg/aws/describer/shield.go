package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/shield"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func ShieldProtectionGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := shield.NewFromConfig(cfg)
	paginator := shield.NewListProtectionGroupsPaginator(client, &shield.ListProtectionGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ProtectionGroups {
			tags, err := client.ListTagsForResource(ctx, &shield.ListTagsForResourceInput{
				ResourceARN: v.ProtectionGroupArn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *v.ProtectionGroupArn,
				Name: *v.ProtectionGroupId,
				Description: model.ShieldProtectionGroupDescription{
					ProtectionGroup: v,
					Tags:            tags.Tags,
				},
			})
		}
	}

	return values, nil
}
