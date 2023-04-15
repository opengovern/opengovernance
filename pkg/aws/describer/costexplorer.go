package describer

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func setRowMetrics(row *model.CostExplorerRow, metrics map[string]types.MetricValue) {
	if _, ok := metrics["BlendedCost"]; ok {
		row.BlendedCostAmount = metrics["BlendedCost"].Amount
		row.BlendedCostUnit = metrics["BlendedCost"].Unit
	}
	if _, ok := metrics["UnblendedCost"]; ok {
		row.UnblendedCostAmount = metrics["UnblendedCost"].Amount
		row.UnblendedCostUnit = metrics["UnblendedCost"].Unit
	}
	if _, ok := metrics["NetUnblendedCost"]; ok {
		row.NetUnblendedCostAmount = metrics["NetUnblendedCost"].Amount
		row.NetUnblendedCostUnit = metrics["NetUnblendedCost"].Unit
	}
	if _, ok := metrics["AmortizedCost"]; ok {
		row.AmortizedCostAmount = metrics["AmortizedCost"].Amount
		row.AmortizedCostUnit = metrics["AmortizedCost"].Unit
	}
	if _, ok := metrics["NetAmortizedCost"]; ok {
		row.NetAmortizedCostAmount = metrics["NetAmortizedCost"].Amount
		row.NetAmortizedCostUnit = metrics["NetAmortizedCost"].Unit
	}
	if _, ok := metrics["UsageQuantity"]; ok {
		row.UsageQuantityAmount = metrics["UsageQuantity"].Amount
		row.UsageQuantityUnit = metrics["UsageQuantity"].Unit
	}
	if _, ok := metrics["NormalizedUsageAmount"]; ok {
		row.NormalizedUsageAmount = metrics["NormalizedUsageAmount"].Amount
		row.NormalizedUsageUnit = metrics["NormalizedUsageAmount"].Unit
	}
}

func costMonthly(ctx context.Context, cfg aws.Config, by string, startDate, endDate time.Time) ([]model.CostExplorerRow, error) {
	timeFormat := "2006-01-02"
	endTime := endDate.Format(timeFormat)
	startTime := startDate.Format(timeFormat)

	params := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: aws.String(startTime),
			End:   aws.String(endTime),
		},
		Granularity: types.GranularityMonthly,
		Metrics: []string{
			"BlendedCost",
			"UnblendedCost",
			"NetUnblendedCost",
			"AmortizedCost",
			"NetAmortizedCost",
			"UsageQuantity",
			"NormalizedUsageAmount",
		},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  aws.String(by),
			},
		},
	}

	client := costexplorer.NewFromConfig(cfg)

	var values []model.CostExplorerRow
	for {
		out, err := client.GetCostAndUsage(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, result := range out.ResultsByTime {

			// If there are no groupings, create a row from the totals
			if len(result.Groups) == 0 {
				var row model.CostExplorerRow

				row.Estimated = result.Estimated
				row.PeriodStart = result.TimePeriod.Start
				row.PeriodEnd = result.TimePeriod.End

				setRowMetrics(&row, result.Total)
				values = append(values, row)
			}
			// make a row per group
			for _, group := range result.Groups {
				var row model.CostExplorerRow

				row.Estimated = result.Estimated
				row.PeriodStart = result.TimePeriod.Start
				row.PeriodEnd = result.TimePeriod.End

				if len(group.Keys) > 0 {
					row.Dimension1 = aws.String(group.Keys[0])
					if len(group.Keys) > 1 {
						row.Dimension2 = aws.String(group.Keys[1])
					}
				}
				setRowMetrics(&row, group.Metrics)

				values = append(values, row)
			}
		}

		if out.NextPageToken == nil {
			break
		}

		params.NextPageToken = out.NextPageToken
	}

	return values, nil
}

func CostByServiceLastMonth(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	triggerType := GetTriggerTypeFromContext(ctx)
	startDate := time.Now().AddDate(0, -1, 0)
	if triggerType == enums.DescribeTriggerTypeInitialDiscovery {
		startDate = time.Now().AddDate(0, -3, -7)
	}
	costs, err := costMonthly(ctx, cfg, "SERVICE", startDate, time.Now())
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, cost := range costs {
		if cost.Dimension1 == nil {
			fmt.Println("Dimention is null")
			continue
		}
		values = append(values, Resource{
			ID:          "service-" + *cost.Dimension1 + "-cost-monthly",
			Description: model.CostExplorerByServiceMonthlyDescription{CostExplorerRow: cost},
		})
	}

	return values, nil
}

