package describer

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dax"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func DAXCluster(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := dax.NewFromConfig(cfg)
	out, err := client.DescribeClusters(ctx, &dax.DescribeClustersInput{})
	if err != nil {
		if strings.Contains(err.Error(), "InvalidParameterValueException") || strings.Contains(err.Error(), "no such host") {
			return nil, nil
		}
		return nil, err
	}

	var values []Resource
	for _, cluster := range out.Clusters {
		tags, err := client.ListTags(ctx, &dax.ListTagsInput{
			ResourceName: cluster.ClusterArn,
		})
		if err != nil {
			if strings.Contains(err.Error(), "ClusterNotFoundFault") {
				tags = nil
			} else {
				return nil, err
			}
		}

		values = append(values, Resource{
			ARN:  *cluster.ClusterArn,
			Name: *cluster.ClusterName,
			Description: model.DAXClusterDescription{
				Cluster: cluster,
				Tags:    tags.Tags,
			},
		})
	}

	return values, nil
}

func DAXParameterGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	//describeCtx := GetDescribeContext(ctx)
	client := dax.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		parameterGroups, err := client.DescribeParameterGroups(ctx, &dax.DescribeParameterGroupsInput{
			MaxResults: aws.Int32(100),
			NextToken:  prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, parameterGroup := range parameterGroups.ParameterGroups {
			values = append(values, Resource{
				Name: *parameterGroup.ParameterGroupName,
				Description: model.DAXParameterGroupDescription{
					ParameterGroup: parameterGroup,
				},
			})
		}

		return parameterGroups.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func DAXParameter(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	//describeCtx := GetDescribeContext(ctx)
	client := dax.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		parameterGroups, err := client.DescribeParameterGroups(ctx, &dax.DescribeParameterGroupsInput{
			MaxResults: aws.Int32(100),
			NextToken:  prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, parameterGroup := range parameterGroups.ParameterGroups {
			err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
				parameters, err := client.DescribeParameters(ctx, &dax.DescribeParametersInput{
					ParameterGroupName: parameterGroup.ParameterGroupName,
					MaxResults:         aws.Int32(100),
					NextToken:          prevToken,
				})
				if err != nil {
					return nil, err
				}

				for _, parameter := range parameters.Parameters {
					values = append(values, Resource{
						Name: *parameter.ParameterName,
						Description: model.DAXParameterDescription{
							Parameter:          parameter,
							ParameterGroupName: *parameterGroup.ParameterGroupName,
						},
					})
				}

				return parameters.NextToken, nil
			})
			if err != nil {
				return nil, err
			}
		}

		return parameterGroups.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func DAXSubnetGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := dax.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		subnetGroups, err := client.DescribeSubnetGroups(ctx, &dax.DescribeSubnetGroupsInput{
			MaxResults: aws.Int32(100),
			NextToken:  prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, subnetGroup := range subnetGroups.SubnetGroups {
			arn := fmt.Sprintf("arn:%s:dax:%s::subnetgroup:%s", describeCtx.Partition, describeCtx.Region, *subnetGroup.SubnetGroupName)
			values = append(values, Resource{
				Name: *subnetGroup.SubnetGroupName,
				ARN:  arn,
				Description: model.DAXSubnetGroupDescription{
					SubnetGroup: subnetGroup,
				},
			})
		}

		return subnetGroups.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}
