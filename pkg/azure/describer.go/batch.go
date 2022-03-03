package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/batch/mgmt/2020-09-01/batch"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func BatchAccount(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := insights.NewDiagnosticSettingsClient(subscription)
	client.Authorizer = authorizer

	batchAccountClient := batch.NewAccountClient(subscription)
	batchAccountClient.Authorizer = authorizer

	result, err := batchAccountClient.List(context.Background())
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, account := range result.Values() {
			id := *account.ID
			batchListOp, err := client.List(ctx, id)
			if err != nil {
				return nil, err
			}
			splitID := strings.Split(*account.ID, "/")

			resourceGroup := splitID[4]

			values = append(values, Resource{
				ID:       *account.ID,
				Location: *account.Location,
				Description: model.BatchAccountDescription{
					Account:                     account,
					DiagnosticSettingsResources: batchListOp.Value,
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
