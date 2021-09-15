package aws

import (
	"context"
	"errors"
	"fmt"

	"gitlab.com/anil94/golang-aws-inventory/pkg/aws/describer"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go"
)

type ResourceDescriber func(context.Context, aws.Config) ([]interface{}, error)

var ResourceTypeToDescriber = map[string]ResourceDescriber{
	"AWS::ApplicationInsights::Application":                       describer.ApplicationInsightsApplication,
	"AWS::AutoScaling::AutoScalingGroup":                          describer.AutoScalingAutoScalingGroup,
	"AWS::AutoScaling::LaunchConfiguration":                       describer.AutoScalingLaunchConfiguration,
	"AWS::AutoScaling::LifecycleHook":                             describer.AutoScalingLifecycleHook,
	"AWS::AutoScaling::ScalingPolicy":                             describer.AutoScalingScalingPolicy,
	"AWS::AutoScaling::ScheduledAction":                           describer.AutoScalingScheduledAction,
	"AWS::AutoScaling::WarmPool":                                  describer.AutoScalingWarmPool,
	"AWS::Backup::BackupPlan":                                     describer.BackupBackupPlan,
	"AWS::Backup::BackupSelection":                                describer.BackupBackupSelection,
	"AWS::Backup::BackupVault":                                    describer.BackupBackupVault,
	"AWS::CertificateManager::Account":                            describer.CertificateManagerAccount,
	"AWS::CertificateManager::Certificate":                        describer.CertificateManagerCertificate,
	"AWS::CloudTrail::Trail":                                      describer.CloudTrailTrail,
	"AWS::CloudWatch::Alarm":                                      describer.CloudWatchAlarm,
	"AWS::CloudWatch::AnomalyDetector":                            describer.CloudWatchAnomalyDetector,
	"AWS::CloudWatch::CompositeAlarm":                             describer.CloudWatchCompositeAlarm,
	"AWS::CloudWatch::Dashboard":                                  describer.CloudWatchDashboard,
	"AWS::CloudWatch::InsightRule":                                describer.CloudWatchInsightRule,
	"AWS::CloudWatch::MetricStream":                               describer.CloudWatchMetricStream,
	"AWS::EC2::CapacityReservation":                               describer.EC2CapacityReservation,
	"AWS::EC2::CarrierGateway":                                    describer.EC2CarrierGateway,
	"AWS::EC2::ClientVpnAuthorizationRule":                        describer.EC2ClientVpnAuthorizationRule,
	"AWS::EC2::ClientVpnEndpoint":                                 describer.EC2ClientVpnEndpoint,
	"AWS::EC2::ClientVpnRoute":                                    describer.EC2ClientVpnRoute,
	"AWS::EC2::ClientVpnTargetNetworkAssociation":                 describer.EC2ClientVpnTargetNetworkAssociation,
	"AWS::EC2::CustomerGateway":                                   describer.EC2CustomerGateway,
	"AWS::EC2::DHCPOptions":                                       describer.EC2DHCPOptions,
	// "AWS::EC2::EC2Fleet":                                          describer.EC2EC2Fleet,
	"AWS::EC2::EIP":                                               describer.EC2EIP,
	// "AWS::EC2::EIPAssociation":                                    describer.EC2EIPAssociation,
	"AWS::EC2::EgressOnlyInternetGateway":                         describer.EC2EgressOnlyInternetGateway,
	"AWS::EC2::EnclaveCertificateIamRoleAssociation":              describer.EC2EnclaveCertificateIamRoleAssociation,
	"AWS::EC2::FlowLog":                                           describer.EC2FlowLog,
	// "AWS::EC2::GatewayRouteTableAssociation":                      describer.EC2GatewayRouteTableAssociation,
	"AWS::EC2::Host":                                              describer.EC2Host,
	"AWS::EC2::Instance":                                          describer.EC2Instance,
	"AWS::EC2::InternetGateway":                                   describer.EC2InternetGateway,
	"AWS::EC2::LaunchTemplate":                                    describer.EC2LaunchTemplate,
	// "AWS::EC2::LocalGatewayRoute":                                 describer.EC2LocalGatewayRoute,
	"AWS::EC2::LocalGatewayRouteTableVPCAssociation":              describer.EC2LocalGatewayRouteTableVPCAssociation,
	"AWS::EC2::NatGateway":                                        describer.EC2NatGateway,
	"AWS::EC2::NetworkAcl":                                        describer.EC2NetworkAcl,
	// "AWS::EC2::NetworkAclEntry":                                   describer.EC2NetworkAclEntry,
	"AWS::EC2::NetworkInsightsAnalysis":                           describer.EC2NetworkInsightsAnalysis,
	"AWS::EC2::NetworkInsightsPath":                               describer.EC2NetworkInsightsPath,
	"AWS::EC2::NetworkInterface":                                  describer.EC2NetworkInterface,
	"AWS::EC2::NetworkInterfaceAttachment":                        describer.EC2NetworkInterfaceAttachment,
	"AWS::EC2::NetworkInterfacePermission":                        describer.EC2NetworkInterfacePermission,
	"AWS::EC2::PlacementGroup":                                    describer.EC2PlacementGroup,
	"AWS::EC2::PrefixList":                                        describer.EC2PrefixList,
	// "AWS::EC2::Route":                                             describer.EC2Route,
	"AWS::EC2::RouteTable":                                        describer.EC2RouteTable,
	"AWS::EC2::SecurityGroup":                                     describer.EC2SecurityGroup,
	// "AWS::EC2::SecurityGroupEgress":                               describer.EC2SecurityGroupEgress,
	// "AWS::EC2::SecurityGroupIngress":                              describer.EC2SecurityGroupIngress,
	"AWS::EC2::SpotFleet":                                         describer.EC2SpotFleet,
	"AWS::EC2::Subnet":                                            describer.EC2Subnet,
	"AWS::EC2::SubnetCidrBlock":                                   describer.EC2SubnetCidrBlock,
	// "AWS::EC2::SubnetNetworkAclAssociation":                       describer.EC2SubnetNetworkAclAssociation,
	// "AWS::EC2::SubnetRouteTableAssociation":                       describer.EC2SubnetRouteTableAssociation,
	"AWS::EC2::TrafficMirrorFilter":                               describer.EC2TrafficMirrorFilter,
	// "AWS::EC2::TrafficMirrorFilterRule":                           describer.EC2TrafficMirrorFilterRule,
	"AWS::EC2::TrafficMirrorSession":                              describer.EC2TrafficMirrorSession,
	"AWS::EC2::TrafficMirrorTarget":                               describer.EC2TrafficMirrorTarget,
	"AWS::EC2::TransitGateway":                                    describer.EC2TransitGateway,
	"AWS::EC2::TransitGatewayAttachment":                          describer.EC2TransitGatewayAttachment,
	"AWS::EC2::TransitGatewayConnect":                             describer.EC2TransitGatewayConnect,
	"AWS::EC2::TransitGatewayMulticastDomain":                     describer.EC2TransitGatewayMulticastDomain,
	"AWS::EC2::TransitGatewayMulticastDomainAssociation":          describer.EC2TransitGatewayMulticastDomainAssociation,
	"AWS::EC2::TransitGatewayMulticastGroupMember":                describer.EC2TransitGatewayMulticastGroupMember,
	"AWS::EC2::TransitGatewayMulticastGroupSource":                describer.EC2TransitGatewayMulticastGroupSource,
	"AWS::EC2::TransitGatewayPeeringAttachment":                   describer.EC2TransitGatewayPeeringAttachment,
	// "AWS::EC2::TransitGatewayRoute":                               describer.EC2TransitGatewayRoute,
	"AWS::EC2::TransitGatewayRouteTable":                          describer.EC2TransitGatewayRouteTable,
	"AWS::EC2::TransitGatewayRouteTableAssociation":               describer.EC2TransitGatewayRouteTableAssociation,
	"AWS::EC2::TransitGatewayRouteTablePropagation":               describer.EC2TransitGatewayRouteTablePropagation,
	"AWS::EC2::VPC":                                               describer.EC2VPC,
	// "AWS::EC2::VPCCidrBlock":                                      describer.EC2VPCCidrBlock,
	// "AWS::EC2::VPCDHCPOptionsAssociation":                         describer.EC2VPCDHCPOptionsAssociation,
	"AWS::EC2::VPCEndpoint":                                       describer.EC2VPCEndpoint,
	"AWS::EC2::VPCEndpointConnectionNotification":                 describer.EC2VPCEndpointConnectionNotification,
	"AWS::EC2::VPCEndpointService":                                describer.EC2VPCEndpointService,
	"AWS::EC2::VPCEndpointServicePermissions":                     describer.EC2VPCEndpointServicePermissions,
	// "AWS::EC2::VPCGatewayAttachment":                              describer.EC2VPCGatewayAttachment,
	"AWS::EC2::VPCPeeringConnection":                              describer.EC2VPCPeeringConnection,
	"AWS::EC2::VPNConnection":                                     describer.EC2VPNConnection,
	// "AWS::EC2::VPNConnectionRoute":                                describer.EC2VPNConnectionRoute,
	"AWS::EC2::VPNGateway":                                        describer.EC2VPNGateway,
	// "AWS::EC2::VPNGatewayRoutePropagation":                        describer.EC2VPNGatewayRoutePropagation,
	"AWS::EC2::Volume":                                            describer.EC2Volume,
	// "AWS::EC2::VolumeAttachment":                                  describer.EC2VolumeAttachment,
	"AWS::ECR::PublicRepository":                                  describer.ECRPublicRepository,
	"AWS::ECR::RegistryPolicy":                                    describer.ECRRegistryPolicy,
	// "AWS::ECR::ReplicationConfiguration":                          describer.ECRReplicationConfiguration,
	"AWS::ECR::Repository":                                        describer.ECRRepository,
	"AWS::ECS::CapacityProvider":                                  describer.ECSCapacityProvider,
	"AWS::ECS::Cluster":                                           describer.ECSCluster,
	// "AWS::ECS::ClusterCapacityProviderAssociations":               describer.ECSClusterCapacityProviderAssociations,
	// "AWS::ECS::PrimaryTaskSet":                                    describer.ECSPrimaryTaskSet,
	"AWS::ECS::Service":                                           describer.ECSService,
	"AWS::ECS::TaskDefinition":                                    describer.ECSTaskDefinition,
	// "AWS::ECS::TaskSet":                                           describer.ECSTaskSet,
	"AWS::EFS::AccessPoint":                                       describer.EFSAccessPoint,
	"AWS::EFS::FileSystem":                                        describer.EFSFileSystem,
	"AWS::EFS::MountTarget":                                       describer.EFSMountTarget,
	"AWS::EKS::Addon":                                             describer.EKSAddon,
	"AWS::EKS::Cluster":                                           describer.EKSCluster,
	"AWS::EKS::FargateProfile":                                    describer.EKSFargateProfile,
	"AWS::EKS::Nodegroup":                                         describer.EKSNodegroup,
	"AWS::ElasticLoadBalancing::LoadBalancer":                     describer.ElasticLoadBalancingLoadBalancer,
	"AWS::ElasticLoadBalancingV2::Listener":                       describer.ElasticLoadBalancingV2Listener,
	// "AWS::ElasticLoadBalancingV2::ListenerCertificate":            describer.ElasticLoadBalancingV2ListenerCertificate,
	"AWS::ElasticLoadBalancingV2::ListenerRule":                   describer.ElasticLoadBalancingV2ListenerRule,
	"AWS::ElasticLoadBalancingV2::LoadBalancer":                   describer.ElasticLoadBalancingV2LoadBalancer,
	"AWS::ElasticLoadBalancingV2::TargetGroup":                    describer.ElasticLoadBalancingV2TargetGroup,
	"AWS::IAM::AccessKey":                                         describer.IAMAccessKey,
	"AWS::IAM::Group":                                             describer.IAMGroup,
	"AWS::IAM::InstanceProfile":                                   describer.IAMInstanceProfile,
	"AWS::IAM::ManagedPolicy":                                     describer.IAMManagedPolicy,
	"AWS::IAM::OIDCProvider":                                      describer.IAMOIDCProvider,
	"AWS::IAM::Policy":                                            describer.IAMPolicy,
	"AWS::IAM::Role":                                              describer.IAMRole,
	"AWS::IAM::SAMLProvider":                                      describer.IAMSAMLProvider,
	"AWS::IAM::ServerCertificate":                                 describer.IAMServerCertificate,
	// "AWS::IAM::ServiceLinkedRole":                                 describer.IAMServiceLinkedRole,
	"AWS::IAM::User":                                              describer.IAMUser,
	// "AWS::IAM::UserToGroupAddition":                               describer.IAMUserToGroupAddition,
	"AWS::IAM::VirtualMFADevice":                                  describer.IAMVirtualMFADevice,
	"AWS::KMS::Alias":                                             describer.KMSAlias,
	"AWS::KMS::Key":                                               describer.KMSKey,
	// "AWS::KMS::ReplicaKey":                                        describer.KMSReplicaKey,
	"AWS::Lambda::Alias":                                          describer.LambdaAlias,
	"AWS::Lambda::CodeSigningConfig":                              describer.LambdaCodeSigningConfig,
	"AWS::Lambda::EventInvokeConfig":                              describer.LambdaEventInvokeConfig,
	"AWS::Lambda::EventSourceMapping":                             describer.LambdaEventSourceMapping,
	"AWS::Lambda::Function":                                       describer.LambdaFunction,
	"AWS::Lambda::LayerVersion":                                   describer.LambdaLayerVersion,
	"AWS::Lambda::LayerVersionPermission":                         describer.LambdaLayerVersionPermission,
	"AWS::Lambda::Permission":                                     describer.LambdaPermission,
	// "AWS::Lambda::Version":                                        describer.LambdaVersion,
	"AWS::Logs::Destination":                                      describer.CloudWatchLogsDestination,
	"AWS::Logs::LogGroup":                                         describer.CloudWatchLogsLogGroup,
	"AWS::Logs::LogStream":                                        describer.CloudWatchLogsLogStream,
	"AWS::Logs::MetricFilter":                                     describer.CloudWatchLogsMetricFilter,
	"AWS::Logs::QueryDefinition":                                  describer.CloudWatchLogsQueryDefinition,
	"AWS::Logs::ResourcePolicy":                                   describer.CloudWatchLogsResourcePolicy,
	"AWS::Logs::SubscriptionFilter":                               describer.CloudWatchLogsSubscriptionFilter,
	"AWS::RDS::DBCluster":                                         describer.RDSDBCluster,
	"AWS::RDS::DBClusterParameterGroup":                           describer.RDSDBClusterParameterGroup,
	"AWS::RDS::DBInstance":                                        describer.RDSDBInstance,
	"AWS::RDS::DBParameterGroup":                                  describer.RDSDBParameterGroup,
	"AWS::RDS::DBProxy":                                           describer.RDSDBProxy,
	"AWS::RDS::DBProxyEndpoint":                                   describer.RDSDBProxyEndpoint,
	"AWS::RDS::DBProxyTargetGroup":                                describer.RDSDBProxyTargetGroup,
	"AWS::RDS::DBSecurityGroup":                                   describer.RDSDBSecurityGroup,
	// "AWS::RDS::DBSecurityGroupIngress":                            describer.RDSDBSecurityGroupIngress,
	"AWS::RDS::DBSubnetGroup":                                     describer.RDSDBSubnetGroup,
	"AWS::RDS::EventSubscription":                                 describer.RDSEventSubscription,
	"AWS::RDS::GlobalCluster":                                     describer.RDSGlobalCluster,
	"AWS::RDS::OptionGroup":                                       describer.RDSOptionGroup,
	"AWS::Redshift::Cluster":                                      describer.RedshiftCluster,
	"AWS::Redshift::ClusterParameterGroup":                        describer.RedshiftClusterParameterGroup,
	"AWS::Redshift::ClusterSecurityGroup":                         describer.RedshiftClusterSecurityGroup,
	// "AWS::Redshift::ClusterSecurityGroupIngress":                  describer.RedshiftClusterSecurityGroupIngress,
	"AWS::Redshift::ClusterSubnetGroup":                           describer.RedshiftClusterSubnetGroup,
	"AWS::Route53::DNSSEC":                                        describer.Route53DNSSEC,
	"AWS::Route53::HealthCheck":                                   describer.Route53HealthCheck,
	"AWS::Route53::HostedZone":                                    describer.Route53HostedZone,
	// "AWS::Route53::KeySigningKey":                                 describer.Route53KeySigningKey,
	"AWS::Route53::RecordSet":                                     describer.Route53RecordSet,
	// "AWS::Route53::RecordSetGroup":                                describer.Route53RecordSetGroup,
	"AWS::Route53Resolver::FirewallDomainList":                    describer.Route53ResolverFirewallDomainList,
	"AWS::Route53Resolver::FirewallRuleGroup":                     describer.Route53ResolverFirewallRuleGroup,
	"AWS::Route53Resolver::FirewallRuleGroupAssociation":          describer.Route53ResolverFirewallRuleGroupAssociation,
	"AWS::Route53Resolver::ResolverDNSSECConfig":                  describer.Route53ResolverResolverDNSSECConfig,
	"AWS::Route53Resolver::ResolverEndpoint":                      describer.Route53ResolverResolverEndpoint,
	"AWS::Route53Resolver::ResolverQueryLoggingConfig":            describer.Route53ResolverResolverQueryLoggingConfig,
	"AWS::Route53Resolver::ResolverQueryLoggingConfigAssociation": describer.Route53ResolverResolverQueryLoggingConfigAssociation,
	"AWS::Route53Resolver::ResolverRule":                          describer.Route53ResolverResolverRule,
	"AWS::Route53Resolver::ResolverRuleAssociation":               describer.Route53ResolverResolverRuleAssociation,
	"AWS::S3::AccessPoint":                                        describer.S3AccessPoint,
	"AWS::S3::Bucket":                                             describer.S3Bucket,
	"AWS::S3::BucketPolicy":                                       describer.S3BucketPolicy,
	"AWS::S3::StorageLens":                                        describer.S3StorageLens,
	"AWS::SES::ConfigurationSet":                                  describer.SESConfigurationSet,
	// "AWS::SES::ConfigurationSetEventDestination":                  describer.SESConfigurationSetEventDestination,
	"AWS::SES::ContactList":                                       describer.SESContactList,
	"AWS::SES::ReceiptFilter":                                     describer.SESReceiptFilter,
	// "AWS::SES::ReceiptRule":                                       describer.SESReceiptRule,
	"AWS::SES::ReceiptRuleSet":                                    describer.SESReceiptRuleSet,
	"AWS::SES::Template":                                          describer.SESTemplate,
	"AWS::SNS::Subscription":                                      describer.SNSSubscription,
	"AWS::SNS::Topic":                                             describer.SNSTopic,
	// "AWS::SNS::TopicPolicy":                                       describer.SNSTopicPolicy,
	"AWS::SQS::Queue":                                             describer.SQSQueue,
	// "AWS::SQS::QueuePolicy":                                       describer.SQSQueuePolicy,
	"AWS::SSM::Association":                                       describer.SSMAssociation,
	"AWS::SSM::Document":                                          describer.SSMDocument,
	"AWS::SSM::MaintenanceWindow":                                 describer.SSMMaintenanceWindow,
	"AWS::SSM::MaintenanceWindowTarget":                           describer.SSMMaintenanceWindowTarget,
	"AWS::SSM::MaintenanceWindowTask":                             describer.SSMMaintenanceWindowTask,
	"AWS::SSM::Parameter":                                         describer.SSMParameter,
	"AWS::SSM::PatchBaseline":                                     describer.SSMPatchBaseline,
	"AWS::SSM::ResourceDataSync":                                  describer.SSMResourceDataSync,
	"AWS::Synthetics::Canary":                                     describer.SyntheticsCanary,
	// "AWS::WAFRegional::ByteMatchSet":                              describer.WAFRegionalByteMatchSet,
	// "AWS::WAFRegional::GeoMatchSet":                               describer.WAFRegionalGeoMatchSet,
	// "AWS::WAFRegional::IPSet":                                     describer.WAFRegionalIPSet,
	// "AWS::WAFRegional::RateBasedRule":                             describer.WAFRegionalRateBasedRule,
	// "AWS::WAFRegional::RegexPatternSet":                           describer.WAFRegionalRegexPatternSet,
	// "AWS::WAFRegional::Rule":                                      describer.WAFRegionalRule,
	// "AWS::WAFRegional::SizeConstraintSet":                         describer.WAFRegionalSizeConstraintSet,
	// "AWS::WAFRegional::SqlInjectionMatchSet":                      describer.WAFRegionalSqlInjectionMatchSet,
	// "AWS::WAFRegional::WebACL":                                    describer.WAFRegionalWebACL,
	// "AWS::WAFRegional::WebACLAssociation":                         describer.WAFRegionalWebACLAssociation,
	// "AWS::WAFRegional::XssMatchSet":                               describer.WAFRegionalXssMatchSet,
	// "AWS::WAFv2::IPSet":                                           describer.WAFv2IPSet,
	// "AWS::WAFv2::LoggingConfiguration":                            describer.WAFv2LoggingConfiguration,
	// "AWS::WAFv2::RegexPatternSet":                                 describer.WAFv2RegexPatternSet,
	// "AWS::WAFv2::RuleGroup":                                       describer.WAFv2RuleGroup,
	// "AWS::WAFv2::WebACL":                                          describer.WAFv2WebACL,
	// "AWS::WAFv2::WebACLAssociation":                               describer.WAFv2WebACLAssociation,
	"AWS::WorkSpaces::ConnectionAlias":                            describer.WorkSpacesConnectionAlias,
	"AWS::WorkSpaces::Workspace":                                  describer.WorkSpacesWorkspace,
}