func CostByAccountLastMonth(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	triggerType := GetTriggerTypeFromContext(ctx)
	startDate := time.Now().AddDate(0, -1, 0)
	if triggerType == enums.DescribeTriggerTypeInitialDiscovery {
		startDate = time.Now().AddDate(0, -3, -7)
	}

	costs, err := costMonthly(ctx, cfg, "LINKED_ACCOUNT", startDate, time.Now())
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, cost := range costs {
		if cost.Dimension1 == nil {
			fmt.Println("Dimention is null")
			continue
		}
		values = append(values, Resource{
			ID:          "account-" + *cost.Dimension1 + "-cost-monthly",
			Description: model.CostExplorerByAccountMonthlyDescription{CostExplorerRow: cost},
		})
	}
	return values, nil
}

func costDaily(ctx context.Context, cfg aws.Config, by string, startDate, endDate time.Time) ([]model.CostExplorerRow, error) {
	timeFormat := "2006-01-02"
	endTime := endDate.Format(timeFormat)
	startTime := startDate.Format(timeFormat)

	params := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: aws.String(startTime),
			End:   aws.String(endTime),
		},
		Granularity: types.GranularityDaily,
		Metrics: []string{
			"BlendedCost",
			"UnblendedCost",
			"NetUnblendedCost",
			"AmortizedCost",
			"NetAmortizedCost",
			"UsageQuantity",
			"NormalizedUsageAmount",
		},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  aws.String(by),
			},
		},
	}

	client := costexplorer.NewFromConfig(cfg)

	var values []model.CostExplorerRow
	for {
		out, err := client.GetCostAndUsage(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, result := range out.ResultsByTime {

			// If there are no groupings, create a row from the totals
			if len(result.Groups) == 0 {
				var row model.CostExplorerRow

				row.Estimated = result.Estimated
				row.PeriodStart = result.TimePeriod.Start
				row.PeriodEnd = result.TimePeriod.End

				setRowMetrics(&row, result.Total)
				values = append(values, row)
			}
			// make a row per group
			for _, group := range result.Groups {
				var row model.CostExplorerRow

				row.Estimated = result.Estimated
				row.PeriodStart = result.TimePeriod.Start
				row.PeriodEnd = result.TimePeriod.End

				if len(group.Keys) > 0 {
					row.Dimension1 = aws.String(group.Keys[0])
					if len(group.Keys) > 1 {
						row.Dimension2 = aws.String(group.Keys[1])
					}
				}
				setRowMetrics(&row, group.Metrics)

				values = append(values, row)
			}
		}

		if out.NextPageToken == nil {
			break
		}

		params.NextPageToken = out.NextPageToken
	}

	return values, nil
}

func CostByServiceLastDay(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	triggerType := GetTriggerTypeFromContext(ctx)
	startDate := time.Now().AddDate(0, 0, -7)
	if triggerType == enums.DescribeTriggerTypeInitialDiscovery {
		startDate = time.Now().AddDate(0, -3, -7)
	}

	costs, err := costDaily(ctx, cfg, "SERVICE", startDate, time.Now())
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, cost := range costs {
		if cost.Dimension1 == nil {
			fmt.Println("Dimention is null")
			continue
		}
		values = append(values, Resource{
			ID:          "service-" + *cost.Dimension1 + "-cost-" + *cost.PeriodEnd,
			Description: model.CostExplorerByServiceDailyDescription{CostExplorerRow: cost},
		})
	}

	return values, nil
}

func CostByAccountLastDay(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	triggerType := GetTriggerTypeFromContext(ctx)
	startDate := time.Now().AddDate(0, 0, -7)
	if triggerType == enums.DescribeTriggerTypeInitialDiscovery {
		startDate = time.Now().AddDate(0, -3, -7)
	}

	costs, err := costDaily(ctx, cfg, "LINKED_ACCOUNT", startDate, time.Now())
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, cost := range costs {
		if cost.Dimension1 == nil {
			fmt.Println("Dimention is null")
			continue
		}
		values = append(values, Resource{
			ID:          "account-" + *cost.Dimension1 + "-cost-" + *cost.PeriodEnd,
			Description: model.CostExplorerByAccountDailyDescription{CostExplorerRow: cost},
		})
	}
	return values, nil
}

