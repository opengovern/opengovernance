package describer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func ECSCapacityProvider(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ecs.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeCapacityProviders(ctx, &ecs.DescribeCapacityProvidersInput{NextToken: prevToken})
		if err != nil {
			return nil, err
		}
		if len(output.Failures) != 0 {
			return nil, failuresToError(output.Failures)
		}

		for _, v := range output.CapacityProviders {
			values = append(values, Resource{
				ARN:         *v.CapacityProviderArn,
				Name:        *v.Name,
				Description: v,
			})
		}

		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func ECSCluster(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	clusters, err := listEcsClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ecs.NewFromConfig(cfg)

	var values []Resource
	// Describe in batch of 100 which is the limit
	for i := 0; i < len(clusters); i = i + 100 {
		j := i + 100
		if j > len(clusters) {
			j = len(clusters)
		}

		output, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
			Clusters: clusters[i:j],
		})
		if err != nil {
			return nil, err
		}
		if len(output.Failures) != 0 {
			return nil, failuresToError(output.Failures)
		}

		for _, v := range output.Clusters {
			values = append(values, Resource{
				ARN:  *v.ClusterArn,
				Name: *v.ClusterName,
				Description: model.ECSClusterDescription{
					Cluster: v,
				},
			})
		}
	}

	return values, nil
}

func GetECSCluster(ctx context.Context, cfg aws.Config, cluster string) ([]Resource, error) {
	client := ecs.NewFromConfig(cfg)

	var values []Resource
	output, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{cluster},
	})
	if err != nil {
		return nil, err
	}
	if len(output.Failures) != 0 {
		return nil, failuresToError(output.Failures)
	}

	for _, v := range output.Clusters {
		values = append(values, Resource{
			ARN:  *v.ClusterArn,
			Name: *v.ClusterName,
			Description: model.ECSClusterDescription{
				Cluster: v,
			},
		})
	}

	return values, nil
}

func ECSService(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	clusters, err := listEcsClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ecs.NewFromConfig(cfg)

	var values []Resource
	for _, cluster := range clusters {
		// This prevents Implicit memory aliasing in for loop
		cluster := cluster
		services, err := listECsServices(ctx, cfg, cluster)
		if err != nil {
			return nil, err
		}

		// Describe in batch of 10 which is the limit
		for i := 0; i < len(services); i = i + 10 {
			j := i + 10
			if j > len(services) {
				j = len(services)
			}

			output, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
				Cluster:  &cluster,
				Services: services[i:j],
			})
			if err != nil {
				return nil, err
			}
			if len(output.Failures) != 0 {
				return nil, failuresToError(output.Failures)
			}

			for _, v := range output.Services {
				values = append(values, Resource{
					ARN:  *v.ServiceArn,
					Name: *v.ServiceName,
					Description: model.ECSServiceDescription{
						Service: v,
					},
				})
			}
		}
	}

	return values, nil
}

func GetECSService(ctx context.Context, cfg aws.Config, cluster, service string) ([]Resource, error) {
	client := ecs.NewFromConfig(cfg)

	var values []Resource
	output, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &cluster,
		Services: []string{service},
	})
	if err != nil {
		return nil, err
	}
	if len(output.Failures) != 0 {
		return nil, failuresToError(output.Failures)
	}

	for _, v := range output.Services {
		values = append(values, Resource{
			ARN:  *v.ServiceArn,
			Name: *v.ServiceName,
			Description: model.ECSServiceDescription{
				Service: v,
			},
		})
	}

	return values, nil
}

func ECSTaskDefinition(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := ecs.NewFromConfig(cfg)
	paginator := ecs.NewListTaskDefinitionsPaginator(client, &ecs.ListTaskDefinitionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, arn := range page.TaskDefinitionArns {
			output, err := client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: &arn,
				Include: []types.TaskDefinitionField{
					types.TaskDefinitionFieldTags,
				},
			})
			if err != nil {
				return nil, err
			}

			// From Steampipe
			splitArn := strings.Split(arn, ":")
			name := splitArn[len(splitArn)-1]

			values = append(values, Resource{
				ARN:  arn,
				Name: name,
				Description: model.ECSTaskDefinitionDescription{
					TaskDefinition: output.TaskDefinition,
					Tags:           output.Tags,
				},
			})
		}
	}

	return values, nil
}

func GetECSTaskDefinition(ctx context.Context, cfg aws.Config, taskDefinitionARN string) ([]Resource, error) {
	client := ecs.NewFromConfig(cfg)

	var values []Resource
	output, err := client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: &taskDefinitionARN,
		Include: []types.TaskDefinitionField{
			types.TaskDefinitionFieldTags,
		},
	})
	if err != nil {
		return nil, err
	}

	// From Steampipe
	splitArn := strings.Split(taskDefinitionARN, ":")
	name := splitArn[len(splitArn)-1]

	values = append(values, Resource{
		ARN:  taskDefinitionARN,
		Name: name,
		Description: model.ECSTaskDefinitionDescription{
			TaskDefinition: output.TaskDefinition,
			Tags:           output.Tags,
		},
	})

	return values, nil
}

