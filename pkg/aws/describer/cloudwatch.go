package describer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
)

func CloudWatchAlarm(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatch.NewFromConfig(cfg)
	paginator := cloudwatch.NewDescribeAlarmsPaginator(client, &cloudwatch.DescribeAlarmsInput{
		AlarmTypes: []types.AlarmType{types.AlarmTypeMetricAlarm},
	})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.MetricAlarms {
			tags, err := client.ListTagsForResource(ctx, &cloudwatch.ListTagsForResourceInput{
				ResourceARN: v.AlarmArn,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *v.AlarmArn,
				Name: *v.AlarmName,
				Description: model.CloudWatchAlarmDescription{
					MetricAlarm: v,
					Tags:        tags.Tags,
				},
			})
		}
	}

	return values, nil
}

func CloudWatchAnomalyDetector(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatch.NewFromConfig(cfg)
	output, err := client.DescribeAnomalyDetectors(ctx, &cloudwatch.DescribeAnomalyDetectorsInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.AnomalyDetectors {
		values = append(values, Resource{
			ID:          CompositeID(*v.SingleMetricAnomalyDetector.Namespace, *v.SingleMetricAnomalyDetector.MetricName),
			Name:        *v.SingleMetricAnomalyDetector.MetricName,
			Description: v,
		})
	}

	return values, nil
}

func CloudWatchCompositeAlarm(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatch.NewFromConfig(cfg)
	paginator := cloudwatch.NewDescribeAlarmsPaginator(client, &cloudwatch.DescribeAlarmsInput{
		AlarmTypes: []types.AlarmType{types.AlarmTypeCompositeAlarm},
	})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.MetricAlarms {
			values = append(values, Resource{
				ARN:         *v.AlarmArn,
				Name:        *v.AlarmName,
				Description: v,
			})
		}
	}

	return values, nil
}

func CloudWatchDashboard(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatch.NewFromConfig(cfg)
	output, err := client.ListDashboards(ctx, &cloudwatch.ListDashboardsInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.DashboardEntries {
		values = append(values, Resource{
			ARN:         *v.DashboardArn,
			Name:        *v.DashboardName,
			Description: v,
		})
	}

	return values, nil
}

func CloudWatchInsightRule(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatch.NewFromConfig(cfg)
	paginator := cloudwatch.NewDescribeInsightRulesPaginator(client, &cloudwatch.DescribeInsightRulesInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.InsightRules {
			values = append(values, Resource{
				ID:          *v.Name,
				Name:        *v.Name,
				Description: v,
			})
		}
	}

	return values, nil
}

func CloudWatchMetricStream(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatch.NewFromConfig(cfg)
	output, err := client.ListMetricStreams(ctx, &cloudwatch.ListMetricStreamsInput{})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range output.Entries {
		values = append(values, Resource{
			ARN:         *v.Arn,
			Name:        *v.Name,
			Description: v,
		})
	}

	return values, nil
}

func CloudWatchLogsDestination(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatchlogs.NewFromConfig(cfg)
	paginator := cloudwatchlogs.NewDescribeDestinationsPaginator(client, &cloudwatchlogs.DescribeDestinationsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Destinations {
			values = append(values, Resource{
				ARN:         *v.Arn,
				Name:        *v.DestinationName,
				Description: v,
			})
		}
	}

	return values, nil
}

func CloudWatchLogsLogGroup(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatchlogs.NewFromConfig(cfg)
	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(client, &cloudwatchlogs.DescribeLogGroupsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.LogGroups {
			tags, err := client.ListTagsLogGroup(ctx, &cloudwatchlogs.ListTagsLogGroupInput{
				LogGroupName: v.LogGroupName,
			})
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ARN:  *v.Arn,
				Name: *v.LogGroupName,
				Description: model.CloudWatchLogsLogGroupDescription{
					LogGroup: v,
					Tags:     tags.Tags,
				},
			})
		}
	}

	return values, nil
}

func CloudWatchLogsLogStream(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	logGroups, err := CloudWatchLogsLogGroup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, logGroup := range logGroups {
		client := cloudwatchlogs.NewFromConfig(cfg)
		paginator := cloudwatchlogs.NewDescribeLogStreamsPaginator(client, &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: logGroup.Description.(model.CloudWatchLogsLogGroupDescription).LogGroup.LogGroupName,
			Limit:        aws.Int32(50),
			OrderBy:      logstypes.OrderByLastEventTime,
			Descending:   aws.Bool(true),
		})

		// To avoid throttling, don't fetching everything. Only the first 5 pages!
		page := 0
		for paginator.HasMorePages() && page < 5 {
			page++

			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.LogStreams {
				values = append(values, Resource{
					ARN:         *v.Arn,
					Name:        *v.LogStreamName,
					Description: v,
				})
			}
		}
	}

	return values, nil
}

func CloudWatchLogsMetricFilter(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatchlogs.NewFromConfig(cfg)
	paginator := cloudwatchlogs.NewDescribeMetricFiltersPaginator(client, &cloudwatchlogs.DescribeMetricFiltersInput{})

	describeCtx := GetDescribeContext(ctx)

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.MetricFilters {
			arn := "arn:" + describeCtx.Partition + ":logs:" + describeCtx.Region + ":" + describeCtx.AccountID + ":log-group:" + *v.LogGroupName + ":metric-filter:" + *v.FilterName
			values = append(values, Resource{
				ARN:  arn,
				ID:   *v.FilterName,
				Name: *v.FilterName,
				Description: model.CloudWatchLogsMetricFilterDescription{
					MetricFilter: v,
				},
			})
		}
	}

	return values, nil
}

func CloudWatchLogsQueryDefinition(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatchlogs.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeQueryDefinitions(ctx, &cloudwatchlogs.DescribeQueryDefinitionsInput{NextToken: prevToken})
		if err != nil {
			return nil, err
		}

		for _, v := range output.QueryDefinitions {
			values = append(values, Resource{
				ID:          *v.QueryDefinitionId,
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

func CloudWatchLogsResourcePolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatchlogs.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeResourcePolicies(ctx, &cloudwatchlogs.DescribeResourcePoliciesInput{NextToken: prevToken})
		if err != nil {
			return nil, err
		}

		for _, v := range output.ResourcePolicies {
			values = append(values, Resource{
				ID:          *v.PolicyName,
				Name:        *v.PolicyName,
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

func CloudWatchLogsSubscriptionFilter(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	logGroups, err := CloudWatchLogsLogGroup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, logGroup := range logGroups {
		client := cloudwatchlogs.NewFromConfig(cfg)

		paginator := cloudwatchlogs.NewDescribeSubscriptionFiltersPaginator(client, &cloudwatchlogs.DescribeSubscriptionFiltersInput{
			LogGroupName: logGroup.Description.(model.CloudWatchLogsLogGroupDescription).LogGroup.LogGroupName,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.SubscriptionFilters {
				values = append(values, Resource{
					ID:          CompositeID(*v.LogGroupName, *v.FilterName),
					Name:        *v.LogGroupName,
					Description: v,
				})
			}
		}
	}

	return values, nil
}
