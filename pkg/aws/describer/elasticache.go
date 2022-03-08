package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func ElastiCacheReplicationGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := elasticache.NewFromConfig(cfg)
	paginator := elasticache.NewDescribeReplicationGroupsPaginator(client, &elasticache.DescribeReplicationGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.ReplicationGroups {
			values = append(values, Resource{
				ARN:  *item.ARN,
				Name: *item.ARN,
				Description: model.ElastiCacheReplicationGroupDescription{
					ReplicationGroup: item,
				},
			})
		}
	}
	return values, nil
}
