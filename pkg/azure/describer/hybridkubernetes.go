package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/hybridkubernetes/mgmt/2021-10-01/hybridkubernetes"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func HybridKubernetesConnectedCluster(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := hybridkubernetes.NewConnectedClusterClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListBySubscription(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, connectedCluster := range result.Values() {
			resourceGroup := strings.Split(*connectedCluster.ID, "/")[4]

			resource := Resource{
				ID:       *connectedCluster.ID,
				Name:     *connectedCluster.Name,
				Location: *connectedCluster.Location,
				Description: model.HybridKubernetesConnectedClusterDescription{
					ConnectedCluster: connectedCluster,
					ResourceGroup:    resourceGroup,
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
