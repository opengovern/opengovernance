//go:generate go run ../../keibi-es-sdk/gen/main.go --file $GOFILE --output ../../keibi-es-sdk/aws_resources_clients.go --type aws

package model

import (
	"time"

	accessanalyzer "github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"
	account "github.com/aws/aws-sdk-go-v2/service/account/types"
	acm "github.com/aws/aws-sdk-go-v2/service/acm/types"
	acmpca "github.com/aws/aws-sdk-go-v2/service/acmpca/types"
	amp "github.com/aws/aws-sdk-go-v2/service/amp/types"
	amplify "github.com/aws/aws-sdk-go-v2/service/amplify/types"
	apigateway "github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	apigatewayv2 "github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	appconfig "github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	applicationautoscaling "github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
	appstream "github.com/aws/aws-sdk-go-v2/service/appstream/types"
	auditmanager "github.com/aws/aws-sdk-go-v2/service/auditmanager/types"
	autoscaling "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	backupop "github.com/aws/aws-sdk-go-v2/service/backup"
	backupservice "github.com/aws/aws-sdk-go-v2/service/backup"
	backup "github.com/aws/aws-sdk-go-v2/service/backup/types"
	batch "github.com/aws/aws-sdk-go-v2/service/batch/types"
	cloudcontrol "github.com/aws/aws-sdk-go-v2/service/cloudcontrol/types"
	cloudformationop "github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cloudformation "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	cloudfrontop "github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cloudfront "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	cloudsearch "github.com/aws/aws-sdk-go-v2/service/cloudsearch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	cloudtrailtypes "github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	cloudwatch "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	cloudwatchlogs "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	codeartifact "github.com/aws/aws-sdk-go-v2/service/codeartifact/types"
	codebuild "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	codecommit "github.com/aws/aws-sdk-go-v2/service/codecommit/types"
	codedeploy "github.com/aws/aws-sdk-go-v2/service/codedeploy/types"
	codepipeline "github.com/aws/aws-sdk-go-v2/service/codepipeline/types"
	codestarop "github.com/aws/aws-sdk-go-v2/service/codestar"
	configservice "github.com/aws/aws-sdk-go-v2/service/configservice/types"
	dms "github.com/aws/aws-sdk-go-v2/service/databasemigrationservice/types"
	dax "github.com/aws/aws-sdk-go-v2/service/dax/types"
	directconnect "github.com/aws/aws-sdk-go-v2/service/directconnect/types"
	directoryservice "github.com/aws/aws-sdk-go-v2/service/directoryservice/types"
	dlm "github.com/aws/aws-sdk-go-v2/service/dlm/types"
	docdb "github.com/aws/aws-sdk-go-v2/service/docdb/types"
	drs "github.com/aws/aws-sdk-go-v2/service/drs/types"
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
	eventbridgeop "github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridge "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	fms "github.com/aws/aws-sdk-go-v2/service/fms/types"
	fsx "github.com/aws/aws-sdk-go-v2/service/fsx/types"
	glacier "github.com/aws/aws-sdk-go-v2/service/glacier/types"
	globalaccelerator "github.com/aws/aws-sdk-go-v2/service/globalaccelerator/types"
	glueop "github.com/aws/aws-sdk-go-v2/service/glue"
	glue "github.com/aws/aws-sdk-go-v2/service/glue/types"
	grafana "github.com/aws/aws-sdk-go-v2/service/grafana/types"
	guarddutyop "github.com/aws/aws-sdk-go-v2/service/guardduty"
	guardduty "github.com/aws/aws-sdk-go-v2/service/guardduty/types"
	health "github.com/aws/aws-sdk-go-v2/service/health/types"
	iamop "github.com/aws/aws-sdk-go-v2/service/iam"
	iam "github.com/aws/aws-sdk-go-v2/service/iam/types"
	identitystore "github.com/aws/aws-sdk-go-v2/service/identitystore/types"
	imagebuilder "github.com/aws/aws-sdk-go-v2/service/imagebuilder/types"
	inspector "github.com/aws/aws-sdk-go-v2/service/inspector/types"
	kafka "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	keyspaces "github.com/aws/aws-sdk-go-v2/service/keyspaces/types"
	kinesis "github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	kinesisanalyticsv2 "github.com/aws/aws-sdk-go-v2/service/kinesisanalyticsv2/types"
	kms "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	memorydb "github.com/aws/aws-sdk-go-v2/service/memorydb/types"
	mq "github.com/aws/aws-sdk-go-v2/service/mq/types"
	mwaa "github.com/aws/aws-sdk-go-v2/service/mwaa/types"
	neptune "github.com/aws/aws-sdk-go-v2/service/neptune/types"
	networkfirewall "github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	opensearch "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	opsworkscm "github.com/aws/aws-sdk-go-v2/service/opsworkscm/types"
	organizations "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	rds "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	redshifttypes "github.com/aws/aws-sdk-go-v2/service/redshift/types"
	redshiftserverlesstypes "github.com/aws/aws-sdk-go-v2/service/redshiftserverless/types"
	route53op "github.com/aws/aws-sdk-go-v2/service/route53"
	route53 "github.com/aws/aws-sdk-go-v2/service/route53/types"
	s3 "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	s3controltypes "github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	sagemakertypes "github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	ses "github.com/aws/aws-sdk-go-v2/service/ses/types"
	sesv2 "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	shield "github.com/aws/aws-sdk-go-v2/service/shield/types"
	sns "github.com/aws/aws-sdk-go-v2/service/sns/types"
	ssm "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	ssoadmin "github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	storagegateway "github.com/aws/aws-sdk-go-v2/service/storagegateway/types"
	waf "github.com/aws/aws-sdk-go-v2/service/waf/types"
	wafregional "github.com/aws/aws-sdk-go-v2/service/wafregional/types"
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

//index:aws_apigateway_apikey
//getfilter:id=description.ApiKey.Id
//listfilter:customer_id=description.ApiKey.CustomerId
type ApiGatewayApiKeyDescription struct {
	ApiKey apigateway.ApiKey
}

//index:aws_apigateway_usageplan
//getfilter:id=description.UsagePlan.Id
type ApiGatewayUsagePlanDescription struct {
	UsagePlan apigateway.UsagePlan
}

//index:aws_apigateway_authorizer
//getfilter:id=description.Authorizer.Id
//getfilter:rest_api_id=description.RestApiId
type ApiGatewayAuthorizerDescription struct {
	Authorizer apigateway.Authorizer
	RestApiId  string
}

//index:aws_apigatewayv2_api
//getfilter:api_id=description.API.ApiId
type ApiGatewayV2APIDescription struct {
	API apigatewayv2.Api
}

//index:aws_apigatewayv2_domainname
//getfilter:domain_name=description.DomainName.DomainName
type ApiGatewayV2DomainNameDescription struct {
	DomainName apigatewayv2.DomainName
}

//index:aws_apigatewayv2_integration
//getfilter:integration_id=description.Integration.IntegrationId
//getfilter:api_id=description.ApiId
type ApiGatewayV2IntegrationDescription struct {
	Integration apigatewayv2.Integration
	ApiId       string
}

//  ===================   ElasticBeanstalk   ===================

//index:aws_elasticbeanstalk_environment
//getfilter:environment_name=description.EnvironmentDescription.EnvironmentName
type ElasticBeanstalkEnvironmentDescription struct {
	EnvironmentDescription elasticbeanstalk.EnvironmentDescription
	Tags                   []elasticbeanstalk.Tag
}

//index:aws_elasticbeanstalk_application
//getfilter:name=description.Application.ApplicationName
type ElasticBeanstalkApplicationDescription struct {
	Application elasticbeanstalk.ApplicationDescription
	Tags        []elasticbeanstalk.Tag
}

//index:aws_elasticbeanstalk_platform
//getfilter:platform_name=description.Platform.PlatformName
type ElasticBeanstalkPlatformDescription struct {
	Platform elasticbeanstalk.PlatformDescription
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

//index:aws_elasticache_parametergroup
//getfilter:cache_parameter_group_name=description.ParameterGroup.CacheParameterGroupName
type ElastiCacheParameterGroupDescription struct {
	ParameterGroup elasticache.CacheParameterGroup
}

//index:aws_elasticache_reservedcachenode
//getfilter:reserved_cache_node_id=description.ReservedCacheNode.ReservedCacheNodeId
//listfilter:cache_node_type=description.ReservedCacheNode.CacheNodeType
//listfilter:duration=description.ReservedCacheNode.Duration
//listfilter:offering_type=description.ReservedCacheNode.OfferingType
//listfilter:reserved_cache_nodes_offering_id=description.ReservedCacheNode.ReservedCacheNodesOfferingId
type ElastiCacheReservedCacheNodeDescription struct {
	ReservedCacheNode elasticache.ReservedCacheNode
}

//index:aws_elasticache_subnetgroup
//getfilter:cache_subnet_group_name=description.SubnetGroup.CacheSubnetGroupName
type ElastiCacheSubnetGroupDescription struct {
	SubnetGroup elasticache.CacheSubnetGroup
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

//index:aws_emr_instance
//listfilter:cluster_id=description.ClusterID
//listfilter:instance_fleet_id=description.Instance.InstanceFleetId
//listfilter:instance_group_id=description.Instance.InstanceGroupId
type EMRInstanceDescription struct {
	Instance  emr.Instance
	ClusterID string
}

//index:aws_emr_instancefleet
type EMRInstanceFleetDescription struct {
	InstanceFleet emr.InstanceFleet
	ClusterID     string
}

//index:aws_emr_instancegroup
type EMRInstanceGroupDescription struct {
	InstanceGroup emr.InstanceGroup
	ClusterID     string
}

//  ===================   GuardDuty   ===================

//index:aws_guardduty_finding
type GuardDutyFindingDescription struct {
	Finding guardduty.Finding
}

//index:aws_guardduty_detector
//getfilter:detector_id=description.DetectorId
type GuardDutyDetectorDescription struct {
	DetectorId string
	Detector   *guarddutyop.GetDetectorOutput
}

//index:aws_guardduty_filter
//getfilter:name=description.Filter.Name
//getfilter:detector_id=description.DetectorId
//listfilter:detector_id=description.DetectorId
type GuardDutyFilterDescription struct {
	Filter     guarddutyop.GetFilterOutput
	DetectorId string
}

//index:aws_guardduty_ipset
//getfilter:ipset_id=description.IPSetId
//getfilter:detector_id=description.DetectorId
//listfilter:detector_id=description.DetectorId
type GuardDutyIPSetDescription struct {
	IPSet      guarddutyop.GetIPSetOutput
	IPSetId    string
	DetectorId string
}

//index:aws_guardduty_member
//getfilter:member_account_id=description.Member.AccountId
//getfilter:detector_id=description.Member.DetectorId
//listfilter:detector_id=description.Member.DetectorId
type GuardDutyMemberDescription struct {
	Member guardduty.Member
}

//index:aws_guardduty_publishingdestination
//getfilter:destination_id=description.PublishingDestination.DestinationId
//getfilter:detector_id=description.DetectorId
//listfilter:detector_id=description.DetectorId
type GuardDutyPublishingDestinationDescription struct {
	PublishingDestination guarddutyop.DescribePublishingDestinationOutput
	DetectorId            string
}

//index:aws_guardduty_threatintelset
//getfilter:threat_intel_set_id=description.ThreatIntelSetID
//getfilter:detector_id=description.DetectorId
//listfilter:detector_id=description.DetectorId
type GuardDutyThreatIntelSetDescription struct {
	ThreatIntelSet   guarddutyop.GetThreatIntelSetOutput
	DetectorId       string
	ThreatIntelSetID string
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

//index:aws_backup_framework
//getfilter:framework_name=description.Framework.FrameworkName
type BackupFrameworkDescription struct {
	Framework backupop.DescribeFrameworkOutput
	Tags      map[string]string
}

//index:aws_backup_legalhold
//getfilter:legal_hold_id=description.Framework.LegalHoldId
type BackupLegalHoldDescription struct {
	LegalHold backupop.GetLegalHoldOutput
}

//  ===================   CloudFront   ===================

//index:aws_cloudfront_distribution
//getfilter:id=description.Distribution.Id
type CloudFrontDistributionDescription struct {
	Distribution *cloudfront.Distribution
	ETag         *string
	Tags         []cloudfront.Tag
}

//index:aws_cloudfront_originaccesscontrol
//getfilter:id=description.OriginAccessControl.Id
type CloudFrontOriginAccessControlDescription struct {
	OriginAccessControl cloudfront.OriginAccessControlSummary
	Tags                []cloudfront.Tag
}

//index:aws_cloudfront_cachepolicy
//getfilter:id=description.CachePolicy.Id
type CloudFrontCachePolicyDescription struct {
	CachePolicy cloudfrontop.GetCachePolicyOutput
}

//index:aws_cloudfront_function
//getfilter:name=description.Function.FunctionSummary.Name
type CloudFrontFunctionDescription struct {
	Function cloudfrontop.DescribeFunctionOutput
}

//index:aws_cloudfront_originaccessidentity
//getfilter:id=description.OriginAccessIdentity.CloudFrontOriginAccessIdentity.Id
type CloudFrontOriginAccessIdentityDescription struct {
	OriginAccessIdentity cloudfrontop.GetCloudFrontOriginAccessIdentityOutput
}

//index:aws_cloudfront_originrequestpolicy
//getfilter:id=description.OriginRequestPolicy.OriginRequestPolicy.Id
type CloudFrontOriginRequestPolicyDescription struct {
	OriginRequestPolicy cloudfrontop.GetOriginRequestPolicyOutput
}

//index:aws_cloudfront_responseheaderspolicy
type CloudFrontResponseHeadersPolicyDescription struct {
	ResponseHeadersPolicy cloudfrontop.GetResponseHeadersPolicyOutput
}

//  ===================   CloudWatch   ===================

type CloudWatchMetricRow struct {
	// The (single) metric Dimension name
	DimensionName *string

	// The value for the (single) metric Dimension
	DimensionValue *string

	// The namespace of the metric
	Namespace *string

	// The name of the metric
	MetricName *string

	// The average of the metric values that correspond to the data point.
	Average *float64

	// The percentile statistic for the data point.
	//ExtendedStatistics map[string]*float64 `type:"map"`

	// The maximum metric value for the data point.
	Maximum *float64

	// The minimum metric value for the data point.
	Minimum *float64

	// The number of metric values that contributed to the aggregate value of this
	// data point.
	SampleCount *float64

	// The sum of the metric values for the data point.
	Sum *float64

	// The time stamp used for the data point.
	Timestamp *time.Time

	// The standard unit for the data point.
	Unit *string
}

//index:aws_cloudwatch_alarm
//getfilter:name=description.MetricAlarm.AlarmName
//listfilter:name=description.MetricAlarm.AlarmName
//listfilter:state_value=description.MetricAlarm.StateValue
type CloudWatchAlarmDescription struct {
	MetricAlarm cloudwatch.MetricAlarm
	Tags        []cloudwatch.Tag
}

//index:aws_cloudwatch_logevent
//listfilter:log_stream_name=description.LogEvent.LogStreamName
//listfilter:timestamp=description.LogEvent.Timestamp
type CloudWatchLogEventDescription struct {
	LogEvent     cloudwatchlogs.FilteredLogEvent
	LogGroupName string
}

//index:aws_cloudwatch_logresourcepolicy
type CloudWatchLogResourcePolicyDescription struct {
	ResourcePolicy cloudwatchlogs.ResourcePolicy
}

//index:aws_cloudwatch_logstream
//getfilter:name=description.LogStream.LogStreamName
//listfilter:name=description.LogStream.LogStreamName
type CloudWatchLogStreamDescription struct {
	LogStream    cloudwatchlogs.LogStream
	LogGroupName string
}

//index:aws_cloudwatch_logsubscriptionfilter
//getfilter:name=description.SubscriptionFilter.FilterName
//getfilter:log_group_name=description.SubscriptionFilter.LogGroupName
//listfilter:name=description.SubscriptionFilter.FilterName
//listfilter:log_group_name=description.SubscriptionFilter.LogGroupName
type CloudWatchLogSubscriptionFilterDescription struct {
	SubscriptionFilter cloudwatchlogs.SubscriptionFilter
	LogGroupName       string
}

//index:aws_cloudwatch_metric
//listfilter:metric_name=description.Metric.MetricName
//listfilter:namespace=description.Metric.Namespace
type CloudWatchMetricDescription struct {
	Metric cloudwatch.Metric
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

//index:aws_config_aggregationauthorization
type ConfigAggregationAuthorizationDescription struct {
	AggregationAuthorization configservice.AggregationAuthorization
	Tags                     []configservice.Tag
}

//index:aws_config_conformancepack
//getfilter:name=description.ConformancePack.ConformancePackName
type ConfigConformancePackDescription struct {
	ConformancePack configservice.ConformancePackDetail
}

//index:aws_config_rule
//getfilter:name=description.Rule.ConfigRuleName
type ConfigRuleDescription struct {
	Rule       configservice.ConfigRule
	Compliance configservice.ComplianceByConfigRule
	Tags       []configservice.Tag
}

//  ===================   Dax   ===================

//index:aws_dax_cluster
//getfilter:cluster_name=description.Cluster.ClusterName
//listfilter:cluster_name=description.Cluster.ClusterName
type DAXClusterDescription struct {
	Cluster dax.Cluster
	Tags    []dax.Tag
}

//index:aws_dax_parametergroup
//listfilter:parameter_group_name=description.ParameterGroup.ParameterGroupName
type DAXParameterGroupDescription struct {
	ParameterGroup dax.ParameterGroup
}

//index:aws_dax_parameter
//listfilter:parameter_group_name=description.ParameterGroupName
type DAXParameterDescription struct {
	Parameter          dax.Parameter
	ParameterGroupName string
}

//index:aws_dax_subnetgroup
//listfilter:subnet_group_name=description.SubnetGroup.SubnetGroupName
type DAXSubnetGroupDescription struct {
	SubnetGroup dax.SubnetGroup
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

//index:aws_dynamodb_tableexport
//getfilter:arn=description.Export.ExportArn
//listfilter:arn=description.Export.ExportArn
type DynamoDbTableExportDescription struct {
	Export dynamodb.ExportDescription
}

//index:aws_dynamodb_metricaccountprovisionedreadcapacityutilization
type DynamoDBMetricAccountProvisionedReadCapacityUtilizationDescription struct {
	CloudWatchMetricRow
}

//index:aws_dynamodb_metricaccountprovisionedwritecapacityutilization
type DynamoDBMetricAccountProvisionedWriteCapacityUtilizationDescription struct {
	CloudWatchMetricRow
}

//  ===================   EC2   ===================

//index:aws_ec2_volumesnapshot
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

//index:aws_ec2_ebsvolumemetricreadops
type EbsVolumeMetricReadOpsDescription struct {
	CloudWatchMetricRow
}

//index:aws_ec2_ebsvolumemetricreadopsdaily
type EbsVolumeMetricReadOpsDailyDescription struct {
	CloudWatchMetricRow
}

//index:aws_ec2_ebsvolumemetricreadopshourly
type EbsVolumeMetricReadOpsHourlyDescription struct {
	CloudWatchMetricRow
}

//index:aws_ec2_ebsvolumemetricwriteops
type EbsVolumeMetricWriteOpsDescription struct {
	CloudWatchMetricRow
}

//index:aws_ec2_ebsvolumemetricwriteopsdaily
type EbsVolumeMetricWriteOpsDailyDescription struct {
	CloudWatchMetricRow
}

//index:aws_ec2_ebsvolumemetricwriteopshourly
type EbsVolumeMetricWriteOpsHourlyDescription struct {
	CloudWatchMetricRow
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

//index:aws_ec2_availabilityzone
//getfilter:name=description.AvailabilityZone.ZoneName
//getfilter:region_name=description.AvailabilityZone.RegionName
//listfilter:name=description.AvailabilityZone.ZoneName
//listfilter:zone_id=description.AvailabilityZone.ZoneId
type EC2AvailabilityZoneDescription struct {
	AvailabilityZone ec2.AvailabilityZone
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

//index:aws_ec2_transitgateway
//getfilter:transit_gateway_id=description.TransitGateway.TransitGatewayId
type EC2TransitGatewayDescription struct {
	TransitGateway ec2.TransitGateway
}

//index:aws_ec2_transitgatewayroutetable
//getfilter:transit_gateway_route_table_id=description.TransitGatewayRouteTable.TransitGatewayRouteTableId
type EC2TransitGatewayRouteTableDescription struct {
	TransitGatewayRouteTable ec2.TransitGatewayRouteTable
}

//index:aws_ec2_dhcpoptions
//getfilter:dhcp_options_id=description.DhcpOptions.DhcpOptionsId
type EC2DhcpOptionsDescription struct {
	DhcpOptions ec2.DhcpOptions
}

//index:aws_ec2_egressonlyinternetgateway
//getfilter:id=description.EgressOnlyInternetGateway.EgressOnlyInternetGatewayId
type EC2EgressOnlyInternetGatewayDescription struct {
	EgressOnlyInternetGateway ec2.EgressOnlyInternetGateway
}

//index:aws_ec2_vpcpeeringconnection
type EC2VpcPeeringConnectionDescription struct {
	VpcPeeringConnection ec2.VpcPeeringConnection
}

//index:aws_ec2_securitygrouprule
type EC2SecurityGroupRuleDescription struct {
	Group           ec2.SecurityGroup
	Permission      ec2.IpPermission
	IPRange         *ec2.IpRange
	Ipv6Range       *ec2.Ipv6Range
	UserIDGroupPair *ec2.UserIdGroupPair
	PrefixListId    *ec2.PrefixListId
	Type            string
}

//index:aws_ec2_ipampool
//getfilter:ipam_pool_id=description.IpamPool.IpamPoolId
type EC2IpamPoolDescription struct {
	IpamPool ec2.IpamPool
}

//index:aws_ec2_ipam
//getfilter:ipam_id=description.Ipam.IpamId
type EC2IpamDescription struct {
	Ipam ec2.Ipam
}

//index:aws_ec2_vpcendpointservice
//getfilter:service_name=description.VPCEndpoint.ServiceName
type EC2VPCEndpointServiceDescription struct {
	VpcEndpointService ec2.ServiceDetail
}

//index:aws_ec2_instanceavailability
//listfilter:instance_type=description.InstanceAvailability.InstanceType
type EC2InstanceAvailabilityDescription struct {
	InstanceAvailability ec2.InstanceTypeOffering
}

//index:aws_ec2_instancetype
//getfilter:instance_type=description.InstanceType.InstanceType
type EC2InstanceTypeDescription struct {
	InstanceType ec2.InstanceTypeInfo
}

//index:aws_ec2_managedprefixlist
//listfilter:name=description.ManagedPrefixList.PrefixListName
//listfilter:id=description.ManagedPrefixList.PrefixListId
//listfilter:owner_id=description.ManagedPrefixList.OwnerId
type EC2ManagedPrefixListDescription struct {
	ManagedPrefixList ec2.ManagedPrefixList
}

//index:aws_ec2_spotprice
//listfilter:availability_zone=description.SpotPrice.AvailabilityZone
//listfilter:instance_type=description.SpotPrice.InstanceType
//listfilter:product_description=description.SpotPrice.ProductDescription
type EC2SpotPriceDescription struct {
	SpotPrice ec2.SpotPrice
}

//index:aws_ec2_transitgatewayroute
//listfilter:prefix_list_id=description.TransitGatewayRoute.PrefixListId
//listfilter:state=description.TransitGatewayRoute.State
//listfilter:type=description.TransitGatewayRoute.Type
type EC2TransitGatewayRouteDescription struct {
	TransitGatewayRoute        ec2.TransitGatewayRoute
	TransitGatewayRouteTableId string
}

//index:aws_ec2_transitgatewayvpcattachment
//getfilter:transit_gateway_attachment_id=description.TransitGatewayAttachment.TransitGatewayAttachmentId
//listfilter:association_state=description.TransitGatewayAttachment.Association.State
//listfilter:association_transit_gateway_route_table_id=description.TransitGatewayAttachment.Association.TransitGatewayRouteTableId
//listfilter:resource_id=description.TransitGatewayAttachment.ResourceId
//listfilter:resource_owner_id=description.TransitGatewayAttachment.ResourceOwnerId
//listfilter:resource_type=description.TransitGatewayAttachment.ResourceType
//listfilter:state=description.TransitGatewayAttachment.State
//listfilter:transit_gateway_id=description.TransitGatewayAttachment.TransitGatewayId
//listfilter:transit_gateway_owner_id=description.TransitGatewayAttachment.TransitGatewayOwnerId
type EC2TransitGatewayAttachmentDescription struct {
	TransitGatewayAttachment ec2.TransitGatewayAttachment
}

//  ===================  Elastic Load Balancing  ===================

//index:aws_elasticloadbalancingv2_sslpolicy
//getfilter:name=description.SslPolicy.Name
//getfilter:region=metadata.Region
type ElasticLoadBalancingV2SslPolicyDescription struct {
	SslPolicy elasticloadbalancingv2.SslPolicy
}

//index:aws_elasticloadbalancingv2_targetgroup
//getfilter:target_group_arn=description.TargetGroup.TargetGroupArn
//listfilter:target_group_name=description.TargetGroup.TargetGroupName
type ElasticLoadBalancingV2TargetGroupDescription struct {
	TargetGroup elasticloadbalancingv2.TargetGroup
	Health      []elasticloadbalancingv2.TargetHealthDescription
	Tags        []elasticloadbalancingv2.Tag
}

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

//index:aws_elasticloadbalancingv2_rule
//getfilter:arn=description.Rule.RuleArn
type ElasticLoadBalancingV2RuleDescription struct {
	Rule elasticloadbalancingv2.Rule
}

//index:aws_elasticloadbalancingv2_applicationloadbalancermetricrequestcount
type ApplicationLoadBalancerMetricRequestCountDescription struct {
	CloudWatchMetricRow
}

//index:aws_elasticloadbalancingv2_applicationloadbalancermetricrequestcountdaily
type ApplicationLoadBalancerMetricRequestCountDailyDescription struct {
	CloudWatchMetricRow
}

//index:aws_elasticloadbalancingv2_networkloadbalancermetricnetflowcount
type NetworkLoadBalancerMetricNetFlowCountDescription struct {
	CloudWatchMetricRow
}

//index:aws_elasticloadbalancingv2_networkloadbalancermetricnetflowcountdaily
type NetworkLoadBalancerMetricNetFlowCountDailyDescription struct {
	CloudWatchMetricRow
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

//index:aws_cloudtrail_channel
//getfilter:arn=description.Channel.ChannelArn
type CloudTrailChannelDescription struct {
	Channel cloudtrail.GetChannelOutput
}

//index:aws_cloudtrail_eventdatastore
//getfilter:arn=description.EventDataStore.EventDataStoreArn
type CloudTrailEventDataStoreDescription struct {
	EventDataStore cloudtrail.GetEventDataStoreOutput
}

//index:aws_cloudtrail_import
//getfilter:import_id=description.Import.ImportId
//listfilter:import_status=description.Import.ImportStatus
type CloudTrailImportDescription struct {
	Import cloudtrail.GetImportOutput
}

//index:aws_cloudtrail_query
//getfilter:event_data_store_arn=description.EventDataStoreARN
//getfilter:query_id=description.Query.QueryId
//listfilter:event_data_store_arn=description.EventDataStoreARN
//listfilter:query_status=description.Query.QueryStatus
//listfilter:creation_time=description.Query.QueryStatistics.CreationTime
type CloudTrailQueryDescription struct {
	Query             cloudtrail.DescribeQueryOutput
	EventDataStoreARN string
}

//index:aws_cloudtrail_trailevent
//listfilter:log_stream_name=description.TrailEvent.LogStreamName
//listfilter:timestamp=description.TrailEvent.Timestamp
type CloudTrailTrailEventDescription struct {
	TrailEvent   cloudwatchlogs.FilteredLogEvent
	LogGroupName string
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

//index:aws_iam_policyattachment
//getfilter:is_attached=description.IsAttached
type IAMPolicyAttachmentDescription struct {
	PolicyArn             string
	PolicyAttachmentCount int32
	IsAttached            bool
	PolicyGroups          []iam.PolicyGroup
	PolicyRoles           []iam.PolicyRole
	PolicyUsers           []iam.PolicyUser
}

//index:aws_iam_samlprovider
//getfilter:arn=ARN
type IAMSamlProviderDescription struct {
	SamlProvider iamop.GetSAMLProviderOutput
}

//index:aws_iam_servicespecificcredential
//listfilter:service_name=description.ServiceSpecificCredential.ServiceName
//listfilter:user_name=description.ServiceSpecificCredential.UserName
type IAMServiceSpecificCredentialDescription struct {
	ServiceSpecificCredential iam.ServiceSpecificCredentialMetadata
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

//index:aws_ecs_task
//listfilter:container_instance_arn=description.Task.ContainerInstanceArn
//listfilter:desired_status=description.Task.DesiredStatus
//listfilter:launch_type=description.Task.LaunchType
//listfilter:service_name=description.ServiceName
type ECSTaskDescription struct {
	Task           ecs.Task
	TaskProtection *ecs.ProtectedTask
	ServiceName    string
}

//  ===================  EFS  ===================

//index:aws_efs_filesystem
//getfilter:aws_efs_file_system=description.FileSystem.FileSystemId
type EFSFileSystemDescription struct {
	FileSystem efs.FileSystemDescription
	Policy     *string
}

//index:aws_efs_accesspoint
//getfilter:access_point_id=description.AccessPoint.AccessPointId
//listfilter:file_system_id=description.AccessPoint.FileSystemId
type EFSAccessPointDescription struct {
	AccessPoint efs.AccessPointDescription
}

//index:aws_efs_mounttarget
//getfilter:mount_target_id=description.MountTarget.MountTargetId
type EFSMountTargetDescription struct {
	MountTarget    efs.MountTargetDescription
	SecurityGroups []string
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
//getfilter:cluster_name=description.Nodegroup.ClusterName
//listfilter:cluster_name=description.Nodegroup.ClusterName
type EKSNodegroupDescription struct {
	Nodegroup eks.Nodegroup
}

//index:aws_eks_addonversion
//listfilter:addon_name=description.AddonName
type EKSAddonVersionDescription struct {
	AddonVersion       eks.AddonVersionInfo
	AddonConfiguration string
	AddonName          string
	AddonType          string
}

//index:aws_eks_fargateprofile
//getfilter:cluster_name=description.Fargate.ClusterName
//getfilter:fargate_profile_name=description.Fargate.FargateProfileName
//listfilter:cluster_name=description.Fargate.ClusterName
type EKSFargateProfileDescription struct {
	FargateProfile eks.FargateProfile
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

//index:aws_lambda_function_version
//getfilter:id=description.ID
type LambdaFunctionVersionDescription struct {
	ID              string
	FunctionVersion lambdatypes.FunctionConfiguration
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

	MeanValue *string
}

//index:aws_costexplorer_byaccountmonthly
type CostExplorerByAccountMonthlyDescription struct {
	CostExplorerRow
}

//index:aws_costexplorer_byservicemonthly
type CostExplorerByServiceMonthlyDescription struct {
	CostExplorerRow
}

//index:aws_costexplorer_byrecordtypemonthly
type CostExplorerByRecordTypeMonthlyDescription struct {
	CostExplorerRow
}

//index:aws_costexplorer_byusagetypemonthly
type CostExplorerByServiceUsageTypeMonthlyDescription struct {
	CostExplorerRow
}

//index:aws_costexplorer_forcastmonthly
type CostExplorerForcastMonthlyDescription struct {
	CostExplorerRow
}

//index:aws_costexplorer_byaccountdaily
type CostExplorerByAccountDailyDescription struct {
	CostExplorerRow
}

//index:aws_costexplorer_byservicedaily
type CostExplorerByServiceDailyDescription struct {
	CostExplorerRow
}

//index:aws_costexplorer_byrecordtypedaily
type CostExplorerByRecordTypeDailyDescription struct {
	CostExplorerRow
}

//index:aws_costexplorer_byusagetypedaily
type CostExplorerByServiceUsageTypeDailyDescription struct {
	CostExplorerRow
}

//index:aws_costexplorer_forcastdaily
type CostExplorerForcastDailyDescription struct {
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

//index:aws_ecr_image
//listfilter:repository_name=description.Image.RepositoryName
//listfilter:registry_id=description.Image.RegistryId
type ECRImageDescription struct {
	Image ecr.ImageDetail
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

//index:aws_eventbridge_eventbus
//getfilter:arn=description.Bus.Arn
type EventBridgeBusDescription struct {
	Bus  eventbridge.EventBus
	Tags []eventbridge.Tag
}

//index:aws_eventbridge_eventrule
//getfilter:name=description.Rule.Name
//listfilter:event_bus_name=description.Rule.EventBusName
//listfilter:name_prefix=description.Rule.Name
type EventBridgeRuleDescription struct {
	Rule    eventbridgeop.DescribeRuleOutput
	Targets []eventbridge.Target
	Tags    []eventbridge.Tag
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

//index:aws_kinesisanalyticsv2_application
//getfilter:application_name=description.Application.ApplicationName
type KinesisAnalyticsV2ApplicationDescription struct {
	Application kinesisanalyticsv2.ApplicationDetail
	Tags        []kinesisanalyticsv2.Tag
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
//getfilter:db_instance_identifier=description.Database.DBInstanceIdentifier
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

//  ===================  CloudFormation  ===================

//index:aws_cloudformation_stack
//getfilter:name=description.Stack.StackName
//listfilter:name=description.Stack.StackName
type CloudFormationStackDescription struct {
	Stack          cloudformation.Stack
	StackTemplate  cloudformationop.GetTemplateOutput
	StackResources []cloudformation.StackResource
}

//index:aws_cloudformation_stackset
//getfilter:stack_set_name=description.StackSet.StackSetName
type CloudFormationStackSetDescription struct {
	StackSet cloudformation.StackSet
}

//  ===================  CodeCommit  ===================

//index:aws_codecommit_repository
type CodeCommitRepositoryDescription struct {
	Repository codecommit.RepositoryMetadata
	Tags       map[string]string
}

//  ===================  CodePipeline  ===================

//index:aws_codepipeline_pipeline
//getfilter:name=description.Pipeline.Name
type CodePipelinePipelineDescription struct {
	Pipeline codepipeline.PipelineDeclaration
	Metadata codepipeline.PipelineMetadata
	Tags     []codepipeline.Tag
}

//  ===================  DirectoryService  ===================

//index:aws_directoryservice_directory
//getfilter:name=description.Directory.DirectoryId
type DirectoryServiceDirectoryDescription struct {
	Directory directoryservice.DirectoryDescription
	Tags      []directoryservice.Tag
}

//  ===================  SSOAdmin  ===================

//index:aws_ssoadmin_instance
type SSOAdminInstanceDescription struct {
	Instance ssoadmin.InstanceMetadata
}

//  ===================  WAF  ===================

//index:aws_waf_rule
//getfilter:rule_id=description.Rule.RuleId
type WAFRuleDescription struct {
	Rule waf.Rule
	Tags []waf.Tag
}

//index:aws_wafregional_rule
//getfilter:rule_id=description.Rule.RuleId
type WAFRegionalRuleDescription struct {
	Rule wafregional.RuleSummary
	Tags []wafregional.Tag
}

//  ===================  Route53  ===================

//index:aws_route53_hostedzone
//getfilter:id=description.ID
type Route53HostedZoneDescription struct {
	ID                  string
	HostedZone          route53.HostedZone
	QueryLoggingConfigs []route53.QueryLoggingConfig
	DNSSec              route53op.GetDNSSECOutput
	Tags                []route53.Tag
}

//  ===================  Batch  ===================

//index:aws_batch_computeenvironment
//getfilter:compute_environment_name=description.ComputeEnvironment.ComputeEnvironmentName
type BatchComputeEnvironmentDescription struct {
	ComputeEnvironment batch.ComputeEnvironmentDetail
}

//index:aws_batch_job
//getfilter:job_name=description.Job.JobName
type BatchJobDescription struct {
	Job batch.JobSummary
}

//  ===================  CodeArtifact  ===================

//index:aws_codeartifact_repository
//getfilter:name=description.Repository.Name
type CodeArtifactRepositoryDescription struct {
	Repository codeartifact.RepositorySummary
	Tags       []codeartifact.Tag
}

//index:aws_codeartifact_domain
//getfilter:name=description.Domain.Name
//getfilter:name=description.Domain.Owner
type CodeArtifactDomainDescription struct {
	Domain codeartifact.DomainDescription
	Policy codeartifact.ResourcePolicy
	Tags   []codeartifact.Tag
}

//  ===================  CodeDeploy  ===================

//index:aws_codedeploy_deploymentgroup
//getfilter:deployment_group_name=description.DeploymentGroup.DeploymentGroupName
type CodeDeployDeploymentGroupDescription struct {
	DeploymentGroup codedeploy.DeploymentGroupInfo
	Tags            []codedeploy.Tag
}

//index:aws_codedeploy_application
//getfilter:application_name=description.Application.ApplicationName
type CodeDeployApplicationDescription struct {
	Application codedeploy.ApplicationInfo
	Tags        []codedeploy.Tag
}

//  ===================  CodeStar  ===================

//index:aws_codestar_project
//getfilter:id=description.Project.Id
type CodeStarProjectDescription struct {
	Project codestarop.DescribeProjectOutput
	Tags    map[string]string
}

//  ===================  DirectConnect  ===================

//index:aws_directconnect_connection
//getfilter:connection_id=description.Connection.ConnectionId
type DirectConnectConnectionDescription struct {
	Connection directconnect.Connection
}

//index:aws_directconnect_gateway
//getfilter:direct_connect_gateway_id=description.Gateway.DirectConnectGatewayId
type DirectConnectGatewayDescription struct {
	Gateway directconnect.DirectConnectGateway
	Tags    []directconnect.Tag
}

//  ===================  Elastic Disaster Recovery (DRS)  ===================

//index:aws_drs_sourceserver
//getfilter:source_server_id=description.SourceServer.SourceServerID
type DRSSourceServerDescription struct {
	SourceServer drs.SourceServer
}

//index:aws_drs_recoveryinstance
//getfilter:recovery_instance_id=description.RecoveryInstance.RecoveryInstanceID
type DRSRecoveryInstanceDescription struct {
	RecoveryInstance drs.RecoveryInstance
}

//index:aws_drs_job
//listfilter:job_id=description.Job.JobID
//listfilter:creation_date_time=description.Job.CreationDateTime
//listfilter:end_date_time=description.Job.EndDateTime
type DRSJobDescription struct {
	Job drs.Job
}

//index:aws_drs_recoverysnapshot
//listfilter:source_server_id=description.RecoveryInstance.SourceServerID
//listfilter:timestamp=description.RecoveryInstance.Timestamp
type DRSRecoverySnapshotDescription struct {
	RecoverySnapshot drs.RecoverySnapshot
}

//  ===================  Firewall Manager Policy (FMS)  ===================

//index:aws_fms_policy
//getfilter:policy_name=description.Policy.PolicyName
type FMSPolicyDescription struct {
	Policy fms.PolicySummary
	Tags   []fms.Tag
}

//  ===================  Network Firewall ===================

//index:aws_networkfirewall_firewall
//getfilter:firewall_name=description.Firewall.FirewallName
type NetworkFirewallFirewallDescription struct {
	Firewall networkfirewall.Firewall
}

//  ===================  OpsWork ===================

//index:aws_opsworkscm_server
//getfilter:server_name=description.Server.ServerName
type OpsWorksCMServerDescription struct {
	Server opsworkscm.Server
	Tags   []opsworkscm.Tag
}

//  ===================  Organizations ===================

//index:aws_organizations_organization
//getfilter:id=description.Organization.Id
type OrganizationsOrganizationDescription struct {
	Organization organizations.Organization
}

//  ===================  ACM ===================

//index:aws_acmpca_certificateauthority
//getfilter:arn=description.CertificateAuthority.Arn
type ACMPCACertificateAuthorityDescription struct {
	CertificateAuthority acmpca.CertificateAuthority
	Tags                 []acmpca.Tag
}

//  ===================  Shield ===================

//index:aws_shield_protectiongroup
//getfilter:protection_group_id=description.ProtectionGroup.ProtectionGroupId
type ShieldProtectionGroupDescription struct {
	ProtectionGroup shield.ProtectionGroup
	Tags            []shield.Tag
}

//  ===================  Storage Gateway ===================

//index:aws_storagegateway_storagegateway
//getfilter:gateway_id=description.StorageGateway.GatewayId
type StorageGatewayStorageGatewayDescription struct {
	StorageGateway storagegateway.GatewayInfo
	Tags           []storagegateway.Tag
}

//  ===================  Image Builder ===================

//index:aws_imagebuilder_image
//getfilter:name=description.Image.Name
type ImageBuilderImageDescription struct {
	Image imagebuilder.Image
}

// ===================  Account ===================

//index:aws_account_alternatecontact
//listfilter:linked_account_id=description.LinkedAccountID
//listfilter:contact_type=description.AlternateContact.AlternateContactType
type AccountAlternateContactDescription struct {
	AlternateContact account.AlternateContact
	LinkedAccountID  string
}

//index:aws_account_contact
//listfilter:linked_account_id=description.LinkedAccountID
type AccountContactDescription struct {
	AlternateContact account.ContactInformation
	LinkedAccountID  string
}

// ===================  Amplify ===================

//index:aws_amplify_app
//getfilter:app_id=description.App.AppId
type AmplifyAppDescription struct {
	App amplify.App
}

// ===================  App Config (appconfig) ===================

//index:aws_appconfig_application
//getfilter:id=description.Application.Id
type AppConfigApplicationDescription struct {
	Application appconfig.Application
	Tags        map[string]string
}

// ===================  Audit Manager ===================

//index:aws_auditmanager_assessment
//getfilter:assessment_id=description.Assessment.Metadata.Id
type AuditManagerAssessmentDescription struct {
	Assessment auditmanager.Assessment
}

//index:aws_auditmanager_control
//getfilter:control_id=description.Control.Id
type AuditManagerControlDescription struct {
	Control auditmanager.Control
}

//index:aws_auditmanager_evidence
//getfilter:id=description.Evidence.Id
//getfilter:evidence_folder_id=description.Evidence.EvidenceFolderId
//getfilter:assessment_id=description.AssessmentID
//getfilter:control_set_id=description.ControlSetID
type AuditManagerEvidenceDescription struct {
	Evidence     auditmanager.Evidence
	ControlSetID string
	AssessmentID string
}

//index:aws_auditmanager_evidencefolder
//getfilter:id=description.EvidenceFolder.Id
//getfilter:assessment_id=description.AssessmentID
//getfilter:control_set_id=description.ControlSetID
type AuditManagerEvidenceFolderDescription struct {
	EvidenceFolder auditmanager.AssessmentEvidenceFolder
	AssessmentID   string
}

//index:aws_auditmanager_framework
//getfilter:id=description.Framework.Id
//getfilter:region=metadata.Region
type AuditManagerFrameworkDescription struct {
	Framework auditmanager.Framework
}

// ===================  CloudControl ===================

//index:aws_cloudcontrol_resource
//getfilter:identifier=description.Resource.Identifier
type CloudControlResourceDescription struct {
	Resource cloudcontrol.ResourceDescription
}

// ===================  CloudSearch ===================

//index:aws_cloudsearch_domain
//getfilter:domain_name=description.DomainStatus.DomainName
type CloudSearchDomainDescription struct {
	DomainStatus cloudsearch.DomainStatus
}

// ===================  DLM ===================

//index:aws_dlm_lifecyclepolicy
//getfilter:id=description.LifecyclePolicy.PolicyId
type DLMLifecyclePolicyDescription struct {
	LifecyclePolicy dlm.LifecyclePolicy
}

// ===================  DocDB ===================

//index:aws_docdb_cluster
//getfilter:db_cluster_identifier=description.DBCluster.DBClusterIdentifier
type DocDBClusterDescription struct {
	DBCluster docdb.DBCluster
	Tags      []docdb.Tag
}

// ===================  Global Accelerator ===================

//index:aws_globalaccelerator_accelerator
//getfilter:arn=description.Accelerator.AcceleratorArn
type GlobalAcceleratorAcceleratorDescription struct {
	Accelerator           globalaccelerator.Accelerator
	AcceleratorAttributes *globalaccelerator.AcceleratorAttributes
	Tags                  []globalaccelerator.Tag
}

//index:aws_globalaccelerator_endpointgroup
//getfilter:arn=description.EndpointGroup.EndpointGroupArn
//listfilter:listener_arn=description.ListenerArn
type GlobalAcceleratorEndpointGroupDescription struct {
	EndpointGroup  globalaccelerator.EndpointGroup
	ListenerArn    string
	AcceleratorArn string
}

//index:aws_globalaccelerator_listener
//getfilter:arn=description.Listener.ListenerArn
//listfilter:accelerator_arn=description.AcceleratorArn
type GlobalAcceleratorListenerDescription struct {
	Listener       globalaccelerator.Listener
	AcceleratorArn string
}

// ===================  Glue ===================

//index:aws_glue_catalogdatabase
//getfilter:name=description.Database.Name
type GlueCatalogDatabaseDescription struct {
	Database glue.Database
}

//index:aws_glue_catalogtable
//getfilter:name=description.Table.Name
//getfilter:database_name=description.DatabaseName
//listfilter:catalog_id=description.Table.CatalogId
//listfilter:database_name=description.Table.DatabaseName
type GlueCatalogTableDescription struct {
	Table glue.Table
}

//index:aws_glue_connection
//getfilter:name=description.Connection.Name
//listfilter:connection_type=description.Connection.ConnectionType
type GlueConnectionDescription struct {
	Connection glue.Connection
}

//index:aws_glue_crawler
//getfilter:name=description.Crawler.Name
type GlueCrawlerDescription struct {
	Crawler glue.Crawler
}

//index:aws_glue_datacatalogencryptionsettings
type GlueDataCatalogEncryptionSettingsDescription struct {
	DataCatalogEncryptionSettings glue.DataCatalogEncryptionSettings
}

//index:aws_glue_dataqualityruleset
//getfilter:name=description.DataQualityRuleset.Name
//listfilter:created_on=description.DataQualityRuleset.CreatedOn
//listfilter:last_modified_on=description.DataQualityRuleset.LastModifiedOn
type GlueDataQualityRulesetDescription struct {
	DataQualityRuleset glueop.GetDataQualityRulesetOutput
}

//index:aws_glue_devendpoint
//getfilter:endpoint_name=description.DevEndpoint.EndpointName
type GlueDevEndpointDescription struct {
	DevEndpoint glue.DevEndpoint
}

//index:aws_glue_job
//getfilter:name=description.Job.Name
type GlueJobDescription struct {
	Job      glue.Job
	Bookmark glue.JobBookmarkEntry
}

//index:aws_glue_securityconfiguration
//getfilter:name=description.SecurityConfiguration.Name
type GlueSecurityConfigurationDescription struct {
	SecurityConfiguration glue.SecurityConfiguration
}

// ===================  Health ===================

//index:aws_health_event
//listfilter:arn=description.Event.Arn
//listfilter:availability_zone=description.Event.AvailabilityZone
//listfilter:end_time=description.Event.EndTime
//listfilter:event_type_category=description.Event.EventTypeCategory
//listfilter:event_type_code=description.Event.EventTypeCode
//listfilter:last_updated_time=description.Event.LastUpdatedTime
//listfilter:service=description.Event.Service
//listfilter:start_time=description.Event.StartTime
//listfilter:status_code=description.Event.StatusCode
type HealthEventDescription struct {
	Event health.Event
}

// ===================  Identity Store ===================

//index:aws_identitystore_group
//getfilter:id=description.Group.GroupId
//getfilter:identity_store_id=description.Group.IdentityStoreId
//listfilter:identity_store_id=description.Group.IdentityStoreId
type IdentityStoreGroupDescription struct {
	Group identitystore.Group
}

//index:aws_identitystore_user
//getfilter:id=description.User.UserId
//getfilter:identity_store_id=description.User.IdentityStoreId
//listfilter:identity_store_id=description.User.IdentityStoreId
type IdentityStoreUserDescription struct {
	User identitystore.User
}

// ===================  Inspector ===================

//index:aws_inspector_assessmentrun
//listfilter:assessment_template_arn=description.AssessmentRun.AssessmentTemplateArn
//listfilter:name=description.AssessmentRun.Name
//listfilter:state=description.AssessmentRun.State
type InspectorAssessmentRunDescription struct {
	AssessmentRun inspector.AssessmentRun
}

//index:aws_inspector_assessmenttarget
//getfilter:arn=description.AssessmentTarget.Arn
type InspectorAssessmentTargetDescription struct {
	AssessmentTarget inspector.AssessmentTarget
}

//index:aws_inspector_assessmenttemplate
//getfilter:arn=description.AssessmentTemplate.Arn
//listfilter:name=description.AssessmentTemplate.Name
//listfilter:assessment_target_arn=description.AssessmentTemplate.AssessmentTargetArn
type InspectorAssessmentTemplateDescription struct {
	AssessmentTemplate inspector.AssessmentTemplate
	EventSubscriptions []inspector.Subscription
	Tags               []inspector.Tag
}

//index:aws_inspector_exclusion
//listfilter:assessment_run_arn=description.Exclusion.Arn
type InspectorExclusionDescription struct {
	Exclusion        inspector.Exclusion
	AssessmentRunArn string
}

//index:aws_inspector_finding
//listfilter:agent_id=description.Finding.AssetAttributes.AgentId
//listfilter:auto_scaling_group=description.Finding.AssetAttributes.AutoScalingGroup
//listfilter:severity=description.Finding.Severity
//getfilter:arn=description.Finding.Arn
type InspectorFindingDescription struct {
	Finding     inspector.Finding
	FailedItems map[string]inspector.FailedItemDetails
}
