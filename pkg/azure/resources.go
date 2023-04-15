package azure

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resourcegraph/mgmt/resourcegraph"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	hamiltonAuth "github.com/manicminer/hamilton/auth"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/describer"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
)

type ResourceDescriber interface {
	DescribeResources(context.Context, autorest.Authorizer, hamiltonAuth.Authorizer, []string, string, enums.DescribeTriggerType) ([]describer.Resource, error)
}

type ResourceDescribeFunc func(context.Context, autorest.Authorizer, hamiltonAuth.Authorizer, []string, string, enums.DescribeTriggerType) ([]describer.Resource, error)

func (fn ResourceDescribeFunc) DescribeResources(c context.Context, a autorest.Authorizer, ah hamiltonAuth.Authorizer, s []string, t string, triggerType enums.DescribeTriggerType) ([]describer.Resource, error) {
	return fn(c, a, ah, s, t, triggerType)
}

type ResourceType struct {
	Name          string
	ServiceName   string
	ListDescriber ResourceDescriber
	GetDescriber  ResourceDescriber // TODO: Change the type?

	TerraformName        []string
	TerraformServiceName string
}

var resourceTypes = map[string]ResourceType{
	"Microsoft.Compute/virtualMachineScaleSetsVm": {
		Name:                 "Microsoft.Compute/virtualMachineScaleSetsVm",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineScaleSetVm),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.EventGrid/domains/topics": {
		Name:                 "Microsoft.EventGrid/domains/topics",
		ServiceName:          "EventGrid",
		ListDescriber:        DescribeBySubscription(describer.EventGridDomainTopic),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_eventgrid_domain_topic"},
		TerraformServiceName: "eventgrid",
	},
	"Microsoft.Network/networkWatchers": {
		Name:                 "Microsoft.Network/networkWatchers",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkWatcher),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_network_watcher"},
		TerraformServiceName: "network",
	},
	"Microsoft.Resources/resourceGroups": {
		Name:                 "Microsoft.Resources/resourceGroups",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.ResourceGroup),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_resource_group"},
		TerraformServiceName: "resource",
	},
	"Microsoft.Web/staticSites": {
		Name:                 "Microsoft.Web/staticSites",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServiceWebApp),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_linux_web_app", "azurerm_windows_web_app"},
		TerraformServiceName: "appservice",
	},
	"Microsoft.Resources/serviceprincipals": {
		Name:                 "Microsoft.Resources/serviceprincipals",
		ServiceName:          "Resources",
		ListDescriber:        DescribeADByTenantID(describer.AdServicePrinciple),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.CognitiveServices/accounts": {
		Name:                 "Microsoft.CognitiveServices/accounts",
		ServiceName:          "CognitiveServices",
		ListDescriber:        DescribeBySubscription(describer.CognitiveAccount),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_cognitive_account"},
		TerraformServiceName: "cognitive",
	},
	"Microsoft.Sql/managedInstances": {
		Name:                 "Microsoft.Sql/managedInstances",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.MssqlManagedInstance),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_mssql_managed_instance"},
		TerraformServiceName: "mssql",
	},
	"Microsoft.Sql/servers/databases": {
		Name:                 "Microsoft.Sql/servers/databases",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlDatabase),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_sql_database"},
		TerraformServiceName: "sql",
	},
	"Microsoft.Storage/fileShares": {
		Name:                 "Microsoft.Storage/fileShares",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageFileShare),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_share_file"},
		TerraformServiceName: "storage",
	},
	"Microsoft.DBforPostgreSQL/servers": {
		Name:                 "Microsoft.DBforPostgreSQL/servers",
		ServiceName:          "DBforPostgreSQL",
		ListDescriber:        DescribeBySubscription(describer.PostgresqlServer),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_postgresql_server"},
		TerraformServiceName: "postgres",
	},
	"Microsoft.Security/pricings": {
		Name:                 "Microsoft.Security/pricings",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterSubscriptionPricing),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_security_center_subscription_pricing"},
		TerraformServiceName: "securitycenter",
	},
	"Microsoft.Insights/guestDiagnosticSettings": {
		Name:                 "Microsoft.Insights/guestDiagnosticSettings",
		ServiceName:          "Insights",
		ListDescriber:        DescribeBySubscription(describer.DiagnosticSetting),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Resources/groups": {
		Name:                 "Microsoft.Resources/groups",
		ServiceName:          "Resources",
		ListDescriber:        DescribeADByTenantID(describer.AdGroup),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Web/hostingEnvironments": {
		Name:                 "Microsoft.Web/hostingEnvironments",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServiceEnvironment),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_app_service_environment"},
		TerraformServiceName: "web",
	},
	"Microsoft.Cache/redis": {
		Name:                 "Microsoft.Cache/redis",
		ServiceName:          "Cache",
		ListDescriber:        DescribeBySubscription(describer.RedisCache),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_redis_cache"},
		TerraformServiceName: "redis",
	},
	"Microsoft.ContainerRegistry/registries": {
		Name:                 "Microsoft.ContainerRegistry/registries",
		ServiceName:          "ContainerRegistry",
		ListDescriber:        DescribeBySubscription(describer.ContainerRegistry),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_container_registry"},
		TerraformServiceName: "containers",
	},
	"Microsoft.DataFactory/factoriesPipelines": {
		Name:                 "Microsoft.DataFactory/factoriesPipelines",
		ServiceName:          "DataFactory",
		ListDescriber:        DescribeBySubscription(describer.DataFactoryPipeline),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_data_factory_pipeline"},
		TerraformServiceName: "datafactory",
	},
	"Microsoft.Compute/resourceSku": {
		Name:                 "Microsoft.Compute/resourceSku",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeResourceSKU),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/expressRouteCircuits": {
		Name:                 "Microsoft.Network/expressRouteCircuits",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.ExpressRouteCircuit),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_express_route_circuit"},
		TerraformServiceName: "network",
	},
	"Microsoft.Management/groups": {
		Name:                 "Microsoft.Management/groups",
		ServiceName:          "Management",
		ListDescriber:        DescribeBySubscription(describer.ManagementGroup),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_management_group"},
		TerraformServiceName: "managementgroup",
	},
	"Microsoft.Sql/virtualMachines": {
		Name:                 "Microsoft.Sql/virtualMachines",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServerVirtualMachine),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Storage/tableServices": {
		Name:                 "Microsoft.Storage/tableServices",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageTableService),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Synapse/workspaces": {
		Name:                 "Microsoft.Synapse/workspaces",
		ServiceName:          "Synapse",
		ListDescriber:        DescribeBySubscription(describer.SynapseWorkspace),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_synapse_workspace"},
		TerraformServiceName: "synapse",
	},
	"Microsoft.StreamAnalytics/streamingJobs": {
		Name:                 "Microsoft.StreamAnalytics/streamingJobs",
		ServiceName:          "StreamAnalytics",
		ListDescriber:        DescribeBySubscription(describer.StreamAnalyticsJob),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_stream_analytics_job"},
		TerraformServiceName: "streamanalytics",
	},
	"Microsoft.CostManagement/CostBySubscription": {
		Name:                 "Microsoft.CostManagement/CostBySubscription",
		ServiceName:          "CostManagement",
		ListDescriber:        DescribeBySubscription(describer.DailyCostBySubscription),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.ContainerService/managedClusters": {
		Name:                 "Microsoft.ContainerService/managedClusters",
		ServiceName:          "ContainerService",
		ListDescriber:        DescribeBySubscription(describer.KubernetesCluster),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_kubernetes_cluster"},
		TerraformServiceName: "containers",
	},
	"Microsoft.DataFactory/factories": {
		Name:                 "Microsoft.DataFactory/factories",
		ServiceName:          "DataFactory",
		ListDescriber:        DescribeBySubscription(describer.DataFactory),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_data_factory"},
		TerraformServiceName: "datafactory",
	},
	"Microsoft.Sql/servers": {
		Name:                 "Microsoft.Sql/servers",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServer),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_sql_server"},
		TerraformServiceName: "sql",
	},
	"Microsoft.Security/autoProvisioningSettings": {
		Name:                 "Microsoft.Security/autoProvisioningSettings",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterAutoProvisioning),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_security_center_auto_provisioning"},
		TerraformServiceName: "securitycenter",
	},
	"Microsoft.Insights/logProfiles": {
		Name:                 "Microsoft.Insights/logProfiles",
		ServiceName:          "Insights",
		ListDescriber:        DescribeBySubscription(describer.LogProfile),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.DataBoxEdge/dataBoxEdgeDevices": {
		Name:                 "Microsoft.DataBoxEdge/dataBoxEdgeDevices",
		ServiceName:          "DataBoxEdge",
		ListDescriber:        DescribeBySubscription(describer.DataboxEdgeDevice),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_databox_edge_device"},
		TerraformServiceName: "databoxedge",
	},
	"Microsoft.Network/loadBalancers": {
		Name:                 "Microsoft.Network/loadBalancers",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancer),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.Network/azureFirewalls": {
		Name:                 "Microsoft.Network/azureFirewalls",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkAzureFirewall),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_firewall"},
		TerraformServiceName: "firewall",
	},
	"Microsoft.Management/locks": {
		Name:                 "Microsoft.Management/locks",
		ServiceName:          "Management",
		ListDescriber:        DescribeBySubscription(describer.ManagementLock),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineScaleSetNetworkInterface": {
		Name:                 "Microsoft.Compute/virtualMachineScaleSetNetworkInterface",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineScaleSetNetworkInterface),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/frontDoors": {
		Name:                 "Microsoft.Network/frontDoors",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.FrontDoor),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_frontdoor"},
		TerraformServiceName: "frontdoor",
	},
	"Microsoft.Resources/subscriptions/resourceGroups": {
		Name:                 "Microsoft.Resources/subscriptions/resourceGroups",
		ServiceName:          "Resources",
		ListDescriber:        describer.GenericResourceGraph{Table: "ResourceContainers", Type: "Microsoft.Resources/subscriptions/resourceGroups"},
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Authorization/policyAssignments": {
		Name:                 "Microsoft.Authorization/policyAssignments",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.PolicyAssignment),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Search/searchServices": {
		Name:                 "Microsoft.Search/searchServices",
		ServiceName:          "Search",
		ListDescriber:        DescribeBySubscription(describer.SearchService),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_search_service"},
		TerraformServiceName: "search",
	},
	"Microsoft.Security/settings": {
		Name:                 "Microsoft.Security/settings",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterSetting),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_security_center_setting"},
		TerraformServiceName: "securitycenter",
	},
	"Microsoft.RecoveryServices/vaults": {
		Name:                 "Microsoft.RecoveryServices/vaults",
		ServiceName:          "RecoveryServices",
		ListDescriber:        DescribeBySubscription(describer.RecoveryServicesVault),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_recovery_services_vault"},
		TerraformServiceName: "recoveryservices",
	},
	"Microsoft.Compute/diskEncryptionSets": {
		Name:                 "Microsoft.Compute/diskEncryptionSets",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskEncryptionSet),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_disk_encryption_set"},
		TerraformServiceName: "compute",
	},
	"Microsoft.DocumentDB/SqlDatabases": {
		Name:                 "Microsoft.DocumentDB/SqlDatabases",
		ServiceName:          "DocumentDB",
		ListDescriber:        DescribeBySubscription(describer.DocumentDBSQLDatabase),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_cosmosdb_sql_database"},
		TerraformServiceName: "cosmos",
	},
	"Microsoft.EventGrid/topics": {
		Name:                 "Microsoft.EventGrid/topics",
		ServiceName:          "EventGrid",
		ListDescriber:        DescribeBySubscription(describer.EventGridTopic),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_eventgrid_topic"},
		TerraformServiceName: "eventgrid",
	},
	"Microsoft.EventHub/namespaces": {
		Name:                 "Microsoft.EventHub/namespaces",
		ServiceName:          "EventHub",
		ListDescriber:        DescribeBySubscription(describer.EventhubNamespace),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_eventhub_namespace"},
		TerraformServiceName: "eventhub",
	},
	"Microsoft.MachineLearningServices/workspaces": {
		Name:                 "Microsoft.MachineLearningServices/workspaces",
		ServiceName:          "MachineLearningServices",
		ListDescriber:        DescribeBySubscription(describer.MachineLearningWorkspace),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_machine_learning_workspace"},
		TerraformServiceName: "machinelearning",
	},
	"Microsoft.CostManagement/CostByResourceType": {
		Name:                 "Microsoft.CostManagement/CostByResourceType",
		ServiceName:          "CostManagement",
		ListDescriber:        DescribeBySubscription(describer.DailyCostByResourceType),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/networkInterfaces": {
		Name:                 "Microsoft.Network/networkInterfaces",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkInterface),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_network_interface"},
		TerraformServiceName: "network",
	},
	"Microsoft.Network/publicIPAddresses": {
		Name:                 "Microsoft.Network/publicIPAddresses",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.PublicIPAddress),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_public_ip"},
		TerraformServiceName: "network",
	},
	"Microsoft.HealthcareApis/services": {
		Name:                 "Microsoft.HealthcareApis/services",
		ServiceName:          "HealthcareApis",
		ListDescriber:        DescribeBySubscription(describer.HealthcareService),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_healthcare_service"},
		TerraformServiceName: "healthcare",
	},
	"Microsoft.ServiceBus/namespaces": {
		Name:                 "Microsoft.ServiceBus/namespaces",
		ServiceName:          "ServiceBus",
		ListDescriber:        DescribeBySubscription(describer.ServicebusNamespace),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_servicebus_namespace"},
		TerraformServiceName: "servicebus",
	},
	"Microsoft.Web/sites": {
		Name:                 "Microsoft.Web/sites",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServiceFunctionApp),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_function_app"},
		TerraformServiceName: "web",
	},
	"Microsoft.Compute/availabilitySets": {
		Name:                 "Microsoft.Compute/availabilitySets",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeAvailabilitySet),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_availability_set"},
		TerraformServiceName: "compute",
	},
	"Microsoft.Network/virtualNetworks": {
		Name:                 "Microsoft.Network/virtualNetworks",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.VirtualNetwork),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_virtual_network"},
		TerraformServiceName: "network",
	},
	"Microsoft.Security/securityContacts": {
		Name:                 "Microsoft.Security/securityContacts",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterContact),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_security_center_contact"},
		TerraformServiceName: "securitycenter",
	},
	"Microsoft.Compute/diskswriteops": {
		Name:                 "Microsoft.Compute/diskswriteops",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskWriteOps),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/diskswriteopshourly": {
		Name:                 "Microsoft.Compute/diskswriteopshourly",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskWriteOpsHourly),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.EventGrid/domains": {
		Name:                 "Microsoft.EventGrid/domains",
		ServiceName:          "EventGrid",
		ListDescriber:        DescribeBySubscription(describer.EventGridDomain),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_eventgrid_domain"},
		TerraformServiceName: "eventgrid",
	},
	"Microsoft.KeyVault/deletedVaults": {
		Name:                 "Microsoft.KeyVault/deletedVaults",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.DeletedVault),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Storage/tables": {
		Name:                 "Microsoft.Storage/tables",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageTable),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_table"},
		TerraformServiceName: "storage",
	},
	"Microsoft.Resources/users": {
		Name:                 "Microsoft.Resources/users",
		ServiceName:          "Resources",
		ListDescriber:        DescribeADByTenantID(describer.AdUsers),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/snapshots": {
		Name:                 "Microsoft.Compute/snapshots",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeSnapshots),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_snapshot"},
		TerraformServiceName: "compute",
	},
	"Microsoft.Kusto/clusters": {
		Name:                 "Microsoft.Kusto/clusters",
		ServiceName:          "Kusto",
		ListDescriber:        DescribeBySubscription(describer.KustoCluster),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_kusto_cluster"},
		TerraformServiceName: "kusto",
	},
	"Microsoft.StorageSync/storageSyncServices": {
		Name:                 "Microsoft.StorageSync/storageSyncServices",
		ServiceName:          "StorageSync",
		ListDescriber:        DescribeBySubscription(describer.StorageSync),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_sync"},
		TerraformServiceName: "storage",
	},
	"Microsoft.Security/locations/jitNetworkAccessPolicies": {
		Name:                 "Microsoft.Security/locations/jitNetworkAccessPolicies",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterJitNetworkAccessPolicy),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/virtualNetworks/subnets": {
		Name:                 "Microsoft.Network/virtualNetworks/subnets",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.Subnet),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_subnet"},
		TerraformServiceName: "network",
	},
	"Microsoft.LoadBalancer/backendAddressPools": {
		Name:                 "Microsoft.LoadBalancer/backendAddressPools",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerBackendAddressPool),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb_backend_address_pool"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.LoadBalancer/rules": {
		Name:                 "Microsoft.LoadBalancer/rules",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerRule),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb_rule"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.Compute/virtualMachineCpuUtilizationDaily": {
		Name:                 "Microsoft.Compute/virtualMachineCpuUtilizationDaily",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineCpuUtilizationDaily),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.DataLakeStore/accounts": {
		Name:                 "Microsoft.DataLakeStore/accounts",
		ServiceName:          "DataLakeStore",
		ListDescriber:        DescribeBySubscription(describer.DataLakeStore),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.StorageCache/caches": {
		Name:                 "Microsoft.StorageCache/caches",
		ServiceName:          "StorageCache",
		ListDescriber:        DescribeBySubscription(describer.HpcCache),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_hpc_cache"},
		TerraformServiceName: "hpccache",
	},
	"Microsoft.Batch/batchAccounts": {
		Name:                 "Microsoft.Batch/batchAccounts",
		ServiceName:          "Batch",
		ListDescriber:        DescribeBySubscription(describer.BatchAccount),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_batch_account"},
		TerraformServiceName: "batch",
	},
	"Microsoft.ClassicNetwork/networkSecurityGroups": {
		Name:                 "Microsoft.ClassicNetwork/networkSecurityGroups",
		ServiceName:          "ClassicNetwork",
		ListDescriber:        DescribeBySubscription(describer.NetworkSecurityGroup),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_network_security_group"},
		TerraformServiceName: "network",
	},
	"Microsoft.Authorization/roleDefinitions": {
		Name:                 "Microsoft.Authorization/roleDefinitions",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.RoleDefinition),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_role_definition"},
		TerraformServiceName: "authorization",
	},
	"Microsoft.Network/applicationSecurityGroups": {
		Name:                 "Microsoft.Network/applicationSecurityGroups",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkApplicationSecurityGroups),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_application_security_group"},
		TerraformServiceName: "network",
	},
	"Microsoft.Authorization/elevateAccessRoleAssignment": {
		Name:                 "Microsoft.Authorization/elevateAccessRoleAssignment",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.RoleAssignment),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.DocumentDB/MongoDatabases": {
		Name:                 "Microsoft.DocumentDB/MongoDatabases",
		ServiceName:          "DocumentDB",
		ListDescriber:        DescribeBySubscription(describer.DocumentDBMongoDatabase),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_cosmosdb_mongo_database"},
		TerraformServiceName: "cosmos",
	},
	"Microsoft.Network/networkWatchers/flowLogs": {
		Name:                 "Microsoft.Network/networkWatchers/flowLogs",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkWatcherFlowLog),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_network_watcher_flow_log"},
		TerraformServiceName: "network",
	},
	"Microsoft.Sql/elasticPools": {
		Name:                 "Microsoft.Sql/elasticPools",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServerElasticPool),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_sql_elasticpool"},
		TerraformServiceName: "sql",
	},
	"Microsoft.Security/subAssessments": {
		Name:                 "Microsoft.Security/subAssessments",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterSubAssessment),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/disks": {
		Name:                 "Microsoft.Compute/disks",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDisk),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_managed_disk"},
		TerraformServiceName: "",
	},
	"Microsoft.Devices/iotHubDpses": {
		Name:                 "Microsoft.Devices/iotHubDpses",
		ServiceName:          "Devices",
		ListDescriber:        DescribeBySubscription(describer.IOTHubDps),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.HDInsight/clusters": {
		Name:                 "Microsoft.HDInsight/clusters",
		ServiceName:          "HDInsight",
		ListDescriber:        DescribeBySubscription(describer.HdInsightCluster),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_hdinsight_cluster"},
		TerraformServiceName: "hdinsight",
	},
	"Microsoft.ServiceFabric/clusters": {
		Name:                 "Microsoft.ServiceFabric/clusters",
		ServiceName:          "ServiceFabric",
		ListDescriber:        DescribeBySubscription(describer.ServiceFabricCluster),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_service_fabric_cluster"},
		TerraformServiceName: "servicefabric",
	},
	"Microsoft.SignalRService/signalR": {
		Name:                 "Microsoft.SignalRService/signalR",
		ServiceName:          "SignalRService",
		ListDescriber:        DescribeBySubscription(describer.SignalrService),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_signalr_service"},
		TerraformServiceName: "signalr",
	},
	"Microsoft.Storage/blobs": {
		Name:                 "Microsoft.Storage/blobs",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageBlob),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_blob"},
		TerraformServiceName: "storage",
	},
	"Microsoft.Storage/blobServives": {
		Name:                 "Microsoft.Storage/blobServives",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageBlobService),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Storage/queues": {
		Name:                 "Microsoft.Storage/queues",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageQueue),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_queue"},
		TerraformServiceName: "storage",
	},
	"Microsoft.ApiManagement/service": {
		Name:                 "Microsoft.ApiManagement/service",
		ServiceName:          "ApiManagement",
		ListDescriber:        DescribeBySubscription(describer.APIManagement),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_api_management"},
		TerraformServiceName: "apimanagement",
	},
	"Microsoft.Compute/disksreadops": {
		Name:                 "Microsoft.Compute/disksreadops",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskReadOps),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineScaleSets": {
		Name:                 "Microsoft.Compute/virtualMachineScaleSets",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineScaleSet),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_windows_virtual_machine_scale_set", "azurerm_linux_virtual_machine_scale_set", "azurerm_orchestrated_virtual_machine_scale_set"},
		TerraformServiceName: "compute",
	},
	"Microsoft.DataFactory/factoriesDatasets": {
		Name:                 "Microsoft.DataFactory/factoriesDatasets",
		ServiceName:          "DataFactory",
		ListDescriber:        DescribeBySubscription(describer.DataFactoryDataset),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_data_factory_dataset_azure_blob", "azurerm_data_factory_dataset_binary", "azurerm_data_factory_dataset_cosmosdb_sqlapi", "azurerm_data_factory_dataset_delimited_text", "azurerm_data_factory_dataset_http", "azurerm_data_factory_dataset_json", "azurerm_data_factory_dataset_mysql", "azurerm_data_factory_dataset_parquet", "azurerm_data_factory_dataset_postgresql", "azurerm_data_factory_dataset_snowflake", "azurerm_data_factory_dataset_sql_server_table", "azurerm_data_factory_custom_dataset"},
		TerraformServiceName: "datafactory",
	},
	"Microsoft.Authorization/policyDefinitions": {
		Name:                 "Microsoft.Authorization/policyDefinitions",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.PolicyDefinition),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Resources/subscriptions/locations": {
		Name:                 "Microsoft.Resources/subscriptions/locations",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.Location),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/diskAccesses": {
		Name:                 "Microsoft.Compute/diskAccesses",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskAccess),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_disk_access"},
		TerraformServiceName: "compute",
	},
	"Microsoft.DBforMySQL/servers": {
		Name:                 "Microsoft.DBforMySQL/servers",
		ServiceName:          "DBforMySQL",
		ListDescriber:        DescribeBySubscription(describer.MysqlServer),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_mysql_server"},
		TerraformServiceName: "mysql",
	},
	"Microsoft.DataLakeAnalytics/accounts": {
		Name:                 "Microsoft.DataLakeAnalytics/accounts",
		ServiceName:          "DataLakeAnalytics",
		ListDescriber:        DescribeBySubscription(describer.DataLakeAnalyticsAccount),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Insights/activityLogAlerts": {
		Name:                 "Microsoft.Insights/activityLogAlerts",
		ServiceName:          "Insights",
		ListDescriber:        DescribeBySubscription(describer.LogAlert),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineCpuUtilizationHourly": {
		Name:                 "Microsoft.Compute/virtualMachineCpuUtilizationHourly",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineCpuUtilizationHourly),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.LoadBalancer/outboundRules": {
		Name:                 "Microsoft.LoadBalancer/outboundRules",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerOutboundRule),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb_outbound_rule"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.HybridCompute/machines": {
		Name:                 "Microsoft.HybridCompute/machines",
		ServiceName:          "HybridCompute",
		ListDescriber:        DescribeBySubscription(describer.HybridComputeMachine),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_hybrid_compute_machine"},
		TerraformServiceName: "hybridcompute",
	},
	"Microsoft.LoadBalancer/natRules": {
		Name:                 "Microsoft.LoadBalancer/natRules",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerNatRule),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb_nat_rule"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.Resources/providers": {
		Name:                 "Microsoft.Resources/providers",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.ResourceProvider),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/routeTables": {
		Name:                 "Microsoft.Network/routeTables",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.RouteTables),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_route_table"},
		TerraformServiceName: "network",
	},
	"Microsoft.DocumentDB/databaseAccounts": {
		Name:                 "Microsoft.DocumentDB/databaseAccounts",
		ServiceName:          "DocumentDB",
		ListDescriber:        DescribeBySubscription(describer.CosmosdbAccount),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_cosmosdb_account"},
		TerraformServiceName: "cosmos",
	},
	"Microsoft.Network/applicationGateways": {
		Name:                 "Microsoft.Network/applicationGateways",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.ApplicationGateway),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_application_gateway"},
		TerraformServiceName: "network",
	},
	"Microsoft.Security/automations": {
		Name:                 "Microsoft.Security/automations",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterAutomation),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_security_center_automation"},
		TerraformServiceName: "securitycenter",
	},
	"Microsoft.Kubernetes/connectedClusters": {
		Name:                 "Microsoft.Kubernetes/connectedClusters",
		ServiceName:          "Kubernetes",
		ListDescriber:        DescribeBySubscription(describer.HybridKubernetesConnectedCluster),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.KeyVault/vaults/keys": {
		Name:                 "Microsoft.KeyVault/vaults/keys",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVaultKey),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.DBforMariaDB/servers": {
		Name:                 "Microsoft.DBforMariaDB/servers",
		ServiceName:          "DBforMariaDB",
		ListDescriber:        DescribeBySubscription(describer.MariadbServer),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_mariadb_server"},
		TerraformServiceName: "mariadb",
	},
	"Microsoft.Compute/disksreadopsdaily": {
		Name:                 "Microsoft.Compute/disksreadopsdaily",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskReadOpsDaily),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Web/plan": {
		Name:                 "Microsoft.Web/plan",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServicePlan),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_app_service_plan"},
		TerraformServiceName: "web",
	},
	"Microsoft.Compute/disksreadopshourly": {
		Name:                 "Microsoft.Compute/disksreadopshourly",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskReadOpsHourly),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/diskswriteopsdaily": {
		Name:                 "Microsoft.Compute/diskswriteopsdaily",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskWriteOpsDaily),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Resources/tenants": {
		Name:                 "Microsoft.Resources/tenants",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.Tenant),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/virtualNetworkGateways": {
		Name:                 "Microsoft.Network/virtualNetworkGateways",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.VirtualNetworkGateway),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_virtual_network_gateway"},
		TerraformServiceName: "network",
	},
	"Microsoft.Devices/iotHubs": {
		Name:                 "Microsoft.Devices/iotHubs",
		ServiceName:          "Devices",
		ListDescriber:        DescribeBySubscription(describer.IOTHub),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_iothub"},
		TerraformServiceName: "iothub",
	},
	"Microsoft.Logic/workflows": {
		Name:                 "Microsoft.Logic/workflows",
		ServiceName:          "Logic",
		ListDescriber:        DescribeBySubscription(describer.LogicAppWorkflow),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_logic_app_workflow"},
		TerraformServiceName: "logic",
	},
	"Microsoft.Sql/flexibleServers": {
		Name:                 "Microsoft.Sql/flexibleServers",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServerFlexibleServer),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Resources/links": {
		Name:                 "Microsoft.Resources/links",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.ResourceLink),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Resources/subscriptions": {
		Name:                 "Microsoft.Resources/subscriptions",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.Subscription),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_subscription"},
		TerraformServiceName: "subscription",
	},
	"Microsoft.Compute/image": {
		Name:                 "Microsoft.Compute/image",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeImage),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_image"},
		TerraformServiceName: "compute",
	},
	"Microsoft.Compute/virtualMachines": {
		Name:                 "Microsoft.Compute/virtualMachines",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachine),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine"},
		TerraformServiceName: "compute",
	},
	"Microsoft.Network/natGateways": {
		Name:                 "Microsoft.Network/natGateways",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NatGateway),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_nat_gateway"},
		TerraformServiceName: "network",
	},
	"Microsoft.LoadBalancer/probes": {
		Name:                 "Microsoft.LoadBalancer/probes",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerProbe),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb_probe"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.KeyVault/vaults": {
		Name:                 "Microsoft.KeyVault/vaults",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVault),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_key_vault"},
		TerraformServiceName: "keyvault",
	},
	"Microsoft.KeyVault/managedHsms": {
		Name:                 "Microsoft.KeyVault/managedHsms",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVaultManagedHardwareSecurityModule),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_key_vault_managed_hardware_security_module"},
		TerraformServiceName: "keyvault",
	},
	"Microsoft.KeyVault/vaults/secrets": {
		Name:                 "Microsoft.KeyVault/vaults/secrets",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVaultSecret),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_key_vault_secret"},
		TerraformServiceName: "keyvault",
	},
	"Microsoft.AppConfiguration/configurationStores": {
		Name:                 "Microsoft.AppConfiguration/configurationStores",
		ServiceName:          "AppConfiguration",
		ListDescriber:        DescribeBySubscription(describer.AppConfiguration),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineCpuUtilization": {
		Name:                 "Microsoft.Compute/virtualMachineCpuUtilization",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineCpuUtilization),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Storage/storageAccounts": {
		Name:                 "Microsoft.Storage/storageAccounts",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageAccount),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_account"},
		TerraformServiceName: "storage",
	},
	"Microsoft.AppPlatform/Spring": {
		Name:                 "Microsoft.AppPlatform/Spring",
		ServiceName:          "AppPlatform",
		ListDescriber:        DescribeBySubscription(describer.SpringCloudService),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_spring_cloud_service"},
		TerraformServiceName: "springcloud",
	},
}

