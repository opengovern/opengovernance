package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/emr"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func EMRCluster(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := emr.NewFromConfig(cfg)
	paginator := emr.NewListClustersPaginator(client, &emr.ListClustersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.Clusters {
			out, err := client.DescribeCluster(ctx, &emr.DescribeClusterInput{
				ClusterId: item.Id,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *out.Cluster.ClusterArn,
				Name: *out.Cluster.Name,
				Description: model.EMRClusterDescription{
					Cluster: out.Cluster,
				},
			})
		}
	}
	return values, nil
}
