package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/memorydb"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func MemoryDbCluster(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := memorydb.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		clusters, err := client.DescribeClusters(ctx, &memorydb.DescribeClustersInput{
			NextToken: prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, cluster := range clusters.Clusters {
			tags, err := client.ListTags(ctx, &memorydb.ListTagsInput{
				ResourceArn: cluster.ARN,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *cluster.ARN,
				Name: *cluster.Name,
				Description: model.MemoryDbClusterDescription{
					Cluster: cluster,
					Tags:    tags.TagList,
				},
			})
		}

		return clusters.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}
