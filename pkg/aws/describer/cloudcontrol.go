package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudcontrol"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func CloudControlResource(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudcontrol.NewFromConfig(cfg)
	paginator := cloudcontrol.NewListResourcesPaginator(client, &cloudcontrol.ListResourcesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ResourceDescriptions {
			values = append(values, Resource{
				ID: *v.Identifier,
				Description: model.CloudControlResourceDescription{
					Resource: v,
				},
			})
		}
	}

	return values, nil
}
