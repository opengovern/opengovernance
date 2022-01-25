//go:generate go run ../../keibi-es-sdk/gen/main.go --file $GOFILE --output ../../keibi-es-sdk/azure_resources_clients.go --type azure

package model

import (
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/healthcareapis/mgmt/healthcareapis"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/hybridcompute/mgmt/hybridcompute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/links"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/policy"
	"github.com/Azure/azure-sdk-for-go/services/apimanagement/mgmt/2020-12-01/apimanagement"
	"github.com/Azure/azure-sdk-for-go/services/appconfiguration/mgmt/2020-06-01/appconfiguration"
	"github.com/Azure/azure-sdk-for-go/services/appplatform/mgmt/2020-07-01/appplatform"
	"github.com/Azure/azure-sdk-for-go/services/batch/mgmt/2020-09-01/batch"
	"github.com/Azure/azure-sdk-for-go/services/cognitiveservices/mgmt/2021-04-30/cognitiveservices"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-02-01/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/databoxedge/mgmt/2019-07-01/databoxedge"
	"github.com/Azure/azure-sdk-for-go/services/datafactory/mgmt/2018-06-01/datafactory"
	"github.com/Azure/azure-sdk-for-go/services/datalake/analytics/mgmt/2016-11-01/account"
	"github.com/Azure/azure-sdk-for-go/services/datalake/store/mgmt/2016-11-01/account"
	"github.com/Azure/azure-sdk-for-go/services/frontdoor/mgmt/2020-05-01/frontdoor"
	"github.com/Azure/azure-sdk-for-go/services/hdinsight/mgmt/2018-06-01/hdinsight"
	"github.com/Azure/azure-sdk-for-go/services/iothub/mgmt/2020-03-01/devices"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/kusto/mgmt/2021-01-01/kusto"
	"github.com/Azure/azure-sdk-for-go/services/logic/mgmt/2019-05-01/logic"
	"github.com/Azure/azure-sdk-for-go/services/mariadb/mgmt/2020-01-01/mariadb"
	"github.com/Azure/azure-sdk-for-go/services/mysql/mgmt/2020-01-01/mysql"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/azure-sdk-for-go/services/postgresql/mgmt/2020-01-01/postgresql"
	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
	"github.com/Azure/azure-sdk-for-go/services/preview/cosmos-db/mgmt/2020-04-01-preview/documentdb"
	"github.com/Azure/azure-sdk-for-go/services/preview/eventgrid/mgmt/2021-06-01-preview/eventgrid"
	"github.com/Azure/azure-sdk-for-go/services/preview/eventhub/mgmt/2018-01-01-preview/eventhub"
	"github.com/Azure/azure-sdk-for-go/services/preview/keyvault/mgmt/2020-04-01-preview/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/preview/machinelearningservices/mgmt/2020-02-18-preview/machinelearningservices"
	"github.com/Azure/azure-sdk-for-go/services/preview/security/mgmt/v1.0/security"
	"github.com/Azure/azure-sdk-for-go/services/preview/servicebus/mgmt/2021-06-01-preview/servicebus"
	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/v5.0/sql"
	"github.com/Azure/azure-sdk-for-go/services/redis/mgmt/2020-06-01/redis"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-06-01/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/search/mgmt/2020-08-01/search"
	"github.com/Azure/azure-sdk-for-go/services/servicefabric/mgmt/2019-03-01/servicefabric"
	"github.com/Azure/azure-sdk-for-go/services/signalr/mgmt/2020-05-01/signalr"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/azure-sdk-for-go/services/storagecache/mgmt/2021-05-01/storagecache"
	"github.com/Azure/azure-sdk-for-go/services/storagesync/mgmt/2020-03-01/storagesync"
	"github.com/Azure/azure-sdk-for-go/services/streamanalytics/mgmt/2016-03-01/streamanalytics"
	"github.com/Azure/azure-sdk-for-go/services/synapse/mgmt/2021-03-01/synapse"
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

//  =================== network ==================

//index:Microsoft_Network_applicationGateways
//getfilter:name=
//getfilter:resource_group=
type ApplicationGatewayDescription struct {
	obj0 network.TypeName

	obj1 network.TypeName
}

//  =================== batch ==================

//index:Microsoft_Batch_batchAccounts
//getfilter:name=
//getfilter:resource_group=
type BatchAccountDescription struct {
	obj0 batch.TypeName

	obj1 batch.TypeName
}

//  =================== cognitiveservices ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type CognitiveAccountDescription struct {
	obj0 cognitiveservices.TypeName

	obj1 cognitiveservices.TypeName
}

