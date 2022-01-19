package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	typesv2 "github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
)

type ApiGatewayStageDescription struct {
	RestApiId *string
	Stage     types.Stage
}

func ApiGatewayStage(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := apigateway.NewFromConfig(cfg)
	paginator := apigateway.NewGetRestApisPaginator(client, &apigateway.GetRestApisInput{})

	var values []Resource
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
					Description: ApiGatewayStageDescription{
						RestApiId: restItem.Id,
						Stage:     stageItem,
					},
				})
			}
		}
	}
	return values, nil
}

type ApiGatewayV2StageDescription struct {
	ApiId *string
	Stage typesv2.Stage
}

func ApiGatewayV2Stage(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := apigatewayv2.NewFromConfig(cfg)

	var apis []typesv2.Api
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.GetApis(ctx, &apigatewayv2.GetApisInput{
			NextToken: prevToken,
		})
		if err != nil {
			return nil, err
		}

		apis = append(apis, output.Items...)
		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, api := range apis {
		var stages []typesv2.Stage
		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.GetStages(ctx, &apigatewayv2.GetStagesInput{
				ApiId:     api.ApiId,
				NextToken: prevToken,
			})
			if err != nil {
				return nil, err
			}

			stages = append(stages, output.Items...)
			return output.NextToken, nil
		})
		if err != nil {
			return nil, err
		}

		for _, stage := range stages {
			values = append(values, Resource{
				ID: CompositeID(*api.ApiId, *stage.StageName),
				Description: ApiGatewayV2StageDescription{
					ApiId: api.ApiId,
					Stage: stage,
				},
			})
		}
	}

	return values, nil
}
