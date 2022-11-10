package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/storagegateway"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func StorageGatewayStorageGateway(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := storagegateway.NewFromConfig(cfg)
	paginator := storagegateway.NewListGatewaysPaginator(client, &storagegateway.ListGatewaysInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Gateways {
			tags, err := client.ListTagsForResource(ctx, &storagegateway.ListTagsForResourceInput{
				ResourceARN: v.GatewayARN,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *v.GatewayARN,
				Name: *v.GatewayId,
				Description: model.StorageGatewayStorageGatewayDescription{
					StorageGateway: v,
					Tags:           tags.Tags,
				},
			})
		}
	}

	return values, nil
}
