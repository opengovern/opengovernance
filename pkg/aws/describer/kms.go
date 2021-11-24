package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

func KMSAlias(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := kms.NewFromConfig(cfg)
	paginator := kms.NewListAliasesPaginator(client, &kms.ListAliasesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Aliases {
			values = append(values, Resource{
				ARN:         *v.AliasArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func KMSKey(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := kms.NewFromConfig(cfg)
	paginator := kms.NewListKeysPaginator(client, &kms.ListKeysInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Keys {
			values = append(values, Resource{
				ARN:         *v.KeyArn,
				Description: v,
			})
		}
	}

	return values, nil
}
