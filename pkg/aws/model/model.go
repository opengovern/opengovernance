//go:generate go run ../../keibi-es-sdk/gen/main.go --file $GOFILE --output ../../keibi-es-sdk/aws_resources_clients.go --type aws

package model

import (
	"time"

	accessanalyzer "github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"
	acm "github.com/aws/aws-sdk-go-v2/service/acm/types"
	amp "github.com/aws/aws-sdk-go-v2/service/amp/types"
	apigateway "github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	apigatewayv2 "github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	applicationautoscaling "github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
	appstream "github.com/aws/aws-sdk-go-v2/service/appstream/types"
	autoscaling "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	backupservice "github.com/aws/aws-sdk-go-v2/service/backup"
	backup "github.com/aws/aws-sdk-go-v2/service/backup/types"
	cloudfront "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	cloudtrailtypes "github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	cloudwatch "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	cloudwatchlogs "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	codebuild "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	configservice "github.com/aws/aws-sdk-go-v2/service/configservice/types"
	dms "github.com/aws/aws-sdk-go-v2/service/databasemigrationservice/types"
	dax "github.com/aws/aws-sdk-go-v2/service/dax/types"
	dynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodbstream "github.com/aws/aws-sdk-go-v2/service/dynamodbstreams/types"
	ec2op "github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ecrop "github.com/aws/aws-sdk-go-v2/service/ecr"
	ecr "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	ecrpublicop "github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	ecrpublic "github.com/aws/aws-sdk-go-v2/service/ecrpublic/types"
	ecs "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	efs "github.com/aws/aws-sdk-go-v2/service/efs/types"
	eks "github.com/aws/aws-sdk-go-v2/service/eks/types"
	elasticache "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	elasticbeanstalk "github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk/types"
	elasticloadbalancing "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"
	elasticloadbalancingv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	es "github.com/aws/aws-sdk-go-v2/service/elasticsearchservice/types"
	emr "github.com/aws/aws-sdk-go-v2/service/emr/types"
	eventbridge "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	fsx "github.com/aws/aws-sdk-go-v2/service/fsx/types"
	glacier "github.com/aws/aws-sdk-go-v2/service/glacier/types"
	grafana "github.com/aws/aws-sdk-go-v2/service/grafana/types"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	guarddutytypes "github.com/aws/aws-sdk-go-v2/service/guardduty/types"
	iam "github.com/aws/aws-sdk-go-v2/service/iam/types"
	kafka "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	keyspaces "github.com/aws/aws-sdk-go-v2/service/keyspaces/types"
	kinesis "github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	kms "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	memorydb "github.com/aws/aws-sdk-go-v2/service/memorydb/types"
	mq "github.com/aws/aws-sdk-go-v2/service/mq/types"
	mwaa "github.com/aws/aws-sdk-go-v2/service/mwaa/types"
	neptune "github.com/aws/aws-sdk-go-v2/service/neptune/types"
	opensearch "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	organizations "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	rds "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	redshifttypes "github.com/aws/aws-sdk-go-v2/service/redshift/types"
	redshiftserverlesstypes "github.com/aws/aws-sdk-go-v2/service/redshiftserverless/types"
	s3 "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	s3controltypes "github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	sagemakertypes "github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	ses "github.com/aws/aws-sdk-go-v2/service/ses/types"
	sesv2 "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	sns "github.com/aws/aws-sdk-go-v2/service/sns/types"
	ssm "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	wafv2 "github.com/aws/aws-sdk-go-v2/service/wafv2/types"
	workspaces "github.com/aws/aws-sdk-go-v2/service/workspaces/types"
)

type Metadata struct {
	Name         string `json:"name"`
	AccountID    string `json:"account_id"`
	SourceID     string `json:"source_id"`
	Region       string `json:"region"`
	Partition    string `json:"partition"`
	ResourceType string `json:"resource_type"`
}

//  ===================  Access Analyzer ==================

//index:aws_accessanalyzer_analyzer
//getfilter:name=description.Analyzer.Name
//listfilter:type=description.Analyzer.Type
type AccessAnalyzerAnalyzerDescription struct {
	Analyzer accessanalyzer.AnalyzerSummary
	Findings []accessanalyzer.FindingSummary
}

//  ===================   ApiGateway   ===================

//index:aws_apigateway_stage
//getfilter:rest_api_id=description.RestApiId
//getfilter:name=description.Stage.StageName
type ApiGatewayStageDescription struct {
	RestApiId *string
	Stage     apigateway.Stage
}

//index:aws_apigatewayv2_stage
//getfilter:api_id=description.ApiId
//getfilter:name=description.Stage.StageName
type ApiGatewayV2StageDescription struct {
	ApiId *string
	Stage apigatewayv2.Stage
}

//index:aws_apigateway_restapi
//getfilter:api_id=description.RestAPI.Id
type ApiGatewayRestAPIDescription struct {
	RestAPI apigateway.RestApi
}

//index:aws_apigatewayv2_api
//getfilter:api_id=description.API.ApiId
type ApiGatewayV2APIDescription struct {
	API apigatewayv2.Api
}

//  ===================   ElasticBeanstalk   ===================

//index:aws_elasticbeanstalk_environment
//getfilter:environment_name=description.EnvironmentDescription.EnvironmentName
type ElasticBeanstalkEnvironmentDescription struct {
	EnvironmentDescription elasticbeanstalk.EnvironmentDescription
	Tags                   []elasticbeanstalk.Tag
}

//  ===================   ElastiCache   ===================

//index:aws_elasticache_replicationgroup
//getfilter:replication_group_id=description.ReplicationGroup.ReplicationGroupId
type ElastiCacheReplicationGroupDescription struct {
	ReplicationGroup elasticache.ReplicationGroup
}

//index:aws_elasticache_cluster
//getfilter:cache_cluster_id=description.Cluster.CacheClusterId
type ElastiCacheClusterDescription struct {
	Cluster elasticache.CacheCluster
	TagList []elasticache.Tag
}

