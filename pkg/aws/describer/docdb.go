package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/docdb"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func DocDBCluster(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	//describeCtx := GetDescribeContext(ctx)
	client := docdb.NewFromConfig(cfg)
	paginator := docdb.NewDescribeDBClustersPaginator(client, &docdb.DescribeDBClustersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, cluster := range page.DBClusters {
			tags, err := client.ListTagsForResource(ctx, &docdb.ListTagsForResourceInput{
				ResourceName: cluster.DBClusterArn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:  *cluster.DBClusterIdentifier,
				ARN: *cluster.DBClusterArn,
				Description: model.DocDBClusterDescription{
					DBCluster: cluster,
					Tags:      tags.TagList,
				},
			})
		}
	}

	return values, nil
}
