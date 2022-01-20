package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-02-01/containerservice"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
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
			values = append(values, Resource{
				ID: *v.ID,
				Description: JSONAllFieldsMarshaller{
					azure.KubernetesClusterDescription{
						ManagedCluster: v,
					},
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
