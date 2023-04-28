package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func ElastiCacheReplicationGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := elasticache.NewFromConfig(cfg)
	paginator := elasticache.NewDescribeReplicationGroupsPaginator(client, &elasticache.DescribeReplicationGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.ReplicationGroups {
			resource := Resource{
				ARN:  *item.ARN,
				Name: *item.ARN,
				Description: model.ElastiCacheReplicationGroupDescription{
					ReplicationGroup: item,
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

func ElastiCacheCluster(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
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

			resource := Resource{
				ARN:  *cluster.ARN,
				Name: *cluster.ARN,
				Description: model.ElastiCacheClusterDescription{
					Cluster: cluster,
					TagList: tagsOutput.TagList,
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

func GetElastiCacheCluster(ctx context.Context, cfg aws.Config, fields map[string]string) ([]Resource, error) {
	clusterID := fields["id"]
	client := elasticache.NewFromConfig(cfg)
	out, err := client.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
		CacheClusterId: &clusterID,
	})
	if err != nil {
		if isErr(err, "CacheClusterNotFound") || isErr(err, "InvalidParameterValue") {
			return nil, nil
		}
		return nil, err
	}

	var values []Resource
	for _, cluster := range out.CacheClusters {
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
	return values, nil
}

func ElastiCacheParameterGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := elasticache.NewFromConfig(cfg)
	paginator := elasticache.NewDescribeCacheParameterGroupsPaginator(client, &elasticache.DescribeCacheParameterGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cacheParameterGroup := range page.CacheParameterGroups {
			resource := Resource{
				ARN:  *cacheParameterGroup.ARN,
				Name: *cacheParameterGroup.CacheParameterGroupName,
				Description: model.ElastiCacheParameterGroupDescription{
					ParameterGroup: cacheParameterGroup,
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

func ElastiCacheReservedCacheNode(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := elasticache.NewFromConfig(cfg)
	paginator := elasticache.NewDescribeReservedCacheNodesPaginator(client, &elasticache.DescribeReservedCacheNodesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, reservedCacheNode := range page.ReservedCacheNodes {
			resource := Resource{
				ARN: *reservedCacheNode.ReservationARN,
				ID:  *reservedCacheNode.ReservedCacheNodeId,
				Description: model.ElastiCacheReservedCacheNodeDescription{
					ReservedCacheNode: reservedCacheNode,
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

func ElastiCacheSubnetGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := elasticache.NewFromConfig(cfg)
	paginator := elasticache.NewDescribeCacheSubnetGroupsPaginator(client, &elasticache.DescribeCacheSubnetGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cacheSubnetGroup := range page.CacheSubnetGroups {
			resource := Resource{
				ARN:  *cacheSubnetGroup.ARN,
				Name: *cacheSubnetGroup.CacheSubnetGroupName,
				Description: model.ElastiCacheSubnetGroupDescription{
					SubnetGroup: cacheSubnetGroup,
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
