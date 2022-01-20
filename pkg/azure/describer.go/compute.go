package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
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
					azure.ComputeDiskDescription{
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
					azure.ComputeDiskAccessDescription{
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
					azure.ComputeVirtualMachineScaleSetDescription{
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
