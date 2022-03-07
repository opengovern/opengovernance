package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk/types"
)

type ElasticBeanstalkEnvironmentDescription struct {
	EnvironmentDescription types.EnvironmentDescription
	Tags                   []types.Tag
}

func ElasticBeanstalkEnvironment(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := elasticbeanstalk.NewFromConfig(cfg)
	out, err := client.DescribeEnvironments(ctx, &elasticbeanstalk.DescribeEnvironmentsInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, item := range out.Environments {
		tags, err := client.ListTagsForResource(ctx, &elasticbeanstalk.ListTagsForResourceInput{
			ResourceArn: item.EnvironmentArn,
		})
		if err != nil {
			return nil, err
		}

		values = append(values, Resource{
			ARN:  *item.EnvironmentArn,
			Name: *item.EnvironmentName,
			Description: ElasticBeanstalkEnvironmentDescription{
				EnvironmentDescription: item,
				Tags:                   tags.ResourceTags,
			},
		})
	}

	return values, nil
}
