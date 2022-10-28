package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/amp"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func AMPWorkspace(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := amp.NewFromConfig(cfg)
	paginator := amp.NewListWorkspacesPaginator(client, &amp.ListWorkspacesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Workspaces {
			values = append(values, Resource{
				ARN:  *v.Arn,
				Name: *v.WorkspaceId,
				Description: model.AMPWorkspaceDescription{
					Workspace: v,
				},
			})
		}
	}

	return values, nil
}
