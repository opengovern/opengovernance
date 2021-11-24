package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

func ElasticLoadBalancingV2LoadBalancer(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := elasticloadbalancingv2.NewFromConfig(cfg)
	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(client, &elasticloadbalancingv2.DescribeLoadBalancersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LoadBalancers {
			values = append(values, Resource{
				ARN:         *v.LoadBalancerArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func ElasticLoadBalancingV2Listener(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	lbs, err := ElasticLoadBalancingV2LoadBalancer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := elasticloadbalancingv2.NewFromConfig(cfg)

	var values []Resource
	for _, lb := range lbs {
		arn := lb.Description.(types.LoadBalancer).LoadBalancerArn
		paginator := elasticloadbalancingv2.NewDescribeListenersPaginator(client, &elasticloadbalancingv2.DescribeListenersInput{
			LoadBalancerArn: arn,
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.Listeners {
				values = append(values, Resource{
					ARN:         *v.ListenerArn,
					Description: v,
				})
			}
		}

	}

	return values, nil
}

func ElasticLoadBalancingV2ListenerRule(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	listeners, err := ElasticLoadBalancingV2Listener(ctx, cfg)
	if err != nil {
		return nil, err
	}

	client := elasticloadbalancingv2.NewFromConfig(cfg)
	var values []Resource
	for _, l := range listeners {
		arn := l.Description.(types.Listener).ListenerArn
		err = PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
			output, err := client.DescribeRules(ctx, &elasticloadbalancingv2.DescribeRulesInput{
				ListenerArn: aws.String(*arn),
				Marker:      prevToken,
			})
			if err != nil {
				return nil, err
			}

			for _, v := range output.Rules {
				values = append(values, Resource{
					ARN:         *v.RuleArn,
					Description: v,
				})
			}

			return output.NextMarker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func ElasticLoadBalancingV2TargetGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := elasticloadbalancingv2.NewFromConfig(cfg)
	paginator := elasticloadbalancingv2.NewDescribeTargetGroupsPaginator(client, &elasticloadbalancingv2.DescribeTargetGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TargetGroups {
			values = append(values, Resource{
				ARN:         *v.TargetGroupArn,
				Description: v,
			})
		}
	}

	return values, nil
}

func ElasticLoadBalancingLoadBalancer(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := elasticloadbalancing.NewFromConfig(cfg)
	paginator := elasticloadbalancing.NewDescribeLoadBalancersPaginator(client, &elasticloadbalancing.DescribeLoadBalancersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LoadBalancerDescriptions {
			values = append(values, Resource{
				ARN:         *v.DNSName, // DNSName is unique
				Description: v,
			})
		}
	}

	return values, nil
}