//  ===================   ElasticSearch   ===================

//index:aws_elasticsearch_domain
//getfilter:domain_name=description.Domain.DomainName
type ESDomainDescription struct {
	Domain es.ElasticsearchDomainStatus
	Tags   []es.Tag
}

//  ===================   EMR   ===================

//index:aws_emr_cluster
//getfilter:id=description.Cluster.Id
type EMRClusterDescription struct {
	Cluster *emr.Cluster
}

//  ===================   GuardDuty   ===================

//index:aws_guardduty_finding
type GuardDutyFindingDescription struct {
	Finding guarddutytypes.Finding
}

//index:aws_guardduty_detector
//getfilter:detector_id=description.DetectorId
type GuardDutyDetectorDescription struct {
	DetectorId string
	Detector   *guardduty.GetDetectorOutput
}

//  ===================   Backup   ===================

//index:aws_backup_plan
//getfilter:backup_plan_id=description.BackupPlan.BackupPlanId
type BackupPlanDescription struct {
	BackupPlan backup.BackupPlansListMember
}

//index:aws_backup_selection
//getfilter:backup_plan_id=description.BackupSelection.BackupPlanId
//getfilter:selection_id=description.BackupSelection.SelectionId
type BackupSelectionDescription struct {
	BackupSelection backup.BackupSelectionsListMember
	ListOfTags      []backup.Condition
	Resources       []string
}

//index:aws_backup_vault
//getfilter:name=description.BackupVault.BackupVaultName
type BackupVaultDescription struct {
	BackupVault       backup.BackupVaultListMember
	Policy            *string
	BackupVaultEvents []backup.BackupVaultEvent
	SNSTopicArn       *string
}

//index:aws_backup_recoverypoint
//getfilter:backup_vault_name=description.RecoveryPoint.BackupVaultName
//getfilter:recovery_point_arn=description.RecoveryPoint.RecoveryPointArn
//listfilter:recovery_point_arn=description.RecoveryPoint.RecoveryPointArn
//listfilter:resource_type=description.RecoveryPoint.ResourceType
//listfilter:completion_date=description.RecoveryPoint.CompletionDate
type BackupRecoveryPointDescription struct {
	RecoveryPoint *backupservice.DescribeRecoveryPointOutput
}

//index:aws_backup_protectedresource
//getfilter:resource_arn=description.ProtectedResource.ResourceArn
type BackupProtectedResourceDescription struct {
	ProtectedResource backup.ProtectedResource
}

//  ===================   CloudFront   ===================

//index:aws_cloudfront_distribution
//getfilter:id=description.Distribution.Id
type CloudFrontDistributionDescription struct {
	Distribution *cloudfront.Distribution
	ETag         *string
	Tags         []cloudfront.Tag
}

//  ===================   CloudWatch   ===================

//index:aws_cloudwatch_alarm
//getfilter:name=description.MetricAlarm.AlarmName
//listfilter:name=description.MetricAlarm.AlarmName
//listfilter:state_value=description.MetricAlarm.StateValue
type CloudWatchAlarmDescription struct {
	MetricAlarm cloudwatch.MetricAlarm
	Tags        []cloudwatch.Tag
}

//index:aws_logs_loggroup
//getfilter:name=description.LogGroup.LogGroupName
//listfilter:name=description.LogGroup.LogGroupName
type CloudWatchLogsLogGroupDescription struct {
	LogGroup cloudwatchlogs.LogGroup
	Tags     map[string]string
}

//index:aws_logs_metricfilter
//getfilter:name=decsription.MetricFilter.FilterName
//listfilter:name=decsription.MetricFilter.FilterName
//listfilter:log_group_name=decsription.MetricFilter.LogGroupName
//listfilter:metric_transformation_name=decsription.MetricFilter.MetricTransformations.MetricName
//listfilter:metric_transformation_namespace=decsription.MetricFilter.MetricTransformations.MetricNamespace
type CloudWatchLogsMetricFilterDescription struct {
	MetricFilter cloudwatchlogs.MetricFilter
}

//  ===================   CodeBuild   ===================

//index:aws_codebuild_project
//getfilter:name=description.Project.Name
type CodeBuildProjectDescription struct {
	Project codebuild.Project
}

//index:aws_codebuild_sourcecredential
type CodeBuildSourceCredentialDescription struct {
	SourceCredentialsInfo codebuild.SourceCredentialsInfo
}

//  ===================   Config   ===================

//index:aws_config_configurationrecorder
//getfilter:name=description.ConfigurationRecorder.Name
//listfilter:name=description.ConfigurationRecorder.Name
type ConfigConfigurationRecorderDescription struct {
	ConfigurationRecorder        configservice.ConfigurationRecorder
	ConfigurationRecordersStatus configservice.ConfigurationRecorderStatus
}

//  ===================   Dax   ===================

//index:aws_dax_cluster
//getfilter:cluster_name=description.Cluster.ClusterName
//listfilter:cluster_name=description.Cluster.ClusterName
type DAXClusterDescription struct {
	Cluster dax.Cluster
	Tags    []dax.Tag
}

//  ===================   Database Migration Service   ===================

//index:aws_dms_replicationinstance
//getfilter:arn=description.ReplicationInstance.ReplicationInstanceArn
//listfilter:replication_instance_identifier=description.ReplicationInstance.ReplicationInstanceIdentifier
//listfilter:arn=description.ReplicationInstance.ReplicationInstanceArn
//listfilter:replication_instance_class=description.ReplicationInstance.ReplicationInstanceClass
//listfilter:engine_version=description.ReplicationInstance.EngineVersion
type DMSReplicationInstanceDescription struct {
	ReplicationInstance dms.ReplicationInstance
	Tags                []dms.Tag
}

//  ===================   DynamoDb   ===================

