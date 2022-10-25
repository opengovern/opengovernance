package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/aws/aws-sdk-go/aws/awserr"
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
			return nil, err
		}

		for _, cluster := range page.CacheClusters {
			var tags []types.Tag = nil
			tagsOutput, err := client.ListTagsForResource(ctx, &elasticache.ListTagsForResourceInput{
				ResourceName: cluster.ARN,
			})
			if err != nil {
				if a, ok := err.(awserr.Error); ok {
					if a.Code() != "CacheClusterNotFound" {
						return nil, err
					}
				} else {
					return nil, err
				}
			}
			if tagsOutput != nil {
				tags = tagsOutput.TagList
			}

			values = append(values, Resource{
				ARN:  *cluster.ARN,
				Name: *cluster.ARN,
				Description: model.ElastiCacheClusterDescription{
					Cluster: cluster,
					TagList: tags,
				},
			})
		}
	}
	return values, nil
}
