package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
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

		values = append(values, Resource{
			ID:       *logProfile.ID,
			Location: *logProfile.Location,
			Description: model.LogProfileDescription{
				LogProfileResource: logProfile,
				ResourceGroup:      resourceGroup,
			},
		})
	}

	return values, nil
}
