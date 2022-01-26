package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/kusto/mgmt/2021-01-01/kusto"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func KustoCluster(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	kustoClient := kusto.NewClustersClient(subscription)
	kustoClient.Authorizer = authorizer
	result, err := kustoClient.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, kusto := range *result.Value {
		resourceGroup := strings.Split(*kusto.ID, "/")[4]

		values = append(values, Resource{
			ID: *kusto.ID,
			Description: model.KustoClusterDescription{
				Cluster:       kusto,
				ResourceGroup: resourceGroup,
			},
		})
	}
	return values, nil
}