//  =================== compute ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type ComputeVirtualMachineDescription struct {
	obj0 compute.TypeName

	obj1 compute.TypeName

	obj2 compute.TypeName

	obj3 compute.TypeName
	obj4 compute.TypeName
	obj5 compute.TypeName
}

//  =================== containerregistry ==================

//index:Microsoft_ContainerRegistry_registries
//getfilter:name=
//getfilter:resource_group=
type ContainerRegistryDescription struct {
	obj0 containerregistry.TypeName

	obj1 containerregistry.TypeName

	obj2 containerregistry.TypeName
}

//  =================== containerregistry ==================

//index:Microsoft_ContainerRegistry_registries
//getfilter:name=
//getfilter:resource_group=
type ContainerRegistryDescription struct {
	obj0 containerregistry.TypeName

	obj1 containerregistry.TypeName

	obj2 containerregistry.TypeName
}

//  =================== containerregistry ==================

//index:Microsoft_ContainerRegistry_registries
//getfilter:name=
//getfilter:resource_group=
type ContainerRegistryDescription struct {
	obj0 containerregistry.TypeName

	obj1 containerregistry.TypeName

	obj2 containerregistry.TypeName
}

//  =================== documentdb ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type CosmosdbAccountDescription struct {
	obj0 documentdb.TypeName
}

//  =================== datafactory ==================

//index:Microsoft_DataFactory_dataFactories
//getfilter:name=
//getfilter:resource_group=
type DataFactoryDescription struct {
	obj0 datafactory.TypeName

	obj1 datafactory.TypeName
}

//  =================== account ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type DataLakeAnalyticsAccountDescription struct {
	obj0 account.TypeName

	obj1 account.TypeName

	obj2 account.TypeName
}

//  =================== account ==================

//index:Microsoft_DataLakeStore_accounts
//getfilter:name=
//getfilter:resource_group=
type DataLakeStoreDescription struct {
	obj0 account.TypeName

	obj1 account.TypeName

	obj2 account.TypeName
}

//  =================== insights ==================

//index:microsoft_insights_guestdiagnosticsettings
//getfilter:name=
//getfilter:resource_group=
type DiagnosticSettingDescription struct {
	obj0 insights.TypeName
}

//  =================== eventgrid ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type EventgridDomainDescription struct {
	obj0 eventgrid.TypeName

	obj1 eventgrid.TypeName
}

//  =================== eventgrid ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type EventgridDomainDescription struct {
	obj0 eventgrid.TypeName

	obj1 eventgrid.TypeName
}

//  =================== eventgrid ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type EventgridTopicDescription struct {
	obj0 eventgrid.TypeName

	obj1 eventgrid.TypeName
}

//  =================== eventhub ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type EventhubNamespaceDescription struct {
	obj0 eventhub.TypeName

	obj1 eventhub.TypeName

	obj2 eventhub.TypeName

	obj3 eventhub.TypeName
}

//  =================== eventhub ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type EventhubNamespaceDescription struct {
	obj0 eventhub.TypeName

	obj1 eventhub.TypeName

	obj2 eventhub.TypeName

	obj3 eventhub.TypeName
}

//  =================== eventhub ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type EventhubNamespaceDescription struct {
	obj0 eventhub.TypeName

	obj1 eventhub.TypeName

	obj2 eventhub.TypeName

	obj3 eventhub.TypeName
}

