package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/kusto/mgmt/2021-01-01/kusto"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func KustoCluster(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	kustoClient := kusto.NewClustersClient(subscription)
	kustoClient.Authorizer = authorizer
	result, err := kustoClient.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, kusto := range *result.Value {
		resourceGroup := strings.Split(*kusto.ID, "/")[4]

		resource := Resource{
			ID:       *kusto.ID,
			Name:     *kusto.Name,
			Location: *kusto.Location,
			Description: model.KustoClusterDescription{
				Cluster:       kusto,
				ResourceGroup: resourceGroup,
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
	return values, nil
}