//index:aws_dynamodb_table
//getfilter:name=description.Table.TableName
//listfilter:name=description.Table.TableName
type DynamoDbTableDescription struct {
	Table            *dynamodb.TableDescription
	ContinuousBackup *dynamodb.ContinuousBackupsDescription
	Tags             []dynamodb.Tag
}

//index:aws_dynamodb_globalsecondaryindex
//getfilter:index_arn=description.GlobalSecondaryIndex.IndexArn
type DynamoDbGlobalSecondaryIndexDescription struct {
	GlobalSecondaryIndex dynamodb.GlobalSecondaryIndexDescription
}

//index:aws_dynamodb_localsecondaryindex
//getfilter:index_arn=description.LocalSecondaryIndex.IndexArn
type DynamoDbLocalSecondaryIndexDescription struct {
	LocalSecondaryIndex dynamodb.LocalSecondaryIndexDescription
}

//index:aws_dynamodbstreams_stream
//getfilter:stream_arn=description.Stream.StreamArn
type DynamoDbStreamDescription struct {
	Stream dynamodbstream.Stream
}

//index:aws_dynamodb_backup
//getfilter:arn=description.Backup.BackupArn
//listfilter:backup_type=description.Backup.BackupType
//listfilter:arn=description.Backup.BackupArn
//listfilter:table_name=description.Backup.TableName
type DynamoDbBackupDescription struct {
	Backup dynamodb.BackupSummary
}

//index:aws_dynamodb_globaltable
//getfilter:global_table_name=description.GlobalTable.GlobalTableName
//listfilter:global_table_name=description.GlobalTable.GlobalTableName
type DynamoDbGlobalTableDescription struct {
	GlobalTable dynamodb.GlobalTableDescription
}

//  ===================   EC2   ===================

//index:aws_ec2_snapshot
//getfilter:snapshot_id=description.Snapshot.SnapshotId
//listfilter:description=description.Snapshot.Description
//listfilter:encrypted=description.Snapshot.Encrypted
//listfilter:owner_alias=description.Snapshot.OwnerAlias
//listfilter:owner_id=description.Snapshot.OwnerId
//listfilter:snapshot_id=description.Snapshot.SnapshotId
//listfilter:state=description.Snapshot.State
//listfilter:progress=description.Snapshot.Progress
//listfilter:volume_id=description.Snapshot.VolumeId
//listfilter:volume_size=description.Snapshot.VolumeSize
type EC2VolumeSnapshotDescription struct {
	Snapshot                *ec2.Snapshot
	CreateVolumePermissions []ec2.CreateVolumePermission
}

//index:aws_ec2_volume
//getfilter:volume_id=description.Volume.VolumeId
type EC2VolumeDescription struct {
	Volume     *ec2.Volume
	Attributes struct {
		AutoEnableIO bool
		ProductCodes []ec2.ProductCode
	}
}

//index:aws_ec2_instance
//getfilter:instance_id=description.Instance.InstanceId
//listfilter:hypervisor=description.Instance.Hypervisor
//listfilter:iam_instance_profile_arn=description.Instance.IamInstanceProfile.Arn
//listfilter:image_id=description.Instance.ImageId
//listfilter:instance_lifecycle=description.Instance.InstanceLifecycle
//listfilter:instance_state=description.Instance.State.Name
//listfilter:instance_type=description.Instance.InstanceType
//listfilter:monitoring_state=description.Instance.Monitoring.State
//listfilter:outpost_arn=description.Instance.OutpostArn
//listfilter:placement_availability_zone=description.Instance.Placement.AvailabilityZone
//listfilter:placement_group_name=description.Instance.Placement.GroupName
//listfilter:public_dns_name=description.Instance.PublicDnsName
//listfilter:ram_disk_id=description.Instance.RamdiskId
//listfilter:root_device_name=description.Instance.RootDeviceName
//listfilter:root_device_type=description.Instance.RootDeviceType
//listfilter:subnet_id=description.Instance.SubnetId
//listfilter:placement_tenancy=description.Instance.Placement.Tenancy
//listfilter:virtualization_type=description.Instance.VirtualizationType
//listfilter:vpc_id=description.Instance.VpcId
type EC2InstanceDescription struct {
	Instance       *ec2.Instance
	InstanceStatus *ec2.InstanceStatus
	Attributes     struct {
		UserData                          string
		InstanceInitiatedShutdownBehavior string
		DisableApiTermination             bool
	}
}

//index:aws_ec2_vpc
//getfilter:vpc_id=description.Vpc.VpcId
type EC2VpcDescription struct {
	Vpc ec2.Vpc
}

//index:aws_ec2_networkinterface
//getfilter:network_interface_id=description.NetworkInterface.NetworkInterfaceId
type EC2NetworkInterfaceDescription struct {
	NetworkInterface ec2.NetworkInterface
}

//index:aws_ec2_regionalsettings
type EC2RegionalSettingsDescription struct {
	EbsEncryptionByDefault *bool
	KmsKeyId               *string
}

//index:aws_ec2_subnet
//getfilter:subnet_id=description.Subnet.SubnetId
type EC2SubnetDescription struct {
	Subnet ec2.Subnet
}

//index:aws_ec2_vpcendpoint
//getfilter:vpc_endpoint_id=description.VpcEndpoint.VpcEndpointId
type EC2VPCEndpointDescription struct {
	VpcEndpoint ec2.VpcEndpoint
}

//index:aws_ec2_securitygroup
//getfilter:group_id=description.SecurityGroup.GroupId
type EC2SecurityGroupDescription struct {
	SecurityGroup ec2.SecurityGroup
}

//index:aws_ec2_eip
//getfilter:allocation_id=description.SecurityGroup.AllocationId
type EC2EIPDescription struct {
	Address ec2.Address
}

//index:aws_ec2_internetgateway
//getfilter:internet_gateway_id=description.InternetGateway.InternetGatewayId
type EC2InternetGatewayDescription struct {
	InternetGateway ec2.InternetGateway
}