//  =================== frontdoor ==================

//index:Microsoft_Network_frontdoors
//getfilter:name=
//getfilter:resource_group=
type FrontdoorDescription struct {
	obj0 frontdoor.TypeName

	obj1 frontdoor.TypeName
}

//  =================== hdinsight ==================

//index:Microsoft_HDInsight_clusterpools
//getfilter:name=
//getfilter:resource_group=
type HdinsightClusterDescription struct {
	obj0 hdinsight.TypeName

	obj1 hdinsight.TypeName
}

//  =================== hybridcompute ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type HybridComputeMachineDescription struct {
	obj0 hybridcompute.TypeName

	obj1 hybridcompute.TypeName
}

//  =================== devices ==================

//index:microsoft_devices_elasticpools_iothubtenants
//getfilter:name=
//getfilter:resource_group=
type IothubDescription struct {
	obj0 devices.TypeName

	obj1 devices.TypeName
}

//  =================== keyvault ==================

//index:microsoft_keyvault_hsmpools
//getfilter:name=
//getfilter:resource_group=
type KeyVaultDescription struct {
	obj0 keyvault.TypeName

	obj1 keyvault.TypeName

	obj2 keyvault.TypeName
}

//  =================== keyvault ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type KeyVaultManagedHardwareSecurityModuleDescription struct {
	obj0 keyvault.TypeName

	obj1 keyvault.TypeName
}

//  =================== keyvault ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type KeyVaultManagedHardwareSecurityModuleDescription struct {
	obj0 keyvault.TypeName

	obj1 keyvault.TypeName
}

//  =================== keyvault ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type KeyVaultManagedHardwareSecurityModuleDescription struct {
	obj0 keyvault.TypeName

	obj1 keyvault.TypeName
}

//  =================== secret ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type KeyVaultSecretDescription struct {
	obj0 secret.TypeName

	obj1 secret.TypeName

	obj2 secret.TypeName
}

//  =================== kusto ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type KustoClusterDescription struct {
	obj0 kusto.TypeName
}

//  =================== kusto ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type KustoClusterDescription struct {
	obj0 kusto.TypeName
}

//  =================== kusto ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type KustoClusterDescription struct {
	obj0 kusto.TypeName
}

//  =================== sub ==================

//index:microsoft_desktopvirtualization_hostpools_sessionhosts
//getfilter:name=
//getfilter:resource_group=
type LocationDescription struct {
	obj0 sub.TypeName
}

//  =================== insights ==================

//index:microsoft_insights_activitylogalerts
//getfilter:name=
//getfilter:resource_group=
type LogAlertDescription struct {
	obj0 insights.TypeName
}

//  =================== insights ==================

//index:microsoft_insights_activitylogalerts
//getfilter:name=
//getfilter:resource_group=
type LogAlertDescription struct {
	obj0 insights.TypeName
}

//  =================== insights ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type LogProfileDescription struct {
	obj0 insights.TypeName
}

//  =================== insights ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type LogProfileDescription struct {
	obj0 insights.TypeName
}

//  =================== insights ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type LogProfileDescription struct {
	obj0 insights.TypeName
}

//  =================== logic ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type LogicAppWorkflowDescription struct {
	obj0 logic.TypeName

	obj1 logic.TypeName
}

//  =================== logic ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type LogicAppWorkflowDescription struct {
	obj0 logic.TypeName

	obj1 logic.TypeName
}

//  =================== logic ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type LogicAppWorkflowDescription struct {
	obj0 logic.TypeName

	obj1 logic.TypeName
}

//  =================== machinelearningservices ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type MachineLearningWorkspaceDescription struct {
	obj0 machinelearningservices.TypeName

	obj1 machinelearningservices.TypeName
}

//  =================== mariadb ==================

//index:Microsoft_DBforMariaDB_servers
//getfilter:name=
//getfilter:resource_group=
type MariadbServerDescription struct {
	obj0 mariadb.TypeName
}

