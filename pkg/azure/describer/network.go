package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	newnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
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
			resourceGroup := strings.Split(*v.ID, "/")[4]

			values = append(values, Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: *v.Location,
				Description: model.NetworkInterfaceDescription{
					Interface:     v,
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
					ID:       *v.ID,
					Name:     *v.Name,
					Location: *v.Location,
					Description: model.NetworkWatcherFlowLogDescription{
						NetworkWatcherName: *networkWatcherDetails.Name,
						FlowLog:            v,
						ResourceGroup:      resourceGroupID,
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
						ID:       *v.ID,
						Name:     *v.Name,
						Location: "global",
						Description: model.SubnetDescription{
							VirtualNetworkName: *virtualNetwork.Name,
							Subnet:             v,
							ResourceGroup:      *resourceGroupName,
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
			resourceGroup := strings.Split(*v.ID, "/")[4]

			values = append(values, Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: *v.Location,
				Description: model.VirtualNetworkDescription{
					VirtualNetwork: v,
					ResourceGroup:  resourceGroup,
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
func ApplicationGateway(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	client := newnetwork.NewApplicationGatewaysClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, gateway := range result.Values() {
			resourceGroup := strings.Split(*gateway.ID, "/")[4]

			networkListOp, err := insightsClient.List(ctx, *gateway.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *gateway.ID,
				Name:     *gateway.Name,
				Location: *gateway.Location,
				Description: model.ApplicationGatewayDescription{
					ApplicationGateway:          gateway,
					DiagnosticSettingsResources: networkListOp.Value,
					ResourceGroup:               resourceGroup,
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

func NetworkSecurityGroup(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := insights.NewDiagnosticSettingsClient(subscription)
	client.Authorizer = authorizer

	NetworkSecurityGroupClient := newnetwork.NewSecurityGroupsClient(subscription)
	NetworkSecurityGroupClient.Authorizer = authorizer

	result, err := NetworkSecurityGroupClient.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, networkSecurityGroup := range result.Values() {
			resourceGroup := strings.Split(*networkSecurityGroup.ID, "/")[4]

			id := *networkSecurityGroup.ID
			networkListOp, err := client.List(ctx, id)
			if err != nil {
				if strings.Contains(err.Error(), "ResourceNotFound") || strings.Contains(err.Error(), "SubscriptionNotRegistered") {
					// ignore
				} else {
					return nil, err
				}
			}

			values = append(values, Resource{
				ID:       *networkSecurityGroup.ID,
				Name:     *networkSecurityGroup.Name,
				Location: *networkSecurityGroup.Location,
				Description: model.NetworkSecurityGroupDescription{
					SecurityGroup:               networkSecurityGroup,
					DiagnosticSettingsResources: networkListOp.Value,
					ResourceGroup:               resourceGroup,
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

func NetworkWatcher(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	networkWatcherClient := newnetwork.NewWatchersClient(subscription)
	networkWatcherClient.Authorizer = authorizer
	result, err := networkWatcherClient.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, networkWatcher := range *result.Value {
		resourceGroup := strings.Split(*networkWatcher.ID, "/")[4]

		values = append(values, Resource{
			ID:       *networkWatcher.ID,
			Name:     *networkWatcher.Name,
			Location: *networkWatcher.Location,
			Description: model.NetworkWatcherDescription{
				Watcher:       networkWatcher,
				ResourceGroup: resourceGroup,
			},
		})
	}

	return values, nil
}

func RouteTables(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := newnetwork.NewRouteTablesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, routeTable := range result.Values() {
			resourceGroup := strings.Split(*routeTable.ID, "/")[4]

			values = append(values, Resource{
				ID:       *routeTable.ID,
				Name:     *routeTable.Name,
				Location: *routeTable.Location,
				Description: model.RouteTablesDescription{
					ResourceGroup: resourceGroup,
					RouteTable:    routeTable,
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

func NetworkApplicationSecurityGroups(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := newnetwork.NewApplicationSecurityGroupsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for _, applicationSecurityGroup := range result.Values() {
		resourceGroup := strings.Split(*applicationSecurityGroup.ID, "/")[4]

		values = append(values, Resource{
			ID:       *applicationSecurityGroup.ID,
			Name:     *applicationSecurityGroup.Name,
			Location: *applicationSecurityGroup.Location,
			Description: model.NetworkApplicationSecurityGroupsDescription{
				ApplicationSecurityGroup: applicationSecurityGroup,
				ResourceGroup:            resourceGroup,
			},
		})
	}

	for result.NotDone() {
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}

		for _, applicationSecurityGroup := range result.Values() {
			resourceGroup := strings.Split(*applicationSecurityGroup.ID, "/")[4]

			values = append(values, Resource{
				ID:       *applicationSecurityGroup.ID,
				Name:     *applicationSecurityGroup.Name,
				Location: *applicationSecurityGroup.Location,
				Description: model.NetworkApplicationSecurityGroupsDescription{
					ApplicationSecurityGroup: applicationSecurityGroup,
					ResourceGroup:            resourceGroup,
				},
			})
		}
	}

	return values, nil
}

func NetworkAzureFirewall(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := newnetwork.NewAzureFirewallsClient(subscription)
	client.Authorizer = authorizer
	result, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource

	for {
		for _, azureFirewall := range result.Values() {
			resourceGroup := strings.Split(*azureFirewall.ID, "/")[4]

			values = append(values, Resource{
				ID:       *azureFirewall.ID,
				Name:     *azureFirewall.Name,
				Location: *azureFirewall.Location,
				Description: model.NetworkAzureFirewallDescription{
					AzureFirewall: azureFirewall,
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
