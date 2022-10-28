package describer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/redshift/types"
	"github.com/aws/aws-sdk-go-v2/service/redshiftserverless"
	"github.com/aws/smithy-go"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
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
			logStatus, err := client.DescribeLoggingStatus(ctx, &redshift.DescribeLoggingStatusInput{
				ClusterIdentifier: v.ClusterIdentifier,
			})
			if err != nil {
				return nil, err
			}

			sactions, err := client.DescribeScheduledActions(ctx, &redshift.DescribeScheduledActionsInput{
				Filters: []types.ScheduledActionFilter{
					{
						Name:   types.ScheduledActionFilterNameClusterIdentifier,
						Values: []string{*v.ClusterIdentifier},
					},
				},
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *v.ClusterNamespaceArn,
				Name: *v.ClusterIdentifier,
				Description: model.RedshiftClusterDescription{
					Cluster:          v,
					LoggingStatus:    logStatus,
					ScheduledActions: sactions.ScheduledActions,
				},
			})
		}
	}

	return values, nil
}

func RedshiftClusterParameterGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := redshift.NewFromConfig(cfg)
	paginator := redshift.NewDescribeClusterParameterGroupsPaginator(client, &redshift.DescribeClusterParameterGroupsInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ParameterGroups {
			param, err := client.DescribeClusterParameters(ctx, &redshift.DescribeClusterParametersInput{
				ParameterGroupName: v.ParameterGroupName,
			})
			if err != nil {
				return nil, err
			}

			arn := "arn:" + describeCtx.Partition + ":redshift:" + describeCtx.Region + ":" + describeCtx.AccountID + ":parametergroup"
			if strings.HasPrefix(*v.ParameterGroupName, ":") {
				arn = arn + *v.ParameterGroupName
			} else {
				arn = arn + ":" + *v.ParameterGroupName
			}
			values = append(values, Resource{
				ARN:  arn,
				Name: *v.ParameterGroupName,
				Description: model.RedshiftClusterParameterGroupDescription{
					ClusterParameterGroup: v,
					Parameters:            param.Parameters,
				},
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
				Name:        *v.ClusterSecurityGroupName,
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
				Name:        *v.ClusterSubnetGroupName,
				Description: v,
			})
		}
	}

	return values, nil
}

func RedshiftSnapshot(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)

	client := redshift.NewFromConfig(cfg)
	paginator := redshift.NewDescribeClusterSnapshotsPaginator(client, &redshift.DescribeClusterSnapshotsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if isErr(err, "ClusterSnapshotNotFound") {
				continue
			}
			return nil, err
		}

		for _, v := range page.Snapshots {
			arn := fmt.Sprintf("arn:%s:redshift:%s:%s:snapshot:%s/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.ClusterIdentifier, *v.SnapshotIdentifier)
			values = append(values, Resource{
				ARN:  arn,
				Name: *v.SnapshotIdentifier,
				Description: model.RedshiftSnapshotDescription{
					Snapshot: v,
				},
			})
		}
	}

	return values, nil
}

func RedshiftServerlessNamespace(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := redshiftserverless.NewFromConfig(cfg)
	paginator := redshiftserverless.NewListNamespacesPaginator(client, &redshiftserverless.ListNamespacesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Namespaces {
			tags, err := client.ListTagsForResource(ctx, &redshiftserverless.ListTagsForResourceInput{
				ResourceArn: v.NamespaceArn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *v.NamespaceArn,
				Name: *v.NamespaceName,
				Description: model.RedshiftServerlessNamespaceDescription{
					Namespace: v,
					Tags:      tags.Tags,
				},
			})
		}
	}

	return values, nil
}

func RedshiftServerlessSnapshot(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := redshiftserverless.NewFromConfig(cfg)
	paginator := redshiftserverless.NewListSnapshotsPaginator(client, &redshiftserverless.ListSnapshotsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Snapshots {
			tags, err := client.ListTagsForResource(ctx, &redshiftserverless.ListTagsForResourceInput{
				ResourceArn: v.NamespaceArn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *v.NamespaceArn,
				Name: *v.NamespaceName,
				Description: model.RedshiftServerlessSnapshotDescription{
					Snapshot: v,
					Tags:     tags.Tags,
				},
			})
		}
	}

	return values, nil
}