//  =================== sql ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type MssqlManagedInstanceDescription struct {
	obj0 sql.TypeName

	obj1 sql.TypeName

	obj2 sql.TypeName

	obj3 sql.TypeName
}

//  =================== mysql ==================

//index:Microsoft_DBforMySQL_servers
//getfilter:name=
//getfilter:resource_group=
type MysqlServerDescription struct {
	obj0 mysql.TypeName

	obj1 mysql.TypeName

	obj2 mysql.TypeName
}

//  =================== network ==================

//index:Microsoft_ClassicNetwork_networkSecurityGroups
//getfilter:name=
//getfilter:resource_group=
type NetworkSecurityGroupDescription struct {
	obj0 network.TypeName

	obj1 network.TypeName
}

//  =================== network ==================

//index:microsoft_network_networkwatchers
//getfilter:name=
//getfilter:resource_group=
type NetworkWatcherDescription struct {
	obj0 network.TypeName
}

//  =================== network ==================

//index:microsoft_network_networkwatchers
//getfilter:name=
//getfilter:resource_group=
type NetworkWatcherDescription struct {
	obj0 network.TypeName
}

//  =================== network ==================

//index:microsoft_network_networkwatchers
//getfilter:name=
//getfilter:resource_group=
type NetworkWatcherDescription struct {
	obj0 network.TypeName
}

//  =================== postgresql ==================

//index:Microsoft_DBforPostgreSQL_serverGroups
//getfilter:name=
//getfilter:resource_group=
type PostgresqlServerDescription struct {
	obj0 postgresql.TypeName

	obj1 postgresql.TypeName

	obj2 postgresql.TypeName

	obj3 postgresql.TypeName

	obj4 postgresql.TypeName
}

//  =================== postgresql ==================

//index:Microsoft_DBforPostgreSQL_serverGroups
//getfilter:name=
//getfilter:resource_group=
type PostgresqlServerDescription struct {
	obj0 postgresql.TypeName

	obj1 postgresql.TypeName

	obj2 postgresql.TypeName

	obj3 postgresql.TypeName

	obj4 postgresql.TypeName
}

//  =================== postgresql ==================

//index:Microsoft_DBforPostgreSQL_serverGroups
//getfilter:name=
//getfilter:resource_group=
type PostgresqlServerDescription struct {
	obj0 postgresql.TypeName

	obj1 postgresql.TypeName

	obj2 postgresql.TypeName

	obj3 postgresql.TypeName

	obj4 postgresql.TypeName
}

//  =================== search ==================

//index:Microsoft_Search_searchServices
//getfilter:name=
//getfilter:resource_group=
type SearchServiceDescription struct {
	obj0 search.TypeName

	obj1 search.TypeName
	obj2 search.TypeName
}

//  =================== servicefabric ==================

//index:Microsoft_ServiceFabric_clusters
//getfilter:name=
//getfilter:resource_group=
type ServiceFabricClusterDescription struct {
	obj0 servicefabric.TypeName
}

//  =================== servicebus ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type ServicebusNamespaceDescription struct {
	obj0 servicebus.TypeName

	obj1 servicebus.TypeName

	obj2 servicebus.TypeName

	obj3 servicebus.TypeName
}

//  =================== signalr ==================

//index:Microsoft_SignalRService_SignalR
//getfilter:name=
//getfilter:resource_group=
type SignalrServiceDescription struct {
	obj0 signalr.TypeName

	obj1 signalr.TypeName
}

//  =================== appplatform ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type SpringCloudServiceDescription struct {
	obj0 appplatform.TypeName

	obj1 appplatform.TypeName
}

//  =================== sql ==================

//index:microsoft_synapse_workspaces_sqldatabases
//getfilter:name=
//getfilter:resource_group=
type SqlDatabaseDescription struct {
	obj0 sql.TypeName

	obj1 sql.TypeName

	obj2 sql.TypeName

	obj3 sql.TypeName

	obj4 sql.TypeName

	obj5 sql.TypeName
}