//index:aws_ec2_networkacl
//getfilter:network_acl_id=description.NetworkAcl.NetworkAclId
type EC2NetworkAclDescription struct {
	NetworkAcl ec2.NetworkAcl
}

//index:aws_ec2_vpnconnection
//getfilter:vpn_connection_id=description.VpnConnection.VpnConnectionId
type EC2VPNConnectionDescription struct {
	VpnConnection ec2.VpnConnection
}

//index:aws_ec2_routetable
//getfilter:route_table_id=description.RouteTable.RouteTableId
type EC2RouteTableDescription struct {
	RouteTable ec2.RouteTable
}

//index:aws_ec2_natgateway
//getfilter:nat_gateway_id=description.NatGateway.NatGatewayId
type EC2NatGatewayDescription struct {
	NatGateway ec2.NatGateway
}

//index:aws_ec2_region
//getfilter:name=description.Region.RegionName
type EC2RegionDescription struct {
	Region ec2.Region
}

//index:aws_ec2_flowlog
//getfilter:flow_log_id=description.FlowLog.FlowLogId
type EC2FlowLogDescription struct {
	FlowLog ec2.FlowLog
}

//index:aws_ec2_capacityreservation
//getfilter:capacity_reservation_id=description.CapacityReservation.CapacityReservationId
type EC2CapacityReservationDescription struct {
	CapacityReservation ec2.CapacityReservation
}

//index:aws_ec2_keypair
//getfilter:key_name=description.KeyPair.KeyName
type EC2KeyPairDescription struct {
	KeyPair ec2.KeyPairInfo
}

//index:aws_ec2_ami
//getfilter:image_id=description.AMI.ImageId
type EC2AMIDescription struct {
	AMI               ec2.Image
	LaunchPermissions ec2op.DescribeImageAttributeOutput
}

//index:aws_ec2_reservedinstance
//getfilter:reserved_instance_id=description.ReservedInstance.ReservedInstancesId
type EC2ReservedInstancesDescription struct {
	ReservedInstances   ec2.ReservedInstances
	ModificationDetails []ec2.ReservedInstancesModification
}

//index:aws_ec2_capacityreservationfleet
//getfilter:capacity_reservation_fleet_id=description.CapacityReservationFleet.CapacityReservationFleetId
type EC2CapacityReservationFleetDescription struct {
	CapacityReservationFleet ec2.CapacityReservationFleet
}

//index:aws_ec2_fleet
//getfilter:fleet_id=description.Fleet.FleetId
type EC2FleetDescription struct {
	Fleet ec2.FleetData
}

//index:aws_ec2_host
//getfilter:host_id=description.Host.HostId
type EC2HostDescription struct {
	Host ec2.Host
}

//index:aws_ec2_placementgroup
//getfilter:group_name=description.PlacementGroup.GroupName
type EC2PlacementGroupDescription struct {
	PlacementGroup ec2.PlacementGroup
}

//  ===================  Elastic Load Balancing  ===================

//index:aws_elasticloadbalancingv2_loadbalancer
//getfilter:arn=description.LoadBalancer.LoadBalancerArn
//getfilter:type=description.LoadBalancer.Type
//listfilter:type=description.LoadBalancer.Type
type ElasticLoadBalancingV2LoadBalancerDescription struct {
	LoadBalancer elasticloadbalancingv2.LoadBalancer
	Attributes   []elasticloadbalancingv2.LoadBalancerAttribute
	Tags         []elasticloadbalancingv2.Tag
}

//index:aws_elasticloadbalancing_loadbalancer
//getfilter:name=description.LoadBalancer.LoadBalancerName
type ElasticLoadBalancingLoadBalancerDescription struct {
	LoadBalancer elasticloadbalancing.LoadBalancerDescription
	Attributes   *elasticloadbalancing.LoadBalancerAttributes
	Tags         []elasticloadbalancing.Tag
}

//index:aws_elasticloadbalancingv2_listener
//getfilter:arn=description.Listener.ListenerArn
type ElasticLoadBalancingV2ListenerDescription struct {
	Listener elasticloadbalancingv2.Listener
}

//  ===================  FSX  ===================

//index:aws_fsx_filesystem
//getfilter:file_system_id=description.FileSystem.FileSystemId
type FSXFileSystemDescription struct {
	FileSystem fsx.FileSystem
}

//index:aws_fsx_storagevirtualmachine
//getfilter:storage_virtual_machine_id=description.StorageVirtualMachine.StorageVirtualMachineId
type FSXStorageVirtualMachineDescription struct {
	StorageVirtualMachine fsx.StorageVirtualMachine
}

//index:aws_fsx_task
//getfilter:task_id=description.Task.TaskId
type FSXTaskDescription struct {
	Task fsx.DataRepositoryTask
}

//index:aws_fsx_volume
//getfilter:volume_id=description.Volume.VolumeId
type FSXVolumeDescription struct {
	Volume fsx.Volume
}

//index:aws_fsx_snapshot
//getfilter:snapshot_id=description.Snapshot.SnapshotId
type FSXSnapshotDescription struct {
	Snapshot fsx.Snapshot
}

//  ===================  Application Auto Scaling  ===================

//index:aws_applicationautoscaling_target
//getfilter:service_namespace=description.ScalableTarget.ServiceNamespace
//getfilter:resource_id=description.ScalableTarget.ResourceId
//listfilter:service_namespace=description.ScalableTarget.ServiceNamespace
//listfilter:resource_id=description.ScalableTarget.ResourceId
//listfilter:scalable_dimension=description.ScalableTarget.ScalableDimension
type ApplicationAutoScalingTargetDescription struct {
	ScalableTarget applicationautoscaling.ScalableTarget
}

//  ===================  Auto Scaling  ===================

//index:aws_autoscaling_autoscalinggroup
//getfilter:name=description.AutoScalingGroup.AutoScalingGroupName
type AutoScalingGroupDescription struct {
	AutoScalingGroup *autoscaling.AutoScalingGroup
	Policies         []autoscaling.ScalingPolicy
}

