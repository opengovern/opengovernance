package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func RDSDBCluster(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeDBClustersPaginator(client, &rds.DescribeDBClustersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DBClusters {
			resource := Resource{
				ARN:  *v.DBClusterArn,
				Name: *v.DBClusterIdentifier,
				Description: model.RDSDBClusterDescription{
					DBCluster: v,
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
	}

	return values, nil
}

func GetRDSDBCluster(ctx context.Context, cfg aws.Config, fields map[string]string) ([]Resource, error) {
	arn := fields["arn"]
	client := rds.NewFromConfig(cfg)

	out, err := client.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
		DBClusterIdentifier: &arn,
	})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range out.DBClusters {
		values = append(values, Resource{
			ARN:  *v.DBClusterArn,
			Name: *v.DBClusterIdentifier,
			Description: model.RDSDBClusterDescription{
				DBCluster: v,
			},
		})
	}

	return values, nil
}

func RDSDBClusterSnapshot(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeDBClusterSnapshotsPaginator(client, &rds.DescribeDBClusterSnapshotsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DBClusterSnapshots {
			attr, err := client.DescribeDBClusterSnapshotAttributes(ctx, &rds.DescribeDBClusterSnapshotAttributesInput{
				DBClusterSnapshotIdentifier: v.DBClusterSnapshotIdentifier,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *v.DBClusterSnapshotArn,
				Name: *v.DBClusterSnapshotIdentifier,
				Description: model.RDSDBClusterSnapshotDescription{
					DBClusterSnapshot: v,
					Attributes:        attr.DBClusterSnapshotAttributesResult,
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
	}

	return values, nil
}

func RDSDBClusterParameterGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeDBClusterParameterGroupsPaginator(client, &rds.DescribeDBClusterParameterGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DBClusterParameterGroups {
			resource := Resource{
				ARN:         *v.DBClusterParameterGroupArn,
				Name:        *v.DBClusterParameterGroupName,
				Description: v,
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
		}
	}

	return values, nil
}

func RDSDBInstance(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DBInstances {
			resource := Resource{
				ARN:  *v.DBInstanceArn,
				Name: *v.DBInstanceIdentifier,
				Description: model.RDSDBInstanceDescription{
					DBInstance: v,
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
	}

	return values, nil
}

func GetRDSDBInstance(ctx context.Context, cfg aws.Config, fields map[string]string) ([]Resource, error) {
	dbInstanceId := fields["id"]
	client := rds.NewFromConfig(cfg)
	out, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &dbInstanceId,
	})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range out.DBInstances {
		values = append(values, Resource{
			ARN:  *v.DBInstanceArn,
			Name: *v.DBInstanceIdentifier,
			Description: model.RDSDBInstanceDescription{
				DBInstance: v,
			},
		})
	}

	return values, nil
}

func RDSDBParameterGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeDBParameterGroupsPaginator(client, &rds.DescribeDBParameterGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DBParameterGroups {
			resource := Resource{
				ARN:         *v.DBParameterGroupArn,
				Name:        *v.DBParameterGroupName,
				Description: v,
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
		}
	}

	return values, nil
}

func RDSDBProxy(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeDBProxiesPaginator(client, &rds.DescribeDBProxiesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DBProxies {
			resource := Resource{
				ARN:         *v.DBProxyArn,
				Name:        *v.DBProxyName,
				Description: v,
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
		}
	}

	return values, nil
}

func RDSDBProxyEndpoint(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeDBProxyEndpointsPaginator(client, &rds.DescribeDBProxyEndpointsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DBProxyEndpoints {
			resource := Resource{
				ARN:         *v.DBProxyEndpointArn,
				Name:        *v.DBProxyEndpointName,
				Description: v,
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
		}
	}

	return values, nil
}

func RDSDBProxyTargetGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	proxies, err := RDSDBProxy(ctx, cfg, nil)
	if err != nil {
		return nil, err
	}

	client := rds.NewFromConfig(cfg)

	var values []Resource
	for _, p := range proxies {
		proxy := p.Description.(types.DBProxy)
		paginator := rds.NewDescribeDBProxyTargetGroupsPaginator(client, &rds.DescribeDBProxyTargetGroupsInput{
			DBProxyName: proxy.DBProxyName,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.TargetGroups {
				resource := Resource{
					ARN:         *v.TargetGroupArn,
					Name:        *v.TargetGroupName,
					Description: v,
				}
				if stream != nil {
					if err := (*stream)(resource); err != nil {
						return nil, err
					}
				} else {
					values = append(values, resource)
				}
			}
		}
	}

	return values, nil
}

func RDSDBSecurityGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeDBSecurityGroupsPaginator(client, &rds.DescribeDBSecurityGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DBSecurityGroups {
			resource := Resource{
				ARN:         *v.DBSecurityGroupArn,
				Name:        *v.DBSecurityGroupName,
				Description: v,
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
		}
	}

	return values, nil
}

func RDSDBSubnetGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeDBSubnetGroupsPaginator(client, &rds.DescribeDBSubnetGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DBSubnetGroups {
			resource := Resource{
				ARN:         *v.DBSubnetGroupArn,
				Name:        *v.DBSubnetGroupName,
				Description: v,
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
		}
	}

	return values, nil
}

func RDSDBEventSubscription(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeEventSubscriptionsPaginator(client, &rds.DescribeEventSubscriptionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.EventSubscriptionsList {
			resource := Resource{
				ARN:  *v.EventSubscriptionArn,
				Name: *v.CustSubscriptionId,
				Description: model.RDSDBEventSubscriptionDescription{
					EventSubscription: v,
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
	}

	return values, nil
}

func RDSGlobalCluster(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeGlobalClustersPaginator(client, &rds.DescribeGlobalClustersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.GlobalClusters {
			tags, err := client.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
				ResourceName: v.GlobalClusterArn,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *v.GlobalClusterArn,
				Name: *v.GlobalClusterIdentifier,
				Description: model.RDSGlobalClusterDescription{
					GlobalCluster: v,
					Tags:          tags.TagList,
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
	}

	return values, nil
}

func RDSOptionGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeOptionGroupsPaginator(client, &rds.DescribeOptionGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.OptionGroupsList {
			resource := Resource{
				ARN:         *v.OptionGroupArn,
				Name:        *v.OptionGroupName,
				Description: v,
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
		}
	}

	return values, nil
}

func RDSDBSnapshot(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := rds.NewFromConfig(cfg)
	paginator := rds.NewDescribeDBSnapshotsPaginator(client, &rds.DescribeDBSnapshotsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.DBSnapshots {
			attrs, err := client.DescribeDBSnapshotAttributes(ctx, &rds.DescribeDBSnapshotAttributesInput{
				DBSnapshotIdentifier: v.DBSnapshotIdentifier,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *v.DBSnapshotArn,
				Name: *v.DBSnapshotIdentifier,
				Description: model.RDSDBSnapshotDescription{
					DBSnapshot:           v,
					DBSnapshotAttributes: attrs.DBSnapshotAttributesResult.DBSnapshotAttributes,
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
	}

	return values, nil
}