//  =================== sqlv3 ==================

//index:Microsoft_AzureArcData_sqlServerInstances
//getfilter:name=
//getfilter:resource_group=
type SqlServerDescription struct {
	obj0 sqlv3.TypeName

	obj1 sqlv3.TypeName

	obj2 sqlv3.TypeName

	obj3 sqlv3.TypeName

	obj4 sqlv3.TypeName

	obj5 sqlv3.TypeName

	obj6 sqlv3.TypeName

	obj7 sqlv3.TypeName

	obj8 sqlv3.TypeName
}

//  =================== sqlv3 ==================

//index:Microsoft_AzureArcData_sqlServerInstances
//getfilter:name=
//getfilter:resource_group=
type SqlServerDescription struct {
	obj0 sqlv3.TypeName

	obj1 sqlv3.TypeName

	obj2 sqlv3.TypeName

	obj3 sqlv3.TypeName

	obj4 sqlv3.TypeName

	obj5 sqlv3.TypeName

	obj6 sqlv3.TypeName

	obj7 sqlv3.TypeName

	obj8 sqlv3.TypeName
}

//  =================== sqlv3 ==================

//index:Microsoft_AzureArcData_sqlServerInstances
//getfilter:name=
//getfilter:resource_group=
type SqlServerDescription struct {
	obj0 sqlv3.TypeName

	obj1 sqlv3.TypeName

	obj2 sqlv3.TypeName

	obj3 sqlv3.TypeName

	obj4 sqlv3.TypeName

	obj5 sqlv3.TypeName

	obj6 sqlv3.TypeName

	obj7 sqlv3.TypeName

	obj8 sqlv3.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storagesync ==================

//index:Microsoft_StorageSync_storageSyncServices
//getfilter:name=
//getfilter:resource_group=
type StorageSyncDescription struct {
	obj0 storagesync.TypeName
}

//  =================== storagesync ==================

//index:Microsoft_StorageSync_storageSyncServices
//getfilter:name=
//getfilter:resource_group=
type StorageSyncDescription struct {
	obj0 storagesync.TypeName
}

//  =================== storagesync ==================

//index:Microsoft_StorageSync_storageSyncServices
//getfilter:name=
//getfilter:resource_group=
type StorageSyncDescription struct {
	obj0 storagesync.TypeName
}

//  =================== storagesync ==================

//index:Microsoft_StorageSync_storageSyncServices
//getfilter:name=
//getfilter:resource_group=
type StorageSyncDescription struct {
	obj0 storagesync.TypeName
}

//  =================== storagesync ==================

//index:Microsoft_StorageSync_storageSyncServices
//getfilter:name=
//getfilter:resource_group=
type StorageSyncDescription struct {
	obj0 storagesync.TypeName
}

//  =================== storagesync ==================

//index:Microsoft_StorageSync_storageSyncServices
//getfilter:name=
//getfilter:resource_group=
type StorageSyncDescription struct {
	obj0 storagesync.TypeName
}

//  =================== storagesync ==================

//index:Microsoft_StorageSync_storageSyncServices
//getfilter:name=
//getfilter:resource_group=
type StorageSyncDescription struct {
	obj0 storagesync.TypeName
}

//  =================== storagesync ==================

//index:Microsoft_StorageSync_storageSyncServices
//getfilter:name=
//getfilter:resource_group=
type StorageSyncDescription struct {
	obj0 storagesync.TypeName
}

//  =================== streamanalytics ==================

//index:Microsoft_StreamAnalytics_StreamingJobs
//getfilter:name=
//getfilter:resource_group=
type StreamAnalyticsJobDescription struct {
	obj0 streamanalytics.TypeName

	obj1 streamanalytics.TypeName
}

//  =================== synapse ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type SynapseWorkspaceDescription struct {
	obj0 synapse.TypeName

	obj1 synapse.TypeName

	obj2 synapse.TypeName
}

