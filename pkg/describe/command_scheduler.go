package describe

import (
	"errors"
	"github.com/spf13/cobra"
	"os"
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

	HttpServerAddress       = os.Getenv("HTTP_ADDRESS")
	GRPCServerAddress       = os.Getenv("GRPC_ADDRESS")
	DescribeDeliverEndpoint = os.Getenv("DESCRIBE_DELIVER_ENDPOINT")

	RedisAddress = os.Getenv("REDIS_ADDRESS")

	DescribeIntervalHours      = os.Getenv("DESCRIBE_INTERVAL_HOURS")
	DescribeTimeoutHours       = os.Getenv("DESCRIBE_TIMEOUT_HOURS")
	ComplianceIntervalHours    = os.Getenv("COMPLIANCE_INTERVAL_HOURS")
	ComplianceTimeoutHours     = os.Getenv("COMPLIANCE_TIMEOUT_HOURS")
	InsightIntervalHours       = os.Getenv("INSIGHT_INTERVAL_HOURS")
	CheckupIntervalHours       = os.Getenv("CHECKUP_INTERVAL_HOURS")
	MustSummarizeIntervalHours = os.Getenv("MUST_SUMMARIZE_INTERVAL_HOURS")
	CurrentWorkspaceID         = os.Getenv("CURRENT_NAMESPACE")
	WorkspaceBaseURL           = os.Getenv("WORKSPACE_BASE_URL")
	MetadataBaseURL            = os.Getenv("METADATA_BASE_URL")
	ComplianceBaseURL          = os.Getenv("COMPLIANCE_BASE_URL")
	OnboardBaseURL             = os.Getenv("ONBOARD_BASE_URL")

	LambdaFuncURL = os.Getenv("LAMBDA_FUNC_URL")
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
				SourceEventsQueueName,
				PostgreSQLUser,
				PostgreSQLPassword,
				PostgreSQLHost,
				PostgreSQLPort,
				PostgreSQLDb,
				PostgreSQLSSLMode,
				HttpServerAddress,
				DescribeIntervalHours,
				DescribeTimeoutHours,
				ComplianceIntervalHours,
				ComplianceTimeoutHours,
				InsightIntervalHours,
				CheckupIntervalHours,
				MustSummarizeIntervalHours,
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
