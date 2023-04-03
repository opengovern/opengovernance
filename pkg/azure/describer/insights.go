package describer

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2021-04-01-preview/insights"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func DiagnosticSetting(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	diagnosticSettingClient := insights.NewDiagnosticSettingsClient(subscription)
	diagnosticSettingClient.Authorizer = authorizer
	resourceURI := "/subscriptions/" + subscription
	result, err := diagnosticSettingClient.List(ctx, resourceURI)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, diagnosticSetting := range *result.Value {
		resourceGroup := strings.Split(*diagnosticSetting.ID, "/")[4]

		values = append(values, Resource{
			ID:       *diagnosticSetting.ID,
			Name:     *diagnosticSetting.Name,
			Location: "global",
			Description: model.DiagnosticSettingDescription{
				DiagnosticSettingsResource: diagnosticSetting,
				ResourceGroup:              resourceGroup,
			},
		})
	}
	return values, nil
}
func LogAlert(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	logAlertClient := insights.NewActivityLogAlertsClient(subscription)
	logAlertClient.Authorizer = authorizer
	result, err := logAlertClient.ListBySubscriptionID(ctx)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, logAlert := range *result.Value {
		resourceGroup := strings.Split(*logAlert.ID, "/")[4]

		values = append(values, Resource{
			ID:       *logAlert.ID,
			Name:     *logAlert.Name,
			Location: *logAlert.Location,
			Description: model.LogAlertDescription{
				ActivityLogAlertResource: logAlert,
				ResourceGroup:            resourceGroup,
			},
		})
	}

	return values, nil
}
func LogProfile(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	logProfileClient := insights.NewLogProfilesClient(subscription)
	logProfileClient.Authorizer = authorizer
	result, err := logProfileClient.List(ctx)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, logProfile := range *result.Value {
		resourceGroup := strings.Split(*logProfile.ID, "/")[4]
		location := ""
		if logProfile.Location != nil {
			location = *logProfile.Location
		}
		values = append(values, Resource{
			ID:       *logProfile.ID,
			Name:     *logProfile.Name,
			Location: location,
			Description: model.LogProfileDescription{
				LogProfileResource: logProfile,
				ResourceGroup:      resourceGroup,
			},
		})
	}

	return values, nil
}

func getMonitoringIntervalForGranularity(granularity string) string {
	switch strings.ToUpper(granularity) {
	case "DAILY":
		// 24 hours
		return "PT24H"
	case "HOURLY":
		// 1 hour
		return "PT1H"
	}
	// else 5 minutes
	return "PT5M"
}

func getMonitoringStartDateForGranularity(granularity string) string {
	switch strings.ToUpper(granularity) {
	case "DAILY":
		// Last 1 year
		return time.Now().UTC().AddDate(-1, 0, 0).Format(time.RFC3339)
	case "HOURLY":
		// Last 60 days
		return time.Now().UTC().AddDate(0, 0, -60).Format(time.RFC3339)
	}
	// Last 5 days
	return time.Now().UTC().AddDate(0, 0, -5).Format(time.RFC3339)
}

func listAzureMonitorMetricStatistics(ctx context.Context, authorizer autorest.Authorizer, subscription string, granularity string, metricNameSpace string, metricNames string, dimensionValue string) ([]model.MonitoringMetric, error) {
	metricClient := insights.NewMetricsClient(subscription)
	metricClient.Authorizer = authorizer

	interval := getMonitoringIntervalForGranularity(granularity)
	aggregation := "average,count,maximum,minimum,total"
	timeSpan := getMonitoringStartDateForGranularity(granularity) + "/" + time.Now().UTC().AddDate(0, 0, 1).Format(time.RFC3339) // Retrieve data within a year
	orderBy := "timestamp"
	top := int32(1000) // Maximum number of record fetch with given interval
	filter := ""

	result, err := metricClient.List(ctx, dimensionValue, timeSpan, &interval, metricNames, aggregation, &top, orderBy, filter, insights.ResultTypeData, metricNameSpace)
	if err != nil {
		return nil, err
	}

	var values []model.MonitoringMetric
	for _, metric := range *result.Value {
		for _, timeseries := range *metric.Timeseries {
			for _, data := range *timeseries.Data {
				if data.Average != nil {
					values = append(values, model.MonitoringMetric{
						DimensionValue: dimensionValue,
						TimeStamp:      data.TimeStamp.Format(time.RFC3339),
						Maximum:        data.Maximum,
						Minimum:        data.Minimum,
						Average:        data.Average,
						Sum:            data.Total,
						SampleCount:    data.Count,
						Unit:           string(metric.Unit),
					})
				}
			}
		}
	}

	return values, nil
}
