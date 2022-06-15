package describer

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/smithy-go"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func LambdaFunction(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := lambda.NewFromConfig(cfg)
	paginator := lambda.NewListFunctionsPaginator(client, &lambda.ListFunctionsInput{
		FunctionVersion: types.FunctionVersionAll,
	})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Functions {
			policy, err := client.GetPolicy(ctx, &lambda.GetPolicyInput{
				FunctionName: v.FunctionName,
			})
			if err != nil {
				var ae smithy.APIError
				if errors.As(err, &ae) && ae.ErrorCode() == "ResourceNotFoundException" {
					policy = &lambda.GetPolicyOutput{}
					err = nil
				}

				if awsErr, ok := err.(awserr.Error); ok {
					log.Println("Describe Lambda Error:", awsErr.Code(), awsErr.Message())
					if awsErr.Code() == "ResourceNotFoundException" {
						policy = &lambda.GetPolicyOutput{}
						err = nil
					}
				}

				if err != nil {
					return nil, err
				}
			}

			function, err := client.GetFunction(ctx, &lambda.GetFunctionInput{
				FunctionName: v.FunctionName,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *v.FunctionArn,
				Name: *v.FunctionName,
				Description: model.LambdaFunctionDescription{
					Function: function,
					Policy:   policy,
				},
			})
		}
	}

	return values, nil
}

func LambdaAlias(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	fns, err := LambdaFunction(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := lambda.NewFromConfig(cfg)

	var values []Resource
	for _, f := range fns {
		fn := f.Description.(model.LambdaFunctionDescription).Function.Configuration
		paginator := lambda.NewListAliasesPaginator(client, &lambda.ListAliasesInput{
			FunctionName:    fn.FunctionName,
			FunctionVersion: fn.Version,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.Aliases {
				values = append(values, Resource{
					ARN:         *v.AliasArn,
					Name:        *v.Name,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func LambdaPermission(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	fns, err := LambdaFunction(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := lambda.NewFromConfig(cfg)

	var values []Resource
	for _, f := range fns {
		fn := f.Description.(model.LambdaFunctionDescription).Function.Configuration
		v, err := client.GetPolicy(ctx, &lambda.GetPolicyInput{
			FunctionName: fn.FunctionArn,
		})
		if err != nil {
			var ae smithy.APIError
			if errors.As(err, &ae) && ae.ErrorCode() == "ResourceNotFoundException" {
				continue
			}

			return nil, err
		}

		values = append(values, Resource{
			ID:          CompositeID(*fn.FunctionArn, *v.Policy),
			Name:        *v.Policy,
			Description: v,
		})
	}

	return values, nil
}

func LambdaEventInvokeConfig(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	fns, err := LambdaFunction(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := lambda.NewFromConfig(cfg)

	var values []Resource
	for _, f := range fns {
		fn := f.Description.(model.LambdaFunctionDescription).Function.Configuration
		paginator := lambda.NewListFunctionEventInvokeConfigsPaginator(client, &lambda.ListFunctionEventInvokeConfigsInput{
			FunctionName: fn.FunctionName,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.FunctionEventInvokeConfigs {
				values = append(values, Resource{
					ID:          *fn.FunctionName, // Invoke Config is unique per function
					Name:        *fn.FunctionName,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func LambdaCodeSigningConfig(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := lambda.NewFromConfig(cfg)
	paginator := lambda.NewListCodeSigningConfigsPaginator(client, &lambda.ListCodeSigningConfigsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.CodeSigningConfigs {
			values = append(values, Resource{
				ARN:         *v.CodeSigningConfigArn,
				Name:        *v.CodeSigningConfigArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func LambdaEventSourceMapping(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := lambda.NewFromConfig(cfg)
	paginator := lambda.NewListEventSourceMappingsPaginator(client, &lambda.ListEventSourceMappingsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.EventSourceMappings {
			values = append(values, Resource{
				ARN:         *v.EventSourceArn,
				Name:        *v.UUID,
				Description: v,
			})
		}
	}

	return values, nil
}

func LambdaLayerVersion(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	layers, err := listLayers(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := lambda.NewFromConfig(cfg)

	var values []Resource
	for _, layer := range layers {
		paginator := lambda.NewListLayerVersionsPaginator(client, &lambda.ListLayerVersionsInput{
			LayerName: layer.LayerArn,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.LayerVersions {
				values = append(values, Resource{
					ARN:         *v.LayerVersionArn,
					Name:        *v.LayerVersionArn,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func LambdaLayerVersionPermission(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	lvs, err := LambdaLayerVersion(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := lambda.NewFromConfig(cfg)

	var values []Resource
	for _, lv := range lvs {
		arn := lv.Description.(types.LayerVersionsListItem).LayerVersionArn
		version := lv.Description.(types.LayerVersionsListItem).Version
		v, err := client.GetLayerVersionPolicy(ctx, &lambda.GetLayerVersionPolicyInput{
			LayerName:     arn,
			VersionNumber: version,
		})
		if err != nil {
			return nil, err
		}

		values = append(values, Resource{
			ID:          CompositeID(*arn, fmt.Sprintf("%d", version)),
			Name:        *arn,
			Description: v,
		})
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
