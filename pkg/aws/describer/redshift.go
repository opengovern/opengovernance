package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
)

func RedshiftCluster(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := redshift.NewFromConfig(cfg)
	paginator := redshift.NewDescribeClustersPaginator(client, &redshift.DescribeClustersInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Clusters {
			values = append(values, v)
		}
	}

	return values, nil
}

func RedshiftClusterParameterGroup(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := redshift.NewFromConfig(cfg)
	paginator := redshift.NewDescribeClusterParameterGroupsPaginator(client, &redshift.DescribeClusterParameterGroupsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ParameterGroups {
			values = append(values, v)
		}
	}

	return values, nil
}

// TODO: Catch this error and return empty
// * An error occurred (InvalidParameterValue) when calling the CreateClusterSecurityGroup operation: VPC-by-Default customers cannot use cluster security groups
func RedshiftClusterSecurityGroup(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := redshift.NewFromConfig(cfg)
	paginator := redshift.NewDescribeClusterSecurityGroupsPaginator(client, &redshift.DescribeClusterSecurityGroupsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ClusterSecurityGroups {
			values = append(values, v)
		}
	}

	return values, nil
}

// OMIT: Already included in RedshiftClusterSecurityGroup
// func RedshiftClusterSecurityGroupIngress(ctx context.Context, cfg aws.Config) ([]interface{}, error) {

// }

func RedshiftClusterSubnetGroup(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := redshift.NewFromConfig(cfg)
	paginator := redshift.NewDescribeClusterSubnetGroupsPaginator(client, &redshift.DescribeClusterSubnetGroupsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ClusterSubnetGroups {
			values = append(values, v)
		}
	}

	return values, nil
}