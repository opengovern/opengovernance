package describer

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dax"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func DAXCluster(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := dax.NewFromConfig(cfg)
	out, err := client.DescribeClusters(ctx, &dax.DescribeClustersInput{})
	if err != nil {
		if strings.Contains(err.Error(), "InvalidParameterValueException") || strings.Contains(err.Error(), "no such host") {
			return nil, nil
		}
		return nil, err
	}

	var values []Resource
	for _, cluster := range out.Clusters {
		tags, err := client.ListTags(ctx, &dax.ListTagsInput{
			ResourceName: cluster.ClusterArn,
		})
		if err != nil {
			if strings.Contains(err.Error(), "ClusterNotFoundFault") {
				tags = nil
			} else {
				return nil, err
			}
		}

		values = append(values, Resource{
			ARN:  *cluster.ClusterArn,
			Name: *cluster.ClusterName,
			Description: model.DAXClusterDescription{
				Cluster: cluster,
				Tags:    tags.Tags,
			},
		})
	}

	return values, nil
}
