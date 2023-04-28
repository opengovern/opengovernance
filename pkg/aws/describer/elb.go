package describer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func ElasticLoadBalancingV2LoadBalancer(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
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

			resource := Resource{
				ARN:         *v.LoadBalancerArn,
				Name:        *v.LoadBalancerName,
				Description: description,
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

func GetElasticLoadBalancingV2LoadBalancer(ctx context.Context, cfg aws.Config, fields map[string]string) ([]Resource, error) {
	lbARN := fields["arn"]
	client := elasticloadbalancingv2.NewFromConfig(cfg)
	out, err := client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{lbARN},
	})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range out.LoadBalancers {
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

	return values, nil
}

func ElasticLoadBalancingV2Listener(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	lbs, err := ElasticLoadBalancingV2LoadBalancer(ctx, cfg, nil)
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
				resource := Resource{
					ARN:  *v.ListenerArn,
					Name: nameFromArn(*v.ListenerArn),
					Description: model.ElasticLoadBalancingV2ListenerDescription{
						Listener: v,
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

	return values, nil
}

func GetElasticLoadBalancingV2Listener(ctx context.Context, cfg aws.Config, fields map[string]string) ([]Resource, error) {
	lbArn := fields["load_balancer_arn"]
	listenerARN := fields["arn"]
	client := elasticloadbalancingv2.NewFromConfig(cfg)
	out, err := client.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{
		ListenerArns:    []string{listenerARN},
		LoadBalancerArn: &lbArn,
	})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range out.Listeners {
		values = append(values, Resource{
			ARN:  *v.ListenerArn,
			Name: nameFromArn(*v.ListenerArn),
			Description: model.ElasticLoadBalancingV2ListenerDescription{
				Listener: v,
			},
		})
	}

	return values, nil
}

func ElasticLoadBalancingV2ListenerRule(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	listeners, err := ElasticLoadBalancingV2Listener(ctx, cfg, nil)
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
				resource := Resource{
					ARN:  *v.RuleArn,
					Name: *v.RuleArn,
					Description: model.ElasticLoadBalancingV2RuleDescription{
						Rule: v,
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

			return output.NextMarker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func GetElasticLoadBalancingV2ListenerRule(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	listeners, err := ElasticLoadBalancingV2Listener(ctx, cfg, nil)
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
				resource := Resource{
					ARN:  *v.RuleArn,
					Name: *v.RuleArn,
					Description: model.ElasticLoadBalancingV2RuleDescription{
						Rule: v,
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

			return output.NextMarker, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func ElasticLoadBalancingLoadBalancer(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
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
			resource := Resource{
				ARN:         arn,
				Name:        *v.LoadBalancerName,
				Description: description,
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

func ElasticLoadBalancingV2SslPolicy(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := elasticloadbalancingv2.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeSSLPolicies(ctx, &elasticloadbalancingv2.DescribeSSLPoliciesInput{
			Marker: prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.SslPolicies {
			arn := fmt.Sprintf("arn:%s:elbv2:%s:%s:ssl-policy/%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *v.Name)
			resource := Resource{
				Name: *v.Name,
				ARN:  arn,
				Description: model.ElasticLoadBalancingV2SslPolicyDescription{
					SslPolicy: v,
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

		return output.NextMarker, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func ElasticLoadBalancingV2TargetGroup(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := elasticloadbalancingv2.NewFromConfig(cfg)
	paginator := elasticloadbalancingv2.NewDescribeTargetGroupsPaginator(client, &elasticloadbalancingv2.DescribeTargetGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.TargetGroups {
			healthDescriptions, err := client.DescribeTargetHealth(ctx, &elasticloadbalancingv2.DescribeTargetHealthInput{
				TargetGroupArn: v.TargetGroupArn,
			})
			if err != nil {
				return nil, err
			}

			tags, err := client.DescribeTags(ctx, &elasticloadbalancingv2.DescribeTagsInput{
				ResourceArns: []string{*v.TargetGroupArn},
			})
			if err != nil {
				return nil, err
			}

			var tagsA []types.Tag
			if tags.TagDescriptions != nil && len(tags.TagDescriptions) > 0 {
				tagsA = tags.TagDescriptions[0].Tags
			}

			resource := Resource{
				ARN:  *v.TargetGroupArn,
				Name: *v.TargetGroupName,
				Description: model.ElasticLoadBalancingV2TargetGroupDescription{
					TargetGroup: v,
					Health:      healthDescriptions.TargetHealthDescriptions,
					Tags:        tagsA,
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

func ApplicationLoadBalancerMetricRequestCount(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := elasticloadbalancingv2.NewFromConfig(cfg)
	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(client, &elasticloadbalancingv2.DescribeLoadBalancersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, loadBalancer := range page.LoadBalancers {
			if loadBalancer.Type != types.LoadBalancerTypeEnumApplication {
				continue
			}
			arn := strings.SplitN(*loadBalancer.LoadBalancerArn, "/", 2)[1]
			metrics, err := listCloudWatchMetricStatistics(ctx, cfg, "5_MIN", "AWS/ApplicationELB", "RequestCount", "LoadBalancer", arn)
			if err != nil {
				return nil, err
			}
			for _, metric := range metrics {
				resource := Resource{
					ID: fmt.Sprintf("%s:%s:%s:%s", arn, metric.Timestamp.Format(time.RFC3339), *metric.DimensionName, *metric.DimensionValue),
					Description: model.ApplicationLoadBalancerMetricRequestCountDescription{
						CloudWatchMetricRow: metric,
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

	return values, nil
}

func GetApplicationLoadBalancerMetricRequestCount(ctx context.Context, cfg aws.Config, fields map[string]string) ([]Resource, error) {
	loadBalancerARN := fields["arn"]
	client := elasticloadbalancingv2.NewFromConfig(cfg)
	out, err := client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{loadBalancerARN},
	})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, loadBalancer := range out.LoadBalancers {
		if loadBalancer.Type != types.LoadBalancerTypeEnumApplication {
			continue
		}
		arn := strings.SplitN(*loadBalancer.LoadBalancerArn, "/", 2)[1]
		metrics, err := listCloudWatchMetricStatistics(ctx, cfg, "5_MIN", "AWS/ApplicationELB", "RequestCount", "LoadBalancer", arn)
		if err != nil {
			return nil, err
		}
		for _, metric := range metrics {
			values = append(values, Resource{
				ID: fmt.Sprintf("%s:%s:%s:%s", arn, metric.Timestamp.Format(time.RFC3339), *metric.DimensionName, *metric.DimensionValue),
				Description: model.ApplicationLoadBalancerMetricRequestCountDescription{
					CloudWatchMetricRow: metric,
				},
			})
		}
	}

	return values, nil
}

func ApplicationLoadBalancerMetricRequestCountDaily(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := elasticloadbalancingv2.NewFromConfig(cfg)
	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(client, &elasticloadbalancingv2.DescribeLoadBalancersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, loadBalancer := range page.LoadBalancers {
			if loadBalancer.Type != types.LoadBalancerTypeEnumApplication {
				continue
			}
			arn := strings.SplitN(*loadBalancer.LoadBalancerArn, "/", 2)[1]
			metrics, err := listCloudWatchMetricStatistics(ctx, cfg, "DAILY", "AWS/ApplicationELB", "RequestCount", "LoadBalancer", arn)
			if err != nil {
				return nil, err
			}
			for _, metric := range metrics {
				resource := Resource{
					ID: fmt.Sprintf("%s:%s:%s:%s", arn, metric.Timestamp.Format(time.RFC3339), *metric.DimensionName, *metric.DimensionValue),
					Description: model.ApplicationLoadBalancerMetricRequestCountDailyDescription{
						CloudWatchMetricRow: metric,
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

	return values, nil
}

func GetApplicationLoadBalancerMetricRequestCountDaily(ctx context.Context, cfg aws.Config, fields map[string]string) ([]Resource, error) {
	loadBalancerARN := fields["arn"]

	client := elasticloadbalancingv2.NewFromConfig(cfg)
	out, err := client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{LoadBalancerArns: []string{loadBalancerARN}})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, loadBalancer := range out.LoadBalancers {
		if loadBalancer.Type != types.LoadBalancerTypeEnumApplication {
			continue
		}
		arn := strings.SplitN(*loadBalancer.LoadBalancerArn, "/", 2)[1]
		metrics, err := listCloudWatchMetricStatistics(ctx, cfg, "DAILY", "AWS/ApplicationELB", "RequestCount", "LoadBalancer", arn)
		if err != nil {
			return nil, err
		}
		for _, metric := range metrics {
			values = append(values, Resource{
				ID: fmt.Sprintf("%s:%s:%s:%s", arn, metric.Timestamp.Format(time.RFC3339), *metric.DimensionName, *metric.DimensionValue),
				Description: model.ApplicationLoadBalancerMetricRequestCountDailyDescription{
					CloudWatchMetricRow: metric,
				},
			})
		}
	}

	return values, nil
}

func NetworkLoadBalancerMetricNetFlowCount(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := elasticloadbalancingv2.NewFromConfig(cfg)
	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(client, &elasticloadbalancingv2.DescribeLoadBalancersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, loadBalancer := range page.LoadBalancers {
			if loadBalancer.Type != types.LoadBalancerTypeEnumNetwork {
				continue
			}
			arn := strings.SplitN(*loadBalancer.LoadBalancerArn, "/", 2)[1]
			metrics, err := listCloudWatchMetricStatistics(ctx, cfg, "5_MIN", "AWS/NetworkELB", "NewFlowCount", "LoadBalancer", arn)
			if err != nil {
				return nil, err
			}
			for _, metric := range metrics {
				resource := Resource{
					ID: fmt.Sprintf("%s:%s:%s:%s", arn, metric.Timestamp.Format(time.RFC3339), *metric.DimensionName, *metric.DimensionValue),
					Description: model.NetworkLoadBalancerMetricNetFlowCountDescription{
						CloudWatchMetricRow: metric,
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

	return values, nil
}

func NetworkLoadBalancerMetricNetFlowCountDaily(ctx context.Context, cfg aws.Config, stream *StreamSender) ([]Resource, error) {
	client := elasticloadbalancingv2.NewFromConfig(cfg)
	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(client, &elasticloadbalancingv2.DescribeLoadBalancersInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, loadBalancer := range page.LoadBalancers {
			if loadBalancer.Type != types.LoadBalancerTypeEnumNetwork {
				continue
			}
			arn := strings.SplitN(*loadBalancer.LoadBalancerArn, "/", 2)[1]
			metrics, err := listCloudWatchMetricStatistics(ctx, cfg, "DAILY", "AWS/NetworkELB", "NewFlowCount", "LoadBalancer", arn)
			if err != nil {
				return nil, err
			}
			for _, metric := range metrics {
				resource := Resource{
					ID: fmt.Sprintf("%s:%s:%s:%s", arn, metric.Timestamp.Format(time.RFC3339), *metric.DimensionName, *metric.DimensionValue),
					Description: model.NetworkLoadBalancerMetricNetFlowCountDailyDescription{
						CloudWatchMetricRow: metric,
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

	return values, nil
}
