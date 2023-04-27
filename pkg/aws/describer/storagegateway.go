package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/storagegateway/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/storagegateway"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func StorageGatewayStorageGateway(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
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

			resource := Resource{
				ARN:  *v.GatewayARN,
				Name: *v.GatewayId,
				Description: model.StorageGatewayStorageGatewayDescription{
					StorageGateway: v,
					Tags:           tags.Tags,
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

func GetStorageGatewayStorageGateway(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := storagegateway.NewFromConfig(cfg)
	out, err := client.DescribeGatewayInformation(ctx, &storagegateway.DescribeGatewayInformationInput{
		GatewayARN: nil,
	})
	if err != nil {
		return nil, err
	}

	var values []Resource
	tags, err := client.ListTagsForResource(ctx, &storagegateway.ListTagsForResourceInput{
		ResourceARN: out.GatewayARN,
	})
	if err != nil {
		return nil, err
	}

	resource := Resource{
		ARN:  *out.GatewayARN,
		Name: *out.GatewayId,
		Description: model.StorageGatewayStorageGatewayDescription{
			StorageGateway: types.GatewayInfo{
				Ec2InstanceId:     out.Ec2InstanceId,
				Ec2InstanceRegion: out.Ec2InstanceRegion,
				GatewayARN:        out.GatewayARN,
				GatewayId:         out.GatewayId,
				GatewayName:       out.GatewayName,
				//GatewayOperationalState: out.GatewayOperationalState, //TODO-Saleh
				GatewayType:       out.GatewayType,
				HostEnvironment:   out.HostEnvironment,
				HostEnvironmentId: out.HostEnvironmentId,
			},
			Tags: tags.Tags,
		},
	}
	if stream != nil {
		if err := (*stream)(resource); err != nil {
			return nil, err
		}
	} else {
		values = append(values, resource)
	}

	return values, nil
}
