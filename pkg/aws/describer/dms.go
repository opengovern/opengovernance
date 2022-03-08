package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	dms "github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func DMSReplicationInstance(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := dms.NewFromConfig(cfg)

	paginator := dms.NewDescribeReplicationInstancesPaginator(client,
		&dms.DescribeReplicationInstancesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.ReplicationInstances {
			tags, err := client.ListTagsForResource(ctx, &dms.ListTagsForResourceInput{
				ResourceArn: item.ReplicationInstanceArn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *item.ReplicationInstanceArn,
				Name: *item.ReplicationInstanceIdentifier,
				Description: model.DMSReplicationInstanceDescription{
					ReplicationInstance: item,
					Tags:                tags.TagList,
				},
			})
		}
	}

	return values, nil
}
