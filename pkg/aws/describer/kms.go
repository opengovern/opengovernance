package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

func KMSAlias(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := kms.NewFromConfig(cfg)
	paginator := kms.NewListAliasesPaginator(client, &kms.ListAliasesInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Aliases {
			values = append(values, v)
		}
	}

	return values, nil
}

func KMSKey(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := kms.NewFromConfig(cfg)
	paginator := kms.NewListKeysPaginator(client, &kms.ListKeysInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Keys {
			values = append(values, v)
		}
	}

	return values, nil
}