//index:aws_autoscaling_launchconfiguration
//getfilter:name=description.LaunchConfiguration.LaunchConfigurationName
type AutoScalingLaunchConfigurationDescription struct {
	LaunchConfiguration autoscaling.LaunchConfiguration
}

// ======================== ACM ==========================

//index:aws_certificatemanager_certificate
//getfilter:certificate_arn=description.Certificate.CertificateArn
//listfilter:status=description.Certificate.Status
type CertificateManagerCertificateDescription struct {
	Certificate acm.CertificateDetail
	Attributes  struct {
		Certificate      *string
		CertificateChain *string
	}
	Tags []acm.Tag
}

// =====================  CloudTrail  =====================

//index:aws_cloudtrail_trail
//getfilter:name=description.Trail.Name
//getfilter:arn=description.Trail.TrailARN
type CloudTrailTrailDescription struct {
	Trail                  cloudtrailtypes.Trail
	TrailStatus            cloudtrail.GetTrailStatusOutput
	EventSelectors         []cloudtrailtypes.EventSelector
	AdvancedEventSelectors []cloudtrailtypes.AdvancedEventSelector
	Tags                   []cloudtrailtypes.Tag
}

// ====================== IAM =========================

//index:aws_iam_account
type IAMAccountDescription struct {
	Aliases      []string
	Organization *organizations.Organization
}

type AccountSummary struct {
	AccountMFAEnabled                 int32
	AccessKeysPerUserQuota            int32
	AccountAccessKeysPresent          int32
	AccountSigningCertificatesPresent int32
	AssumeRolePolicySizeQuota         int32
	AttachedPoliciesPerGroupQuota     int32
	AttachedPoliciesPerRoleQuota      int32
	AttachedPoliciesPerUserQuota      int32
	GlobalEndpointTokenVersion        int32
	GroupPolicySizeQuota              int32
	Groups                            int32
	GroupsPerUserQuota                int32
	GroupsQuota                       int32
	InstanceProfiles                  int32
	InstanceProfilesQuota             int32
	MFADevices                        int32
	MFADevicesInUse                   int32
	Policies                          int32
	PoliciesQuota                     int32
	PolicySizeQuota                   int32
	PolicyVersionsInUse               int32
	PolicyVersionsInUseQuota          int32
	Providers                         int32
	RolePolicySizeQuota               int32
	Roles                             int32
	RolesQuota                        int32
	ServerCertificates                int32
	ServerCertificatesQuota           int32
	SigningCertificatesPerUserQuota   int32
	UserPolicySizeQuota               int32
	Users                             int32
	UsersQuota                        int32
	VersionsPerPolicyQuota            int32
}

//index:aws_iam_accountsummary
type IAMAccountSummaryDescription struct {
	AccountSummary AccountSummary
}

//index:aws_iam_accesskey
type IAMAccessKeyDescription struct {
	AccessKey iam.AccessKeyMetadata
}

//index:aws_iam_accountpasswordpolicy
type IAMAccountPasswordPolicyDescription struct {
	PasswordPolicy iam.PasswordPolicy
}

type InlinePolicy struct {
	PolicyName     string
	PolicyDocument string
}

//index:aws_iam_user
//getfilter:name=description.User.UserName
//getfilter:arn=description.User.Arn
type IAMUserDescription struct {
	User               iam.User
	Groups             []iam.Group
	InlinePolicies     []InlinePolicy
	AttachedPolicyArns []string
	MFADevices         []iam.MFADevice
}

//index:aws_iam_group
//getfilter:name=description.Group.GroupName
//getfilter:arn=description.Group.Arn
type IAMGroupDescription struct {
	Group              iam.Group
	Users              []iam.User
	InlinePolicies     []InlinePolicy
	AttachedPolicyArns []string
}

//index:aws_iam_role
//getfilter:name=description.Role.RoleName
//getfilter:arn=description.Role.Arn
type IAMRoleDescription struct {
	Role                iam.Role
	InstanceProfileArns []string
	InlinePolicies      []InlinePolicy
	AttachedPolicyArns  []string
}

//index:aws_iam_servercertificate
//getfilter:name=description.ServerCertificate.ServerCertificateMetadata.ServerCertificateName
type IAMServerCertificateDescription struct {
	ServerCertificate iam.ServerCertificate
}

//index:aws_iam_policy
//getfilter:arn=description.Policy.Arn
type IAMPolicyDescription struct {
	Policy        iam.Policy
	PolicyVersion iam.PolicyVersion
}

type CredentialReport struct {
	GeneratedTime             *time.Time `csv:"-"`
	UserArn                   string     `csv:"arn"`
	UserName                  string     `csv:"user"`
	UserCreationTime          string     `csv:"user_creation_time"`
	AccessKey1Active          bool       `csv:"access_key_1_active"`
	AccessKey1LastRotated     string     `csv:"access_key_1_last_rotated"`
	AccessKey1LastUsedDate    string     `csv:"access_key_1_last_used_date"`
	AccessKey1LastUsedRegion  string     `csv:"access_key_1_last_used_region"`
	AccessKey1LastUsedService string     `csv:"access_key_1_last_used_service"`
	AccessKey2Active          bool       `csv:"access_key_2_active"`
	AccessKey2LastRotated     string     `csv:"access_key_2_last_rotated"`
	AccessKey2LastUsedDate    string     `csv:"access_key_2_last_used_date"`
	AccessKey2LastUsedRegion  string     `csv:"access_key_2_last_used_region"`
	AccessKey2LastUsedService string     `csv:"access_key_2_last_used_service"`
	Cert1Active               bool       `csv:"cert_1_active"`
	Cert1LastRotated          string     `csv:"cert_1_last_rotated"`
	Cert2Active               bool       `csv:"cert_2_active"`
	Cert2LastRotated          string     `csv:"cert_2_last_rotated"`
	MFAActive                 bool       `csv:"mfa_active"`
	PasswordEnabled           string     `csv:"password_enabled"`
	PasswordLastChanged       string     `csv:"password_last_changed"`
	PasswordLastUsed          string     `csv:"password_last_used"`
	PasswordNextRotation      string     `csv:"password_next_rotation"`
}

