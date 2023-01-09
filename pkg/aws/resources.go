package aws

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/describer"
)

type ResourceDescriber func(context.Context, aws.Config, string, []string, string) (*Resources, error)

var resourceTypeToDescriber = map[string]ResourceDescriber{
	"AWS::AccessAnalyzer::Analyzer":     ParallelDescribeRegional(describer.AccessAnalyzerAnalyzer),
	"AWS::ACMPCA::CertificateAuthority": ParallelDescribeRegional(describer.ACMPCACertificateAuthority),
	// "AWS::ApplicationInsights::Application":                       ParallelDescribeRegional(describer.ApplicationInsightsApplication),
	"AWS::ApplicationAutoScaling::Target":   ParallelDescribeRegional(describer.ApplicationAutoScalingTarget), // IGNORE
	"AWS::AutoScaling::AutoScalingGroup":    ParallelDescribeRegional(describer.AutoScalingAutoScalingGroup),
	"AWS::AutoScaling::LaunchConfiguration": ParallelDescribeRegional(describer.AutoScalingLaunchConfiguration),
	// "AWS::AutoScaling::LifecycleHook":                             ParallelDescribeRegional(describer.AutoScalingLifecycleHook),
	// "AWS::AutoScaling::ScalingPolicy":                             ParallelDescribeRegional(describer.AutoScalingScalingPolicy),
	// "AWS::AutoScaling::ScheduledAction":                           ParallelDescribeRegional(describer.AutoScalingScheduledAction),
	// "AWS::AutoScaling::WarmPool":                                  ParallelDescribeRegional(describer.AutoScalingWarmPool),
	"AWS::Backup::Plan":              ParallelDescribeRegional(describer.BackupPlan),
	"AWS::Backup::Selection":         ParallelDescribeRegional(describer.BackupSelection),
	"AWS::Backup::RecoveryPoint":     ParallelDescribeRegional(describer.BackupRecoveryPoint),
	"AWS::Backup::ProtectedResource": ParallelDescribeRegional(describer.BackupProtectedResource),
	"AWS::Backup::Vault":             ParallelDescribeRegional(describer.BackupVault),
	"AWS::Batch::ComputeEnvironment": ParallelDescribeRegional(describer.BatchComputeEnvironment),
	"AWS::Batch::Job":                ParallelDescribeRegional(describer.BatchJob),
	// "AWS::CertificateManager::Account":                            SequentialDescribeGlobal(describer.CertificateManagerAccount),
	"AWS::CertificateManager::Certificate": ParallelDescribeRegional(describer.CertificateManagerCertificate),
	"AWS::CloudFormation::Stack":           ParallelDescribeRegional(describer.CloudFormationStack),
	"AWS::CloudFormation::StackSet":        ParallelDescribeRegional(describer.CloudFormationStackSet),
	"AWS::CloudFront::Distribution":        SequentialDescribeGlobal(describer.CloudFrontDistribution),
	"AWS::CloudFront::OriginAccessControl": SequentialDescribeGlobal(describer.CloudFrontOriginAccessControl),
	"AWS::CloudTrail::Trail":               ParallelDescribeRegional(describer.CloudTrailTrail),
	"AWS::CloudWatch::Alarm":               ParallelDescribeRegional(describer.CloudWatchAlarm),
	// "AWS::CloudWatch::AnomalyDetector":                            ParallelDescribeRegional(describer.CloudWatchAnomalyDetector),
	// "AWS::CloudWatch::CompositeAlarm":                             ParallelDescribeRegional(describer.CloudWatchCompositeAlarm),
	// "AWS::CloudWatch::Dashboard":                                  ParallelDescribeRegional(describer.CloudWatchDashboard),
	// "AWS::CloudWatch::InsightRule":                                ParallelDescribeRegional(describer.CloudWatchInsightRule),
	// "AWS::CloudWatch::MetricStream":                               ParallelDescribeRegional(describer.CloudWatchMetricStream),
	"AWS::CodeArtifact::Repository":       ParallelDescribeRegional(describer.CodeArtifactRepository),
	"AWS::CodeBuild::Project":             ParallelDescribeRegional(describer.CodeBuildProject),
	"AWS::CodeBuild::SourceCredential":    ParallelDescribeRegional(describer.CodeBuildSourceCredential),
	"AWS::CodeCommit::Repository":         ParallelDescribeRegional(describer.CodeCommitRepository),
	"AWS::CodeDeploy::Application":        ParallelDescribeRegional(describer.CodeDeployApplication),
	"AWS::CodeDeploy::DeploymentGroup":    ParallelDescribeRegional(describer.CodeDeployDeploymentGroup),
	"AWS::CodePipeline::Pipeline":         ParallelDescribeRegional(describer.CodePipelinePipeline),
	"AWS::CodeStar::Project":              ParallelDescribeRegional(describer.CodeStarProject),
	"AWS::Config::ConfigurationRecorder":  ParallelDescribeRegional(describer.ConfigConfigurationRecorder),
	"AWS::DAX::Cluster":                   ParallelDescribeRegional(describer.DAXCluster),
	"AWS::DirectConnect::Connection":      ParallelDescribeRegional(describer.DirectConnectConnection),
	"AWS::DirectConnect::Gateway":         ParallelDescribeRegional(describer.DirectConnectGateway),
	"AWS::DirectoryService::Directory":    ParallelDescribeRegional(describer.DirectoryServiceDirectory),
	"AWS::DMS::ReplicationInstance":       ParallelDescribeRegional(describer.DMSReplicationInstance),
	"AWS::DRS::RecoveryInstance":          ParallelDescribeRegional(describer.DRSRecoveryInstance),
	"AWS::DRS::SourceServer":              ParallelDescribeRegional(describer.DRSSourceServer),
	"AWS::DynamoDb::BackUp":               ParallelDescribeRegional(describer.DynamoDbBackUp),
	"AWS::DynamoDb::GlobalSecondaryIndex": ParallelDescribeRegional(describer.DynamoDbGlobalSecondaryIndex),
	"AWS::DynamoDb::LocalSecondaryIndex":  ParallelDescribeRegional(describer.DynamoDbLocalSecondaryIndex),
	"AWS::DynamoDb::GlobalTable":          ParallelDescribeRegional(describer.DynamoDbGlobalTable),
	"AWS::DynamoDbStreams::Stream":        ParallelDescribeRegional(describer.DynamoDbStream),
	"AWS::DynamoDb::Table":                ParallelDescribeRegional(describer.DynamoDbTable),
	"AWS::EC2::VolumeSnapshot":            ParallelDescribeRegional(describer.EC2VolumeSnapshot),
	"AWS::EC2::Volume":                    ParallelDescribeRegional(describer.EC2Volume),
	"AWS::EC2::CapacityReservation":       ParallelDescribeRegional(describer.EC2CapacityReservation),
	"AWS::EC2::CapacityReservationFleet":  ParallelDescribeRegional(describer.EC2CapacityReservationFleet),
	// "AWS::EC2::CarrierGateway":                                    ParallelDescribeRegional(describer.EC2CarrierGateway),
	// "AWS::EC2::ClientVpnAuthorizationRule":                        ParallelDescribeRegional(describer.EC2ClientVpnAuthorizationRule),
	// "AWS::EC2::ClientVpnEndpoint":                                 ParallelDescribeRegional(describer.EC2ClientVpnEndpoint),
	// "AWS::EC2::ClientVpnRoute":                                    ParallelDescribeRegional(describer.EC2ClientVpnRoute),
	// "AWS::EC2::ClientVpnTargetNetworkAssociation":                 ParallelDescribeRegional(describer.EC2ClientVpnTargetNetworkAssociation),
	// "AWS::EC2::CustomerGateway":                                   ParallelDescribeRegional(describer.EC2CustomerGateway),
	"AWS::EC2::DHCPOptions":               ParallelDescribeRegional(describer.EC2DHCPOptions),
	"AWS::EC2::Fleet":                     ParallelDescribeRegional(describer.EC2Fleet),
	"AWS::EC2::EIP":                       ParallelDescribeRegional(describer.EC2EIP),
	"AWS::EC2::EgressOnlyInternetGateway": ParallelDescribeRegional(describer.EC2EgressOnlyInternetGateway),
	// "AWS::EC2::EnclaveCertificateIamRoleAssociation":              ParallelDescribeRegional(describer.EC2EnclaveCertificateIamRoleAssociation),
	"AWS::EC2::FlowLog":                   ParallelDescribeRegional(describer.EC2FlowLog),
	"AWS::EC2::Host":                      ParallelDescribeRegional(describer.EC2Host),
	"AWS::EC2::Image":                     ParallelDescribeRegional(describer.EC2AMI),
	"AWS::EC2::Instance":                  ParallelDescribeRegional(describer.EC2Instance),
	"AWS::EC2::Ipam":                      ParallelDescribeRegional(describer.EC2Ipam),
	"AWS::EC2::IpamPool":                  ParallelDescribeRegional(describer.EC2IpamPool),
	"AWS::CostExplorer::ByAccountMonthly": SequentialDescribeGlobal(describer.CostByAccountLastMonth),
	"AWS::CostExplorer::ByServiceMonthly": SequentialDescribeGlobal(describer.CostByServiceLastMonth),
	"AWS::CostExplorer::ByAccountDaily":   SequentialDescribeGlobal(describer.CostByAccountLastDay),
	"AWS::CostExplorer::ByServiceDaily":   SequentialDescribeGlobal(describer.CostByServiceLastDay),
	"AWS::EC2::InternetGateway":           ParallelDescribeRegional(describer.EC2InternetGateway),
	"AWS::EC2::KeyPair":                   ParallelDescribeRegional(describer.EC2KeyPair),
	// "AWS::EC2::LaunchTemplate":                                    ParallelDescribeRegional(describer.EC2LaunchTemplate),
	// "AWS::EC2::LocalGatewayRouteTable":                            ParallelDescribeRegional(describer.EC2LocalGatewayRouteTable),
	// "AWS::EC2::LocalGatewayRouteTableVPCAssociation":              ParallelDescribeRegional(describer.EC2LocalGatewayRouteTableVPCAssociation),
	"AWS::EC2::NatGateway": ParallelDescribeRegional(describer.EC2NatGateway),
	"AWS::EC2::NetworkAcl": ParallelDescribeRegional(describer.EC2NetworkAcl),
	// "AWS::EC2::NetworkInsightsAnalysis":                           ParallelDescribeRegional(describer.EC2NetworkInsightsAnalysis),
	// "AWS::EC2::NetworkInsightsPath":                               ParallelDescribeRegional(describer.EC2NetworkInsightsPath),
	"AWS::EC2::NetworkInterface": ParallelDescribeRegional(describer.EC2NetworkInterface),
	// "AWS::EC2::NetworkInterfacePermission":                        ParallelDescribeRegional(describer.EC2NetworkInterfacePermission),
	"AWS::EC2::PlacementGroup": ParallelDescribeRegional(describer.EC2PlacementGroup),
	// "AWS::EC2::PrefixList":                                        ParallelDescribeRegional(describer.EC2PrefixList),
	"AWS::EC2::RouteTable":        ParallelDescribeRegional(describer.EC2RouteTable),
	"AWS::EC2::Region":            SequentialDescribeGlobal(describer.EC2Region),
	"AWS::EC2::RegionalSettings":  ParallelDescribeRegional(describer.EC2RegionalSettings), // IGNORE
	"AWS::EC2::ReservedInstances": ParallelDescribeRegional(describer.EC2ReservedInstances),
	"AWS::EC2::SecurityGroup":     ParallelDescribeRegional(describer.EC2SecurityGroup),
	"AWS::EC2::SecurityGroupRule": ParallelDescribeRegional(describer.EC2SecurityGroupRule),
	// "AWS::EC2::SpotFleet":                                         ParallelDescribeRegional(describer.EC2SpotFleet),
	"AWS::EC2::Subnet": ParallelDescribeRegional(describer.EC2Subnet),
	// "AWS::EC2::TrafficMirrorFilter":                               ParallelDescribeRegional(describer.EC2TrafficMirrorFilter),
	// "AWS::EC2::TrafficMirrorSession":                              ParallelDescribeRegional(describer.EC2TrafficMirrorSession),
	// "AWS::EC2::TrafficMirrorTarget":                               ParallelDescribeRegional(describer.EC2TrafficMirrorTarget),
	"AWS::EC2::TransitGateway": ParallelDescribeRegional(describer.EC2TransitGateway),
	// "AWS::EC2::TransitGatewayAttachment":                          ParallelDescribeRegional(describer.EC2TransitGatewayAttachment),
	// "AWS::EC2::TransitGatewayConnect":                             ParallelDescribeRegional(describer.EC2TransitGatewayConnect),
	// "AWS::EC2::TransitGatewayMulticastDomain":                     ParallelDescribeRegional(describer.EC2TransitGatewayMulticastDomain),
	// "AWS::EC2::TransitGatewayMulticastDomainAssociation":          ParallelDescribeRegional(describer.EC2TransitGatewayMulticastDomainAssociation),
	// "AWS::EC2::TransitGatewayMulticastGroupMember":                ParallelDescribeRegional(describer.EC2TransitGatewayMulticastGroupMember),
	// "AWS::EC2::TransitGatewayMulticastGroupSource":                ParallelDescribeRegional(describer.EC2TransitGatewayMulticastGroupSource),
	// "AWS::EC2::TransitGatewayPeeringAttachment":                   ParallelDescribeRegional(describer.EC2TransitGatewayPeeringAttachment),
	"AWS::EC2::TransitGatewayRouteTable": ParallelDescribeRegional(describer.EC2TransitGatewayRouteTable),
	// "AWS::EC2::TransitGatewayRouteTableAssociation":               ParallelDescribeRegional(describer.EC2TransitGatewayRouteTableAssociation),
	// "AWS::EC2::TransitGatewayRouteTablePropagation":               ParallelDescribeRegional(describer.EC2TransitGatewayRouteTablePropagation),
	"AWS::EC2::VPC":         ParallelDescribeRegional(describer.EC2VPC),
	"AWS::EC2::VPCEndpoint": ParallelDescribeRegional(describer.EC2VPCEndpoint),
	// "AWS::EC2::VPCEndpointConnectionNotification":                 ParallelDescribeRegional(describer.EC2VPCEndpointConnectionNotification),
	"AWS::EC2::VPCEndpointService": ParallelDescribeRegional(describer.EC2VPCEndpointService),
	// "AWS::EC2::VPCEndpointServicePermissions":                     ParallelDescribeRegional(describer.EC2VPCEndpointServicePermissions),
	"AWS::EC2::VPCPeeringConnection": ParallelDescribeRegional(describer.EC2VPCPeeringConnection),
	"AWS::EC2::VPNConnection":        ParallelDescribeRegional(describer.EC2VPNConnection),
	// "AWS::EC2::VPNGateway":                                        ParallelDescribeRegional(describer.EC2VPNGateway),
	"AWS::ECR::PublicRepository": ParallelDescribeRegional(describer.ECRPublicRepository),
	"AWS::ECR::PublicRegistry":   SequentialDescribeGlobal(describer.ECRPublicRegistry),
	// "AWS::ECR::Registry":                                          SequentialDescribeGlobal(describer.ECRRegistry),
	// "AWS::ECR::RegistryPolicy":                                    SequentialDescribeGlobal(describer.ECRRegistryPolicy),
	"AWS::ECR::Repository": ParallelDescribeRegional(describer.ECRRepository),
	// "AWS::ECS::CapacityProvider":                                  ParallelDescribeRegional(describer.ECSCapacityProvider),
	"AWS::ECS::Cluster":           ParallelDescribeRegional(describer.ECSCluster),
	"AWS::ECS::ContainerInstance": ParallelDescribeRegional(describer.ECSContainerInstance),
	"AWS::ECS::Service":           ParallelDescribeRegional(describer.ECSService),
	"AWS::ECS::TaskDefinition":    ParallelDescribeRegional(describer.ECSTaskDefinition),
	"AWS::ECS::TaskSet":           ParallelDescribeRegional(describer.ECSTaskSet),
	//"AWS::EFS::AccessPoint":    ParallelDescribeRegional(describer.EFSAccessPoint),
	"AWS::EFS::FileSystem": ParallelDescribeRegional(describer.EFSFileSystem),
	//"AWS::EFS::MountTarget":    ParallelDescribeRegional(describer.EFSMountTarget),
	"AWS::EKS::Addon":   ParallelDescribeRegional(describer.EKSAddon),
	"AWS::EKS::Cluster": ParallelDescribeRegional(describer.EKSCluster),
	// "AWS::EKS::FargateProfile":                                    ParallelDescribeRegional(describer.EKSFargateProfile),
	"AWS::EKS::Nodegroup": ParallelDescribeRegional(describer.EKSNodegroup),
	// "AWS::EKS::IdentityProviderConfig":                            ParallelDescribeRegional(describer.EKSIdentityProviderConfig),
	"AWS::ElasticBeanstalk::Application":        ParallelDescribeRegional(describer.ElasticBeanstalkApplication),
	"AWS::ElasticBeanstalk::Environment":        ParallelDescribeRegional(describer.ElasticBeanstalkEnvironment),
	"AWS::ElasticBeanstalk::Platform":           ParallelDescribeRegional(describer.ElasticBeanstalkPlatform),
	"AWS::ElastiCache::ReplicationGroup":        ParallelDescribeRegional(describer.ElastiCacheReplicationGroup),
	"AWS::ElastiCache::Cluster":                 ParallelDescribeRegional(describer.ElastiCacheCluster),
	"AWS::ElasticLoadBalancing::LoadBalancer":   ParallelDescribeRegional(describer.ElasticLoadBalancingLoadBalancer), // IGNORE
	"AWS::ElasticLoadBalancingV2::Listener":     ParallelDescribeRegional(describer.ElasticLoadBalancingV2Listener),
	"AWS::ElasticLoadBalancingV2::ListenerRule": ParallelDescribeRegional(describer.ElasticLoadBalancingV2ListenerRule),
	"AWS::ElasticLoadBalancingV2::LoadBalancer": ParallelDescribeRegional(describer.ElasticLoadBalancingV2LoadBalancer),
	"AWS::ElasticLoadBalancingV2::TargetGroup":  ParallelDescribeRegional(describer.ElasticLoadBalancingV2TargetGroup),
	"AWS::ElasticSearch::Domain":                ParallelDescribeRegional(describer.ESDomain),
	"AWS::EMR::Cluster":                         ParallelDescribeRegional(describer.EMRCluster),
	"AWS::EventBridge::EventBus":                ParallelDescribeRegional(describer.EventBridgeBus),
	"AWS::FMS::Policy":                          ParallelDescribeRegional(describer.FMSPolicy),
	"AWS::FSX::FileSystem":                      ParallelDescribeRegional(describer.FSXFileSystem),
	"AWS::FSX::Snapshot":                        ParallelDescribeRegional(describer.FSXSnapshot),
	"AWS::FSX::StorageVirtualMachine":           ParallelDescribeRegional(describer.FSXStorageVirtualMachine),
	"AWS::FSX::Task":                            ParallelDescribeRegional(describer.FSXTask),
	"AWS::FSX::Volume":                          ParallelDescribeRegional(describer.FSXVolume),
	"AWS::GuardDuty::Finding":                   ParallelDescribeRegional(describer.GuardDutyFinding),
	"AWS::GuardDuty::Detector":                  ParallelDescribeRegional(describer.GuardDutyDetector),
	"AWS::IAM::AccessKey":                       SequentialDescribeGlobal(describer.IAMAccessKey),
	"AWS::IAM::Account":                         SequentialDescribeGlobal(describer.IAMAccount),
	"AWS::IAM::IAMAccountPasswordPolicy":        SequentialDescribeGlobal(describer.IAMAccountPasswordPolicy),
	"AWS::IAM::AccountSummary":                  SequentialDescribeGlobal(describer.IAMAccountSummary),
	"AWS::IAM::CredentialReport":                SequentialDescribeGlobal(describer.IAMCredentialReport), // IGNORE
	"AWS::IAM::Group":                           SequentialDescribeGlobal(describer.IAMGroup),
	// "AWS::IAM::InstanceProfile":                                   SequentialDescribeGlobal(describer.IAMInstanceProfile),
	// "AWS::IAM::ManagedPolicy":                                     SequentialDescribeGlobal(describer.IAMManagedPolicy),
	// "AWS::IAM::OIDCProvider":                                      SequentialDescribeGlobal(describer.IAMOIDCProvider),
	// "AWS::IAM::GroupPolicy":                                       SequentialDescribeGlobal(describer.IAMGroupPolicy),
	"AWS::IAM::Policy": SequentialDescribeGlobal(describer.IAMPolicy),
	// "AWS::IAM::UserPolicy":                                        SequentialDescribeGlobal(describer.IAMUserPolicy),
	// "AWS::IAM::RolePolicy":                                        SequentialDescribeGlobal(describer.IAMRolePolicy),
	"AWS::IAM::Role": SequentialDescribeGlobal(describer.IAMRole),
	// "AWS::IAM::SAMLProvider":                                      SequentialDescribeGlobal(describer.IAMSAMLProvider),
	"AWS::IAM::ServerCertificate": SequentialDescribeGlobal(describer.IAMServerCertificate),
	"AWS::IAM::User":              SequentialDescribeGlobal(describer.IAMUser),
	"AWS::IAM::VirtualMFADevice":  SequentialDescribeGlobal(describer.IAMVirtualMFADevice), // IGNORE
	"AWS::ImageBuilder::Image":    ParallelDescribeRegional(describer.ImageBuilderImage),
	"AWS::AMP::Workspace":         ParallelDescribeRegional(describer.AMPWorkspace),
	"AWS::ApiGateway::Stage":      ParallelDescribeRegional(describer.ApiGatewayStage),
	"AWS::ApiGateway::RestApi":    ParallelDescribeRegional(describer.ApiGatewayRestAPI),
	"AWS::ApiGatewayV2::Api":      ParallelDescribeRegional(describer.ApiGatewayV2API),
	"AWS::AppStream::Application": ParallelDescribeRegional(describer.AppStreamApplication),
	"AWS::AppStream::Fleet":       ParallelDescribeRegional(describer.AppStreamFleet),
	"AWS::AppStream::Stack":       ParallelDescribeRegional(describer.AppStreamStack),
	"AWS::Glacier::Vault":         ParallelDescribeRegional(describer.GlacierVault),
	"AWS::Grafana::Workspace":     ParallelDescribeRegional(describer.GrafanaWorkspace),
	// "AWS::KMS::Alias":                                             ParallelDescribeRegional(describer.KMSAlias),
	"AWS::Kafka::Cluster":      ParallelDescribeRegional(describer.KafkaCluster),
	"AWS::Keyspaces::Keyspace": ParallelDescribeRegional(describer.KeyspacesKeyspace),
	"AWS::Keyspaces::Table":    ParallelDescribeRegional(describer.KeyspacesTable),
	"AWS::KMS::Key":            ParallelDescribeRegional(describer.KMSKey),
	"AWS::Kinesis::Stream":     ParallelDescribeRegional(describer.KinesisStream),
	// "AWS::Lambda::Alias":                                          ParallelDescribeRegional(describer.LambdaAlias),
	// "AWS::Lambda::CodeSigningConfig":                              ParallelDescribeRegional(describer.LambdaCodeSigningConfig),
	// "AWS::Lambda::EventInvokeConfig":                              ParallelDescribeRegional(describer.LambdaEventInvokeConfig),
	// "AWS::Lambda::EventSourceMapping":                             ParallelDescribeRegional(describer.LambdaEventSourceMapping),
	"AWS::Lambda::Function":        ParallelDescribeRegional(describer.LambdaFunction),
	"AWS::Lambda::FunctionVersion": ParallelDescribeRegional(describer.LambdaFunctionVersion),
	// "AWS::Lambda::LayerVersion":                                   ParallelDescribeRegional(describer.LambdaLayerVersion),
	// "AWS::Lambda::LayerVersionPermission":                         ParallelDescribeRegional(describer.LambdaLayerVersionPermission),
	// "AWS::Lambda::Permission":                                     ParallelDescribeRegional(describer.LambdaPermission),
	// "AWS::Logs::Destination":                                      ParallelDescribeRegional(describer.CloudWatchLogsDestination),
	"AWS::Logs::LogGroup": ParallelDescribeRegional(describer.CloudWatchLogsLogGroup),
	// "AWS::Logs::LogStream":                                        ParallelDescribeRegional(describer.CloudWatchLogsLogStream),
	"AWS::Logs::MetricFilter": ParallelDescribeRegional(describer.CloudWatchLogsMetricFilter),
	// "AWS::Logs::QueryDefinition":                                  ParallelDescribeRegional(describer.CloudWatchLogsQueryDefinition),
	// "AWS::Logs::ResourcePolicy":                                   ParallelDescribeRegional(describer.CloudWatchLogsResourcePolicy),
	// "AWS::Logs::SubscriptionFilter":                               ParallelDescribeRegional(describer.CloudWatchLogsSubscriptionFilter),
	"AWS::MemoryDb::Cluster":           ParallelDescribeRegional(describer.MemoryDbCluster),
	"AWS::MQ::Broker":                  ParallelDescribeRegional(describer.MQBroker),
	"AWS::MWAA::Environment":           ParallelDescribeRegional(describer.MWAAEnvironment),
	"AWS::Neptune::Database":           ParallelDescribeRegional(describer.NeptuneDatabase),
	"AWS::NetworkFirewall::Firewall":   ParallelDescribeRegional(describer.NetworkFirewallFirewall),
	"AWS::OpenSearch::Domain":          ParallelDescribeRegional(describer.OpenSearchDomain),
	"AWS::OpsWorksCM::Server":          ParallelDescribeRegional(describer.OpsWorksCMServer),
	"AWS::Organizations::Organization": ParallelDescribeRegional(describer.OrganizationsOrganization),
	"AWS::RDS::DBCluster":              ParallelDescribeRegional(describer.RDSDBCluster),
	"AWS::RDS::DBClusterSnapshot":      ParallelDescribeRegional(describer.RDSDBClusterSnapshot),
	// "AWS::RDS::DBClusterParameterGroup":                           ParallelDescribeRegional(describer.RDSDBClusterParameterGroup),
	"AWS::RDS::DBInstance": ParallelDescribeRegional(describer.RDSDBInstance),
	// "AWS::RDS::DBParameterGroup":                                  ParallelDescribeRegional(describer.RDSDBParameterGroup),
	// "AWS::RDS::DBProxy":                                           ParallelDescribeRegional(describer.RDSDBProxy),
	// "AWS::RDS::DBProxyEndpoint":                                   ParallelDescribeRegional(describer.RDSDBProxyEndpoint),
	// "AWS::RDS::DBProxyTargetGroup":                                ParallelDescribeRegional(describer.RDSDBProxyTargetGroup),
	// "AWS::RDS::DBSecurityGroup":                                   ParallelDescribeRegional(describer.RDSDBSecurityGroup),
	// "AWS::RDS::DBSubnetGroup":                                     ParallelDescribeRegional(describer.RDSDBSubnetGroup),
	"AWS::RDS::DBEventSubscription": ParallelDescribeRegional(describer.RDSDBEventSubscription),
	"AWS::RDS::DBSnapshot":          ParallelDescribeRegional(describer.RDSDBSnapshot),
	"AWS::RDS::GlobalCluster":       ParallelDescribeRegional(describer.RDSGlobalCluster),
	// "AWS::RDS::OptionGroup":                                       ParallelDescribeRegional(describer.RDSOptionGroup),
	"AWS::Redshift::Cluster":               ParallelDescribeRegional(describer.RedshiftCluster),
	"AWS::Redshift::ClusterParameterGroup": ParallelDescribeRegional(describer.RedshiftClusterParameterGroup),
	"AWS::Redshift::Snapshot":              ParallelDescribeRegional(describer.RedshiftSnapshot),
	"AWS::RedshiftServerless::Namespace":   ParallelDescribeRegional(describer.RedshiftServerlessNamespace),
	"AWS::RedshiftServerless::Snapshot":    ParallelDescribeRegional(describer.RedshiftServerlessSnapshot),
	// "AWS::Redshift::ClusterSecurityGroup":                         ParallelDescribeRegional(describer.RedshiftClusterSecurityGroup),
	// "AWS::Redshift::ClusterSubnetGroup":                           ParallelDescribeRegional(describer.RedshiftClusterSubnetGroup),
	// "AWS::Route53::DNSSEC":                                        SequentialDescribeGlobal(describer.Route53DNSSEC),
	// "AWS::Route53::HealthCheck":                                   SequentialDescribeGlobal(describer.Route53HealthCheck),
	"AWS::Route53::HostedZone": SequentialDescribeGlobal(describer.Route53HostedZone),
	// "AWS::Route53::RecordSet":                                     SequentialDescribeGlobal(describer.Route53RecordSet),
	// "AWS::Route53Resolver::FirewallDomainList":                    ParallelDescribeRegional(describer.Route53ResolverFirewallDomainList),
	// "AWS::Route53Resolver::FirewallRuleGroup":                     ParallelDescribeRegional(describer.Route53ResolverFirewallRuleGroup),
	// "AWS::Route53Resolver::FirewallRuleGroupAssociation":          ParallelDescribeRegional(describer.Route53ResolverFirewallRuleGroupAssociation),
	// "AWS::Route53Resolver::ResolverDNSSECConfig":                  ParallelDescribeRegional(describer.Route53ResolverResolverDNSSECConfig),
	// "AWS::Route53Resolver::ResolverEndpoint":                      ParallelDescribeRegional(describer.Route53ResolverResolverEndpoint),
	// "AWS::Route53Resolver::ResolverQueryLoggingConfig":            ParallelDescribeRegional(describer.Route53ResolverResolverQueryLoggingConfig),
	// "AWS::Route53Resolver::ResolverQueryLoggingConfigAssociation": ParallelDescribeRegional(describer.Route53ResolverResolverQueryLoggingConfigAssociation),
	// "AWS::Route53Resolver::ResolverRule":                          ParallelDescribeRegional(describer.Route53ResolverResolverRule),
	// "AWS::Route53Resolver::ResolverRuleAssociation":               ParallelDescribeRegional(describer.Route53ResolverResolverRuleAssociation),
	"AWS::S3::AccessPoint": ParallelDescribeRegional(describer.S3AccessPoint),
	//"AWS::S3::AccountSetting":               SequentialDescribeGlobal(describer.S3AccountSetting),
	"AWS::S3::Bucket": SequentialDescribeS3(describer.S3Bucket),
	//"AWS::S3::StorageLens":                  ParallelDescribeRegional(describer.S3StorageLens),
	"AWS::SageMaker::EndpointConfiguration": ParallelDescribeRegional(describer.SageMakerEndpointConfiguration),
	"AWS::SageMaker::NotebookInstance":      ParallelDescribeRegional(describer.SageMakerNotebookInstance),
	"AWS::SecretsManager::Secret":           ParallelDescribeRegional(describer.SecretsManagerSecret),
	"AWS::SecurityHub::Hub":                 ParallelDescribeRegional(describer.SecurityHubHub),
	"AWS::SES::ConfigurationSet":            ParallelDescribeRegional(describer.SESConfigurationSet),
	"AWS::SES::Identity":                    ParallelDescribeRegional(describer.SESIdentity),
	// "AWS::SES::ContactList":                                       ParallelDescribeRegional(describer.SESContactList),
	// "AWS::SES::ReceiptFilter":                                     ParallelDescribeRegional(describer.SESReceiptFilter),
	// "AWS::SES::ReceiptRuleSet":                                    ParallelDescribeRegional(describer.SESReceiptRuleSet),
	// "AWS::SES::Template":                                          ParallelDescribeRegional(describer.SESTemplate),
	"AWS::Shield::ProtectionGroup": ParallelDescribeRegional(describer.ShieldProtectionGroup),
	//"AWS::SNS::Subscription": ParallelDescribeRegional(describer.SNSSubscription),
	"AWS::SNS::Topic": ParallelDescribeRegional(describer.SNSTopic),
	"AWS::SQS::Queue": ParallelDescribeRegional(describer.SQSQueue),
	// "AWS::SSM::Association":                                       ParallelDescribeRegional(describer.SSMAssociation),
	// "AWS::SSM::Document":                                          ParallelDescribeRegional(describer.SSMDocument),
	// "AWS::SSM::MaintenanceWindow":                                 ParallelDescribeRegional(describer.SSMMaintenanceWindow),
	// "AWS::SSM::MaintenanceWindowTarget":                           ParallelDescribeRegional(describer.SSMMaintenanceWindowTarget),
	// "AWS::SSM::MaintenanceWindowTask":                             ParallelDescribeRegional(describer.SSMMaintenanceWindowTask),
	"AWS::SSM::ManagedInstance":           ParallelDescribeRegional(describer.SSMManagedInstance),
	"AWS::SSM::ManagedInstanceCompliance": ParallelDescribeRegional(describer.SSMManagedInstanceCompliance),
	// "AWS::SSM::Parameter":                                         ParallelDescribeRegional(describer.SSMParameter),
	// "AWS::SSM::PatchBaseline":                                     ParallelDescribeRegional(describer.SSMPatchBaseline),
	// "AWS::SSM::ResourceDataSync":                                  ParallelDescribeRegional(describer.SSMResourceDataSync),
	"AWS::SSOAdmin::Instance":             ParallelDescribeRegional(describer.SSOAdminInstance),
	"AWS::StorageGateway::StorageGateway": ParallelDescribeRegional(describer.StorageGatewayStorageGateway),
	// "AWS::Synthetics::Canary":                                     ParallelDescribeRegional(describer.SyntheticsCanary),
	// "AWS::WAFRegional::ByteMatchSet":                              ParallelDescribeRegional(describer.WAFRegionalByteMatchSet),
	// "AWS::WAFRegional::GeoMatchSet":                               ParallelDescribeRegional(describer.WAFRegionalGeoMatchSet),
	// "AWS::WAFRegional::IPSet":                                     ParallelDescribeRegional(describer.WAFRegionalIPSet),
	// "AWS::WAFRegional::RateBasedRule":                             ParallelDescribeRegional(describer.WAFRegionalRateBasedRule),
	// "AWS::WAFRegional::RegexPatternSet":                           ParallelDescribeRegional(describer.WAFRegionalRegexPatternSet),
	"AWS::WAFRegional::Rule": ParallelDescribeRegional(describer.WAFRegionalRule),
	// "AWS::WAFRegional::SizeConstraintSet":                         ParallelDescribeRegional(describer.WAFRegionalSizeConstraintSet),
	// "AWS::WAFRegional::SqlInjectionMatchSet":                      ParallelDescribeRegional(describer.WAFRegionalSqlInjectionMatchSet),
	// "AWS::WAFRegional::WebACL":                                    ParallelDescribeRegional(describer.WAFRegionalWebACL),
	// "AWS::WAFRegional::WebACLAssociation":                         ParallelDescribeRegional(describer.WAFRegionalWebACLAssociation),
	// "AWS::WAFRegional::XssMatchSet":                               ParallelDescribeRegional(describer.WAFRegionalXssMatchSet),
	"AWS::WAF::Rule": ParallelDescribeRegional(describer.WAFRule),
	// "AWS::WAFv2::IPSet":                                           ParallelDescribeRegional(describer.WAFv2IPSet),
	// "AWS::WAFv2::LoggingConfiguration":                            ParallelDescribeRegional(describer.WAFv2LoggingConfiguration),
	// "AWS::WAFv2::RegexPatternSet":                                 ParallelDescribeRegional(describer.WAFv2RegexPatternSet),
	// "AWS::WAFv2::RuleGroup":                                       ParallelDescribeRegional(describer.WAFv2RuleGroup),
	"AWS::WAFv2::WebACL": ParallelDescribeRegional(describer.WAFv2WebACL),
	// "AWS::WAFv2::WebACLAssociation":                               ParallelDescribeRegional(describer.WAFv2WebACLAssociation),
	// "AWS::WorkSpaces::ConnectionAlias":                            ParallelDescribeRegional(describer.WorkSpacesConnectionAlias),
	"AWS::Workspaces::Bundle":    ParallelDescribeRegional(describer.WorkspacesBundle),
	"AWS::Workspaces::Workspace": ParallelDescribeRegional(describer.WorkspacesWorkspace),
}

func ListResourceTypes() []string {
	var list []string
	for k := range resourceTypeToDescriber {
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

	resources, err := describe(ctx, cfg, accountId, regions, resourceType)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func describe(
	ctx context.Context,
	cfg aws.Config,
	account string,
	regions []string,
	resourceType string) (*Resources, error) {
	describe, ok := resourceTypeToDescriber[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	return describe(ctx, cfg, account, regions, resourceType)
}

// Parallel describe the resources across the reigons. Failure in one regions won't affect
// other regions.
func ParallelDescribeRegional(describe func(context.Context, aws.Config) ([]describer.Resource, error)) ResourceDescriber {
	type result struct {
		region    string
		resources []describer.Resource
		err       error
	}
	return func(ctx context.Context, cfg aws.Config, account string, regions []string, rType string) (*Resources, error) {
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
	return func(ctx context.Context, cfg aws.Config, account string, regions []string, rType string) (*Resources, error) {
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
	return func(ctx context.Context, cfg aws.Config, account string, regions []string, rType string) (*Resources, error) {
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
