package describer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	typesv2 "github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func ApiGatewayStage(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := apigateway.NewFromConfig(cfg)
	paginator := apigateway.NewGetRestApisPaginator(client, &apigateway.GetRestApisInput{})

	describeCtx := GetDescribeContext(ctx)

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
				arn := "arn:" + describeCtx.Partition + ":apigateway:" + describeCtx.Region + "::/restapis/" + *restItem.Id + "/stages/" + *stageItem.StageName
				values = append(values, Resource{
					ARN:  arn,
					Name: *restItem.Name,
					Description: model.ApiGatewayStageDescription{
						RestApiId: restItem.Id,
						Stage:     stageItem,
					},
				})
			}
		}
	}
	return values, nil
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
				ID:   CompositeID(*api.ApiId, *stage.StageName),
				Name: *api.Name,
				Description: model.ApiGatewayV2StageDescription{
					ApiId: api.ApiId,
					Stage: stage,
				},
			})
		}
	}

	return values, nil
}

func ApiGatewayRestAPI(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := apigateway.NewFromConfig(cfg)
	paginator := apigateway.NewGetRestApisPaginator(client, &apigateway.GetRestApisInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if isErr(err, "NotFoundException") {
				continue
			}
			return nil, err
		}

		for _, restItem := range page.Items {
			arn := fmt.Sprintf("arn:%s:apigateway:%s::/restapis/%s", describeCtx.Partition, describeCtx.Region, *restItem.Id)
			values = append(values, Resource{
				ARN:  arn,
				Name: *restItem.Name,
				Description: model.ApiGatewayRestAPIDescription{
					RestAPI: restItem,
				},
			})
		}
	}
	return values, nil
}

func ApiGatewayV2API(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := apigatewayv2.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.GetApis(ctx, &apigatewayv2.GetApisInput{
			NextToken: prevToken,
		})
		if err != nil {
			if isErr(err, "NotFoundException") {
				return nil, nil
			}
			return nil, err
		}

		for _, api := range output.Items {
			arn := fmt.Sprintf("arn:%s:apigateway:%s::/apis/%s", describeCtx.Partition, describeCtx.Region, *api.ApiId)
			values = append(values, Resource{
				ARN:  arn,
				Name: *api.Name,
				Description: model.ApiGatewayV2APIDescription{
					API: api,
				},
			})
		}
		return output.NextToken, nil
	})
	if err != nil {
		if isErr(err, "NotFoundException") {
			return nil, nil
		}
		return nil, err
	}

	return values, nil
}
