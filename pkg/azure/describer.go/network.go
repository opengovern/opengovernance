package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"strings"
)

func NetworkInterface(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := network.NewInterfacesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			values = append(values, Resource{
				ID: *v.ID,
				Description: JSONAllFieldsMarshaller{
					azure.NetworkInterfaceDescription{
						Interface: v,
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

func NetworkWatcherFlowLog(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := network.NewFlowLogsClient(subscription)
	client.Authorizer = authorizer

	networkWatcherClient := network.NewWatchersClient(subscription)
	networkWatcherClient.Authorizer = authorizer

	resultWatchers, err := networkWatcherClient.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	if resultWatchers.Value == nil || len(*resultWatchers.Value) == 0 {
		return nil, nil
	}

	var values []Resource
	for _, networkWatcherDetails := range *resultWatchers.Value {
		resourceGroupID := strings.Split(*networkWatcherDetails.ID, "/")[4]
		result, err := client.List(ctx, resourceGroupID, *networkWatcherDetails.Name)
		if err != nil {
			return nil, err
		}

		for {
			for _, v := range result.Values() {
				values = append(values, Resource{
					ID: *v.ID,
					Description: JSONAllFieldsMarshaller{
						azure.NetworkWatcherFlowLogDescription{
							FlowLog: v,
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
	}

	return values, nil
}

//TODO-Saleh resource ??
func Subnet(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	networkClient := network.NewVirtualNetworksClient(subscription)
	networkClient.Authorizer = authorizer

	client := network.NewSubnetsClient(subscription)
	client.Authorizer = authorizer

	resultVirtualNetworks, err := networkClient.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, virtualNetwork := range resultVirtualNetworks.Values() {
			resourceGroupName := &strings.Split(*virtualNetwork.ID, "/")[4]
			result, err := client.List(ctx, *resourceGroupName, *virtualNetwork.Name)
			if err != nil {
				return nil, err
			}

			for {
				for _, v := range result.Values() {
					values = append(values, Resource{
						ID: *v.ID,
						Description: JSONAllFieldsMarshaller{
							azure.SubnetDescription{
								Subnet: v,
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
		}

		if !resultVirtualNetworks.NotDone() {
			break
		}

		err = resultVirtualNetworks.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func VirtualNetwork(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := network.NewVirtualNetworksClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			values = append(values, Resource{
				ID: *v.ID,
				Description: JSONAllFieldsMarshaller{
					azure.VirtualNetworkDescription{
						VirtualNetwork: v,
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
