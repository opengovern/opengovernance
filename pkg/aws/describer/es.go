package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	es "github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func ESDomain(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	var values []Resource

	client := es.NewFromConfig(cfg)
	listNamesOut, err := client.ListDomainNames(ctx, &es.ListDomainNamesInput{})
	if err != nil {
		return nil, err
	}

	var domainNamesList []string
	for _, dn := range listNamesOut.DomainNames {
		domainNamesList = append(domainNamesList, *dn.DomainName)
	}

	if len(domainNamesList) == 0 {
		return values, nil
	}

	out, err := client.DescribeElasticsearchDomains(ctx, &es.DescribeElasticsearchDomainsInput{
		DomainNames: domainNamesList,
	})
	if err != nil {
		return nil, err
	}

	for _, v := range out.DomainStatusList {
		out, err := client.ListTags(ctx, &es.ListTagsInput{
			ARN: v.ARN,
		})
		if err != nil {
			return nil, err
		}

		resource := Resource{
			ARN:  *v.ARN,
			Name: *v.DomainName,
			Description: model.ESDomainDescription{
				Domain: v,
				Tags:   out.TagList,
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
