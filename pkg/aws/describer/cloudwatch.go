package describer

import (
	"context"
	"fmt"
	"strings"
	"time"

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

func GetCloudWatchAlarm(ctx context.Context, cfg aws.Config, alarmName string) ([]Resource, error) {
	client := cloudwatch.NewFromConfig(cfg)
	out, err := client.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{
		AlarmNames: []string{alarmName},
	})
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, v := range out.MetricAlarms {
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

func CloudWatchLogEvent(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatchlogs.NewFromConfig(cfg)
	logGroups, err := CloudWatchLogsLogGroup(ctx, cfg)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, logGroup := range logGroups {
		logGroupName := logGroup.Description.(model.CloudWatchLogsLogGroupDescription).LogGroup.LogGroupName
		paginator := cloudwatchlogs.NewFilterLogEventsPaginator(client, &cloudwatchlogs.FilterLogEventsInput{
			LogGroupName: logGroupName,
			Limit:        aws.Int32(100),
		})

		// To avoid throttling, don't fetching everything. Only the first 5 pages!
		pageNo := 0
		for paginator.HasMorePages() && pageNo < 5 {
			pageNo++
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.Events {
				values = append(values, Resource{
					ID:   *v.EventId,
					Name: *v.LogStreamName,
					Description: model.CloudWatchLogEventDescription{
						LogEvent:     v,
						LogGroupName: *logGroupName,
					},
				})
			}
		}
	}

	return values, nil
}

func CloudWatchLogsResourcePolicy(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatchlogs.NewFromConfig(cfg)

	var values []Resource
	err := PaginateRetrieveAll(func(prevToken *string) (nextToken *string, err error) {
		output, err := client.DescribeResourcePolicies(ctx, &cloudwatchlogs.DescribeResourcePoliciesInput{
			NextToken: prevToken,
		})
		if err != nil {
			return nil, err
		}

		for _, v := range output.ResourcePolicies {
			values = append(values, Resource{
				Name: *v.PolicyName,
				Description: model.CloudWatchLogResourcePolicyDescription{
					ResourcePolicy: v,
				},
			})
		}

		return output.NextToken, nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func CloudWatchLogStream(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatchlogs.NewFromConfig(cfg)
	logGroups, err := CloudWatchLogsLogGroup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, logGroup := range logGroups {
		logGroupName := logGroup.Description.(model.CloudWatchLogsLogGroupDescription).LogGroup.LogGroupName
		paginator := cloudwatchlogs.NewDescribeLogStreamsPaginator(client, &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: logGroupName,
			Limit:        aws.Int32(50),
			OrderBy:      logstypes.OrderByLastEventTime,
			Descending:   aws.Bool(true),
		})

		// To avoid throttling, don't fetching everything. Only the first 5 pages!
		pageNo := 0
		for paginator.HasMorePages() && pageNo < 5 {
			pageNo++

			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.LogStreams {
				values = append(values, Resource{
					ARN:  *v.Arn,
					Name: *v.LogStreamName,
					Description: model.CloudWatchLogStreamDescription{
						LogStream:    v,
						LogGroupName: *logGroupName,
					},
				})
			}
		}
	}

	return values, nil
}

func CloudWatchLogsSubscriptionFilter(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	describeCtx := GetDescribeContext(ctx)
	client := cloudwatchlogs.NewFromConfig(cfg)
	logGroups, err := CloudWatchLogsLogGroup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, logGroup := range logGroups {
		logGroupName := logGroup.Description.(model.CloudWatchLogsLogGroupDescription).LogGroup.LogGroupName
		paginator := cloudwatchlogs.NewDescribeSubscriptionFiltersPaginator(client, &cloudwatchlogs.DescribeSubscriptionFiltersInput{
			LogGroupName: logGroupName,
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, v := range page.SubscriptionFilters {
				arn := fmt.Sprintf("arn:%s:logs:%s:%s:log-group:%s:subscription-filter:%s", describeCtx.Partition, describeCtx.Region, describeCtx.AccountID, *logGroupName, *v.FilterName)
				values = append(values, Resource{
					ID:   CompositeID(*v.LogGroupName, *v.FilterName),
					ARN:  arn,
					Name: *v.LogGroupName,
					Description: model.CloudWatchLogSubscriptionFilterDescription{
						SubscriptionFilter: v,
						LogGroupName:       *logGroupName,
					},
				})
			}
		}
	}

	return values, nil
}

func CloudWatchMetrics(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := cloudwatch.NewFromConfig(cfg)
	paginator := cloudwatch.NewListMetricsPaginator(client, &cloudwatch.ListMetricsInput{})

	var values []Resource
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, v := range page.Metrics {
			values = append(values, Resource{
				Name: *v.MetricName,
				Description: model.CloudWatchMetricDescription{
					Metric: v,
				},
			})
		}
	}

	return values, nil
}

func getCloudWatchPeriodForGranularity(granularity string) int32 {
	switch strings.ToUpper(granularity) {
	case "DAILY":
		// 24 hours
		return 86400
	case "HOURLY":
		// 1 hour
		return 3600
	}
	// else 5 minutes
	return 300
}

func getCloudWatchStartDateForGranularity(granularity string) time.Time {
	switch strings.ToUpper(granularity) {
	case "DAILY":
		// 1 year
		return time.Now().AddDate(-1, 0, 0)
	case "HOURLY":
		// 60 days
		return time.Now().AddDate(0, 0, -60)
	}
	// else 5 days
	return time.Now().AddDate(0, 0, -5)
}

func listCloudWatchMetricStatistics(ctx context.Context, cfg aws.Config, granularity string, namespace string, metricName string, dimensionName string, dimensionValue string) ([]model.CloudWatchMetricRow, error) {
	client := cloudwatch.NewFromConfig(cfg)
	endTime := time.Now()
	startTime := getCloudWatchStartDateForGranularity(granularity)
	period := getCloudWatchPeriodForGranularity(granularity)

	params := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricName),
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(period),
		Statistics: []types.Statistic{
			types.StatisticAverage,
			types.StatisticSampleCount,
			types.StatisticSum,
			types.StatisticMinimum,
			types.StatisticMaximum,
		},
	}

	if dimensionName != "" && dimensionValue != "" {
		params.Dimensions = []types.Dimension{
			{
				Name:  aws.String(dimensionName),
				Value: aws.String(dimensionValue),
			},
		}
	}

	stats, err := client.GetMetricStatistics(ctx, params)
	if err != nil {
		return nil, err
	}

	var values []model.CloudWatchMetricRow
	for _, datapoint := range stats.Datapoints {
		values = append(values, model.CloudWatchMetricRow{
			DimensionValue: aws.String(dimensionValue),
			DimensionName:  aws.String(dimensionName),
			Namespace:      aws.String(namespace),
			MetricName:     aws.String(metricName),
			Average:        datapoint.Average,
			Maximum:        datapoint.Maximum,
			Minimum:        datapoint.Minimum,
			Timestamp:      datapoint.Timestamp,
			SampleCount:    datapoint.SampleCount,
			Sum:            datapoint.Sum,
			Unit:           aws.String(fmt.Sprint(datapoint.Unit)),
		})
	}

	return values, nil
}
