package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudsearch"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func CloudSearchDomain(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := cloudsearch.NewFromConfig(cfg)

	var values []Resource

	output, err := client.ListDomainNames(ctx, &cloudsearch.ListDomainNamesInput{})
	if err != nil {
		return nil, err
	}

	var domainList []string
	for domainName := range output.DomainNames {
		domainList = append(domainList, domainName)
	}

	domains, err := client.DescribeDomains(ctx, &cloudsearch.DescribeDomainsInput{
		DomainNames: domainList,
	})
	if err != nil {
		return nil, err
	}

	for _, domain := range domains.DomainStatusList {
		resource := Resource{
			ARN:  *domain.ARN,
			Name: *domain.DomainName,
			ID:   *domain.DomainId,
			Description: model.CloudSearchDomainDescription{
				DomainStatus: domain,
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
	return values, nil
}

func GetCloudSearchDomain(ctx context.Context, cfg aws.Config, domainList []string) ([]Resource, error) {
	client := cloudsearch.NewFromConfig(cfg)

	var values []Resource
	domains, err := client.DescribeDomains(ctx, &cloudsearch.DescribeDomainsInput{
		DomainNames: domainList,
	})
	if err != nil {
		return nil, err
	}

	for _, domain := range domains.DomainStatusList {
		values = append(values, Resource{
			ARN:  *domain.ARN,
			Name: *domain.DomainName,
			ID:   *domain.DomainId,
			Description: model.CloudSearchDomainDescription{
				DomainStatus: domain,
			},
		})
	}
	return values, nil
}
