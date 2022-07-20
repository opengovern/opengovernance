package cloudservice

import "gitlab.com/keibiengine/keibi-engine/pkg/source"

type ResourceList struct {
	Provider         source.Type
	ResourceTypeName string
	ServiceNamespace string
}

var azureResourceListCSV = `
Resource type,Resource provider namespace/Entity
Private Link,Microsoft.Network/privateLinkServices
Azure Analysis Services server,Microsoft.AnalysisServices/servers
API management service instance,Microsoft.ApiManagement/service
App Configuration store,Microsoft.AppConfiguration/configurationStores
Policy definition,Microsoft.Authorization/policyDefinitions
Automation account,Microsoft.Automation/automationAccounts
Blueprint,Microsoft.Blueprint/blueprints
Blueprint assignment,Microsoft.Blueprint/blueprints/artifacts
Azure Cache for Redis instance,Microsoft.Cache/Redis
CDN profile,Microsoft.Cdn/profiles
CDN endpoint,Microsoft.Cdn/profiles/endpoints
Azure Cognitive Services,Microsoft.CognitiveServices/accounts
Availability set,Microsoft.Compute/availabilitySets
Cloud service,Microsoft.Compute/cloudServices
Disk encryption set,Microsoft.Compute/diskEncryptionSets
Managed disk (data),Microsoft.Compute/disks
Gallery,Microsoft.Compute/galleries
Snapshot,Microsoft.Compute/snapshots
Virtual machine,Microsoft.Compute/virtualMachines
Virtual machine scale set,Microsoft.Compute/virtualMachineScaleSets
Container instance,Microsoft.ContainerInstance/containerGroups
Container registry,Microsoft.ContainerRegistry/registries
AKS cluster,Microsoft.ContainerService/managedClusters
Azure Databricks workspace,Microsoft.Databricks/workspaces
Azure Data Factory,Microsoft.DataFactory/factories
Data Lake Analytics account,Microsoft.DataLakeAnalytics/accounts
Data Lake Store account,Microsoft.DataLakeStore/accounts
Database Migration Service instance,Microsoft.DataMigration/services
MySQL database,Microsoft.DBforMySQL/servers
PostgreSQL database,Microsoft.DBforPostgreSQL/servers
IoT hub,Microsoft.Devices/IotHubs
Provisioning services,Microsoft.Devices/provisioningServices
Provisioning services certificate,Microsoft.Devices/provisioningServices/certificates
Azure Cosmos DB database,Microsoft.DocumentDB/databaseAccounts/sqlDatabases
Event Grid domain,Microsoft.EventGrid/domains
Event Grid topic,Microsoft.EventGrid/domains/topics
Event Grid subscriptions,Microsoft.EventGrid/eventSubscriptions
Event Hubs namespace,Microsoft.EventHub/namespaces
Event hub,Microsoft.EventHub/namespaces/eventHubs
HDInsight - cluster,Microsoft.HDInsight/clusters
Azure Arc enabled server,Microsoft.HybridCompute/machines
Azure Monitor action group,Microsoft.Insights/actionGroups
Application Insights,Microsoft.Insights/components
Key vault,Microsoft.KeyVault/vaults
Azure Arc enabled Kubernetes cluster,Microsoft.Kubernetes/connectedClusters
Azure Data Explorer cluster,Microsoft.Kusto/clusters
Azure Data Explorer cluster database,Microsoft.Kusto/clusters/databases
Integration account,Microsoft.Logic/integrationAccounts
Logic apps,Microsoft.Logic/workflows
Azure Machine Learning workspace,Microsoft.MachineLearningServices/workspaces
Managed Identity,Microsoft.ManagedIdentity/userAssignedIdentities
Management group,Microsoft.Management/managementGroups
Azure Migrate project,Microsoft.Migrate/assessmentProjects
Application gateway,Microsoft.Network/applicationGateways
Application security group (ASG),Microsoft.Network/applicationSecurityGroups
Firewall,Microsoft.Network/azureFirewalls
Bastion,Microsoft.Network/bastionHosts
Connections,Microsoft.Network/connections
DNS,Microsoft.Network/dnsZones
ExpressRoute circuit,Microsoft.Network/expressRouteCircuits
Firewall policy,Microsoft.Network/firewallPolicies
Web Application Firewall (WAF) policy rule group,Microsoft.Network/firewallPolicies/ruleGroups
Front Door instance,Microsoft.Network/frontDoors
Front Door firewall policy,Microsoft.Network/frontdoorWebApplicationFirewallPolicies
Load balancer (internal),Microsoft.Network/loadBalancers
Load balancer (external),Microsoft.Network/loadBalancers
Load balancer rule,Microsoft.Network/loadBalancers/inboundNatRules
Local network gateway,Microsoft.Network/localNetworkGateways
NAT gateway,Microsoft.Network/natGateways
Network interface (NIC),Microsoft.Network/networkInterfaces
Network security group (NSG),Microsoft.Network/networkSecurityGroups
Network security group (NSG) security rules,Microsoft.Network/networkSecurityGroups/securityRules
Network Watcher,Microsoft.Network/networkWatchers
DNS zone,Microsoft.Network/privateDnsZones
Public IP address,Microsoft.Network/publicIPAddresses
Public IP address prefix,Microsoft.Network/publicIPPrefixes
Route filter,Microsoft.Network/routeFilters
Route table,Microsoft.Network/routeTables
User defined route (UDR),Microsoft.Network/routeTables/routes
Traffic Manager profile,Microsoft.Network/trafficManagerProfiles
Virtual network gateway,Microsoft.Network/virtualNetworkGateways
Virtual network,Microsoft.Network/virtualNetworks
Virtual network subnet,Microsoft.Network/virtualNetworks/subnets
Virtual network peering,Microsoft.Network/virtualNetworks/virtualNetworkPeerings
Virtual WAN,Microsoft.Network/virtualWans
VPN Gateway,Microsoft.Network/vpnGateways
VPN connection,Microsoft.Network/vpnGateways/vpnConnections
VPN site,Microsoft.Network/vpnGateways/vpnSites
Notification Hubs namespace,Microsoft.NotificationHubs/namespaces
Notification Hubs,Microsoft.NotificationHubs/namespaces/notificationHubs
Log Analytics workspace,Microsoft.OperationalInsights/workspaces
Power BI Embedded,Microsoft.PowerBIDedicated/capacities
Azure Purview instance,Microsoft.Purview/accounts
Recovery Services vault,Microsoft.RecoveryServices/vaults
Resource group,Microsoft.Resources/resourceGroups
Azure Cognitive Search,Microsoft.Search/searchServices
Service Bus,Microsoft.ServiceBus/namespaces
Service Bus queue,Microsoft.ServiceBus/namespaces/queues
Service Bus topic,Microsoft.ServiceBus/namespaces/topics
Service endpoint,Microsoft.serviceEndPointPolicies
Service Fabric cluster,Microsoft.ServiceFabric/clusters
SignalR,Microsoft.SignalRService/SignalR
SQL Managed Instance,Microsoft.Sql/managedInstances
Azure SQL Database server,Microsoft.Sql/servers
Azure SQL database,Microsoft.Sql/servers/databases
SQL Server Stretch Database,Microsoft.Sql/servers/databases
Storage account,Microsoft.Storage/storageAccounts
Azure StorSimple,Microsoft.StorSimple/managers
Azure Stream Analytics,Microsoft.StreamAnalytics/cluster
Azure Synapse Analytics Workspaces,Microsoft.Synapse/workspaces
Azure Synapse Analytics Pool,Microsoft.Synapse/workspaces/sqlPools
Time Series Insights environment,Microsoft.TimeSeriesInsights/environments
App Service plan,Microsoft.Web/serverFarms
App Service,Microsoft.Web/sites
Static web app,Microsoft.Web/staticSites
`

