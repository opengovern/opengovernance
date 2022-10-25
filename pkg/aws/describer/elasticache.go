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

func ElastiCacheCluster(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := elasticache.NewFromConfig(cfg)
	paginator := elasticache.NewDescribeCacheClustersPaginator(client, &elasticache.DescribeCacheClustersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if isErr(err, "CacheClusterNotFound") || isErr(err, "InvalidParameterValue") {
				continue
			}
			return nil, err
		}

		for _, cluster := range page.CacheClusters {
			tagsOutput, err := client.ListTagsForResource(ctx, &elasticache.ListTagsForResourceInput{
				ResourceName: cluster.ARN,
			})
			if err != nil {
				if !isErr(err, "CacheClusterNotFound") && !isErr(err, "InvalidParameterValue") {
					return nil, err
				} else {
					tagsOutput = &elasticache.ListTagsForResourceOutput{}
				}
			}

			values = append(values, Resource{
				ARN:  *cluster.ARN,
				Name: *cluster.ARN,
				Description: model.ElastiCacheClusterDescription{
					Cluster: cluster,
					TagList: tagsOutput.TagList,
				},
			})
		}
	}
	return values, nil
}
