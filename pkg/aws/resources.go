package aws

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/describer"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
)

type ResourceDescriber func(context.Context, aws.Config, string, []string, string, enums.DescribeTriggerType) (*Resources, error)
type SingleResourceDescriber func(context.Context, aws.Config, string, []string, string, map[string]string, enums.DescribeTriggerType) (*Resources, error)

type ResourceType struct {
	Name          string
	ServiceName   string
	ListDescriber ResourceDescriber
	GetDescriber  SingleResourceDescriber

	TerraformName        string
	TerraformServiceName string
}

var resourceTypes = map[string]ResourceType{
	"AWS::Redshift::Snapshot": {
		Name:                 "AWS::Redshift::Snapshot",
		ServiceName:          "Redshift",
		ListDescriber:        ParallelDescribeRegional(describer.RedshiftSnapshot),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetRedshiftSnapshot),
		TerraformName:        "aws_redshift_cluster_snapshot",
		TerraformServiceName: "redshift",
	},
	"AWS::AuditManager::Control": {
		Name:                 "AWS::AuditManager::Control",
		ServiceName:          "AuditManager",
		ListDescriber:        ParallelDescribeRegional(describer.AuditManagerControl),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetAuditManagerControl),
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ElasticLoadBalancingV2::ApplicationLoadBalancerMetricRequestCount": {
		Name:                 "AWS::ElasticLoadBalancingV2::ApplicationLoadBalancerMetricRequestCount",
		ServiceName:          "ElasticLoadBalancingV2",
		ListDescriber:        ParallelDescribeRegional(describer.ApplicationLoadBalancerMetricRequestCount),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetApplicationLoadBalancerMetricRequestCount),
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ElasticLoadBalancingV2::ApplicationLoadBalancerMetricRequestCountDaily": {
		Name:                 "AWS::ElasticLoadBalancingV2::ApplicationLoadBalancerMetricRequestCountDaily",
		ServiceName:          "ElasticLoadBalancingV2",
		ListDescriber:        ParallelDescribeRegional(describer.ApplicationLoadBalancerMetricRequestCountDaily),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetApplicationLoadBalancerMetricRequestCountDaily),
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::IAM::AccountSummary": {
		Name:                 "AWS::IAM::AccountSummary",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMAccountSummary),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::Glacier::Vault": {
		Name:                 "AWS::Glacier::Vault",
		ServiceName:          "Glacier",
		ListDescriber:        ParallelDescribeRegional(describer.GlacierVault),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetGlacierVault),
		TerraformName:        "aws_glacier_vault",
		TerraformServiceName: "glacier",
	},
	"AWS::Organizations::Organization": {
		Name:                 "AWS::Organizations::Organization",
		ServiceName:          "Organizations",
		ListDescriber:        ParallelDescribeRegional(describer.OrganizationsOrganization),
		GetDescriber:         nil,
		TerraformName:        "aws_organizations_organization",
		TerraformServiceName: "organizations",
	},
	"AWS::CloudSearch::Domain": {
		Name:                 "AWS::CloudSearch::Domain",
		ServiceName:          "CloudSearch",
		ListDescriber:        SequentialDescribeGlobal(describer.CloudSearchDomain),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudsearch_domain",
		TerraformServiceName: "cloudsearch",
	},
	"AWS::DynamoDb::GlobalSecondaryIndex": {
		Name:                 "AWS::DynamoDb::GlobalSecondaryIndex",
		ServiceName:          "DynamoDb",
		ListDescriber:        ParallelDescribeRegional(describer.DynamoDbGlobalSecondaryIndex),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetDynamoDbGlobalSecondaryIndex),
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::RouteTable": {
		Name:                 "AWS::EC2::RouteTable",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2RouteTable),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetEC2RouteTable),
		TerraformName:        "aws_route_table",
		TerraformServiceName: "ec2",
	},
	"AWS::SecurityHub::Hub": {
		Name:                 "AWS::SecurityHub::Hub",
		ServiceName:          "SecurityHub",
		ListDescriber:        ParallelDescribeRegional(describer.SecurityHubHub),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::StorageGateway::StorageGateway": {
		Name:                 "AWS::StorageGateway::StorageGateway",
		ServiceName:          "StorageGateway",
		ListDescriber:        ParallelDescribeRegional(describer.StorageGatewayStorageGateway),
		GetDescriber:         nil,
		TerraformName:        "aws_storagegateway_gateway",
		TerraformServiceName: "storagegateway",
	},
	"AWS::FMS::Policy": {
		Name:                 "AWS::FMS::Policy",
		ServiceName:          "FMS",
		ListDescriber:        ParallelDescribeRegional(describer.FMSPolicy),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetFMSPolicy),
		TerraformName:        "aws_fms_policy",
		TerraformServiceName: "fms",
	},
	"AWS::Inspector::AssessmentTemplate": {
		Name:                 "AWS::Inspector::AssessmentTemplate",
		ServiceName:          "Inspector",
		ListDescriber:        ParallelDescribeRegional(describer.InspectorAssessmentTemplate),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetInspectorAssessmentTemplate),
		TerraformName:        "aws_inspector_assessment_template",
		TerraformServiceName: "inspector",
	},
	"AWS::ElasticLoadBalancingV2::ListenerRule": {
		Name:                 "AWS::ElasticLoadBalancingV2::ListenerRule",
		ServiceName:          "ElasticLoadBalancingV2",
		ListDescriber:        ParallelDescribeRegional(describer.ElasticLoadBalancingV2ListenerRule),
		GetDescriber:         nil,
		TerraformName:        "aws_alb_listener_rule",
		TerraformServiceName: "elbv2",
	},
	"AWS::IAM::Role": {
		Name:                 "AWS::IAM::Role",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMRole),
		GetDescriber:         nil,
		TerraformName:        "aws_iam_role",
		TerraformServiceName: "iam",
	},
	"AWS::Backup::ProtectedResource": {
		Name:                 "AWS::Backup::ProtectedResource",
		ServiceName:          "Backup",
		ListDescriber:        ParallelDescribeRegional(describer.BackupProtectedResource),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CodeCommit::Repository": {
		Name:                 "AWS::CodeCommit::Repository",
		ServiceName:          "CodeCommit",
		ListDescriber:        ParallelDescribeRegional(describer.CodeCommitRepository),
		GetDescriber:         nil,
		TerraformName:        "aws_codecommit_repository",
		TerraformServiceName: "codecommit",
	},
	"AWS::EC2::VPCEndpoint": {
		Name:                 "AWS::EC2::VPCEndpoint",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2VPCEndpoint),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EventBridge::EventRule": {
		Name:                 "AWS::EventBridge::EventRule",
		ServiceName:          "EventBridge",
		ListDescriber:        ParallelDescribeRegional(describer.EventBridgeRule),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CloudFront::OriginAccessControl": {
		Name:                 "AWS::CloudFront::OriginAccessControl",
		ServiceName:          "CloudFront",
		ListDescriber:        SequentialDescribeGlobal(describer.CloudFrontOriginAccessControl),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudfront_origin_access_control",
		TerraformServiceName: "cloudfront",
	},
	"AWS::CodeBuild::Project": {
		Name:                 "AWS::CodeBuild::Project",
		ServiceName:          "CodeBuild",
		ListDescriber:        ParallelDescribeRegional(describer.CodeBuildProject),
		GetDescriber:         nil,
		TerraformName:        "aws_codebuild_project",
		TerraformServiceName: "codebuild",
	},
	"AWS::ElastiCache::ParameterGroup": {
		Name:                 "AWS::ElastiCache::ParameterGroup",
		ServiceName:          "ElastiCache",
		ListDescriber:        ParallelDescribeRegional(describer.ElastiCacheParameterGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_elasticache_parameter_group",
		TerraformServiceName: "elasticache",
	},
	"AWS::MemoryDb::Cluster": {
		Name:                 "AWS::MemoryDb::Cluster",
		ServiceName:          "MemoryDb",
		ListDescriber:        ParallelDescribeRegional(describer.MemoryDbCluster),
		GetDescriber:         nil,
		TerraformName:        "aws_memorydb_cluster",
		TerraformServiceName: "memorydb",
	},
	"AWS::Glue::Crawler": {
		Name:                 "AWS::Glue::Crawler",
		ServiceName:          "Glue",
		ListDescriber:        ParallelDescribeRegional(describer.GlueCrawler),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetGlueCrawler),
		TerraformName:        "aws_glue_crawler",
		TerraformServiceName: "glue",
	},
	"AWS::DirectConnect::Gateway": {
		Name:                 "AWS::DirectConnect::Gateway",
		ServiceName:          "DirectConnect",
		ListDescriber:        ParallelDescribeRegional(describer.DirectConnectGateway),
		GetDescriber:         nil,
		TerraformName:        "aws_dx_gateway",
		TerraformServiceName: "directconnect",
	},
	"AWS::DynamoDb::BackUp": {
		Name:                 "AWS::DynamoDb::BackUp",
		ServiceName:          "DynamoDb",
		ListDescriber:        ParallelDescribeRegional(describer.DynamoDbBackUp),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::EIP": {
		Name:                 "AWS::EC2::EIP",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2EIP),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetEC2EIP),
		TerraformName:        "aws_eip",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::InternetGateway": {
		Name:                 "AWS::EC2::InternetGateway",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2InternetGateway),
		GetDescriber:         nil,
		TerraformName:        "aws_internet_gateway",
		TerraformServiceName: "ec2",
	},
	"AWS::GuardDuty::PublishingDestination": {
		Name:                 "AWS::GuardDuty::PublishingDestination",
		ServiceName:          "GuardDuty",
		ListDescriber:        ParallelDescribeRegional(describer.GuardDutyPublishingDestination),
		GetDescriber:         nil,
		TerraformName:        "aws_guardduty_publishing_destination",
		TerraformServiceName: "guardduty",
	},
	"AWS::KinesisAnalyticsV2::Application": {
		Name:                 "AWS::KinesisAnalyticsV2::Application",
		ServiceName:          "KinesisAnalyticsV2",
		ListDescriber:        ParallelDescribeRegional(describer.KinesisAnalyticsV2Application),
		GetDescriber:         nil,
		TerraformName:        "aws_kinesisanalyticsv2_application",
		TerraformServiceName: "kinesisanalyticsv2",
	},
	"AWS::CostExplorer::ByUsageTypeMonthly": {
		Name:                 "AWS::CostExplorer::ByUsageTypeMonthly",
		ServiceName:          "CostExplorer",
		ListDescriber:        SequentialDescribeGlobal(describer.CostByServiceUsageLastMonth),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EMR::Instance": {
		Name:                 "AWS::EMR::Instance",
		ServiceName:          "EMR",
		ListDescriber:        ParallelDescribeRegional(describer.EMRInstance),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ApiGateway::RestApi": {
		Name:                 "AWS::ApiGateway::RestApi",
		ServiceName:          "ApiGateway",
		ListDescriber:        ParallelDescribeRegional(describer.ApiGatewayRestAPI),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetApiGatewayRestAPI),
		TerraformName:        "aws_api_gateway_rest_api",
		TerraformServiceName: "apigateway",
	},
	"AWS::ApiGatewayV2::Integration": {
		Name:                 "AWS::ApiGatewayV2::Integration",
		ServiceName:          "ApiGatewayV2",
		ListDescriber:        ParallelDescribeRegional(describer.ApiGatewayV2Integration),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetApiGatewayV2Integration),
		TerraformName:        "aws_apigatewayv2_integration",
		TerraformServiceName: "apigatewayv2",
	},
	"AWS::AutoScaling::AutoScalingGroup": {
		Name:                 "AWS::AutoScaling::AutoScalingGroup",
		ServiceName:          "AutoScaling",
		ListDescriber:        ParallelDescribeRegional(describer.AutoScalingAutoScalingGroup),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetAutoScalingAutoScalingGroup),
		TerraformName:        "aws_autoscaling_group",
		TerraformServiceName: "autoscaling",
	},
	"AWS::DynamoDb::TableExport": {
		Name:                 "AWS::DynamoDb::TableExport",
		ServiceName:          "DynamoDb",
		ListDescriber:        ParallelDescribeRegional(describer.DynamoDbTableExport),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::KeyPair": {
		Name:                 "AWS::EC2::KeyPair",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2KeyPair),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetEC2KeyPair),
		TerraformName:        "aws_key_pair",
		TerraformServiceName: "ec2",
	},
	"AWS::EFS::FileSystem": {
		Name:                 "AWS::EFS::FileSystem",
		ServiceName:          "EFS",
		ListDescriber:        ParallelDescribeRegional(describer.EFSFileSystem),
		GetDescriber:         nil,
		TerraformName:        "aws_efs_file_system",
		TerraformServiceName: "efs",
	},
	"AWS::Kafka::Cluster": {
		Name:                 "AWS::Kafka::Cluster",
		ServiceName:          "Kafka",
		ListDescriber:        ParallelDescribeRegional(describer.KafkaCluster),
		GetDescriber:         nil,
		TerraformName:        "aws_msk_cluster",
		TerraformServiceName: "kafka",
	},
	"AWS::SecretsManager::Secret": {
		Name:                 "AWS::SecretsManager::Secret",
		ServiceName:          "SecretsManager",
		ListDescriber:        ParallelDescribeRegional(describer.SecretsManagerSecret),
		GetDescriber:         nil,
		TerraformName:        "aws_secretsmanager_secret",
		TerraformServiceName: "secretsmanager",
	},
	"AWS::Backup::LegalHold": {
		Name:                 "AWS::Backup::LegalHold",
		ServiceName:          "Backup",
		ListDescriber:        ParallelDescribeRegional(describer.BackupLegalHold),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CloudFront::Function": {
		Name:                 "AWS::CloudFront::Function",
		ServiceName:          "CloudFront",
		ListDescriber:        SequentialDescribeGlobal(describer.CloudFrontFunction),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudfront_function",
		TerraformServiceName: "cloudfront",
	},
	"AWS::EC2::SpotPrice": {
		Name:                 "AWS::EC2::SpotPrice",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2SpotPrice),
		GetDescriber:         nil,
		TerraformName:        "aws_ec2_spot_price",
		TerraformServiceName: "ec2",
	},
	"AWS::GlobalAccelerator::EndpointGroup": {
		Name:                 "AWS::GlobalAccelerator::EndpointGroup",
		ServiceName:          "GlobalAccelerator",
		ListDescriber:        ParallelDescribeRegional(describer.GlobalAcceleratorEndpointGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_globalaccelerator_endpoint_group",
		TerraformServiceName: "globalaccelerator",
	},
	"AWS::DAX::ParameterGroup": {
		Name:                 "AWS::DAX::ParameterGroup",
		ServiceName:          "DAX",
		ListDescriber:        ParallelDescribeRegional(describer.DAXParameterGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_dax_parameter_group",
		TerraformServiceName: "dax",
	},
	"AWS::SQS::Queue": {
		Name:                 "AWS::SQS::Queue",
		ServiceName:          "SQS",
		ListDescriber:        ParallelDescribeRegional(describer.SQSQueue),
		GetDescriber:         nil,
		TerraformName:        "aws_sqs_queue",
		TerraformServiceName: "sqs",
	},
	"AWS::Config::Rule": {
		Name:                 "AWS::Config::Rule",
		ServiceName:          "Config",
		ListDescriber:        ParallelDescribeRegional(describer.ConfigRule),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::GuardDuty::Member": {
		Name:                 "AWS::GuardDuty::Member",
		ServiceName:          "GuardDuty",
		ListDescriber:        ParallelDescribeRegional(describer.GuardDutyMember),
		GetDescriber:         nil,
		TerraformName:        "aws_guardduty_member",
		TerraformServiceName: "guardduty",
	},
	"AWS::IdentityStore::User": {
		Name:                 "AWS::IdentityStore::User",
		ServiceName:          "IdentityStore",
		ListDescriber:        ParallelDescribeRegional(describer.IdentityStoreUser),
		GetDescriber:         nil,
		TerraformName:        "aws_identitystore_user",
		TerraformServiceName: "identitystore",
	},
	"AWS::Inspector::Exclusion": {
		Name:                 "AWS::Inspector::Exclusion",
		ServiceName:          "Inspector",
		ListDescriber:        ParallelDescribeRegional(describer.InspectorExclusion),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::DirectoryService::Directory": {
		Name:                 "AWS::DirectoryService::Directory",
		ServiceName:          "DirectoryService",
		ListDescriber:        ParallelDescribeRegional(describer.DirectoryServiceDirectory),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EFS::AccessPoint": {
		Name:                 "AWS::EFS::AccessPoint",
		ServiceName:          "EFS",
		ListDescriber:        ParallelDescribeRegional(describer.EFSAccessPoint),
		GetDescriber:         nil,
		TerraformName:        "aws_efs_access_point",
		TerraformServiceName: "efs",
	},
	"AWS::IAM::PolicyAttachment": {
		Name:                 "AWS::IAM::PolicyAttachment",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMPolicyAttachment),
		GetDescriber:         nil,
		TerraformName:        "aws_iam_group_policy_attachment",
		TerraformServiceName: "iam",
	},
	"AWS::IAM::CredentialReport": {
		Name:                 "AWS::IAM::CredentialReport",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMCredentialReport),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::RDS::GlobalCluster": {
		Name:                 "AWS::RDS::GlobalCluster",
		ServiceName:          "RDS",
		ListDescriber:        ParallelDescribeRegional(describer.RDSGlobalCluster),
		GetDescriber:         nil,
		TerraformName:        "aws_rds_global_cluster",
		TerraformServiceName: "rds",
	},
	"AWS::EC2::EbsVolumeMetricReadOpsHourly": {
		Name:                 "AWS::EC2::EbsVolumeMetricReadOpsHourly",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EbsVolumeMetricReadOpsHourly),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CostExplorer::ForcastDaily": {
		Name:                 "AWS::CostExplorer::ForcastDaily",
		ServiceName:          "CostExplorer",
		ListDescriber:        SequentialDescribeGlobal(describer.CostForecastDaily),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ElasticLoadBalancingV2::NetworkLoadBalancerMetricNetFlowCount": {
		Name:                 "AWS::ElasticLoadBalancingV2::NetworkLoadBalancerMetricNetFlowCount",
		ServiceName:          "ElasticLoadBalancingV2",
		ListDescriber:        ParallelDescribeRegional(describer.NetworkLoadBalancerMetricNetFlowCount),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::GuardDuty::Detector": {
		Name:                 "AWS::GuardDuty::Detector",
		ServiceName:          "GuardDuty",
		ListDescriber:        ParallelDescribeRegional(describer.GuardDutyDetector),
		GetDescriber:         nil,
		TerraformName:        "aws_guardduty_detector",
		TerraformServiceName: "guardduty",
	},
	"AWS::SNS::Topic": {
		Name:                 "AWS::SNS::Topic",
		ServiceName:          "SNS",
		ListDescriber:        ParallelDescribeRegional(describer.SNSTopic),
		GetDescriber:         nil,
		TerraformName:        "aws_sns_topic",
		TerraformServiceName: "sns",
	},
	"AWS::AppConfig::Application": {
		Name:                 "AWS::AppConfig::Application",
		ServiceName:          "AppConfig",
		ListDescriber:        ParallelDescribeRegional(describer.AppConfigApplication),
		GetDescriber:         nil,
		TerraformName:        "aws_appconfig_application",
		TerraformServiceName: "appconfig",
	},
	"AWS::Batch::Job": {
		Name:                 "AWS::Batch::Job",
		ServiceName:          "Batch",
		ListDescriber:        ParallelDescribeRegional(describer.BatchJob),
		GetDescriber:         nil,
		TerraformName:        "aws_batch_job_definition",
		TerraformServiceName: "batch",
	},
	"AWS::CloudTrail::TrailEvent": {
		Name:                 "AWS::CloudTrail::TrailEvent",
		ServiceName:          "CloudTrail",
		ListDescriber:        ParallelDescribeRegional(describer.CloudTrailTrailEvent),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ECS::Service": {
		Name:                 "AWS::ECS::Service",
		ServiceName:          "ECS",
		ListDescriber:        ParallelDescribeRegional(describer.ECSService),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetECSService),
		TerraformName:        "aws_ecs_service",
		TerraformServiceName: "ecs",
	},
	"AWS::FSX::Task": {
		Name:                 "AWS::FSX::Task",
		ServiceName:          "FSX",
		ListDescriber:        ParallelDescribeRegional(describer.FSXTask),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::IAM::VirtualMFADevice": {
		Name:                 "AWS::IAM::VirtualMFADevice",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMVirtualMFADevice),
		GetDescriber:         nil,
		TerraformName:        "aws_iam_virtual_mfa_device",
		TerraformServiceName: "iam",
	},
	"AWS::WAFv2::WebACL": {
		Name:                 "AWS::WAFv2::WebACL",
		ServiceName:          "WAFv2",
		ListDescriber:        ParallelDescribeRegional(describer.WAFv2WebACL),
		GetDescriber:         nil,
		TerraformName:        "aws_wafv2_web_acl",
		TerraformServiceName: "wafv2",
	},
	"AWS::AuditManager::EvidenceFolder": {
		Name:                 "AWS::AuditManager::EvidenceFolder",
		ServiceName:          "AuditManager",
		ListDescriber:        ParallelDescribeRegional(describer.AuditManagerEvidenceFolder),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ApplicationAutoScaling::Target": {
		Name:                 "AWS::ApplicationAutoScaling::Target",
		ServiceName:          "ApplicationAutoScaling",
		ListDescriber:        ParallelDescribeRegional(describer.ApplicationAutoScalingTarget),
		GetDescriber:         nil,
		TerraformName:        "aws_appautoscaling_target",
		TerraformServiceName: "appautoscaling",
	},
	"AWS::Backup::Vault": {
		Name:                 "AWS::Backup::Vault",
		ServiceName:          "Backup",
		ListDescriber:        ParallelDescribeRegional(describer.BackupVault),
		GetDescriber:         nil,
		TerraformName:        "aws_backup_vault",
		TerraformServiceName: "backup",
	},
	"AWS::ElastiCache::Cluster": {
		Name:                 "AWS::ElastiCache::Cluster",
		ServiceName:          "ElastiCache",
		ListDescriber:        ParallelDescribeRegional(describer.ElastiCacheCluster),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetElastiCacheCluster),
		TerraformName:        "aws_elasticache_cluster",
		TerraformServiceName: "elasticache",
	},
	"AWS::CloudWatch::LogEvent": {
		Name:                 "AWS::CloudWatch::LogEvent",
		ServiceName:          "CloudWatch",
		ListDescriber:        ParallelDescribeRegional(describer.CloudWatchLogEvent),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::S3::Bucket": {
		Name:                 "AWS::S3::Bucket",
		ServiceName:          "S3",
		ListDescriber:        SequentialDescribeS3(describer.S3Bucket),
		GetDescriber:         nil,
		TerraformName:        "aws_s3_bucket",
		TerraformServiceName: "s3",
	},
	"AWS::CertificateManager::Certificate": {
		Name:                 "AWS::CertificateManager::Certificate",
		ServiceName:          "CertificateManager",
		ListDescriber:        ParallelDescribeRegional(describer.CertificateManagerCertificate),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EKS::AddonVersion": {
		Name:                 "AWS::EKS::AddonVersion",
		ServiceName:          "EKS",
		ListDescriber:        ParallelDescribeRegional(describer.EKSAddonVersion),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ApiGatewayV2::Api": {
		Name:                 "AWS::ApiGatewayV2::Api",
		ServiceName:          "ApiGatewayV2",
		ListDescriber:        ParallelDescribeRegional(describer.ApiGatewayV2API),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetApiGatewayV2API),
		TerraformName:        "aws_apigatewayv2_api",
		TerraformServiceName: "apigatewayv2",
	},
	"AWS::EC2::Volume": {
		Name:                 "AWS::EC2::Volume",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2Volume),
		GetDescriber:         nil,
		TerraformName:        "aws_ebs_volume",
		TerraformServiceName: "ec2",
	},
	"AWS::ApiGateway::ApiKey": {
		Name:                 "AWS::ApiGateway::ApiKey",
		ServiceName:          "ApiGateway",
		ListDescriber:        ParallelDescribeRegional(describer.ApiGatewayApiKey),
		GetDescriber:         nil,
		TerraformName:        "aws_api_gateway_api_key",
		TerraformServiceName: "apigateway",
	},
	"AWS::Glue::Connection": {
		Name:                 "AWS::Glue::Connection",
		ServiceName:          "Glue",
		ListDescriber:        ParallelDescribeRegional(describer.GlueConnection),
		GetDescriber:         nil,
		TerraformName:        "aws_glue_connection",
		TerraformServiceName: "glue",
	},
	"AWS::ECS::Task": {
		Name:                 "AWS::ECS::Task",
		ServiceName:          "ECS",
		ListDescriber:        ParallelDescribeRegional(describer.ECSTask),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::SSM::ManagedInstance": {
		Name:                 "AWS::SSM::ManagedInstance",
		ServiceName:          "SSM",
		ListDescriber:        ParallelDescribeRegional(describer.SSMManagedInstance),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::Lambda::Function": {
		Name:                 "AWS::Lambda::Function",
		ServiceName:          "Lambda",
		ListDescriber:        ParallelDescribeRegional(describer.LambdaFunction),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetLambdaFunction),
		TerraformName:        "aws_lambda_function",
		TerraformServiceName: "lambda",
	},
	"AWS::RDS::DBSnapshot": {
		Name:                 "AWS::RDS::DBSnapshot",
		ServiceName:          "RDS",
		ListDescriber:        ParallelDescribeRegional(describer.RDSDBSnapshot),
		GetDescriber:         nil,
		TerraformName:        "aws_db_snapshot",
		TerraformServiceName: "rds",
	},
	"AWS::CodeDeploy::Application": {
		Name:                 "AWS::CodeDeploy::Application",
		ServiceName:          "CodeDeploy",
		ListDescriber:        ParallelDescribeRegional(describer.CodeDeployApplication),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EMR::Cluster": {
		Name:                 "AWS::EMR::Cluster",
		ServiceName:          "EMR",
		ListDescriber:        ParallelDescribeRegional(describer.EMRCluster),
		GetDescriber:         nil,
		TerraformName:        "aws_emr_cluster",
		TerraformServiceName: "emr",
	},
	"AWS::IAM::AccessKey": {
		Name:                 "AWS::IAM::AccessKey",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMAccessKey),
		GetDescriber:         nil,
		TerraformName:        "aws_iam_access_key",
		TerraformServiceName: "iam",
	},
	"AWS::Glue::CatalogTable": {
		Name:                 "AWS::Glue::CatalogTable",
		ServiceName:          "Glue",
		ListDescriber:        ParallelDescribeRegional(describer.GlueCatalogTable),
		GetDescriber:         nil,
		TerraformName:        "aws_glue_catalog_table",
		TerraformServiceName: "glue",
	},
	"AWS::CloudTrail::Channel": {
		Name:                 "AWS::CloudTrail::Channel",
		ServiceName:          "CloudTrail",
		ListDescriber:        ParallelDescribeRegional(describer.CloudTrailChannel),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::NetworkAcl": {
		Name:                 "AWS::EC2::NetworkAcl",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2NetworkAcl),
		GetDescriber:         nil,
		TerraformName:        "aws_network_acl",
		TerraformServiceName: "ec2",
	},
	"AWS::ECS::ContainerInstance": {
		Name:                 "AWS::ECS::ContainerInstance",
		ServiceName:          "ECS",
		ListDescriber:        ParallelDescribeRegional(describer.ECSContainerInstance),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::RedshiftServerless::Snapshot": {
		Name:                 "AWS::RedshiftServerless::Snapshot",
		ServiceName:          "RedshiftServerless",
		ListDescriber:        ParallelDescribeRegional(describer.RedshiftServerlessSnapshot),
		GetDescriber:         nil,
		TerraformName:        "aws_redshiftserverless_snapshot",
		TerraformServiceName: "redshiftserverless",
	},
	"AWS::Workspaces::Bundle": {
		Name:                 "AWS::Workspaces::Bundle",
		ServiceName:          "Workspaces",
		ListDescriber:        ParallelDescribeRegional(describer.WorkspacesBundle),
		GetDescriber:         nil,
		TerraformName:        "aws_workspaces_bundle",
		TerraformServiceName: "workspaces",
	},
	"AWS::CloudTrail::Trail": {
		Name:                 "AWS::CloudTrail::Trail",
		ServiceName:          "CloudTrail",
		ListDescriber:        ParallelDescribeRegional(describer.CloudTrailTrail),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudtrail",
		TerraformServiceName: "cloudtrail",
	},
	"AWS::DAX::Parameter": {
		Name:                 "AWS::DAX::Parameter",
		ServiceName:          "DAX",
		ListDescriber:        ParallelDescribeRegional(describer.DAXParameter),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ECR::Image": {
		Name:                 "AWS::ECR::Image",
		ServiceName:          "ECR",
		ListDescriber:        ParallelDescribeRegional(describer.ECRImage),
		GetDescriber:         nil,
		TerraformName:        "aws_ecr_image",
		TerraformServiceName: "ecr",
	},
	"AWS::IAM::ServerCertificate": {
		Name:                 "AWS::IAM::ServerCertificate",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMServerCertificate),
		GetDescriber:         nil,
		TerraformName:        "aws_iam_server_certificate",
		TerraformServiceName: "iam",
	},
	"AWS::Keyspaces::Keyspace": {
		Name:                 "AWS::Keyspaces::Keyspace",
		ServiceName:          "Keyspaces",
		ListDescriber:        ParallelDescribeRegional(describer.KeyspacesKeyspace),
		GetDescriber:         nil,
		TerraformName:        "aws_keyspaces_keyspace",
		TerraformServiceName: "keyspaces",
	},
	"AWS::S3::AccessPoint": {
		Name:                 "AWS::S3::AccessPoint",
		ServiceName:          "S3",
		ListDescriber:        ParallelDescribeRegional(describer.S3AccessPoint),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::SageMaker::EndpointConfiguration": {
		Name:                 "AWS::SageMaker::EndpointConfiguration",
		ServiceName:          "SageMaker",
		ListDescriber:        ParallelDescribeRegional(describer.SageMakerEndpointConfiguration),
		GetDescriber:         nil,
		TerraformName:        "aws_sagemaker_endpoint_configuration",
		TerraformServiceName: "sagemaker",
	},
	"AWS::ElastiCache::ReservedCacheNode": {
		Name:                 "AWS::ElastiCache::ReservedCacheNode",
		ServiceName:          "ElastiCache",
		ListDescriber:        ParallelDescribeRegional(describer.ElastiCacheReservedCacheNode),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EMR::InstanceFleet": {
		Name:                 "AWS::EMR::InstanceFleet",
		ServiceName:          "EMR",
		ListDescriber:        ParallelDescribeRegional(describer.EMRInstanceFleet),
		GetDescriber:         nil,
		TerraformName:        "aws_emr_instance_fleet",
		TerraformServiceName: "emr",
	},
	"AWS::IAM::Account": {
		Name:                 "AWS::IAM::Account",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMAccount),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::VPCPeeringConnection": {
		Name:                 "AWS::EC2::VPCPeeringConnection",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2VPCPeeringConnection),
		GetDescriber:         nil,
		TerraformName:        "aws_vpc_peering_connection",
		TerraformServiceName: "ec2",
	},
	"AWS::EKS::FargateProfile": {
		Name:                 "AWS::EKS::FargateProfile",
		ServiceName:          "EKS",
		ListDescriber:        ParallelDescribeRegional(describer.EKSFargateProfile),
		GetDescriber:         nil,
		TerraformName:        "aws_eks_fargate_profile",
		TerraformServiceName: "eks",
	},
	"AWS::ElasticLoadBalancingV2::NetworkLoadBalancerMetricNetFlowCountDaily": {
		Name:                 "AWS::ElasticLoadBalancingV2::NetworkLoadBalancerMetricNetFlowCountDaily",
		ServiceName:          "ElasticLoadBalancingV2",
		ListDescriber:        ParallelDescribeRegional(describer.NetworkLoadBalancerMetricNetFlowCountDaily),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::IAM::AccountPasswordPolicy": {
		Name:                 "AWS::IAM::AccountPasswordPolicy",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMAccountPasswordPolicy),
		GetDescriber:         nil,
		TerraformName:        "aws_iam_account_password_policy",
		TerraformServiceName: "iam",
	},
	"AWS::Logs::MetricFilter": {
		Name:                 "AWS::Logs::MetricFilter",
		ServiceName:          "Logs",
		ListDescriber:        ParallelDescribeRegional(describer.CloudWatchLogsMetricFilter),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudwatch_log_metric_filter",
		TerraformServiceName: "logs",
	},
	"AWS::CodePipeline::Pipeline": {
		Name:                 "AWS::CodePipeline::Pipeline",
		ServiceName:          "CodePipeline",
		ListDescriber:        ParallelDescribeRegional(describer.CodePipelinePipeline),
		GetDescriber:         nil,
		TerraformName:        "aws_codepipeline",
		TerraformServiceName: "codepipeline",
	},
	"AWS::DAX::Cluster": {
		Name:                 "AWS::DAX::Cluster",
		ServiceName:          "DAX",
		ListDescriber:        ParallelDescribeRegional(describer.DAXCluster),
		GetDescriber:         nil,
		TerraformName:        "aws_dax_cluster",
		TerraformServiceName: "dax",
	},
	"AWS::DLM::LifecyclePolicy": {
		Name:                 "AWS::DLM::LifecyclePolicy",
		ServiceName:          "DLM",
		ListDescriber:        ParallelDescribeRegional(describer.DLMLifecyclePolicy),
		GetDescriber:         nil,
		TerraformName:        "aws_dlm_lifecycle_policy",
		TerraformServiceName: "dlm",
	},
	"AWS::OpsWorksCM::Server": {
		Name:                 "AWS::OpsWorksCM::Server",
		ServiceName:          "OpsWorksCM",
		ListDescriber:        ParallelDescribeRegional(describer.OpsWorksCMServer),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::AccessAnalyzer::Analyzer": {
		Name:                 "AWS::AccessAnalyzer::Analyzer",
		ServiceName:          "AccessAnalyzer",
		ListDescriber:        ParallelDescribeRegional(describer.AccessAnalyzerAnalyzer),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetAccessAnalyzerAnalyzer),
		TerraformName:        "aws_accessanalyzer_analyzer",
		TerraformServiceName: "accessanalyzer",
	},
	"AWS::ElastiCache::SubnetGroup": {
		Name:                 "AWS::ElastiCache::SubnetGroup",
		ServiceName:          "ElastiCache",
		ListDescriber:        ParallelDescribeRegional(describer.ElastiCacheSubnetGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_elasticache_subnet_group",
		TerraformServiceName: "elasticache",
	},
	"AWS::FSX::Volume": {
		Name:                 "AWS::FSX::Volume",
		ServiceName:          "FSX",
		ListDescriber:        ParallelDescribeRegional(describer.FSXVolume),
		GetDescriber:         nil,
		TerraformName:        "aws_fsx_ontap_volume",
		TerraformServiceName: "fsx",
	},
	"AWS::Amplify::App": {
		Name:                 "AWS::Amplify::App",
		ServiceName:          "Amplify",
		ListDescriber:        ParallelDescribeRegional(describer.AmplifyApp),
		GetDescriber:         nil,
		TerraformName:        "aws_amplify_app",
		TerraformServiceName: "amplify",
	},
	"AWS::AuditManager::Evidence": {
		Name:                 "AWS::AuditManager::Evidence",
		ServiceName:          "AuditManager",
		ListDescriber:        ParallelDescribeRegional(describer.AuditManagerEvidence),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CloudControl::Resource": {
		Name:                 "AWS::CloudControl::Resource",
		ServiceName:          "CloudControl",
		ListDescriber:        ParallelDescribeRegional(describer.CloudControlResource),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudcontrolapi_resource",
		TerraformServiceName: "cloudcontrol",
	},
	"AWS::CloudTrail::Query": {
		Name:                 "AWS::CloudTrail::Query",
		ServiceName:          "CloudTrail",
		ListDescriber:        ParallelDescribeRegional(describer.CloudTrailQuery),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CostExplorer::ByAccountMonthly": {
		Name:                 "AWS::CostExplorer::ByAccountMonthly",
		ServiceName:          "CostExplorer",
		ListDescriber:        SequentialDescribeGlobal(describer.CostByAccountLastMonth),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ECR::PublicRegistry": {
		Name:                 "AWS::ECR::PublicRegistry",
		ServiceName:          "ECR",
		ListDescriber:        SequentialDescribeGlobal(describer.ECRPublicRegistry),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::NetworkInterface": {
		Name:                 "AWS::EC2::NetworkInterface",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2NetworkInterface),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetEC2NetworkInterface),
		TerraformName:        "aws_network_interface",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::VPNConnection": {
		Name:                 "AWS::EC2::VPNConnection",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2VPNConnection),
		GetDescriber:         nil,
		TerraformName:        "aws_vpn_connection",
		TerraformServiceName: "ec2",
	},
	"AWS::FSX::StorageVirtualMachine": {
		Name:                 "AWS::FSX::StorageVirtualMachine",
		ServiceName:          "FSX",
		ListDescriber:        ParallelDescribeRegional(describer.FSXStorageVirtualMachine),
		GetDescriber:         nil,
		TerraformName:        "aws_fsx_ontap_storage_virtual_machine",
		TerraformServiceName: "fsx",
	},
	"AWS::ApiGateway::Authorizer": {
		Name:                 "AWS::ApiGateway::Authorizer",
		ServiceName:          "ApiGateway",
		ListDescriber:        ParallelDescribeRegional(describer.ApiGatewayAuthorizer),
		GetDescriber:         nil,
		TerraformName:        "aws_api_gateway_authorizer",
		TerraformServiceName: "apigateway",
	},
	"AWS::AppStream::Stack": {
		Name:                 "AWS::AppStream::Stack",
		ServiceName:          "AppStream",
		ListDescriber:        ParallelDescribeRegional(describer.AppStreamStack),
		GetDescriber:         nil,
		TerraformName:        "aws_appstream_stack",
		TerraformServiceName: "appstream",
	},
	"AWS::CloudWatch::Alarm": {
		Name:                 "AWS::CloudWatch::Alarm",
		ServiceName:          "CloudWatch",
		ListDescriber:        ParallelDescribeRegional(describer.CloudWatchAlarm),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetCloudWatchAlarm),
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CloudWatch::LogSubscriptionFilter": {
		Name:                 "AWS::CloudWatch::LogSubscriptionFilter",
		ServiceName:          "CloudWatch",
		ListDescriber:        ParallelDescribeRegional(describer.CloudWatchLogsSubscriptionFilter),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CostExplorer::ByRecordTypeMonthly": {
		Name:                 "AWS::CostExplorer::ByRecordTypeMonthly",
		ServiceName:          "CostExplorer",
		ListDescriber:        SequentialDescribeGlobal(describer.CostByRecordTypeLastMonth),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::RDS::DBCluster": {
		Name:                 "AWS::RDS::DBCluster",
		ServiceName:          "RDS",
		ListDescriber:        ParallelDescribeRegional(describer.RDSDBCluster),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetRDSDBCluster),
		TerraformName:        "aws_rds_cluster",
		TerraformServiceName: "rds",
	},
	"AWS::RDS::DBClusterSnapshot": {
		Name:                 "AWS::RDS::DBClusterSnapshot",
		ServiceName:          "RDS",
		ListDescriber:        ParallelDescribeRegional(describer.RDSDBClusterSnapshot),
		GetDescriber:         nil,
		TerraformName:        "aws_db_cluster_snapshot",
		TerraformServiceName: "rds",
	},
	"AWS::Backup::Framework": {
		Name:                 "AWS::Backup::Framework",
		ServiceName:          "Backup",
		ListDescriber:        ParallelDescribeRegional(describer.BackupFramework),
		GetDescriber:         nil,
		TerraformName:        "aws_backup_framework",
		TerraformServiceName: "backup",
	},
	"AWS::CodeBuild::SourceCredential": {
		Name:                 "AWS::CodeBuild::SourceCredential",
		ServiceName:          "CodeBuild",
		ListDescriber:        ParallelDescribeRegional(describer.CodeBuildSourceCredential),
		GetDescriber:         nil,
		TerraformName:        "aws_codebuild_source_credential",
		TerraformServiceName: "codebuild",
	},
	"AWS::IAM::ServiceSpecificCredential": {
		Name:                 "AWS::IAM::ServiceSpecificCredential",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMServiceSpecificCredential),
		GetDescriber:         nil,
		TerraformName:        "aws_iam_service_specific_credential",
		TerraformServiceName: "iam",
	},
	"AWS::EC2::EbsVolumeMetricWriteOpsDaily": {
		Name:                 "AWS::EC2::EbsVolumeMetricWriteOpsDaily",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EbsVolumeMetricWriteOpsDaily),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::CapacityReservationFleet": {
		Name:                 "AWS::EC2::CapacityReservationFleet",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2CapacityReservationFleet),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::NetworkFirewall::Firewall": {
		Name:                 "AWS::NetworkFirewall::Firewall",
		ServiceName:          "NetworkFirewall",
		ListDescriber:        ParallelDescribeRegional(describer.NetworkFirewallFirewall),
		GetDescriber:         nil,
		TerraformName:        "aws_networkfirewall_firewall",
		TerraformServiceName: "networkfirewall",
	},
	"AWS::Workspaces::Workspace": {
		Name:                 "AWS::Workspaces::Workspace",
		ServiceName:          "Workspaces",
		ListDescriber:        ParallelDescribeRegional(describer.WorkspacesWorkspace),
		GetDescriber:         nil,
		TerraformName:        "aws_workspaces_workspace",
		TerraformServiceName: "workspaces",
	},
	"AWS::DynamoDb::MetricAccountProvisionedReadCapacityUtilization": {
		Name:                 "AWS::DynamoDb::MetricAccountProvisionedReadCapacityUtilization",
		ServiceName:          "DynamoDb",
		ListDescriber:        ParallelDescribeRegional(describer.DynamoDBMetricAccountProvisionedReadCapacityUtilization),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ElasticSearch::Domain": {
		Name:                 "AWS::ElasticSearch::Domain",
		ServiceName:          "ElasticSearch",
		ListDescriber:        ParallelDescribeRegional(describer.ESDomain),
		GetDescriber:         nil,
		TerraformName:        "aws_elasticsearch_domain",
		TerraformServiceName: "elasticsearch",
	},
	"AWS::RDS::DBInstance": {
		Name:                 "AWS::RDS::DBInstance",
		ServiceName:          "RDS",
		ListDescriber:        ParallelDescribeRegional(describer.RDSDBInstance),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetRDSDBInstance),
		TerraformName:        "aws_db_instance",
		TerraformServiceName: "rds",
	},
	"AWS::EFS::MountTarget": {
		Name:                 "AWS::EFS::MountTarget",
		ServiceName:          "EFS",
		ListDescriber:        ParallelDescribeRegional(describer.EFSMountTarget),
		GetDescriber:         nil,
		TerraformName:        "aws_efs_mount_target",
		TerraformServiceName: "efs",
	},
	"AWS::AuditManager::Assessment": {
		Name:                 "AWS::AuditManager::Assessment",
		ServiceName:          "AuditManager",
		ListDescriber:        ParallelDescribeRegional(describer.AuditManagerAssessment),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::GlobalAccelerator::Listener": {
		Name:                 "AWS::GlobalAccelerator::Listener",
		ServiceName:          "GlobalAccelerator",
		ListDescriber:        ParallelDescribeRegional(describer.GlobalAcceleratorListener),
		GetDescriber:         nil,
		TerraformName:        "aws_globalaccelerator_listener",
		TerraformServiceName: "globalaccelerator",
	},
	"AWS::EC2::EbsVolumeMetricReadOpsDaily": {
		Name:                 "AWS::EC2::EbsVolumeMetricReadOpsDaily",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EbsVolumeMetricReadOpsDaily),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CostExplorer::ByUsageTypeDaily": {
		Name:                 "AWS::CostExplorer::ByUsageTypeDaily",
		ServiceName:          "CostExplorer",
		ListDescriber:        SequentialDescribeGlobal(describer.CostByServiceUsageLastDay),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EKS::Addon": {
		Name:                 "AWS::EKS::Addon",
		ServiceName:          "EKS",
		ListDescriber:        ParallelDescribeRegional(describer.EKSAddon),
		GetDescriber:         nil,
		TerraformName:        "aws_eks_addon",
		TerraformServiceName: "eks",
	},
	"AWS::IdentityStore::Group": {
		Name:                 "AWS::IdentityStore::Group",
		ServiceName:          "IdentityStore",
		ListDescriber:        ParallelDescribeRegional(describer.IdentityStoreGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_identitystore_group",
		TerraformServiceName: "identitystore",
	},
	"AWS::CostExplorer::ByServiceMonthly": {
		Name:                 "AWS::CostExplorer::ByServiceMonthly",
		ServiceName:          "CostExplorer",
		ListDescriber:        SequentialDescribeGlobal(describer.CostByServiceLastMonth),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::IAM::Policy": {
		Name:                 "AWS::IAM::Policy",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMPolicy),
		GetDescriber:         nil,
		TerraformName:        "aws_iam_policy",
		TerraformServiceName: "iam",
	},
	"AWS::Redshift::Cluster": {
		Name:                 "AWS::Redshift::Cluster",
		ServiceName:          "Redshift",
		ListDescriber:        ParallelDescribeRegional(describer.RedshiftCluster),
		GetDescriber:         nil,
		TerraformName:        "aws_redshift_cluster",
		TerraformServiceName: "redshift",
	},
	"AWS::WAFRegional::Rule": {
		Name:                 "AWS::WAFRegional::Rule",
		ServiceName:          "WAFRegional",
		ListDescriber:        ParallelDescribeRegional(describer.WAFRegionalRule),
		GetDescriber:         nil,
		TerraformName:        "aws_wafregional_rule",
		TerraformServiceName: "wafregional",
	},
	"AWS::Glue::DataCatalogEncryptionSettings": {
		Name:                 "AWS::Glue::DataCatalogEncryptionSettings",
		ServiceName:          "Glue",
		ListDescriber:        ParallelDescribeRegional(describer.GlueDataCatalogEncryptionSettings),
		GetDescriber:         nil,
		TerraformName:        "aws_glue_data_catalog_encryption_settings",
		TerraformServiceName: "glue",
	},
	"AWS::EC2::FlowLog": {
		Name:                 "AWS::EC2::FlowLog",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2FlowLog),
		GetDescriber:         nil,
		TerraformName:        "aws_flow_log",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::IpamPool": {
		Name:                 "AWS::EC2::IpamPool",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2IpamPool),
		GetDescriber:         nil,
		TerraformName:        "aws_vpc_ipam_pool",
		TerraformServiceName: "ec2",
	},
	"AWS::IAM::SamlProvider": {
		Name:                 "AWS::IAM::SamlProvider",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMSamlProvider),
		GetDescriber:         nil,
		TerraformName:        "aws_iam_saml_provider",
		TerraformServiceName: "iam",
	},
	"AWS::Route53::HostedZone": {
		Name:                 "AWS::Route53::HostedZone",
		ServiceName:          "Route53",
		ListDescriber:        SequentialDescribeGlobal(describer.Route53HostedZone),
		GetDescriber:         nil,
		TerraformName:        "aws_route53_zone",
		TerraformServiceName: "route53",
	},
	"AWS::EC2::PlacementGroup": {
		Name:                 "AWS::EC2::PlacementGroup",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2PlacementGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_placement_group",
		TerraformServiceName: "ec2",
	},
	"AWS::FSX::Snapshot": {
		Name:                 "AWS::FSX::Snapshot",
		ServiceName:          "FSX",
		ListDescriber:        ParallelDescribeRegional(describer.FSXSnapshot),
		GetDescriber:         nil,
		TerraformName:        "aws_fsx_openzfs_snapshot",
		TerraformServiceName: "fsx",
	},
	"AWS::KMS::Key": {
		Name:                 "AWS::KMS::Key",
		ServiceName:          "KMS",
		ListDescriber:        ParallelDescribeRegional(describer.KMSKey),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetKMSKey),
		TerraformName:        "aws_kms_key",
		TerraformServiceName: "kms",
	},
	"AWS::EC2::Ipam": {
		Name:                 "AWS::EC2::Ipam",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2Ipam),
		GetDescriber:         nil,
		TerraformName:        "aws_vpc_ipam",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::VPCEndpointService": {
		Name:                 "AWS::EC2::VPCEndpointService",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2VPCEndpointService),
		GetDescriber:         nil,
		TerraformName:        "aws_vpc_endpoint_service",
		TerraformServiceName: "ec2",
	},
	"AWS::ElasticBeanstalk::Environment": {
		Name:                 "AWS::ElasticBeanstalk::Environment",
		ServiceName:          "ElasticBeanstalk",
		ListDescriber:        ParallelDescribeRegional(describer.ElasticBeanstalkEnvironment),
		GetDescriber:         nil,
		TerraformName:        "aws_elastic_beanstalk_environment",
		TerraformServiceName: "elasticbeanstalk",
	},
	"AWS::Lambda::FunctionVersion": {
		Name:                 "AWS::Lambda::FunctionVersion",
		ServiceName:          "Lambda",
		ListDescriber:        ParallelDescribeRegional(describer.LambdaFunctionVersion),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::Glue::DevEndpoint": {
		Name:                 "AWS::Glue::DevEndpoint",
		ServiceName:          "Glue",
		ListDescriber:        ParallelDescribeRegional(describer.GlueDevEndpoint),
		GetDescriber:         nil,
		TerraformName:        "aws_glue_dev_endpoint",
		TerraformServiceName: "glue",
	},
	"AWS::Backup::RecoveryPoint": {
		Name:                 "AWS::Backup::RecoveryPoint",
		ServiceName:          "Backup",
		ListDescriber:        ParallelDescribeRegional(describer.BackupRecoveryPoint),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::DynamoDbStreams::Stream": {
		Name:                 "AWS::DynamoDbStreams::Stream",
		ServiceName:          "DynamoDbStreams",
		ListDescriber:        ParallelDescribeRegional(describer.DynamoDbStream),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::EgressOnlyInternetGateway": {
		Name:                 "AWS::EC2::EgressOnlyInternetGateway",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2EgressOnlyInternetGateway),
		GetDescriber:         nil,
		TerraformName:        "aws_egress_only_internet_gateway",
		TerraformServiceName: "ec2",
	},
	"AWS::CloudFront::Distribution": {
		Name:                 "AWS::CloudFront::Distribution",
		ServiceName:          "CloudFront",
		ListDescriber:        SequentialDescribeGlobal(describer.CloudFrontDistribution),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudfront_distribution",
		TerraformServiceName: "cloudfront",
	},
	"AWS::Glue::Job": {
		Name:                 "AWS::Glue::Job",
		ServiceName:          "Glue",
		ListDescriber:        ParallelDescribeRegional(describer.GlueJob),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetGlueJob),
		TerraformName:        "aws_glue_job",
		TerraformServiceName: "glue",
	},
	"AWS::DynamoDb::MetricAccountProvisionedWriteCapacityUtilization": {
		Name:                 "AWS::DynamoDb::MetricAccountProvisionedWriteCapacityUtilization",
		ServiceName:          "DynamoDb",
		ListDescriber:        ParallelDescribeRegional(describer.DynamoDBMetricAccountProvisionedWriteCapacityUtilization),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::AppStream::Fleet": {
		Name:                 "AWS::AppStream::Fleet",
		ServiceName:          "AppStream",
		ListDescriber:        ParallelDescribeRegional(describer.AppStreamFleet),
		GetDescriber:         nil,
		TerraformName:        "aws_appstream_fleet",
		TerraformServiceName: "appstream",
	},
	"AWS::SES::ConfigurationSet": {
		Name:                 "AWS::SES::ConfigurationSet",
		ServiceName:          "SES",
		ListDescriber:        ParallelDescribeRegional(describer.SESConfigurationSet),
		GetDescriber:         nil,
		TerraformName:        "aws_ses_configuration_set",
		TerraformServiceName: "ses",
	},
	"AWS::IAM::User": {
		Name:                 "AWS::IAM::User",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMUser),
		GetDescriber:         nil,
		TerraformName:        "aws_iam_user",
		TerraformServiceName: "iam",
	},
	"AWS::CloudFront::OriginRequestPolicy": {
		Name:                 "AWS::CloudFront::OriginRequestPolicy",
		ServiceName:          "CloudFront",
		ListDescriber:        SequentialDescribeGlobal(describer.CloudFrontOriginRequestPolicy),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudfront_origin_request_policy",
		TerraformServiceName: "cloudfront",
	},
	"AWS::EC2::SecurityGroup": {
		Name:                 "AWS::EC2::SecurityGroup",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2SecurityGroup),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetEC2SecurityGroup),
		TerraformName:        "aws_security_group",
		TerraformServiceName: "ec2",
	},
	"AWS::GuardDuty::IPSet": {
		Name:                 "AWS::GuardDuty::IPSet",
		ServiceName:          "GuardDuty",
		ListDescriber:        ParallelDescribeRegional(describer.GuardDutyIPSet),
		GetDescriber:         nil,
		TerraformName:        "aws_guardduty_ipset",
		TerraformServiceName: "guardduty",
	},
	"AWS::EKS::Cluster": {
		Name:                 "AWS::EKS::Cluster",
		ServiceName:          "EKS",
		ListDescriber:        ParallelDescribeRegional(describer.EKSCluster),
		GetDescriber:         nil,
		TerraformName:        "aws_eks_cluster",
		TerraformServiceName: "eks",
	},
	"AWS::Grafana::Workspace": {
		Name:                 "AWS::Grafana::Workspace",
		ServiceName:          "Grafana",
		ListDescriber:        ParallelDescribeRegional(describer.GrafanaWorkspace),
		GetDescriber:         nil,
		TerraformName:        "aws_grafana_workspace",
		TerraformServiceName: "grafana",
	},
	"AWS::Glue::CatalogDatabase": {
		Name:                 "AWS::Glue::CatalogDatabase",
		ServiceName:          "Glue",
		ListDescriber:        ParallelDescribeRegional(describer.GlueCatalogDatabase),
		GetDescriber:         nil,
		TerraformName:        "aws_glue_catalog_database",
		TerraformServiceName: "glue",
	},
	"AWS::Health::Event": {
		Name:                 "AWS::Health::Event",
		ServiceName:          "Health",
		ListDescriber:        ParallelDescribeRegional(describer.HealthEvent),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CloudFormation::StackSet": {
		Name:                 "AWS::CloudFormation::StackSet",
		ServiceName:          "CloudFormation",
		ListDescriber:        ParallelDescribeRegional(describer.CloudFormationStackSet),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudformation_stack_set",
		TerraformServiceName: "cloudformation",
	},
	"AWS::EC2::AvailabilityZone": {
		Name:                 "AWS::EC2::AvailabilityZone",
		ServiceName:          "EC2",
		ListDescriber:        SequentialDescribeGlobal(describer.EC2AvailabilityZone),
		GetDescriber:         nil,
		TerraformName:        "aws_availability_zone",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::TransitGateway": {
		Name:                 "AWS::EC2::TransitGateway",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2TransitGateway),
		GetDescriber:         nil,
		TerraformName:        "aws_ec2_transit_gateway",
		TerraformServiceName: "ec2",
	},
	"AWS::ApiGateway::UsagePlan": {
		Name:                 "AWS::ApiGateway::UsagePlan",
		ServiceName:          "ApiGateway",
		ListDescriber:        ParallelDescribeRegional(describer.ApiGatewayUsagePlan),
		GetDescriber:         nil,
		TerraformName:        "aws_api_gateway_usage_plan",
		TerraformServiceName: "apigateway",
	},
	"AWS::Inspector::Finding": {
		Name:                 "AWS::Inspector::Finding",
		ServiceName:          "Inspector",
		ListDescriber:        ParallelDescribeRegional(describer.InspectorFinding),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::Fleet": {
		Name:                 "AWS::EC2::Fleet",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2Fleet),
		GetDescriber:         nil,
		TerraformName:        "aws_ec2_fleet",
		TerraformServiceName: "ec2",
	},
	"AWS::ElasticBeanstalk::Application": {
		Name:                 "AWS::ElasticBeanstalk::Application",
		ServiceName:          "ElasticBeanstalk",
		ListDescriber:        ParallelDescribeRegional(describer.ElasticBeanstalkApplication),
		GetDescriber:         nil,
		TerraformName:        "aws_elastic_beanstalk_application",
		TerraformServiceName: "elasticbeanstalk",
	},
	"AWS::ElasticLoadBalancingV2::LoadBalancer": {
		Name:                 "AWS::ElasticLoadBalancingV2::LoadBalancer",
		ServiceName:          "ElasticLoadBalancingV2",
		ListDescriber:        ParallelDescribeRegional(describer.ElasticLoadBalancingV2LoadBalancer),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetElasticLoadBalancingV2LoadBalancer),
		TerraformName:        "aws_alb",
		TerraformServiceName: "elbv2",
	},
	"AWS::OpenSearch::Domain": {
		Name:                 "AWS::OpenSearch::Domain",
		ServiceName:          "OpenSearch",
		ListDescriber:        ParallelDescribeRegional(describer.OpenSearchDomain),
		GetDescriber:         nil,
		TerraformName:        "aws_opensearch_domain",
		TerraformServiceName: "opensearch",
	},
	"AWS::RDS::DBEventSubscription": {
		Name:                 "AWS::RDS::DBEventSubscription",
		ServiceName:          "RDS",
		ListDescriber:        ParallelDescribeRegional(describer.RDSDBEventSubscription),
		GetDescriber:         nil,
		TerraformName:        "aws_db_event_subscription",
		TerraformServiceName: "rds",
	},
	"AWS::SSOAdmin::Instance": {
		Name:                 "AWS::SSOAdmin::Instance",
		ServiceName:          "SSOAdmin",
		ListDescriber:        ParallelDescribeRegional(describer.SSOAdminInstance),
		GetDescriber:         nil,
		TerraformName:        "aws_ssoadmin_instances",
		TerraformServiceName: "ssoadmin",
	},
	"AWS::EC2::RegionalSettings": {
		Name:                 "AWS::EC2::RegionalSettings",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2RegionalSettings),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::SecurityGroupRule": {
		Name:                 "AWS::EC2::SecurityGroupRule",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2SecurityGroupRule),
		GetDescriber:         nil,
		TerraformName:        "aws_security_group_rule",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::TransitGatewayAttachment": {
		Name:                 "AWS::EC2::TransitGatewayAttachment",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2TransitGatewayAttachment),
		GetDescriber:         nil,
		TerraformName:        "aws_ec2_transit_gateway_attachment",
		TerraformServiceName: "ec2",
	},
	"AWS::SES::Identity": {
		Name:                 "AWS::SES::Identity",
		ServiceName:          "SES",
		ListDescriber:        ParallelDescribeRegional(describer.SESIdentity),
		GetDescriber:         nil,
		TerraformName:        "aws_ses_email_identity",
		TerraformServiceName: "ses",
	},
	"AWS::WAF::Rule": {
		Name:                 "AWS::WAF::Rule",
		ServiceName:          "WAF",
		ListDescriber:        ParallelDescribeRegional(describer.WAFRule),
		GetDescriber:         nil,
		TerraformName:        "aws_waf_rule",
		TerraformServiceName: "waf",
	},
	"AWS::AutoScaling::LaunchConfiguration": {
		Name:                 "AWS::AutoScaling::LaunchConfiguration",
		ServiceName:          "AutoScaling",
		ListDescriber:        ParallelDescribeRegional(describer.AutoScalingLaunchConfiguration),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetAutoScalingLaunchConfiguration),
		TerraformName:        "aws_launch_configuration",
		TerraformServiceName: "autoscaling",
	},
	"AWS::CloudTrail::EventDataStore": {
		Name:                 "AWS::CloudTrail::EventDataStore",
		ServiceName:          "CloudTrail",
		ListDescriber:        ParallelDescribeRegional(describer.CloudTrailEventDataStore),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudtrail_event_data_store",
		TerraformServiceName: "cloudtrail",
	},
	"AWS::CodeDeploy::DeploymentGroup": {
		Name:                 "AWS::CodeDeploy::DeploymentGroup",
		ServiceName:          "CodeDeploy",
		ListDescriber:        ParallelDescribeRegional(describer.CodeDeployDeploymentGroup),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ImageBuilder::Image": {
		Name:                 "AWS::ImageBuilder::Image",
		ServiceName:          "ImageBuilder",
		ListDescriber:        ParallelDescribeRegional(describer.ImageBuilderImage),
		GetDescriber:         nil,
		TerraformName:        "aws_imagebuilder_image",
		TerraformServiceName: "imagebuilder",
	},
	"AWS::Redshift::ClusterParameterGroup": {
		Name:                 "AWS::Redshift::ClusterParameterGroup",
		ServiceName:          "Redshift",
		ListDescriber:        ParallelDescribeRegional(describer.RedshiftClusterParameterGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_redshift_parameter_group",
		TerraformServiceName: "redshift",
	},
	"AWS::Account::AlternateContact": {
		Name:                 "AWS::Account::AlternateContact",
		ServiceName:          "Account",
		ListDescriber:        ParallelDescribeRegional(describer.AccountAlternateContact),
		GetDescriber:         nil,
		TerraformName:        "aws_account_alternate_contact",
		TerraformServiceName: "account",
	},
	"AWS::Inspector::AssessmentTarget": {
		Name:                 "AWS::Inspector::AssessmentTarget",
		ServiceName:          "Inspector",
		ListDescriber:        ParallelDescribeRegional(describer.InspectorAssessmentTarget),
		GetDescriber:         nil,
		TerraformName:        "aws_inspector_assessment_target",
		TerraformServiceName: "inspector",
	},
	"AWS::CloudFront::ResponseHeadersPolicy": {
		Name:                 "AWS::CloudFront::ResponseHeadersPolicy",
		ServiceName:          "CloudFront",
		ListDescriber:        SequentialDescribeGlobal(describer.CloudFrontResponseHeadersPolicy),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudfront_response_headers_policy",
		TerraformServiceName: "cloudfront",
	},
	"AWS::EC2::Instance": {
		Name:                 "AWS::EC2::Instance",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2Instance),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetEC2Instance),
		TerraformName:        "aws_instance",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::InstanceAvailability": {
		Name:                 "AWS::EC2::InstanceAvailability",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2InstanceAvailability),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CostExplorer::ByRecordTypeDaily": {
		Name:                 "AWS::CostExplorer::ByRecordTypeDaily",
		ServiceName:          "CostExplorer",
		ListDescriber:        SequentialDescribeGlobal(describer.CostByRecordTypeLastDay),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::ReservedInstances": {
		Name:                 "AWS::EC2::ReservedInstances",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2ReservedInstances),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ECR::Repository": {
		Name:                 "AWS::ECR::Repository",
		ServiceName:          "ECR",
		ListDescriber:        ParallelDescribeRegional(describer.ECRRepository),
		GetDescriber:         nil,
		TerraformName:        "aws_ecr_repository",
		TerraformServiceName: "ecr",
	},
	"AWS::ElasticLoadBalancingV2::Listener": {
		Name:                 "AWS::ElasticLoadBalancingV2::Listener",
		ServiceName:          "ElasticLoadBalancingV2",
		ListDescriber:        ParallelDescribeRegional(describer.ElasticLoadBalancingV2Listener),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetElasticLoadBalancingV2Listener),
		TerraformName:        "aws_alb_listener",
		TerraformServiceName: "elbv2",
	},
	"AWS::IAM::Group": {
		Name:                 "AWS::IAM::Group",
		ServiceName:          "IAM",
		ListDescriber:        SequentialDescribeGlobal(describer.IAMGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_iam_group",
		TerraformServiceName: "iam",
	},
	"AWS::Backup::Plan": {
		Name:                 "AWS::Backup::Plan",
		ServiceName:          "Backup",
		ListDescriber:        ParallelDescribeRegional(describer.BackupPlan),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::Config::ConformancePack": {
		Name:                 "AWS::Config::ConformancePack",
		ServiceName:          "Config",
		ListDescriber:        ParallelDescribeRegional(describer.ConfigConformancePack),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CostExplorer::ByAccountDaily": {
		Name:                 "AWS::CostExplorer::ByAccountDaily",
		ServiceName:          "CostExplorer",
		ListDescriber:        SequentialDescribeGlobal(describer.CostByAccountLastDay),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::Account::Contact": {
		Name:                 "AWS::Account::Contact",
		ServiceName:          "Account",
		ListDescriber:        ParallelDescribeRegional(describer.AccountContact),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::Glue::DataQualityRuleset": {
		Name:                 "AWS::Glue::DataQualityRuleset",
		ServiceName:          "Glue",
		ListDescriber:        ParallelDescribeRegional(describer.GlueDataQualityRuleset),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EventBridge::EventBus": {
		Name:                 "AWS::EventBridge::EventBus",
		ServiceName:          "EventBridge",
		ListDescriber:        ParallelDescribeRegional(describer.EventBridgeBus),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ApiGateway::Stage": {
		Name:                 "AWS::ApiGateway::Stage",
		ServiceName:          "ApiGateway",
		ListDescriber:        ParallelDescribeRegional(describer.ApiGatewayStage),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetApiGatewayStage),
		TerraformName:        "aws_api_gateway_stage",
		TerraformServiceName: "apigateway",
	},
	"AWS::AuditManager::Framework": {
		Name:                 "AWS::AuditManager::Framework",
		ServiceName:          "AuditManager",
		ListDescriber:        ParallelDescribeRegional(describer.AuditManagerFramework),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::DynamoDb::LocalSecondaryIndex": {
		Name:                 "AWS::DynamoDb::LocalSecondaryIndex",
		ServiceName:          "DynamoDb",
		ListDescriber:        ParallelDescribeRegional(describer.DynamoDbLocalSecondaryIndex),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::EbsVolumeMetricWriteOps": {
		Name:                 "AWS::EC2::EbsVolumeMetricWriteOps",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EbsVolumeMetricWriteOps),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::Image": {
		Name:                 "AWS::EC2::Image",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2AMI),
		GetDescriber:         nil,
		TerraformName:        "aws_ami",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::Subnet": {
		Name:                 "AWS::EC2::Subnet",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2Subnet),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetEC2Subnet),
		TerraformName:        "aws_subnet",
		TerraformServiceName: "ec2",
	},
	"AWS::ECS::TaskSet": {
		Name:                 "AWS::ECS::TaskSet",
		ServiceName:          "ECS",
		ListDescriber:        ParallelDescribeRegional(describer.ECSTaskSet),
		GetDescriber:         nil,
		TerraformName:        "aws_ecs_task_set",
		TerraformServiceName: "ecs",
	},
	"AWS::Kinesis::Stream": {
		Name:                 "AWS::Kinesis::Stream",
		ServiceName:          "Kinesis",
		ListDescriber:        ParallelDescribeRegional(describer.KinesisStream),
		GetDescriber:         nil,
		TerraformName:        "aws_kinesis_stream",
		TerraformServiceName: "kinesis",
	},
	"AWS::DocDB::Cluster": {
		Name:                 "AWS::DocDB::Cluster",
		ServiceName:          "DocDB",
		ListDescriber:        ParallelDescribeRegional(describer.DocDBCluster),
		GetDescriber:         nil,
		TerraformName:        "aws_docdb_cluster",
		TerraformServiceName: "docdb",
	},
	"AWS::ElastiCache::ReplicationGroup": {
		Name:                 "AWS::ElastiCache::ReplicationGroup",
		ServiceName:          "ElastiCache",
		ListDescriber:        ParallelDescribeRegional(describer.ElastiCacheReplicationGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_elasticache_replication_group",
		TerraformServiceName: "elasticache",
	},
	"AWS::GlobalAccelerator::Accelerator": {
		Name:                 "AWS::GlobalAccelerator::Accelerator",
		ServiceName:          "GlobalAccelerator",
		ListDescriber:        ParallelDescribeRegional(describer.GlobalAcceleratorAccelerator),
		GetDescriber:         nil,
		TerraformName:        "aws_globalaccelerator_accelerator",
		TerraformServiceName: "globalaccelerator",
	},
	"AWS::CloudWatch::Metric": {
		Name:                 "AWS::CloudWatch::Metric",
		ServiceName:          "CloudWatch",
		ListDescriber:        ParallelDescribeRegional(describer.CloudWatchMetrics),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CostExplorer::ForcastMonthly": {
		Name:                 "AWS::CostExplorer::ForcastMonthly",
		ServiceName:          "CostExplorer",
		ListDescriber:        SequentialDescribeGlobal(describer.CostForecastMonthly),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EMR::InstanceGroup": {
		Name:                 "AWS::EMR::InstanceGroup",
		ServiceName:          "EMR",
		ListDescriber:        ParallelDescribeRegional(describer.EMRInstanceGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_emr_instance_group",
		TerraformServiceName: "emr",
	},
	"AWS::EC2::ManagedPrefixList": {
		Name:                 "AWS::EC2::ManagedPrefixList",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2ManagedPrefixList),
		GetDescriber:         nil,
		TerraformName:        "aws_ec2_managed_prefix_list",
		TerraformServiceName: "ec2",
	},
	"AWS::MWAA::Environment": {
		Name:                 "AWS::MWAA::Environment",
		ServiceName:          "MWAA",
		ListDescriber:        ParallelDescribeRegional(describer.MWAAEnvironment),
		GetDescriber:         nil,
		TerraformName:        "aws_mwaa_environment",
		TerraformServiceName: "mwaa",
	},
	"AWS::CloudWatch::LogResourcePolicy": {
		Name:                 "AWS::CloudWatch::LogResourcePolicy",
		ServiceName:          "CloudWatch",
		ListDescriber:        ParallelDescribeRegional(describer.CloudWatchLogsResourcePolicy),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CodeArtifact::Domain": {
		Name:                 "AWS::CodeArtifact::Domain",
		ServiceName:          "CodeArtifact",
		ListDescriber:        ParallelDescribeRegional(describer.CodeArtifactDomain),
		GetDescriber:         nil,
		TerraformName:        "aws_codeartifact_domain",
		TerraformServiceName: "codeartifact",
	},
	"AWS::CodeStar::Project": {
		Name:                 "AWS::CodeStar::Project",
		ServiceName:          "CodeStar",
		ListDescriber:        ParallelDescribeRegional(describer.CodeStarProject),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::Neptune::Database": {
		Name:                 "AWS::Neptune::Database",
		ServiceName:          "Neptune",
		ListDescriber:        ParallelDescribeRegional(describer.NeptuneDatabase),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::EbsVolumeMetricReadOps": {
		Name:                 "AWS::EC2::EbsVolumeMetricReadOps",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EbsVolumeMetricReadOps),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::TransitGatewayRoute": {
		Name:                 "AWS::EC2::TransitGatewayRoute",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2TransitGatewayRoute),
		GetDescriber:         nil,
		TerraformName:        "aws_ec2_transit_gateway_route",
		TerraformServiceName: "ec2",
	},
	"AWS::GuardDuty::Filter": {
		Name:                 "AWS::GuardDuty::Filter",
		ServiceName:          "GuardDuty",
		ListDescriber:        ParallelDescribeRegional(describer.GuardDutyFilter),
		GetDescriber:         nil,
		TerraformName:        "aws_guardduty_filter",
		TerraformServiceName: "guardduty",
	},
	"AWS::ECS::TaskDefinition": {
		Name:                 "AWS::ECS::TaskDefinition",
		ServiceName:          "ECS",
		ListDescriber:        ParallelDescribeRegional(describer.ECSTaskDefinition),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetECSTaskDefinition),
		TerraformName:        "aws_ecs_task_definition",
		TerraformServiceName: "ecs",
	},
	"AWS::GuardDuty::ThreatIntelSet": {
		Name:                 "AWS::GuardDuty::ThreatIntelSet",
		ServiceName:          "GuardDuty",
		ListDescriber:        ParallelDescribeRegional(describer.GuardDutyThreatIntelSet),
		GetDescriber:         nil,
		TerraformName:        "aws_guardduty_threatintelset",
		TerraformServiceName: "guardduty",
	},
	"AWS::ApiGatewayV2::DomainName": {
		Name:                 "AWS::ApiGatewayV2::DomainName",
		ServiceName:          "ApiGatewayV2",
		ListDescriber:        ParallelDescribeRegional(describer.ApiGatewayV2DomainName),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetApiGatewayV2DomainName),
		TerraformName:        "aws_apigatewayv2_domain_name",
		TerraformServiceName: "apigatewayv2",
	},
	"AWS::MQ::Broker": {
		Name:                 "AWS::MQ::Broker",
		ServiceName:          "MQ",
		ListDescriber:        ParallelDescribeRegional(describer.MQBroker),
		GetDescriber:         nil,
		TerraformName:        "aws_mq_broker",
		TerraformServiceName: "mq",
	},
	"AWS::ACMPCA::CertificateAuthority": {
		Name:                 "AWS::ACMPCA::CertificateAuthority",
		ServiceName:          "ACMPCA",
		ListDescriber:        ParallelDescribeRegional(describer.ACMPCACertificateAuthority),
		GetDescriber:         nil,
		TerraformName:        "aws_acmpca_certificate_authority",
		TerraformServiceName: "acmpca",
	},
	"AWS::CloudFormation::Stack": {
		Name:                 "AWS::CloudFormation::Stack",
		ServiceName:          "CloudFormation",
		ListDescriber:        ParallelDescribeRegional(describer.CloudFormationStack),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudformation_stack",
		TerraformServiceName: "cloudformation",
	},
	"AWS::DirectConnect::Connection": {
		Name:                 "AWS::DirectConnect::Connection",
		ServiceName:          "DirectConnect",
		ListDescriber:        ParallelDescribeRegional(describer.DirectConnectConnection),
		GetDescriber:         nil,
		TerraformName:        "aws_dx_connection",
		TerraformServiceName: "directconnect",
	},
	"AWS::FSX::FileSystem": {
		Name:                 "AWS::FSX::FileSystem",
		ServiceName:          "FSX",
		ListDescriber:        ParallelDescribeRegional(describer.FSXFileSystem),
		GetDescriber:         nil,
		TerraformName:        "aws_fsx_ontap_file_system",
		TerraformServiceName: "fsx",
	},
	"AWS::Glue::SecurityConfiguration": {
		Name:                 "AWS::Glue::SecurityConfiguration",
		ServiceName:          "Glue",
		ListDescriber:        ParallelDescribeRegional(describer.GlueSecurityConfiguration),
		GetDescriber:         nil,
		TerraformName:        "aws_glue_security_configuration",
		TerraformServiceName: "glue",
	},
	"AWS::Inspector::AssessmentRun": {
		Name:                 "AWS::Inspector::AssessmentRun",
		ServiceName:          "Inspector",
		ListDescriber:        ParallelDescribeRegional(describer.InspectorAssessmentRun),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::Config::ConfigurationRecorder": {
		Name:                 "AWS::Config::ConfigurationRecorder",
		ServiceName:          "Config",
		ListDescriber:        ParallelDescribeRegional(describer.ConfigConfigurationRecorder),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::NatGateway": {
		Name:                 "AWS::EC2::NatGateway",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2NatGateway),
		GetDescriber:         nil,
		TerraformName:        "aws_nat_gateway",
		TerraformServiceName: "ec2",
	},
	"AWS::ECR::PublicRepository": {
		Name:                 "AWS::ECR::PublicRepository",
		ServiceName:          "ECR",
		ListDescriber:        ParallelDescribeRegional(describer.ECRPublicRepository),
		GetDescriber:         nil,
		TerraformName:        "aws_ecrpublic_repository",
		TerraformServiceName: "ecrpublic",
	},
	"AWS::ECS::Cluster": {
		Name:                 "AWS::ECS::Cluster",
		ServiceName:          "ECS",
		ListDescriber:        ParallelDescribeRegional(describer.ECSCluster),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetECSCluster),
		TerraformName:        "aws_ecs_cluster",
		TerraformServiceName: "ecs",
	},
	"AWS::ElasticLoadBalancingV2::TargetGroup": {
		Name:                 "AWS::ElasticLoadBalancingV2::TargetGroup",
		ServiceName:          "ElasticLoadBalancingV2",
		ListDescriber:        ParallelDescribeRegional(describer.ElasticLoadBalancingV2TargetGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_alb_target_group",
		TerraformServiceName: "elbv2",
	},
	"AWS::CloudFront::CachePolicy": {
		Name:                 "AWS::CloudFront::CachePolicy",
		ServiceName:          "CloudFront",
		ListDescriber:        SequentialDescribeGlobal(describer.CloudFrontCachePolicy),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudfront_cache_policy",
		TerraformServiceName: "cloudfront",
	},
	"AWS::CloudWatch::LogStream": {
		Name:                 "AWS::CloudWatch::LogStream",
		ServiceName:          "CloudWatch",
		ListDescriber:        ParallelDescribeRegional(describer.CloudWatchLogStream),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CodeArtifact::Repository": {
		Name:                 "AWS::CodeArtifact::Repository",
		ServiceName:          "CodeArtifact",
		ListDescriber:        ParallelDescribeRegional(describer.CodeArtifactRepository),
		GetDescriber:         nil,
		TerraformName:        "aws_codeartifact_repository",
		TerraformServiceName: "codeartifact",
	},
	"AWS::AMP::Workspace": {
		Name:                 "AWS::AMP::Workspace",
		ServiceName:          "AMP",
		ListDescriber:        ParallelDescribeRegional(describer.AMPWorkspace),
		GetDescriber:         nil,
		TerraformName:        "aws_prometheus_workspace",
		TerraformServiceName: "amp",
	},
	"AWS::EC2::CapacityReservation": {
		Name:                 "AWS::EC2::CapacityReservation",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2CapacityReservation),
		GetDescriber:         nil,
		TerraformName:        "aws_ec2_capacity_reservation",
		TerraformServiceName: "ec2",
	},
	"AWS::SageMaker::NotebookInstance": {
		Name:                 "AWS::SageMaker::NotebookInstance",
		ServiceName:          "SageMaker",
		ListDescriber:        ParallelDescribeRegional(describer.SageMakerNotebookInstance),
		GetDescriber:         nil,
		TerraformName:        "aws_sagemaker_notebook_instance",
		TerraformServiceName: "sagemaker",
	},
	"AWS::EC2::VolumeSnapshot": {
		Name:                 "AWS::EC2::VolumeSnapshot",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2VolumeSnapshot),
		GetDescriber:         nil,
		TerraformName:        "aws_ebs_snapshot",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::EbsVolumeMetricWriteOpsHourly": {
		Name:                 "AWS::EC2::EbsVolumeMetricWriteOpsHourly",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EbsVolumeMetricWriteOpsHourly),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::Region": {
		Name:                 "AWS::EC2::Region",
		ServiceName:          "EC2",
		ListDescriber:        SequentialDescribeGlobal(describer.EC2Region),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::Keyspaces::Table": {
		Name:                 "AWS::Keyspaces::Table",
		ServiceName:          "Keyspaces",
		ListDescriber:        ParallelDescribeRegional(describer.KeyspacesTable),
		GetDescriber:         nil,
		TerraformName:        "aws_keyspaces_table",
		TerraformServiceName: "keyspaces",
	},
	"AWS::Config::AggregationAuthorization": {
		Name:                 "AWS::Config::AggregationAuthorization",
		ServiceName:          "Config",
		ListDescriber:        ParallelDescribeRegional(describer.ConfigAggregateAuthorization),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::DAX::SubnetGroup": {
		Name:                 "AWS::DAX::SubnetGroup",
		ServiceName:          "DAX",
		ListDescriber:        ParallelDescribeRegional(describer.DAXSubnetGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_dax_subnet_group",
		TerraformServiceName: "dax",
	},
	"AWS::DynamoDb::GlobalTable": {
		Name:                 "AWS::DynamoDb::GlobalTable",
		ServiceName:          "DynamoDb",
		ListDescriber:        ParallelDescribeRegional(describer.DynamoDbGlobalTable),
		GetDescriber:         nil,
		TerraformName:        "aws_dynamodb_global_table",
		TerraformServiceName: "dynamodb",
	},
	"AWS::ElasticLoadBalancing::LoadBalancer": {
		Name:                 "AWS::ElasticLoadBalancing::LoadBalancer",
		ServiceName:          "ElasticLoadBalancing",
		ListDescriber:        ParallelDescribeRegional(describer.ElasticLoadBalancingLoadBalancer),
		GetDescriber:         nil,
		TerraformName:        "aws_elb",
		TerraformServiceName: "elb",
	},
	"AWS::AppStream::Application": {
		Name:                 "AWS::AppStream::Application",
		ServiceName:          "AppStream",
		ListDescriber:        ParallelDescribeRegional(describer.AppStreamApplication),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::RedshiftServerless::Namespace": {
		Name:                 "AWS::RedshiftServerless::Namespace",
		ServiceName:          "RedshiftServerless",
		ListDescriber:        ParallelDescribeRegional(describer.RedshiftServerlessNamespace),
		GetDescriber:         nil,
		TerraformName:        "aws_redshiftserverless_namespace",
		TerraformServiceName: "redshiftserverless",
	},
	"AWS::CloudFront::OriginAccessIdentity": {
		Name:                 "AWS::CloudFront::OriginAccessIdentity",
		ServiceName:          "CloudFront",
		ListDescriber:        SequentialDescribeGlobal(describer.CloudFrontOriginAccessIdentity),
		GetDescriber:         nil,
		TerraformName:        "aws_cloudfront_origin_access_identity",
		TerraformServiceName: "cloudfront",
	},
	"AWS::EC2::Host": {
		Name:                 "AWS::EC2::Host",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2Host),
		GetDescriber:         nil,
		TerraformName:        "aws_ec2_host",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::VPC": {
		Name:                 "AWS::EC2::VPC",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2VPC),
		GetDescriber:         ParallelDescribeRegionalSingleResource(describer.GetEC2VPC),
		TerraformName:        "aws_vpc",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::TransitGatewayRouteTable": {
		Name:                 "AWS::EC2::TransitGatewayRouteTable",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2TransitGatewayRouteTable),
		GetDescriber:         nil,
		TerraformName:        "aws_ec2_transit_gateway_route_table",
		TerraformServiceName: "ec2",
	},
	"AWS::EKS::Nodegroup": {
		Name:                 "AWS::EKS::Nodegroup",
		ServiceName:          "EKS",
		ListDescriber:        ParallelDescribeRegional(describer.EKSNodegroup),
		GetDescriber:         nil,
		TerraformName:        "aws_eks_node_group",
		TerraformServiceName: "eks",
	},
	"AWS::Backup::Selection": {
		Name:                 "AWS::Backup::Selection",
		ServiceName:          "Backup",
		ListDescriber:        ParallelDescribeRegional(describer.BackupSelection),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CloudTrail::Import": {
		Name:                 "AWS::CloudTrail::Import",
		ServiceName:          "CloudTrail",
		ListDescriber:        ParallelDescribeRegional(describer.CloudTrailImport),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::CostExplorer::ByServiceDaily": {
		Name:                 "AWS::CostExplorer::ByServiceDaily",
		ServiceName:          "CostExplorer",
		ListDescriber:        SequentialDescribeGlobal(describer.CostByServiceLastDay),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::ElasticLoadBalancingV2::SslPolicy": {
		Name:                 "AWS::ElasticLoadBalancingV2::SslPolicy",
		ServiceName:          "ElasticLoadBalancingV2",
		ListDescriber:        ParallelDescribeRegional(describer.ElasticLoadBalancingV2SslPolicy),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::GuardDuty::Finding": {
		Name:                 "AWS::GuardDuty::Finding",
		ServiceName:          "GuardDuty",
		ListDescriber:        ParallelDescribeRegional(describer.GuardDutyFinding),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::SSM::ManagedInstanceCompliance": {
		Name:                 "AWS::SSM::ManagedInstanceCompliance",
		ServiceName:          "SSM",
		ListDescriber:        ParallelDescribeRegional(describer.SSMManagedInstanceCompliance),
		GetDescriber:         nil,
		TerraformName:        "",
		TerraformServiceName: "",
	},
	"AWS::EC2::DHCPOptions": {
		Name:                 "AWS::EC2::DHCPOptions",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2DHCPOptions),
		GetDescriber:         nil,
		TerraformName:        "aws_vpc_dhcp_options",
		TerraformServiceName: "ec2",
	},
	"AWS::EC2::InstanceType": {
		Name:                 "AWS::EC2::InstanceType",
		ServiceName:          "EC2",
		ListDescriber:        ParallelDescribeRegional(describer.EC2InstanceType),
		GetDescriber:         nil,
		TerraformName:        "aws_ec2_instance_type",
		TerraformServiceName: "ec2",
	},
	"AWS::Batch::ComputeEnvironment": {
		Name:                 "AWS::Batch::ComputeEnvironment",
		ServiceName:          "Batch",
		ListDescriber:        ParallelDescribeRegional(describer.BatchComputeEnvironment),
		GetDescriber:         nil,
		TerraformName:        "aws_batch_compute_environment",
		TerraformServiceName: "batch",
	},
	"AWS::DMS::ReplicationInstance": {
		Name:                 "AWS::DMS::ReplicationInstance",
		ServiceName:          "DMS",
		ListDescriber:        ParallelDescribeRegional(describer.DMSReplicationInstance),
		GetDescriber:         nil,
		TerraformName:        "aws_dms_replication_instance",
		TerraformServiceName: "dms",
	},
	"AWS::DynamoDb::Table": {
		Name:                 "AWS::DynamoDb::Table",
		ServiceName:          "DynamoDb",
		ListDescriber:        ParallelDescribeRegional(describer.DynamoDbTable),
		GetDescriber:         nil,
		TerraformName:        "aws_dynamodb_table",
		TerraformServiceName: "dynamodb",
	},
	"AWS::Shield::ProtectionGroup": {
		Name:                 "AWS::Shield::ProtectionGroup",
		ServiceName:          "Shield",
		ListDescriber:        ParallelDescribeRegional(describer.ShieldProtectionGroup),
		GetDescriber:         nil,
		TerraformName:        "aws_shield_protection_group",
		TerraformServiceName: "shield",
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

type Resources struct {
	Resources map[string][]describer.Resource
	Errors    map[string]string
}

func GetResources(
	ctx context.Context,
	resourceType string,
	triggerType enums.DescribeTriggerType,
	accountId string,
	regions []string,
	accessKey,
	secretKey,
	sessionToken,
	assumeRoleArn string,
	includeDisabledRegions bool,
) (*Resources, error) {
	cfg, err := GetConfig(ctx, accessKey, secretKey, sessionToken, assumeRoleArn)
	if err != nil {
		return nil, err
	}

	if len(regions) == 0 {
		cfgClone := cfg.Copy()
		cfgClone.Region = "us-east-1"

		rs, err := getAllRegions(ctx, cfgClone, includeDisabledRegions)
		if err != nil {
			return nil, err
		}

		for _, r := range rs {
			regions = append(regions, *r.RegionName)
		}
	}

	resources, err := describe(ctx, cfg, accountId, regions, resourceType, triggerType)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func GetSingleResource(
	ctx context.Context,
	resourceType string,
	triggerType enums.DescribeTriggerType,
	accountId string,
	regions []string,
	accessKey,
	secretKey,
	sessionToken,
	assumeRoleArn string,
	includeDisabledRegions bool,
	fields map[string]string,
) (*Resources, error) {
	cfg, err := GetConfig(ctx, accessKey, secretKey, sessionToken, assumeRoleArn)
	if err != nil {
		return nil, err
	}

	if len(regions) == 0 {
		cfgClone := cfg.Copy()
		cfgClone.Region = "us-east-1"

		rs, err := getAllRegions(ctx, cfgClone, includeDisabledRegions)
		if err != nil {
			return nil, err
		}

		for _, r := range rs {
			regions = append(regions, *r.RegionName)
		}
	}

	resources, err := describeSingle(ctx, cfg, accountId, regions, resourceType, fields, triggerType)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func describeSingle(
	ctx context.Context,
	cfg aws.Config,
	account string,
	regions []string,
	resourceType string,
	fields map[string]string,
	triggerType enums.DescribeTriggerType) (*Resources, error) {
	resourceTypeObject, ok := resourceTypes[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	return resourceTypeObject.GetDescriber(ctx, cfg, account, regions, resourceType, fields, triggerType)
}

func describe(
	ctx context.Context,
	cfg aws.Config,
	account string,
	regions []string,
	resourceType string,
	triggerType enums.DescribeTriggerType) (*Resources, error) {
	resourceTypeObject, ok := resourceTypes[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	return resourceTypeObject.ListDescriber(ctx, cfg, account, regions, resourceType, triggerType)
}

func ParallelDescribeRegionalSingleResource(describe func(context.Context, aws.Config, map[string]string) ([]describer.Resource, error)) SingleResourceDescriber {
	type result struct {
		region    string
		resources []describer.Resource
		err       error
	}
	return func(ctx context.Context, cfg aws.Config, account string, regions []string, rType string, fields map[string]string, triggerType enums.DescribeTriggerType) (*Resources, error) {
		input := make(chan result, len(regions))
		for _, region := range regions {
			go func(r string) {
				defer func() {
					if err := recover(); err != nil {
						input <- result{region: r, resources: nil, err: fmt.Errorf("paniced: %v", err)}
					}
				}()
				// Make a shallow copy and override the default region
				rCfg := cfg.Copy()
				rCfg.Region = r

				partition, _ := partitionOf(r)
				ctx = describer.WithDescribeContext(ctx, describer.DescribeContext{
					AccountID: account,
					Region:    r,
					Partition: partition,
				})
				ctx = describer.WithTriggerType(ctx, triggerType)
				resources, err := describe(ctx, rCfg, fields)
				input <- result{region: r, resources: resources, err: err}
			}(region)
		}

		output := Resources{
			Resources: make(map[string][]describer.Resource, len(regions)),
			Errors:    make(map[string]string, len(regions)),
		}
		for range regions {
			resp := <-input
			if resp.err != nil {
				if !IsUnsupportedOrInvalidError(rType, resp.region, resp.err) {
					output.Errors[resp.region] = resp.err.Error()
					continue
				}
			}

			if resp.resources == nil {
				resp.resources = []describer.Resource{}
			}

			partition, _ := partitionOf(resp.region)
			for i := range resp.resources {
				resp.resources[i].Account = account
				resp.resources[i].Region = resp.region
				resp.resources[i].Partition = partition
				resp.resources[i].Type = rType
			}

			output.Resources[resp.region] = resp.resources
		}

		return &output, nil
	}
}

// Parallel describe the resources across the reigons. Failure in one regions won't affect
// other regions.
func ParallelDescribeRegional(describe func(context.Context, aws.Config) ([]describer.Resource, error)) ResourceDescriber {
	type result struct {
		region    string
		resources []describer.Resource
		err       error
	}
	return func(ctx context.Context, cfg aws.Config, account string, regions []string, rType string, triggerType enums.DescribeTriggerType) (*Resources, error) {
		input := make(chan result, len(regions))
		for _, region := range regions {
			go func(r string) {
				defer func() {
					if err := recover(); err != nil {
						input <- result{region: r, resources: nil, err: fmt.Errorf("paniced: %v", err)}
					}
				}()
				// Make a shallow copy and override the default region
				rCfg := cfg.Copy()
				rCfg.Region = r

				partition, _ := partitionOf(r)
				ctx = describer.WithDescribeContext(ctx, describer.DescribeContext{
					AccountID: account,
					Region:    r,
					Partition: partition,
				})
				ctx = describer.WithTriggerType(ctx, triggerType)
				resources, err := describe(ctx, rCfg)
				input <- result{region: r, resources: resources, err: err}
			}(region)
		}

		output := Resources{
			Resources: make(map[string][]describer.Resource, len(regions)),
			Errors:    make(map[string]string, len(regions)),
		}
		for range regions {
			resp := <-input
			if resp.err != nil {
				if !IsUnsupportedOrInvalidError(rType, resp.region, resp.err) {
					output.Errors[resp.region] = resp.err.Error()
					continue
				}
			}

			if resp.resources == nil {
				resp.resources = []describer.Resource{}
			}

			partition, _ := partitionOf(resp.region)
			for i := range resp.resources {
				resp.resources[i].Account = account
				resp.resources[i].Region = resp.region
				resp.resources[i].Partition = partition
				resp.resources[i].Type = rType
			}

			output.Resources[resp.region] = resp.resources
		}

		return &output, nil
	}
}

// Sequentially describe the resources. If anyone of the regions fails, it will move on to the next region.
func SequentialDescribeGlobal(describe func(context.Context, aws.Config) ([]describer.Resource, error)) ResourceDescriber {
	return func(ctx context.Context, cfg aws.Config, account string, regions []string, rType string, triggerType enums.DescribeTriggerType) (*Resources, error) {
		output := Resources{
			Resources: make(map[string][]describer.Resource, len(regions)),
			Errors:    make(map[string]string, len(regions)),
		}

		for _, region := range regions {
			// Make a shallow copy and override the default region
			rCfg := cfg.Copy()
			rCfg.Region = region

			partition, _ := partitionOf(region)
			ctx = describer.WithDescribeContext(ctx, describer.DescribeContext{
				AccountID: account,
				Region:    region,
				Partition: partition,
			})
			ctx = describer.WithTriggerType(ctx, triggerType)
			resources, err := describe(ctx, rCfg)
			if err != nil {
				if !IsUnsupportedOrInvalidError(rType, region, err) {
					output.Errors[region] = err.Error()
				}
				continue
			}

			if resources == nil {
				resources = []describer.Resource{}
			}

			for i := range resources {
				resources[i].Account = account
				resources[i].Region = "global"
				resources[i].Partition = partition
				resources[i].Type = rType
			}

			output.Resources[region] = resources

			// Stop describing as soon as one region has returned a successful response
			break
		}

		return &output, nil
	}
}

// Sequentially describe the resources. If anyone of the regions fails, it will move on to the next region.
// This utility is specific to S3 usecase.
func SequentialDescribeS3(describe func(context.Context, aws.Config, []string) (map[string][]describer.Resource, error)) ResourceDescriber {
	return func(ctx context.Context, cfg aws.Config, account string, regions []string, rType string, triggerType enums.DescribeTriggerType) (*Resources, error) {
		output := Resources{
			Resources: make(map[string][]describer.Resource, len(regions)),
			Errors:    make(map[string]string, len(regions)),
		}

		for _, region := range regions {
			// Make a shallow copy and override the default region
			rCfg := cfg.Copy()
			rCfg.Region = region

			partition, _ := partitionOf(region)
			ctx = describer.WithDescribeContext(ctx, describer.DescribeContext{
				AccountID: account,
				Region:    region,
				Partition: partition,
			})
			ctx = describer.WithTriggerType(ctx, triggerType)
			resources, err := describe(ctx, rCfg, regions)
			if err != nil {
				if !IsUnsupportedOrInvalidError(rType, region, err) {
					output.Errors[region] = err.Error()
				}
				continue
			}

			if resources != nil {
				output.Resources = resources

			}

			// Stop describing as soon as one region has returned a successful response
			break
		}

		for region, resources := range output.Resources {
			partition, _ := partitionOf(region)

			for j, resource := range resources {
				resource.Account = account
				resource.Region = region
				resource.Partition = partition
				resource.Type = rType

				output.Resources[region][j] = resource
			}
		}

		return &output, nil
	}
}
