package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
)

type SageMakerEndpointConfigurationDescription struct {
	EndpointConfig *sagemaker.DescribeEndpointConfigOutput
	Tags           []types.Tag
}

func SageMakerEndpointConfiguration(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := sagemaker.NewFromConfig(cfg)
	paginator := sagemaker.NewListEndpointConfigsPaginator(client, &sagemaker.ListEndpointConfigsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.EndpointConfigs {
			out, err := client.DescribeEndpointConfig(ctx, &sagemaker.DescribeEndpointConfigInput{
				EndpointConfigName: item.EndpointConfigName,
			})
			if err != nil {
				return nil, err
			}

			tags, err := client.ListTags(ctx, &sagemaker.ListTagsInput{
				ResourceArn: item.EndpointConfigArn,
			})

			values = append(values, Resource{
				ARN: *out.EndpointConfigArn,
				Description: SageMakerEndpointConfigurationDescription{
					EndpointConfig: out,
					Tags:           tags.Tags,
				},
			})
		}
	}
	return values, nil
}

type SageMakerNotebookInstanceDescription struct {
	NotebookInstance *sagemaker.DescribeNotebookInstanceOutput
	Tags             []types.Tag
}

func SageMakerNotebookInstance(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := sagemaker.NewFromConfig(cfg)
	paginator := sagemaker.NewListNotebookInstancesPaginator(client, &sagemaker.ListNotebookInstancesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.NotebookInstances {
			out, err := client.DescribeNotebookInstance(ctx, &sagemaker.DescribeNotebookInstanceInput{
				NotebookInstanceName: item.NotebookInstanceName,
			})
			if err != nil {
				return nil, err
			}

			tags, err := client.ListTags(ctx, &sagemaker.ListTagsInput{
				ResourceArn: out.NotebookInstanceArn,
			})

			values = append(values, Resource{
				ARN: *out.NotebookInstanceArn,
				Description: SageMakerNotebookInstanceDescription{
					NotebookInstance: out,
					Tags:             tags.Tags,
				},
			})
		}
	}

	return values, nil
}
