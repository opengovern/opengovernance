package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/logic/mgmt/2019-05-01/logic"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func LogicAppWorkflow(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := insights.NewDiagnosticSettingsClient(subscription)
	client.Authorizer = authorizer

	workflowClient := logic.NewWorkflowsClient(subscription)
	workflowClient.Authorizer = authorizer

	result, err := workflowClient.ListBySubscription(ctx, nil, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, workflow := range result.Values() {
			resourceGroup := strings.Split(*workflow.ID, "/")[4]

			logicListOp, err := client.List(ctx, *workflow.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *workflow.ID,
				Location: *workflow.Location,
				Description: model.LogicAppWorkflowDescription{
					Workflow:                    workflow,
					DiagnosticSettingsResources: logicListOp.Value,
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
