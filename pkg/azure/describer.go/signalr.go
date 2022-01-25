package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/signalr/mgmt/2020-05-01/signalr"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func SignalrService(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	client := signalr.NewClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListBySubscription(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, service := range result.Values() {
			signalrListOp, err := insightsClient.List(ctx, *service.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID: *service.ID,
				Description: model.SignalrServiceDescription{
					service,
					signalrListOp,
				},
			})
		}
		if !result.NotDone() {
			break
		}
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}
