package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func LambdaFunction(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := lambda.NewFromConfig(cfg)
	paginator := lambda.NewListFunctionsPaginator(client, &lambda.ListFunctionsInput{
		FunctionVersion: types.FunctionVersionAll,
	})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Functions {
			values = append(values, v)
		}
	}

	return values, nil
}

func LambdaAlias(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	fns, err := LambdaFunction(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := lambda.NewFromConfig(cfg)

	var values []interface{}
	for _, fn := range fns {
		paginator := lambda.NewListAliasesPaginator(client, &lambda.ListAliasesInput{
			FunctionName:    fn.(types.FunctionConfiguration).FunctionName,
			FunctionVersion: fn.(types.FunctionConfiguration).Version,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.Aliases {
				values = append(values, v)
			}
		}
	}

	return values, nil
}

func LambdaPermission(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	fns, err := LambdaFunction(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := lambda.NewFromConfig(cfg)

	var values []interface{}
	for _, fn := range fns {
		output, err := client.GetPolicy(ctx, &lambda.GetPolicyInput{
			FunctionName: fn.(types.FunctionConfiguration).FunctionName,
		})
		if err != nil {
			return nil, err
		}

		values = append(values, output)
	}

	return values, nil
}

func LambdaEventInvokeConfig(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	fns, err := LambdaFunction(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := lambda.NewFromConfig(cfg)

	var values []interface{}
	for _, fn := range fns {
		paginator := lambda.NewListFunctionEventInvokeConfigsPaginator(client, &lambda.ListFunctionEventInvokeConfigsInput{
			FunctionName: fn.(types.FunctionConfiguration).FunctionName,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.FunctionEventInvokeConfigs {
				values = append(values, v)
			}
		}
	}

	return values, nil
}

func LambdaCodeSigningConfig(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := lambda.NewFromConfig(cfg)
	paginator := lambda.NewListCodeSigningConfigsPaginator(client, &lambda.ListCodeSigningConfigsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.CodeSigningConfigs {
			values = append(values, v)
		}
	}

	return values, nil
}

func LambdaEventSourceMapping(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := lambda.NewFromConfig(cfg)
	paginator := lambda.NewListEventSourceMappingsPaginator(client, &lambda.ListEventSourceMappingsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.EventSourceMappings {
			values = append(values, v)
		}
	}

	return values, nil
}

func LambdaLayerVersion(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	layers, err := listLayers(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := lambda.NewFromConfig(cfg)

	var values []interface{}
	for _, layer := range layers {
		paginator := lambda.NewListLayerVersionsPaginator(client, &lambda.ListLayerVersionsInput{LayerName: layer.LayerArn})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.LayerVersions {
				values = append(values, v)
			}
		}
	}

	return values, nil
}

func LambdaLayerVersionPermission(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	lvs, err := LambdaLayerVersion(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := lambda.NewFromConfig(cfg)

	var values []interface{}
	for _, lv := range lvs {
		output, err := client.GetLayerVersionPolicy(ctx, &lambda.GetLayerVersionPolicyInput{
			LayerName:     lv.(types.LayerVersionsListItem).LayerVersionArn,
			VersionNumber: lv.(types.LayerVersionsListItem).Version,
		})
		if err != nil {
			return nil, err
		}

		values = append(values, output)
	}

	return values, nil
}

func listLayers(ctx context.Context, cfg aws.Config) ([]types.LayersListItem, error) {
	client := lambda.NewFromConfig(cfg)
	paginator := lambda.NewListLayersPaginator(client, &lambda.ListLayersInput{})

	var values []types.LayersListItem
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		values = append(values, page.Layers...)
	}

	return values, nil
}

// OMIT: Already included in LambdaFunction
// func LambdaVersion(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }
