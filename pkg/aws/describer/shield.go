package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/shield"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func ShieldProtectionGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := shield.NewFromConfig(cfg)
	paginator := shield.NewListProtectionGroupsPaginator(client, &shield.ListProtectionGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "ResourceNotFoundException") {
				return nil, err
			}
			continue
		}

		for _, v := range page.ProtectionGroups {
			tags, err := client.ListTagsForResource(ctx, &shield.ListTagsForResourceInput{
				ResourceARN: v.ProtectionGroupArn,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *v.ProtectionGroupArn,
				Name: *v.ProtectionGroupId,
				Description: model.ShieldProtectionGroupDescription{
					ProtectionGroup: v,
					Tags:            tags.Tags,
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
