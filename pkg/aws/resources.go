package aws

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws/describer"
)

type ResourceDescriber func(context.Context, aws.Config, string, []string, string) (*ResourceDescribeOutput, error)

var resourceTypeToDescriber = map[string]ResourceDescriber{
	"AWS::ApplicationInsights::Application":                       ParallelDescribeRegional(describer.ApplicationInsightsApplication),
	"AWS::AutoScaling::AutoScalingGroup":                          ParallelDescribeRegional(describer.AutoScalingAutoScalingGroup),
	"AWS::AutoScaling::LaunchConfiguration":                       ParallelDescribeRegional(describer.AutoScalingLaunchConfiguration),
	"AWS::AutoScaling::LifecycleHook":                             ParallelDescribeRegional(describer.AutoScalingLifecycleHook),
	"AWS::AutoScaling::ScalingPolicy":                             ParallelDescribeRegional(describer.AutoScalingScalingPolicy),
	"AWS::AutoScaling::ScheduledAction":                           ParallelDescribeRegional(describer.AutoScalingScheduledAction),
	"AWS::AutoScaling::WarmPool":                                  ParallelDescribeRegional(describer.AutoScalingWarmPool),
	"AWS::Backup::BackupPlan":                                     ParallelDescribeRegional(describer.BackupBackupPlan),
	"AWS::Backup::BackupSelection":                                ParallelDescribeRegional(describer.BackupBackupSelection),
	"AWS::Backup::BackupVault":                                    ParallelDescribeRegional(describer.BackupBackupVault),
	"AWS::CertificateManager::Account":                            SequentialDescribeGlobal(describer.CertificateManagerAccount),
	"AWS::CertificateManager::Certificate":                        ParallelDescribeRegional(describer.CertificateManagerCertificate),
	"AWS::CloudTrail::Trail":                                      ParallelDescribeRegional(describer.CloudTrailTrail),
	"AWS::CloudWatch::Alarm":                                      ParallelDescribeRegional(describer.CloudWatchAlarm),
	"AWS::CloudWatch::AnomalyDetector":                            ParallelDescribeRegional(describer.CloudWatchAnomalyDetector),
	"AWS::CloudWatch::CompositeAlarm":                             ParallelDescribeRegional(describer.CloudWatchCompositeAlarm),
	"AWS::CloudWatch::Dashboard":                                  ParallelDescribeRegional(describer.CloudWatchDashboard),
	"AWS::CloudWatch::InsightRule":                                ParallelDescribeRegional(describer.CloudWatchInsightRule),
	"AWS::CloudWatch::MetricStream":                               ParallelDescribeRegional(describer.CloudWatchMetricStream),
	"AWS::EC2::CapacityReservation":                               ParallelDescribeRegional(describer.EC2CapacityReservation),
	"AWS::EC2::CarrierGateway":                                    ParallelDescribeRegional(describer.EC2CarrierGateway),
	"AWS::EC2::ClientVpnAuthorizationRule":                        ParallelDescribeRegional(describer.EC2ClientVpnAuthorizationRule),
	"AWS::EC2::ClientVpnEndpoint":                                 ParallelDescribeRegional(describer.EC2ClientVpnEndpoint),
	"AWS::EC2::ClientVpnRoute":                                    ParallelDescribeRegional(describer.EC2ClientVpnRoute),
	"AWS::EC2::ClientVpnTargetNetworkAssociation":                 ParallelDescribeRegional(describer.EC2ClientVpnTargetNetworkAssociation),
	"AWS::EC2::CustomerGateway":                                   ParallelDescribeRegional(describer.EC2CustomerGateway),
	"AWS::EC2::DHCPOptions":                                       ParallelDescribeRegional(describer.EC2DHCPOptions),
	"AWS::EC2::EC2Fleet":                                          ParallelDescribeRegional(describer.EC2Fleet),
	"AWS::EC2::EIP":                                               ParallelDescribeRegional(describer.EC2EIP),
	"AWS::EC2::EgressOnlyInternetGateway":                         ParallelDescribeRegional(describer.EC2EgressOnlyInternetGateway),
	"AWS::EC2::EnclaveCertificateIamRoleAssociation":              ParallelDescribeRegional(describer.EC2EnclaveCertificateIamRoleAssociation),
	"AWS::EC2::FlowLog":                                           ParallelDescribeRegional(describer.EC2FlowLog),
	"AWS::EC2::Host":                                              ParallelDescribeRegional(describer.EC2Host),
	"AWS::EC2::Instance":                                          ParallelDescribeRegional(describer.EC2Instance),
	"AWS::EC2::InternetGateway":                                   ParallelDescribeRegional(describer.EC2InternetGateway),
	"AWS::EC2::LaunchTemplate":                                    ParallelDescribeRegional(describer.EC2LaunchTemplate),
	"AWS::EC2::LocalGatewayRouteTable":                            ParallelDescribeRegional(describer.EC2LocalGatewayRouteTable),
	"AWS::EC2::LocalGatewayRouteTableVPCAssociation":              ParallelDescribeRegional(describer.EC2LocalGatewayRouteTableVPCAssociation),
	"AWS::EC2::NatGateway":                                        ParallelDescribeRegional(describer.EC2NatGateway),
	"AWS::EC2::NetworkAcl":                                        ParallelDescribeRegional(describer.EC2NetworkAcl),
	"AWS::EC2::NetworkInsightsAnalysis":                           ParallelDescribeRegional(describer.EC2NetworkInsightsAnalysis),
	"AWS::EC2::NetworkInsightsPath":                               ParallelDescribeRegional(describer.EC2NetworkInsightsPath),
	"AWS::EC2::NetworkInterface":                                  ParallelDescribeRegional(describer.EC2NetworkInterface),
	"AWS::EC2::NetworkInterfacePermission":                        ParallelDescribeRegional(describer.EC2NetworkInterfacePermission),
	"AWS::EC2::PlacementGroup":                                    ParallelDescribeRegional(describer.EC2PlacementGroup),
	"AWS::EC2::PrefixList":                                        ParallelDescribeRegional(describer.EC2PrefixList),
	"AWS::EC2::RouteTable":                                        ParallelDescribeRegional(describer.EC2RouteTable),
	"AWS::EC2::SecurityGroup":                                     ParallelDescribeRegional(describer.EC2SecurityGroup),
	"AWS::EC2::SpotFleet":                                         ParallelDescribeRegional(describer.EC2SpotFleet),
	"AWS::EC2::Subnet":                                            ParallelDescribeRegional(describer.EC2Subnet),
	"AWS::EC2::TrafficMirrorFilter":                               ParallelDescribeRegional(describer.EC2TrafficMirrorFilter),
	"AWS::EC2::TrafficMirrorSession":                              ParallelDescribeRegional(describer.EC2TrafficMirrorSession),
	"AWS::EC2::TrafficMirrorTarget":                               ParallelDescribeRegional(describer.EC2TrafficMirrorTarget),
	"AWS::EC2::TransitGateway":                                    ParallelDescribeRegional(describer.EC2TransitGateway),
	"AWS::EC2::TransitGatewayAttachment":                          ParallelDescribeRegional(describer.EC2TransitGatewayAttachment),
	"AWS::EC2::TransitGatewayConnect":                             ParallelDescribeRegional(describer.EC2TransitGatewayConnect),
	"AWS::EC2::TransitGatewayMulticastDomain":                     ParallelDescribeRegional(describer.EC2TransitGatewayMulticastDomain),
	"AWS::EC2::TransitGatewayMulticastDomainAssociation":          ParallelDescribeRegional(describer.EC2TransitGatewayMulticastDomainAssociation),
	"AWS::EC2::TransitGatewayMulticastGroupMember":                ParallelDescribeRegional(describer.EC2TransitGatewayMulticastGroupMember),
	"AWS::EC2::TransitGatewayMulticastGroupSource":                ParallelDescribeRegional(describer.EC2TransitGatewayMulticastGroupSource),
	"AWS::EC2::TransitGatewayPeeringAttachment":                   ParallelDescribeRegional(describer.EC2TransitGatewayPeeringAttachment),
	"AWS::EC2::TransitGatewayRouteTable":                          ParallelDescribeRegional(describer.EC2TransitGatewayRouteTable),
	"AWS::EC2::TransitGatewayRouteTableAssociation":               ParallelDescribeRegional(describer.EC2TransitGatewayRouteTableAssociation),
	"AWS::EC2::TransitGatewayRouteTablePropagation":               ParallelDescribeRegional(describer.EC2TransitGatewayRouteTablePropagation),
	"AWS::EC2::VPC":                                               ParallelDescribeRegional(describer.EC2VPC),
	"AWS::EC2::VPCEndpoint":                                       ParallelDescribeRegional(describer.EC2VPCEndpoint),
	"AWS::EC2::VPCEndpointConnectionNotification":                 ParallelDescribeRegional(describer.EC2VPCEndpointConnectionNotification),
	"AWS::EC2::VPCEndpointService":                                ParallelDescribeRegional(describer.EC2VPCEndpointService),
	"AWS::EC2::VPCEndpointServicePermissions":                     ParallelDescribeRegional(describer.EC2VPCEndpointServicePermissions),
	"AWS::EC2::VPCPeeringConnection":                              ParallelDescribeRegional(describer.EC2VPCPeeringConnection),
	"AWS::EC2::VPNConnection":                                     ParallelDescribeRegional(describer.EC2VPNConnection),
	"AWS::EC2::VPNGateway":                                        ParallelDescribeRegional(describer.EC2VPNGateway),
	"AWS::EC2::Volume":                                            ParallelDescribeRegional(describer.EC2Volume),
	"AWS::ECR::PublicRepository":                                  ParallelDescribeRegional(describer.ECRPublicRepository),
	"AWS::ECR::Registry":                                          SequentialDescribeGlobal(describer.ECRRegistry),
	"AWS::ECR::RegistryPolicy":                                    SequentialDescribeGlobal(describer.ECRRegistryPolicy),
	"AWS::ECR::Repository":                                        ParallelDescribeRegional(describer.ECRRepository),
	"AWS::ECS::CapacityProvider":                                  ParallelDescribeRegional(describer.ECSCapacityProvider),
	"AWS::ECS::Cluster":                                           ParallelDescribeRegional(describer.ECSCluster),
	"AWS::ECS::Service":                                           ParallelDescribeRegional(describer.ECSService),
	"AWS::ECS::TaskDefinition":                                    ParallelDescribeRegional(describer.ECSTaskDefinition),
	"AWS::EFS::AccessPoint":                                       ParallelDescribeRegional(describer.EFSAccessPoint),
	"AWS::EFS::FileSystem":                                        ParallelDescribeRegional(describer.EFSFileSystem),
	"AWS::EFS::MountTarget":                                       ParallelDescribeRegional(describer.EFSMountTarget),
	"AWS::EKS::Addon":                                             ParallelDescribeRegional(describer.EKSAddon),
	"AWS::EKS::Cluster":                                           ParallelDescribeRegional(describer.EKSCluster),
	"AWS::EKS::FargateProfile":                                    ParallelDescribeRegional(describer.EKSFargateProfile),
	"AWS::EKS::Nodegroup":                                         ParallelDescribeRegional(describer.EKSNodegroup),
	"AWS::ElasticLoadBalancing::LoadBalancer":                     ParallelDescribeRegional(describer.ElasticLoadBalancingLoadBalancer),
	"AWS::ElasticLoadBalancingV2::Listener":                       ParallelDescribeRegional(describer.ElasticLoadBalancingV2Listener),
	"AWS::ElasticLoadBalancingV2::ListenerRule":                   ParallelDescribeRegional(describer.ElasticLoadBalancingV2ListenerRule),
	"AWS::ElasticLoadBalancingV2::LoadBalancer":                   ParallelDescribeRegional(describer.ElasticLoadBalancingV2LoadBalancer),
	"AWS::ElasticLoadBalancingV2::TargetGroup":                    ParallelDescribeRegional(describer.ElasticLoadBalancingV2TargetGroup),
	"AWS::IAM::AccessKey":                                         SequentialDescribeGlobal(describer.IAMAccessKey),
	"AWS::IAM::Group":                                             SequentialDescribeGlobal(describer.IAMGroup),
	"AWS::IAM::InstanceProfile":                                   SequentialDescribeGlobal(describer.IAMInstanceProfile),
	"AWS::IAM::ManagedPolicy":                                     SequentialDescribeGlobal(describer.IAMManagedPolicy),
	"AWS::IAM::OIDCProvider":                                      SequentialDescribeGlobal(describer.IAMOIDCProvider),
	"AWS::IAM::GroupPolicy":                                       SequentialDescribeGlobal(describer.IAMGroupPolicy),
	"AWS::IAM::UserPolicy":                                        SequentialDescribeGlobal(describer.IAMUserPolicy),
	"AWS::IAM::RolePolicy":                                        SequentialDescribeGlobal(describer.IAMRolePolicy),
	"AWS::IAM::Role":                                              SequentialDescribeGlobal(describer.IAMRole),
	"AWS::IAM::SAMLProvider":                                      SequentialDescribeGlobal(describer.IAMSAMLProvider),
	"AWS::IAM::ServerCertificate":                                 SequentialDescribeGlobal(describer.IAMServerCertificate),
	"AWS::IAM::User":                                              SequentialDescribeGlobal(describer.IAMUser),
	"AWS::IAM::VirtualMFADevice":                                  SequentialDescribeGlobal(describer.IAMVirtualMFADevice),
	"AWS::KMS::Alias":                                             ParallelDescribeRegional(describer.KMSAlias),
	"AWS::KMS::Key":                                               ParallelDescribeRegional(describer.KMSKey),
	"AWS::Lambda::Alias":                                          ParallelDescribeRegional(describer.LambdaAlias),
	"AWS::Lambda::CodeSigningConfig":                              ParallelDescribeRegional(describer.LambdaCodeSigningConfig),
	"AWS::Lambda::EventInvokeConfig":                              ParallelDescribeRegional(describer.LambdaEventInvokeConfig),
	"AWS::Lambda::EventSourceMapping":                             ParallelDescribeRegional(describer.LambdaEventSourceMapping),
	"AWS::Lambda::Function":                                       ParallelDescribeRegional(describer.LambdaFunction),
	"AWS::Lambda::LayerVersion":                                   ParallelDescribeRegional(describer.LambdaLayerVersion),
	"AWS::Lambda::LayerVersionPermission":                         ParallelDescribeRegional(describer.LambdaLayerVersionPermission),
	"AWS::Lambda::Permission":                                     ParallelDescribeRegional(describer.LambdaPermission),
	"AWS::Logs::Destination":                                      ParallelDescribeRegional(describer.CloudWatchLogsDestination),
	"AWS::Logs::LogGroup":                                         ParallelDescribeRegional(describer.CloudWatchLogsLogGroup),
	"AWS::Logs::LogStream":                                        ParallelDescribeRegional(describer.CloudWatchLogsLogStream),
	"AWS::Logs::MetricFilter":                                     ParallelDescribeRegional(describer.CloudWatchLogsMetricFilter),
	"AWS::Logs::QueryDefinition":                                  ParallelDescribeRegional(describer.CloudWatchLogsQueryDefinition),
	"AWS::Logs::ResourcePolicy":                                   ParallelDescribeRegional(describer.CloudWatchLogsResourcePolicy),
	"AWS::Logs::SubscriptionFilter":                               ParallelDescribeRegional(describer.CloudWatchLogsSubscriptionFilter),
	"AWS::RDS::DBCluster":                                         ParallelDescribeRegional(describer.RDSDBCluster),
	"AWS::RDS::DBClusterParameterGroup":                           ParallelDescribeRegional(describer.RDSDBClusterParameterGroup),
	"AWS::RDS::DBInstance":                                        ParallelDescribeRegional(describer.RDSDBInstance),
	"AWS::RDS::DBParameterGroup":                                  ParallelDescribeRegional(describer.RDSDBParameterGroup),
	"AWS::RDS::DBProxy":                                           ParallelDescribeRegional(describer.RDSDBProxy),
	"AWS::RDS::DBProxyEndpoint":                                   ParallelDescribeRegional(describer.RDSDBProxyEndpoint),
	"AWS::RDS::DBProxyTargetGroup":                                ParallelDescribeRegional(describer.RDSDBProxyTargetGroup),
	"AWS::RDS::DBSecurityGroup":                                   ParallelDescribeRegional(describer.RDSDBSecurityGroup),
	"AWS::RDS::DBSubnetGroup":                                     ParallelDescribeRegional(describer.RDSDBSubnetGroup),
	"AWS::RDS::EventSubscription":                                 ParallelDescribeRegional(describer.RDSEventSubscription),
	"AWS::RDS::GlobalCluster":                                     ParallelDescribeRegional(describer.RDSGlobalCluster),
	"AWS::RDS::OptionGroup":                                       ParallelDescribeRegional(describer.RDSOptionGroup),
	"AWS::Redshift::Cluster":                                      ParallelDescribeRegional(describer.RedshiftCluster),
	"AWS::Redshift::ClusterParameterGroup":                        ParallelDescribeRegional(describer.RedshiftClusterParameterGroup),
	"AWS::Redshift::ClusterSecurityGroup":                         ParallelDescribeRegional(describer.RedshiftClusterSecurityGroup),
	"AWS::Redshift::ClusterSubnetGroup":                           ParallelDescribeRegional(describer.RedshiftClusterSubnetGroup),
	"AWS::Route53::DNSSEC":                                        SequentialDescribeGlobal(describer.Route53DNSSEC),
	"AWS::Route53::HealthCheck":                                   SequentialDescribeGlobal(describer.Route53HealthCheck),
	"AWS::Route53::HostedZone":                                    SequentialDescribeGlobal(describer.Route53HostedZone),
	"AWS::Route53::RecordSet":                                     SequentialDescribeGlobal(describer.Route53RecordSet),
	"AWS::Route53Resolver::FirewallDomainList":                    ParallelDescribeRegional(describer.Route53ResolverFirewallDomainList),
	"AWS::Route53Resolver::FirewallRuleGroup":                     ParallelDescribeRegional(describer.Route53ResolverFirewallRuleGroup),
	"AWS::Route53Resolver::FirewallRuleGroupAssociation":          ParallelDescribeRegional(describer.Route53ResolverFirewallRuleGroupAssociation),
	"AWS::Route53Resolver::ResolverDNSSECConfig":                  ParallelDescribeRegional(describer.Route53ResolverResolverDNSSECConfig),
	"AWS::Route53Resolver::ResolverEndpoint":                      ParallelDescribeRegional(describer.Route53ResolverResolverEndpoint),
	"AWS::Route53Resolver::ResolverQueryLoggingConfig":            ParallelDescribeRegional(describer.Route53ResolverResolverQueryLoggingConfig),
	"AWS::Route53Resolver::ResolverQueryLoggingConfigAssociation": ParallelDescribeRegional(describer.Route53ResolverResolverQueryLoggingConfigAssociation),
	"AWS::Route53Resolver::ResolverRule":                          ParallelDescribeRegional(describer.Route53ResolverResolverRule),
	"AWS::Route53Resolver::ResolverRuleAssociation":               ParallelDescribeRegional(describer.Route53ResolverResolverRuleAssociation),
	"AWS::S3::AccessPoint":                                        ParallelDescribeRegional(describer.S3AccessPoint),
	"AWS::S3::Bucket":                                             SequentialDescribeS3(describer.S3Bucket),       // Global
	"AWS::S3::BucketPolicy":                                       SequentialDescribeS3(describer.S3BucketPolicy), // Semi-Global
	"AWS::S3::StorageLens":                                        ParallelDescribeRegional(describer.S3StorageLens),
	"AWS::SES::ConfigurationSet":                                  ParallelDescribeRegional(describer.SESConfigurationSet),
	"AWS::SES::ContactList":                                       ParallelDescribeRegional(describer.SESContactList),
	"AWS::SES::ReceiptFilter":                                     ParallelDescribeRegional(describer.SESReceiptFilter),
	"AWS::SES::ReceiptRuleSet":                                    ParallelDescribeRegional(describer.SESReceiptRuleSet),
	"AWS::SES::Template":                                          ParallelDescribeRegional(describer.SESTemplate),
	"AWS::SNS::Subscription":                                      ParallelDescribeRegional(describer.SNSSubscription),
	"AWS::SNS::Topic":                                             ParallelDescribeRegional(describer.SNSTopic),
	"AWS::SQS::Queue":                                             ParallelDescribeRegional(describer.SQSQueue),
	"AWS::SSM::Association":                                       ParallelDescribeRegional(describer.SSMAssociation),
	"AWS::SSM::Document":                                          ParallelDescribeRegional(describer.SSMDocument),
	"AWS::SSM::MaintenanceWindow":                                 ParallelDescribeRegional(describer.SSMMaintenanceWindow),
	"AWS::SSM::MaintenanceWindowTarget":                           ParallelDescribeRegional(describer.SSMMaintenanceWindowTarget),
	"AWS::SSM::MaintenanceWindowTask":                             ParallelDescribeRegional(describer.SSMMaintenanceWindowTask),
	"AWS::SSM::Parameter":                                         ParallelDescribeRegional(describer.SSMParameter),
	"AWS::SSM::PatchBaseline":                                     ParallelDescribeRegional(describer.SSMPatchBaseline),
	"AWS::SSM::ResourceDataSync":                                  ParallelDescribeRegional(describer.SSMResourceDataSync),
	"AWS::Synthetics::Canary":                                     ParallelDescribeRegional(describer.SyntheticsCanary),
	"AWS::WAFRegional::ByteMatchSet":                              ParallelDescribeRegional(describer.WAFRegionalByteMatchSet),
	"AWS::WAFRegional::GeoMatchSet":                               ParallelDescribeRegional(describer.WAFRegionalGeoMatchSet),
	"AWS::WAFRegional::IPSet":                                     ParallelDescribeRegional(describer.WAFRegionalIPSet),
	"AWS::WAFRegional::RateBasedRule":                             ParallelDescribeRegional(describer.WAFRegionalRateBasedRule),
	"AWS::WAFRegional::RegexPatternSet":                           ParallelDescribeRegional(describer.WAFRegionalRegexPatternSet),
	"AWS::WAFRegional::Rule":                                      ParallelDescribeRegional(describer.WAFRegionalRule),
	"AWS::WAFRegional::SizeConstraintSet":                         ParallelDescribeRegional(describer.WAFRegionalSizeConstraintSet),
	"AWS::WAFRegional::SqlInjectionMatchSet":                      ParallelDescribeRegional(describer.WAFRegionalSqlInjectionMatchSet),
	"AWS::WAFRegional::WebACL":                                    ParallelDescribeRegional(describer.WAFRegionalWebACL),
	"AWS::WAFRegional::WebACLAssociation":                         ParallelDescribeRegional(describer.WAFRegionalWebACLAssociation),
	"AWS::WAFRegional::XssMatchSet":                               ParallelDescribeRegional(describer.WAFRegionalXssMatchSet),
	"AWS::WAFv2::IPSet":                                           ParallelDescribeRegional(describer.WAFv2IPSet),
	"AWS::WAFv2::LoggingConfiguration":                            ParallelDescribeRegional(describer.WAFv2LoggingConfiguration),
	"AWS::WAFv2::RegexPatternSet":                                 ParallelDescribeRegional(describer.WAFv2RegexPatternSet),
	"AWS::WAFv2::RuleGroup":                                       ParallelDescribeRegional(describer.WAFv2RuleGroup),
	"AWS::WAFv2::WebACL":                                          ParallelDescribeRegional(describer.WAFv2WebACL),
	"AWS::WAFv2::WebACLAssociation":                               ParallelDescribeRegional(describer.WAFv2WebACLAssociation),
	"AWS::WorkSpaces::ConnectionAlias":                            ParallelDescribeRegional(describer.WorkSpacesConnectionAlias),
	"AWS::WorkSpaces::Workspace":                                  ParallelDescribeRegional(describer.WorkSpacesWorkspace),
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
	ResourceDescribeOutput
	Metadata ResourceDescribeMetadata
}

