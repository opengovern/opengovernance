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
	"gitlab.com/anil94/golang-aws-inventory/pkg/azure/describer.go"
)

type ResourceDescriber interface {
	DescribeResources(context.Context, autorest.Authorizer, []string) ([]describer.Resource, error)
}

type ResourceDescribeFunc func(context.Context, autorest.Authorizer, []string) ([]describer.Resource, error)

func (fn ResourceDescribeFunc) DescribeResources(c context.Context, a autorest.Authorizer, s []string) ([]describer.Resource, error) {
	return fn(c, a, s)
}

var resourceTypeToDescriber = map[string]ResourceDescriber{
	"Microsoft.AnalysisServices/servers":                        nil,
	"Microsoft.ApiManagement/service":                           nil,
	"Microsoft.AppConfiguration/configurationStores":            nil,
	"Microsoft.Authorization/policyDefinitions":                 nil,
	"Microsoft.Automation/automationAccounts":                   nil,
	"Microsoft.Blueprint/blueprints":                            DescribeBySubscription(describer.BlueprintBlueprint),
	"Microsoft.Blueprint/blueprints/artifacts":                  DescribeBySubscription(describer.BlueprintArtifact),
	"Microsoft.Cache/Redis":                                     nil,
	"Microsoft.Cdn/profiles":                                    nil,
	"Microsoft.Cdn/profiles/endpoints":                          nil,
	"Microsoft.CognitiveServices/accounts":                      nil,
	"Microsoft.Compute/availabilitySets":                        nil,
	"Microsoft.Compute/cloudServices":                           nil,
	"Microsoft.Compute/diskEncryptionSets":                      nil,
	"Microsoft.Compute/disks":                                   nil,
	"Microsoft.Compute/galleries":                               nil,
	"Microsoft.Compute/snapshots":                               nil,
	"Microsoft.Compute/virtualMachineScaleSets":                 nil,
	"Microsoft.Compute/virtualMachines":                         nil,
	"Microsoft.ContainerInstance/containerGroups":               nil,
	"Microsoft.ContainerRegistry/registries":                    nil,
	"Microsoft.ContainerService/managedClusters":                nil,
	"Microsoft.DBforMySQL/servers":                              nil,
	"Microsoft.DBforPostgreSQL/servers":                         nil,
	"Microsoft.DataFactory/factories":                           nil,
	"Microsoft.DataLakeAnalytics/accounts":                      nil,
	"Microsoft.DataLakeStore/accounts":                          nil,
	"Microsoft.DataMigration/services":                          nil,
	"Microsoft.Databricks/workspaces":                           nil,
	"Microsoft.Devices/IotHubs":                                 nil,
	"Microsoft.Devices/provisioningServices":                    nil,
	"Microsoft.Devices/provisioningServices/certificates":       DescribeBySubscription(describer.DevicesProvisioningServicesCertificates),
	"Microsoft.DocumentDB/databaseAccounts":                     nil,
	"Microsoft.DocumentDB/databaseAccounts/sqlDatabases":        DescribeBySubscription(describer.DocumentDBDatabaseAccountsSQLDatabase),
	"Microsoft.EventGrid/domains":                               nil,
	"Microsoft.EventGrid/domains/topics":                        DescribeBySubscription(describer.EventGridDomainTopic),
	"Microsoft.EventGrid/topics":                                nil,
	"Microsoft.EventHub/namespaces":                             nil,
	"Microsoft.HDInsight/clusters":                              nil,
	"Microsoft.HybridCompute/machines":                          nil,
	"Microsoft.Insights/actionGroups":                           nil,
	"Microsoft.Insights/components":                             nil,
	"Microsoft.KeyVault/vaults":                                 nil,
	"Microsoft.Kubernetes/connectedClusters":                    nil,
	"Microsoft.Kusto/clusters":                                  nil,
	"Microsoft.Kusto/clusters/databases":                        nil,
	"Microsoft.Logic/integrationAccounts":                       nil,
	"Microsoft.Logic/workflows":                                 nil,
	"Microsoft.MachineLearningServices/workspaces":              nil,
	"Microsoft.ManagedIdentity/userAssignedIdentities":          nil,
	"Microsoft.Management/managementGroups":                     nil,
	"Microsoft.Migrate/assessmentProjects":                      nil,
	"Microsoft.Network/applicationGateways":                     nil,
	"Microsoft.Network/applicationSecurityGroups":               nil,
	"Microsoft.Network/azureFirewalls":                          nil,
	"Microsoft.Network/bastionHosts":                            nil,
	"Microsoft.Network/connections":                             nil,
	"Microsoft.Network/dnsZones":                                nil,
	"Microsoft.Network/expressRouteCircuits":                    nil,
	"Microsoft.Network/firewallPolicies":                        nil,
	"Microsoft.Network/frontDoors":                              nil,
	"Microsoft.Network/frontdoorWebApplicationFirewallPolicies": nil,
	"Microsoft.Network/loadBalancers":                           nil,
	"Microsoft.Network/localNetworkGateways":                    nil,
	"Microsoft.Network/natGateways":                             nil,
	"Microsoft.Network/networkInterfaces":                       nil,
	"Microsoft.Network/networkSecurityGroups":                   nil,
	"Microsoft.Network/networkWatchers":                         nil,
	"Microsoft.Network/privateDnsZones":                         nil,
	"Microsoft.Network/privateLinkServices":                     nil,
	"Microsoft.Network/publicIPAddresses":                       nil,
	"Microsoft.Network/publicIPPrefixes":                        nil,
	"Microsoft.Network/routeFilters":                            nil,
	"Microsoft.Network/routeTables":                             nil,
	"Microsoft.Network/serviceEndpointPolicies":                 nil,
	"Microsoft.Network/trafficManagerProfiles":                  nil,
	"Microsoft.Network/virtualNetworkGateways":                  nil,
	"Microsoft.Network/virtualNetworks":                         nil,
	"Microsoft.Network/virtualWans":                             nil,
	"Microsoft.Network/vpnGateways":                             nil,
	"Microsoft.NotificationHubs/namespaces":                     nil,
	"Microsoft.NotificationHubs/namespaces/notificationHubs":    nil,
	"Microsoft.OperationalInsights/workspaces":                  nil,
	"Microsoft.PowerBIDedicated/capacities":                     nil,
	"Microsoft.Purview/accounts":                                nil,
	"Microsoft.RecoveryServices/vaults":                         nil,
	"Microsoft.Resources/subscriptions/resourceGroups":          describer.GenericResourceGraph{Table: "ResourceContainers", Type: "Microsoft.Resources/subscriptions/resourceGroups"},
	"Microsoft.Search/searchServices":                           nil,
	"Microsoft.ServiceBus/namespaces":                           nil,
	"Microsoft.ServiceBus/namespaces/queues":                    DescribeBySubscription(describer.ServiceBusQueue),
	"Microsoft.ServiceBus/namespaces/topics":                    DescribeBySubscription(describer.ServiceBusTopic),
	"Microsoft.ServiceFabric/clusters":                          nil,
	"Microsoft.SignalRService/SignalR":                          nil,
	"Microsoft.Sql/managedInstances":                            nil,
	"Microsoft.Sql/servers":                                     nil,
	"Microsoft.Sql/servers/databases":                           nil,
	"Microsoft.StorSimple/managers":                             nil,
	"Microsoft.Storage/storageAccounts":                         nil,
	"Microsoft.StreamAnalytics/cluster":                         nil,
	"Microsoft.Synapse/workspaces":                              nil,
	"Microsoft.Synapse/workspaces/sqlPools":                     nil,
	"Microsoft.TimeSeriesInsights/environments":                 nil,
	"Microsoft.Web/serverFarms":                                 nil,
	"Microsoft.Web/sites":                                       nil,
	"Microsoft.Web/staticSites":                                 nil,
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
	ResourceType    string
	SubscriptionIds []string
}

