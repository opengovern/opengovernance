package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codebuild/types"
)

type CodeBuildProjectDescription struct {
	Project types.Project
}

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
				ARN: *project.Arn,
				Description: CodeBuildProjectDescription {
					Project: project,
				},
			})
		}
	}

	return values, nil
}