func ECSTaskSet(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	clusters, err := listEcsClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ecs.NewFromConfig(cfg)
	var values []Resource

	for _, cluster := range clusters {
		cluster := cluster
		services, err := listECsServices(ctx, cfg, cluster)
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(services); i = i + 10 {
			j := i + 10
			if j > len(services) {
				j = len(services)
			}

			serviceOutput, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
				Cluster:  &cluster,
				Services: services[i:j],
			})
			if err != nil {
				return nil, err
			}
			if len(serviceOutput.Failures) != 0 {
				return nil, failuresToError(serviceOutput.Failures)
			}

			for _, service := range serviceOutput.Services {
				service := service
				if err != nil {
					return nil, err
				}
				for _, v := range service.TaskSets {
					values = append(values, Resource{
						ARN:  *v.TaskSetArn,
						Name: *v.Id,
						Description: model.ECSTaskSetDescription{
							TaskSet: v,
						},
					})
				}
			}
		}

	}

	return values, nil
}

func listECsServices(ctx context.Context, cfg aws.Config, cluster string) ([]string, error) {
	client := ecs.NewFromConfig(cfg)
	paginator := ecs.NewListServicesPaginator(client, &ecs.ListServicesInput{
		Cluster: &cluster,
	})

	var services []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		services = append(services, page.ServiceArns...)
	}

	return services, nil
}

func listEcsClusters(ctx context.Context, cfg aws.Config) ([]string, error) {
	client := ecs.NewFromConfig(cfg)
	paginator := ecs.NewListClustersPaginator(client, &ecs.ListClustersInput{})

	var clusters []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		clusters = append(clusters, page.ClusterArns...)
	}

	return clusters, nil
}

func ECSContainerInstance(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	clusters, err := listEcsClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ecs.NewFromConfig(cfg)

	var values []Resource
	for _, cluster := range clusters {
		paginator := ecs.NewListContainerInstancesPaginator(client, &ecs.ListContainerInstancesInput{
			Cluster: &cluster,
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			if page.ContainerInstanceArns == nil || len(page.ContainerInstanceArns) == 0 {
				continue
			}
			output, err := client.DescribeContainerInstances(ctx, &ecs.DescribeContainerInstancesInput{
				Cluster:            &cluster,
				ContainerInstances: page.ContainerInstanceArns,
			})
			if err != nil {
				return nil, err
			}
			if len(output.Failures) != 0 {
				return nil, failuresToError(output.Failures)
			}

			for _, v := range output.ContainerInstances {
				values = append(values, Resource{
					ARN:  *v.ContainerInstanceArn,
					Name: *v.ContainerInstanceArn,
					Description: model.ECSContainerInstanceDescription{
						ContainerInstance: v,
					},
				})
			}
		}
	}

	return values, nil
}

func ECSTask(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	clusters, err := listEcsClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ecs.NewFromConfig(cfg)
	var values []Resource

	for _, cluster := range clusters {
		cluster := cluster
		services, err := listECsServices(ctx, cfg, cluster)
		if err != nil {
			return nil, err
		}

		for _, service := range services {
			service := service
			paginator := ecs.NewListTasksPaginator(client, &ecs.ListTasksInput{
				Cluster:     &cluster,
				ServiceName: &service,
			})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}

				if page.TaskArns == nil || len(page.TaskArns) == 0 {
					continue
				}
				output, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
					Cluster: &cluster,
					Tasks:   page.TaskArns,
				})
				if err != nil {
					return nil, err
				}
				if len(output.Failures) != 0 {
					return nil, failuresToError(output.Failures)
				}
				taskProtections, err := client.GetTaskProtection(ctx, &ecs.GetTaskProtectionInput{
					Cluster: &cluster,
					Tasks:   page.TaskArns,
				})
				if err != nil {
					return nil, err
				}
				if len(taskProtections.Failures) != 0 {
					return nil, failuresToError(output.Failures)
				}

				taskProtectionMap := make(map[string]types.ProtectedTask)
				for _, taskProtection := range taskProtections.ProtectedTasks {
					taskProtectionMap[*taskProtection.TaskArn] = taskProtection
				}

				for _, v := range output.Tasks {
					description := model.ECSTaskDescription{
						Task:           v,
						ServiceName:    service,
						TaskProtection: nil,
					}
					if taskProtection, ok := taskProtectionMap[*v.TaskArn]; ok {
						description.TaskProtection = &taskProtection
					}
					values = append(values, Resource{
						ARN:         *v.TaskArn,
						Name:        *v.TaskArn,
						Description: description,
					})
				}
			}
		}
	}

	return values, nil
}

func failuresToError(failures []types.Failure) error {
	var errs []string
	for _, f := range failures {
		errs = append(errs, fmt.Sprintf("Arn=%s, Detail=%s, Reason=%s",
			aws.ToString(f.Arn),
			aws.ToString(f.Detail),
			aws.ToString(f.Reason)))
	}

	return errors.New(strings.Join(errs, ";"))
}
