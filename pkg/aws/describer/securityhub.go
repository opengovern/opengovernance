package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

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
		Name: nameFromArn(*out.HubArn),
		Description: model.SecurityHubHubDescription{
			Hub:  out,
			Tags: tags.Tags,
		},
	})

	return values, nil
}
