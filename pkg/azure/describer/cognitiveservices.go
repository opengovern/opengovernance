package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/cognitiveservices/mgmt/2021-04-30/cognitiveservices"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2021-04-01-preview/insights"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func CognitiveAccount(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := insights.NewDiagnosticSettingsClient(subscription)
	client.Authorizer = authorizer

	accountsClient := cognitiveservices.NewAccountsClient(subscription)
	accountsClient.Authorizer = authorizer

	result, err := accountsClient.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, account := range result.Values() {
			resourceGroup := strings.Split(*account.ID, "/")[4]

			id := *account.ID
			cognitiveservicesListOp, err := client.List(ctx, id)
			if err != nil {
				return nil, err
			}
			resource := Resource{
				ID:       *account.ID,
				Name:     *account.Name,
				Location: *account.Location,
				Description: model.CognitiveAccountDescription{
					Account:                     account,
					DiagnosticSettingsResources: cognitiveservicesListOp.Value,
					ResourceGroup:               resourceGroup,
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