func buildCostByRecordTypeInput(granularity string) *costexplorer.GetCostAndUsageInput {
	timeFormat := "2006-01-02"
	if granularity == "HOURLY" {
		timeFormat = "2006-01-02T15:04:05Z"
	}
	endTime := time.Now().Format(timeFormat)
	startTime := time.Now().AddDate(0, -1, 0).Format(timeFormat)

	params := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: aws.String(startTime),
			End:   aws.String(endTime),
		},
		Granularity: types.Granularity(granularity),
		Metrics: []string{
			"BlendedCost",
			"UnblendedCost",
			"NetUnblendedCost",
			"AmortizedCost",
			"NetAmortizedCost",
			"UsageQuantity",
			"NormalizedUsageAmount",
		},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionType("DIMENSION"),
				Key:  aws.String("LINKED_ACCOUNT"),
			},
			{
				Type: types.GroupDefinitionType("DIMENSION"),
				Key:  aws.String("RECORD_TYPE"),
			},
		},
	}

	return params
}

func CostByRecordTypeLastMonth(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := costexplorer.NewFromConfig(cfg)

	params := buildCostByRecordTypeInput("MONTHLY")

	out, err := client.GetCostAndUsage(ctx, params)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, result := range out.ResultsByTime {
		for _, group := range result.Groups {
			var row model.CostExplorerRow

			row.Estimated = result.Estimated
			row.PeriodStart = result.TimePeriod.Start
			row.PeriodEnd = result.TimePeriod.End

			if len(group.Keys) > 0 {
				row.Dimension1 = aws.String(group.Keys[0])
				if len(group.Keys) > 1 {
					row.Dimension2 = aws.String(group.Keys[1])
				}
			}
			setRowMetrics(&row, group.Metrics)

			values = append(values, Resource{
				ID:          "account-" + *row.Dimension1 + "-" + *row.Dimension2 + "-cost-monthly",
				Description: model.CostExplorerByRecordTypeMonthlyDescription{CostExplorerRow: row},
			})
		}
	}

	return values, nil
}

func CostByRecordTypeLastDay(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := costexplorer.NewFromConfig(cfg)

	params := buildCostByRecordTypeInput("DAILY")

	out, err := client.GetCostAndUsage(ctx, params)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, result := range out.ResultsByTime {
		for _, group := range result.Groups {
			var row model.CostExplorerRow

			row.Estimated = result.Estimated
			row.PeriodStart = result.TimePeriod.Start
			row.PeriodEnd = result.TimePeriod.End

			if len(group.Keys) > 0 {
				row.Dimension1 = aws.String(group.Keys[0])
				if len(group.Keys) > 1 {
					row.Dimension2 = aws.String(group.Keys[1])
				}
			}
			setRowMetrics(&row, group.Metrics)

			values = append(values, Resource{
				ID:          "account-" + *row.Dimension1 + "-" + *row.Dimension2 + "-cost-" + *row.PeriodEnd,
				Description: model.CostExplorerByRecordTypeDailyDescription{CostExplorerRow: row},
			})
		}
	}

	return values, nil
}

func buildCostByServiceAndUsageInput(granularity string) *costexplorer.GetCostAndUsageInput {
	timeFormat := "2006-01-02"
	if granularity == "HOURLY" {
		timeFormat = "2006-01-02T15:04:05Z"
	}
	endTime := time.Now().Format(timeFormat)
	startTime := time.Now().AddDate(0, -1, 0).Format(timeFormat)

	params := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: aws.String(startTime),
			End:   aws.String(endTime),
		},
		Granularity: types.Granularity(granularity),
		Metrics: []string{
			"BlendedCost",
			"UnblendedCost",
			"NetUnblendedCost",
			"AmortizedCost",
			"NetAmortizedCost",
			"UsageQuantity",
			"NormalizedUsageAmount",
		},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionType("DIMENSION"),
				Key:  aws.String("SERVICE"),
			},
			{
				Type: types.GroupDefinitionType("DIMENSION"),
				Key:  aws.String("USAGE_TYPE"),
			},
		},
	}

	return params
}

