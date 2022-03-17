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
)

type ResourceDescriber interface {
	DescribeResources(context.Context, autorest.Authorizer, hamiltonAuth.Authorizer, []string, string) ([]describer.Resource, error)
}

type ResourceDescribeFunc func(context.Context, autorest.Authorizer, hamiltonAuth.Authorizer, []string, string) ([]describer.Resource, error)

func (fn ResourceDescribeFunc) DescribeResources(c context.Context, a autorest.Authorizer, ah hamiltonAuth.Authorizer, s []string, t string) ([]describer.Resource, error) {
	return fn(c, a, ah, s, t)
}

var resourceTypeToDescriber = map[string]ResourceDescriber{
	"Microsoft.AnalysisServices/servers":                        nil,
	"Microsoft.ApiManagement/service":                           DescribeBySubscription(describer.APIManagement),
	"Microsoft.AppConfiguration/configurationStores":            DescribeBySubscription(describer.AppConfiguration),
	"Microsoft.Web/HostingEnvironments":                         DescribeBySubscription(describer.AppServiceEnvironment),
	"microsoft.authorization/elevateaccessroleassignment":       DescribeBySubscription(describer.RoleAssignment),
	"Microsoft.Authorization/policyDefinitions":                 nil,
	"Microsoft.Automation/automationAccounts":                   nil,
	"Microsoft.Blueprint/blueprints":                            DescribeBySubscription(describer.BlueprintBlueprint),
	"Microsoft.Blueprint/blueprints/artifacts":                  DescribeBySubscription(describer.BlueprintArtifact),
	"Microsoft.Cache/Redis":                                     DescribeBySubscription(describer.RedisCache),
	"Microsoft.Cdn/profiles":                                    nil,
	"Microsoft.Cdn/profiles/endpoints":                          nil,
	"Microsoft.CognitiveServices/accounts":                      DescribeBySubscription(describer.CognitiveAccount),
	"Microsoft.Compute/availabilitySets":                        nil,
	"Microsoft.Compute/cloudServices":                           nil,
	"Microsoft.Compute/diskEncryptionSets":                      nil,
	"Microsoft.Compute/disks":                                   DescribeBySubscription(describer.ComputeDisk),
	"Microsoft.Compute/diskAccesses":                            DescribeBySubscription(describer.ComputeDiskAccess),
	"Microsoft.Compute/galleries":                               nil,
	"Microsoft.Compute/snapshots":                               nil,
	"Microsoft.Compute/virtualMachineScaleSets":                 DescribeBySubscription(describer.ComputeVirtualMachineScaleSet),
	"Microsoft.Compute/virtualMachines":                         DescribeBySubscription(describer.ComputeVirtualMachine),
	"Microsoft.ContainerInstance/containerGroups":               nil,
	"Microsoft.ContainerRegistry/registries":                    DescribeBySubscription(describer.ContainerRegistry),
	"Microsoft.ContainerService/managedClusters":                DescribeBySubscription(describer.KubernetesCluster),
	"Microsoft.DBforMySQL/servers":                              DescribeBySubscription(describer.MysqlServer),
	"Microsoft.DBforPostgreSQL/servers":                         DescribeBySubscription(describer.PostgresqlServer),
	"Microsoft.DataFactory/factories":                           DescribeBySubscription(describer.DataFactory),
	"Microsoft.DataLakeAnalytics/accounts":                      DescribeBySubscription(describer.DataLakeAnalyticsAccount),
	"Microsoft.DataLakeStore/accounts":                          DescribeBySubscription(describer.DataLakeStore),
	"Microsoft.DataMigration/services":                          nil,
	"Microsoft.Databricks/workspaces":                           nil,
	"Microsoft.Devices/IotHubs":                                 DescribeBySubscription(describer.IOTHub),
	"Microsoft.Devices/provisioningServices":                    nil,
	"Microsoft.Devices/provisioningServices/certificates":       DescribeBySubscription(describer.DevicesProvisioningServicesCertificates),
	"Microsoft.DocumentDB/databaseAccounts":                     DescribeBySubscription(describer.CosmosdbAccount),
	"Microsoft.DocumentDB/databaseAccounts/sqlDatabases":        DescribeBySubscription(describer.DocumentDBDatabaseAccountsSQLDatabase),
	"Microsoft.EventGrid/domains":                               DescribeBySubscription(describer.EventGridDomain),
	"Microsoft.EventGrid/domains/topics":                        DescribeBySubscription(describer.EventGridDomainTopic),
	"Microsoft.EventGrid/topics":                                DescribeBySubscription(describer.EventGridTopic),
	"Microsoft.EventHub/namespaces":                             DescribeBySubscription(describer.EventhubNamespace),
	"Microsoft.HDInsight/clusters":                              DescribeBySubscription(describer.HdInsightCluster),
	"Microsoft.HybridCompute/machines":                          DescribeBySubscription(describer.HybridComputeMachine),
	"Microsoft.Insights/actionGroups":                           nil,
	"Microsoft.Insights/components":                             nil,
	"Microsoft.KeyVault/vaults":                                 DescribeBySubscription(describer.KeyVault),
	"Microsoft.KeyVault/vaults/keys":                            DescribeBySubscription(describer.KeyVaultKey),
	"Microsoft.Kubernetes/connectedClusters":                    nil,
	"Microsoft.Kusto/clusters":                                  DescribeBySubscription(describer.KustoCluster),
	"Microsoft.Kusto/clusters/databases":                        nil,
	"Microsoft.Logic/integrationAccounts":                       nil,
	"Microsoft.Logic/workflows":                                 DescribeBySubscription(describer.LogicAppWorkflow),
	"Microsoft.MachineLearningServices/workspaces":              DescribeBySubscription(describer.MachineLearningWorkspace),
	"Microsoft.ManagedIdentity/userAssignedIdentities":          nil,
	"Microsoft.Management/managementGroups":                     nil,
	"Microsoft.Migrate/assessmentProjects":                      nil,
	"Microsoft.Network/applicationGateways":                     DescribeBySubscription(describer.ApplicationGateway),
	"Microsoft.Network/applicationSecurityGroups":               nil,
	"Microsoft.Network/azureFirewalls":                          nil,
	"Microsoft.Network/bastionHosts":                            nil,
	"Microsoft.Network/connections":                             nil,
	"Microsoft.Network/dnsZones":                                nil,
	"Microsoft.Network/expressRouteCircuits":                    nil,
	"Microsoft.Network/firewallPolicies":                        nil,
	"Microsoft.Network/frontDoors":                              DescribeBySubscription(describer.FrontDoor),
	"Microsoft.Network/frontdoorWebApplicationFirewallPolicies": nil,
	"Microsoft.Network/loadBalancers":                           nil,
	"Microsoft.Network/localNetworkGateways":                    nil,
	"Microsoft.Network/natGateways":                             nil,
	"Microsoft.Network/networkInterfaces":                       DescribeBySubscription(describer.NetworkInterface),
	"Microsoft.Network/networkSecurityGroups":                   nil,
	"Microsoft.Network/networkWatchers":                         DescribeBySubscription(describer.NetworkWatcherFlowLog),
	"Microsoft.Network/privateDnsZones":                         nil,
	"Microsoft.Network/privateLinkServices":                     nil,
	"Microsoft.Network/publicIPAddresses":                       nil,
	"Microsoft.Network/publicIPPrefixes":                        nil,
	"Microsoft.Network/routeFilters":                            nil,
	"Microsoft.Network/routeTables":                             nil,
	"Microsoft.Network/serviceEndpointPolicies":                 nil,
	"Microsoft.Network/trafficManagerProfiles":                  nil,
	"Microsoft.Network/virtualNetworkGateways":                  nil,
	"Microsoft.Network/virtualNetworks":                         DescribeBySubscription(describer.VirtualNetwork),
	"Microsoft.Network/virtualWans":                             nil,
	"Microsoft.Network/vpnGateways":                             nil,
	"Microsoft.NotificationHubs/namespaces":                     nil,
	"Microsoft.NotificationHubs/namespaces/notificationHubs":    nil,
	"Microsoft.OperationalInsights/workspaces":                  nil,
	"Microsoft.PowerBIDedicated/capacities":                     nil,
	"Microsoft.Purview/accounts":                                nil,
	"Microsoft.RecoveryServices/vaults":                         nil,
	"Microsoft.Resources/subscriptions/resourceGroups":          describer.GenericResourceGraph{Table: "ResourceContainers", Type: "Microsoft.Resources/subscriptions/resourceGroups"},
	"Microsoft.Search/searchServices":                           DescribeBySubscription(describer.SearchService),
	"Microsoft.ServiceBus/namespaces":                           DescribeBySubscription(describer.ServicebusNamespace),
	"Microsoft.ServiceBus/namespaces/queues":                    DescribeBySubscription(describer.ServiceBusQueue),
	"Microsoft.ServiceBus/namespaces/topics":                    DescribeBySubscription(describer.ServiceBusTopic),
	"Microsoft.ServiceFabric/clusters":                          DescribeBySubscription(describer.ServiceFabricCluster),
	"Microsoft.SignalRService/SignalR":                          DescribeBySubscription(describer.SignalrService),
	"Microsoft.Sql/managedInstances":                            DescribeBySubscription(describer.MssqlManagedInstance),
	"Microsoft.Sql/servers":                                     DescribeBySubscription(describer.SqlServer),
	"Microsoft.Sql/servers/databases":                           DescribeBySubscription(describer.SqlDatabase),
	"Microsoft.StorSimple/managers":                             nil,
	"Microsoft.Storage/storageAccounts":                         DescribeBySubscription(describer.StorageAccount),
	"Microsoft.StreamAnalytics/cluster":                         nil,
	"Microsoft.Synapse/workspaces":                              DescribeBySubscription(describer.SynapseWorkspace),
	"Microsoft.Synapse/workspaces/sqlPools":                     nil,
	"Microsoft.TimeSeriesInsights/environments":                 nil,
	"Microsoft.Web/serverFarms":                                 nil,
	"Microsoft.Web/sites":                                       DescribeBySubscription(describer.AppServiceFunctionApp),
	"Microsoft.Web/staticSites":                                 DescribeBySubscription(describer.AppServiceWebApp),
	"Microsoft.DataBoxEdge/dataBoxEdgeDevices":                  DescribeBySubscription(describer.DataboxEdgeDevice),
	"Microsoft.HealthcareApis/services":                         DescribeBySubscription(describer.HealthcareService),
	"microsoft.authorization/policyassignments":                 DescribeBySubscription(describer.PolicyAssignment),
	"microsoft.security/pricings":                               DescribeBySubscription(describer.SecurityCenterSubscriptionPricing),
	"Microsoft.StorageCache/caches":                             DescribeBySubscription(describer.HpcCache),
	"microsoft.resources/subscriptions":                         DescribeBySubscription(describer.Subscription),
	"Microsoft.Batch/batchAccounts":                             DescribeBySubscription(describer.BatchAccount),
	"microsoft.insights/guestdiagnosticsettings":                DescribeBySubscription(describer.DiagnosticSetting),
	"microsoft.keyvault/managedhsms":                            DescribeBySubscription(describer.KeyVaultManagedHardwareSecurityModule),
	"microsoft.insights/activitylogalerts":                      DescribeBySubscription(describer.LogAlert),
	"Microsoft.DBforMariaDB/servers":                            DescribeBySubscription(describer.MariadbServer),
	"Microsoft.ClassicNetwork/networkSecurityGroups":            DescribeBySubscription(describer.NetworkSecurityGroup),
	"microsoft.network/networkwatchers":                         DescribeBySubscription(describer.NetworkWatcher),
	"Microsoft.AppPlatform/Spring":                              DescribeBySubscription(describer.SpringCloudService),
	"Microsoft.StreamAnalytics/StreamingJobs":                   DescribeBySubscription(describer.StreamAnalyticsJob),
	"Microsoft.StorageSync/storageSyncServices":                 DescribeBySubscription(describer.StorageSync),
	"Microsoft.Resources/links":                                 DescribeBySubscription(describer.ResourceLink),
	"Microsoft.Authorization/roleDefinitions":                   DescribeBySubscription(describer.RoleDefinition),
	"Microsoft.Security/autoProvisioningSettings":               DescribeBySubscription(describer.SecurityCenterAutoProvisioning),
	"Microsoft.Security/securityContacts":                       DescribeBySubscription(describer.SecurityCenterContact),
	"Microsoft.Security/locations/jitNetworkAccessPolicies":     DescribeBySubscription(describer.SecurityCenterJitNetworkAccessPolicy),
	"Microsoft.Security/settings":                               DescribeBySubscription(describer.SecurityCenterSetting),
	"Microsoft.Storage/storageAccounts/containers":              DescribeBySubscription(describer.StorageContainer),
	"Microsoft.Network/virtualNetworks/subnets":                 DescribeBySubscription(describer.Subnet),
	"Microsoft.Resources/tenants":                               DescribeBySubscription(describer.Tenant),
	"Microsoft.KeyVault/vaults/secrets":                         DescribeBySubscription(describer.KeyVaultSecret),
	"Microsoft.Insights/logProfiles":                            DescribeBySubscription(describer.LogProfile),
	"Microsoft.Resources/subscriptions/locations":               DescribeBySubscription(describer.Location),
	"Microsoft.Resources/users":                                 DescribeADByTenantID(describer.AdUsers),
}

