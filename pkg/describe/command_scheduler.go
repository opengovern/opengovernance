package describe

import (
	"context"
	"errors"
	"os"

	config2 "github.com/kaytu-io/kaytu-engine/pkg/describe/config"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
)

const (
	InsightJobsQueueName    = "insight-jobs-queue"
	InsightResultsQueueName = "insight-results-queue"
	CheckupJobsQueueName    = "checkup-jobs-queue"
	CheckupResultsQueueName = "checkup-results-queue"
	SourceEventsQueueName   = "source-events-queue"
)

var (
	RabbitMQService  = os.Getenv("RABBITMQ_SERVICE")
	RabbitMQPort     = 5672
	RabbitMQUsername = os.Getenv("RABBITMQ_USERNAME")
	RabbitMQPassword = os.Getenv("RABBITMQ_PASSWORD")

	PostgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	PostgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	PostgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	PostgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	PostgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")
	PostgreSQLSSLMode  = os.Getenv("POSTGRESQL_SSLMODE")

	HttpServerAddress = os.Getenv("HTTP_ADDRESS")
	GRPCServerAddress = os.Getenv("GRPC_ADDRESS")

	DescribeIntervalHours      = os.Getenv("DESCRIBE_INTERVAL_HOURS")
	FullDiscoveryIntervalHours = os.Getenv("FULL_DISCOVERY_INTERVAL_HOURS")
	CostDiscoveryIntervalHours = os.Getenv("COST_DISCOVERY_INTERVAL_HOURS")
	DescribeTimeoutHours       = os.Getenv("DESCRIBE_TIMEOUT_HOURS")
	InsightIntervalHours       = os.Getenv("INSIGHT_INTERVAL_HOURS")
	CheckupIntervalHours       = os.Getenv("CHECKUP_INTERVAL_HOURS")
	MustSummarizeIntervalHours = os.Getenv("MUST_SUMMARIZE_INTERVAL_HOURS")
	AnalyticsIntervalHours     = os.Getenv("ANALYTICS_INTERVAL_HOURS")
	CurrentWorkspaceID         = os.Getenv("CURRENT_NAMESPACE")
	WorkspaceBaseURL           = os.Getenv("WORKSPACE_BASE_URL")
	MetadataBaseURL            = os.Getenv("METADATA_BASE_URL")
	ComplianceBaseURL          = os.Getenv("COMPLIANCE_BASE_URL")
	OnboardBaseURL             = os.Getenv("ONBOARD_BASE_URL")
	InventoryBaseURL           = os.Getenv("INVENTORY_BASE_URL")
	AuthGRPCURI                = os.Getenv("AUTH_GRPC_URI")

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
	var id string
	var conf config2.SchedulerConfig
	config.ReadFromEnv(&conf, nil)

	ctx := context.Background()

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
				conf,
				InsightJobsQueueName,
				InsightResultsQueueName,
				CheckupJobsQueueName,
				CheckupResultsQueueName,
				SourceEventsQueueName,
				PostgreSQLUser,
				PostgreSQLPassword,
				PostgreSQLHost,
				PostgreSQLPort,
				PostgreSQLDb,
				PostgreSQLSSLMode,
				HttpServerAddress,
				DescribeTimeoutHours,
				CheckupIntervalHours,
				MustSummarizeIntervalHours,
				KaytuHelmChartLocation,
				FluxSystemNamespace,
			)
			if err != nil {
				return err
			}

			defer s.Stop()

			return s.Run(ctx)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "The scheduler id")

	return cmd
}
