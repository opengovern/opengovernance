package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codestar"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func CodeStarProject(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := codestar.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		projects, err := client.ListProjects(ctx, &codestar.ListProjectsInput{
			MaxResults: aws.Int32(100),
			NextToken:  prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, projectSum := range projects.Projects {
			project, err := client.DescribeProject(ctx, &codestar.DescribeProjectInput{
				Id: projectSum.ProjectId,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *project.Arn,
				Name: *project.Id,
				Description: model.CodeStarProjectDescription{
					Project: *project,
				},
			})
		}

		return projects.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}
