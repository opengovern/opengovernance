package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
)

func CertificateManagerAccount(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := acm.NewFromConfig(cfg)
	output, err := client.GetAccountConfiguration(ctx, &acm.GetAccountConfigurationInput{})
	if err != nil {
		return nil, err
	}

	return []Resource{{
		ID:          "", // No ID or ARN. Per Account Configuration
		Description: output,
	}}, nil
}

func CertificateManagerCertificate(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := acm.NewFromConfig(cfg)
	paginator := acm.NewListCertificatesPaginator(client, &acm.ListCertificatesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.CertificateSummaryList {
			values = append(values, Resource{
				ARN:         *v.CertificateArn,
				Description: v,
			})
		}
	}

	return values, nil
}
