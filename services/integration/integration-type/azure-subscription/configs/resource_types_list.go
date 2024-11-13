package configs

var TablesToResourceTypes = map[string]string{
	"azure_app_containerapps":                                     "Microsoft.App/containerApps",
	"azure_blueprint_blueprints":                                  "Microsoft.Blueprint/blueprints",
	"azure_cdn_profiles":                                          "Microsoft.Cdn/profiles",
	"azure_compute_cloudservices":                                 "Microsoft.Compute/cloudServices",
	"azure_container_group":                                       "Microsoft.ContainerInstance/containerGroups",
	"azure_datamigration_services":                                "Microsoft.DataMigration/services",
	"azure_dataprotection_backupvaults":                           "Microsoft.DataProtection/backupVaults",
	"azure_data_protection_backup_job":                            "Microsoft.DataProtection/backupJobs",
	"azure_dataprotection_backuppolicies":                         "Microsoft.DataProtection/backupVaults/backupPolicies",
	"azure_logic_integrationaccounts":                             "Microsoft.Logic/integrationAccounts",
	"azure_bastion_host":                                          "Microsoft.Network/bastionHosts",
	"azure_network_connections":                                   "Microsoft.Network/connections",
	"azure_firewall_policy":                                       "Microsoft.Network/firewallPolicies",
	"azure_network_localnetworkgateways":                          "Microsoft.Network/localNetworkGateways",
	"azure_network_privatelinkservices":                           "Microsoft.Network/privateLinkServices",
	"azure_network_publicipprefixes":                              "Microsoft.Network/publicIPPrefixes",
	"azure_network_virtualhubs":                                   "Microsoft.Network/virtualHubs",
	"azure_network_virtualwans":                                   "Microsoft.Network/virtualWans",
	"azure_network_vpngateways":                                   "Microsoft.Network/vpnGateways",
	"azure_network_vpnconnections":                                "Microsoft.Network/vpnGateways/vpnConnections",
	"azure_network_vpnsites":                                      "Microsoft.Network/vpnSites",
	"azure_operationalinsights_workspaces":                        "Microsoft.OperationalInsights/workspaces",
	"azure_streamanalytics_cluster":                               "Microsoft.StreamAnalytics/cluster",
	"azure_timeseriesinsights_environments":                       "Microsoft.TimeSeriesInsights/environments",
	"azure_virtualmachineimages_imagetemplates":                   "Microsoft.VirtualMachineImages/imageTemplates",
	"azure_web_serverfarms":                                       "Microsoft.Web/serverFarms",
	"azure_compute_virtual_machine_scale_set_vm":                  "Microsoft.Compute/virtualMachineScaleSets/virtualMachines",
	"azure_automation_account":                                    "Microsoft.Automation/automationAccounts",
	"azure_automation_variable":                                   "Microsoft.Automation/automationAccounts/variables",
	"azure_dns_zone":                                              "Microsoft.Network/dnsZones",
	"azure_databricks_workspaces":                                 "Microsoft.Databricks/workspaces",
	"azure_private_dns_zone":                                      "Microsoft.Network/privateDnsZones",
	"azure_network_privateendpoints":                              "Microsoft.Network/privateEndpoints",
	"azure_network_watcher":                                       "Microsoft.Network/networkWatchers",
	"azure_resource_group":                                        "Microsoft.Resources/subscriptions/resourceGroups",
	"azure_app_service_web_app":                                   "Microsoft.Web/staticSites",
	"azure_app_service_web_app_slot":                              "Microsoft.Web/sites/slots",
	"azure_cognitive_account":                                     "Microsoft.CognitiveServices/accounts",
	"azure_mssql_managed_instance":                                "Microsoft.Sql/managedInstances",
	"azure_sql_virtualclusters":                                   "Microsoft.Sql/virtualclusters",
	"azure_sql_managedinstancesdatabases":                         "Microsoft.Sql/managedInstances/databases",
	"azure_sql_database":                                          "Microsoft.Sql/servers/databases",
	"azure_storage_share_file":                                    "Microsoft.Storage/storageAccounts/largeFileSharesState",
	"azure_postgresql_server":                                     "Microsoft.DBforPostgreSQL/servers",
	"azure_dbforpostgresql_flexibleservers":                       "Microsoft.DBforPostgreSQL/flexibleservers",
	"azure_analysisservices_servers":                              "Microsoft.AnalysisServices/servers",
	"azure_security_center_subscription_pricing":                  "Microsoft.Security/pricings",
	"azure_diagnostic_setting":                                    "Microsoft.Insights/guestDiagnosticSettings",
	"azure_autoscale_setting":                                     "Microsoft.Insights/autoscaleSettings",
	"azure_app_service_environment":                               "Microsoft.Web/hostingEnvironments",
	"azure_redis_cache":                                           "Microsoft.Cache/redis",
	"azure_container_registry":                                    "Microsoft.ContainerRegistry/registries",
	"azure_data_factory_pipeline":                                 "Microsoft.DataFactory/factories/pipelines",
	"azure_compute_resource_sku":                                  "Microsoft.Compute/resourceSku",
	"azure_express_route_circuit":                                 "Microsoft.Network/expressRouteCircuits",
	"azure_management_group":                                      "Microsoft.Management/managementgroups",
	"azure_mssql_virtual_machine":                                 "microsoft.SqlVirtualMachine/SqlVirtualMachines",
	"azure_sql_virtualmachinegroups":                              "Microsoft.SqlVirtualMachine/SqlVirtualMachineGroups",
	"azure_storage_table_service":                                 "Microsoft.Storage/storageAccounts/tableServices",
	"azure_synapse_workspace":                                     "Microsoft.Synapse/workspaces",
	"azure_synapse_workspacesbigdatapools":                        "Microsoft.Synapse/workspaces/bigdatapools",
	"azure_synapse_workspacessqlpools":                            "Microsoft.Synapse/workspaces/sqlpools",
	"azure_stream_analytics_job":                                  "Microsoft.StreamAnalytics/streamingJobs",
	"azure_costmanagement_costbysubscription":                     "Microsoft.CostManagement/CostBySubscription",
	"azure_kubernetes_cluster":                                    "Microsoft.ContainerService/managedClusters",
	"azure_kubernetes_service_version":                            "Microsoft.ContainerService/serviceVersions",
	"azure_data_factory":                                          "Microsoft.DataFactory/factories",
	"azure_sql_server":                                            "Microsoft.Sql/servers",
	"azure_sql_serversjobagents":                                  "Microsoft.Sql/servers/jobagents",
	"azure_security_center_auto_provisioning":                     "Microsoft.Security/autoProvisioningSettings",
	"azure_log_profile":                                           "Microsoft.Insights/logProfiles",
	"azure_databox_edge_device":                                   "Microsoft.DataBoxEdge/dataBoxEdgeDevices",
	"azure_lb":                                                    "Microsoft.Network/loadBalancers",
	"azure_firewall":                                              "Microsoft.Network/azureFirewalls",
	"azure_management_lock":                                       "Microsoft.Management/locks",
	"azure_compute_virtual_machine_scale_set_network_interface":   "Microsoft.Compute/virtualMachineScaleSets/networkInterfaces",
	"azure_frontdoor":                                             "Microsoft.Network/frontDoors",
	"azure_policy_assignment":                                     "Microsoft.Authorization/policyAssignments",
	"azure_user_effective_access":                                 "Microsoft.Authorization/userEffectiveAccess",
	"azure_search_service":                                        "Microsoft.Search/searchServices",
	"azure_security_center_setting":                               "Microsoft.Security/settings",
	"azure_recovery_services_vault":                               "Microsoft.RecoveryServices/vaults",
	"azure_recovery_services_backup_job":                          "Microsoft.RecoveryServices/vaults/backupJobs",
	"azure_recovery_services_backup_policy":                       "Microsoft.RecoveryServices/vaults/backupPolicies",
	"azure_recovery_services_backup_item":                         "Microsoft.RecoveryServices/vaults/backupItems",
	"azure_compute_disk_encryption_set":                           "Microsoft.Compute/diskEncryptionSets",
	"azure_cosmosdb_sql_database":                                 "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
	"azure_eventgrid_topic":                                       "Microsoft.EventGrid/topics",
	"azure_eventhub_namespace":                                    "Microsoft.EventHub/namespaces",
	"azure_eventhub_namespaceeventhubs":                           "Microsoft.EventHub/namespaces/eventHubs",
	"azure_machine_learning_workspace":                            "Microsoft.MachineLearningServices/workspaces",
	"azure_dashboard_grafana":                                     "Microsoft.Dashboard/grafana",
	"azure_desktopvirtualization_workspace":                       "Microsoft.DesktopVirtualization/workspaces",
	"azure_trafficmanager_profile":                                "Microsoft.Network/trafficManagerProfiles",
	"azure_network_dnsresolver":                                   "Microsoft.Network/dnsResolvers",
	"azure_costmanagement_costbyresourcetype":                     "Microsoft.CostManagement/CostByResourceType",
	"azure_network_interface":                                     "Microsoft.Network/networkInterfaces",
	"azure_public_ip":                                             "Microsoft.Network/publicIPAddresses",
	"azure_healthcare_service":                                    "Microsoft.HealthcareApis/services",
	"azure_servicebus_namespace":                                  "Microsoft.ServiceBus/namespaces",
	"azure_app_service_function_app":                              "Microsoft.Web/sites",
	"azure_compute_availability_set":                              "Microsoft.Compute/availabilitySets",
	"azure_virtual_network":                                       "Microsoft.Network/virtualNetworks",
	"azure_security_center_contact":                               "Microsoft.Security/securityContacts",
	"azure_compute_disk_metric_write_ops":                         "Microsoft.Compute/diskswriteops",
	"azure_compute_disk_metric_write_ops_hourly":                  "Microsoft.Compute/diskswriteopshourly",
	"azure_eventgrid_domain":                                      "Microsoft.EventGrid/domains",
	"azure_key_vault_deleted_vault":                               "Microsoft.KeyVault/deletedVaults",
	"azure_storage_table":                                         "Microsoft.Storage/storageAccounts/tableServices/tables",
	"azure_compute_snapshot":                                      "Microsoft.Compute/snapshots",
	"azure_kusto_cluster":                                         "Microsoft.Kusto/clusters",
	"azure_storage_sync":                                          "Microsoft.StorageSync/storageSyncServices",
	"azure_security_center_jit_network_access_policy":             "Microsoft.Security/locations/jitNetworkAccessPolicies",
	"azure_subnet":                                                "Microsoft.Network/virtualNetworks/subnets",
	"azure_lb_backend_address_pool":                               "Microsoft.Network/loadBalancers/backendAddressPools",
	"azure_lb_rule":                                               "Microsoft.Network/loadBalancers/loadBalancingRules",
	"azure_compute_virtual_machine_metric_cpu_utilization_daily":  "Microsoft.Compute/virtualMachineCpuUtilizationDaily",
	"azure_data_lake_store":                                       "Microsoft.DataLakeStore/accounts",
	"azure_hpc_cache":                                             "Microsoft.StorageCache/caches",
	"azure_batch_account":                                         "Microsoft.Batch/batchAccounts",
	"azure_network_security_group":                                "Microsoft.Network/networkSecurityGroups",
	"azure_role_definition":                                       "Microsoft.Authorization/roleDefinitions",
	"azure_application_security_group":                            "Microsoft.Network/applicationSecurityGroups",
	"azure_role_assignment":                                       "Microsoft.Authorization/roleAssignment",
	"azure_cosmosdb_mongo_database":                               "Microsoft.DocumentDB/databaseAccounts/mongodbDatabases",
	"azure_cosmosdb_mongo_collection":                             "Microsoft.DocumentDB/databaseAccounts/mongodbDatabases/collections",
	"azure_network_watcher_flow_log":                              "Microsoft.Network/networkWatchers/flowLogs",
	"azure_mssql_elasticpool":                                     "microsoft.Sql/servers/elasticpools",
	"azure_security_center_sub_assessment":                        "Microsoft.Security/subAssessments",
	"azure_compute_disk":                                          "Microsoft.Compute/disks",
	"azure_iothub_dps":                                            "Microsoft.Devices/ProvisioningServices",
	"azure_hdinsight_cluster":                                     "Microsoft.HDInsight/clusters",
	"azure_service_fabric_cluster":                                "Microsoft.ServiceFabric/clusters",
	"azure_signalr_service":                                       "Microsoft.SignalRService/signalR",
	"azure_storage_blob":                                          "Microsoft.Storage/storageAccounts/blob",
	"azure_storage_container":                                     "Microsoft.Storage/storageaccounts/blobservices/containers",
	"azure_storage_blob_service":                                  "Microsoft.Storage/storageAccounts/blobServices",
	"azure_storage_queue":                                         "Microsoft.Storage/storageAccounts/queueServices",
	"azure_api_management":                                        "Microsoft.ApiManagement/service",
	"azure_api_management_backend":                                "Microsoft.ApiManagement/backend",
	"azure_compute_disk_metric_read_ops":                          "Microsoft.Compute/disksreadops",
	"azure_compute_virtual_machine_scale_set":                     "Microsoft.Compute/virtualMachineScaleSets",
	"azure_data_factory_dataset":                                  "Microsoft.DataFactory/factories/datasets",
	"azure_policy_definition":                                     "Microsoft.Authorization/policyDefinitions",
	"azure_location":                                              "Microsoft.Resources/subscriptions/locations",
	"azure_compute_disk_access":                                   "Microsoft.Compute/diskAccesses",
	"azure_mysql_server":                                          "Microsoft.DBforMySQL/servers",
	"azure_dbformysql_flexibleservers":                            "Microsoft.DBforMySQL/flexibleservers",
	"azure_cache_redisenterprise":                                 "Microsoft.Cache/redisenterprise",
	"azure_data_lake_analytics_account":                           "Microsoft.DataLakeAnalytics/accounts",
	"azure_log_alert":                                             "Microsoft.Insights/activityLogAlerts",
	"azure_compute_virtual_machine_metric_cpu_utilization_hourly": "Microsoft.Compute/virtualMachineCpuUtilizationHourly",
	"azure_lb_outbound_rule":                                      "Microsoft.Network/loadBalancers/outboundRules",
	"azure_hybrid_compute_machine":                                "Microsoft.HybridCompute/machines",
	"azure_lb_nat_rule":                                           "Microsoft.Network/loadBalancers/inboundNatRules",
	"azure_provider":                                              "Microsoft.Resources/providers",
	"azure_route_table":                                           "Microsoft.Network/routeTables",
	"azure_cosmosdb_account":                                      "Microsoft.DocumentDB/databaseAccounts",
	"azure_cosmosdb_restorable_database_account":                  "Microsoft.DocumentDB/restorableDatabaseAccounts",
	"azure_application_gateway":                                   "Microsoft.Network/applicationGateways",
	"azure_security_center_automation":                            "Microsoft.Security/automations",
	"azure_hybrid_kubernetes_connected_cluster":                   "Microsoft.Kubernetes/connectedClusters",
	"azure_key_vault_key":                                         "Microsoft.KeyVault/vaults/keys",
	"azure_key_vault_certificate":                                 "Microsoft.KeyVault/vaults/certificates",
	"azure_key_vault_key_version":                                 "Microsoft.KeyVault/vaults/keys/Versions",
	"azure_mariadb_server":                                        "Microsoft.DBforMariaDB/servers",
	"azure_mariadb_databases":                                     "Microsoft.DBforMariaDB/servers/databases",
	"azure_compute_disk_metric_read_ops_daily":                    "Microsoft.Compute/disksreadopsdaily",
	"azure_app_service_plan":                                      "Microsoft.Web/plan",
	"azure_compute_disk_metric_read_ops_hourly":                   "Microsoft.Compute/disksreadopshourly",
	"azure_compute_disk_metric_write_ops_daily":                   "Microsoft.Compute/diskswriteopsdaily",
	"azure_tenant":                                                "Microsoft.Resources/tenants",
	"azure_virtual_network_gateway":                               "Microsoft.Network/virtualNetworkGateways",
	"azure_iothub":                                                "Microsoft.Devices/iotHubs",
	"azure_logic_app_workflow":                                    "Microsoft.Logic/workflows",
	"azure_mysql_flexible_server":                                 "Microsoft.Sql/flexibleServers",
	"azure_resource_link":                                         "Microsoft.Resources/links",
	"azure_subscription":                                          "Microsoft.Resources/subscriptions",
	"azure_compute_image":                                         "Microsoft.Compute/images",
	"azure_compute_virtual_machine":                               "Microsoft.Compute/virtualMachines",
	"azure_nat_gateway":                                           "Microsoft.Network/natGateways",
	"azure_lb_probe":                                              "Microsoft.Network/loadBalancers/probes",
	"azure_key_vault":                                             "Microsoft.KeyVault/vaults",
	"azure_key_vault_managed_hardware_security_module":            "Microsoft.KeyVault/managedHsms",
	"azure_key_vault_secret":                                      "Microsoft.KeyVault/vaults/secrets",
	"azure_app_configuration":                                     "Microsoft.AppConfiguration/configurationStores",
	"azure_compute_virtual_machine_metric_cpu_utilization":        "Microsoft.Compute/virtualMachineCpuUtilization",
	"azure_storage_account":                                       "Microsoft.Storage/storageAccounts",
	"azure_spring_cloud_service":                                  "Microsoft.AppPlatform/Spring",
	"azure_compute_image_gallery":                                 "Microsoft.Compute/galleries",
	"azure_compute_host_group":                                    "Microsoft.Compute/hostGroups",
	"azure_compute_host":                                          "Microsoft.Compute/hostGroups/hosts",
	"azure_compute_restore_point_collection":                      "Microsoft.Compute/restorePointCollections",
	"azure_compute_ssh_key":                                       "Microsoft.Compute/sshPublicKeys",
	"azure_cdn_endpoint":                                          "Microsoft.Cdn/profiles/endpoints",
	"azure_botservice_bot":                                        "Microsoft.BotService/botServices",
	"azure_cosmosdb_cassandra_cluster":                            "Microsoft.DocumentDB/cassandraClusters",
	"azure_network_ddos_protection_plan":                          "Microsoft.Network/ddosProtectionPlans",
	"azure_sql_instance_pool":                                     "microsoft.Sql/instancePools",
	"azure_netapp_account":                                        "microsoft.NetApp/netAppAccounts",
	"azure_netapp_capacity_pool":                                  "Microsoft.NetApp/netAppAccounts/capacityPools",
	"azure_desktop_virtualization_host_pool":                      "Microsoft.DesktopVirtualization/hostpools",
	"azure_devtestlab_lab":                                        "Microsoft.Devtestlab/labs",
	"azure_purview_account":                                       "Microsoft.Purview/Accounts",
	"azure_powerbidedicated_capacity":                             "Microsoft.PowerBIDedicated/capacities",
	"azure_application_insight":                                   "Microsoft.Insights/components",
	"azure_lighthouse_definition":                                 "Microsoft.Lighthouse/definition",
	"azure_lighthouse_assignment":                                 "Microsoft.Lighthouse/assignment",
	"azure_maintenance_configuration":                             "Microsoft.Maintenance/maintenanceConfigurations",
	"azure_monitor_log_profile":                                   "Microsoft.Monitor/logProfiles",
	"azure_resource":                                              "Microsoft.Resources/subscriptions/resources",
}

