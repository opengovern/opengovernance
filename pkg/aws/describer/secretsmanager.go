package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type SecretsManagerSecretDescription struct {
	Secret         *secretsmanager.DescribeSecretOutput
	ResourcePolicy *string
}

func SecretsManagerSecret(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := secretsmanager.NewFromConfig(cfg)
	paginator := secretsmanager.NewListSecretsPaginator(client, &secretsmanager.ListSecretsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.SecretList {
			out, err := client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
				SecretId: item.ARN,
			})
			if err != nil {
				return nil, err
			}

			policy, err := client.GetResourcePolicy(ctx, &secretsmanager.GetResourcePolicyInput{
				SecretId: item.ARN,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *item.ARN,
				Name: *item.Name,
				Description: SecretsManagerSecretDescription{
					Secret:         out,
					ResourcePolicy: policy.ResourcePolicy,
				},
			})
		}
	}
	return values, nil
}
