package cloudservice

type Category struct {
	Category     string
	SubCategory  string
	Cloud        string
	CloudService string
}

var categoriesCSV = `
Category,Sub-Category,Cloud,Cloud Service
Business Applications,Productivity Applications,AWS,Alexa for Business
Application Integration,API Management,AWS,Amazon API Gateway
Front-End Web & Mobile,RESTful API Gateway,AWS,Amazon API Gateway
Application Integration,No-code API Integration,AWS,Amazon AppFlow
End User,App Virtualization,AWS,Amazon AppStream 2.0
Big Data,Analytics,AWS,Amazon Athena
Database services,Relational Database,AWS,Amazon Aurora
Business Applications,Productivity Applications,AWS,Amazon Chime
Front-End Web & Mobile,Other,AWS,Amazon Chime SDK
Business Applications,Communication Developer Services,AWS,Amazon Chime Voice Connector
Front-End Web & Mobile,Test & monitor,AWS,Amazon CloudWatch
Monitoring and Logging,Cloud and Network Monitoring,AWS,Amazon CloudWatch
"Security, Identity, and Compliance",Detection,AWS,Amazon CloudWatch
Management and Governance,Monitoring,AWS,Amazon CloudWatch
DevOps,DevOps Automation,AWS,Amazon CodeGuru
DevOps,DevOps Automation,AWS,Amazon CodeWhisperer
"Security, Identity, and Compliance",App Identity Management,AWS,Amazon Cognito
Business Applications,General Business Applications,AWS,Amazon Connect
"Security, Identity, and Compliance",Incident response,AWS,Amazon Detective
Monitoring and Logging,Application Availability,AWS,Amazon DevOps Guru
Management and Governance,Automation + AIOps,AWS,Amazon DevOps Guru
Database services,Document Database,AWS,Amazon DocumentDB (with MongoDB compatibility)
Database services,Key-value,AWS,Amazon DynamoDB
Compute,Instances (virtual machines),AWS,Amazon EC2 Autoscaling
Compute,Instances (virtual machines),AWS,Amazon EC2 Spot Instances
Containers,Compute options,AWS,Amazon EC2 Spot Instances
Compute,Containers,AWS,Amazon ECS Anywhere
Containers,On-premises,AWS,Amazon ECS Anywhere
Compute,Containers,AWS,Amazon EKS Anywhere
Containers,On-premises,AWS,Amazon EKS Anywhere
Compute,Instances (virtual machines),AWS,Amazon Elastic Compute Cloud (EC2)
Containers,Compute options,AWS,Amazon Elastic Compute Cloud (EC2)
Compute,Containers,AWS,Amazon Elastic Container Registry (ECR)
Containers,Tools & services with containers support,AWS,Amazon Elastic Container Registry (ECR)
Compute,Containers,AWS,Amazon Elastic Container Service (ECS)
Containers,Container orchestration,AWS,Amazon Elastic Container Service (ECS)
DevOps,Microservices,AWS,Amazon Elastic Container Service (ECS)
Storage,File Storage,AWS,Amazon Elastic File System (EFS)
Compute,Containers,AWS,Amazon Elastic Kubernetes Service (EKS)
Containers,Container orchestration,AWS,Amazon Elastic Kubernetes Service (EKS)
Database services,In-memory Database,AWS,Amazon ElastiCache
Big Data,Analytics,AWS,Amazon EMR
Application Integration,Event Bus,AWS,Amazon EventBridge
Big Data,Analytics,AWS,Amazon FinSpace
Storage,File Storage,AWS,Amazon FSx
"Security, Identity, and Compliance",Detection,AWS,Amazon GuardDuty
Business Applications,Productivity Applications,AWS,Amazon Honeycode
"Security, Identity, and Compliance",Detection,AWS,Amazon Inspector
Database services,Column Database,AWS,Amazon Keyspaces (for Apache Cassandra)
Big Data,Analytics,AWS,Amazon Kinesis
Big Data,Data Movement,AWS,Amazon Kinesis Data Firehose
Big Data,Data Movement,AWS,Amazon Kinesis Data Streams
Big Data,Data Movement,AWS,Amazon Kinesis Video Streams
Compute,Instances (virtual machines),AWS,Amazon Lightsail
Containers,Tools & services with containers support,AWS,Amazon Lightsail
Front-End Web & Mobile,Geolocation,AWS,Amazon Location Service
"Security, Identity, and Compliance",Data Protection,AWS,Amazon Macie
Management and Governance,Monitoring,AWS,Amazon Managed Grafana
Management and Governance,Monitoring,AWS,Amazon Managed Service for Prometheus
Big Data,Data Movement,AWS,Amazon Managed Streaming for Apache Kafka (MSK)
Application Integration,Workflows,AWS,Amazon Managed Workflows for Apache Airflow (MWAA)
Database services,In-memory Database,AWS,Amazon MemoryDB for Redis
Application Integration,Messaging,AWS,Amazon MQ
Database services,Graph Database,AWS,Amazon Neptune
Big Data,Analytics,AWS,Amazon OpenSearch Service
Business Applications,General Business Applications,AWS,Amazon Pinpoint
Front-End Web & Mobile,Engagement Tools,AWS,Amazon Pinpoint
Business Applications,Communication Developer Services,AWS,Amazon Pinpoint APIs
Database services,Ledger,AWS,Amazon Quantum Ledger Database (QLDB)
Big Data,Analytics,AWS,Amazon Quicksight
Database services,Relational Database,AWS,Amazon RDS
Big Data,Analytics,AWS,Amazon Redshift
Database services,Relational Database,AWS,Amazon Redshift
"Security, Identity, and Compliance",Network and application protection,AWS,Amazon Route 53 Resolver DNS Firewall
Big Data,Data Lake,AWS,Amazon S3
Big Data,Data Lake,AWS,Amazon S3 Glacier
Big Data,Predictive analytics and machine learning,AWS,Amazon SageMaker
Business Applications,Communication Developer Services,AWS,Amazon Simple Email Service (SES)
Application Integration,Messaging,AWS,Amazon Simple Notification Service (SNS)
Application Integration,Messaging,AWS,Amazon Simple Queue Service (SQS)
Storage,Object/Blob Storage,AWS,Amazon Simple Storage Service (S3)
Database services,Time series database,AWS,Amazon Timestream
Business Applications,Productivity Applications,AWS,Amazon WorkDocs
Business Applications,Productivity Applications,AWS,Amazon WorkMail
End User,Virtual Desktops (VDI),AWS,Amazon WorkSpaces
End User,Web-based VDI,AWS,Amazon WorkSpaces Web
Front-End Web & Mobile,App Development,AWS,AWS Amplify
Front-End Web & Mobile,App Delivery,AWS,AWS Amplify
Network and Content Delivery,Application networking,AWS,AWS API Gateway
Containers,Tools & services with containers support,AWS,AWS App Mesh
DevOps,Service Mesh,AWS,AWS App Mesh
Compute,Containers,AWS,AWS App Runner
Containers,Tools & services with containers support,AWS,AWS App Runner
Front-End Web & Mobile,App Delivery,AWS,AWS App Runner
Application Integration,API Management,AWS,AWS App Sync
Containers,Tools & services with containers support,AWS,AWS App2Container
Containers,Open Source,AWS,AWS App2Container
Migration & Transfer,Assess and mobilize,AWS,AWS Application Discovery Service
Migration & Transfer,Migration,AWS,AWS Application Migration Service
Network and Content Delivery,Application networking,AWS,AWS AppMesh
Front-End Web & Mobile,GraphQL APIs,AWS,AWS AppSync
"Security, Identity, and Compliance",Compliance,AWS,AWS Artifact
"Security, Identity, and Compliance",Compliance,AWS,AWS Audit Manager
Big Data,Data Lake,AWS,AWS Backup
Storage,Disaster recovery and backup,AWS,AWS Backup
Compute,Instances (virtual machines),AWS,AWS Batch
Financial Management,Cloud Financial Management,AWS,AWS Billing Conductor
Financial Management,Cloud Financial Management,AWS,AWS Budget Service
Management and Governance,Environment Governance,AWS,AWS Budgets
"Security, Identity, and Compliance",Data Protection,AWS,AWS Certificate Manager
Containers,Tools & services with containers support,AWS,AWS Cloud Map
DevOps,IDE,AWS,AWS Cloud9
DevOps,Infrastructure as Code,AWS,AWS CloudFormation
Management and Governance,Resource Provisioning,AWS,AWS CloudFormation
"Security, Identity, and Compliance",Data Protection,AWS,AWS CloudHSM
Network and Content Delivery,Application networking,AWS,AWS CloudMap
Monitoring and Logging,Activity & API Usage Tracking,AWS,AWS CloudTrail
"Security, Identity, and Compliance",Detection,AWS,AWS CloudTrail
Management and Governance,Monitoring,AWS,AWS CloudTrail
DevOps,Artifact Management,AWS,AWS CodeArtifact
DevOps,Build and Test Code,AWS,AWS CodeBuild
DevOps,Version Control,AWS,AWS CodeCommit
DevOps,Source Code Management,AWS,AWS CodeCommit
DevOps,Private Git Hosting,AWS,AWS CodeCommit
DevOps,CI + CD,AWS,AWS CodeCommit
DevOps,Deployment Automation,AWS,AWS CodeDeploy
DevOps,CI + CD,AWS,AWS CodeDeploy
DevOps,Software Release Workflows,AWS,AWS CodePipeline
DevOps,CI + CD,AWS,AWS CodePipeline
DevOps,CI + CD Tool,AWS,AWS CodeStar
DevOps,CI + CD,AWS,AWS CodeStar
Cost Management,Cost Management,AWS,AWS Compute Optimizer
DevOps,Policy as Code,AWS,AWS Config
"Security, Identity, and Compliance",Detection,AWS,AWS Config
Management and Governance,Drift Monitoring,AWS,AWS Config
Management and Governance,Enterprise Governance,AWS,AWS Control Tower
Containers,Tools & services with containers support,AWS,AWS Copilot
Management and Governance,Spend Management,AWS,AWS Cost and Usage Report
Management and Governance,Spend Management,AWS,AWS Cost Explorer
Big Data,Data Lake,AWS,AWS Data Exchange
Big Data,Data Movement,AWS,AWS Database Migration Service (DMS)
Migration & Transfer,Migration,AWS,AWS Database Migration Service (DMS)
Database services,Data Migration,AWS,AWS Database Migration Service (DMS)
Storage,Data migration,AWS,AWS DataSync
Migration & Transfer,Storage Migration,AWS,AWS DataSync
Big Data,Predictive analytics and machine learning,AWS,AWS Deep Learning AMIs
Front-End Web & Mobile,Test & monitor,AWS,AWS Device Farm
Network and Content Delivery,Hybrid connectivity,AWS,AWS Direct Connect
"Security, Identity, and Compliance",Identity and access management,AWS,AWS Directory Service
Management and Governance,Debug + Tracing,AWS,AWS Distro for OpenTelemetry
Compute,Managed App,AWS,AWS Elastic Beanstalk
Platform as a Service,Web Apps,AWS,AWS Elastic Beanstalk
"Security, Identity, and Compliance",Incident response,AWS,AWS Elastic Disaster Recovery
Storage,Disaster recovery and backup,AWS,AWS Elastic Disaster Recovery (DRS)
Compute,Containers,AWS,AWS Fargate
Containers,Compute options,AWS,AWS Fargate
DevOps,Resiliency,AWS,AWS Fault Injection Simulator
Network and Content Delivery,Network Security,AWS,AWS Firewall Manager
"Security, Identity, and Compliance",Network and application protection,AWS,AWS Firewall Manager
Network and Content Delivery,Edge networking,AWS,AWS Global Accelerator
Big Data,Data Movement,AWS,AWS Glue
Big Data,Data Lake,AWS,AWS Glue
Big Data,Analytics,AWS,AWS Glue DataBrew
"Security, Identity, and Compliance",Identity and access management,AWS,AWS IAM Identity Center (successor to SSO)
"Security, Identity, and Compliance",Identity and access management,AWS,AWS Identity and Access Management (IAM)
"Security, Identity, and Compliance",Detection,AWS,AWS IoT Device Defender
"Security, Identity, and Compliance",Data Protection,AWS,AWS Key Management Service (AWS KMS)
Big Data,Data Lake,AWS,AWS Lake Formation
Big Data,Data Lake,AWS,AWS Lake Formation
Compute,Serverless,AWS,AWS Lambda
Containers,Tools & services with containers support,AWS,AWS Lambda
DevOps,Microservices,AWS,AWS Lambda
Management and Governance,Environment Governance,AWS,AWS License Management
Compute,Edge and hybrid,AWS,AWS Local Zones
Migration & Transfer,Migration,AWS,AWS Mainframe Modernization
Management and Governance,Managed Services,AWS,AWS Managed Services
Management and Governance,Resource Provisioning,AWS,AWS Marketplace
Migration & Transfer,Assess and mobilize,AWS,AWS Migration Hub
Network and Content Delivery,Network Security,AWS,AWS Network Firewall
"Security, Identity, and Compliance",Network and application protection,AWS,AWS Network Firewall
DevOps,Configuration Management,AWS,AWS OpsWorks
Management and Governance,Resource Provisioning,AWS,AWS OpsWorks
"Security, Identity, and Compliance",Identity and access management,AWS,AWS Organizations
Management and Governance,Enterprise Governance,AWS,AWS Organizations
Compute,Edge and hybrid,AWS,AWS Outposts
"Security, Identity, and Compliance",Data Protection,AWS,AWS Private Certificate Authority
Containers,Enterprise-scale container management,AWS,AWS Proton
Management and Governance,Automation + AIOps,AWS,AWS Proton
"Security, Identity, and Compliance",Resource Access,AWS,AWS Resource Access Manager
Cost Management,Cost Management,AWS,AWS Savings Plan
"Security, Identity, and Compliance",Data Protection,AWS,AWS Secrets Manager
"Security, Identity, and Compliance",Detection,AWS,AWS Security Hub
Management and Governance,Resource Provisioning,AWS,AWS Service Catalog
Management and Governance,Managed Services Connectivity,AWS,AWS Service Management Connector
Network and Content Delivery,Network Security,AWS,AWS Shield
"Security, Identity, and Compliance",Network and application protection,AWS,AWS Shield
Compute,Edge and hybrid,AWS,AWS Snow Family
Storage,Data migration,AWS,AWS Snow Family
Storage,Hybrid cloud storage and edge computing,AWS,AWS Snow Family
Migration & Transfer,Storage Migration,AWS,AWS Snow Family
Application Integration,Workflows,AWS,AWS Step Functions
Storage,Hybrid cloud storage and edge computing,AWS,AWS Storage Gateway
DevOps,Configuration Management,AWS,AWS Systems Manager
Management and Governance,System Ops,AWS,AWS Systems Manager
Storage,Managed file transfer,AWS,AWS Transfer Family
Migration & Transfer,Storage Migration,AWS,AWS Transfer Family
Network and Content Delivery,Network Security,AWS,AWS WAF
Compute,Edge and hybrid,AWS,AWS Wavelength
"Security, Identity, and Compliance",Network and application protection,AWS,AWS Web Application Firewall (WAF)
Management and Governance,Enterprise Architecture,AWS,AWS Well-Architected Tool
Monitoring and Logging,Distributed Tracing,AWS,AWS X-Ray
Management and Governance,Debug + Tracing,AWS,AWS X-Ray
Network and Content Delivery,Hybrid connectivity,AWS,Client VPN
Network and Content Delivery,Hybrid connectivity,AWS,Cloud WAN
Network and Content Delivery,Edge networking,AWS,CloudFront
Compute,Other,AWS,EC2 Image Builder
Storage,Block Storage,AWS,Elastic Block Store (EBS)
Network and Content Delivery,Load Balancers,AWS,Elastic Load Balancing (ELB)
Migration & Transfer,Migration,AWS,Mainframe on AWS
Migration & Transfer,Assess and mobilize,AWS,Migration Evaluator
Network and Content Delivery,Network Foundation,AWS,Private Link
Containers,Enterprise-scale container management,AWS,Red Hat OpenShift Service on AWS (ROSA)
Network and Content Delivery,Edge networking,AWS,Route53
Migration & Transfer,Migration,AWS,SAP on AWS
Network and Content Delivery,Hybrid connectivity,AWS,Site-to-Site VPN
Network and Content Delivery,Network Foundation,AWS,Transit Gateway
Network and Content Delivery,Network Foundation,AWS,Virtual Private Cloud (VPC)
Compute,Edge and hybrid,AWS,VMware Cloud on AWS
Migration & Transfer,Migration,AWS,VMware Cloud on AWS
Network and Content Delivery,Network Foundation,AWS,VPC IPAM
`
