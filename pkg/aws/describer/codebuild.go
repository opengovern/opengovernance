package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func CodeBuildProject(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := codebuild.NewFromConfig(cfg)
	paginator := codebuild.NewListProjectsPaginator(client, &codebuild.ListProjectsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		if len(page.Projects) == 0 {
			continue
		}

		projects, err := client.BatchGetProjects(ctx, &codebuild.BatchGetProjectsInput{
			Names: page.Projects,
		})
		if err != nil {
			return nil, err
		}

		for _, project := range projects.Projects {
			values = append(values, Resource{
				ARN:  *project.Arn,
				Name: *project.Name,
				Description: model.CodeBuildProjectDescription{
					Project: project,
				},
			})
		}
	}

	return values, nil
}

func CodeBuildSourceCredential(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := codebuild.NewFromConfig(cfg)
	out, err := client.ListSourceCredentials(ctx, &codebuild.ListSourceCredentialsInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, item := range out.SourceCredentialsInfos {
		values = append(values, Resource{
			ARN:  *item.Arn,
			Name: nameFromArn(*item.Arn),
			Description: model.CodeBuildSourceCredentialDescription{
				SourceCredentialsInfo: item,
			},
		})
	}

	return values, nil
}