type ResourceDescribeOutput struct {
	Resources map[string][]describer.Resource
	Errors    map[string]string
}

type ResourceDescribeMetadata struct {
	ResourceType string
	AccountId    string
	Regions      []string
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
	cfg, err := getConfig(ctx, accessKey, secretKey, sessionToken, assumeRoleArn)
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

	response, err := describe(ctx, cfg, accountId, regions, resourceType)
	if err != nil {
		return nil, err
	}

	output := &Resources{
		ResourceDescribeOutput: *response,
		Metadata: ResourceDescribeMetadata{
			ResourceType: resourceType,
			AccountId:    accountId,
			Regions:      regions,
		},
	}

	return output, nil
}

func describe(
	ctx context.Context,
	cfg aws.Config,
	account string,
	regions []string,
	resourceType string) (*ResourceDescribeOutput, error) {
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
	return func(ctx context.Context, cfg aws.Config, account string, regions []string, rType string) (*ResourceDescribeOutput, error) {
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

		output := ResourceDescribeOutput{
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

			for i := range resp.resources {
				resp.resources[i].Account = account
				resp.resources[i].Region = resp.region
				resp.resources[i].Type = rType
			}

			output.Resources[resp.region] = resp.resources
		}

		return &output, nil
	}
}

// Sequentially describe the resources. If anyone of the regions fails, it will move on to the next region.
func SequentialDescribeGlobal(describe func(context.Context, aws.Config) ([]describer.Resource, error)) ResourceDescriber {
	return func(ctx context.Context, cfg aws.Config, account string, regions []string, rType string) (*ResourceDescribeOutput, error) {
		output := ResourceDescribeOutput{
			Resources: make(map[string][]describer.Resource, len(regions)),
			Errors:    make(map[string]string, len(regions)),
		}

		for _, region := range regions {
			// Make a shallow copy and override the default region
			rCfg := cfg.Copy()
			rCfg.Region = region

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
				resources[i].Region = "" // Ignore region for global resources
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
	return func(ctx context.Context, cfg aws.Config, account string, regions []string, rType string) (*ResourceDescribeOutput, error) {
		output := ResourceDescribeOutput{
			Resources: make(map[string][]describer.Resource, len(regions)),
			Errors:    make(map[string]string, len(regions)),
		}

		for _, region := range regions {
			// Make a shallow copy and override the default region
			rCfg := cfg.Copy()
			rCfg.Region = region

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
			for j, resource := range resources {
				resource.Account = account
				resource.Region = "" // Ignore region for global resources
				resource.Type = rType

				output.Resources[region][j] = resource
			}
		}

		return &output, nil
	}
}