//  =================== datafactory ==================

//index:Microsoft_DataFactory_dataFactories
//getfilter:name=
//getfilter:resource_group=
type DataFactoryDescription struct {
	obj0 datafactory.TypeName

	obj1 datafactory.TypeName
}

//  =================== sub ==================

//index:microsoft_desktopvirtualization_hostpools_sessionhosts
//getfilter:name=
//getfilter:resource_group=
type LocationDescription struct {
	obj0 sub.TypeName
}

//  =================== sub ==================

//index:microsoft_desktopvirtualization_hostpools_sessionhosts
//getfilter:name=
//getfilter:resource_group=
type LocationDescription struct {
	obj0 sub.TypeName
}

//  =================== sub ==================

//index:microsoft_desktopvirtualization_hostpools_sessionhosts
//getfilter:name=
//getfilter:resource_group=
type LocationDescription struct {
	obj0 sub.TypeName
}

//  =================== sql ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type MssqlManagedInstanceDescription struct {
	obj0 sql.TypeName

	obj1 sql.TypeName

	obj2 sql.TypeName

	obj3 sql.TypeName
}

//  =================== sql ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type MssqlManagedInstanceDescription struct {
	obj0 sql.TypeName

	obj1 sql.TypeName

	obj2 sql.TypeName

	obj3 sql.TypeName
}

//  =================== sql ==================

//index:
//getfilter:name=
//getfilter:resource_group=
type MssqlManagedInstanceDescription struct {
	obj0 sql.TypeName

	obj1 sql.TypeName

	obj2 sql.TypeName

	obj3 sql.TypeName
}

//  =================== mysql ==================

//index:Microsoft_DBforMySQL_servers
//getfilter:name=
//getfilter:resource_group=
type MysqlServerDescription struct {
	obj0 mysql.TypeName

	obj1 mysql.TypeName

	obj2 mysql.TypeName
}

//  =================== mysql ==================

//index:Microsoft_DBforMySQL_servers
//getfilter:name=
//getfilter:resource_group=
type MysqlServerDescription struct {
	obj0 mysql.TypeName

	obj1 mysql.TypeName

	obj2 mysql.TypeName
}

//  =================== mysql ==================

//index:Microsoft_DBforMySQL_servers
//getfilter:name=
//getfilter:resource_group=
type MysqlServerDescription struct {
	obj0 mysql.TypeName

	obj1 mysql.TypeName

	obj2 mysql.TypeName
}

//  =================== mysql ==================

//index:Microsoft_DBforMySQL_servers
//getfilter:name=
//getfilter:resource_group=
type MysqlServerDescription struct {
	obj0 mysql.TypeName

	obj1 mysql.TypeName

	obj2 mysql.TypeName
}

//  =================== postgresql ==================

//index:Microsoft_DBforPostgreSQL_serverGroups
//getfilter:name=
//getfilter:resource_group=
type PostgresqlServerDescription struct {
	obj0 postgresql.TypeName

	obj1 postgresql.TypeName

	obj2 postgresql.TypeName

	obj3 postgresql.TypeName

	obj4 postgresql.TypeName
}

//  =================== postgresql ==================

//index:Microsoft_DBforPostgreSQL_serverGroups
//getfilter:name=
//getfilter:resource_group=
type PostgresqlServerDescription struct {
	obj0 postgresql.TypeName

	obj1 postgresql.TypeName

	obj2 postgresql.TypeName

	obj3 postgresql.TypeName

	obj4 postgresql.TypeName
}

//  =================== postgresql ==================

//index:Microsoft_DBforPostgreSQL_serverGroups
//getfilter:name=
//getfilter:resource_group=
type PostgresqlServerDescription struct {
	obj0 postgresql.TypeName

	obj1 postgresql.TypeName

	obj2 postgresql.TypeName

	obj3 postgresql.TypeName

	obj4 postgresql.TypeName
}

//  =================== postgresql ==================

