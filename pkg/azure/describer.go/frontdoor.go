package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/frontdoor/mgmt/2020-05-01/frontdoor"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func FrontDoor(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	client := frontdoor.NewFrontDoorsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, door := range result.Values() {
			resourceGroup := strings.Split(*door.ID, "/")[4]

			frontDoorListOp, err := insightsClient.List(ctx, *door.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *door.ID,
				Name:     *door.Name,
				Location: *door.Location,
				Description: model.FrontdoorDescription{
					FrontDoor:                   door,
					DiagnosticSettingsResources: frontDoorListOp.Value,
					ResourceGroup:               resourceGroup,
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
