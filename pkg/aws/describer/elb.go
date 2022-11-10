package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
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
			attrs, err := client.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancingv2.DescribeLoadBalancerAttributesInput{
				LoadBalancerArn: v.LoadBalancerArn,
			})
			if err != nil {
				return nil, err
			}

			tags, err := client.DescribeTags(ctx, &elasticloadbalancingv2.DescribeTagsInput{
				ResourceArns: []string{*v.LoadBalancerArn},
			})
			if err != nil {
				return nil, err
			}

			description := model.ElasticLoadBalancingV2LoadBalancerDescription{
				LoadBalancer: v,
				Attributes:   attrs.Attributes,
			}

			if tags.TagDescriptions != nil && len(tags.TagDescriptions) > 0 {
				description.Tags = tags.TagDescriptions[0].Tags
			}

			values = append(values, Resource{
				ARN:         *v.LoadBalancerArn,
				Name:        *v.LoadBalancerName,
				Description: description,
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
		arn := lb.Description.(model.ElasticLoadBalancingV2LoadBalancerDescription).LoadBalancer.LoadBalancerArn
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
					ARN:  *v.ListenerArn,
					Name: nameFromArn(*v.ListenerArn),
					Description: model.ElasticLoadBalancingV2ListenerDescription{
						Listener: v,
					},
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
		arn := l.Description.(model.ElasticLoadBalancingV2ListenerDescription).Listener.ListenerArn
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
					ARN:  *v.RuleArn,
					Name: *v.RuleArn,
					Description: model.ElasticLoadBalancingV2RuleDescription{
						Rule: v,
					},
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
				ARN:  *v.TargetGroupArn,
				Name: *v.TargetGroupName,
				Description: model.ElasticLoadBalancingV2TargetGroupDescription{
					TargetGroup: v,
				},
			})
		}
	}

	return values, nil
}

func ElasticLoadBalancingLoadBalancer(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := elasticloadbalancing.NewFromConfig(cfg)
	paginator := elasticloadbalancing.NewDescribeLoadBalancersPaginator(client, &elasticloadbalancing.DescribeLoadBalancersInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LoadBalancerDescriptions {
			attrs, err := client.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancing.DescribeLoadBalancerAttributesInput{
				LoadBalancerName: v.LoadBalancerName,
			})
			if err != nil {
				return nil, err
			}

			tags, err := client.DescribeTags(ctx, &elasticloadbalancing.DescribeTagsInput{
				LoadBalancerNames: []string{*v.LoadBalancerName},
			})
			if err != nil {
				return nil, err
			}

			description := model.ElasticLoadBalancingLoadBalancerDescription{
				LoadBalancer: v,
				Attributes:   attrs.LoadBalancerAttributes,
			}

			if tags.TagDescriptions != nil && len(tags.TagDescriptions) > 0 {
				description.Tags = tags.TagDescriptions[0].Tags
			}

			arn := "arn:" + describeCtx.Partition + ":elasticloadbalancing:" + describeCtx.Region + ":" + describeCtx.AccountID + ":loadbalancer/" + *v.LoadBalancerName
			values = append(values, Resource{
				ARN:         arn,
				Name:        *v.LoadBalancerName,
				Description: description,
			})
		}
	}

	return values, nil
}
