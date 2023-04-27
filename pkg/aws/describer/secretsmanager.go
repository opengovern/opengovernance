package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func SecretsManagerSecret(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
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

			resource := Resource{
				ARN:  *item.ARN,
				Name: *item.Name,
				Description: model.SecretsManagerSecretDescription{
					Secret:         out,
					ResourcePolicy: policy.ResourcePolicy,
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
