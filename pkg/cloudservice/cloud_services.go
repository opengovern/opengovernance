package cloudservice

import "gitlab.com/keibiengine/keibi-engine/pkg/source"

type CloudService struct {
	Provider         source.Type
	Category         string
	FullServiceName  string
	ServiceNamespace string
}

var awsCloudServicesCSV = `
Category,Full Service Name,Service Namespace,Important
Management,Account Management,arn:aws:account:::,no
General,Activate,arn:aws:activate:::,no
Other,Alexa for Business,arn:aws:a4b:::,no
Web + Mobile,Amplify,arn:aws:amplify:::,yes
Web + Mobile,Amplify Admin,arn:aws:amplifybackend:::,no
Web + Mobile,Amplify UI Builder,arn:aws:amplifyuibuilder:::,no
Messaging,Apache Kafka APIs for MSK clusters,arn:aws:kafka-cluster:::,no
Networking,API Gateway,arn:aws:execute-api:::,no
Networking,API Gateway Management,arn:aws:apigateway:::,no
Integrations,App Integrations,arn:aws:app-integrations:::,yes
Networking,App Mesh,arn:aws:appmesh:::,yes
App Delivery,App Mesh Preview,arn:aws:appmesh-preview:::,no
Compute,App Runner,arn:aws:apprunner:::,yes
Other,AppConfig,arn:aws:appconfig:::,no
Integrations,AppFlow,arn:aws:appflow:::,no
Elastic Workload,Application Auto Scaling,arn:aws:application-autoscaling:::,no
Spend Management,Application Cost Profiler,arn:aws:application-cost-profiler:::,no
Other,Application Discovery Arsenal,arn:aws:arsenal:::,no
Other,Application Discovery Service,arn:aws:discovery:::,no
Other,Application Migration Service,arn:aws:mgn:::,no
End User,AppStream 2.0,arn:aws:appstream:::,yes
Web + Mobile,AppSync,arn:aws:appsync:::,yes
Security,Artifact,arn:aws:artifact:::,no
Analytics,Athena,arn:aws:athena:::,yes
Security,Audit Manager,arn:aws:auditmanager:::,no
AI + ML,Augmented AI,arn:aws::::,No
Governance,Auto Scaling Plans,arn:aws:autoscaling-plans:::,no
Storage,Backup,arn:aws:backup:::,yes
Storage,Backup Gateway,arn:aws:backup-gateway:::,yes
Storage,Backup storage,arn:aws:backup-storage:::,no
Serverless,Batch,arn:aws:batch:::,yes
Spend Management,Billing and Cost Management,arn:aws:aws-portal:::,no
Spend Management,Billing Conductor,arn:aws:billingconductor:::,no
Other,Braket,arn:aws:braket:::,no
Spend Management,Budgets,arn:aws:budgets:::,no
DevOps,BugBust,arn:aws:bugbust:::,no
Security,Certificate Manager,arn:aws:acm:::,yes
Security,Certificate Manager Private Certificate Authority,arn:aws:acm-pca:::,yes
Governance,Chatbot,arn:aws:chatbot:::,no
Other,Chime,arn:aws:chrime:::,no
IAM,Cloud Directory,arn:aws:clouddirectory:::,yes
Networking,Cloud Map,arn:aws:servicediscovery:::,no
DevOps,Cloud9,arn:aws:cloud9:::,yes
Governance,CloudFormation,arn:aws:cloudformation:::,yes
Networking,CloudFront,arn:aws:cloudfront:::,yes
Security,CloudHSM,arn:aws:cloudhsm:::,no
Analytics,CloudSearch,arn:aws:cloudsearch:::,Yes
DevOps,CloudShell,arn:aws:cloudshell:::,no
Governance,CloudTrail,arn:aws:cloudtrail:::,yes
Monitoring,CloudWatch,arn:aws:cloudwatch:::,yes
Monitoring,CloudWatch Application Insights,arn:aws:applicationinsights:::,yes
Monitoring,CloudWatch Evidently,arn:aws:evidently:::,no
Monitoring,CloudWatch Logs,arn:aws:logs:::,yes
Monitoring,CloudWatch RUM,arn:aws:rum:::,no
Monitoring,CloudWatch Synthetics,arn:aws:synthetics:::,no
DevOps,CodeArtifact,arn:aws:codeartifact:::,yes
DevOps,CodeBuild,arn:aws:codebuild:::,yes
DevOps,CodeCommit,arn:aws:codecommit:::,yes
DevOps,CodeDeploy,arn:aws:codedeploy:::,yes
DevOps,CodeDeploy secure host commands service,arn:aws:codedeploy-commands-secure:::,no
AI + ML,CodeGuru Profiler,arn:aws:codeguru-profiler:::,No
AI + ML,CodeGuru Reviewer,arn:aws:codeguru-reviewer:::,No
DevOps,CodePipeline,arn:aws:codepipeline:::,yes
DevOps,CodeStar,arn:aws:codestar:::,yes
DevOps,CodeStar Connections,arn:aws:codestar-connections:::,no
Web + Mobile,CodeStar Notifications,arn:aws:codestar-notifications:::,no
Web + Mobile,Cognito Identity,arn:aws:cognito-identity:::,no
Web + Mobile,Cognito Sync,arn:aws:cognito-sync:::,no
Web + Mobile,Cognito User Pools,arn:aws:cognito-idp:::,no
AI + ML,Comprehend,arn:aws:comprehend:::,No
AI + ML,Comprehend Medical,arn:aws:comprehendmedical:::,no
General,Compute Optimizer,arn:aws:compute-optimizer:::,no
Governance,Config,arn:aws:config:::,yes
Operations,Connect,arn:aws:connect:::,yes
Operations,Connect Customer Profiles,arn:aws:profile:::,No
Operations,Connect Voice ID,arn:aws:voiceid:::,no
Operations,Connect Wisdom,arn:aws:wisdom:::,no
IAM,Connector Service,arn:aws:awsconnector:::,yes
Governance,Control Tower,arn:aws:controltower:::,no
Spend Management,Cost and Usage Report,arn:aws:cur:::,no
Spend Management,Cost Explorer,arn:aws:ce:::,no
Analytics,Data Exchange,arn:aws:dataexchange:::,no
Data Services,Data Lifecycle Manager,arn:aws:dlm:::,yes
Analytics,Data Pipeline,arn:aws:datapipeline:::,no
Other,Database Migration Service (DMS),arn:aws:dms:::,yes
Data Services,Database Query Metadata Service,arn:aws:dbqms:::,yes
Other,DataSync,arn:aws:datasync:::,no
AI + ML,DeepComposer,arn:aws:deepcomposer:::,No
AI + ML,DeepLens,arn:aws:deeplens:::,No
AI + ML,DeepRacer,arn:aws:deepracer:::,No
Security,Detective,arn:aws:detective:::,no
Web + Mobile,Device Farm,arn:aws:devicefarm:::,no
AI + ML,DevOps Guru,arn:aws:devops-guru:::,No
Networking,Direct Connect,arn:aws:directconnect:::,yes
IAM,Directory Service,arn:aws:ds:::,no
Database,DynamoDB,arn:aws:dynamodb:::,yes
Data Services,DynamoDB Accelerator (DAX),arn:aws:dax:::,Yes
Compute,EC2 Auto Scaling,arn:aws:autoscaling:::,no
Automation,EC2 Image Builder,arn:aws:imagebuilder:::,yes
Infrastructure,EC2 Instance Connect,arn:aws:ec2-instance-connect:::,no
PaaS,Elastic Beanstalk,arn:aws:elasticbeanstalk:::,yes
Infrastructure,Elastic Cloud Compute (EC2),arn:aws:ec2:::,yes
Containers,Elastic Container Registry (ECR),arn:aws:ecr:::,Yes
Containers,Elastic Container Registry (ECR) Public,arn:aws:ecr-public:::,no
Containers,Elastic Container Service (ECS),arn:aws:ecs:::,Yes
Other,Elastic Disaster Recovery,arn:aws:drs:::,no
Storage,Elastic File System (EFS),arn:aws:efs:::,yes
Compute,Elastic Inference,arn:aws:elastic-inference:::,no
Containers,Elastic Kubernetes Service (EKS),arn:aws:eks:::,Yes
Networking,Elastic Load Balancing,arn:aws:elasticloadbalancing:::,Yes
Data Services,Elastic MapReduce (EMR),arn:aws:emr:::,yes
Media,Elastic Transcoder,arn:aws:elastictranscoder:::,no
Database,ElastiCache,arn:aws:elasticache:::,yes
Media,Elemental Appliances and Software,arn:aws:elemental-appliances-software:::,no
Media,Elemental Appliances and Software Activation Service,arn:aws:elemental-activations:::,no
Media,Elemental MediaConnect,arn:aws:mediaconnect:::,no
Media,Elemental MediaConvert,arn:aws:mediaconvert:::,no
Media,Elemental MediaLive,arn:aws:medialive:::,no
Media,Elemental MediaPackage,arn:aws:mediapackage:::,no
Media,Elemental MediaPackage VOD,arn:aws:mediapackage-vod:::,no
Media,Elemental MediaStore,arn:aws:mediastore:::,no
Media,Elemental MediaTailor,arn:aws:mediatailor:::,no
Media,Elemental Support Cases,arn:aws:elemental-support-cases:::,No
Media,Elemental Support Content,arn:aws:elemental-support-content:::,No
Analytics,EMR,arn:aws:elasticmapreduce:::,Yes
Data Services,EMR on EKS (EMR Containers),arn:aws:emr-containers:::,no
Data Services,EMR Serverless,arn:aws:emr-serverless:::,no
Integrations,EventBridge,arn:aws:events:::,no
App Delivery,EventBridge Schemas,arn:aws:schemas:::,no
DevOps,Fault Injection Simulator,arn:aws:fis:::,yes
Analytics,FinSpace,arn:aws:finspace:::,no
Security,Firewall Manager,arn:aws:fms:::,yes
AI + ML,Forecast,arn:aws:forecast:::,No
AI + ML,Fraud Detector,arn:aws:frauddetector:::,No
IoT,FreeRTOS,arn:aws:freertos:::,no
Storage,FSx,arn:aws:fsx:::,yes
GameTech,GameLift,arn:aws:gamelift:::,no
GameTech,GameSparks,arn:aws:gamesparks:::,No
Networking,Global Accelerator,arn:aws:globalaccelerator:::,no
Analytics,Glue,arn:aws:glue:::,yes
Data Services,Glue DataBrew,arn:aws:databrew:::,yes
Other,Ground Station,arn:aws:groundstation:::,no
AI + ML,GroundTruth Labeling,arn:aws:groundtruthlabeling:::,No
Security,GuardDuty,arn:aws:guardduty:::,yes
Management,Health APIs and Notifications,arn:aws:health:::,no
AI + ML,HealthLake,arn:aws:healthlake:::,No
Management,High-volume outbound communications,arn:aws:connect-campaigns:::,no
Other,Honeycode,arn:aws:honeycode:::,no
IAM,IAM Access Analyzer,arn:aws:access-analyzer:::,yes
IAM,Identity and Access Management (IAM),arn:aws:iam:::,yes
IAM,Identity and Access Management Roles Anywhere,arn:aws:rolesanywhere:::,no
IAM,Identity Store,arn:aws:identitystore:::,no
IAM,Identity Synchronization Service,arn:aws:identity-sync:::,no
General,Import Export Disk Service,arn:aws:importexport:::,Yes
Management,Inspector,arn:aws:inspector:::,no
Security,Inspector2,arn:aws:inspector2:::,no
Media,Interactive Video Service,arn:aws:ivs:::,no
Media,Interactive Video Service Chat,arn:aws:ivschat:::,No
IoT,IoT,arn:aws:iot:::,No
IoT,IoT 1-Click,arn:aws:iot1click:::,no
IoT,IoT Analytics,arn:aws:iotanalytics:::,No
IoT,IoT Core Device Advisor,arn:aws:iotdeviceadvisor:::,No
IoT,IoT Core for LoRaWAN,arn:aws:iotwireless:::,no
IoT,IoT Device Tester,arn:aws:iot-device-tester:::,no
IoT,IoT Events,arn:aws:iotevents:::,no
IoT,IoT Fleet Hub for Device Management,arn:aws:iotfleethub:::,no
IoT,IoT FleetWise,arn:aws:iotfleetwise:::,no
IoT,IoT Greengrass,arn:aws:greengrass:::,no
IoT,IoT Jobs DataPlane,arn:aws:iotjobsdata:::,no
IoT,IoT RoboRunner,arn:aws:iotroborunner:::,no
IoT,IoT RoboRunner,arn:aws:iotroborunner:::,no
IoT,IoT SiteWise,arn:aws:iotsitewise:::,no
IoT,IoT Things Graph,arn:aws:iotthingsgraph:::,no
IoT,IoT TwinMaker,arn:aws:iottwinmaker:::,no
Other,IQ,arn:aws:iq:::,no
Other,IQ Permissions,arn:aws:iq-permission:::,no
AI + ML,Kendra,arn:aws:kendra:::,no
Security,Key Management Service (KMS),arn:aws:kms:::,Yes
Database,Keyspaces (for Apache Cassandra),arn:aws:cassandra:::,yes
Analytics,Kinesis,arn:aws:kinesis:::,Yes
Data Services,Kinesis Analytics,arn:aws:kinesisanalytics:::,yes
Analytics,Kinesis Data Analytics,arn:aws:kinesisanalytics:::,Yes
Analytics,Kinesis Data Firehose,arn:aws:firehose:::,Yes
Analytics,Kinesis Video Streams,arn:aws:kinesisvideo:::,no
Analytics,Lake Formation,arn:aws:lakeformation:::,no
Serverless,Lambda,arn:aws:lambda:::,yes
Governance,Launch Wizard,arn:aws:launchwizard:::,no
Database,Ledger Database Service (QLDB),arn:aws:qldb:::,yes
AI + ML,Lex,arn:aws:lex:::,no
Governance,License Manager,arn:aws:license-manager:::,no
Compute,Lightsail,arn:aws:lightsail:::,no
Web + Mobile,Location Service,arn:aws:geo:::,no
AI + ML,Lookout for Equipment,arn:aws:lookoutequipment:::,No
AI + ML,Lookout for Metrics,arn:aws:lookoutmetrics:::,No
AI + ML,Lookout for Vision,arn:aws:lookoutvision:::,No
AI + ML,Machine Learning,arn:aws:machinelearning:::,No
Security,Macie,arn:aws:macie2:::,no
Other,Mainframe Modernization Service,arn:aws:m2:::,no
Other,Managed Blockchain,arn:aws:managedblockchain:::,no
Messaging,Managed Grafana,arn:aws:grafana:::,Yes
Messaging,Managed Service for Prometheus,arn:aws:aps:::,Yes
Analytics,Managed Streaming for Apache Kafka (MSK),arn:aws:kafka:::,yes
Messaging,Managed Streaming for Kafka Connect,arn:aws:kafkaconnect:::,Yes
Messaging,Managed Workflows for Apache Airflow (MWAA),arn:aws:airflow:::,yes
General,Marketplace,arn:aws:aws-marketplace:::,no
Governance,Marketplace Commerce Analytics Service,arn:aws:marketplacecommerceanalytics:::,No
Management,Marketplace Management,arn:aws:aws-marketplace-management:::,No
Management,Marketplace Management Portal,arn:aws:aws-marketplace-management:::,no
Other,Mechanical Turk,arn:aws:mechanicalturk:::,no
Media,Media Import,arn:aws:mediaimport:::,no
Database,MemoryDB,arn:aws:memorydb:::,yes
Messaging,Message Delivery Service,arn:aws:ec2messages:::,no
Messaging,Microservice Extractor for .NET,arn:aws:serviceextract:::,no
General,Migration Hub,arn:aws:mgh:::,Yes
General,Migration Hub Orchestrator,arn:aws:migrationhub-orchestrator:::,no
General,Migration Hub Refactor Spaces,arn:aws:refactor-spaces:::,no
General,Migration Hub Strategy Recommendations,arn:aws:migrationhub-strategy:::,no
Web + Mobile,Mobile Analytics,arn:aws:mobileanalytics:::,no
Web + Mobile,Mobile Hub,arn:aws:mobilehub:::,no
Web + Mobile,Monitron,arn:aws:monitron:::,no
Integrations,MQ,arn:aws:mq:::,yes
Database,Neptune,arn:aws:neptune-db:::,no
Security,Network Firewall,arn:aws:network-firewall:::,yes
Networking,Network Manager,arn:aws:networkmanager:::,Yes
Media,Nimble Studio,arn:aws:nimble:::,no
Analytics,OpenSearch Service,arn:aws:es:::,yes
Governance,OpsWorks,arn:aws:opsworks:::,yes
Governance,OpsWorks Configuration Management,arn:aws:opsworks-cm:::,no
Governance,Organizations,arn:aws:organizations:::,yes
Compute,Outposts,arn:aws:outposts:::,no
AI + ML,Panorama,arn:aws:panorama:::,No
Other,Performance Insights,arn:aws:pi:::,no
AI + ML,Personalize,arn:aws:personalize:::,No
Web + Mobile,Pinpoint,arn:aws:mobiletargeting:::,no
Web + Mobile,Pinpoint SMS and Voice Service,arn:aws:sms-voice:::,no
AI + ML,Polly,arn:aws:polly:::,Yes
Management,Price List,arn:aws:pricing:::,no
Governance,Proton,arn:aws:proton:::,no
Management,Purchase Orders Console,arn:aws:purchase-orders:::,no
Analytics,QuickSight,arn:aws:quicksight:::,yes
Database,RDS Data API,arn:aws:rds-data:::,Yes
IAM,RDS IAM Authentication,arn:aws:rds-db:::,Yes
Other,Recycle Bin,arn:aws:rbin:::,no
Analytics,Redshift,arn:aws:redshift:::,yes
Data Services,Redshift Data API,arn:aws:redshift-data:::,no
Analytics,Redshift Serverless,arn:aws:redshift-serverless:::,no
AI + ML,Rekognition,arn:aws:rekognition:::,no
Other,Resilience Hub Service,arn:aws:resiliencehub:::,No
Security,Resource Access Manager (RAM),arn:aws:ram:::,no
Management,Resource Group Tagging API,arn:aws:tag:::,No
Management,Resource Groups,arn:aws:resource-groups:::,yes
Other,RHEL Knowledgebase Portal,arn:aws:rhelkb:::,no
AI + ML,RoboMaker,arn:aws:robomaker:::,no
Networking,Route 53,arn:aws:route53:::,yes
Networking,Route 53 Domains,arn:aws:route53domains:::,no
Networking,Route 53 Recovery Cluster,arn:aws:route53-recovery-cluster:::,no
Networking,Route 53 Recovery Controls,arn:aws:route53-recovery-control-config:::,no
Networking,Route 53 Recovery Readiness,arn:aws:route53-recovery-readiness:::,no
Networking,Route 53 Resolver,arn:aws:route53resolver:::,no
Storage,S3 Glacier,arn:aws:glacier:::,no
Storage,S3 Object Lambda,arn:aws:s3-object-lambda:::,no
Storage,S3 on Outposts,arn:aws:s3-outposts:::,no
AI + ML,SageMaker,arn:aws:sagemaker:::,no
Other,SageMaker Ground Truth Synthetic,arn:aws:sagemaker-groundtruth-synthetic:::,no
Spend Management,Savings Plans,arn:aws:savingsplans:::,no
Security,Secrets Manager,arn:aws:secretsmanager:::,no
Security,Security Hub,arn:aws:securityhub:::,no
IAM,Security Token Service (STS),arn:aws:sts:::,yes
Other,Server Migration Service,arn:aws:sms:::,no
Serverless,Serverless Application Repository,arn:aws:serverlessrepo:::,no
Governance,Service Catalog,arn:aws:servicecatalog:::,no
Management,Service Quotas,arn:aws:servicequotas:::,No
Messaging,Session Manager Message Gateway Service,arn:aws:ssmmessages:::,No
Security,Shield,arn:aws:shield:::,yes
Other,Signer,arn:aws:signer:::,no
General,Simple Email Service (SES),arn:aws:ses:::,yes
Integrations,Simple Notification Service (SNS),arn:aws:sns:::,Yes
Integrations,Simple Queue Service (SQS),arn:aws:sqs:::,Yes
Storage,Simple Storage Service (S3),arn:aws:s3:::,Yes
Integrations,Simple Workflow Service (SWF),arn:aws:swf:::,Yes
Database,SimpleDB,arn:aws:sdb:::,yes
IAM,Single Sign-On (SSO),arn:aws:sso:::,Yes
Database,Snow Device Management,arn:aws:snow-device-management:::,no
Other,Snowball,arn:aws:snowball:::,no
Database,SQL Workbench,arn:aws:sqlworkbench:::,no
IAM,SSO Directory,arn:aws:sso-directory:::,yes
Integrations,Step Functions,arn:aws:states:::,yes
Storage,Storage Gateway,arn:aws:storagegateway:::,Yes
Other,Sumerian,arn:aws:sumerian:::,no
Management,Support,arn:aws:support:::,no
Governance,Sustainability,arn:aws:sustainability:::,no
Operations,Systems Manager (SSM),arn:aws:ssm:::,yes
Operations,Systems Manager GUI Connect,arn:aws:ssm-guiconnect:::,no
Operations,Systems Manager Incident Manager,arn:aws:ssm-incidents:::,no
Operations,Systems Manager Incident Manager Contacts,arn:aws:ssm-contacts:::,no
Management,Tag Editor,arn:aws:resource-explorer:::,no
Spend Management,Tax Settings,arn:aws:tax:::,no
AI + ML,Textract,arn:aws:texttract:::,no
Database,Timestream,arn:aws:timestream:::,yes
Other,Tiros,arn:aws:tiros:::,no
AI + ML,Transcribe,arn:aws:transcribe:::,no
Other,Transfer Family,arn:aws:transfer:::,no
AI + ML,Translate,arn:aws:translate:::,no
Governance,Trusted Advisor,arn:aws:trustedadvisor:::,no
Security,WAF,arn:aws:waf:::,yes
Security,WAF Regional,arn:aws:waf-regional:::,Yes
Security,WAF V2,arn:aws:wafv2:::,Yes
Compute,Wavelength,arn:aws::::,no
Governance,Well-Architected Tool,arn:aws:wellarchitected:::,no
Other,WorkDocs,arn:aws:workdocs:::,no
Other,WorkLink,arn:aws:worklink:::,no
Other,WorkMail,arn:aws:workmail:::,yes
Other,WorkMail Message Flow,arn:aws:workmailmessageflow:::,no
End User,WorkSpaces,arn:aws:workspaces:::,yes
End User,WorkSpaces Application Manager (WAM),arn:aws:wam:::,no
End User,Workspaces Web,arn:aws:workspaces-web:::,no
DevOps,X-Ray,arn:aws:xray:::,yes
`

