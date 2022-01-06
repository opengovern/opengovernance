package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dax"
	"github.com/aws/aws-sdk-go-v2/service/dax/types"
)

type DAXClusterDescription struct {
	Cluster types.Cluster
	Tags    []types.Tag
}

func DAXCluster(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := dax.NewFromConfig(cfg)
	out, err := client.DescribeClusters(ctx, &dax.DescribeClustersInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, cluster := range out.Clusters {
		tags, err := client.ListTags(ctx, &dax.ListTagsInput{
			ResourceName: cluster.ClusterArn,
		})
		if err != nil {
			return nil, err
		}

		values = append(values, Resource{
			ARN: *cluster.ClusterArn,
			Description: DAXClusterDescription{
				Cluster: cluster,
				Tags:    tags.Tags,
			},
		})
	}

	return values, nil
}
