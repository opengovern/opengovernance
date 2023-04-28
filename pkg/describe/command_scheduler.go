package describe

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	DescribeJobsQueueName                = "describe-jobs-queue"
	DescribeResultsQueueName             = "describe-results-queue"
	DescribeCleanupJobsQueueName         = "describe-cleanup-jobs-queue"
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
	DescribeConnectionJobsQueueName      = "describe-connection-jobs-queue"
	DescribeConnectionResultsQueueName   = "describe-connection-results-queue"
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

	VaultAddress  = os.Getenv("VAULT_ADDRESS")
	VaultToken    = os.Getenv("VAULT_TOKEN")
	VaultRoleName = os.Getenv("VAULT_ROLE")
	VaultCaPath   = os.Getenv("VAULT_TLS_CA_PATH")
	VaultUseTLS   = strings.ToLower(strings.TrimSpace(os.Getenv("VAULT_USE_TLS"))) == "true"

	ElasticSearchAddress  = os.Getenv("ES_ADDRESS")
	ElasticSearchUsername = os.Getenv("ES_USERNAME")
	ElasticSearchPassword = os.Getenv("ES_PASSWORD")

	HttpServerAddress       = os.Getenv("HTTP_ADDRESS")
	GRPCServerAddress       = os.Getenv("GRPC_ADDRESS")
	DescribeDeliverEndpoint = os.Getenv("DESCRIBE_DELIVER_ENDPOINT")

	PrometheusPushAddress = os.Getenv("PROMETHEUS_PUSH_ADDRESS")

	RedisAddress  = os.Getenv("REDIS_ADDRESS")
	CacheAddress  = os.Getenv("CACHE_ADDRESS")
	JaegerAddress = os.Getenv("JAEGER_ADDRESS")

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
	IngressBaseURL             = os.Getenv("BASE_URL")

	CloudNativeAPIBaseURL = os.Getenv("CLOUD_NATIVE_API_BASE_URL")
	CloudNativeAPIAuthKey = os.Getenv("CLOUD_NATIVE_API_AUTH_KEY")

	LambdaFuncURL = os.Getenv("LAMBDA_FUNC_URL")

	// For cloud native connection job command
	AccountConcurrentDescribe = os.Getenv("ACCOUNT_CONCURRENT_DESCRIBE")
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
				DescribeJobsQueueName,
				DescribeResultsQueueName,
				DescribeConnectionResultsQueueName,
				CloudNativeAPIBaseURL,
				CloudNativeAPIAuthKey,
				DescribeCleanupJobsQueueName,
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
				VaultAddress,
				VaultRoleName,
				VaultToken,
				VaultCaPath,
				VaultUseTLS,
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
