package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/preview/machinelearningservices/mgmt/2020-02-18-preview/machinelearningservices"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func MachineLearningWorkspace(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := insights.NewDiagnosticSettingsClient(subscription)
	client.Authorizer = authorizer

	worspaceClient := machinelearningservices.NewWorkspacesClient(subscription)
	worspaceClient.Authorizer = authorizer

	result, err := worspaceClient.ListBySubscription(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, workspace := range result.Values() {
			resourceGroup := strings.Split(*workspace.ID, "/")[4]

			machineLearningServicesListOp, err := client.List(ctx, *workspace.ID)
			if err != nil {
				return nil, err
			}
			values = append(values, Resource{
				ID: *workspace.ID,
				Description: model.MachineLearningWorkspaceDescription{
					Workspace:                   workspace,
					DiagnosticSettingsResources: machineLearningServicesListOp.Value,
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
