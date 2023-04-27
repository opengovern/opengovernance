package describer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/emr"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func EMRCluster(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := emr.NewFromConfig(cfg)
	paginator := emr.NewListClustersPaginator(client, &emr.ListClustersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range page.Clusters {
			out, err := client.DescribeCluster(ctx, &emr.DescribeClusterInput{
				ClusterId: item.Id,
			})
			if err != nil {
				return nil, err
			}

			resource := Resource{
				ARN:  *out.Cluster.ClusterArn,
				Name: *out.Cluster.Name,
				Description: model.EMRClusterDescription{
					Cluster: out.Cluster,
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

func EMRInstance(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := emr.NewFromConfig(cfg)
	clusterPaginator := emr.NewListClustersPaginator(client, &emr.ListClustersInput{})

	var values []Resource
	for clusterPaginator.HasMorePages() {
		clusterPage, err := clusterPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cluster := range clusterPage.Clusters {
			instancePaginator := emr.NewListInstancesPaginator(client, &emr.ListInstancesInput{
				ClusterId: cluster.Id,
			})

			for instancePaginator.HasMorePages() {
				instancePage, err := instancePaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				for _, instance := range instancePage.Instances {
					arn := fmt.Sprintf("arn:%s:emr:%s:%s:instance/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *instance.Id)
					resource := Resource{
						ID:  *instance.Id,
						ARN: arn,
						Description: model.EMRInstanceDescription{
							Instance:  instance,
							ClusterID: *cluster.Id,
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
		}
	}
	return values, nil
}

func EMRInstanceFleet(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := emr.NewFromConfig(cfg)
	clusterPaginator := emr.NewListClustersPaginator(client, &emr.ListClustersInput{})

	var values []Resource
	for clusterPaginator.HasMorePages() {
		clusterPage, err := clusterPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cluster := range clusterPage.Clusters {
			instancePaginator := emr.NewListInstanceFleetsPaginator(client, &emr.ListInstanceFleetsInput{
				ClusterId: cluster.Id,
			})

			for instancePaginator.HasMorePages() {
				instancePage, err := instancePaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				for _, instanceFleet := range instancePage.InstanceFleets {
					arn := fmt.Sprintf("arn:%s:emr:%s:%s:instance-fleet/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *instanceFleet.Id)
					resource := Resource{
						ID:   *instanceFleet.Id,
						Name: *instanceFleet.Name,
						ARN:  arn,
						Description: model.EMRInstanceFleetDescription{
							InstanceFleet: instanceFleet,
							ClusterID:     *cluster.Id,
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
		}
	}
	return values, nil
}

func EMRInstanceGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := emr.NewFromConfig(cfg)
	clusterPaginator := emr.NewListClustersPaginator(client, &emr.ListClustersInput{})

	var values []Resource
	for clusterPaginator.HasMorePages() {
		clusterPage, err := clusterPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cluster := range clusterPage.Clusters {
			instancePaginator := emr.NewListInstanceGroupsPaginator(client, &emr.ListInstanceGroupsInput{
				ClusterId: cluster.Id,
			})

			for instancePaginator.HasMorePages() {
				instancePage, err := instancePaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				for _, instanceGroup := range instancePage.InstanceGroups {
					arn := fmt.Sprintf("arn:%s:emr:%s:%s:instance-group/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *instanceGroup.Id)
					resource := Resource{
						ID:   *instanceGroup.Id,
						Name: *instanceGroup.Name,
						ARN:  arn,
						Description: model.EMRInstanceGroupDescription{
							InstanceGroup: instanceGroup,
							ClusterID:     *cluster.Id,
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
		}
	}
	return values, nil
}
