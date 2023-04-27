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
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type ResourceDescriber interface {
	DescribeResources(context.Context, autorest.Authorizer, hamiltonAuth.Authorizer, []string, string, enums.DescribeTriggerType, *describer.StreamSender) ([]describer.Resource, error)
}

type ResourceDescribeFunc func(context.Context, autorest.Authorizer, hamiltonAuth.Authorizer, []string, string, enums.DescribeTriggerType, *describer.StreamSender) ([]describer.Resource, error)

func (fn ResourceDescribeFunc) DescribeResources(c context.Context, a autorest.Authorizer, ah hamiltonAuth.Authorizer, s []string, t string, triggerType enums.DescribeTriggerType, stream *describer.StreamSender) ([]describer.Resource, error) {
	return fn(c, a, ah, s, t, triggerType, stream)
}

type ResourceType struct {
	Connector source.Type

	ResourceName  string
	ResourceLabel string
	ServiceName   string

	ListDescriber ResourceDescriber
	GetDescriber  ResourceDescriber // TODO: Change the type?

	TerraformName        []string
	TerraformServiceName string
}

var resourceTypes = map[string]ResourceType{
	"Microsoft.Compute/virtualMachineScaleSetsVm": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/virtualMachineScaleSetsVm",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineScaleSetVm),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.EventGrid/domains/topics": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.EventGrid/domains/topics",
		ResourceLabel:        "",
		ServiceName:          "EventGrid",
		ListDescriber:        DescribeBySubscription(describer.EventGridDomainTopic),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_eventgrid_domain_topic"},
		TerraformServiceName: "eventgrid",
	},
	"Microsoft.Network/networkWatchers": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/networkWatchers",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkWatcher),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_network_watcher"},
		TerraformServiceName: "network",
	},
	"Microsoft.Resources/resourceGroups": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Resources/resourceGroups",
		ResourceLabel:        "",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.ResourceGroup),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_resource_group"},
		TerraformServiceName: "resource",
	},
	"Microsoft.Web/staticSites": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Web/staticSites",
		ResourceLabel:        "",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServiceWebApp),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_linux_web_app", "azurerm_windows_web_app"},
		TerraformServiceName: "appservice",
	},
	"Microsoft.Resources/serviceprincipals": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Resources/serviceprincipals",
		ResourceLabel:        "",
		ServiceName:          "Resources",
		ListDescriber:        DescribeADByTenantID(describer.AdServicePrinciple),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.CognitiveServices/accounts": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.CognitiveServices/accounts",
		ResourceLabel:        "",
		ServiceName:          "CognitiveServices",
		ListDescriber:        DescribeBySubscription(describer.CognitiveAccount),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_cognitive_account"},
		TerraformServiceName: "cognitive",
	},
	"Microsoft.Sql/managedInstances": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Sql/managedInstances",
		ResourceLabel:        "",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.MssqlManagedInstance),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_mssql_managed_instance"},
		TerraformServiceName: "mssql",
	},
	"Microsoft.Sql/servers/databases": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Sql/servers/databases",
		ResourceLabel:        "",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlDatabase),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_sql_database"},
		TerraformServiceName: "sql",
	},
	"Microsoft.Storage/fileShares": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Storage/fileShares",
		ResourceLabel:        "",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageFileShare),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_share_file"},
		TerraformServiceName: "storage",
	},
	"Microsoft.DBforPostgreSQL/servers": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DBforPostgreSQL/servers",
		ResourceLabel:        "",
		ServiceName:          "DBforPostgreSQL",
		ListDescriber:        DescribeBySubscription(describer.PostgresqlServer),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_postgresql_server"},
		TerraformServiceName: "postgres",
	},
	"Microsoft.Security/pricings": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Security/pricings",
		ResourceLabel:        "",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterSubscriptionPricing),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_security_center_subscription_pricing"},
		TerraformServiceName: "securitycenter",
	},
	"Microsoft.Insights/guestDiagnosticSettings": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Insights/guestDiagnosticSettings",
		ResourceLabel:        "",
		ServiceName:          "Insights",
		ListDescriber:        DescribeBySubscription(describer.DiagnosticSetting),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Resources/groups": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Resources/groups",
		ResourceLabel:        "",
		ServiceName:          "Resources",
		ListDescriber:        DescribeADByTenantID(describer.AdGroup),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Web/hostingEnvironments": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Web/hostingEnvironments",
		ResourceLabel:        "",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServiceEnvironment),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_app_service_environment"},
		TerraformServiceName: "web",
	},
	"Microsoft.Cache/redis": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Cache/redis",
		ResourceLabel:        "",
		ServiceName:          "Cache",
		ListDescriber:        DescribeBySubscription(describer.RedisCache),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_redis_cache"},
		TerraformServiceName: "redis",
	},
	"Microsoft.ContainerRegistry/registries": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.ContainerRegistry/registries",
		ResourceLabel:        "",
		ServiceName:          "ContainerRegistry",
		ListDescriber:        DescribeBySubscription(describer.ContainerRegistry),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_container_registry"},
		TerraformServiceName: "containers",
	},
	"Microsoft.DataFactory/factoriesPipelines": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DataFactory/factoriesPipelines",
		ResourceLabel:        "",
		ServiceName:          "DataFactory",
		ListDescriber:        DescribeBySubscription(describer.DataFactoryPipeline),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_data_factory_pipeline"},
		TerraformServiceName: "datafactory",
	},
	"Microsoft.Compute/resourceSku": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/resourceSku",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeResourceSKU),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/expressRouteCircuits": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/expressRouteCircuits",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.ExpressRouteCircuit),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_express_route_circuit"},
		TerraformServiceName: "network",
	},
	"Microsoft.Management/groups": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Management/groups",
		ResourceLabel:        "",
		ServiceName:          "Management",
		ListDescriber:        DescribeBySubscription(describer.ManagementGroup),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_management_group"},
		TerraformServiceName: "managementgroup",
	},
	"Microsoft.Sql/virtualMachines": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Sql/virtualMachines",
		ResourceLabel:        "",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServerVirtualMachine),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Storage/tableServices": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Storage/tableServices",
		ResourceLabel:        "",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageTableService),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Synapse/workspaces": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Synapse/workspaces",
		ResourceLabel:        "",
		ServiceName:          "Synapse",
		ListDescriber:        DescribeBySubscription(describer.SynapseWorkspace),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_synapse_workspace"},
		TerraformServiceName: "synapse",
	},
	"Microsoft.StreamAnalytics/streamingJobs": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.StreamAnalytics/streamingJobs",
		ResourceLabel:        "",
		ServiceName:          "StreamAnalytics",
		ListDescriber:        DescribeBySubscription(describer.StreamAnalyticsJob),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_stream_analytics_job"},
		TerraformServiceName: "streamanalytics",
	},
	"Microsoft.CostManagement/CostBySubscription": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.CostManagement/CostBySubscription",
		ResourceLabel:        "",
		ServiceName:          "CostManagement",
		ListDescriber:        DescribeBySubscription(describer.DailyCostBySubscription),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.ContainerService/managedClusters": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.ContainerService/managedClusters",
		ResourceLabel:        "",
		ServiceName:          "ContainerService",
		ListDescriber:        DescribeBySubscription(describer.KubernetesCluster),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_kubernetes_cluster"},
		TerraformServiceName: "containers",
	},
	"Microsoft.DataFactory/factories": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DataFactory/factories",
		ResourceLabel:        "",
		ServiceName:          "DataFactory",
		ListDescriber:        DescribeBySubscription(describer.DataFactory),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_data_factory"},
		TerraformServiceName: "datafactory",
	},
	"Microsoft.Sql/servers": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Sql/servers",
		ResourceLabel:        "",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServer),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_sql_server"},
		TerraformServiceName: "sql",
	},
	"Microsoft.Security/autoProvisioningSettings": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Security/autoProvisioningSettings",
		ResourceLabel:        "",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterAutoProvisioning),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_security_center_auto_provisioning"},
		TerraformServiceName: "securitycenter",
	},
	"Microsoft.Insights/logProfiles": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Insights/logProfiles",
		ResourceLabel:        "",
		ServiceName:          "Insights",
		ListDescriber:        DescribeBySubscription(describer.LogProfile),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.DataBoxEdge/dataBoxEdgeDevices": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DataBoxEdge/dataBoxEdgeDevices",
		ResourceLabel:        "",
		ServiceName:          "DataBoxEdge",
		ListDescriber:        DescribeBySubscription(describer.DataboxEdgeDevice),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_databox_edge_device"},
		TerraformServiceName: "databoxedge",
	},
	"Microsoft.Network/loadBalancers": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/loadBalancers",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancer),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.Network/azureFirewalls": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/azureFirewalls",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkAzureFirewall),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_firewall"},
		TerraformServiceName: "firewall",
	},
	"Microsoft.Management/locks": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Management/locks",
		ResourceLabel:        "",
		ServiceName:          "Management",
		ListDescriber:        DescribeBySubscription(describer.ManagementLock),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineScaleSetNetworkInterface": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/virtualMachineScaleSetNetworkInterface",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineScaleSetNetworkInterface),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/frontDoors": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/frontDoors",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.FrontDoor),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_frontdoor"},
		TerraformServiceName: "frontdoor",
	},
	"Microsoft.Resources/subscriptions/resourceGroups": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Resources/subscriptions/resourceGroups",
		ResourceLabel:        "",
		ServiceName:          "Resources",
		ListDescriber:        describer.GenericResourceGraph{Table: "ResourceContainers", Type: "Microsoft.Resources/subscriptions/resourceGroups"},
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Authorization/policyAssignments": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Authorization/policyAssignments",
		ResourceLabel:        "",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.PolicyAssignment),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Search/searchServices": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Search/searchServices",
		ResourceLabel:        "",
		ServiceName:          "Search",
		ListDescriber:        DescribeBySubscription(describer.SearchService),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_search_service"},
		TerraformServiceName: "search",
	},
	"Microsoft.Security/settings": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Security/settings",
		ResourceLabel:        "",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterSetting),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_security_center_setting"},
		TerraformServiceName: "securitycenter",
	},
	"Microsoft.RecoveryServices/vaults": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.RecoveryServices/vaults",
		ResourceLabel:        "",
		ServiceName:          "RecoveryServices",
		ListDescriber:        DescribeBySubscription(describer.RecoveryServicesVault),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_recovery_services_vault"},
		TerraformServiceName: "recoveryservices",
	},
	"Microsoft.Compute/diskEncryptionSets": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/diskEncryptionSets",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskEncryptionSet),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_disk_encryption_set"},
		TerraformServiceName: "compute",
	},
	"Microsoft.DocumentDB/SqlDatabases": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DocumentDB/SqlDatabases",
		ResourceLabel:        "",
		ServiceName:          "DocumentDB",
		ListDescriber:        DescribeBySubscription(describer.DocumentDBSQLDatabase),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_cosmosdb_sql_database"},
		TerraformServiceName: "cosmos",
	},
	"Microsoft.EventGrid/topics": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.EventGrid/topics",
		ResourceLabel:        "",
		ServiceName:          "EventGrid",
		ListDescriber:        DescribeBySubscription(describer.EventGridTopic),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_eventgrid_topic"},
		TerraformServiceName: "eventgrid",
	},
	"Microsoft.EventHub/namespaces": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.EventHub/namespaces",
		ResourceLabel:        "",
		ServiceName:          "EventHub",
		ListDescriber:        DescribeBySubscription(describer.EventhubNamespace),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_eventhub_namespace"},
		TerraformServiceName: "eventhub",
	},
	"Microsoft.MachineLearningServices/workspaces": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.MachineLearningServices/workspaces",
		ResourceLabel:        "",
		ServiceName:          "MachineLearningServices",
		ListDescriber:        DescribeBySubscription(describer.MachineLearningWorkspace),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_machine_learning_workspace"},
		TerraformServiceName: "machinelearning",
	},
	"Microsoft.CostManagement/CostByResourceType": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.CostManagement/CostByResourceType",
		ResourceLabel:        "",
		ServiceName:          "CostManagement",
		ListDescriber:        DescribeBySubscription(describer.DailyCostByResourceType),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/networkInterfaces": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/networkInterfaces",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkInterface),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_network_interface"},
		TerraformServiceName: "network",
	},
	"Microsoft.Network/publicIPAddresses": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/publicIPAddresses",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.PublicIPAddress),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_public_ip"},
		TerraformServiceName: "network",
	},
	"Microsoft.HealthcareApis/services": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.HealthcareApis/services",
		ResourceLabel:        "",
		ServiceName:          "HealthcareApis",
		ListDescriber:        DescribeBySubscription(describer.HealthcareService),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_healthcare_service"},
		TerraformServiceName: "healthcare",
	},
	"Microsoft.ServiceBus/namespaces": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.ServiceBus/namespaces",
		ResourceLabel:        "",
		ServiceName:          "ServiceBus",
		ListDescriber:        DescribeBySubscription(describer.ServicebusNamespace),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_servicebus_namespace"},
		TerraformServiceName: "servicebus",
	},
	"Microsoft.Web/sites": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Web/sites",
		ResourceLabel:        "",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServiceFunctionApp),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_function_app"},
		TerraformServiceName: "web",
	},
	"Microsoft.Compute/availabilitySets": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/availabilitySets",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeAvailabilitySet),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_availability_set"},
		TerraformServiceName: "compute",
	},
	"Microsoft.Network/virtualNetworks": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/virtualNetworks",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.VirtualNetwork),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_virtual_network"},
		TerraformServiceName: "network",
	},
	"Microsoft.Security/securityContacts": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Security/securityContacts",
		ResourceLabel:        "",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterContact),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_security_center_contact"},
		TerraformServiceName: "securitycenter",
	},
	"Microsoft.Compute/diskswriteops": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/diskswriteops",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskWriteOps),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/diskswriteopshourly": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/diskswriteopshourly",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskWriteOpsHourly),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.EventGrid/domains": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.EventGrid/domains",
		ResourceLabel:        "",
		ServiceName:          "EventGrid",
		ListDescriber:        DescribeBySubscription(describer.EventGridDomain),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_eventgrid_domain"},
		TerraformServiceName: "eventgrid",
	},
	"Microsoft.KeyVault/deletedVaults": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.KeyVault/deletedVaults",
		ResourceLabel:        "",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.DeletedVault),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Storage/tables": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Storage/tables",
		ResourceLabel:        "",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageTable),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_table"},
		TerraformServiceName: "storage",
	},
	"Microsoft.Resources/users": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Resources/users",
		ResourceLabel:        "",
		ServiceName:          "Resources",
		ListDescriber:        DescribeADByTenantID(describer.AdUsers),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/snapshots": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/snapshots",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeSnapshots),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_snapshot"},
		TerraformServiceName: "compute",
	},
	"Microsoft.Kusto/clusters": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Kusto/clusters",
		ResourceLabel:        "",
		ServiceName:          "Kusto",
		ListDescriber:        DescribeBySubscription(describer.KustoCluster),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_kusto_cluster"},
		TerraformServiceName: "kusto",
	},
	"Microsoft.StorageSync/storageSyncServices": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.StorageSync/storageSyncServices",
		ResourceLabel:        "",
		ServiceName:          "StorageSync",
		ListDescriber:        DescribeBySubscription(describer.StorageSync),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_sync"},
		TerraformServiceName: "storage",
	},
	"Microsoft.Security/locations/jitNetworkAccessPolicies": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Security/locations/jitNetworkAccessPolicies",
		ResourceLabel:        "",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterJitNetworkAccessPolicy),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/virtualNetworks/subnets": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/virtualNetworks/subnets",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.Subnet),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_subnet"},
		TerraformServiceName: "network",
	},
	"Microsoft.LoadBalancer/backendAddressPools": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.LoadBalancer/backendAddressPools",
		ResourceLabel:        "",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerBackendAddressPool),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb_backend_address_pool"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.LoadBalancer/rules": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.LoadBalancer/rules",
		ResourceLabel:        "",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerRule),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb_rule"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.Compute/virtualMachineCpuUtilizationDaily": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/virtualMachineCpuUtilizationDaily",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineCpuUtilizationDaily),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.DataLakeStore/accounts": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DataLakeStore/accounts",
		ResourceLabel:        "",
		ServiceName:          "DataLakeStore",
		ListDescriber:        DescribeBySubscription(describer.DataLakeStore),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.StorageCache/caches": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.StorageCache/caches",
		ResourceLabel:        "",
		ServiceName:          "StorageCache",
		ListDescriber:        DescribeBySubscription(describer.HpcCache),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_hpc_cache"},
		TerraformServiceName: "hpccache",
	},
	"Microsoft.Batch/batchAccounts": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Batch/batchAccounts",
		ResourceLabel:        "",
		ServiceName:          "Batch",
		ListDescriber:        DescribeBySubscription(describer.BatchAccount),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_batch_account"},
		TerraformServiceName: "batch",
	},
	"Microsoft.ClassicNetwork/networkSecurityGroups": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.ClassicNetwork/networkSecurityGroups",
		ResourceLabel:        "",
		ServiceName:          "ClassicNetwork",
		ListDescriber:        DescribeBySubscription(describer.NetworkSecurityGroup),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_network_security_group"},
		TerraformServiceName: "network",
	},
	"Microsoft.Authorization/roleDefinitions": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Authorization/roleDefinitions",
		ResourceLabel:        "",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.RoleDefinition),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_role_definition"},
		TerraformServiceName: "authorization",
	},
	"Microsoft.Network/applicationSecurityGroups": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/applicationSecurityGroups",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkApplicationSecurityGroups),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_application_security_group"},
		TerraformServiceName: "network",
	},
	"Microsoft.Authorization/elevateAccessRoleAssignment": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Authorization/elevateAccessRoleAssignment",
		ResourceLabel:        "",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.RoleAssignment),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.DocumentDB/MongoDatabases": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DocumentDB/MongoDatabases",
		ResourceLabel:        "",
		ServiceName:          "DocumentDB",
		ListDescriber:        DescribeBySubscription(describer.DocumentDBMongoDatabase),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_cosmosdb_mongo_database"},
		TerraformServiceName: "cosmos",
	},
	"Microsoft.Network/networkWatchers/flowLogs": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/networkWatchers/flowLogs",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkWatcherFlowLog),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_network_watcher_flow_log"},
		TerraformServiceName: "network",
	},
	"Microsoft.Sql/elasticPools": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Sql/elasticPools",
		ResourceLabel:        "",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServerElasticPool),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_sql_elasticpool"},
		TerraformServiceName: "sql",
	},
	"Microsoft.Security/subAssessments": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Security/subAssessments",
		ResourceLabel:        "",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterSubAssessment),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/disks": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/disks",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDisk),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_managed_disk"},
		TerraformServiceName: "",
	},
	"Microsoft.Devices/iotHubDpses": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Devices/iotHubDpses",
		ResourceLabel:        "",
		ServiceName:          "Devices",
		ListDescriber:        DescribeBySubscription(describer.IOTHubDps),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.HDInsight/clusters": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.HDInsight/clusters",
		ResourceLabel:        "",
		ServiceName:          "HDInsight",
		ListDescriber:        DescribeBySubscription(describer.HdInsightCluster),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_hdinsight_cluster"},
		TerraformServiceName: "hdinsight",
	},
	"Microsoft.ServiceFabric/clusters": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.ServiceFabric/clusters",
		ResourceLabel:        "",
		ServiceName:          "ServiceFabric",
		ListDescriber:        DescribeBySubscription(describer.ServiceFabricCluster),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_service_fabric_cluster"},
		TerraformServiceName: "servicefabric",
	},
	"Microsoft.SignalRService/signalR": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.SignalRService/signalR",
		ResourceLabel:        "",
		ServiceName:          "SignalRService",
		ListDescriber:        DescribeBySubscription(describer.SignalrService),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_signalr_service"},
		TerraformServiceName: "signalr",
	},
	"Microsoft.Storage/blobs": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Storage/blobs",
		ResourceLabel:        "",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageBlob),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_blob"},
		TerraformServiceName: "storage",
	},
	"Microsoft.Storage/blobServives": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Storage/blobServives",
		ResourceLabel:        "",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageBlobService),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Storage/queues": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Storage/queues",
		ResourceLabel:        "",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageQueue),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_queue"},
		TerraformServiceName: "storage",
	},
	"Microsoft.ApiManagement/service": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.ApiManagement/service",
		ResourceLabel:        "",
		ServiceName:          "ApiManagement",
		ListDescriber:        DescribeBySubscription(describer.APIManagement),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_api_management"},
		TerraformServiceName: "apimanagement",
	},
	"Microsoft.Compute/disksreadops": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/disksreadops",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskReadOps),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineScaleSets": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/virtualMachineScaleSets",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineScaleSet),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_windows_virtual_machine_scale_set", "azurerm_linux_virtual_machine_scale_set", "azurerm_orchestrated_virtual_machine_scale_set"},
		TerraformServiceName: "compute",
	},
	"Microsoft.DataFactory/factoriesDatasets": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DataFactory/factoriesDatasets",
		ResourceLabel:        "",
		ServiceName:          "DataFactory",
		ListDescriber:        DescribeBySubscription(describer.DataFactoryDataset),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_data_factory_dataset_azure_blob", "azurerm_data_factory_dataset_binary", "azurerm_data_factory_dataset_cosmosdb_sqlapi", "azurerm_data_factory_dataset_delimited_text", "azurerm_data_factory_dataset_http", "azurerm_data_factory_dataset_json", "azurerm_data_factory_dataset_mysql", "azurerm_data_factory_dataset_parquet", "azurerm_data_factory_dataset_postgresql", "azurerm_data_factory_dataset_snowflake", "azurerm_data_factory_dataset_sql_server_table", "azurerm_data_factory_custom_dataset"},
		TerraformServiceName: "datafactory",
	},
	"Microsoft.Authorization/policyDefinitions": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Authorization/policyDefinitions",
		ResourceLabel:        "",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.PolicyDefinition),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Resources/subscriptions/locations": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Resources/subscriptions/locations",
		ResourceLabel:        "",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.Location),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/diskAccesses": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/diskAccesses",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskAccess),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_disk_access"},
		TerraformServiceName: "compute",
	},
	"Microsoft.DBforMySQL/servers": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DBforMySQL/servers",
		ResourceLabel:        "",
		ServiceName:          "DBforMySQL",
		ListDescriber:        DescribeBySubscription(describer.MysqlServer),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_mysql_server"},
		TerraformServiceName: "mysql",
	},
	"Microsoft.DataLakeAnalytics/accounts": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DataLakeAnalytics/accounts",
		ResourceLabel:        "",
		ServiceName:          "DataLakeAnalytics",
		ListDescriber:        DescribeBySubscription(describer.DataLakeAnalyticsAccount),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Insights/activityLogAlerts": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Insights/activityLogAlerts",
		ResourceLabel:        "",
		ServiceName:          "Insights",
		ListDescriber:        DescribeBySubscription(describer.LogAlert),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineCpuUtilizationHourly": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/virtualMachineCpuUtilizationHourly",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineCpuUtilizationHourly),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.LoadBalancer/outboundRules": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.LoadBalancer/outboundRules",
		ResourceLabel:        "",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerOutboundRule),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb_outbound_rule"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.HybridCompute/machines": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.HybridCompute/machines",
		ResourceLabel:        "",
		ServiceName:          "HybridCompute",
		ListDescriber:        DescribeBySubscription(describer.HybridComputeMachine),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_hybrid_compute_machine"},
		TerraformServiceName: "hybridcompute",
	},
	"Microsoft.LoadBalancer/natRules": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.LoadBalancer/natRules",
		ResourceLabel:        "",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerNatRule),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb_nat_rule"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.Resources/providers": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Resources/providers",
		ResourceLabel:        "",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.ResourceProvider),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/routeTables": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/routeTables",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.RouteTables),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_route_table"},
		TerraformServiceName: "network",
	},
	"Microsoft.DocumentDB/databaseAccounts": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DocumentDB/databaseAccounts",
		ResourceLabel:        "",
		ServiceName:          "DocumentDB",
		ListDescriber:        DescribeBySubscription(describer.CosmosdbAccount),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_cosmosdb_account"},
		TerraformServiceName: "cosmos",
	},
	"Microsoft.Network/applicationGateways": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/applicationGateways",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.ApplicationGateway),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_application_gateway"},
		TerraformServiceName: "network",
	},
	"Microsoft.Security/automations": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Security/automations",
		ResourceLabel:        "",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterAutomation),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_security_center_automation"},
		TerraformServiceName: "securitycenter",
	},
	"Microsoft.Kubernetes/connectedClusters": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Kubernetes/connectedClusters",
		ResourceLabel:        "",
		ServiceName:          "Kubernetes",
		ListDescriber:        DescribeBySubscription(describer.HybridKubernetesConnectedCluster),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.KeyVault/vaults/keys": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.KeyVault/vaults/keys",
		ResourceLabel:        "",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVaultKey),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.DBforMariaDB/servers": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.DBforMariaDB/servers",
		ResourceLabel:        "",
		ServiceName:          "DBforMariaDB",
		ListDescriber:        DescribeBySubscription(describer.MariadbServer),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_mariadb_server"},
		TerraformServiceName: "mariadb",
	},
	"Microsoft.Compute/disksreadopsdaily": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/disksreadopsdaily",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskReadOpsDaily),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Web/plan": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Web/plan",
		ResourceLabel:        "",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServicePlan),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_app_service_plan"},
		TerraformServiceName: "web",
	},
	"Microsoft.Compute/disksreadopshourly": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/disksreadopshourly",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskReadOpsHourly),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/diskswriteopsdaily": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/diskswriteopsdaily",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskWriteOpsDaily),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Resources/tenants": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Resources/tenants",
		ResourceLabel:        "",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.Tenant),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Network/virtualNetworkGateways": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/virtualNetworkGateways",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.VirtualNetworkGateway),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_virtual_network_gateway"},
		TerraformServiceName: "network",
	},
	"Microsoft.Devices/iotHubs": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Devices/iotHubs",
		ResourceLabel:        "",
		ServiceName:          "Devices",
		ListDescriber:        DescribeBySubscription(describer.IOTHub),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_iothub"},
		TerraformServiceName: "iothub",
	},
	"Microsoft.Logic/workflows": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Logic/workflows",
		ResourceLabel:        "",
		ServiceName:          "Logic",
		ListDescriber:        DescribeBySubscription(describer.LogicAppWorkflow),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_logic_app_workflow"},
		TerraformServiceName: "logic",
	},
	"Microsoft.Sql/flexibleServers": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Sql/flexibleServers",
		ResourceLabel:        "",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServerFlexibleServer),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Resources/links": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Resources/links",
		ResourceLabel:        "",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.ResourceLink),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Resources/subscriptions": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Resources/subscriptions",
		ResourceLabel:        "",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.Subscription),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_subscription"},
		TerraformServiceName: "subscription",
	},
	"Microsoft.Compute/image": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/image",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeImage),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_image"},
		TerraformServiceName: "compute",
	},
	"Microsoft.Compute/virtualMachines": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/virtualMachines",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachine),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine"},
		TerraformServiceName: "compute",
	},
	"Microsoft.Network/natGateways": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Network/natGateways",
		ResourceLabel:        "",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NatGateway),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_nat_gateway"},
		TerraformServiceName: "network",
	},
	"Microsoft.LoadBalancer/probes": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.LoadBalancer/probes",
		ResourceLabel:        "",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerProbe),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_lb_probe"},
		TerraformServiceName: "loadbalancer",
	},
	"Microsoft.KeyVault/vaults": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.KeyVault/vaults",
		ResourceLabel:        "",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVault),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_key_vault"},
		TerraformServiceName: "keyvault",
	},
	"Microsoft.KeyVault/managedHsms": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.KeyVault/managedHsms",
		ResourceLabel:        "",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVaultManagedHardwareSecurityModule),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_key_vault_managed_hardware_security_module"},
		TerraformServiceName: "keyvault",
	},
	"Microsoft.KeyVault/vaults/secrets": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.KeyVault/vaults/secrets",
		ResourceLabel:        "",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVaultSecret),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_key_vault_secret"},
		TerraformServiceName: "keyvault",
	},
	"Microsoft.AppConfiguration/configurationStores": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.AppConfiguration/configurationStores",
		ResourceLabel:        "",
		ServiceName:          "AppConfiguration",
		ListDescriber:        DescribeBySubscription(describer.AppConfiguration),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineCpuUtilization": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Compute/virtualMachineCpuUtilization",
		ResourceLabel:        "",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineCpuUtilization),
		GetDescriber:         nil,
		TerraformName:        nil,
		TerraformServiceName: "",
	},
	"Microsoft.Storage/storageAccounts": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.Storage/storageAccounts",
		ResourceLabel:        "",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageAccount),
		GetDescriber:         nil,
		TerraformName:        []string{"azurerm_storage_account"},
		TerraformServiceName: "storage",
	},
	"Microsoft.AppPlatform/Spring": {
		Connector:            source.CloudAzure,
		ResourceName:         "Microsoft.AppPlatform/Spring",
		ResourceLabel:        "",
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

func GetResourceType(resourceType string) (*ResourceType, error) {
	if r, ok := resourceTypes[resourceType]; ok {
		return &r, nil
	}

	return nil, fmt.Errorf("resource type %s not found", resourceType)
}

func GetResourceTypesMap() map[string]ResourceType {
	return resourceTypes
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
	stream *describer.StreamSender,
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

	resources, err := describe(ctx, authorizer, hamiltonAuthorizer, resourceType, subscriptions, cfg.TenantID, triggerType, stream)
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

func describe(ctx context.Context, authorizer autorest.Authorizer, hamiltonAuth hamiltonAuth.Authorizer, resourceType string, subscriptions []string, tenantId string, triggerType enums.DescribeTriggerType, stream *describer.StreamSender) ([]describer.Resource, error) {
	resourceTypeObject, ok := resourceTypes[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	listDescriber := resourceTypeObject.ListDescriber
	if listDescriber == nil {
		listDescriber = describer.GenericResourceGraph{Table: "Resources", Type: resourceType}
	}

	return listDescriber.DescribeResources(ctx, authorizer, hamiltonAuth, subscriptions, tenantId, triggerType, stream)
}

func DescribeBySubscription(describe func(context.Context, autorest.Authorizer, string, *describer.StreamSender) ([]describer.Resource, error)) ResourceDescriber {
	return ResourceDescribeFunc(func(ctx context.Context, authorizer autorest.Authorizer, hamiltonAuth hamiltonAuth.Authorizer, subscriptions []string, tenantId string, triggerType enums.DescribeTriggerType, stream *describer.StreamSender) ([]describer.Resource, error) {
		ctx = describer.WithTriggerType(ctx, triggerType)
		values := []describer.Resource{}
		for _, subscription := range subscriptions {
			result, err := describe(ctx, authorizer, subscription, stream)
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

func DescribeADByTenantID(describe func(context.Context, hamiltonAuth.Authorizer, string, *describer.StreamSender) ([]describer.Resource, error)) ResourceDescriber {
	return ResourceDescribeFunc(func(ctx context.Context, authorizer autorest.Authorizer, hamiltonAuth hamiltonAuth.Authorizer, subscription []string, tenantId string, triggerType enums.DescribeTriggerType, stream *describer.StreamSender) ([]describer.Resource, error) {
		ctx = describer.WithTriggerType(ctx, triggerType)
		var values []describer.Resource
		result, err := describe(ctx, hamiltonAuth, tenantId, stream)
		if err != nil {
			return nil, err
		}

		values = append(values, result...)

		return values, nil
	})
}
