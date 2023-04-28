package describer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func CodeDeployDeploymentGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := codedeploy.NewFromConfig(cfg)
	paginator := codedeploy.NewListApplicationsPaginator(client, &codedeploy.ListApplicationsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, appName := range page.Applications {
			deploymentGroupPaginator := codedeploy.NewListDeploymentGroupsPaginator(client, &codedeploy.ListDeploymentGroupsInput{
				ApplicationName: &appName,
			})

			for deploymentGroupPaginator.HasMorePages() {
				deploymentGroupPage, err := deploymentGroupPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				for _, deploymentGroupName := range deploymentGroupPage.DeploymentGroups {
					deploymentGroup, err := client.GetDeploymentGroup(ctx, &codedeploy.GetDeploymentGroupInput{
						ApplicationName:     &appName,
						DeploymentGroupName: &deploymentGroupName,
					})
					if err != nil {
						return nil, err
					}

					arn := fmt.Sprintf("arn:%s:codedeploy:%s:%s:deploymentgroup:%s/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID,
						*deploymentGroup.DeploymentGroupInfo.ApplicationName, *deploymentGroup.DeploymentGroupInfo.DeploymentGroupName)

					tags, err := client.ListTagsForResource(ctx, &codedeploy.ListTagsForResourceInput{
						ResourceArn: &arn,
					})
					if err != nil {
						return nil, err
					}

					resource := Resource{
						ARN:  arn,
						Name: *deploymentGroup.DeploymentGroupInfo.DeploymentGroupName,
						Description: model.CodeDeployDeploymentGroupDescription{
							DeploymentGroup: *deploymentGroup.DeploymentGroupInfo,
							Tags:            tags.Tags,
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
		}
	}

	return values, nil
}

func CodeDeployApplication(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := codedeploy.NewFromConfig(cfg)
	paginator := codedeploy.NewListApplicationsPaginator(client, &codedeploy.ListApplicationsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, appName := range page.Applications {
			application, err := client.GetApplication(ctx, &codedeploy.GetApplicationInput{
				ApplicationName: &appName,
			})
			if err != nil {
				return nil, err
			}

			arn := fmt.Sprintf("arn:%s:codedeploy:%s:%s:application:%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *application.Application.ApplicationName)

			tags, err := client.ListTagsForResource(ctx, &codedeploy.ListTagsForResourceInput{
				ResourceArn: &arn,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  arn,
				Name: *application.Application.ApplicationName,
				Description: model.CodeDeployApplicationDescription{
					Application: *application.Application,
					Tags:        tags.Tags,
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