//index:Microsoft_DBforPostgreSQL_serverGroups
//getfilter:name=
//getfilter:resource_group=
type PostgresqlServerDescription struct {
	obj0 postgresql.TypeName

	obj1 postgresql.TypeName

	obj2 postgresql.TypeName

	obj3 postgresql.TypeName

	obj4 postgresql.TypeName
}

//  =================== postgresql ==================

//index:Microsoft_DBforPostgreSQL_serverGroups
//getfilter:name=
//getfilter:resource_group=
type PostgresqlServerDescription struct {
	obj0 postgresql.TypeName

	obj1 postgresql.TypeName

	obj2 postgresql.TypeName

	obj3 postgresql.TypeName

	obj4 postgresql.TypeName
}

//  =================== servicefabric ==================

//index:Microsoft_ServiceFabric_clusters
//getfilter:name=
//getfilter:resource_group=
type ServiceFabricClusterDescription struct {
	obj0 servicefabric.TypeName
}

//  =================== servicefabric ==================

//index:Microsoft_ServiceFabric_clusters
//getfilter:name=
//getfilter:resource_group=
type ServiceFabricClusterDescription struct {
	obj0 servicefabric.TypeName
}

//  =================== servicefabric ==================

//index:Microsoft_ServiceFabric_clusters
//getfilter:name=
//getfilter:resource_group=
type ServiceFabricClusterDescription struct {
	obj0 servicefabric.TypeName
}

//  =================== servicefabric ==================

//index:Microsoft_ServiceFabric_clusters
//getfilter:name=
//getfilter:resource_group=
type ServiceFabricClusterDescription struct {
	obj0 servicefabric.TypeName
}

//  =================== servicefabric ==================

//index:Microsoft_ServiceFabric_clusters
//getfilter:name=
//getfilter:resource_group=
type ServiceFabricClusterDescription struct {
	obj0 servicefabric.TypeName
}

//  =================== servicefabric ==================

//index:Microsoft_ServiceFabric_clusters
//getfilter:name=
//getfilter:resource_group=
type ServiceFabricClusterDescription struct {
	obj0 servicefabric.TypeName
}

//  =================== sql ==================

//index:microsoft_synapse_workspaces_sqldatabases
//getfilter:name=
//getfilter:resource_group=
type SqlDatabaseDescription struct {
	obj0 sql.TypeName

	obj1 sql.TypeName

	obj2 sql.TypeName

	obj3 sql.TypeName

	obj4 sql.TypeName

	obj5 sql.TypeName
}

//  =================== sql ==================

//index:microsoft_synapse_workspaces_sqldatabases
//getfilter:name=
//getfilter:resource_group=
type SqlDatabaseDescription struct {
	obj0 sql.TypeName

	obj1 sql.TypeName

	obj2 sql.TypeName

	obj3 sql.TypeName

	obj4 sql.TypeName

	obj5 sql.TypeName
}

//  =================== sql ==================

//index:microsoft_synapse_workspaces_sqldatabases
//getfilter:name=
//getfilter:resource_group=
type SqlDatabaseDescription struct {
	obj0 sql.TypeName

	obj1 sql.TypeName

	obj2 sql.TypeName

	obj3 sql.TypeName

	obj4 sql.TypeName

	obj5 sql.TypeName
}

//  =================== sql ==================

//index:microsoft_synapse_workspaces_sqldatabases
//getfilter:name=
//getfilter:resource_group=
type SqlDatabaseDescription struct {
	obj0 sql.TypeName

	obj1 sql.TypeName

	obj2 sql.TypeName

	obj3 sql.TypeName

	obj4 sql.TypeName

	obj5 sql.TypeName
}

//  =================== sqlv3 ==================

