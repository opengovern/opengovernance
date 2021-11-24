package describer

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/smithy-go"
)

func RedshiftCluster(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := redshift.NewFromConfig(cfg)
	paginator := redshift.NewDescribeClustersPaginator(client, &redshift.DescribeClustersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Clusters {
			values = append(values, Resource{
				ARN:         *v.ClusterNamespaceArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func RedshiftClusterParameterGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := redshift.NewFromConfig(cfg)
	paginator := redshift.NewDescribeClusterParameterGroupsPaginator(client, &redshift.DescribeClusterParameterGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ParameterGroups {
			values = append(values, Resource{
				ID:          *v.ParameterGroupName,
				Description: v,
			})
		}
	}

	return values, nil
}

func RedshiftClusterSecurityGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := redshift.NewFromConfig(cfg)
	paginator := redshift.NewDescribeClusterSecurityGroupsPaginator(client, &redshift.DescribeClusterSecurityGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			var ae smithy.APIError
			if errors.As(err, &ae) && (ae.ErrorMessage() == "VPC-by-Default customers cannot use cluster security groups") {
				return nil, nil
			}

			return nil, err
		}

		for _, v := range page.ClusterSecurityGroups {
			values = append(values, Resource{
				ID:          *v.ClusterSecurityGroupName,
				Description: v,
			})
		}
	}

	return values, nil
}

func RedshiftClusterSubnetGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := redshift.NewFromConfig(cfg)
	paginator := redshift.NewDescribeClusterSubnetGroupsPaginator(client, &redshift.DescribeClusterSubnetGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ClusterSubnetGroups {
			values = append(values, Resource{
				ID:          *v.ClusterSubnetGroupName,
				Description: v,
			})
		}
	}

	return values, nil
}
