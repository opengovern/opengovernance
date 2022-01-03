package describer

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	es "github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice/types"
)

type ESDomainDescription struct {
	Domain types.ElasticsearchDomainStatus
}

func ESDomains(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := es.NewFromConfig(cfg)
	out, err := client.DescribeElasticsearchDomains(ctx, &es.DescribeElasticsearchDomainsInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource

	for _, v := range out.DomainStatusList {
		values = append(values, Resource{
			ID:         *v.DomainId,
			Description: ESDomainDescription {
				Domain: v,
			},
		})
	}

	return values, nil
}

