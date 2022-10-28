package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func OpenSearchDomain(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := opensearch.NewFromConfig(cfg)

	domainNamesOutput, err := client.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
	if err != nil {
		return nil, err
	}
	domainNames := make([]string, 0, len(domainNamesOutput.DomainNames))
	for _, domainName := range domainNamesOutput.DomainNames {
		domainNames = append(domainNames, *domainName.DomainName)
	}

	domains, err := client.DescribeDomains(ctx, &opensearch.DescribeDomainsInput{
		DomainNames: domainNames,
	})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, domain := range domains.DomainStatusList {
		tags, err := client.ListTags(ctx, &opensearch.ListTagsInput{
			ARN: domain.ARN,
		})
		if err != nil {
			return nil, err
		}

		values = append(values, Resource{
			ARN:  *domain.ARN,
			Name: *domain.DomainName,
			Description: model.OpenSearchDomainDescription{
				Domain: domain,
				Tags:   tags.TagList,
			},
		})
	}

	return values, nil
}
