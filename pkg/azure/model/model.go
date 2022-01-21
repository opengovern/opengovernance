//go:generate go run ../../keibi-es-sdk/gen/main.go --file $GOFILE --output ../../keibi-es-sdk/azure_resources_clients.go --type azure

package model

import (
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/healthcareapis/mgmt/healthcareapis"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/links"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/policy"
	"github.com/Azure/azure-sdk-for-go/services/apimanagement/mgmt/2020-12-01/apimanagement"
	"github.com/Azure/azure-sdk-for-go/services/appconfiguration/mgmt/2020-06-01/appconfiguration"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-02-01/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/databoxedge/mgmt/2019-07-01/databoxedge"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/azure-sdk-for-go/services/preview/security/mgmt/v1.0/security"
	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2020-06-01/redis"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-06-01/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/azure-sdk-for-go/services/storagecache/mgmt/2021-05-01/storagecache"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2020-06-01/web"
)

type Metadata struct {
	SubscriptionID   string `json:"subscription_id"`
	CloudEnvironment string `json:"cloud_environment"`
}

//  ===================  APIManagement ==================

//index:microsoft_apimanagement_service
//getfilter:name=description.APIManagement.Name
//getfilter:resource_group=
type APIManagementDescription struct {
	APIManagement               apimanagement.ServiceResource
	DiagnosticSettingsResources []insights.DiagnosticSettingsResource
}

//  ===================  App Configuration ==================

//index:microsoft_appconfiguration_configurationstores
//getfilter:name=description.ConfigurationStore.Name
//getfilter:resource_group=
type AppConfigurationDescription struct {
	ConfigurationStore          appconfiguration.ConfigurationStore
	DiagnosticSettingsResources []insights.DiagnosticSettingsResource
}

//  =================== web ==================

//index:microsoft_web_hostingenvironments
//getfilter:name=
//getfilter:resource_group=
type AppServiceEnvironmentDescription struct {
	AppServiceEnvironmentResource web.AppServiceEnvironmentResource
}

//index:
//getfilter:name=
//getfilter:resource_group=
type AppServiceFunctionAppDescription struct {
	Site               web.Site
	SiteAuthSettings   web.SiteAuthSettings
	SiteConfigResource web.SiteConfigResource
}

//index:
//getfilter:name=
//getfilter:resource_group=
type AppServiceWebAppDescription struct {
	Site               web.Site
	SiteAuthSettings   web.SiteAuthSettings
	SiteConfigResource web.SiteConfigResource
	VnetInfo           web.VnetInfo
}

//  =================== compute ==================

//index:microsoft_compute_disks
//getfilter:name=
//getfilter:resource_group=
type ComputeDiskDescription struct {
	Disk compute.Disk
}

//index:microsoft_compute_diskaccesses
//getfilter:name=
//getfilter:resource_group=
type ComputeDiskAccessDescription struct {
	DiskAccess compute.DiskAccess
}

//index:microsoft_compute_virtualmachinescalesets
//getfilter:name=
//getfilter:resource_group=
type ComputeVirtualMachineScaleSetDescription struct {
	VirtualMachineScaleSet           compute.VirtualMachineScaleSet
	VirtualMachineScaleSetExtensions []compute.VirtualMachineScaleSetExtension
}

//  =================== databoxedge ==================

//index:microsoft_databoxedge_databoxedgedevices
//getfilter:name=
//getfilter:resource_group=
type DataboxEdgeDeviceDescription struct {
	Device databoxedge.Device
}

//  =================== healthcareapis ==================

//index:microsoft_healthcareapis_services
//getfilter:name=
//getfilter:resource_group=
type HealthcareServiceDescription struct {
	ServicesDescription         healthcareapis.ServicesDescription
	DiagnosticSettingsResources *[]insights.DiagnosticSettingsResource
	PrivateEndpointConnections  *[]healthcareapis.PrivateEndpointConnection
}

//  =================== storagecache ==================

//index:microsoft_storagecache_caches
//getfilter:name=
//getfilter:resource_group=
type HpcCacheDescription struct {
	Cache storagecache.Cache
}

//  =================== keyvault ==================

//index:microsoft_keyvault_vaults
//getfilter:vault_name=
//getfilter:name=
//getfilter:resource_group=
type KeyVaultKeyDescription struct {
	Key keyvault.Key
}

//  =================== containerservice ==================

//index:microsoft_containerservice_managedclusters
//getfilter:name=
//getfilter:resource_group=
type KubernetesClusterDescription struct {
	ManagedCluster containerservice.ManagedCluster
}

//  =================== network ==================

//index:microsoft_network_networkinterfaces
//getfilter:name=
//getfilter:resource_group=
type NetworkInterfaceDescription struct {
	Interface network.Interface
}

//index:microsoft_network_networkwatchers
//getfilter:network_watcher_name=
//getfilter:name=
//getfilter:resource_group=
type NetworkWatcherFlowLogDescription struct {
	FlowLog network.FlowLog
}

//  =================== policy ==================

//index:microsoft_authorization_policyassignments
//getfilter:name=
type PolicyAssignmentDescription struct {
	Assignment policy.Assignment
}

//  =================== redis ==================

//index:microsoft_cache_redis
//getfilter:name=
//getfilter:resource_group=
type RedisCacheDescription struct {
	ResourceType redis.ResourceType
}

//  =================== links ==================

//index:
//getfilter:id=
type ResourceLinkDescription struct {
	ResourceLink links.ResourceLink
}

//  =================== authorization ==================

//index:microsoft_authorization_elevateaccessroleassignment
//getfilter:id=
type RoleAssignmentDescription struct {
	RoleAssignment authorization.RoleAssignment
}

//index:
//getfilter:name=
type RoleDefinitionDescription struct {
	RoleDefinition authorization.RoleDefinition
}

//  =================== security ==================

//index:
//getfilter:name=
type SecurityCenterAutoProvisioningDescription struct {
	AutoProvisioningSetting security.AutoProvisioningSetting
}

//index:
//getfilter:name=
type SecurityCenterContactDescription struct {
	Contact security.Contact
}

//index:
type SecurityCenterJitNetworkAccessPolicyDescription struct {
	JitNetworkAccessPolicy security.JitNetworkAccessPolicy
}

//index:
//getfilter:name=
type SecurityCenterSettingDescription struct {
	Setting security.Setting
}

//index:microsoft_security_pricings
//getfilter:name=
type SecurityCenterSubscriptionPricingDescription struct {
	Pricing security.Pricing
}

//  =================== storage ==================

//index:
//getfilter:name=
//getfilter:resource_group=
//getfilter:account_name=
type StorageContainerDescription struct {
	ListContainerItem  storage.ListContainerItem
	ImmutabilityPolicy storage.ImmutabilityPolicy
}

//  =================== network ==================

//index:
//getfilter:name=
//getfilter:resource_group=
//getfilter:virtual_network_name=
type SubnetDescription struct {
	Subnet network.Subnet
}

//index:microsoft_network_virtualnetworks
//getfilter:name=
//getfilter:resource_group=
type VirtualNetworkDescription struct {
	VirtualNetwork network.VirtualNetwork
}

//  =================== subscriptions ==================

//index:
type TenantDescription struct {
	TenantIDDescription subscriptions.TenantIDDescription
}
