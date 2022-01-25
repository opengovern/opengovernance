package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/guestconfiguration/mgmt/2020-06-25/guestconfiguration"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func ComputeDisk(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := compute.NewDisksClient(subscription)
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
					model.ComputeDiskDescription{
						Disk: v,
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

func ComputeDiskAccess(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := compute.NewDiskAccessesClient(subscription)
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
					model.ComputeDiskAccessDescription{
						DiskAccess: v,
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

func ComputeVirtualMachineScaleSet(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := compute.NewVirtualMachineScaleSetsClient(subscription)
	client.Authorizer = authorizer

	clientExtension := compute.NewVirtualMachineScaleSetExtensionsClient(subscription)
	clientExtension.Authorizer = authorizer

	result, err := client.ListAll(context.Background())
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			resourceGroupName := strings.Split(*v.ID, "/")[4]

			op, err := clientExtension.List(context.Background(), resourceGroupName, *v.Name)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID: *v.ID,
				Description: JSONAllFieldsMarshaller{
					model.ComputeVirtualMachineScaleSetDescription{
						VirtualMachineScaleSet:           v,
						VirtualMachineScaleSetExtensions: op.Values(),
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
func ComputeVirtualMachine(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	guestConfigurationClient := guestconfiguration.NewAssignmentsClient(subscription)
	guestConfigurationClient.Authorizer = authorizer

	computeClient := compute.NewVirtualMachineExtensionsClient(subscription)
	computeClient.Authorizer = authorizer

	networkClient := network.NewInterfacesClient(subscription)
	networkClient.Authorizer = authorizer

	networkPublicIPClient := network.NewPublicIPAddressesClient(subscription)
	networkClient.Authorizer = authorizer

	client := compute.NewVirtualMachinesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListAll(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, virtualMachine := range result.Values() {
			resourceGroupName := strings.Split(*virtualMachine.ID, "/")[4]
			computeInstanceViewOp, err := client.InstanceView(ctx, resourceGroupName, *virtualMachine.Name)
			if err != nil {
				return nil, err
			}

			var ipConfigs []network.InterfaceIPConfiguration
			for _, nicRef := range *virtualMachine.NetworkProfile.NetworkInterfaces {
				pathParts := strings.Split(*nicRef.ID, "/")
				resourceGroupName := pathParts[4]
				nicName := pathParts[len(pathParts)-1]

				nic, err := networkClient.Get(ctx, resourceGroupName, nicName, "")
				if err != nil {
					return nil, err
				}

				ipConfigs = append(ipConfigs, *nic.IPConfigurations...)
			}

			var publicIPs []string
			for _, ipConfig := range ipConfigs {
				if ipConfig.PublicIPAddress != nil && ipConfig.PublicIPAddress.ID != nil {
					pathParts := strings.Split(*ipConfig.PublicIPAddress.ID, "/")
					resourceGroup := pathParts[4]
					name := pathParts[len(pathParts)-1]

					publicIP, err := networkPublicIPClient.Get(ctx, resourceGroup, name, "")

					if err != nil {
						return nil, err
					}
					if publicIP.IPAddress != nil {
						publicIPs = append(publicIPs, *publicIP.IPAddress)
					}
				}
			}

			computeListOp, err := computeClient.List(ctx, resourceGroupName, *virtualMachine.Name, "")
			if err != nil {
				return nil, err
			}

			configurationListOp, err := guestConfigurationClient.List(ctx, resourceGroupName, *virtualMachine.Name)
			if err != nil {
				if !strings.Contains(err.Error(), "404") {
					return nil, err
				}
			}
			values = append(values, Resource{
				ID: *virtualMachine.ID,
				Description: model.ComputeVirtualMachineDescription{
					virtualMachine,
					computeInstanceViewOp,
					ipConfigs,
					publicIPs,
					computeListOp,
					configurationListOp,
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
