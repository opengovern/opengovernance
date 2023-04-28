package describer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/grafana"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func GrafanaWorkspace(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := grafana.NewFromConfig(cfg)
	paginator := grafana.NewListWorkspacesPaginator(client, &grafana.ListWorkspacesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Workspaces {
			arn := fmt.Sprintf("arn:%s:grafana:%s:%s:/workspaces/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.Id)
			resource := Resource{
				ARN:  arn,
				Name: *v.Id,
				Description: model.GrafanaWorkspaceDescription{
					Workspace: v,
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