//index:aws_iam_credentialreport
type IAMCredentialReportDescription struct {
	CredentialReport CredentialReport
}

//index:aws_iam_virtualmfadevices
type IAMVirtualMFADeviceDescription struct {
	VirtualMFADevice iam.VirtualMFADevice
	Tags             []iam.Tag
}

//  ===================  RDS  ===================

//index:aws_rds_dbcluster
//getfilter:db_cluster_identifier=description.DBCluster.DBClusterIdentifier
type RDSDBClusterDescription struct {
	DBCluster rds.DBCluster
}

//index:aws_rds_dbclustersnapshot
//getfilter:db_cluster_snapshot_identifier=description.DBClusterSnapshot.DBClusterIdentifier
//listfilter:db_cluster_identifier=description.DBClusterSnapshot.DBClusterIdentifier
//listfilter:db_cluster_snapshot_identifier=description.DBClusterSnapshot.DBClusterSnapshotIdentifier
//listfilter:engine=description.DBClusterSnapshot.Engine
//listfilter:type=description.DBClusterSnapshot.SnapshotType
type RDSDBClusterSnapshotDescription struct {
	DBClusterSnapshot rds.DBClusterSnapshot
	Attributes        *rds.DBClusterSnapshotAttributesResult
}

//index:aws_rds_eventsubscription
//getfilter:cust_subscription_id=description.EventSubscription.CustSubscriptionId
type RDSDBEventSubscriptionDescription struct {
	EventSubscription rds.EventSubscription
}

//index:aws_rds_dbinstance
//getfilter:db_instance_identifier=description.DBInstance.DBInstanceIdentifier
type RDSDBInstanceDescription struct {
	DBInstance rds.DBInstance
}

//index:aws_rds_dbsnapshot
//getfilter:db_snapshot_identifier=description.DBSnapshot.DBInstanceIdentifier
type RDSDBSnapshotDescription struct {
	DBSnapshot           rds.DBSnapshot
	DBSnapshotAttributes []rds.DBSnapshotAttribute
}

//index:aws_rds_globalcluster
//getfilter:global_cluster_identifier=description.DBGlobalCluster.GlobalClusterIdentifier
type RDSGlobalClusterDescription struct {
	GlobalCluster rds.GlobalCluster
	Tags          []rds.Tag
}

//  ===================  Redshift  ===================

//index:aws_redshift_cluster
//getfilter:cluster_identifier=description.Cluster
type RedshiftClusterDescription struct {
	Cluster          redshifttypes.Cluster
	LoggingStatus    *redshift.DescribeLoggingStatusOutput
	ScheduledActions []redshifttypes.ScheduledAction
}

//index:aws_redshift_clusterparametergroup
//getfilter:name=description.ClusterParameterGroup.ParameterGroupName
type RedshiftClusterParameterGroupDescription struct {
	ClusterParameterGroup redshifttypes.ClusterParameterGroup
	Parameters            []redshifttypes.Parameter
}

//index:aws_redshift_snapshot
//getfilter:snapshot_identifier=description.Snapshot.SnapshotIdentifier
type RedshiftSnapshotDescription struct {
	Snapshot redshifttypes.Snapshot
}

//index:aws_redshiftserverless_namespace
//getfilter:namespace_name=description.Namespace.NamespaceName
type RedshiftServerlessNamespaceDescription struct {
	Namespace redshiftserverlesstypes.Namespace
	Tags      []redshiftserverlesstypes.Tag
}

//index:aws_redshiftserverless_snapshot
//getfilter:snapshot_name=description.Snapshot.SnapshotName
type RedshiftServerlessSnapshotDescription struct {
	Snapshot redshiftserverlesstypes.Snapshot
	Tags     []redshiftserverlesstypes.Tag
}

//  ===================  SNS  ===================

//index:aws_sns_topic
//getfilter:topic_arn=description.Attributes.TopicArn
type SNSTopicDescription struct {
	Attributes map[string]string
	Tags       []sns.Tag
}

//index:aws_sns_subscription
//getfilter:subscription_arn=description.Subscription.SubscriptionArn
type SNSSubscriptionDescription struct {
	Subscription sns.Subscription
	Attributes   map[string]string
}

//  ===================  SQS  ===================

//index:aws_sqs_queue
//getfilter:queue_url=description.Attributes.QueueUrl
type SQSQueueDescription struct {
	Attributes map[string]string
	Tags       map[string]string
}

//  ===================  S3  ===================

//index:aws_s3_bucket
//getfilter:name=description.Bucket.Name
type S3BucketDescription struct {
	Bucket    s3.Bucket
	BucketAcl struct {
		Grants []s3.Grant
		Owner  *s3.Owner
	}
	Policy                         *string
	PolicyStatus                   *s3.PolicyStatus
	PublicAccessBlockConfiguration *s3.PublicAccessBlockConfiguration
	Versioning                     struct {
		MFADelete s3.MFADeleteStatus
		Status    s3.BucketVersioningStatus
	}
	LifecycleRules                    string
	LoggingEnabled                    *s3.LoggingEnabled
	ServerSideEncryptionConfiguration *s3.ServerSideEncryptionConfiguration
	ObjectLockConfiguration           *s3.ObjectLockConfiguration
	ReplicationConfiguration          string
	Tags                              []s3.Tag
}

//index:aws_s3_accountsettingdescription
type S3AccountSettingDescription struct {
	PublicAccessBlockConfiguration s3controltypes.PublicAccessBlockConfiguration
}

//  ===================  SageMaker  ===================

//index:aws_sagemaker_endpointconfiguration
//getfilter:name=description.EndpointConfig.EndpointConfigName
type SageMakerEndpointConfigurationDescription struct {
	EndpointConfig *sagemaker.DescribeEndpointConfigOutput
	Tags           []sagemakertypes.Tag
}

