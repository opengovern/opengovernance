package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

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
			Description: model.ElasticBeanstalkEnvironmentDescription{
				EnvironmentDescription: item,
				Tags:                   tags.ResourceTags,
			},
		})
	}

	return values, nil
}

func ElasticBeanstalkApplication(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := elasticbeanstalk.NewFromConfig(cfg)
	out, err := client.DescribeApplications(ctx, &elasticbeanstalk.DescribeApplicationsInput{})
	if err != nil {
		if !isErr(err, "ResourceNotFoundException") {
			return nil, err
		}
		return nil, nil
	}

	var values []Resource
	for _, item := range out.Applications {
		tags, err := client.ListTagsForResource(ctx, &elasticbeanstalk.ListTagsForResourceInput{
			ResourceArn: item.ApplicationArn,
		})
		if err != nil {
			if !isErr(err, "ResourceNotFoundException") {
				return nil, err
			}
			tags = &elasticbeanstalk.ListTagsForResourceOutput{}
		}

		values = append(values, Resource{
			ARN:  *item.ApplicationArn,
			Name: *item.ApplicationName,
			Description: model.ElasticBeanstalkApplicationDescription{
				Application: item,
				Tags:        tags.ResourceTags,
			},
		})
	}

	return values, nil
}