type Resources struct {
	Resources []describer.Resource
	Metadata  ResourceDescriptionMetadata
}

func GetResources(
	ctx context.Context,
	resourceType string,
	subscriptions []string,
	tenantId,
	clientId,
	clientSecret,
	certPath,
	certPass,
	username,
	password,
	azureAuth,
	azureAuthLoc string,
) (*Resources, error) {
	// Create and authorize a ResourceGraph client
	var authorizer autorest.Authorizer
	var err error
	switch v := AuthType(strings.ToUpper(azureAuth)); v {
	case AuthEnv:
		setEnvIfNotEmpty(auth.TenantID, tenantId)
		setEnvIfNotEmpty(auth.ClientID, clientId)
		setEnvIfNotEmpty(auth.ClientSecret, clientSecret)
		setEnvIfNotEmpty(auth.CertificatePath, certPath)
		setEnvIfNotEmpty(auth.CertificatePassword, certPass)
		setEnvIfNotEmpty(auth.Username, username)
		setEnvIfNotEmpty(auth.Password, password)
		authorizer, err = auth.NewAuthorizerFromEnvironment()
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

	resources, err := describe(ctx, authorizer, resourceType, subscriptions)
	if err != nil {
		return nil, err
	}

	output := &Resources{
		Resources: resources,
		Metadata: ResourceDescriptionMetadata{
			ResourceType:    resourceType,
			SubscriptionIds: subscriptions,
		},
	}

	return output, err
}

func setEnvIfNotEmpty(env, s string) {
	if s != "" {
		os.Setenv(env, s)
	}
}

func describe(ctx context.Context, authorizer autorest.Authorizer, resourceType string, subscriptions []string) ([]describer.Resource, error) {
	rd, ok := resourceTypeToDescriber[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	if rd == nil {
		rd = describer.GenericResourceGraph{Table: "Resources", Type: resourceType}
	}

	return rd.DescribeResources(ctx, authorizer, subscriptions)
}

func DescribeBySubscription(describe func(context.Context, autorest.Authorizer, string) ([]describer.Resource, error)) ResourceDescriber {
	return ResourceDescribeFunc(func(ctx context.Context, authorizer autorest.Authorizer, subscriptions []string) ([]describer.Resource, error) {
		values := []describer.Resource{}
		for _, subscription := range subscriptions {
			result, err := describe(ctx, authorizer, subscription)
			if err != nil {
				return nil, err
			}

			values = append(values, result...)
		}

		return values, nil
	})
}