var ResourceTypesList = []string{
	"Microsoft.App/containerApps",
	"Microsoft.Blueprint/blueprints",
	"Microsoft.Cdn/profiles",
	"Microsoft.Compute/cloudServices",
	"Microsoft.ContainerInstance/containerGroups",
	"Microsoft.DataMigration/services",
	"Microsoft.DataProtection/backupVaults",
	"Microsoft.DataProtection/backupJobs",
	"Microsoft.DataProtection/backupVaults/backupPolicies",
	"Microsoft.Logic/integrationAccounts",
	"Microsoft.Network/bastionHosts",
	"Microsoft.Network/connections",
	"Microsoft.Network/firewallPolicies",
	"Microsoft.Network/localNetworkGateways",
	"Microsoft.Network/privateLinkServices",
	"Microsoft.Network/publicIPPrefixes",
	"Microsoft.Network/virtualHubs",
	"Microsoft.Network/virtualWans",
	"Microsoft.Network/vpnGateways",
	"Microsoft.Network/vpnGateways/vpnConnections",
	"Microsoft.Network/vpnSites",
	"Microsoft.OperationalInsights/workspaces",
	"Microsoft.StreamAnalytics/cluster",
	"Microsoft.TimeSeriesInsights/environments",
	"Microsoft.VirtualMachineImages/imageTemplates",
	"Microsoft.Web/serverFarms",
	"Microsoft.Compute/virtualMachineScaleSets/virtualMachines",
	"Microsoft.Automation/automationAccounts",
	"Microsoft.Automation/automationAccounts/variables",
	"Microsoft.Network/dnsZones",
	"Microsoft.Databricks/workspaces",
	"Microsoft.Network/privateDnsZones",
	"Microsoft.Network/privateEndpoints",
	"Microsoft.Network/networkWatchers",
	"Microsoft.Resources/subscriptions/resourceGroups",
	"Microsoft.Web/staticSites",
	"Microsoft.Web/sites/slots",
	"Microsoft.CognitiveServices/accounts",
	"Microsoft.Sql/managedInstances",
	"Microsoft.Sql/virtualclusters",
	"Microsoft.Sql/managedInstances/databases",
	"Microsoft.Sql/servers/databases",
	"Microsoft.Storage/storageAccounts/largeFileSharesState",
	"Microsoft.DBforPostgreSQL/servers",
	"Microsoft.DBforPostgreSQL/flexibleservers",
	"Microsoft.AnalysisServices/servers",
	"Microsoft.Security/pricings",
	"Microsoft.Insights/guestDiagnosticSettings",
	"Microsoft.Insights/autoscaleSettings",
	"Microsoft.Web/hostingEnvironments",
	"Microsoft.Cache/redis",
	"Microsoft.ContainerRegistry/registries",
	"Microsoft.DataFactory/factories/pipelines",
	"Microsoft.Compute/resourceSku",
	"Microsoft.Network/expressRouteCircuits",
	"Microsoft.Management/managementgroups",
	"microsoft.SqlVirtualMachine/SqlVirtualMachines",
	"Microsoft.SqlVirtualMachine/SqlVirtualMachineGroups",
	"Microsoft.Storage/storageAccounts/tableServices",
	"Microsoft.Synapse/workspaces",
	"Microsoft.Synapse/workspaces/bigdatapools",
	"Microsoft.Synapse/workspaces/sqlpools",
	"Microsoft.StreamAnalytics/streamingJobs",
	"Microsoft.CostManagement/CostBySubscription",
	"Microsoft.ContainerService/managedClusters",
	"Microsoft.ContainerService/serviceVersions",
	"Microsoft.DataFactory/factories",
	"Microsoft.Sql/servers",
	"Microsoft.Sql/servers/jobagents",
	"Microsoft.Security/autoProvisioningSettings",
	"Microsoft.Insights/logProfiles",
	"Microsoft.DataBoxEdge/dataBoxEdgeDevices",
	"Microsoft.Network/loadBalancers",
	"Microsoft.Network/azureFirewalls",
	"Microsoft.Management/locks",
	"Microsoft.Compute/virtualMachineScaleSets/networkInterfaces",
	"Microsoft.Network/frontDoors",
	"Microsoft.Authorization/policyAssignments",
	"Microsoft.Authorization/userEffectiveAccess",
	"Microsoft.Search/searchServices",
	"Microsoft.Security/settings",
	"Microsoft.RecoveryServices/vaults",
	"Microsoft.RecoveryServices/vaults/backupJobs",
	"Microsoft.RecoveryServices/vaults/backupPolicies",
	"Microsoft.RecoveryServices/vaults/backupItems",
	"Microsoft.Compute/diskEncryptionSets",
	"Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
	"Microsoft.EventGrid/topics",
	"Microsoft.EventHub/namespaces",
	"Microsoft.EventHub/namespaces/eventHubs",
	"Microsoft.MachineLearningServices/workspaces",
	"Microsoft.Dashboard/grafana",
	"Microsoft.DesktopVirtualization/workspaces",
	"Microsoft.Network/trafficManagerProfiles",
	"Microsoft.Network/dnsResolvers",
	"Microsoft.CostManagement/CostByResourceType",
	"Microsoft.Network/networkInterfaces",
	"Microsoft.Network/publicIPAddresses",
	"Microsoft.HealthcareApis/services",
	"Microsoft.ServiceBus/namespaces",
	"Microsoft.Web/sites",
	"Microsoft.Compute/availabilitySets",
	"Microsoft.Network/virtualNetworks",
	"Microsoft.Security/securityContacts",
	"Microsoft.Compute/diskswriteops",
	"Microsoft.Compute/diskswriteopshourly",
	"Microsoft.EventGrid/domains",
	"Microsoft.KeyVault/deletedVaults",
	"Microsoft.Storage/storageAccounts/tableServices/tables",
	"Microsoft.Compute/snapshots",
	"Microsoft.Kusto/clusters",
	"Microsoft.StorageSync/storageSyncServices",
	"Microsoft.Security/locations/jitNetworkAccessPolicies",
	"Microsoft.Network/virtualNetworks/subnets",
	"Microsoft.Network/loadBalancers/backendAddressPools",
	"Microsoft.Network/loadBalancers/loadBalancingRules",
	"Microsoft.Compute/virtualMachineCpuUtilizationDaily",
	"Microsoft.DataLakeStore/accounts",
	"Microsoft.StorageCache/caches",
	"Microsoft.Batch/batchAccounts",
	"Microsoft.Network/networkSecurityGroups",
	"Microsoft.Authorization/roleDefinitions",
	"Microsoft.Network/applicationSecurityGroups",
	"Microsoft.Authorization/roleAssignment",
	"Microsoft.DocumentDB/databaseAccounts/mongodbDatabases",
	"Microsoft.DocumentDB/databaseAccounts/mongodbDatabases/collections",
	"Microsoft.Network/networkWatchers/flowLogs",
	"microsoft.Sql/servers/elasticpools",
	"Microsoft.Security/subAssessments",
	"Microsoft.Compute/disks",
	"Microsoft.Devices/ProvisioningServices",
	"Microsoft.HDInsight/clusters",
	"Microsoft.ServiceFabric/clusters",
	"Microsoft.SignalRService/signalR",
	"Microsoft.Storage/storageAccounts/blob",
	"Microsoft.Storage/storageaccounts/blobservices/containers",
	"Microsoft.Storage/storageAccounts/blobServices",
	"Microsoft.Storage/storageAccounts/queueServices",
	"Microsoft.ApiManagement/service",
	"Microsoft.ApiManagement/backend",
	"Microsoft.Compute/disksreadops",
	"Microsoft.Compute/virtualMachineScaleSets",
	"Microsoft.DataFactory/factories/datasets",
	"Microsoft.Authorization/policyDefinitions",
	"Microsoft.Resources/subscriptions/locations",
	"Microsoft.Compute/diskAccesses",
	"Microsoft.DBforMySQL/servers",
	"Microsoft.DBforMySQL/flexibleservers",
	"Microsoft.Cache/redisenterprise",
	"Microsoft.DataLakeAnalytics/accounts",
	"Microsoft.Insights/activityLogAlerts",
	"Microsoft.Compute/virtualMachineCpuUtilizationHourly",
	"Microsoft.Network/loadBalancers/outboundRules",
	"Microsoft.HybridCompute/machines",
	"Microsoft.Network/loadBalancers/inboundNatRules",
	"Microsoft.Resources/providers",
	"Microsoft.Network/routeTables",
	"Microsoft.DocumentDB/databaseAccounts",
	"Microsoft.DocumentDB/restorableDatabaseAccounts",
	"Microsoft.Network/applicationGateways",
	"Microsoft.Security/automations",
	"Microsoft.Kubernetes/connectedClusters",
	"Microsoft.KeyVault/vaults/keys",
	"Microsoft.KeyVault/vaults/certificates",
	"Microsoft.KeyVault/vaults/keys/Versions",
	"Microsoft.DBforMariaDB/servers",
	"Microsoft.DBforMariaDB/servers/databases",
	"Microsoft.Compute/disksreadopsdaily",
	"Microsoft.Web/plan",
	"Microsoft.Compute/disksreadopshourly",
	"Microsoft.Compute/diskswriteopsdaily",
	"Microsoft.Resources/tenants",
	"Microsoft.Network/virtualNetworkGateways",
	"Microsoft.Devices/iotHubs",
	"Microsoft.Logic/workflows",
	"Microsoft.Sql/flexibleServers",
	"Microsoft.Resources/links",
	"Microsoft.Resources/subscriptions",
	"Microsoft.Compute/images",
	"Microsoft.Compute/virtualMachines",
	"Microsoft.Network/natGateways",
	"Microsoft.Network/loadBalancers/probes",
	"Microsoft.KeyVault/vaults",
	"Microsoft.KeyVault/managedHsms",
	"Microsoft.KeyVault/vaults/secrets",
	"Microsoft.AppConfiguration/configurationStores",
	"Microsoft.Compute/virtualMachineCpuUtilization",
	"Microsoft.Storage/storageAccounts",
	"Microsoft.AppPlatform/Spring",
	"Microsoft.Compute/galleries",
	"Microsoft.Compute/hostGroups",
	"Microsoft.Compute/hostGroups/hosts",
	"Microsoft.Compute/restorePointCollections",
	"Microsoft.Compute/sshPublicKeys",
	"Microsoft.Cdn/profiles/endpoints",
	"Microsoft.BotService/botServices",
	"Microsoft.DocumentDB/cassandraClusters",
	"Microsoft.Network/ddosProtectionPlans",
	"microsoft.Sql/instancePools",
	"microsoft.NetApp/netAppAccounts",
	"Microsoft.NetApp/netAppAccounts/capacityPools",
	"Microsoft.DesktopVirtualization/hostpools",
	"Microsoft.Devtestlab/labs",
	"Microsoft.Purview/Accounts",
	"Microsoft.PowerBIDedicated/capacities",
	"Microsoft.Insights/components",
	"Microsoft.Lighthouse/definition",
	"Microsoft.Lighthouse/assignment",
	"Microsoft.Maintenance/maintenanceConfigurations",
	"Microsoft.Monitor/logProfiles",
	"Microsoft.Resources/subscriptions/resources",
}
