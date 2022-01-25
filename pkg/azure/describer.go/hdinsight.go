package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/hdinsight/mgmt/2018-06-01/hdinsight"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func HdInsightCluster(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	client := hdinsight.NewClustersClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, cluster := range result.Values() {
			hdinsightListOp, err := insightsClient.List(ctx, *cluster.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID: *cluster.ID,
				Description: model.HdinsightClusterDescription{
					cluster,
					hdinsightListOp,
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
