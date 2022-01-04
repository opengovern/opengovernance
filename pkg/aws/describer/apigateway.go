package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
)

type ApiGatewayDescription struct {
	RestApi types.RestApi
	Stage	types.Stage
}

func ApiGatewayStage(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := apigateway.NewFromConfig(cfg)

	var values []Resource
	paginator := apigateway.NewGetRestApisPaginator(client, &apigateway.GetRestApisInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, restItem := range page.Items {
			out, err := client.GetStages(ctx, &apigateway.GetStagesInput{
				RestApiId: restItem.Id,
			})
			if err != nil {
				return nil, err
			}

			for _, stageItem := range out.Item {
				values = append(values, Resource{
					ID: CompositeID(*restItem.Id, *stageItem.StageName),
					Description: ApiGatewayDescription{
						RestApi: restItem,
						Stage:  stageItem,
					},
				})
			}
		}
	}
	return values, nil
}