func ListResourceTypes() []string {
	var list []string
	for k := range resourceTypes {
		list = append(list, k)
	}

	sort.Strings(list)
	return list
}

type ResourceDescriptionMetadata struct {
	ResourceType     string
	SubscriptionIds  []string
	CloudEnvironment string
}

type Resources struct {
	Resources []describer.Resource
	Metadata  ResourceDescriptionMetadata
}

func GetResources(
	ctx context.Context,
	resourceType string,
	triggerType enums.DescribeTriggerType,
	subscriptions []string,
	cfg AuthConfig,
	azureAuth string,
	azureAuthLoc string,
) (*Resources, error) {
	// Create and authorize a ResourceGraph client
	var authorizer autorest.Authorizer
	var err error
	switch v := AuthType(strings.ToUpper(azureAuth)); v {
	case AuthEnv:
		authorizer, err = NewAuthorizerFromConfig(cfg)
	case AuthFile:
		setEnvIfNotEmpty(AzureAuthLocation, azureAuthLoc)
		authorizer, err = auth.NewAuthorizerFromFile(resourcegraph.DefaultBaseURI)
	case AuthCLI:
		authorizer, err = auth.NewAuthorizerFromCLI()
	default:
		err = fmt.Errorf("invalid auth type: %s", v)
	}

	if err != nil {
		return nil, err
	}

	hamiltonAuthorizer, err := hamiltonAuth.NewAutorestAuthorizerWrapper(authorizer)
	if err != nil {
		return nil, err
	}

	env, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return nil, err
	}

	resources, err := describe(ctx, authorizer, hamiltonAuthorizer, resourceType, subscriptions, cfg.TenantID, triggerType)
	if err != nil {
		return nil, err
	}

	for i, resource := range resources {
		resources[i].Type = resourceType
		if parts := strings.Split(resources[i].ID, "/"); len(parts) > 4 {
			resources[i].ResourceGroup = strings.Split(resources[i].ID, "/")[4]
		}
		resources[i].Description = describer.JSONAllFieldsMarshaller{
			Value: resource.Description,
		}
	}

	output := &Resources{
		Resources: resources,
		Metadata: ResourceDescriptionMetadata{
			ResourceType:     resourceType,
			SubscriptionIds:  subscriptions,
			CloudEnvironment: env.Environment.Name,
		},
	}

	return output, err
}

