package describer

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDbTableDescription struct {
	Table            *types.TableDescription
	ContinuousBackup *types.ContinuousBackupsDescription
	Tags             []types.Tag
}

func DynamoDbTable(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := dynamodb.NewFromConfig(cfg)
	paginator := dynamodb.NewListTablesPaginator(client, &dynamodb.ListTablesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, table := range page.TableNames {
			v, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
				TableName: &table,
			})
			if err != nil {
				return nil, err
			}

			continuousBackup, err := client.DescribeContinuousBackups(ctx, &dynamodb.DescribeContinuousBackupsInput{
				TableName: &table,
			})
			if err != nil {
				return nil, err
			}

			tags, err := client.ListTagsOfResource(ctx, &dynamodb.ListTagsOfResourceInput{
				ResourceArn: v.Table.TableArn,
			})

			values = append(values, Resource{
				ARN: *v.Table.TableArn,
				Description: DynamoDbTableDescription{
					Table:            v.Table,
					ContinuousBackup: continuousBackup.ContinuousBackupsDescription,
					Tags:             tags.Tags,
				},
			})
		}
	}

	return values, nil
}
