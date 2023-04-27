package describer

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-09-01/skus"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/guestconfiguration/mgmt/2020-06-25/guestconfiguration"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2021-04-01-preview/insights"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func ComputeDisk(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewDisksClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			resourceGroup := strings.Split(*v.ID, "/")[4]

			resource := Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: *v.Location,
				Description: model.ComputeDiskDescription{
					Disk:          v,
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

func ComputeDiskAccess(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewDiskAccessesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			resourceGroup := strings.Split(*v.ID, "/")[4]

			resource := Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: *v.Location,
				Description: model.ComputeDiskAccessDescription{
					DiskAccess:    v,
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

func ComputeVirtualMachineScaleSet(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
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

			resource := Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: *v.Location,
				Description: model.ComputeVirtualMachineScaleSetDescription{
					VirtualMachineScaleSet:           v,
					VirtualMachineScaleSetExtensions: op.Values(),
					ResourceGroup:                    resourceGroupName,
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

func ComputeVirtualMachineScaleSetNetworkInterface(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewVirtualMachineScaleSetsClient(subscription)
	client.Authorizer = authorizer

	networkClient := network.NewInterfacesClient(subscription)
	networkClient.Authorizer = authorizer

	vmList, err := client.ListAll(context.Background())
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, vm := range vmList.Values() {
			vmResourceGroupName := strings.Split(*vm.ID, "/")[4]

			result, err := networkClient.ListVirtualMachineScaleSetNetworkInterfaces(ctx, vmResourceGroupName, *vm.Name)
			if err != nil {
				return nil, err
			}
			for {
				for _, v := range result.Values() {
					resourceGroupName := strings.Split(*v.ID, "/")[4]
					resource := Resource{
						ID:       *v.ID,
						Name:     *v.Name,
						Location: *v.Location,
						Description: model.ComputeVirtualMachineScaleSetNetworkInterfaceDescription{
							VirtualMachineScaleSet: vm,
							NetworkInterface:       v,
							ResourceGroup:          resourceGroupName,
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
		}
		if !vmList.NotDone() {
			break
		}

		err = vmList.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func ComputeVirtualMachineScaleSetVm(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewVirtualMachineScaleSetsClient(subscription)
	client.Authorizer = authorizer

	ssVmClient := compute.NewVirtualMachineScaleSetVMsClient(subscription)
	ssVmClient.Authorizer = authorizer

	vmList, err := client.ListAll(context.Background())
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, vm := range vmList.Values() {
			vmResourceGroupName := strings.Split(*vm.ID, "/")[4]

			result, err := ssVmClient.List(ctx, vmResourceGroupName, *vm.Name, "", "", "")
			if err != nil {
				return nil, err
			}
			for {
				for _, v := range result.Values() {
					resourceGroupName := strings.Split(*v.ID, "/")[4]
					resource := Resource{
						ID:       *v.ID,
						Name:     *v.Name,
						Location: *v.Location,
						Description: model.ComputeVirtualMachineScaleSetVmDescription{
							VirtualMachineScaleSet: vm,
							ScaleSetVM:             v,
							ResourceGroup:          resourceGroupName,
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
		}
		if !vmList.NotDone() {
			break
		}

		err = vmList.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

func ComputeVirtualMachine(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
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
			resource := Resource{
				ID:       *virtualMachine.ID,
				Name:     *virtualMachine.Name,
				Location: *virtualMachine.Location,
				Description: model.ComputeVirtualMachineDescription{
					VirtualMachine:             virtualMachine,
					VirtualMachineInstanceView: computeInstanceViewOp,
					InterfaceIPConfigurations:  ipConfigs,
					PublicIPs:                  publicIPs,
					VirtualMachineExtension:    computeListOp.Value,
					Assignments:                configurationListOp.Value,
					ResourceGroup:              resourceGroupName,
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

func ComputeSnapshots(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewSnapshotsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, snapshot := range result.Values() {
			resourceGroupName := strings.Split(*snapshot.ID, "/")[4]

			resource := Resource{
				ID:       *snapshot.ID,
				Name:     *snapshot.Name,
				Location: *snapshot.Location,
				Description: model.ComputeSnapshotsDescription{
					ResourceGroup: resourceGroupName,
					Snapshot:      snapshot,
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

func ComputeAvailabilitySet(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewAvailabilitySetsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListBySubscription(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, availabilitySet := range result.Values() {
			resourceGroupName := strings.Split(*availabilitySet.ID, "/")[4]

			resource := Resource{
				ID:       *availabilitySet.ID,
				Name:     *availabilitySet.Name,
				Location: *availabilitySet.Location,
				Description: model.ComputeAvailabilitySetDescription{
					ResourceGroup:   resourceGroupName,
					AvailabilitySet: availabilitySet,
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

func ComputeDiskEncryptionSet(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewDiskEncryptionSetsClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, diskEncryptionSet := range result.Values() {
			resourceGroupName := strings.Split(*diskEncryptionSet.ID, "/")[4]

			resource := Resource{
				ID:       *diskEncryptionSet.ID,
				Name:     *diskEncryptionSet.Name,
				Location: *diskEncryptionSet.Location,
				Description: model.ComputeDiskEncryptionSetDescription{
					ResourceGroup:     resourceGroupName,
					DiskEncryptionSet: diskEncryptionSet,
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

func ComputeGallery(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewGalleriesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, gallery := range result.Values() {
			resourceGroupName := strings.Split(*gallery.ID, "/")[4]

			resource := Resource{
				ID:       *gallery.ID,
				Name:     *gallery.Name,
				Location: *gallery.Location,
				Description: model.ComputeGalleryDescription{
					ResourceGroup: resourceGroupName,
					Gallery:       gallery,
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

func ComputeImage(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewImagesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			resourceGroup := strings.ToLower(strings.Split(*v.ID, "/")[4])
			resource := Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: *v.Location,
				Description: model.ComputeImageDescription{
					Image:         v,
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

func ComputeDiskReadOps(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewDisksClient(subscription)
	client.Authorizer = authorizer

	clientInsight := insights.NewMetricsClient(subscription)
	clientInsight.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, disk := range result.Values() {
			if disk.ID == nil {
				continue
			}
			metrics, err := listAzureMonitorMetricStatistics(ctx, authorizer, subscription, "FIVE_MINUTES", "Microsoft.Compute/disks", "Composite Disk Read Operations/sec", *disk.ID)
			if err != nil {
				return nil, err
			}
			for _, metric := range metrics {
				resource := Resource{
					ID:       fmt.Sprintf("%s_readops", *disk.ID),
					Name:     fmt.Sprintf("%s readops", *disk.Name),
					Location: *disk.Location,
					Description: model.ComputeDiskReadOpsDescription{
						MonitoringMetric: metric,
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

func ComputeDiskReadOpsDaily(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewDisksClient(subscription)
	client.Authorizer = authorizer

	clientInsight := insights.NewMetricsClient(subscription)
	clientInsight.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, disk := range result.Values() {
			if disk.ID == nil {
				continue
			}
			metrics, err := listAzureMonitorMetricStatistics(ctx, authorizer, subscription, "DAILY", "Microsoft.Compute/disks", "Composite Disk Read Operations/sec", *disk.ID)
			if err != nil {
				return nil, err
			}
			for _, metric := range metrics {
				resource := Resource{
					ID:       fmt.Sprintf("%s_readops_daily", *disk.ID),
					Name:     fmt.Sprintf("%s readops-daily", *disk.Name),
					Location: *disk.Location,
					Description: model.ComputeDiskReadOpsDailyDescription{
						MonitoringMetric: metric,
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

func ComputeDiskReadOpsHourly(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewDisksClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, disk := range result.Values() {
			if disk.ID == nil {
				continue
			}
			metrics, err := listAzureMonitorMetricStatistics(ctx, authorizer, subscription, "HOURLY", "Microsoft.Compute/disks", "Composite Disk Read Operations/sec", *disk.ID)
			if err != nil {
				return nil, err
			}
			for _, metric := range metrics {
				resource := Resource{
					ID:       fmt.Sprintf("%s_readops_hourly", *disk.ID),
					Name:     fmt.Sprintf("%s readops-hourly", *disk.Name),
					Location: *disk.Location,
					Description: model.ComputeDiskReadOpsHourlyDescription{
						MonitoringMetric: metric,
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

func ComputeDiskWriteOps(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewDisksClient(subscription)
	client.Authorizer = authorizer

	clientInsight := insights.NewMetricsClient(subscription)
	clientInsight.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, disk := range result.Values() {
			if disk.ID == nil {
				continue
			}
			metrics, err := listAzureMonitorMetricStatistics(ctx, authorizer, subscription, "FIVE_MINUTES", "Microsoft.Compute/disks", "Composite Disk Write Operations/sec", *disk.ID)
			if err != nil {
				return nil, err
			}
			for _, metric := range metrics {
				resource := Resource{
					ID:       fmt.Sprintf("%s_writeops", *disk.ID),
					Name:     fmt.Sprintf("%s writeops", *disk.Name),
					Location: *disk.Location,
					Description: model.ComputeDiskWriteOpsDescription{
						MonitoringMetric: metric,
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

func ComputeDiskWriteOpsDaily(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewDisksClient(subscription)
	client.Authorizer = authorizer

	clientInsight := insights.NewMetricsClient(subscription)
	clientInsight.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, disk := range result.Values() {
			if disk.ID == nil {
				continue
			}
			metrics, err := listAzureMonitorMetricStatistics(ctx, authorizer, subscription, "DAILY", "Microsoft.Compute/disks", "Composite Disk Write Operations/sec", *disk.ID)
			if err != nil {
				return nil, err
			}
			for _, metric := range metrics {
				resource := Resource{
					ID:       fmt.Sprintf("%s_writeops_daily", *disk.ID),
					Name:     fmt.Sprintf("%s writeops-daily", *disk.Name),
					Location: *disk.Location,
					Description: model.ComputeDiskWriteOpsDailyDescription{
						MonitoringMetric: metric,
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

func ComputeDiskWriteOpsHourly(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewDisksClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, disk := range result.Values() {
			if disk.ID == nil {
				continue
			}
			metrics, err := listAzureMonitorMetricStatistics(ctx, authorizer, subscription, "HOURLY", "Microsoft.Compute/disks", "Composite Disk Write Operations/sec", *disk.ID)
			if err != nil {
				return nil, err
			}
			for _, metric := range metrics {
				resource := Resource{
					ID:       fmt.Sprintf("%s_writeops_hourly", *disk.ID),
					Name:     fmt.Sprintf("%s writeops-hourly", *disk.Name),
					Location: *disk.Location,
					Description: model.ComputeDiskWriteOpsHourlyDescription{
						MonitoringMetric: metric,
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

func ComputeResourceSKU(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := skus.NewResourceSkusClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, resourceSku := range result.Values() {
			resource := Resource{
				ID:       "azure:///subscriptions/" + subscription + "/locations/" + (*resourceSku.Locations)[0] + "/resourcetypes" + *resourceSku.ResourceType + "name/" + *resourceSku.Name,
				Name:     *resourceSku.Name,
				Location: (*resourceSku.Locations)[0],
				Description: model.ComputeResourceSKUDescription{
					ResourceSKU: resourceSku,
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

func ComputeVirtualMachineCpuUtilization(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewVirtualMachinesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListAll(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, virtualMachine := range result.Values() {
			if virtualMachine.ID == nil {
				continue
			}

			metrics, err := listAzureMonitorMetricStatistics(ctx, authorizer, subscription, "FIVE_MINUTES", "Microsoft.Compute/virtualMachines", "Percentage CPU", *virtualMachine.ID)
			if err != nil {
				return nil, err
			}

			for _, metric := range metrics {
				resource := Resource{
					ID:       fmt.Sprintf("%s_cpu_utilization", *virtualMachine.ID),
					Name:     fmt.Sprintf("%s cpu-utilization", *virtualMachine.Name),
					Location: *virtualMachine.Location,
					Description: model.ComputeVirtualMachineCpuUtilizationDescription{
						MonitoringMetric: metric,
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

func ComputeVirtualMachineCpuUtilizationDaily(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewVirtualMachinesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListAll(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, virtualMachine := range result.Values() {
			if virtualMachine.ID == nil {
				continue
			}

			metrics, err := listAzureMonitorMetricStatistics(ctx, authorizer, subscription, "DAILY", "Microsoft.Compute/virtualMachines", "Percentage CPU", *virtualMachine.ID)
			if err != nil {
				return nil, err
			}

			for _, metric := range metrics {
				resource := Resource{
					ID:       fmt.Sprintf("%s_cpu_utilization_daily", *virtualMachine.ID),
					Name:     fmt.Sprintf("%s cpu-utilization-daily", *virtualMachine.Name),
					Location: *virtualMachine.Location,
					Description: model.ComputeVirtualMachineCpuUtilizationDailyDescription{
						MonitoringMetric: metric,
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

func ComputeVirtualMachineCpuUtilizationHourly(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	client := compute.NewVirtualMachinesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.ListAll(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, virtualMachine := range result.Values() {
			if virtualMachine.ID == nil {
				continue
			}

			metrics, err := listAzureMonitorMetricStatistics(ctx, authorizer, subscription, "HOURLY", "Microsoft.Compute/virtualMachines", "Percentage CPU", *virtualMachine.ID)
			if err != nil {
				return nil, err
			}

			for _, metric := range metrics {
				resource := Resource{
					ID:       fmt.Sprintf("%s_cpu_utilization_hourly", *virtualMachine.ID),
					Name:     fmt.Sprintf("%s cpu-utilization-hourly", *virtualMachine.Name),
					Location: *virtualMachine.Location,
					Description: model.ComputeVirtualMachineCpuUtilizationHourlyDescription{
						MonitoringMetric: metric,
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
