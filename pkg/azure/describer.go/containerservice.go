package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-02-01/containerservice"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func KubernetesCluster(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := containerservice.NewManagedClustersClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			resourceGroup := strings.Split(*v.ID, "/")[4]

			values = append(values, Resource{
				ID: *v.ID,
				Description: model.KubernetesClusterDescription{
					ManagedCluster: v,
					ResourceGroup: resourceGroup,
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
