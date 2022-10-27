package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appstream"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func AppStreamApplication(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := appstream.NewFromConfig(cfg)
	output, err := client.DescribeApplications(ctx, &appstream.DescribeApplicationsInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, item := range output.Applications {
		tags, err := client.ListTagsForResource(ctx, &appstream.ListTagsForResourceInput{
			ResourceArn: item.Arn,
		})
		if err != nil {
			return nil, err
		}
		values = append(values, Resource{
			ARN:  *item.Arn,
			Name: *item.Name,
			Description: model.AppStreamApplicationDescription{
				Application: item,
				Tags:        tags.Tags,
			},
		})
	}

	return values, nil
}

func AppStreamStack(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := appstream.NewFromConfig(cfg)
	output, err := client.DescribeStacks(ctx, &appstream.DescribeStacksInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, item := range output.Stacks {
		tags, err := client.ListTagsForResource(ctx, &appstream.ListTagsForResourceInput{
			ResourceArn: item.Arn,
		})
		if err != nil {
			return nil, err
		}
		values = append(values, Resource{
			ARN:  *item.Arn,
			Name: *item.Name,
			Description: model.AppStreamStackDescription{
				Stack: item,
				Tags:  tags.Tags,
			},
		})
	}

	return values, nil
}

func AppStreamFleet(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := appstream.NewFromConfig(cfg)
	output, err := client.DescribeFleets(ctx, &appstream.DescribeFleetsInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, item := range output.Fleets {
		tags, err := client.ListTagsForResource(ctx, &appstream.ListTagsForResourceInput{
			ResourceArn: item.Arn,
		})
		if err != nil {
			return nil, err
		}
		values = append(values, Resource{
			ARN:  *item.Arn,
			Name: *item.Name,
			Description: model.AppStreamFleetDescription{
				Fleet: item,
				Tags:  tags.Tags,
			},
		})
	}

	return values, nil
}