func CostByServiceUsageLastMonth(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := costexplorer.NewFromConfig(cfg)

	params := buildCostByServiceAndUsageInput("MONTHLY")

	out, err := client.GetCostAndUsage(ctx, params)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, result := range out.ResultsByTime {
		for _, group := range result.Groups {
			var row model.CostExplorerRow

			row.Estimated = result.Estimated
			row.PeriodStart = result.TimePeriod.Start
			row.PeriodEnd = result.TimePeriod.End

			if len(group.Keys) > 0 {
				row.Dimension1 = aws.String(group.Keys[0])
				if len(group.Keys) > 1 {
					row.Dimension2 = aws.String(group.Keys[1])
				}
			}
			setRowMetrics(&row, group.Metrics)

			values = append(values, Resource{
				ID:          "service-" + *row.Dimension1 + "-" + *row.Dimension2 + "-cost-monthly",
				Description: model.CostExplorerByServiceUsageTypeMonthlyDescription{CostExplorerRow: row},
			})
		}
	}

	return values, nil
}

func CostByServiceUsageLastDay(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := costexplorer.NewFromConfig(cfg)

	params := buildCostByServiceAndUsageInput("DAILY")

	out, err := client.GetCostAndUsage(ctx, params)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, result := range out.ResultsByTime {
		for _, group := range result.Groups {
			var row model.CostExplorerRow

			row.Estimated = result.Estimated
			row.PeriodStart = result.TimePeriod.Start
			row.PeriodEnd = result.TimePeriod.End

			if len(group.Keys) > 0 {
				row.Dimension1 = aws.String(group.Keys[0])
				if len(group.Keys) > 1 {
					row.Dimension2 = aws.String(group.Keys[1])
				}
			}
			setRowMetrics(&row, group.Metrics)

			values = append(values, Resource{
				ID:          "service-" + *row.Dimension1 + "-" + *row.Dimension2 + "-cost-" + *row.PeriodEnd,
				Description: model.CostExplorerByServiceUsageTypeDailyDescription{CostExplorerRow: row},
			})
		}
	}

	return values, nil
}

func buildCostForecastInput(granularity string) *costexplorer.GetCostForecastInput {
	metric := "UNBLENDED_COST"

	timeFormat := "2006-01-02"
	startTime := time.Now().UTC().Format(timeFormat)
	endTime := time.Now().AddDate(0, -1, 0).Format(timeFormat)

	params := &costexplorer.GetCostForecastInput{
		TimePeriod: &types.DateInterval{
			Start: aws.String(startTime),
			End:   aws.String(endTime),
		},
		Granularity: types.Granularity(granularity),
		Metric:      types.Metric(metric),
	}

	return params
}

func CostForecastMonthly(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := costexplorer.NewFromConfig(cfg)

	params := buildCostForecastInput("MONTHLY")
	output, err := client.GetCostForecast(ctx, params)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, forecast := range output.ForecastResultsByTime {
		values = append(values, Resource{
			ID: "forecast-monthly",
			Description: model.CostExplorerForcastMonthlyDescription{CostExplorerRow: model.CostExplorerRow{
				Estimated:   true,
				PeriodStart: forecast.TimePeriod.Start,
				PeriodEnd:   forecast.TimePeriod.End,
				MeanValue:   forecast.MeanValue}},
		})
	}

	return values, nil
}

func CostForecastDaily(ctx context.Context, cfg aws.Config) ([]Resource, error) {
	client := costexplorer.NewFromConfig(cfg)

	params := buildCostForecastInput("DAILY")
	output, err := client.GetCostForecast(ctx, params)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, forecast := range output.ForecastResultsByTime {
		values = append(values, Resource{
			ID: "forecast-daily",
			Description: model.CostExplorerForcastDailyDescription{CostExplorerRow: model.CostExplorerRow{
				Estimated:   true,
				PeriodStart: forecast.TimePeriod.Start,
				PeriodEnd:   forecast.TimePeriod.End,
				MeanValue:   forecast.MeanValue,
			}},
		})
	}

	return values, nil
}
