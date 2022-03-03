package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/servicefabric/mgmt/2019-03-01/servicefabric"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func ServiceFabricCluster(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	clusterClient := servicefabric.NewClustersClient(subscription)
	clusterClient.Authorizer = authorizer
	result, err := clusterClient.List(ctx)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, cluster := range *result.Value {
		resourceGroup := strings.Split(*cluster.ID, "/")[4]

		values = append(values, Resource{
			ID:          *cluster.ID,
			Location:    *cluster.Location,
			Description: model.ServiceFabricClusterDescription{Cluster: cluster, ResourceGroup: resourceGroup}},
		)
	}
	return values, nil
}
