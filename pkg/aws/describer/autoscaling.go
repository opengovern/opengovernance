package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func AutoScalingAutoScalingGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := autoscaling.NewFromConfig(cfg)
	paginator := autoscaling.NewDescribeAutoScalingGroupsPaginator(client, &autoscaling.DescribeAutoScalingGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.AutoScalingGroups {
			var desc model.AutoScalingGroupDescription

			sg := v // Do this to avoid the pointer being replaces by the for loop

			desc.AutoScalingGroup = &sg
			desc.Policies, err = getAutoScalingPolicies(ctx, cfg, v.AutoScalingGroupName)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:         *v.AutoScalingGroupARN,
				Name:        *v.AutoScalingGroupName,
				Description: desc,
			})
		}
	}

	return values, nil
}

func GetAutoScalingAutoScalingGroup(ctx context.Context, cfg aws.Config, autoScalingGroupName string) ([]Resource, error) {
	client := autoscaling.NewFromConfig(cfg)

	out, err := client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{autoScalingGroupName},
	})
	if err != nil {
		return nil, err
	}

	var values []Resource

	for _, v := range out.AutoScalingGroups {
		var desc model.AutoScalingGroupDescription

		sg := v // Do this to avoid the pointer being replaces by the for loop

		desc.AutoScalingGroup = &sg
		desc.Policies, err = getAutoScalingPolicies(ctx, cfg, v.AutoScalingGroupName)
		if err != nil {
			return nil, err
		}

		values = append(values, Resource{
			ARN:         *v.AutoScalingGroupARN,
			Name:        *v.AutoScalingGroupName,
			Description: desc,
		})
	}

	return values, nil
}

func getAutoScalingPolicies(ctx context.Context, cfg aws.Config, asgName *string) ([]types.ScalingPolicy, error) {
	client := autoscaling.NewFromConfig(cfg)
	paginator := autoscaling.NewDescribePoliciesPaginator(client, &autoscaling.DescribePoliciesInput{
		AutoScalingGroupName: asgName,
	})

	var values []types.ScalingPolicy
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		values = append(values, page.ScalingPolicies...)
	}

	return values, nil
}

func AutoScalingLaunchConfiguration(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := autoscaling.NewFromConfig(cfg)
	paginator := autoscaling.NewDescribeLaunchConfigurationsPaginator(client, &autoscaling.DescribeLaunchConfigurationsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LaunchConfigurations {
			values = append(values, Resource{
				ARN:  *v.LaunchConfigurationARN,
				Name: *v.LaunchConfigurationName,
				Description: model.AutoScalingLaunchConfigurationDescription{
					LaunchConfiguration: v,
				},
			})
		}
	}

	return values, nil
}

func GetAutoScalingLaunchConfiguration(ctx context.Context, cfg aws.Config, launchConfigurationName string) ([]Resource, error) {
	client := autoscaling.NewFromConfig(cfg)
	out, err := client.DescribeLaunchConfigurations(ctx, &autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []string{launchConfigurationName},
	})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range out.LaunchConfigurations {
		values = append(values, Resource{
			ARN:  *v.LaunchConfigurationARN,
			Name: *v.LaunchConfigurationName,
			Description: model.AutoScalingLaunchConfigurationDescription{
				LaunchConfiguration: v,
			},
		})
	}

	return values, nil
}

func AutoScalingLifecycleHook(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	groups, err := AutoScalingAutoScalingGroup(ctx, cfg)
	if groups != nil {
		return nil, err
	}

	client := autoscaling.NewFromConfig(cfg)

	var values []Resource
	for _, g := range groups {
		group := g.Description.(types.AutoScalingGroup)
		output, err := client.DescribeLifecycleHooks(ctx, &autoscaling.DescribeLifecycleHooksInput{
			AutoScalingGroupName: group.AutoScalingGroupName,
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.LifecycleHooks {
			values = append(values, Resource{
				ID:          CompositeID(*v.AutoScalingGroupName, *v.LifecycleHookName),
				Name:        *v.AutoScalingGroupName,
				Description: v,
			})
		}
	}

	return values, nil
}

func AutoScalingScalingPolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := autoscaling.NewFromConfig(cfg)
	paginator := autoscaling.NewDescribePoliciesPaginator(client, &autoscaling.DescribePoliciesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ScalingPolicies {
			values = append(values, Resource{
				ARN:         *v.PolicyARN,
				Name:        *v.PolicyName,
				Description: v,
			})
		}
	}

	return values, nil
}

func AutoScalingScheduledAction(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := autoscaling.NewFromConfig(cfg)
	paginator := autoscaling.NewDescribeScheduledActionsPaginator(client, &autoscaling.DescribeScheduledActionsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.ScheduledUpdateGroupActions {
			values = append(values, Resource{
				ARN:         *v.ScheduledActionARN,
				Name:        *v.ScheduledActionName,
				Description: v,
			})
		}
	}

	return values, nil
}

func AutoScalingWarmPool(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	groups, err := AutoScalingAutoScalingGroup(ctx, cfg)
	if groups != nil {
		return nil, err
	}

	client := autoscaling.NewFromConfig(cfg)

	var values []Resource
	for _, g := range groups {
		group := g.Description.(types.AutoScalingGroup)

		err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.DescribeWarmPool(ctx, &autoscaling.DescribeWarmPoolInput{
				AutoScalingGroupName: group.AutoScalingGroupName,
				NextToken:            prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, v := range output.Instances {
				values = append(values, Resource{
					ID:          CompositeID(*group.AutoScalingGroupName, *v.InstanceId), // TODO
					Name:        *v.LaunchConfigurationName,
					Description: v,
				})
			}

			return output.NextToken, nil
		})
		if err != nil {
			return nil, err
		}

	}

	return values, nil
}