var awsResourceListCSV = `
ARN,Resource Type Name
arn:aws:acm-pca:::certificate-authority/,Private Certificate Authority
arn:aws:acm:::certificate/,ACM Certificate
arn:aws:airflow:::environment/,Airflow Environment
arn:aws:amplify:::apps/,Amplify Apps
arn:aws:app-integrations:::data-integration/,Data Integration
arn:aws:app-integrations:::event-integration/,Event Integration
arn:aws:appconfig:::application/,AppConfig Application
arn:aws:appmesh:::mesh/,Application Mesh
arn:aws:apprunner:::connection/,AppRunner Connection
arn:aws:apprunner:::service/,AppRunner Service
arn:aws:appstream:::application/,AppStream Application
arn:aws:appstream:::fleet/,AppStream Fleet
arn:aws:appsync:::apis/,AppSync GraphQL API
arn:aws:athena:::datacatalog/,Athena Datacatalog
arn:aws:backup:::backup-plan:,Backup Plan
arn:aws:backup:::backup-vault:,Backup Vault
arn:aws:batch:::compute-environment/,Batch Compute Environment
arn:aws:batch:::job/,Batch Job
arn:aws:cassandra:::keyspace/,Cassandra Keyspace
arn:aws:cloudformation:::changeSet/,CloudFormation Change
arn:aws:cloudformation:::stack/,CloudFormation Stack
arn:aws:cloudformation:::stackset/,CloudFormation StackSet
arn:aws:cloudfront:::distribution/,CloudFront Distribution
arn:aws:cloudsearch:::domain/,CloudSearch Domain
arn:aws:cloudtrail:::trail/,CloudTrail
arn:aws:cloudwatch:::alarm:,CloudWatch Alarm
arn:aws:comprehend:::document-classifier/,Comprehend Document Classifier
arn:aws:dax:::cache/,DynamoDB Accelerator
arn:aws:directconnect:::dx-gateway/,Direct Connect Gateway
arn:aws:dms:::rep:,Data Migration Replication Instance
arn:aws:dynamodb:::global-table/,DynamoDB Global Table
arn:aws:dynamodb:::table/,DynamoDB Table
arn:aws:ec2:::CarrierGateway/,EC2 Carrier Gateway
arn:aws:ec2:::CustomerGateway/,EC2 Customer Gateway
arn:aws:ec2:::DHCPOptions/,EC2 DHCP Options
arn:aws:ec2:::EC2Fleet/,EC2 Fleet
arn:aws:ec2:::EgressOnlyInternetGateway/,EC2 EgressOnly Internet Gateway
arn:aws:ec2:::EIP/,EC2 Elastic IP
arn:aws:ec2:::FlowLog/,EC2 Flow Log
arn:aws:ec2:::Host/,EC2 Host
arn:aws:ec2:::Instance/,EC2 Instance
arn:aws:ec2:::instance/,SSM Instance
arn:aws:ec2:::InternetGateway/,EC2 InternetGateway
arn:aws:ec2:::IPAM/,EC2 IPAM
arn:aws:ec2:::KeyPair/,EC2 KeyPair
arn:aws:ec2:::LocalGatewayRoute/,EC2 LocalGatewayRoute
arn:aws:ec2:::NatGateway/,EC2 NAT Gateway
arn:aws:ec2:::NetworkAcl/,EC2 Network ACL
arn:aws:ec2:::NetworkInterface/,EC2 Network Interface
arn:aws:ec2:::PlacementGroup/,EC2 PlacementGroup
arn:aws:ec2:::Route/,EC2 Route
arn:aws:ec2:::RouteTable/,EC2 RouteTable
arn:aws:ec2:::SecurityGroup/,EC2 SecurityGroup
arn:aws:ec2:::snapshot/,EBS Snapshot
arn:aws:ec2:::SpotFleet/,EC2 SpotFleet
arn:aws:ec2:::Subnet/,EC2 Subnet
arn:aws:ec2:::TransitGateway/,EC2 Transit Gateway
arn:aws:ec2:::Volume/,EC2 Volume
arn:aws:ec2:::VPC/,EC2 VPC
arn:aws:ec2:::VPNGateway/,EC2 VPN Gateway
arn:aws:ecr-public:::registry/,ECR Public Registry
arn:aws:ecr-public:::repository/,ECR Public Repo
arn:aws:ecr:::repository/,ECR Repositor
arn:aws:ecs:::cluster/,ECS Cluster
arn:aws:ecs:::service/,ECS Service
arn:aws:ecs:::task/,ECS Task
arn:aws:ecs:::task/,SSM Task
arn:aws:eks:::cluster/,EKS Clusters
arn:aws:eks:::nodegroup/,EKS Node Group
arn:aws:elasticache:::cluster:,ElastiCache Cluster
arn:aws:elasticbeanstalk:::application/,Elastic Bean Stalk Application
arn:aws:elasticbeanstalk:::environment/,Elastic Bean Stalk Environment
arn:aws:elasticfilesystem:::file-system/,EFS FileSystem
arn:aws:elasticloadbalancing:::loadbalancer/,Elastic Load Balancer
arn:aws:elasticloadbalancing:::loadbalancer/app/,ELB (Application)
arn:aws:elasticloadbalancing:::loadbalancer/net/,ELB (Network)
arn:aws:elasticmapreduce:::cluster/,EMR Cluster
arn:aws:elasticmapreduce:::Studio/,EMR Studio
arn:aws:es:::domain/,OpenSearch Domain
arn:aws:firehose:::deliverystream/,Kenesis Firehose Delivery Stream
arn:aws:fms:::policy/,Firewall Manager Resource
arn:aws:fsx:::file-system/,FSx File System
arn:aws:fsx:::snapshot/,FSx Snapshot
arn:aws:fsx:::storage-virtual-machine/,FSx Storage VM
arn:aws:fsx:::volume/,FSx Snapshot
arn:aws:glacier:::vaults/,S3 Glacier Vault
arn:aws:glue:::database/,Glue Database
arn:aws:guardduty:::detector/,GuardDuty Detector
arn:aws:iam:::group/,IAM Group
arn:aws:iam:::policy/,IAM Policy
arn:aws:iam:::role/,IAM Role
arn:aws:iam:::user/,IAM User
arn:aws:imagebuilder:::image/,Image Builder Image
arn:aws:kafka:::cluster/,Kafka Cluster
arn:aws:kinesis:::stream/,Kenesis Stream
arn:aws:kinesisanalytics:::application/,Kenesis Anaytics App
arn:aws:kinesisvideo:::stream/,Kenesis Video Stream
arn:aws:kms:::key/,KMS Key
arn:aws:kms:::key/,KMS Key
arn:aws:lambda:::function:,Lambda Function
arn:aws:logs:::log-group:,CloudWatch Log Group
arn:aws:memorydb:::cluster/,MemoryDB Cluster
arn:aws:memorydb:::snapshot/,MemoryDB Snapshot
arn:aws:mobiletargeting:::apps/,Pinpoint App
arn:aws:mq:::broker:,MQ Broker
arn:aws:network-firewall:::firewall/,Network Firewall
arn:aws:opsworks-cm:::server/,OpsWork CM Server
arn:aws:opsworks:::stack/,OpsWork Stack
arn:aws:organizations::${MasterAccountId}:account/,Organization Account
arn:aws:organizations::${MasterAccountId}:organization/,Organization
arn:aws:organizations::${MasterAccountId}:ou/,Organization OU
arn:aws:organizations::${MasterAccountId}:policy/,Organization Policy
arn:aws:polly:::lexicon/,Polly Lexicon
arn:aws:qldb:::ledger/,QLDB Ledger
arn:aws:quicksight:::namespace/,QuickSight Namespace
arn:aws:quicksight:::namespace/,QuickSight Namespace
arn:aws:rds:::cluster-snapshot:,RDS Cluster Snapshot
arn:aws:rds:::cluster:,RDS Cluster
arn:aws:rds:::db-proxy:,RDS DB Proxy
arn:aws:rds:::db:,RDS DB
arn:aws:rds:::global-cluster:,RDS Global Cluster
arn:aws:rds:::snapshot:,RDS Snapshot
arn:aws:rds:::subgrp:,RDS Snubnet Group
arn:aws:redshift:::cluster:,Redshift Cluster
arn:aws:route53:::hostedzone/,Route 53 Hosted Zone
arn:aws:s3:::${BucketName},S3 Bucket
arn:aws:sdb:::domain/,SimpleDB Domain
arn:aws:ses:::configuration-set/,SES Configuration Set
arn:aws:shield:::,Shield Resource
arn:aws:sns:::,SNS Topic
arn:aws:sqs:::,SQS Queue
arn:aws:ssm:::document/,SSM Document
arn:aws:states:::stateMachine:,Step Function State Machine
arn:aws:storagegateway:::gateway/,Storage Gateway
arn:aws:storagegateway:::share/,Storage Gateways Share
arn:aws:storagegateway:::tape/,Storage Gateway Tape
arn:aws:swf:::/domain/,SWF Domain
arn:aws:timestream:::database/,timestream
arn:aws:waf-regional:::,WAF Regional
arn:aws:waf:::,WAF Resource
arn:aws:wafv2:::,WAF V2 Resource
arn:aws:workmail:::organization/,Workmail Organization
arn:aws:workspaces:::directory/,Workspace Directory
arn:aws:workspaces:::workspace/,Workspace
arn:aws:workspaces:::workspacebundle/,Workspace Bundle
`
