package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/hybridcompute/mgmt/hybridcompute"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func HybridComputeMachine(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	hybridComputeClient := hybridcompute.NewMachineExtensionsClient(subscription)
	hybridComputeClient.Authorizer = authorizer

	client := hybridcompute.NewMachinesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListBySubscription(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, machine := range result.Values() {
			resourceGroup := strings.Split(*machine.ID, "/")[4]

			hybridComputeListResult, err := hybridComputeClient.List(ctx, resourceGroup, *machine.Name, "")
			if err != nil {
				return nil, err
			}
			v := hybridComputeListResult.Values()
			for hybridComputeListResult.NotDone() {
				err := hybridComputeListResult.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				v = append(v, hybridComputeListResult.Values()...)
			}

			values = append(values, Resource{
				ID:       *machine.ID,
				Name:     *machine.Name,
				Location: *machine.Location,
				Description: model.HybridComputeMachineDescription{
					Machine:           machine,
					MachineExtensions: v,
					ResourceGroup:     resourceGroup,
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
