package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/neptune"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func NeptuneDatabase(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := neptune.NewFromConfig(cfg)
	paginator := neptune.NewDescribeDBInstancesPaginator(client, &neptune.DescribeDBInstancesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DBInstances {
			tags, err := client.ListTagsForResource(ctx, &neptune.ListTagsForResourceInput{
				ResourceName: v.DBInstanceArn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *v.DBInstanceArn,
				Name: *v.DBClusterIdentifier,
				Description: model.NeptuneDatabaseDescription{
					Database: v,
					Tags:     tags.TagList,
				},
			})
		}
	}

	return values, nil
}