//index:aws_sagemaker_notebookinstance
//getfilter:name=description.NotebookInstance.NotebookInstanceName
type SageMakerNotebookInstanceDescription struct {
	NotebookInstance *sagemaker.DescribeNotebookInstanceOutput
	Tags             []sagemakertypes.Tag
}

//  ===================  SecretsManager  ===================

//index:aws_secretsmanager_secret
//getfilter:arn=description.Secret.ARN
type SecretsManagerSecretDescription struct {
	Secret         *secretsmanager.DescribeSecretOutput
	ResourcePolicy *string
}

//  ===================  SecurityHub  ===================

//index:aws_securityhub_hub
//getfilter:hub_arn=description.Hub.HubArn
type SecurityHubHubDescription struct {
	Hub  *securityhub.DescribeHubOutput
	Tags map[string]string
}

//  ===================  SSM  ===================

//index:aws_ssm_managedinstance
type SSMManagedInstanceDescription struct {
	InstanceInformation ssm.InstanceInformation
}

//index:aws_ssm_managedinstancecompliance
//listfilter:resource_id=description.ComplianceItem.ResourceId
type SSMManagedInstanceComplianceDescription struct {
	ComplianceItem ssm.ComplianceItem
}

//  ===================  ECS  ===================

//index:aws_ecs_taskdefinition
//getfilter:task_definition_arn=description.TaskDefinition.TaskDefinitionArn
type ECSTaskDefinitionDescription struct {
	TaskDefinition *ecs.TaskDefinition
	Tags           []ecs.Tag
}

//index:aws_ecs_cluster
//getfilter:cluster_arn=description.Cluster.ClusterArn
type ECSClusterDescription struct {
	Cluster ecs.Cluster
}

//index:aws_ecs_service
type ECSServiceDescription struct {
	Service ecs.Service
}

//index:aws_ecs_containerinstance
type ECSContainerInstanceDescription struct {
	ContainerInstance ecs.ContainerInstance
}

//index:aws_ecs_taskset
//getfilter:id=description.TaskSet.Id
type ECSTaskSetDescription struct {
	TaskSet ecs.TaskSet
}

//  ===================  EFS  ===================

//index:aws_efs_filesystem
//getfilter:aws_efs_file_system=description.FileSystem.FileSystemId
type EFSFileSystemDescription struct {
	FileSystem efs.FileSystemDescription
	Policy     *string
}

//  ===================  EKS  ===================

//index:aws_eks_cluster
//getfilter:name=description.Cluster.Name
type EKSClusterDescription struct {
	Cluster eks.Cluster
}

//index:aws_eks_addon
//getfilter:addon_name=description.Addon.AddonName
//getfilter:cluster_name=description.Addon.ClusterName
type EKSAddonDescription struct {
	Addon eks.Addon
}

//index:aws_eks_identityproviderconfig
//getfilter:name=description.ConfigName
//getfilter:type=description.ConfigType
//getfilter:cluster_name=description.IdentityProviderConfig.ClusterName
type EKSIdentityProviderConfigDescription struct {
	ConfigName             string
	ConfigType             string
	IdentityProviderConfig eks.OidcIdentityProviderConfig
}

//index:aws_eks_nodegroup
//getfilter:nodegroup_name=description.Nodegroup.NodegroupName
type EKSNodegroupDescription struct {
	Nodegroup eks.Nodegroup
}

//  ===================  WAFv2  ===================

//index:aws_wafv2_webacl
//getfilter:id=description.WebACL.Id
//getfilter:name=description.WebACL.Name
//getfilter:scope=description.Scope
type WAFv2WebACLDescription struct {
	WebACL               *wafv2.WebACL
	Scope                wafv2.Scope
	LoggingConfiguration *wafv2.LoggingConfiguration
	TagInfoForResource   *wafv2.TagInfoForResource
	LockToken            *string
}

//  ===================  KMS  ===================

//index:aws_kms_key
//getfilter:id=description.Metadata.KeyId
type KMSKeyDescription struct {
	Metadata           *kms.KeyMetadata
	Aliases            []kms.AliasListEntry
	KeyRotationEnabled bool
	Policy             *string
	Tags               []kms.Tag
}

//  ===================  Lambda  ===================

//index:aws_lambda_function
//getfilter:name=description.Function.Configuration.FunctionName
type LambdaFunctionDescription struct {
	Function *lambda.GetFunctionOutput
	Policy   *lambda.GetPolicyOutput
}

//index:aws_s3_accesspoint
//getfilter:name=description.AccessPoint.Name
//getfilter:region=metadata.region
type S3AccessPointDescription struct {
	AccessPoint  *s3control.GetAccessPointOutput
	Policy       *string
	PolicyStatus *s3controltypes.PolicyStatus
}

type CostExplorerRow struct {
	Estimated bool

	// The time period that the result covers.
	PeriodStart *string
	PeriodEnd   *string

	Dimension1 *string
	Dimension2 *string
	//Tag *string

	BlendedCostAmount      *string
	UnblendedCostAmount    *string
	NetUnblendedCostAmount *string
	AmortizedCostAmount    *string
	NetAmortizedCostAmount *string
	UsageQuantityAmount    *string
	NormalizedUsageAmount  *string

	BlendedCostUnit      *string
	UnblendedCostUnit    *string
	NetUnblendedCostUnit *string
	AmortizedCostUnit    *string
	NetAmortizedCostUnit *string
	UsageQuantityUnit    *string
	NormalizedUsageUnit  *string
}

//index:aws_costexplorer_byaccountmonthly
type CostExplorerByAccountMonthlyDescription struct {
	CostExplorerRow
}

//index:aws_costexplorer_byservicemonthly
type CostExplorerByServiceMonthlyDescription struct {
	CostExplorerRow
}

//  ===================  ECR  ===================

