package describer

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
)

type SecurityHubHubDescription struct {
	Hub  *securityhub.DescribeHubOutput
	Tags map[string]string
}

func SecurityHubHub(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := securityhub.NewFromConfig(cfg)
	out, err := client.DescribeHub(ctx, &securityhub.DescribeHubInput{})
	if err != nil {
		if isErr(err, "InvalidAccessException") {
			return nil, nil
		}
		return nil, err
	}

	var values []Resource

	tags, err := client.ListTagsForResource(ctx, &securityhub.ListTagsForResourceInput{ResourceArn: out.HubArn})
	if err != nil {
		return nil, err
	}

	values = append(values, Resource{
		ARN:  *out.HubArn,
		Name: *out.HubArn,
		Description: SecurityHubHubDescription{
			Hub:  out,
			Tags: tags.Tags,
		},
	})

	return values, nil
}
