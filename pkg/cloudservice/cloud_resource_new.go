package cloudservice

import "gitlab.com/keibiengine/keibi-engine/pkg/source"

type CloudResource struct {
	Cloud                     source.Type
	CloudService              string
	ResourceTypeName          string
	ResourceProviderNamespace string
}

var cloudResourcesCSV = `
Cloud,Cloud Service,Resource Type,Resource provider namespace
Azure,Application Gateway,Application gateway,Microsoft.Network/applicationGateways
Azure,Virtual Network (vNet),Application security group (ASG),Microsoft.Network/applicationSecurityGroups
Azure,BastionHost,Bastion,Microsoft.Network/bastionHosts
Azure,Azure CDN,CDN profile,Microsoft.Cdn/profiles
Azure,Azure CDN,CDN endpoint,Microsoft.Cdn/profiles/endpoints
Azure,Virtual Network (vNet),Connections,Microsoft.Network/connections
Azure,Azure DNS,DNS,Microsoft.Network/dnsZones
Azure,Azure DNS,DNS zone,Microsoft.Network/privateDnsZones
Azure,Azure Firewall,Firewall,Microsoft.Network/azureFirewalls
Azure,Azure Firewall,Firewall policy,Microsoft.Network/firewallPolicies
Azure,ExpressRoute,ExpressRoute circuit,Microsoft.Network/expressRouteCircuits
Azure,Frontdoor,Front Door instance,Microsoft.Network/frontDoors
Azure,Frontdoor,Front Door firewall policy,Microsoft.Network/frontdoorWebApplicationFirewallPolicies
Azure,Load Balancer,Load balancer (internal),Microsoft.Network/loadBalancers
Azure,Load Balancer,Load balancer (external),Microsoft.Network/loadBalancers
Azure,Load Balancer,Load balancer rule,Microsoft.Network/loadBalancers/inboundNatRules
Azure,Virtual Network (vNet),Local network gateway,Microsoft.Network/localNetworkGateways
Azure,Virtual Network (vNet),NAT gateway,Microsoft.Network/natGateways
Azure,Virtual Network (vNet),Network interface (NIC),Microsoft.Network/networkInterfaces
Azure,Virtual Network (vNet),Network security group (NSG),Microsoft.Network/networkSecurityGroups
Azure,Virtual Network (vNet),Network Watcher,Microsoft.Network/networkWatchers
Azure,Virtual Network (vNet),Private Link,Microsoft.Network/privateLinkServices
Azure,Virtual Network (vNet),Public IP address,Microsoft.Network/publicIPAddresses
Azure,Virtual Network (vNet),Public IP address prefix,Microsoft.Network/publicIPPrefixes
Azure,Virtual Network (vNet),Route filter,Microsoft.Network/routeFilters
Azure,Virtual Network (vNet),Route table,Microsoft.Network/routeTables
Azure,Virtual Network (vNet),Service endpoint,Microsoft.serviceEndPointPolicies
Azure,Traffic Manager,Traffic Manager profile,Microsoft.Network/trafficManagerProfiles
Azure,Virtual Network (vNet),User defined route (UDR),Microsoft.Network/routeTables/routes
Azure,Virtual Network (vNet),Virtual network,Microsoft.Network/virtualNetworks
Azure,Virtual Network (vNet),Virtual network subnet,Microsoft.Network/virtualNetworks/subnets
Azure,Virtual Network (vNet),Virtual WAN,Microsoft.Network/virtualWans
Azure,Site-to-Site VPN,VPN Gateway,Microsoft.Network/vpnGateways
Azure,Site-to-Site VPN,VPN connection,Microsoft.Network/vpnGateways/vpnConnections
Azure,Site-to-Site VPN,VPN site,Microsoft.Network/vpnGateways/vpnSites
Azure,Virtual Network (vNet),Virtual network gateway,Microsoft.Network/virtualNetworkGateways
Azure,WAF,Web Application Firewall (WAF) policy,Microsoft.Network/firewallPolicies
Azure,WAF,Web Application Firewall (WAF) policy rule group,Microsoft.Network/firewallPolicies/ruleGroups
Azure,App Service,App Service environment,Microsoft.Web/sites
Azure,App Service,App Service plan,Microsoft.Web/serverFarms
Azure,Virtual Machine,Availability set,Microsoft.Compute/availabilitySets
Azure,Virtual Machine,Azure Arc enabled server,Microsoft.HybridCompute/machines
Azure,Containers,Azure Arc enabled Kubernetes cluster,Microsoft.Kubernetes/connectedClusters
Azure,PaaS,Cloud service,Microsoft.Compute/cloudServices
Azure,Virtual Machine,Disk encryption set,Microsoft.Compute/diskEncryptionSets
Azure,App Service,Function app,Microsoft.Web/sites
Azure,Other,Gallery,Microsoft.Compute/galleries
Azure,Disk Storage,Managed disk,Microsoft.Compute/disks
Azure,Notification,Notification Hubs,Microsoft.NotificationHubs/namespaces/notificationHubs
Azure,Notification,Notification Hubs namespace,Microsoft.NotificationHubs/namespaces
Azure,Disk Storage,Snapshot,Microsoft.Compute/snapshots
Azure,Azure App,Static web app,Microsoft.Web/staticSites
Azure,Virtual Machine,Virtual machine,Microsoft.Compute/virtualMachines
Azure,Virtual Machine ScaleSet,Virtual machine scale set,Microsoft.Compute/virtualMachineScaleSets
Azure,Storage Account,VM storage account,Microsoft.Storage/storageAccounts
Azure,Azure App,Web app,Microsoft.Web/sites
Azure,Azure Kubernetes Service,AKS cluster,Microsoft.ContainerService/managedClusters
Azure,Container Register,Container registry,Microsoft.ContainerRegistry/registries
Azure,Azure Container,Container instance,Microsoft.ContainerInstance/containerGroups
Azure,Service Fabric,Service Fabric cluster,Microsoft.ServiceFabric/clusters
Azure,DocumentDB,Azure Cosmos DB database,Microsoft.DocumentDB/databaseAccounts/sqlDatabases
Azure,Azure Cache,Azure Cache for Redis instance,Microsoft.Cache/Redis
Azure,Azure SQL,Azure SQL Database server,Microsoft.Sql/servers
Azure,Azure SQL,Azure SQL database,Microsoft.Sql/servers/databases
Azure,Azure Synapse,Azure Synapse Analytics Workspaces,Microsoft.Synapse/workspaces
Azure,Azure Synapse,Azure Synapse Analytics SQL Pools,Microsoft.Synapse/workspaces/sqlPools
Azure,DB for MySQL,MySQL database,Microsoft.DBforMySQL/servers
Azure,DB for PostgreSQL,PostgreSQL database,Microsoft.DBforPostgreSQL/servers
Azure,Database,SQL Server Stretch Database,Microsoft.Sql/servers/databases
Azure,Database Instance,SQL Managed Instance,Microsoft.Sql/managedInstances
Azure,Azure Migrate,Azure Migrate project,Microsoft.Migrate/assessmentProjects
Azure,Data Migration Service,Database Migration Service instance,Microsoft.DataMigration/services
Azure,Recovery Services,Recovery Services vault,Microsoft.RecoveryServices/vaults
Azure,Automation,Automation account,Microsoft.Automation/automationAccounts
Azure,Application Insights,Application Insights,Microsoft.Insights/components
Azure,Application Insights,Azure Monitor action group,Microsoft.Insights/actionGroups
Azure,Purview,Azure Purview instance,Microsoft.Purview/accounts
Azure,Blueprint,Blueprint,Microsoft.Blueprint/blueprints
Azure,Blueprint,Blueprint assignment,Microsoft.Blueprint/blueprints/artifacts
Azure,Key vault,Vault,Microsoft.KeyVault/vaults
Azure,Operational Insight,Log Analytics workspace,Microsoft.OperationalInsights/workspaces
Azure,Azure App,Integration account,Microsoft.Logic/integrationAccounts
Azure,Azure App,Logic apps,Microsoft.Logic/workflows
Azure,,Service Bus,Microsoft.ServiceBus/namespaces
Azure,,Service Bus queue,Microsoft.ServiceBus/namespaces/queues
Azure,,Service Bus topic,Microsoft.ServiceBus/namespaces/topics
Azure,App Configuration,App Configuration store,Microsoft.AppConfiguration/configurationStores
Azure,SignalR,SignalR,Microsoft.SignalRService/SignalR
Azure,Azure Analysis Services,Azure Analysis Services server,Microsoft.AnalysisServices/servers
Azure,Databrics,Azure Databricks workspace,Microsoft.Databricks/workspaces
Azure,Azure Stream Analytics,Azure Stream Analytics Cluster,Microsoft.StreamAnalytics/cluster
Azure,Kusto,Azure Data Explorer cluster,Microsoft.Kusto/clusters
Azure,Kusto,Azure Data Explorer cluster database,Microsoft.Kusto/clusters/databases
Azure,Data Factory,Azure Data Factory,Microsoft.DataFactory/factories
Azure,Data Lake,Data Lake Store account,Microsoft.DataLakeStore/accounts
Azure,Data Lake,Data Lake Analytics account,Microsoft.DataLakeAnalytics/accounts
Azure,Event Hub,Event Hubs namespace,Microsoft.EventHub/namespaces
Azure,Event Hub,Event hub,Microsoft.EventHub/namespaces/eventHubs
Azure,Event Grid,Event Grid domain,Microsoft.EventGrid/domains
Azure,Event Grid,Event Grid subscriptions,Microsoft.EventGrid/eventSubscriptions
Azure,Event Grid,Event Grid topic,Microsoft.EventGrid/domains/topics
Azure,HDInsight,HDInsight  cluster,Microsoft.HDInsight/clusters
Azure,Microsoft Devices,IoT hub,Microsoft.Devices/IotHubs
Azure,Microsoft Devices,Provisioning services,Microsoft.Devices/provisioningServices
Azure,Microsoft Devices,Provisioning services certificate,Microsoft.Devices/provisioningServices/certificates
Azure,PowerBI,Power BI Embedded,Microsoft.PowerBIDedicated/capacities
Azure,Time Series Insights,Time Series Insights environment,Microsoft.TimeSeriesInsights/environments
Azure,Microsoft Search,Azure Cognitive Search,Microsoft.Search/searchServices
Azure,Cognitive Search,Azure Cognitive Services,Microsoft.CognitiveServices/accounts
Azure,Azure Machine Learning,Azure Machine Learning workspace,Microsoft.MachineLearningServices/workspaces
Azure,Storage,Storage account,Microsoft.Storage/storageAccounts
Azure,Stor Simple,Azure StorSimple,Microsoft.StorSimple/managers
Azure,API Management,API management service instance,Microsoft.ApiManagement/service
Azure,Managed Identity,Managed Identity,Microsoft.ManagedIdentity/userAssignedIdentities
Azure,Management Group,Management group,Microsoft.Management/managementGroups
Azure,Azure Policy,Policy definition,Microsoft.Authorization/policyDefinitions
Azure,Resource Group,Resource group,Microsoft.Resources/resourceGroups
Azure,Virtual Private Cloud (VPC),DHCP Option Set,arn:${Partition}:ec2:${Region}:${Account}:dhcp-options/${DhcpOptionsId}
AWS,Virtual Private Cloud (VPC),Internet Gateway (Egress Only),arn:${Partition}:ec2:${Region}:${Account}:egress-only-internet-gateway/${EgressOnlyInternetGatewayId}
AWS,Virtual Private Cloud (VPC),Elastic IP,arn:${Partition}:ec2:${Region}:${Account}:elastic-ip/${AllocationId}
AWS,Virtual Private Cloud (VPC),Internet Gateway,arn:${Partition}:ec2:${Region}:${Account}:internet-gateway/${InternetGatewayId}
AWS,Virtual Private Cloud (VPC),VPC Flow Log,arn:${Partition}:ec2:${Region}:${Account}:vpc-flow-log/${VpcFlowLogId}
AWS,Virtual Private Cloud (VPC),VPC Peering Connection,arn:${Partition}:ec2:${Region}:${Account}:vpc-peering-connection/${VpcPeeringConnectionId}
AWS,Virtual Private Cloud (VPC),VPC (Virtual Network),arn:${Partition}:ec2:${Region}:${Account}:vpc/${VpcId}
AWS,Virtual Private Cloud (VPC),Network Interface,arn:${Partition}:ec2:${Region}:${Account}:network-interface/${NetworkInterfaceId}
AWS,Virtual Private Cloud (VPC),Security Group,arn:${Partition}:ec2:${Region}:${Account}:security-group/${SecurityGroupId}
AWS,Virtual Private Cloud (VPC),Network ACL,arn:${Partition}:ec2:${Region}:${Account}:network-acl/${NaclId}
AWS,Virtual Private Cloud (VPC),Route Table,arn:${Partition}:ec2:${Region}:${Account}:route-table/${RouteTableId}
AWS,Virtual Private Cloud (VPC),NAT Gateway,arn:${Partition}:ec2:${Region}:${Account}:natgateway/${NatGatewayId}
AWS,Virtual Private Cloud (VPC),Security Group Rule,arn:${Partition}:ec2:${Region}:${Account}:security-group-rule/${SecurityGroupRuleId}
AWS,Virtual Private Cloud (VPC),Subnet,arn:${Partition}:ec2:${Region}:${Account}:subnet/${SubnetId}
AWS,VPC IPAM,IPAM Pool,arn:${Partition}:ec2::${Account}:ipam-pool/${IpamPoolId}
AWS,VPC IPAM,IPAM,arn:${Partition}:ec2::${Account}:ipam/${IpamId}
AWS,Transit Gateway,Transit Gateway,arn:${Partition}:ec2:${Region}:${Account}:transit-gateway/${TransitGatewayId}
AWS,Transit Gateway,Transit Gateway Route Table,arn:${Partition}:ec2:${Region}:${Account}:transit-gateway-route-table/${TransitGatewayRouteTableId}
AWS,Amazon EC2 Instance,EC2 Capacity Reservation,arn:${Partition}:ec2:${Region}:${Account}:capacity-reservation/${CapacityReservationId}
AWS,Amazon EC2 Instance,EC2 Capacity Reservation Fleet,arn:${Partition}:ec2:${Region}:${Account}:capacity-reservation-fleet/${CapacityReservationFleetId}
AWS,Amazon EC2 Instance,EC2 Fleet,arn:${Partition}:ec2:${Region}:${Account}:fleet/${FleetId}
AWS,Amazon EC2 Instance,EC2 Host,arn:${Partition}:ec2:${Region}:${Account}:host-reservation/${HostReservationId}
AWS,Amazon EC2 Instance,EC2 Instance,arn:${Partition}:ec2:${Region}:${Account}:instance/${InstanceId}
AWS,Amazon EC2 Instance,Access Key Pair,arn:${Partition}:ec2:${Region}:${Account}:key-pair/${KeyPairName}
AWS,Amazon EC2 Instance,EBS Volume,arn:${Partition}:ec2:${Region}:${Account}:volume/${VolumeId}
AWS,Amazon EC2 Instance,EC2 Placement Group,arn:${Partition}:ec2:${Region}:${Account}:placement-group/${PlacementGroupName}
AWS,Amazon EC2 Instance,EC2 Image (AMI),arn:${Partition}:ec2:${Region}::image/${ImageId}
AWS,Amazon EC2 Instance,Reserved Instances,arn:${Partition}:ec2:${Region}:${Account}:reserved-instances/${ReservationId}
AWS,Amazon EC2 Instance,Disk Snapshot,arn:${Partition}:ec2:${Region}::snapshot/${SnapshotId}
AWS,Amazon Kenesis,Kenesis Stream,arn:${Partition}:kinesis:${Region}:${Account}:stream/${StreamName}
AWS,Amazon RDS,RDS Cluster,arn:${Partition}:rds:${Region}:${Account}:cluster:${DbClusterInstanceName}
AWS,Amazon RDS,RDS Database,arn:${Partition}:rds:${Region}:${Account}:db:${DbInstanceName}
AWS,Amazon RDS,RDS Snapshot,arn:${Partition}:rds:${Region}:${Account}:snapshot:${SnapshotName}
AWS,Amazon RDS,RDS Global Cluster,arn:${Partition}:rds::${Account}:global-cluster:${GlobalCluster}
AWS,Amazon OpenSearch Service,OpenSearch Domain,arn:${Partition}:es:${Region}:${Account}:domain/${DomainName}
AWS,Amazon Redshift,Redshift Snapshot,arn:${Partition}:redshift:${Region}:${Account}:snapshot:${ClusterName}/${SnapshotName}
AWS,Amazon Redshift,Redshift Cluster,arn:${Partition}:redshift:${Region}:${Account}:cluster:${ClusterName}
AWS,Amazon Redshift,Redshift Database Name,arn:${Partition}:redshift:${Region}:${Account}:dbname:${ClusterName}/${DbName}
AWS,Amazon Redshift Serverless,Redshift Serverless Namespace,arn:${Partition}:redshift-serverless:${Region}:${Account}:namespace/${NamespaceId}
AWS,Amazon Redshift Serverless,Redshift Serverless Snapshot,arn:${Partition}:redshift-serverless:${Region}:${Account}:snapshot/${SnapshotId}
AWS,Amazon API Gateway,API Gateway (v2) API,arn:${Partition}:apigateway:${Region}::/apis/${ApiId}
AWS,Amazon API Gateway,API Gateway (v1) RestAPI,arn:${Partition}:apigateway:${Region}::/restapis/${ApiId}
AWS,Amazon Elastic Container Service (ECS),ECS Cluster,arn:${Partition}:ecs:${Region}:${Account}:cluster/${ClusterName}
AWS,Amazon Elastic Container Service (ECS),ECS Container Instance,arn:${Partition}:ecs:${Region}:${Account}:container-instance/${ClusterName}/${ContainerInstanceId}
AWS,Amazon Elastic Container Service (ECS),ECS Service,arn:${Partition}:ecs:${Region}:${Account}:service/${ClusterName}/${ServiceName}
AWS,Amazon Elastic Container Service (ECS),ECS Task,arn:${Partition}:ecs:${Region}:${Account}:task/${ClusterName}/${TaskId}
AWS,Amazon Elastic Container Service (ECS),ECS Task Set,arn:${Partition}:ecs:${Region}:${Account}:task-set/${ClusterName}/${ServiceName}/${TaskSetId}
AWS,Amazon Elastic Container Registry (ECR),ECR Repository,arn:${Partition}:ecr:${Region}:${Account}:repository/${RepositoryName}
AWS,Amazon Elastic Container Registry (ECR),ECR Public Repository,arn:${Partition}:ecr-public::${Account}:repository/${RepositoryName}
AWS,Amazon Elastic Container Registry (ECR),ECR Public Refistry,arn:${Partition}:ecr-public::${Account}:registry/${RegistryId}
AWS,Amazon Elastic Kubernetes Service (EKS),EKS Cluster,arn:${Partition}:eks:${Region}:${Account}:cluster/${ClusterName}
AWS,Amazon Elastic Kubernetes Service (EKS),EKS Node Group,arn:${Partition}:eks:${Region}:${Account}:nodegroup/${ClusterName}/${NodegroupName}/${UUID}
AWS,Amazon Elastic File System (EFS),EFS File System,arn:${Partition}:elasticfilesystem:${Region}:${Account}:file-system/${FileSystemId}
AWS,Amazon EC2 Autoscaling,EC2 Auto Scaling Group,arn:${Partition}:autoscaling:${Region}:${Account}:autoScalingGroup:${GroupId}:autoScalingGroupName/${GroupFriendlyName}
AWS,Amazon ElastiCache,ElastiCache Cluster,arn:${Partition}:elasticache:${Region}:${Account}:cluster:${CacheClusterId}
AWS,Amazon ElastiCache,ElastiCache Global Replication Group,arn:${Partition}:elasticache::${Account}:globalreplicationgroup:${GlobalReplicationGroupId}
AWS,Amazon EventBridge,EventBridge Event Bus,arn:${Partition}:events:${Region}:${Account}:event-bus/${EventBusName}
AWS,AWS Lambda,Lambda Function,arn:${Partition}:lambda:${Region}:${Account}:function:${FunctionName}
AWS,AWS Lambda,Lambda Function Version,arn:${Partition}:lambda:${Region}:${Account}:function:${FunctionName}:${Version}
AWS,AWS Elastic Beanstalk,Elastic Beanstalk Application,arn:${Partition}:elasticbeanstalk:${Region}:${Account}:application/${ApplicationName}
AWS,AWS Elastic Beanstalk,Elastic Beanstalk Environment,arn:${Partition}:elasticbeanstalk:${Region}:${Account}:environment/${ApplicationName}/${EnvironmentName}
AWS,AWS Elastic Beanstalk,Elastic Beanstalk Platform,arn:${Partition}:elasticbeanstalk:${Region}::platform/${PlatformNameWithVersion}
AWS,Elastic Load Balancing (ELB),Elastic Load Balancer,arn:${Partition}:elasticloadbalancing:${Region}:${Account}:loadbalancer/${LoadBalancerName}
AWS,Elastic Load Balancing (ELB),Network Load Balancer,arn:${Partition}:elasticloadbalancing:${Region}:${Account}:loadbalancer/net/${LoadBalancerName}/${LoadBalancerId}
AWS,Elastic Load Balancing (ELB),Application Load Balancer (ALB),arn:${Partition}:elasticloadbalancing:${Region}:${Account}:loadbalancer/app/${LoadBalancerName}/${LoadBalancerId}
AWS,Elastic Load Balancing (ELB),ALB Listener,arn:${Partition}:elasticloadbalancing:${Region}:${Account}:listener/app/${LoadBalancerName}/${LoadBalancerId}/${ListenerId}
AWS,Elastic Load Balancing (ELB),NLB Listener,arn:${Partition}:elasticloadbalancing:${Region}:${Account}:listener/net/${LoadBalancerName}/${LoadBalancerId}/${ListenerId}
AWS,Elastic Load Balancing (ELB),ALB Listener Rule,arn:${Partition}:elasticloadbalancing:${Region}:${Account}:listener-rule/app/${LoadBalancerName}/${LoadBalancerId}/${ListenerId}/${ListenerRuleId}
AWS,Elastic Load Balancing (ELB),NLB Listener Rule,arn:${Partition}:elasticloadbalancing:${Region}:${Account}:listener-rule/net/${LoadBalancerName}/${LoadBalancerId}/${ListenerId}/${ListenerRuleId}
AWS,Elastic Load Balancing (ELB),Load Balancer Target Group,arn:${Partition}:elasticloadbalancing:${Region}:${Account}:targetgroup/${TargetGroupName}/${TargetGroupId}
AWS,Amazon FSx,FSx File System,arn:${Partition}:fsx:${Region}:${Account}:file-system/${FileSystemId}
AWS,Amazon FSx,FSx Backup,arn:${Partition}:fsx:${Region}:${Account}:backup/${BackupId}
AWS,Amazon FSx,FSx Storage Virtual Machine,arn:${Partition}:fsx:${Region}:${Account}:storage-virtual-machine/${FileSystemId}/${StorageVirtualMachineId}
AWS,Amazon FSx,FSx Task,arn:${Partition}:fsx:${Region}:${Account}:task/${TaskId}
AWS,Amazon FSx,FSx Volume,arn:${Partition}:fsx:${Region}:${Account}:volume/${FileSystemId}/${VolumeId}
AWS,Amazon FSx,FSx Snapshot,arn:${Partition}:fsx:${Region}:${Account}:snapshot/${VolumeId}/${SnapshotId}
AWS,Amazon DynamoDB,DynamoDB Index,arn:${Partition}:dynamodb:${Region}:${Account}:table/${TableName}/index/${IndexName}
AWS,Amazon DynamoDB,DynamoDB Stream,arn:${Partition}:dynamodb:${Region}:${Account}:table/${TableName}/stream/${StreamLabel}
AWS,Amazon DynamoDB,DynamoDB Table,arn:${Partition}:dynamodb:${Region}:${Account}:table/${TableName}
AWS,Amazon DynamoDB,DynamoDB Backup,arn:${Partition}:dynamodb:${Region}:${Account}:table/${TableName}/backup/${BackupName}
AWS,Amazon DynamoDB,DynamoDB Global Table,arn:${Partition}:dynamodb::${Account}:global-table/${GlobalTableName}
AWS,Amazon Keyspaces (for Apache Cassandra),Keyspace,arn:${Partition}:cassandra:${Region}:${Account}:/keyspace/${KeyspaceName}/
AWS,Amazon Keyspaces (for Apache Cassandra),Keyspaces Table,arn:${Partition}:cassandra:${Region}:${Account}:/keyspace/${KeyspaceName}/table/${TableName}
AWS,Amazon MQ,MQ Broker,arn:${Partition}:mq:${Region}:${Account}:broker:${BrokerId}
AWS,Amazon Neptune,Neptune Database,arn:${Partition}:neptune-db:${Region}:${Account}:${RelativeId}/database
AWS,Amazon MemoryDB,MemoryDB Cluster,arn:${Partition}:memorydb:${Region}:${Account}:cluster/${ClusterName}
AWS,Amazon Managed Workflows for Apache Airflow (MWAA),Airflow Environment,arn:${Partition}:airflow:${Region}:${Account}:environment/${EnvironmentName}
AWS,Amazon Managed Streaming for Apache Kafka (MSK),Kafka Cluster,arn:${Partition}:kafka:${Region}:${Account}:cluster/${ClusterName}/${Uuid}
AWS,Amazon Managed Streaming for Apache Kafka (MSK),Kafka Topic,arn:${Partition}:kafka:${Region}:${Account}:topic/${ClusterName}/${ClusterUuid}/${TopicName}
AWS,Amazon Managed Service for Prometheus,Prometheus Workspace,arn:${Partition}:aps:${Region}:${Account}:workspace/${WorkspaceId}
AWS,Amazon Managed Grafana,Grafana Workspace,arn:${Partition}:grafana:${Region}:${Account}:/workspaces/${ResourceId}
AWS,Amazon Simple Storage Service (S3),S3 Bucket,arn:${Partition}:s3:::${BucketName}
AWS,Amazon S3 Glacier,S3 Glacier Vault,arn:${Partition}:glacier:${Region}:${Account}:vaults/${VaultName}
AWS,Amazon CloudWatch,CloudWatch Alarm,arn:${Partition}:cloudwatch:${Region}:${Account}:alarm:${AlarmName}
AWS,Amazon AppStream 2.0,AppStream 2.0 Application,arn:${Partition}:appstream:${Region}:${Account}:application/${ApplicationName}
AWS,Amazon AppStream 2.0,AppStream 2.0 Stack,arn:${Partition}:appstream:${Region}:${Account}:stack/${StackName}
AWS,Amazon AppStream 2.0,AppSteam 2.0 Fleet,arn:${Partition}:appstream:${Region}:${Account}:fleet/${FleetName}
AWS,Amazon Simple Email Service (SES),SES Configuration Set,arn:${Partition}:ses:${Region}:${Account}:configuration-set/${ConfigurationSetName}
AWS,Amazon Simple Email Service (SES),SES Identity,arn:${Partition}:ses:${Region}:${Account}:identity/${IdentityName}
AWS,Amazon Simple Notification Service (SNS),SNS Topic,arn:${Partition}:sns:${Region}:${Account}:${TopicName}
AWS,Amazon Simple Queue Service (SQS),SQS Queue,arn:${Partition}:sqs:${Region}:${Account}:${QueueName}
AWS,Amazon WorkSpaces,Workspace,arn:${Partition}:workspaces:${Region}:${Account}:workspace/${WorkspaceId}
AWS,Amazon WorkSpaces,Workspace Bundle,arn:${Partition}:workspaces:${Region}:${Account}:workspacebundle/${BundleId}
AWS,AWS Backup,Backup Vault,arn:${Partition}:backup:${Region}:${Account}:backup-vault:${BackupVaultName}
AWS,AWS Backup,Backup Plan,arn:${Partition}:backup:${Region}:${Account}:backup-plan:${BackupPlanId}
AWS,AWS Batch,Batch Compute Environment,arn:${Partition}:batch:${Region}:${Account}:compute-environment/${ComputeEnvironmentName}
AWS,AWS Batch,Batch Job,arn:${Partition}:batch:${Region}:${Account}:job/${JobId}
AWS,CloudFront,CloudFront Distribution,AWS::CloudFront::Distribution
AWS,CloudFront,CloudFront Origin,AWS::CloudFront::OriginAccessControl
AWS,AWS Direct Connect,Direct Connect Connection,arn:${Partition}:directconnect:${Region}:${Account}:dxcon/${ConnectionId}
AWS,AWS Direct Connect,Direct Connect Gateway,arn:${Partition}:directconnect::${Account}:dx-gateway/${DirectConnectGatewayId}
AWS,EC2 Image Builder,Golden Image,arn:${Partition}:imagebuilder:${Region}:${Account}:image/${ImageName}
AWS,Private Link,VPC Endpoint,arn:${Partition}:ec2:${Region}:${Account}:vpc-endpoint/${VpcEndpointId}
AWS,Private Link,VPC Endpoint Service,arn:${Partition}:ec2:${Region}:${Account}:vpc-endpoint-service/${VpcEndpointServiceId}
AWS,Route53,Route53 Hosted Zone,arn:${Partition}:route53:::hostedzone/${Id}
AWS,Identity And Access Management (IAM),IAM Group,arn:${Partition}:iam::${Account}:group/${GroupNameWithPath}
AWS,Identity And Access Management (IAM),IAM Role,arn:${Partition}:iam::${Account}:role/${RoleNameWithPath}
AWS,Identity And Access Management (IAM),IAM User,arn:${Partition}:iam::${Account}:user/${UserNameWithPath}
AWS,Identity And Access Management (IAM),IAM Policy,arn:${Partition}:iam::${Account}:policy/${PolicyNameWithPath}
AWS,AWS Certificate Manager,ACM Certificate,arn:${Partition}:acm:${Region}:${Account}:certificate/${CertificateId}
AWS,AWS Private Certificate Authority,ACM Private CA,arn:${Partition}:acm-pca:${Region}:${Account}:certificate-authority/${CertificateAuthorityId}
AWS,AWS CloudFormation,CloudFormation Stack,arn:${Partition}:cloudformation:${Region}:${Account}:stack/${StackName}/${Id}
AWS,AWS CloudFormation,CloudFormation StackSet,arn:${Partition}:cloudformation:${Region}:${Account}:stackset/${StackSetName}:${Id}
AWS,AWS CloudTrail,CloudTrail,arn:${Partition}:cloudtrail:${Region}:${Account}:trail/${TrailName}
AWS,AWS CloudArtifact,CodeArtifact Repository,arn:${Partition}:codeartifact:${Region}:${Account}:repository/${DomainName}/${RepositoryName}
AWS,AWS CodeBuild,CodeBuild Project,arn:${Partition}:codebuild:${Region}:${Account}:project/${ProjectName}
AWS,AWS CodeCommit,CodeCommit Repository,arn:${Partition}:codecommit:${Region}:${Account}:${RepositoryName}
AWS,AWS CodeDeploy,Deployment Group,arn:${Partition}:codedeploy:${Region}:${Account}:deploymentgroup:${ApplicationName}/${DeploymentGroupName}
AWS,AWS CodeDeploy,CodeDeploy Application,arn:${Partition}:codedeploy:${Region}:${Account}:application:${ApplicationName}
AWS,AWS CodePipeline,CodePipeline,arn:${Partition}:codepipeline:${Region}:${Account}:${PipelineName}
AWS,AWS CodeStar,CodeStar Project,arn:${Partition}:codestar:${Region}:${Account}:project/${ProjectId}
AWS,AWS Database Migration Service (DMS),Replication Instance,arn:${Partition}:dms:${Region}:${Account}:rep:*
AWS,AWS Directory Service,Directory,arn:${Partition}:ds:${Region}:${Account}:directory/${DirectoryId}
AWS,AWS Elastic Disaster Recovery (DRS),EDR Source Server,arn:${Partition}:drs:${Region}:${Account}:source-server/${SourceServerID}
AWS,AWS Elastic Disaster Recovery (DRS),EDR Recovery Instance,arn:${Partition}:drs:${Region}:${Account}:recovery-instance/${RecoveryInstanceID}
AWS,AWS Firewall Management,Firewall Manager Policy,arn:${Partition}:fms:${Region}:${Account}:policy/${Id}
AWS,AWS IAM Identity Center (successor to SSO),SSO Instance,arn:${Partition}:sso:::instance/${InstanceId}
AWS,AWS Key Management Service (AWS KMS),KMS Key,arn:${Partition}:kms:${Region}:${Account}:key/${KeyId}
AWS,AWS Network Firewall,Network Firewall,arn:${Partition}:network-firewall:${Region}:${Account}:firewall/${Name}
AWS,AWS OpsWork,OpsWork Server,arn:${Partition}:opsworks-cm::${Account}:server/${ServerName}/${UniqueId}
AWS,AWS Organizations,Organization,arn:${Partition}:organizations::${MasterAccountId}:organization/o-${OrganizationId}
AWS,AWS Secrets Management,Secret,arn:${Partition}:secretsmanager:${Region}:${Account}:secret:${SecretId}
AWS,AWS Shield,Shield Protection Group,arn:${Partition}:shield::${Account}:protection-group/${Id}
AWS,AWS Storage Gateway,Storage Gateway,arn:${Partition}:storagegateway:${Region}:${Account}:gateway/${GatewayId}
AWS,AWS Systems Manager (SSM),SSM Managed Instance,arn:${Partition}:ssm:${Region}:${Account}:managed-instance/${InstanceId}
AWS,AWS Web Application Firewall (WAF),Classic WAF Rules,arn:${Partition}:waf::${Account}:rule/${Id}
AWS,AWS Web Application Firewall (WAF),WAF (v2) Rules,arn:${Partition}:wafv2:${Region}:${Account}:${Scope}/webacl/${Name}/${Id}
AWS,AWS Web Application Firewall (WAF),Classic Regional WAF Rules,arn:${Partition}:waf-regional:${Region}:${Account}:rule/${Id}
AWS,Site-to-Site VPN,Site-to-Site VPN Connection,arn:${Partition}:ec2:${Region}:${Account}:vpn-connection/${VpnConnectionId}
`