//index:Microsoft_AzureArcData_sqlServerInstances
//getfilter:name=
//getfilter:resource_group=
type SqlServerDescription struct {
	obj0 sqlv3.TypeName

	obj1 sqlv3.TypeName

	obj2 sqlv3.TypeName

	obj3 sqlv3.TypeName

	obj4 sqlv3.TypeName

	obj5 sqlv3.TypeName

	obj6 sqlv3.TypeName

	obj7 sqlv3.TypeName

	obj8 sqlv3.TypeName
}

//  =================== sqlv3 ==================

//index:Microsoft_AzureArcData_sqlServerInstances
//getfilter:name=
//getfilter:resource_group=
type SqlServerDescription struct {
	obj0 sqlv3.TypeName

	obj1 sqlv3.TypeName

	obj2 sqlv3.TypeName

	obj3 sqlv3.TypeName

	obj4 sqlv3.TypeName

	obj5 sqlv3.TypeName

	obj6 sqlv3.TypeName

	obj7 sqlv3.TypeName

	obj8 sqlv3.TypeName
}

//  =================== sqlv3 ==================

//index:Microsoft_AzureArcData_sqlServerInstances
//getfilter:name=
//getfilter:resource_group=
type SqlServerDescription struct {
	obj0 sqlv3.TypeName

	obj1 sqlv3.TypeName

	obj2 sqlv3.TypeName

	obj3 sqlv3.TypeName

	obj4 sqlv3.TypeName

	obj5 sqlv3.TypeName

	obj6 sqlv3.TypeName

	obj7 sqlv3.TypeName

	obj8 sqlv3.TypeName
}

//  =================== sqlv3 ==================

//index:Microsoft_AzureArcData_sqlServerInstances
//getfilter:name=
//getfilter:resource_group=
type SqlServerDescription struct {
	obj0 sqlv3.TypeName

	obj1 sqlv3.TypeName

	obj2 sqlv3.TypeName

	obj3 sqlv3.TypeName

	obj4 sqlv3.TypeName

	obj5 sqlv3.TypeName

	obj6 sqlv3.TypeName

	obj7 sqlv3.TypeName

	obj8 sqlv3.TypeName
}

//  =================== sqlv3 ==================

//index:Microsoft_AzureArcData_sqlServerInstances
//getfilter:name=
//getfilter:resource_group=
type SqlServerDescription struct {
	obj0 sqlv3.TypeName

	obj1 sqlv3.TypeName

	obj2 sqlv3.TypeName

	obj3 sqlv3.TypeName

	obj4 sqlv3.TypeName

	obj5 sqlv3.TypeName

	obj6 sqlv3.TypeName

	obj7 sqlv3.TypeName

	obj8 sqlv3.TypeName
}

//  =================== sqlv3 ==================

//index:Microsoft_AzureArcData_sqlServerInstances
//getfilter:name=
//getfilter:resource_group=
type SqlServerDescription struct {
	obj0 sqlv3.TypeName

	obj1 sqlv3.TypeName

	obj2 sqlv3.TypeName

	obj3 sqlv3.TypeName

	obj4 sqlv3.TypeName

	obj5 sqlv3.TypeName

	obj6 sqlv3.TypeName

	obj7 sqlv3.TypeName

	obj8 sqlv3.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storage ==================

//index:Microsoft_ClassicStorage_StorageAccounts
//getfilter:name=
//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//getfilter:resource_group=
type StorageAccountDescription struct {
	obj0 storage.TypeName

	obj1 storage.TypeName

	obj2 storage.TypeName

	obj3 storage.TypeName

	obj4 storage.TypeName

	obj5 storage.TypeName

	obj6 storage.TypeName
}

//  =================== storagesync ==================

//index:Microsoft_StorageSync_storageSyncServices
//getfilter:name=
//getfilter:resource_group=
type StorageSyncDescription struct {
	obj0 storagesync.TypeName
}

//  =================== storagesync ==================

//index:Microsoft_StorageSync_storageSyncServices
//getfilter:name=
//getfilter:resource_group=
type StorageSyncDescription struct {
	obj0 storagesync.TypeName
}
{
	
		obj0 storagesync.TypeName 
	
}

