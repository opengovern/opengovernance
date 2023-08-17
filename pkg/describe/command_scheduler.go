package describe

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
)

const (
	DescribeResultsQueueName             = "describe-results-queue"
	ComplianceReportJobsQueueName        = "compliance-report-jobs-queue"
	ComplianceReportResultsQueueName     = "compliance-report-results-queue"
	ComplianceReportCleanupJobsQueueName = "compliance-report-cleanup-jobs-queue"
	InsightJobsQueueName                 = "insight-jobs-queue"
	InsightResultsQueueName              = "insight-results-queue"
	CheckupJobsQueueName                 = "checkup-jobs-queue"
	CheckupResultsQueueName              = "checkup-results-queue"
	SummarizerJobsQueueName              = "summarizer-jobs-queue"
	SummarizerResultsQueueName           = "summarizer-results-queue"
	AnalyticsJobsQueueName               = "analytics-jobs-queue"
	AnalyticsResultsQueueName            = "analytics-results-queue"
	SourceEventsQueueName                = "source-events-queue"
)

var (
	RabbitMQService  = os.Getenv("RABBITMQ_SERVICE")
	RabbitMQPort     = 5672
	RabbitMQUsername = os.Getenv("RABBITMQ_USERNAME")
	RabbitMQPassword = os.Getenv("RABBITMQ_PASSWORD")

	KafkaService        = os.Getenv("KAFKA_SERVICE")
	KafkaResourcesTopic = os.Getenv("KAFKA_RESOURCE_TOPIC")

	PostgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	PostgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	PostgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	PostgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	PostgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")
	PostgreSQLSSLMode  = os.Getenv("POSTGRESQL_SSLMODE")

	ElasticSearchAddress  = os.Getenv("ES_ADDRESS")
	ElasticSearchUsername = os.Getenv("ES_USERNAME")
	ElasticSearchPassword = os.Getenv("ES_PASSWORD")

	HttpServerAddress = os.Getenv("HTTP_ADDRESS")
	GRPCServerAddress = os.Getenv("GRPC_ADDRESS")

	RedisAddress = os.Getenv("REDIS_ADDRESS")

	DescribeIntervalHours      = os.Getenv("DESCRIBE_INTERVAL_HOURS")
	FullDiscoveryIntervalHours = os.Getenv("FULL_DISCOVERY_INTERVAL_HOURS")
	DescribeTimeoutHours       = os.Getenv("DESCRIBE_TIMEOUT_HOURS")
	ComplianceIntervalHours    = os.Getenv("COMPLIANCE_INTERVAL_HOURS")
	ComplianceTimeoutHours     = os.Getenv("COMPLIANCE_TIMEOUT_HOURS")
	InsightIntervalHours       = os.Getenv("INSIGHT_INTERVAL_HOURS")
	CheckupIntervalHours       = os.Getenv("CHECKUP_INTERVAL_HOURS")
	MustSummarizeIntervalHours = os.Getenv("MUST_SUMMARIZE_INTERVAL_HOURS")
	AnalyticsIntervalHours     = os.Getenv("ANALYTICS_INTERVAL_HOURS")
	CurrentWorkspaceID         = os.Getenv("CURRENT_NAMESPACE")
	WorkspaceBaseURL           = os.Getenv("WORKSPACE_BASE_URL")
	MetadataBaseURL            = os.Getenv("METADATA_BASE_URL")
	ComplianceBaseURL          = os.Getenv("COMPLIANCE_BASE_URL")
	OnboardBaseURL             = os.Getenv("ONBOARD_BASE_URL")
	AuthGRPCURI                = os.Getenv("AUTH_GRPC_URI")

	LambdaFuncsBaseURL      = os.Getenv("LAMBDA_FUNCS_BASE_URL")
	KeyARN                  = os.Getenv("KMS_KEY_ARN")
	KeyRegion               = os.Getenv("KMS_ACCOUNT_REGION")
	DescribeDeliverEndpoint = os.Getenv("DESCRIBE_DELIVER_ENDPOINT")

	DoDeleteOldResources  = os.Getenv("DO_DELETE_OLD_RESOURCES")
	OperationModeConfig   = os.Getenv("OPERATION_MODE_CONFIG")
	DoProcessReceivedMsgs = os.Getenv("DO_PROCESS_RECEIVED_MSGS")

	MaxConcurrentCall = os.Getenv("MAX_CONCURRENT_CALL")

	KaytuHelmChartLocation = os.Getenv("KAYTU_STACK_HELM_CHART_LOCATION")
	FluxSystemNamespace    = os.Getenv("FLUX_SYSTEM_NAMESPACE")
)

func SchedulerCommand() *cobra.Command {
	var (
		id string
	)
	cmd := &cobra.Command{
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case id == "":
				return errors.New("missing required flag 'id'")
			default:
				return nil
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := InitializeScheduler(
				id,
				DescribeResultsQueueName,
				ComplianceReportJobsQueueName,
				ComplianceReportResultsQueueName,
				ComplianceReportCleanupJobsQueueName,
				InsightJobsQueueName,
				InsightResultsQueueName,
				CheckupJobsQueueName,
				CheckupResultsQueueName,
				SummarizerJobsQueueName,
				SummarizerResultsQueueName,
				AnalyticsJobsQueueName,
				AnalyticsResultsQueueName,
				SourceEventsQueueName,
				PostgreSQLUser,
				PostgreSQLPassword,
				PostgreSQLHost,
				PostgreSQLPort,
				PostgreSQLDb,
				PostgreSQLSSLMode,
				HttpServerAddress,
				DescribeIntervalHours,
				FullDiscoveryIntervalHours,
				DescribeTimeoutHours,
				ComplianceIntervalHours,
				ComplianceTimeoutHours,
				InsightIntervalHours,
				CheckupIntervalHours,
				MustSummarizeIntervalHours,
				AnalyticsIntervalHours,
				KaytuHelmChartLocation,
				FluxSystemNamespace,
			)
			if err != nil {
				return err
			}

			defer s.Stop()

			return s.Run()
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "The scheduler id")

	return cmd
}