func setEnvIfNotEmpty(env, s string) {
	if s != "" {
		err := os.Setenv(env, s)
		if err != nil {
			panic(err)
		}
	}
}

func describe(ctx context.Context, authorizer autorest.Authorizer, hamiltonAuth hamiltonAuth.Authorizer, resourceType string, subscriptions []string, tenantId string, triggerType enums.DescribeTriggerType) ([]describer.Resource, error) {
	resourceTypeObject, ok := resourceTypes[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	listDescriber := resourceTypeObject.ListDescriber
	if listDescriber == nil {
		listDescriber = describer.GenericResourceGraph{Table: "Resources", Type: resourceType}
	}

	return listDescriber.DescribeResources(ctx, authorizer, hamiltonAuth, subscriptions, tenantId, triggerType)
}

func DescribeBySubscription(describe func(context.Context, autorest.Authorizer, string) ([]describer.Resource, error)) ResourceDescriber {
	return ResourceDescribeFunc(func(ctx context.Context, authorizer autorest.Authorizer, hamiltonAuth hamiltonAuth.Authorizer, subscriptions []string, tenantId string, triggerType enums.DescribeTriggerType) ([]describer.Resource, error) {
		ctx = describer.WithTriggerType(ctx, triggerType)
		values := []describer.Resource{}
		for _, subscription := range subscriptions {
			result, err := describe(ctx, authorizer, subscription)
			if err != nil {
				return nil, err
			}

			for _, resource := range result {
				resource.SubscriptionID = subscription
			}
			values = append(values, result...)
		}

		return values, nil
	})
}

func DescribeADByTenantID(describe func(context.Context, hamiltonAuth.Authorizer, string) ([]describer.Resource, error)) ResourceDescriber {
	return ResourceDescribeFunc(func(ctx context.Context, authorizer autorest.Authorizer, hamiltonAuth hamiltonAuth.Authorizer, subscription []string, tenantId string, triggerType enums.DescribeTriggerType) ([]describer.Resource, error) {
		ctx = describer.WithTriggerType(ctx, triggerType)
		var values []describer.Resource
		result, err := describe(ctx, hamiltonAuth, tenantId)
		if err != nil {
			return nil, err
		}

		values = append(values, result...)

		return values, nil
	})
}