//index:aws_ecr_repository
//getfilter:repository_name=description.Repository.RepositoryName
type ECRRepositoryDescription struct {
	Repository      ecr.Repository
	LifecyclePolicy *ecrop.GetLifecyclePolicyOutput
	ImageDetails    []ecr.ImageDetail
	Policy          *ecrop.GetRepositoryPolicyOutput
	Tags            []ecr.Tag
}

//index:aws_ecrpublic_repository
//getfilter:repository_name=description.PublicRepository.RepositoryName
type ECRPublicRepositoryDescription struct {
	PublicRepository ecrpublic.Repository
	ImageDetails     []ecrpublic.ImageDetail
	Policy           *ecrpublicop.GetRepositoryPolicyOutput
	Tags             []ecrpublic.Tag
}

//index:aws_ecrpublic_registry
//getfilter:registry_id=description.PublicRegistry.RegistryId
type ECRPublicRegistryDescription struct {
	PublicRegistry ecrpublic.Registry
	Tags           []ecrpublic.Tag
}

//  ===================  EventBridge  ===================

//index:aws_eventbridge_bus
//getfilter:arn=description.Bus.Arn
type EventBridgeBusDescription struct {
	Bus  eventbridge.EventBus
	Tags []eventbridge.Tag
}

//  ===================  AppStream  ===================

//index:aws_appstream_application
//getfilter:name=description.Application.Name
type AppStreamApplicationDescription struct {
	Application appstream.Application
	Tags        map[string]string
}

//index:aws_appstream_stack
//getfilter:name=description.Stack.Name
type AppStreamStackDescription struct {
	Stack appstream.Stack
	Tags  map[string]string
}

//index:aws_appstream_fleet
//getfilter:name=description.Fleet.Name
type AppStreamFleetDescription struct {
	Fleet appstream.Fleet
	Tags  map[string]string
}

//  ===================  Kinesis  ===================

//index:aws_kinesis_stream
//getfilter:stream_name=description.Stream.StreamName
type KinesisStreamDescription struct {
	Stream             kinesis.StreamDescription
	DescriptionSummary kinesis.StreamDescriptionSummary
	Tags               []kinesis.Tag
}

//  ===================  Glacier  ===================

//index:aws_glacier_vault
//getfilter:vault_name=description.Vault.VaultName
type GlacierVaultDescription struct {
	Vault        glacier.DescribeVaultOutput
	AccessPolicy glacier.VaultAccessPolicy
	LockPolicy   glacier.VaultLockPolicy
	Tags         map[string]string
}

//  ===================  Workspace  ===================

//index:aws_workspaces_workspace
//getfilter:workspace_id=description.Workspace.WorkspaceId
type WorkspacesWorkspaceDescription struct {
	Workspace workspaces.Workspace
	Tags      []workspaces.Tag
}

//index:aws_workspaces_bundle
//getfilter:bundle_id=description.Bundle.BundleId
type WorkspacesBundleDescription struct {
	Bundle workspaces.WorkspaceBundle
	Tags   []workspaces.Tag
}

//  ===================  KeySpaces (For Apache Cassandra)  ===================

//index:aws_keyspaces_keyspace
//getfilter:keyspace_name=description.Keyspace.KeyspaceName
type KeyspacesKeyspaceDescription struct {
	Keyspace keyspaces.KeyspaceSummary
	Tags     []keyspaces.Tag
}

//index:aws_keyspaces_table
//getfilter:table_name=description.Table.TableName
type KeyspacesTableDescription struct {
	Table keyspaces.TableSummary
	Tags  []keyspaces.Tag
}

//  ===================  Grafana  ===================

//index:aws_grafana_workspace
//getfilter:id=description.Workspace.Id
type GrafanaWorkspaceDescription struct {
	Workspace grafana.WorkspaceSummary
}

//  ===================  AMP (Amazon Managed Service for Prometheus)  ===================

//index:aws_amp_workspace
//getfilter:workspace_id=description.Workspace.WorkspaceId
type AMPWorkspaceDescription struct {
	Workspace amp.WorkspaceSummary
}

//  ===================  Kafka  ===================

//index:aws_kafka_cluster
//getfilter:cluster_name=description.Cluster.ClusterName
type KafkaClusterDescription struct {
	Cluster kafka.Cluster
}

//  ===================  MWAA (Managed Workflows for Apache Airflow) ===================

//index:aws_mwaa_environment
//getfilter:name=description.Environment.Name
type MWAAEnvironmentDescription struct {
	Environment mwaa.Environment
}

//  ===================  MemoryDb  ===================

//index:aws_memorydb_cluster
//getfilter:name=description.Cluster.Name
type MemoryDbClusterDescription struct {
	Cluster memorydb.Cluster
	Tags    []memorydb.Tag
}

//  ===================  MQ  ===================

//index:aws_mq_broker
//getfilter:broker_name=description.Broker.BrokerName
type MQBrokerDescription struct {
	Broker mq.BrokerSummary
	Tags   map[string]string
}

//  ===================  Neptune  ===================

//index:aws_neptune_database
//getfilter:db_name=description.Database.DBName
type NeptuneDatabaseDescription struct {
	Database neptune.DBInstance
	Tags     []neptune.Tag
}

//  ===================  OpenSearch  ===================

//index:aws_opensearch_domain
//getfilter:domain_name=description.Domain.DomainName
type OpenSearchDomainDescription struct {
	Domain opensearch.DomainStatus
	Tags   []opensearch.Tag
}

//  ===================  SES (Simple Email Service)  ===================

//index:aws_ses_configurtionset
//getfilter:name=description.ConfigurationSet.Name
type SESConfigurationSetDescription struct {
	ConfigurationSet ses.ConfigurationSet
}

//index:aws_ses_identity
//getfilter:identity_name=description.Identity.IdentityName
type SESIdentityDescription struct {
	Identity sesv2.IdentityInfo
	Tags     []sesv2.Tag
}
