package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/keyspaces"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func KeyspacesKeyspace(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := keyspaces.NewFromConfig(cfg)
	paginator := keyspaces.NewListKeyspacesPaginator(client, &keyspaces.ListKeyspacesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Keyspaces {
			tags, err := client.ListTagsForResource(ctx, &keyspaces.ListTagsForResourceInput{
				ResourceArn: v.ResourceArn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *v.ResourceArn,
				Name: *v.KeyspaceName,
				Description: model.KeyspacesKeyspaceDescription{
					Keyspace: v,
					Tags:     tags.Tags,
				},
			})
		}
	}

	return values, nil
}

func KeyspacesTable(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := keyspaces.NewFromConfig(cfg)
	paginator := keyspaces.NewListTablesPaginator(client, &keyspaces.ListTablesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Tables {
			tags, err := client.ListTagsForResource(ctx, &keyspaces.ListTagsForResourceInput{
				ResourceArn: v.ResourceArn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:   *v.ResourceArn,
				Name: *v.KeyspaceName,
				Description: model.KeyspacesTableDescription{
					Table: v,
					Tags:  tags.Tags,
				},
			})
		}
	}

	return values, nil
}
