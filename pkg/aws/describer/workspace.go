package describer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/workspaces"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func WorkSpacesConnectionAlias(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := workspaces.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeConnectionAliases(ctx, &workspaces.DescribeConnectionAliasesInput{
			NextToken: prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.ConnectionAliases {
			resource := Resource{
				ID:          *v.AliasId,
				Name:        *v.AliasId,
				Description: v,
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}

		}

		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func WorkspacesWorkspace(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := workspaces.NewFromConfig(cfg)
	paginator := workspaces.NewDescribeWorkspacesPaginator(client, &workspaces.DescribeWorkspacesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if !isErr(err, "ValidationException") {
				return nil, err
			}
			continue
		}

		for _, v := range page.Workspaces {
			tags, err := client.DescribeTags(ctx, &workspaces.DescribeTagsInput{
				ResourceId: v.WorkspaceId,
			})
			if err != nil {
				if !isErr(err, "ValidationException") {
					return nil, err
				}
				tags = &workspaces.DescribeTagsOutput{}
			}

			arn := fmt.Sprintf("arn:%s:workspaces:%s:%s:workspace/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.WorkspaceId)
			resource := Resource{
				ARN:  arn,
				Name: *v.WorkspaceId,
				Description: model.WorkspacesWorkspaceDescription{
					Workspace: v,
					Tags:      tags.TagList,
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

func WorkspacesBundle(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := workspaces.NewFromConfig(cfg)
	paginator := workspaces.NewDescribeWorkspaceBundlesPaginator(client, &workspaces.DescribeWorkspaceBundlesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Bundles {
			tags, err := client.DescribeTags(ctx, &workspaces.DescribeTagsInput{
				ResourceId: v.BundleId,
			})
			if err != nil {
				return nil, err
			}

			arn := fmt.Sprintf("arn:%s:workspaces:%s:%s:workspacebundle/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.BundleId)
			resource := Resource{
				ARN:  arn,
				Name: *v.BundleId,
				Description: model.WorkspacesBundleDescription{
					Bundle: v,
					Tags:   tags.TagList,
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
