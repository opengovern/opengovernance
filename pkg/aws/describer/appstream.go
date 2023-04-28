package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appstream"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func AppStreamApplication(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := appstream.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeApplications(ctx, &appstream.DescribeApplicationsInput{
			NextToken: prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, item := range output.Applications {
			tags, err := client.ListTagsForResource(ctx, &appstream.ListTagsForResourceInput{
				ResourceArn: item.Arn,
			})
			if err != nil {
				return nil, err
			}
			resource := Resource{
				ARN:  *item.Arn,
				Name: *item.Name,
				Description: model.AppStreamApplicationDescription{
					Application: item,
					Tags:        tags.Tags,
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
		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func AppStreamStack(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := appstream.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeStacks(ctx, &appstream.DescribeStacksInput{
			NextToken: prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, item := range output.Stacks {
			tags, err := client.ListTagsForResource(ctx, &appstream.ListTagsForResourceInput{
				ResourceArn: item.Arn,
			})
			if err != nil {
				return nil, err
			}
			resource := Resource{
				ARN:  *item.Arn,
				Name: *item.Name,
				Description: model.AppStreamStackDescription{
					Stack: item,
					Tags:  tags.Tags,
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
		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func AppStreamFleet(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := appstream.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeFleets(ctx, &appstream.DescribeFleetsInput{
			NextToken: prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, item := range output.Fleets {
			tags, err := client.ListTagsForResource(ctx, &appstream.ListTagsForResourceInput{
				ResourceArn: item.Arn,
			})
			if err != nil {
				return nil, err
			}
			resource := Resource{
				ARN:  *item.Arn,
				Name: *item.Name,
				Description: model.AppStreamFleetDescription{
					Fleet: item,
					Tags:  tags.Tags,
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
		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}