var azureCloudServicesCSV = `
Category,Cloud Services Namespace,Azure Cloud Service
IAM,Microsoft.AAD,Azure Active Directory Domain Services
Other,Microsoft.Addons,Addons
PaaS,Microsoft.App,Azure Container Apps
IAM,Microsoft.ADHybridHealthService,Azure Active Directory
Management,Microsoft.Advisor,Azure Advisor
Monitoring,Microsoft.AlertsManagement,Azure Monitor
Data Service,Microsoft.AnalysisServices,Azure Analysis Services
Management,Microsoft.ApiManagement,API Management
Management,Microsoft.AppConfiguration,Azure App Configuration
Management,Microsoft.AppPlatform,Azure Spring Cloud
Management,Microsoft.Attestation,Azure Attestation Service
Governance,Microsoft.Authorization,Azure Resource Manager
Automation,Microsoft.Automation,Automation
Automation,Microsoft.AutonomousSystems,Autonomous Systems
Automation,Microsoft.AVS,Azure VMware Solution
IAM,Microsoft.AzureActiveDirectory,Azure Active Directory B2C
Data Service,Microsoft.AzureArcData,Azure Arc-enabled data services
Data Service,Microsoft.AzureData,SQL Server registry
Data Service,Microsoft.AzureStack,Azure Stack
Data Service,Microsoft.AzureStackHCI,Azure Stack HCI
Compute,Microsoft.Batch,Batch
Spend Management,Microsoft.Billing,Cost Management and Billing
Other,Microsoft.BingMaps,Bing Maps
Other,Microsoft.Blockchain,Azure Blockchain Service
Other,Microsoft.BlockchainTokens,Azure Blockchain Tokens
Governance,Microsoft.Blueprint,Azure Blueprints
Other,Microsoft.BotService,Azure Bot Service
Data Service,Microsoft.Cache,Azure Cache for Redis
Governance,Microsoft.Capacity,Capacity
Networking,Microsoft.Cdn,Content Delivery Network
Networking,Microsoft.CertificateRegistration,App Service Certificates
Monitoring,Microsoft.ChangeAnalysis,Azure Monitor
Other,Microsoft.ClassicCompute,Classic deployment model virtual machine
Other,Microsoft.ClassicInfrastructureMigrate,Classic deployment model migration
Other,Microsoft.ClassicNetwork,Classic deployment model virtual network
Other,Microsoft.ClassicStorage,Classic deployment model storage
Other,Microsoft.ClassicSubscription,Classic deployment model
Other,Microsoft.CognitiveServices,Cognitive Services
Other,Microsoft.Commerce,Commerce
Compute,Microsoft.Compute,Virtual Machines
Other,Microsoft.Consumption,Cost Management
Containers,Microsoft.ContainerInstance,Container Instances
Containers,Microsoft.ContainerRegistry,Container Registry
Containers,Microsoft.ContainerService,Azure Kubernetes Service (AKS)
Spend Management,Microsoft.CostManagement,Cost Management
Spend Management,Microsoft.CostManagementExports,Cost Management
Other,Microsoft.CustomerLockbox,Customer Lockbox for Microsoft Azure
Other,Microsoft.CustomProviders,Azure Custom Providers
Data Service,Microsoft.DataBox,Azure Data Box
Data Service,Microsoft.DataBoxEdge,Azure Stack Edge
Data Service,Microsoft.Databricks,Azure Databricks
Data Service,Microsoft.DataCatalog,Data Catalog
Data Service,Microsoft.DataFactory,Data Factory
Data Service,Microsoft.DataLakeAnalytics,Data Lake Analytics
Data Service,Microsoft.DataLakeStore,Azure Data Lake Storage Gen2
Data Service,Microsoft.DataMigration,Azure Database Migration Service
Data Service,Microsoft.DataProtection,Data Protection
Data Service,Microsoft.DataShare,Azure Data Share
Database,Microsoft.DBforMariaDB,Azure Database for MariaDB
Database,Microsoft.DBforMySQL,Azure Database for MySQL
Database,Microsoft.DBforPostgreSQL,Azure Database for PostgreSQL
End User,Microsoft.DesktopVirtualization,Azure Virtual Desktop
End User,Microsoft.Devices,Azure IoT Hub
End User,Microsoft.DeviceUpdate,Device Update for IoT Hub
DevOps,Microsoft.DevOps,Azure DevOps
DevOps,Microsoft.DevSpaces,Azure Dev Spaces
DevOps,Microsoft.DevTestLab,Azure Lab Services
Other,Microsoft.DigitalTwins,Azure Digital Twins
Database,Microsoft.DocumentDB,Azure Cosmos DB
General,Microsoft.DomainRegistration,Domain Registration
Governance,Microsoft.DynamicsLcs,Lifecycle Services
Other,Microsoft.EnterpriseKnowledgeGraph,Enterprise Knowledge Graph
Data Service,Microsoft.EventGrid,Event Grid
Data Service,Microsoft.EventHub,Event Hubs
Other,Microsoft.Features,Azure Resource Manager
General,Microsoft.GuestConfiguration,Azure Policy
Big Data,Microsoft.HanaOnAzure,SAP HANA on Azure Large Instances
Big Data,Microsoft.HardwareSecurityModules,Azure Dedicated HSM
Big Data,Microsoft.HDInsight,HDInsight
Governance,Microsoft.HealthcareApis (Azure API for FHIR),Azure API for FHIR
Governance,Microsoft.HealthcareApis (Healthcare APIs),Healthcare APIs
Compute,Microsoft.HybridCompute,Azure Arc-enabled servers
Compute,Microsoft.HybridData,StorSimple
Compute,Microsoft.HybridNetwork,Network Function Manager
Compute,Microsoft.ImportExport,Azure Import/Export
Governance,Microsoft.Insights,Azure Monitor
IoT,Microsoft.IoTCentral,Azure IoT Central
IoT,Microsoft.IoTSpaces,Azure Digital Twins
Management,Microsoft.Intune,Azure Monitor
Security,Microsoft.KeyVault,Key Vault
Containers,Microsoft.Kubernetes,Azure Kubernetes
Data Service,Microsoft.Kusto,Azure Data Explorer
DevOps,Microsoft.LabServices,Azure Lab Services
PaaS,Microsoft.Logic,Logic Apps
AI + ML,Microsoft.MachineLearning,Machine Learning Studio
AI + ML,Microsoft.MachineLearningServices,Azure Machine Learning
Management,Microsoft.Maintenance,Azure Maintenance
IAM,Microsoft.ManagedIdentity,Managed identities for Azure resources
Management,Microsoft.ManagedNetwork,Virtual networks managed by PaaS services
Management,Microsoft.ManagedServices,Azure Lighthouse
Management,Microsoft.Management,Management Groups
Other,Microsoft.Maps,Azure Maps
Management,Microsoft.Marketplace,Marketplace
Management,Microsoft.MarketplaceApps,Marketplace Apps
Management,Microsoft.MarketplaceOrdering,Marketplace Ordering
Media,Microsoft.Media,Media Services
PaaS,Microsoft.Microservices4Spring,Azure Spring Cloud
Other,Microsoft.Migrate,Azure Migrate
Other,Microsoft.MixedReality,Azure Spatial Anchors
Storage,Microsoft.NetApp,Azure NetApp Files
Networking,Microsoft.Network,Application Gateway
Other,Microsoft.Notebooks,Azure Notebooks
Other,Microsoft.NotificationHubs,Notification Hubs
Other,Microsoft.ObjectStore,Object Store
Other,Microsoft.OffAzure,Azure Migrate
Other,Microsoft.OperationalInsights,Azure Monitor
Operations,Microsoft.OperationsManagement,Azure Monitor
Other,Microsoft.Peering,Azure Peering Service
Other,Microsoft.PolicyInsights,Azure Policy
Other,Microsoft.Portal,Azure portal
Analytics,Microsoft.PowerBI,Power BI
Analytics,Microsoft.PowerBIDedicated,Power BI Embedded
Analytics,Microsoft.PowerPlatform,Power Platform
Data Service,Microsoft.ProjectBabylon,Azure Data Catalog
Other,Microsoft.Quantum,Azure Quantum
Recovery,Microsoft.RecoveryServices,Azure Site Recovery
Other,Microsoft.RedHatOpenShift,Azure Red Hat OpenShift
Other,Microsoft.Relay,Azure Relay
Management,Microsoft.ResourceGraph,Azure Resource Graph
Monitoring,Microsoft.ResourceHealth,Azure Service Health
Management,Microsoft.Resources,Azure Resource Manager
SaaS,Microsoft.SaaS,SaaS
Other,Microsoft.Scheduler,Scheduler
Other,Microsoft.Search,Azure Cognitive Search
Security,Microsoft.Security,Security Center
Other,Microsoft.SecurityInsights,Microsoft Sentinel
Other,Microsoft.SerialConsole,Azure Serial Console for Windows
PaaS,Microsoft.ServiceBus,Service Bus
PaaS,Microsoft.ServiceFabric,Service Fabric
Other,Microsoft.Services,Service
Other,Microsoft.SignalRService,Azure SignalR Service
Governance,Microsoft.SoftwarePlan,License
Database,Microsoft.Solutions,Azure Managed Applications
Database,Microsoft.Sql,SQL Service
Database,Microsoft.SqlVirtualMachine,SQL Server on Azure Virtual Machines
Storage,Microsoft.Storage,Storage
Storage,Microsoft.StorageCache,Azure HPC Cache
Storage,Microsoft.StorageSync,Storage
Storage,Microsoft.StorSimple,StorSimple
Networking,Microsoft.StreamAnalytics,Azure Stream Analytics
General,Microsoft.Subscription,Subscription
Management,microsoft.support,Support
Analytics,Microsoft.Synapse,Azure Synapse Analytics
Data Service,Microsoft.TimeSeriesInsights,Azure Time Series Insights
Other,Microsoft.Token,Token
Compute,Microsoft.VirtualMachineImages,Azure Image Builder
DevOps,microsoft.visualstudio,Azure DevOps
Other,Microsoft.VMware,Azure VMware Solution
Other,Microsoft.VMwareCloudSimple,Azure VMware Solution by CloudSimple
DevOps,Microsoft.VSOnline,Azure DevOps
PaaS,Microsoft.Web,Web
End User,Microsoft.WindowsDefenderATP,Microsoft Defender Advanced Threat Protection
End User,Microsoft.WindowsESU,Extended Security Updates
End User,Microsoft.WindowsIoT,Windows 10 IoT Core Services
Monitoring,Microsoft.WorkloadMonitor,Azure Monitor
`
