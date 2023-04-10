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

	TerraformName        string
	TerraformServiceName string
}

var resourceTypes = map[string]ResourceType{
	"Microsoft.Compute/virtualMachineScaleSetsVm": {
		Name:                 "Microsoft.Compute/virtualMachineScaleSetsVm",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineScaleSetVm),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.EventGrid/domains/topics": {
		Name:                 "Microsoft.EventGrid/domains/topics",
		ServiceName:          "EventGrid",
		ListDescriber:        DescribeBySubscription(describer.EventGridDomainTopic),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/networkWatchers": {
		Name:                 "Microsoft.Network/networkWatchers",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkWatcher),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Resources/resourceGroups": {
		Name:                 "Microsoft.Resources/resourceGroups",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.ResourceGroup),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Web/staticSites": {
		Name:                 "Microsoft.Web/staticSites",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServiceWebApp),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Resources/serviceprincipals": {
		Name:                 "Microsoft.Resources/serviceprincipals",
		ServiceName:          "Resources",
		ListDescriber:        DescribeADByTenantID(describer.AdServicePrinciple),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.CognitiveServices/accounts": {
		Name:                 "Microsoft.CognitiveServices/accounts",
		ServiceName:          "CognitiveServices",
		ListDescriber:        DescribeBySubscription(describer.CognitiveAccount),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Sql/managedInstances": {
		Name:                 "Microsoft.Sql/managedInstances",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.MssqlManagedInstance),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Sql/servers/databases": {
		Name:                 "Microsoft.Sql/servers/databases",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlDatabase),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Storage/fileShares": {
		Name:                 "Microsoft.Storage/fileShares",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageFileShare),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DBforPostgreSQL/servers": {
		Name:                 "Microsoft.DBforPostgreSQL/servers",
		ServiceName:          "DBforPostgreSQL",
		ListDescriber:        DescribeBySubscription(describer.PostgresqlServer),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Security/pricings": {
		Name:                 "Microsoft.Security/pricings",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterSubscriptionPricing),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Insights/guestDiagnosticSettings": {
		Name:                 "Microsoft.Insights/guestDiagnosticSettings",
		ServiceName:          "Insights",
		ListDescriber:        DescribeBySubscription(describer.DiagnosticSetting),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Resources/groups": {
		Name:                 "Microsoft.Resources/groups",
		ServiceName:          "Resources",
		ListDescriber:        DescribeADByTenantID(describer.AdGroup),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Web/hostingEnvironments": {
		Name:                 "Microsoft.Web/hostingEnvironments",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServiceEnvironment),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Cache/redis": {
		Name:                 "Microsoft.Cache/redis",
		ServiceName:          "Cache",
		ListDescriber:        DescribeBySubscription(describer.RedisCache),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.ContainerRegistry/registries": {
		Name:                 "Microsoft.ContainerRegistry/registries",
		ServiceName:          "ContainerRegistry",
		ListDescriber:        DescribeBySubscription(describer.ContainerRegistry),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DataFactory/factoriesPipelines": {
		Name:                 "Microsoft.DataFactory/factoriesPipelines",
		ServiceName:          "DataFactory",
		ListDescriber:        DescribeBySubscription(describer.DataFactoryPipeline),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/resourceSku": {
		Name:                 "Microsoft.Compute/resourceSku",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeResourceSKU),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/expressRouteCircuits": {
		Name:                 "Microsoft.Network/expressRouteCircuits",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.ExpressRouteCircuit),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Management/groups": {
		Name:                 "Microsoft.Management/groups",
		ServiceName:          "Management",
		ListDescriber:        DescribeBySubscription(describer.ManagementGroup),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Sql/virtualMachines": {
		Name:                 "Microsoft.Sql/virtualMachines",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServerVirtualMachine),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Storage/tableServices": {
		Name:                 "Microsoft.Storage/tableServices",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageTableService),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Synapse/workspaces": {
		Name:                 "Microsoft.Synapse/workspaces",
		ServiceName:          "Synapse",
		ListDescriber:        DescribeBySubscription(describer.SynapseWorkspace),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.StreamAnalytics/streamingJobs": {
		Name:                 "Microsoft.StreamAnalytics/streamingJobs",
		ServiceName:          "StreamAnalytics",
		ListDescriber:        DescribeBySubscription(describer.StreamAnalyticsJob),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.CostManagement/CostBySubscription": {
		Name:                 "Microsoft.CostManagement/CostBySubscription",
		ServiceName:          "CostManagement",
		ListDescriber:        DescribeBySubscription(describer.DailyCostBySubscription),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.ContainerService/managedClusters": {
		Name:                 "Microsoft.ContainerService/managedClusters",
		ServiceName:          "ContainerService",
		ListDescriber:        DescribeBySubscription(describer.KubernetesCluster),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DataFactory/factories": {
		Name:                 "Microsoft.DataFactory/factories",
		ServiceName:          "DataFactory",
		ListDescriber:        DescribeBySubscription(describer.DataFactory),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Sql/servers": {
		Name:                 "Microsoft.Sql/servers",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServer),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Security/autoProvisioningSettings": {
		Name:                 "Microsoft.Security/autoProvisioningSettings",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterAutoProvisioning),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Insights/logProfiles": {
		Name:                 "Microsoft.Insights/logProfiles",
		ServiceName:          "Insights",
		ListDescriber:        DescribeBySubscription(describer.LogProfile),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DataBoxEdge/dataBoxEdgeDevices": {
		Name:                 "Microsoft.DataBoxEdge/dataBoxEdgeDevices",
		ServiceName:          "DataBoxEdge",
		ListDescriber:        DescribeBySubscription(describer.DataboxEdgeDevice),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/loadBalancers": {
		Name:                 "Microsoft.Network/loadBalancers",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancer),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/azureFirewalls": {
		Name:                 "Microsoft.Network/azureFirewalls",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkAzureFirewall),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Management/locks": {
		Name:                 "Microsoft.Management/locks",
		ServiceName:          "Management",
		ListDescriber:        DescribeBySubscription(describer.ManagementLock),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineScaleSetNetworkInterface": {
		Name:                 "Microsoft.Compute/virtualMachineScaleSetNetworkInterface",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineScaleSetNetworkInterface),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/frontDoors": {
		Name:                 "Microsoft.Network/frontDoors",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.FrontDoor),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Resources/subscriptions/resourceGroups": {
		Name:                 "Microsoft.Resources/subscriptions/resourceGroups",
		ServiceName:          "Resources",
		ListDescriber:        describer.GenericResourceGraph{Table: "ResourceContainers", Type: "Microsoft.Resources/subscriptions/resourceGroups"},
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Authorization/policyAssignments": {
		Name:                 "Microsoft.Authorization/policyAssignments",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.PolicyAssignment),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Search/searchServices": {
		Name:                 "Microsoft.Search/searchServices",
		ServiceName:          "Search",
		ListDescriber:        DescribeBySubscription(describer.SearchService),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Security/settings": {
		Name:                 "Microsoft.Security/settings",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterSetting),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.RecoveryServices/vaults": {
		Name:                 "Microsoft.RecoveryServices/vaults",
		ServiceName:          "RecoveryServices",
		ListDescriber:        DescribeBySubscription(describer.RecoveryServicesVault),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/diskEncryptionSets": {
		Name:                 "Microsoft.Compute/diskEncryptionSets",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskEncryptionSet),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DocumentDB/databaseAccountsSqlDatabases": {
		Name:                 "Microsoft.DocumentDB/databaseAccountsSqlDatabases",
		ServiceName:          "DocumentDB",
		ListDescriber:        DescribeBySubscription(describer.DocumentDBDatabaseAccountsSQLDatabase),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.EventGrid/topics": {
		Name:                 "Microsoft.EventGrid/topics",
		ServiceName:          "EventGrid",
		ListDescriber:        DescribeBySubscription(describer.EventGridTopic),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.EventHub/namespaces": {
		Name:                 "Microsoft.EventHub/namespaces",
		ServiceName:          "EventHub",
		ListDescriber:        DescribeBySubscription(describer.EventhubNamespace),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.MachineLearningServices/workspaces": {
		Name:                 "Microsoft.MachineLearningServices/workspaces",
		ServiceName:          "MachineLearningServices",
		ListDescriber:        DescribeBySubscription(describer.MachineLearningWorkspace),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.CostManagement/CostByResourceType": {
		Name:                 "Microsoft.CostManagement/CostByResourceType",
		ServiceName:          "CostManagement",
		ListDescriber:        DescribeBySubscription(describer.DailyCostByResourceType),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/networkInterfaces": {
		Name:                 "Microsoft.Network/networkInterfaces",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkInterface),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/publicIPAddresses": {
		Name:                 "Microsoft.Network/publicIPAddresses",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.PublicIPAddress),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.HealthcareApis/services": {
		Name:                 "Microsoft.HealthcareApis/services",
		ServiceName:          "HealthcareApis",
		ListDescriber:        DescribeBySubscription(describer.HealthcareService),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.ServiceBus/namespaces": {
		Name:                 "Microsoft.ServiceBus/namespaces",
		ServiceName:          "ServiceBus",
		ListDescriber:        DescribeBySubscription(describer.ServicebusNamespace),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Web/sites": {
		Name:                 "Microsoft.Web/sites",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServiceFunctionApp),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/availabilitySets": {
		Name:                 "Microsoft.Compute/availabilitySets",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeAvailabilitySet),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/virtualNetworks": {
		Name:                 "Microsoft.Network/virtualNetworks",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.VirtualNetwork),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Security/securityContacts": {
		Name:                 "Microsoft.Security/securityContacts",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterContact),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/diskswriteops": {
		Name:                 "Microsoft.Compute/diskswriteops",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskWriteOps),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/diskswriteopshourly": {
		Name:                 "Microsoft.Compute/diskswriteopshourly",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskWriteOpsHourly),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.EventGrid/domains": {
		Name:                 "Microsoft.EventGrid/domains",
		ServiceName:          "EventGrid",
		ListDescriber:        DescribeBySubscription(describer.EventGridDomain),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.KeyVault/deletedVaults": {
		Name:                 "Microsoft.KeyVault/deletedVaults",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.DeletedVault),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Storage/tables": {
		Name:                 "Microsoft.Storage/tables",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageTable),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Resources/users": {
		Name:                 "Microsoft.Resources/users",
		ServiceName:          "Resources",
		ListDescriber:        DescribeADByTenantID(describer.AdUsers),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/snapshots": {
		Name:                 "Microsoft.Compute/snapshots",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeSnapshots),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Kusto/clusters": {
		Name:                 "Microsoft.Kusto/clusters",
		ServiceName:          "Kusto",
		ListDescriber:        DescribeBySubscription(describer.KustoCluster),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.StorageSync/storageSyncServices": {
		Name:                 "Microsoft.StorageSync/storageSyncServices",
		ServiceName:          "StorageSync",
		ListDescriber:        DescribeBySubscription(describer.StorageSync),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Security/locations/jitNetworkAccessPolicies": {
		Name:                 "Microsoft.Security/locations/jitNetworkAccessPolicies",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterJitNetworkAccessPolicy),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/virtualNetworks/subnets": {
		Name:                 "Microsoft.Network/virtualNetworks/subnets",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.Subnet),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.LoadBalancer/backendAddressPools": {
		Name:                 "Microsoft.LoadBalancer/backendAddressPools",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerBackendAddressPool),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.LoadBalancer/rules": {
		Name:                 "Microsoft.LoadBalancer/rules",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerRule),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineCpuUtilizationDaily": {
		Name:                 "Microsoft.Compute/virtualMachineCpuUtilizationDaily",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineCpuUtilizationDaily),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DataLakeStore/accounts": {
		Name:                 "Microsoft.DataLakeStore/accounts",
		ServiceName:          "DataLakeStore",
		ListDescriber:        DescribeBySubscription(describer.DataLakeStore),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.StorageCache/caches": {
		Name:                 "Microsoft.StorageCache/caches",
		ServiceName:          "StorageCache",
		ListDescriber:        DescribeBySubscription(describer.HpcCache),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Batch/batchAccounts": {
		Name:                 "Microsoft.Batch/batchAccounts",
		ServiceName:          "Batch",
		ListDescriber:        DescribeBySubscription(describer.BatchAccount),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.ClassicNetwork/networkSecurityGroups": {
		Name:                 "Microsoft.ClassicNetwork/networkSecurityGroups",
		ServiceName:          "ClassicNetwork",
		ListDescriber:        DescribeBySubscription(describer.NetworkSecurityGroup),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Authorization/roleDefinitions": {
		Name:                 "Microsoft.Authorization/roleDefinitions",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.RoleDefinition),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/applicationSecurityGroups": {
		Name:                 "Microsoft.Network/applicationSecurityGroups",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkApplicationSecurityGroups),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Authorization/elevateAccessRoleAssignment": {
		Name:                 "Microsoft.Authorization/elevateAccessRoleAssignment",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.RoleAssignment),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DocumentDB/databaseAccountsMongoDatabases": {
		Name:                 "Microsoft.DocumentDB/databaseAccountsMongoDatabases",
		ServiceName:          "DocumentDB",
		ListDescriber:        DescribeBySubscription(describer.DocumentDBDatabaseAccountsMongoDatabase),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/networkWatchers/flowLogs": {
		Name:                 "Microsoft.Network/networkWatchers/flowLogs",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NetworkWatcherFlowLog),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Sql/elasticPools": {
		Name:                 "Microsoft.Sql/elasticPools",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServerElasticPool),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Security/subAssessments": {
		Name:                 "Microsoft.Security/subAssessments",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterSubAssessment),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/disks": {
		Name:                 "Microsoft.Compute/disks",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDisk),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Devices/iotHubDpses": {
		Name:                 "Microsoft.Devices/iotHubDpses",
		ServiceName:          "Devices",
		ListDescriber:        DescribeBySubscription(describer.IOTHubDps),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.HDInsight/clusters": {
		Name:                 "Microsoft.HDInsight/clusters",
		ServiceName:          "HDInsight",
		ListDescriber:        DescribeBySubscription(describer.HdInsightCluster),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.ServiceFabric/clusters": {
		Name:                 "Microsoft.ServiceFabric/clusters",
		ServiceName:          "ServiceFabric",
		ListDescriber:        DescribeBySubscription(describer.ServiceFabricCluster),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.SignalRService/signalR": {
		Name:                 "Microsoft.SignalRService/signalR",
		ServiceName:          "SignalRService",
		ListDescriber:        DescribeBySubscription(describer.SignalrService),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Storage/blobs": {
		Name:                 "Microsoft.Storage/blobs",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageBlob),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Storage/blobServives": {
		Name:                 "Microsoft.Storage/blobServives",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageBlobService),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Storage/queues": {
		Name:                 "Microsoft.Storage/queues",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageQueue),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.ApiManagement/service": {
		Name:                 "Microsoft.ApiManagement/service",
		ServiceName:          "ApiManagement",
		ListDescriber:        DescribeBySubscription(describer.APIManagement),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/disksreadops": {
		Name:                 "Microsoft.Compute/disksreadops",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskReadOps),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineScaleSets": {
		Name:                 "Microsoft.Compute/virtualMachineScaleSets",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineScaleSet),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DataFactory/factoriesDatasets": {
		Name:                 "Microsoft.DataFactory/factoriesDatasets",
		ServiceName:          "DataFactory",
		ListDescriber:        DescribeBySubscription(describer.DataFactoryDataset),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Authorization/policyDefinitions": {
		Name:                 "Microsoft.Authorization/policyDefinitions",
		ServiceName:          "Authorization",
		ListDescriber:        DescribeBySubscription(describer.PolicyDefinition),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Resources/subscriptions/locations": {
		Name:                 "Microsoft.Resources/subscriptions/locations",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.Location),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/diskAccesses": {
		Name:                 "Microsoft.Compute/diskAccesses",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskAccess),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DBforMySQL/servers": {
		Name:                 "Microsoft.DBforMySQL/servers",
		ServiceName:          "DBforMySQL",
		ListDescriber:        DescribeBySubscription(describer.MysqlServer),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DataLakeAnalytics/accounts": {
		Name:                 "Microsoft.DataLakeAnalytics/accounts",
		ServiceName:          "DataLakeAnalytics",
		ListDescriber:        DescribeBySubscription(describer.DataLakeAnalyticsAccount),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Insights/activityLogAlerts": {
		Name:                 "Microsoft.Insights/activityLogAlerts",
		ServiceName:          "Insights",
		ListDescriber:        DescribeBySubscription(describer.LogAlert),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineCpuUtilizationHourly": {
		Name:                 "Microsoft.Compute/virtualMachineCpuUtilizationHourly",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineCpuUtilizationHourly),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.LoadBalancer/outboundRules": {
		Name:                 "Microsoft.LoadBalancer/outboundRules",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerOutboundRule),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.HybridCompute/machines": {
		Name:                 "Microsoft.HybridCompute/machines",
		ServiceName:          "HybridCompute",
		ListDescriber:        DescribeBySubscription(describer.HybridComputeMachine),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.LoadBalancer/natRules": {
		Name:                 "Microsoft.LoadBalancer/natRules",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerNatRule),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Resources/providers": {
		Name:                 "Microsoft.Resources/providers",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.ResourceProvider),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/routeTables": {
		Name:                 "Microsoft.Network/routeTables",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.RouteTables),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DocumentDB/databaseAccounts": {
		Name:                 "Microsoft.DocumentDB/databaseAccounts",
		ServiceName:          "DocumentDB",
		ListDescriber:        DescribeBySubscription(describer.CosmosdbAccount),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/applicationGateways": {
		Name:                 "Microsoft.Network/applicationGateways",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.ApplicationGateway),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Security/automations": {
		Name:                 "Microsoft.Security/automations",
		ServiceName:          "Security",
		ListDescriber:        DescribeBySubscription(describer.SecurityCenterAutomation),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Kubernetes/connectedClusters": {
		Name:                 "Microsoft.Kubernetes/connectedClusters",
		ServiceName:          "Kubernetes",
		ListDescriber:        DescribeBySubscription(describer.HybridKubernetesConnectedCluster),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.KeyVault/vaults/keys": {
		Name:                 "Microsoft.KeyVault/vaults/keys",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVaultKey),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.DBforMariaDB/servers": {
		Name:                 "Microsoft.DBforMariaDB/servers",
		ServiceName:          "DBforMariaDB",
		ListDescriber:        DescribeBySubscription(describer.MariadbServer),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/disksreadopsdaily": {
		Name:                 "Microsoft.Compute/disksreadopsdaily",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskReadOpsDaily),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Web/plan": {
		Name:                 "Microsoft.Web/plan",
		ServiceName:          "Web",
		ListDescriber:        DescribeBySubscription(describer.AppServicePlan),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/disksreadopshourly": {
		Name:                 "Microsoft.Compute/disksreadopshourly",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskReadOpsHourly),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/diskswriteopsdaily": {
		Name:                 "Microsoft.Compute/diskswriteopsdaily",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeDiskWriteOpsDaily),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Resources/tenants": {
		Name:                 "Microsoft.Resources/tenants",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.Tenant),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/virtualNetworkGateways": {
		Name:                 "Microsoft.Network/virtualNetworkGateways",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.VirtualNetworkGateway),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Devices/iotHubs": {
		Name:                 "Microsoft.Devices/iotHubs",
		ServiceName:          "Devices",
		ListDescriber:        DescribeBySubscription(describer.IOTHub),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Logic/workflows": {
		Name:                 "Microsoft.Logic/workflows",
		ServiceName:          "Logic",
		ListDescriber:        DescribeBySubscription(describer.LogicAppWorkflow),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Sql/flexibleServers": {
		Name:                 "Microsoft.Sql/flexibleServers",
		ServiceName:          "Sql",
		ListDescriber:        DescribeBySubscription(describer.SqlServerFlexibleServer),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Resources/links": {
		Name:                 "Microsoft.Resources/links",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.ResourceLink),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Resources/subscriptions": {
		Name:                 "Microsoft.Resources/subscriptions",
		ServiceName:          "Resources",
		ListDescriber:        DescribeBySubscription(describer.Subscription),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/image": {
		Name:                 "Microsoft.Compute/image",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeImage),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachines": {
		Name:                 "Microsoft.Compute/virtualMachines",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachine),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Network/natGateways": {
		Name:                 "Microsoft.Network/natGateways",
		ServiceName:          "Network",
		ListDescriber:        DescribeBySubscription(describer.NatGateway),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.LoadBalancer/probes": {
		Name:                 "Microsoft.LoadBalancer/probes",
		ServiceName:          "LoadBalancer",
		ListDescriber:        DescribeBySubscription(describer.LoadBalancerProbe),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.KeyVault/vaults": {
		Name:                 "Microsoft.KeyVault/vaults",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVault),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.KeyVault/managedHsms": {
		Name:                 "Microsoft.KeyVault/managedHsms",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVaultManagedHardwareSecurityModule),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.KeyVault/vaults/secrets": {
		Name:                 "Microsoft.KeyVault/vaults/secrets",
		ServiceName:          "KeyVault",
		ListDescriber:        DescribeBySubscription(describer.KeyVaultSecret),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.AppConfiguration/configurationStores": {
		Name:                 "Microsoft.AppConfiguration/configurationStores",
		ServiceName:          "AppConfiguration",
		ListDescriber:        DescribeBySubscription(describer.AppConfiguration),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Compute/virtualMachineCpuUtilization": {
		Name:                 "Microsoft.Compute/virtualMachineCpuUtilization",
		ServiceName:          "Compute",
		ListDescriber:        DescribeBySubscription(describer.ComputeVirtualMachineCpuUtilization),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.Storage/storageAccounts": {
		Name:                 "Microsoft.Storage/storageAccounts",
		ServiceName:          "Storage",
		ListDescriber:        DescribeBySubscription(describer.StorageAccount),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"Microsoft.AppPlatform/Spring": {
		Name:                 "Microsoft.AppPlatform/Spring",
		ServiceName:          "AppPlatform",
		ListDescriber:        DescribeBySubscription(describer.SpringCloudService),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
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
