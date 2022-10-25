package steampipe

import (
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

var awsMap = map[string]string{
	"AWS::CostExplorer::ByAccountMonthly":       "aws_cost_by_account_monthly",
	"AWS::CostExplorer::ByServiceMonthly":       "aws_cost_by_service_monthly",
	"AWS::SSM::ManagedInstanceCompliance":       "aws_ssm_managed_instance_compliance",
	"AWS::ApplicationAutoScaling::Target":       "aws_appautoscaling_target",
	"AWS::EKS::Cluster":                         "aws_eks_cluster",
	"AWS::ElasticLoadBalancingV2::LoadBalancer": "aws_ec2_application_load_balancer",
	"AWS::IAM::User":                            "aws_iam_user",
	"AWS::SecretsManager::Secret":               "aws_secretsmanager_secret",
	"AWS::GuardDuty::Finding":                   "aws_guardduty_finding",
	"AWS::IAM::Role":                            "aws_iam_role",
	"AWS::Redshift::ClusterParameterGroup":      "aws_redshift_parameter_group",
	"AWS::CertificateManager::Certificate":      "aws_acm_certificate",
	"AWS::EC2::RegionalSettings":                "aws_ec2_regional_settings",
	"AWS::EC2::Subnet":                          "aws_vpc_subnet",
	"AWS::EC2::VPC":                             "aws_vpc",
	"AWS::ElasticBeanstalk::Environment":        "aws_elastic_beanstalk_environment",
	"AWS::EC2::VPNConnection":                   "aws_vpc_vpn_connection",
	"AWS::EMR::Cluster":                         "aws_emr_cluster",
	"AWS::IAM::AccountSummary":                  "aws_iam_account_summary",
	"AWS::S3::AccessPoint":                      "aws_s3_access_point",
	"AWS::DynamoDb::Table":                      "aws_dynamodb_table",
	"AWS::EC2::VPCEndpoint":                     "aws_vpc_endpoint",
	"AWS::IAM::AccessKey":                       "aws_iam_access_key",
	"AWS::SageMaker::EndpointConfiguration":     "aws_sagemaker_endpoint_configuration",
	"AWS::RDS::DBInstance":                      "aws_rds_db_instance",
	"AWS::RDS::DBEventSubscription":             "aws_rds_db_event_subscription",
	"AWS::SQS::Queue":                           "aws_sqs_queue",
	"AWS::Backup::RecoveryPoint":                "aws_backup_recovery_point",
	"AWS::Config::ConfigurationRecorder":        "aws_config_configuration_recorder",
	"AWS::ElasticSearch::Domain":                "aws_elasticsearch_domain",
	"AWS::IAM::Policy":                          "aws_iam_policy",
	"AWS::RDS::DBCluster":                       "aws_rds_db_cluster",
	"AWS::CloudWatch::Alarm":                    "aws_cloudwatch_alarm",
	"AWS::EFS::FileSystem":                      "aws_efs_file_system",
	"AWS::IAM::CredentialReport":                "aws_iam_credential_report",
	"AWS::CloudFront::Distribution":             "aws_cloudfront_distribution",
	"AWS::EC2::EIP":                             "aws_vpc_eip",
	"AWS::ECS::TaskDefinition":                  "aws_ecs_task_definition",
	"AWS::IAM::Group":                           "aws_iam_group",
	"AWS::EC2::NetworkInterface":                "aws_ec2_network_interface",
	"AWS::AccessAnalyzer::Analyzer":             "aws_accessanalyzer_analyzer",
	"AWS::AutoScaling::AutoScalingGroup":        "aws_ec2_autoscaling_group",
	"AWS::EC2::Volume":                          "aws_ebs_volume",
	"AWS::EC2::NatGateway":                      "aws_vpc_nat_gateway",
	"AWS::EC2::NetworkAcl":                      "aws_vpc_network_acl",
	"AWS::Lambda::Function":                     "aws_lambda_function",
	"AWS::Logs::MetricFilter":                   "aws_cloudwatch_log_metric_filter",
	"AWS::S3::AccountSetting":                   "aws_s3_account_settings",
	"AWS::SNS::Subscription":                    "aws_sns_topic_subscription",
	"AWS::SSM::ManagedInstance":                 "aws_ssm_managed_instance",
	"AWS::WAFv2::WebACL":                        "aws_wafv2_web_acl",
	"AWS::CloudTrail::Trail":                    "aws_cloudtrail_trail",
	"AWS::CodeBuild::Project":                   "aws_codebuild_project",
	"AWS::EC2::InternetGateway":                 "aws_vpc_internet_gateway",
	"AWS::EC2::RouteTable":                      "aws_vpc_route_table",
	"AWS::SecurityHub::Hub":                     "aws_securityhub_hub",
	"AWS::AutoScaling::LaunchConfiguration":     "aws_ec2_launch_configuration",
	"AWS::Backup::ProtectedResource":            "aws_backup_protected_resource",
	"AWS::Backup::Vault":                        "aws_backup_vault",
	"AWS::ECS::Cluster":                         "aws_ecs_cluster",
	"AWS::Logs::LogGroup":                       "aws_cloudwatch_log_group",
	"AWS::Backup::Selection":                    "aws_backup_selection",
	"AWS::EKS::Addon":                           "aws_eks_addon",
	"AWS::IAM::VirtualMFADevice":                "aws_iam_virtual_mfa_device",
	"AWS::KMS::Key":                             "aws_kms_key",
	"AWS::IAM::Account":                         "aws_account",
	"AWS::IAM::IAMAccountPasswordPolicy":        "aws_iam_account_password_policy",
	"AWS::IAM::ServerCertificate":               "aws_iam_server_certificate",
	"AWS::DAX::Cluster":                         "aws_dax_cluster",
	"AWS::DMS::ReplicationInstance":             "aws_dms_replication_instance",
	"AWS::EC2::Instance":                        "aws_ec2_instance",
	"AWS::ECS::Service":                         "aws_ecs_service",
	"AWS::GuardDuty::Detector":                  "aws_guardduty_detector",
	"AWS::SageMaker::NotebookInstance":          "aws_sagemaker_notebook_instance",
	"AWS::Backup::Plan":                         "aws_backup_plan",
	"AWS::EC2::SecurityGroup":                   "aws_vpc_security_group",
	"AWS::CodeBuild::SourceCredential":          "aws_codebuild_source_credential",
	"AWS::ElasticLoadBalancing::LoadBalancer":   "aws_ec2_classic_load_balancer",
	"AWS::SNS::Topic":                           "aws_sns_topic",
	"AWS::ElastiCache::ReplicationGroup":        "aws_elasticache_replication_group",
	"AWS::ElasticLoadBalancingV2::Listener":     "aws_ec2_load_balancer_listener",
	"AWS::ApiGateway::Stage":                    "aws_api_gateway_stage",
	"AWS::Redshift::Cluster":                    "aws_redshift_cluster",
	"AWS::S3::Bucket":                           "aws_s3_bucket",
	"AWS::EC2::VolumeSnapshot":                  "aws_ebs_snapshot",
	"AWS::EC2::FlowLog":                         "aws_vpc_flow_log",
	"AWS::EC2::Region":                          "aws_region",
	"AWS::FSX::FileSystem":                      "aws_fsx_file_system",
	"AWS::RDS::DBClusterSnapshot":               "aws_rds_db_cluster_snapshot",
	"AWS::EC2::CapacityReservation":             "aws_ec2_capacity_reservation",
	"AWS::EC2::KeyPair":                         "aws_ec2_key_pair",
	"AWS::EC2::Image":                           "aws_ec2_ami",
	"AWS::EC2::ReservedInstances":               "aws_ec2_reserved_instance",
	"AWS::ECR::Repository":                      "aws_ecr_repository",
	"AWS::ECR::PublicRepository":                "aws_ecrpublic_repository",
	"AWS::ECS::ContainerInstance":               "aws_ecs_container_instance",
	"AWS::ElastiCache::Cluster":                 "aws_elasticache_cluster",
	"AWS::EventBridge::EventBus":                "aws_eventbridge_bus",
	//"AWS::EFS::AccessPoint":                     "aws_efs_access_point",
	//"AWS::EFS::MountTarget":                     "aws_efs_mount_target",
}
var azureMap = map[string]string{
	"Microsoft.Compute/disks":                               "azure_compute_disk",
	"Microsoft.DBforMySQL/servers":                          "azure_mysql_server",
	"Microsoft.Network/networkWatchers":                     "azure_network_watcher",
	"Microsoft.Resources/links":                             "azure_resource_link",
	"Microsoft.Network/virtualNetworks/subnets":             "azure_subnet",
	"Microsoft.ContainerRegistry/registries":                "azure_container_registry",
	"Microsoft.ContainerService/managedClusters":            "azure_kubernetes_cluster",
	"Microsoft.HDInsight/clusters":                          "azure_hdinsight_cluster",
	"Microsoft.KeyVault/vaults/keys":                        "azure_key_vault_key",
	"Microsoft.Kusto/clusters":                              "azure_kusto_cluster",
	"Microsoft.DataFactory/factories":                       "azure_data_factory",
	"Microsoft.Search/searchServices":                       "azure_search_service",
	"Microsoft.SignalRService/signalR":                      "azure_signalr_service",
	"Microsoft.ApiManagement/service":                       "azure_api_management",
	"Microsoft.Authorization/elevateAccessRoleAssignment":   "azure_role_assignment",
	"Microsoft.DocumentDB/databaseAccounts":                 "azure_cosmosdb_account",
	"Microsoft.DocumentDB/databaseAccounts/sqlDatabases":    "azure_cosmosdb_sql_database",
	"Microsoft.DBforMariaDB/servers":                        "azure_mariadb_server",
	"Microsoft.Authorization/roleDefinitions":               "azure_role_definition",
	"Microsoft.Network/applicationGateways":                 "azure_application_gateway",
	"Microsoft.KeyVault/vaults/secrets":                     "azure_key_vault_secret",
	"Microsoft.Web/hostingEnvironments":                     "azure_app_service_environment",
	"Microsoft.Compute/virtualMachineScaleSets":             "azure_compute_virtual_machine_scale_set",
	"Microsoft.EventGrid/domains":                           "azure_eventgrid_domain",
	"Microsoft.EventGrid/domains/topics":                    "azure_eventgrid_topic",
	"Microsoft.DataBoxEdge/dataBoxEdgeDevices":              "azure_databox_edge_device",
	"Microsoft.StorageSync/storageSyncServices":             "azure_storage_sync",
	"Microsoft.Insights/logProfiles":                        "azure_log_profile",
	"Microsoft.Resources/users":                             "azuread_user",
	"Microsoft.Network/networkWatchers/flowLogs":            "azure_network_watcher_flow_log",
	"Microsoft.Sql/servers":                                 "azure_sql_server",
	"Microsoft.Storage/storageAccounts":                     "azure_storage_account",
	"Microsoft.StorageCache/caches":                         "azure_hpc_cache",
	"Microsoft.KeyVault/managedHsms":                        "azure_key_vault_managed_hardware_security_module",
	"Microsoft.Devices/iotHubs":                             "azure_iothub",
	"Microsoft.Sql/managedInstances":                        "azure_mssql_managed_instance",
	"Microsoft.AppPlatform/Spring":                          "azure_spring_cloud_service",
	"Microsoft.Security/autoProvisioningSettings":           "azure_security_center_auto_provisioning",
	"Microsoft.Security/securityContacts":                   "azure_security_center_contact",
	"Microsoft.Security/locations/jitNetworkAccessPolicies": "azure_security_center_jit_network_access_policy",
	"Microsoft.CognitiveServices/accounts":                  "azure_cognitive_account",
	"Microsoft.HybridCompute/machines":                      "azure_hybrid_compute_machine",
	"Microsoft.ServiceBus/namespaces":                       "azure_servicebus_namespace",
	"Microsoft.Resources/tenants":                           "azure_tenant",
	"Microsoft.Sql/servers/databases":                       "azure_sql_database",
	"Microsoft.Network/networkInterfaces":                   "azure_network_interface",
	"Microsoft.Web/sites":                                   "azure_app_service_function_app",
	"Microsoft.Authorization/policyAssignments":             "azure_policy_assignment",
	"Microsoft.Resources/subscriptions":                     "azure_subscription",
	"Microsoft.DataLakeAnalytics/accounts":                  "azure_data_lake_analytics_account",
	"Microsoft.Network/frontDoors":                          "azure_frontdoor",
	"Microsoft.KeyVault/vaults":                             "azure_key_vault",
	"Microsoft.Network/virtualNetworks":                     "azure_virtual_network",
	"Microsoft.ServiceFabric/clusters":                      "azure_service_fabric_cluster",
	"Microsoft.Web/staticSites":                             "azure_app_service_web_app",
	"Microsoft.Insights/activityLogAlerts":                  "azure_log_alert",
	"Microsoft.Resources/subscriptions/locations":           "azure_location",
	"Microsoft.Resources/subscriptions/resourceGroups":      "azure_resource_group",
	"Microsoft.Compute/virtualMachines":                     "azure_compute_virtual_machine",
	"Microsoft.DBforPostgreSQL/servers":                     "azure_postgresql_server",
	"Microsoft.EventGrid/topics":                            "azure_eventgrid_topic",
	"Microsoft.MachineLearningServices/workspaces":          "azure_machine_learning_workspace",
	"Microsoft.Insights/guestDiagnosticSettings":            "azure_diagnostic_setting",
	"Microsoft.Cache/redis":                                 "azure_redis_cache",
	"Microsoft.Compute/diskAccesses":                        "azure_compute_disk_access",
	"Microsoft.DataLakeStore/accounts":                      "azure_data_lake_store",
	"Microsoft.Synapse/workspaces":                          "azure_synapse_workspace",
	"Microsoft.HealthcareApis/services":                     "azure_healthcare_service",
	"Microsoft.Security/pricings":                           "azure_security_center_subscription_pricing",
	"Microsoft.Logic/workflows":                             "azure_logic_app_workflow",
	"Microsoft.Batch/batchAccounts":                         "azure_batch_account",
	"Microsoft.ClassicNetwork/networkSecurityGroups":        "azure_network_security_group",
	"Microsoft.StreamAnalytics/streamingJobs":               "azure_stream_analytics_job",
	"Microsoft.Security/settings":                           "azure_security_center_setting",
	"Microsoft.AppConfiguration/configurationStores":        "azure_app_configuration",
	"Microsoft.EventHub/namespaces":                         "azure_eventhub_namespace",
	"Microsoft.Storage/storageAccounts/containers":          "azure_storage_container",
	"Microsoft.Network/applicationSecurityGroups":           "azure_network_applicationsecuritygroups",
	"Microsoft.RecoveryServices/vaults":                     "azure_recoveryservices_vault",
	"Microsoft.Network/azureFirewalls":                      "azure_network_azurefirewall",
	"Microsoft.Network/expressRouteCircuits":                "azure_network_expressroutecircuit",
	"Microsoft.Network/loadBalancers":                       "azure_network_loadbalancers",
	"Microsoft.Network/routeTables":                         "azure_network_routetables",
	"Microsoft.Compute/snapshots":                           "azure_compute_snapshots",
	"Microsoft.Network/virtualNetworkGateways":              "azure_network_virtualnetworkgateway",
	"Microsoft.Compute/availabilitySets":                    "azure_compute_availabilityset",
	"Microsoft.Compute/diskEncryptionSets":                  "azure_compute_diskencryptionset",
	"Microsoft.Kubernetes/connectedClusters":                "azure_hybridkubernetes_connectedcluster",
	"Microsoft.Authorization/policyDefinitions":             "azure_authorization_policydefinition",
}
var AWSDescriptionMap = map[string]interface{}{
	"AWS::CostExplorer::ByAccountMonthly": &keibi.CostExplorerByAccountMonthly{},
	"AWS::CostExplorer::ByServiceMonthly": &keibi.CostExplorerByServiceMonthly{},
	//"AWS::EFS::AccessPoint":                     &keibi.EFSAccesspoint{},
	//"AWS::EFS::MountTarget":                     &keibi.Mount,
	"AWS::Logs::MetricFilter":                   &keibi.CloudWatchLogsMetricFilter{},
	"AWS::RDS::DBCluster":                       &keibi.RDSDBCluster{},
	"AWS::SQS::Queue":                           &keibi.SQSQueue{},
	"AWS::CloudWatch::Alarm":                    &keibi.CloudWatchAlarm{},
	"AWS::Config::ConfigurationRecorder":        &keibi.ConfigConfigurationRecorder{},
	"AWS::ECS::Cluster":                         &keibi.ECSCluster{},
	"AWS::KMS::Key":                             &keibi.KMSKey{},
	"AWS::RDS::DBInstance":                      &keibi.RDSDBInstance{},
	"AWS::S3::AccessPoint":                      &keibi.S3AccessPoint{},
	"AWS::IAM::AccountSummary":                  &keibi.IAMAccountSummary{},
	"AWS::Backup::ProtectedResource":            &keibi.BackupProtectedResource{},
	"AWS::CloudTrail::Trail":                    &keibi.CloudTrailTrail{},
	"AWS::CodeBuild::SourceCredential":          &keibi.CodeBuildSourceCredential{},
	"AWS::EC2::NetworkInterface":                &keibi.EC2NetworkInterface{},
	"AWS::EC2::SecurityGroup":                   &keibi.EC2SecurityGroup{},
	"AWS::EFS::FileSystem":                      &keibi.EFSFileSystem{},
	"AWS::EMR::Cluster":                         &keibi.EMRCluster{},
	"AWS::SageMaker::EndpointConfiguration":     &keibi.SageMakerEndpointConfiguration{},
	"AWS::AutoScaling::AutoScalingGroup":        &keibi.AutoScalingGroup{},
	"AWS::Backup::Vault":                        &keibi.BackupVault{},
	"AWS::CertificateManager::Certificate":      &keibi.CertificateManagerCertificate{},
	"AWS::EC2::InternetGateway":                 &keibi.EC2InternetGateway{},
	"AWS::SecurityHub::Hub":                     &keibi.SecurityHubHub{},
	"AWS::DynamoDb::Table":                      &keibi.DynamoDbTable{},
	"AWS::EC2::Instance":                        &keibi.EC2Instance{},
	"AWS::ElasticLoadBalancing::LoadBalancer":   &keibi.ElasticLoadBalancingLoadBalancer{},
	"AWS::ElasticLoadBalancingV2::LoadBalancer": &keibi.ElasticLoadBalancingV2LoadBalancer{},
	"AWS::IAM::Account":                         &keibi.IAMAccount{},
	"AWS::SecretsManager::Secret":               &keibi.SecretsManagerSecret{},
	"AWS::ElasticSearch::Domain":                &keibi.ESDomain{},
	"AWS::WAFv2::WebACL":                        &keibi.WAFv2WebACL{},
	"AWS::EC2::FlowLog":                         &keibi.EC2FlowLog{},
	"AWS::EC2::NetworkAcl":                      &keibi.EC2NetworkAcl{},
	"AWS::EKS::Addon":                           &keibi.EKSAddon{},
	"AWS::RDS::DBClusterSnapshot":               &keibi.RDSDBClusterSnapshot{},
	"AWS::Redshift::Cluster":                    &keibi.RedshiftCluster{},
	"AWS::Redshift::ClusterParameterGroup":      &keibi.RedshiftClusterParameterGroup{},
	"AWS::SSM::ManagedInstance":                 &keibi.SSMManagedInstance{},
	"AWS::SNS::Subscription":                    &keibi.SNSSubscription{},
	"AWS::EC2::EIP":                             &keibi.EC2EIP{},
	"AWS::EC2::RouteTable":                      &keibi.EC2RouteTable{},
	"AWS::EKS::Cluster":                         &keibi.EKSCluster{},
	"AWS::ElastiCache::ReplicationGroup":        &keibi.ElastiCacheReplicationGroup{},
	"AWS::ElasticLoadBalancingV2::Listener":     &keibi.ElasticLoadBalancingV2Listener{},
	"AWS::RDS::DBEventSubscription":             &keibi.RDSDBEventSubscription{},
	"AWS::AutoScaling::LaunchConfiguration":     &keibi.AutoScalingLaunchConfiguration{},
	"AWS::Backup::Plan":                         &keibi.BackupPlan{},
	"AWS::Backup::Selection":                    &keibi.BackupSelection{},
	"AWS::FSX::FileSystem":                      &keibi.FSXFileSystem{},
	"AWS::GuardDuty::Detector":                  &keibi.GuardDutyDetector{},
	"AWS::IAM::VirtualMFADevice":                &keibi.IAMVirtualMFADevice{},
	"AWS::Logs::LogGroup":                       &keibi.CloudWatchLogsLogGroup{},
	"AWS::EC2::Region":                          &keibi.EC2Region{},
	"AWS::EC2::VPC":                             &keibi.EC2Vpc{},
	"AWS::IAM::Policy":                          &keibi.IAMPolicy{},
	"AWS::IAM::ServerCertificate":               &keibi.IAMServerCertificate{},
	"AWS::SNS::Topic":                           &keibi.SNSTopic{},
	"AWS::IAM::Role":                            &keibi.IAMRole{},
	"AWS::ApplicationAutoScaling::Target":       &keibi.ApplicationAutoScalingTarget{},
	"AWS::Backup::RecoveryPoint":                &keibi.BackupRecoveryPoint{},
	"AWS::DMS::ReplicationInstance":             &keibi.DMSReplicationInstance{},
	"AWS::EC2::VolumeSnapshot":                  &keibi.EC2VolumeSnapshot{},
	"AWS::EC2::NatGateway":                      &keibi.EC2NatGateway{},
	"AWS::ECS::TaskDefinition":                  &keibi.ECSTaskDefinition{},
	"AWS::IAM::User":                            &keibi.IAMUser{},
	"AWS::S3::AccountSetting":                   &keibi.S3AccountSetting{},
	"AWS::DAX::Cluster":                         &keibi.DAXCluster{},
	"AWS::EC2::Volume":                          &keibi.EC2Volume{},
	"AWS::EC2::Subnet":                          &keibi.EC2Subnet{},
	"AWS::ECS::Service":                         &keibi.ECSService{},
	"AWS::IAM::AccessKey":                       &keibi.IAMAccessKey{},
	"AWS::CodeBuild::Project":                   &keibi.CodeBuildProject{},
	"AWS::EC2::RegionalSettings":                &keibi.EC2RegionalSettings{},
	"AWS::IAM::Group":                           &keibi.IAMGroup{},
	"AWS::IAM::CredentialReport":                &keibi.IAMCredentialReport{},
	"AWS::S3::Bucket":                           &keibi.S3Bucket{},
	"AWS::SSM::ManagedInstanceCompliance":       &keibi.SSMManagedInstanceCompliance{},
	"AWS::AccessAnalyzer::Analyzer":             &keibi.AccessAnalyzerAnalyzer{},
	"AWS::EC2::VPNConnection":                   &keibi.EC2VPNConnection{},
	"AWS::ElasticBeanstalk::Environment":        &keibi.ElasticBeanstalkEnvironment{},
	"AWS::IAM::IAMAccountPasswordPolicy":        &keibi.IAMAccountPasswordPolicy{},
	"AWS::ApiGateway::Stage":                    &keibi.ApiGatewayStage{},
	"AWS::CloudFront::Distribution":             &keibi.CloudFrontDistribution{},
	"AWS::EC2::VPCEndpoint":                     &keibi.EC2VPCEndpoint{},
	"AWS::GuardDuty::Finding":                   &keibi.GuardDutyFinding{},
	"AWS::Lambda::Function":                     &keibi.LambdaFunction{},
	"AWS::SageMaker::NotebookInstance":          &keibi.SageMakerNotebookInstance{},
	"AWS::EC2::CapacityReservation":             &keibi.EC2CapacityReservation{},
	"AWS::EC2::KeyPair":                         &keibi.EC2KeyPair{},
	"AWS::EC2::Image":                           &keibi.EC2AMI{},
	"AWS::EC2::ReservedInstances":               &keibi.EC2ReservedInstances{},
	"AWS::ECR::Repository":                      &keibi.ECRRepository{},
	"AWS::ECR::PublicRepository":                &keibi.ECRPublicRepository{},
	"AWS::ECS::ContainerInstance":               &keibi.ECSContainerInstance{},
	"AWS::ElastiCache::Cluster":                 &keibi.ElastiCacheCluster{},
	"AWS::EventBridge::EventBus":                &keibi.EventBridgeBus{},
}
var AzureDescriptionMap = map[string]interface{}{
	"Microsoft.CognitiveServices/accounts":                  &keibi.CognitiveAccount{},
	"Microsoft.DataLakeStore/accounts":                      &keibi.DataLakeStore{},
	"Microsoft.Web/sites":                                   &keibi.AppServiceFunctionApp{},
	"Microsoft.Security/securityContacts":                   &keibi.SecurityCenterContact{},
	"Microsoft.Security/settings":                           &keibi.SecurityCenterSetting{},
	"Microsoft.AppConfiguration/configurationStores":        &keibi.AppConfiguration{},
	"Microsoft.Network/virtualNetworks":                     &keibi.VirtualNetwork{},
	"Microsoft.Web/staticSites":                             &keibi.AppServiceWebApp{},
	"Microsoft.Storage/storageAccounts/containers":          &keibi.StorageContainer{},
	"Microsoft.Network/virtualNetworks/subnets":             &keibi.Subnet{},
	"Microsoft.Devices/iotHubs":                             &keibi.IOTHub{},
	"Microsoft.SignalRService/signalR":                      &keibi.SignalrService{},
	"Microsoft.Storage/storageAccounts":                     &keibi.StorageAccount{},
	"Microsoft.HealthcareApis/services":                     &keibi.HealthcareService{},
	"Microsoft.DataFactory/factories":                       &keibi.DataFactory{},
	"Microsoft.EventGrid/domains":                           &keibi.EventGridDomain{},
	"Microsoft.EventGrid/domains/topics":                    &keibi.EventGridTopic{},
	"Microsoft.HybridCompute/machines":                      &keibi.HybridComputeMachine{},
	"Microsoft.ServiceBus/namespaces":                       &keibi.ServicebusNamespace{},
	"Microsoft.ServiceFabric/clusters":                      &keibi.ServiceFabricCluster{},
	"Microsoft.Sql/managedInstances":                        &keibi.MssqlManagedInstance{},
	"Microsoft.KeyVault/managedHsms":                        &keibi.KeyVaultManagedHardwareSecurityModule{},
	"Microsoft.StorageSync/storageSyncServices":             &keibi.StorageSync{},
	"Microsoft.DocumentDB/databaseAccounts":                 &keibi.CosmosdbAccount{},
	"Microsoft.Network/frontDoors":                          &keibi.Frontdoor{},
	"Microsoft.Kusto/clusters":                              &keibi.KustoCluster{},
	"Microsoft.Resources/subscriptions/locations":           &keibi.Location{},
	"Microsoft.Web/hostingEnvironments":                     &keibi.AppServiceEnvironment{},
	"Microsoft.DataBoxEdge/dataBoxEdgeDevices":              &keibi.DataboxEdgeDevice{},
	"Microsoft.Batch/batchAccounts":                         &keibi.BatchAccount{},
	"Microsoft.Insights/activityLogAlerts":                  &keibi.LogAlert{},
	"Microsoft.Compute/virtualMachines":                     &keibi.ComputeVirtualMachine{},
	"Microsoft.Search/searchServices":                       &keibi.SearchService{},
	"Microsoft.AppPlatform/Spring":                          &keibi.SpringCloudService{},
	"Microsoft.Resources/subscriptions":                     &keibi.Subscription{},
	"Microsoft.Compute/diskAccesses":                        &keibi.ComputeDiskAccess{},
	"Microsoft.ContainerRegistry/registries":                &keibi.ContainerRegistry{},
	"Microsoft.DBforPostgreSQL/servers":                     &keibi.PostgresqlServer{},
	"Microsoft.DataLakeAnalytics/accounts":                  &keibi.DataLakeAnalyticsAccount{},
	"Microsoft.KeyVault/vaults/keys":                        &keibi.KeyVaultKey{},
	"Microsoft.Logic/workflows":                             &keibi.LogicAppWorkflow{},
	"Microsoft.Synapse/workspaces":                          &keibi.SynapseWorkspace{},
	"Microsoft.Authorization/roleDefinitions":               &keibi.RoleDefinition{},
	"Microsoft.DBforMariaDB/servers":                        &keibi.MariadbServer{},
	"Microsoft.ApiManagement/service":                       &keibi.APIManagement{},
	"Microsoft.ContainerService/managedClusters":            &keibi.KubernetesCluster{},
	"Microsoft.EventHub/namespaces":                         &keibi.EventhubNamespace{},
	"Microsoft.Network/networkInterfaces":                   &keibi.NetworkInterface{},
	"Microsoft.MachineLearningServices/workspaces":          &keibi.MachineLearningWorkspace{},
	"Microsoft.Sql/servers":                                 &keibi.SqlServer{},
	"Microsoft.StorageCache/caches":                         &keibi.HpcCache{},
	"Microsoft.Network/networkWatchers":                     &keibi.NetworkWatcher{},
	"Microsoft.Insights/logProfiles":                        &keibi.LogProfile{},
	"Microsoft.EventGrid/topics":                            &keibi.EventGridTopic{},
	"Microsoft.Insights/guestDiagnosticSettings":            &keibi.DiagnosticSetting{},
	"Microsoft.KeyVault/vaults/secrets":                     &keibi.KeyVaultSecret{},
	"Microsoft.DBforMySQL/servers":                          &keibi.MysqlServer{},
	"Microsoft.Security/pricings":                           &keibi.SecurityCenterSubscriptionPricing{},
	"Microsoft.ClassicNetwork/networkSecurityGroups":        &keibi.NetworkSecurityGroup{},
	"Microsoft.Security/autoProvisioningSettings":           &keibi.SecurityCenterAutoProvisioning{},
	"Microsoft.Network/networkWatchers/flowLogs":            &keibi.NetworkWatcherFlowLog{},
	"Microsoft.Sql/servers/databases":                       &keibi.SqlDatabase{},
	"Microsoft.Cache/redis":                                 &keibi.RedisCache{},
	"Microsoft.HDInsight/clusters":                          &keibi.HdinsightCluster{},
	"Microsoft.KeyVault/vaults":                             &keibi.KeyVault{},
	"Microsoft.StreamAnalytics/streamingJobs":               &keibi.StreamAnalyticsJob{},
	"Microsoft.Resources/links":                             &keibi.ResourceLink{},
	"Microsoft.Compute/virtualMachineScaleSets":             &keibi.ComputeVirtualMachineScaleSet{},
	"Microsoft.Network/applicationGateways":                 &keibi.ApplicationGateway{},
	"Microsoft.Authorization/policyAssignments":             &keibi.PolicyAssignment{},
	"Microsoft.Resources/tenants":                           &keibi.Tenant{},
	"Microsoft.Resources/users":                             &keibi.AdUsers{},
	"Microsoft.Authorization/elevateAccessRoleAssignment":   &keibi.RoleAssignment{},
	"Microsoft.Compute/disks":                               &keibi.ComputeDisk{},
	"Microsoft.Security/locations/jitNetworkAccessPolicies": &keibi.SecurityCenterJitNetworkAccessPolicy{},
	"Microsoft.Network/applicationSecurityGroups":           &keibi.NetworkApplicationSecurityGroups{},
	"Microsoft.RecoveryServices/vaults":                     &keibi.RecoveryServicesVault{},
	"Microsoft.Network/azureFirewalls":                      &keibi.NetworkAzureFirewall{},
	"Microsoft.Network/expressRouteCircuits":                &keibi.ExpressRouteCircuit{},
	"Microsoft.Network/loadBalancers":                       &keibi.LoadBalancers{}, // TODO: Find proper table
	"Microsoft.Network/routeTables":                         &keibi.RouteTables{},
	"Microsoft.Compute/snapshots":                           &keibi.ComputeSnapshots{},
	"Microsoft.Network/virtualNetworkGateways":              &keibi.VirtualNetworkGateway{},
	"Microsoft.Compute/availabilitySets":                    &keibi.ComputeAvailabilitySet{},
	"Microsoft.Compute/diskEncryptionSets":                  &keibi.ComputeDiskEncryptionSet{},
	"Microsoft.Kubernetes/connectedClusters":                &keibi.HybridKubernetesConnectedCluster{},
	"Microsoft.Authorization/policyDefinitions":             &keibi.PolicyDefinition{},
}

type SteampipePlugin string

const (
	SteampipePluginAWS     = "aws"
	SteampipePluginAzure   = "azure"
	SteampipePluginAzureAD = "azuread"
	SteampipePluginUnknown = ""
)

func ExtractPlugin(resourceType string) SteampipePlugin {
	resourceType = strings.ToLower(resourceType)
	if strings.HasPrefix(resourceType, "aws::") {
		return SteampipePluginAWS
	} else if strings.HasPrefix(resourceType, "microsoft") {
		if resourceType == "microsoft.resources/users" {
			return SteampipePluginAzureAD
		}
		return SteampipePluginAzure
	}
	return SteampipePluginUnknown
}

func ExtractTableName(resourceType string) string {
	resourceType = strings.ToLower(resourceType)
	if strings.HasPrefix(resourceType, "aws::") {
		for k, v := range awsMap {
			if resourceType == strings.ToLower(k) {
				return v
			}
		}
	} else {
		for k, v := range azureMap {
			if resourceType == strings.ToLower(k) {
				return v
			}
		}
	}
	return ""
}
