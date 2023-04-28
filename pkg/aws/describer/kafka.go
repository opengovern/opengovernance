package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func KafkaCluster(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := kafka.NewFromConfig(cfg)
	paginator := kafka.NewListClustersV2Paginator(client, &kafka.ListClustersV2Input{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ClusterInfoList {
			resource := Resource{
				ARN:  *v.ClusterArn,
				Name: *v.ClusterName,
				Description: model.KafkaClusterDescription{
					Cluster: v,
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