func ListResourceTypes() []string {
	var list []string
	for k := range resourceTypeToDescriber {
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

	hamiltonAuthorizer, err := hamiltonAuth.NewAutorestAuthorizerWrapper(authorizer)
	if err != nil {
		return nil, err
	}

	env, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return nil, err
	}

	resources, err := describe(ctx, authorizer, hamiltonAuthorizer, resourceType, subscriptions, cfg.TenantID)
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
		os.Setenv(env, s)
	}
}

func describe(ctx context.Context, authorizer autorest.Authorizer, hamiltonAuth hamiltonAuth.Authorizer, resourceType string, subscriptions []string, tenantId string) ([]describer.Resource, error) {
	rd, ok := resourceTypeToDescriber[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	if rd == nil {
		rd = describer.GenericResourceGraph{Table: "Resources", Type: resourceType}
	}

	return rd.DescribeResources(ctx, authorizer, hamiltonAuth, subscriptions, tenantId)
}

func DescribeBySubscription(describe func(context.Context, autorest.Authorizer, string) ([]describer.Resource, error)) ResourceDescriber {
	return ResourceDescribeFunc(func(ctx context.Context, authorizer autorest.Authorizer, hamiltonAuth hamiltonAuth.Authorizer, subscriptions []string, tenantId string) ([]describer.Resource, error) {
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
	return ResourceDescribeFunc(func(ctx context.Context, authorizer autorest.Authorizer, hamiltonAuth hamiltonAuth.Authorizer, subscription []string, tenantId string) ([]describer.Resource, error) {
		var values []describer.Resource
		result, err := describe(ctx, hamiltonAuth, tenantId)
		if err != nil {
			return nil, err
		}

		values = append(values, result...)

		return values, nil
	})
}
