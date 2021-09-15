package describer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func ECSCapacityProvider(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ecs.NewFromConfig(cfg)

	var values []interface{}
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeCapacityProviders(ctx, &ecs.DescribeCapacityProvidersInput{NextToken: prevToken})
		if err != nil {
			return nil, err
		}
		if len(output.Failures) != 0 {
			return nil, failuresToError(output.Failures)
		}

		for _, v := range output.CapacityProviders {
			values = append(values, v)
		}

		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func ECSCluster(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	clusters, err := listEcsClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ecs.NewFromConfig(cfg)

	var values []interface{}
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
			values = append(values, v)
		}
	}

	return values, nil
}

// // Omit. Already included in the ECSCluster
// func ECSClusterCapacityProviderAssociations(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

func ECSService(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	clusters, err := listEcsClusters(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := ecs.NewFromConfig(cfg)

	var values []interface{}
	for _, cluster := range clusters {
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
				values = append(values, v)
			}
		}
	}

	return values, nil
}

func ECSTaskDefinition(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
	client := ecs.NewFromConfig(cfg)
	paginator := ecs.NewListTaskDefinitionsPaginator(client, &ecs.ListTaskDefinitionsInput{})

	var values []interface{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, arn := range page.TaskDefinitionArns {
			output, err := client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: &arn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, output.TaskDefinition)
		}
	}

	return values, nil
}

// OMIT part of ECSService
// func ECSTaskSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

// Specifies which task set in a service is the primary task set. 
// OMIT: Not really a seperate type
// func ECSPrimaryTaskSet(ctx context.Context, cfg aws.Config) ([]interface{}, error) {
// }

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
