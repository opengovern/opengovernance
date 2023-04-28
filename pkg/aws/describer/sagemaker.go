package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func SageMakerEndpointConfiguration(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
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
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *out.EndpointConfigArn,
				Name: *out.EndpointConfigName,
				Description: model.SageMakerEndpointConfigurationDescription{
					EndpointConfig: out,
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

func SageMakerNotebookInstance(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
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
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *out.NotebookInstanceArn,
				Name: *out.NotebookInstanceName,
				Description: model.SageMakerNotebookInstanceDescription{
					NotebookInstance: out,
					Tags:             tags.Tags,
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
