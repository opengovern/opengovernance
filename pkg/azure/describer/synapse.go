package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/synapse/mgmt/2021-03-01/synapse"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func SynapseWorkspace(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	synapseClient := synapse.NewWorkspaceManagedSQLServerVulnerabilityAssessmentsClient(subscription)
	synapseClient.Authorizer = authorizer

	client := synapse.NewWorkspacesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, config := range result.Values() {
			resourceGroup := strings.Split(*config.ID, "/")[4]

			synapseListResult, err := synapseClient.List(ctx, resourceGroup, *config.Name)
			if err != nil {
				return nil, err
			}

			var serverVulnerabilityAssessments []synapse.ServerVulnerabilityAssessment
			serverVulnerabilityAssessments = append(serverVulnerabilityAssessments, synapseListResult.Values()...)

			for synapseListResult.NotDone() {
				err = synapseListResult.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}
				serverVulnerabilityAssessments = append(serverVulnerabilityAssessments, synapseListResult.Values()...)
			}

			synapseListOp, err := insightsClient.List(ctx, *config.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *config.ID,
				Name:     *config.Name,
				Location: *config.Location,
				Description: model.SynapseWorkspaceDescription{
					Workspace:                      config,
					ServerVulnerabilityAssessments: serverVulnerabilityAssessments,
					DiagnosticSettingsResources:    synapseListOp.Value,
					ResourceGroup:                  resourceGroup,
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
