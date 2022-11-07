package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func CloudFormationStack(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudformation.NewFromConfig(cfg)
	paginator := cloudformation.NewDescribeStacksPaginator(client, &cloudformation.DescribeStacksInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "ValidationError") && !isErr(err, "ResourceNotFoundException") {
				return nil, err
			}
			continue
		}

		for _, v := range page.Stacks {
			template, err := client.GetTemplate(ctx, &cloudformation.GetTemplateInput{
				StackName: v.StackName,
			})
			if err != nil {
				if !isErr(err, "ValidationError") && !isErr(err, "ResourceNotFoundException") {
					return nil, err
				}
				template = &cloudformation.GetTemplateOutput{}
			}

			stackResources, err := client.DescribeStackResources(ctx, &cloudformation.DescribeStackResourcesInput{
				StackName: v.StackName,
			})
			if err != nil {
				if !isErr(err, "ValidationError") && !isErr(err, "ResourceNotFoundException") {
					return nil, err
				}
				stackResources = &cloudformation.DescribeStackResourcesOutput{}
			}

			values = append(values, Resource{
				ARN:  *v.StackId,
				Name: *v.StackName,
				Description: model.CloudFormationStackDescription{
					Stack:          v,
					StackTemplate:  *template,
					StackResources: stackResources.StackResources,
				},
			})
		}
	}

	return values, nil
}
