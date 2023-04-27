package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/machinelearningservices/mgmt/2020-02-18-preview/machinelearningservices"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2021-04-01-preview/insights"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func MachineLearningWorkspace(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
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
			resource := Resource{
				ID:       *workspace.ID,
				Name:     *workspace.Name,
				Location: *workspace.Location,
				Description: model.MachineLearningWorkspaceDescription{
					Workspace:                   workspace,
					DiagnosticSettingsResources: machineLearningServicesListOp.Value,
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