type RegionalResponse struct {
	Resources map[string][]interface{}
	Errors    map[string]string
}

func GetResources(
	ctx context.Context,
	cfg aws.Config,
	regions []string,
	resourceType string) (*RegionalResponse, error) {

	type result struct {
		region    string
		resources []interface{}
		err       error
	}

	describe, ok := ResourceTypeToDescriber[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	input := make(chan result, len(regions))
	for _, region := range regions {
		go func(r string) {
			// Make a shallow copy and override the default region
			rCfg := cfg.Copy()
			rCfg.Region = r

			resources, err := describe(ctx, rCfg)
			input <- result{region: r, resources: resources, err: err}
		}(region)
	}

	response := RegionalResponse{
		Resources: make(map[string][]interface{}, len(regions)),
		Errors:    make(map[string]string, len(regions)),
	}
	for range regions {
		resp := <-input
		if resp.err != nil {
			// If an action is not supported in a region, we will get InvalidAction error code. In that case,
			// just send empty list as the response. Since we are using the AWS SDK, if we hit an InvalidAction
			// we can be certain that the API operation is not supported in that particular region.
			var ae smithy.APIError
			if errors.As(resp.err, &ae) && ae.ErrorCode() == "InvalidAction" {
				resp.resources, resp.err = []interface{}{}, nil
			} else {
				response.Errors[resp.region] = resp.err.Error()
				continue
			}
		}

		if resp.resources == nil {
			resp.resources = []interface{}{}
		}
		response.Resources[resp.region] = resp.resources
	}

	return &response, nil
}
