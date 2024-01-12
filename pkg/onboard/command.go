package onboard

import (
	"context"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	SourceEventsQueueName = "source-events-queue"
)

var (
	PostgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	PostgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	PostgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	PostgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	PostgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")
	PostgreSQLSSLMode  = os.Getenv("POSTGRESQL_SSLMODE")

	SteampipeHost     = os.Getenv("STEAMPIPE_HOST")
	SteampipePort     = os.Getenv("STEAMPIPE_PORT")
	SteampipeDb       = os.Getenv("STEAMPIPE_DB")
	SteampipeUser     = os.Getenv("STEAMPIPE_USERNAME")
	SteampipePassword = os.Getenv("STEAMPIPE_PASSWORD")

	AWSPermissionCheckURL = os.Getenv("AWS_PERMISSION_CHECK_URL")
	InventoryBaseURL      = os.Getenv("INVENTORY_BASE_URL")
	DescribeBaseURL       = os.Getenv("DESCRIBE_BASE_URL")

	KeyARN           = os.Getenv("KMS_KEY_ARN")
	KMSAccountRegion = os.Getenv("KMS_ACCOUNT_REGION")

	HttpAddress     = os.Getenv("HTTP_ADDRESS")
	MasterAccessKey = os.Getenv("MASTER_ACCESS_KEY")
	MasterSecretKey = os.Getenv("MASTER_SECRET_KEY")
	MetadataBaseUrl = os.Getenv("METADATA_BASE_URL")
)

func Command() *cobra.Command {
	return &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			return start(cmd.Context())
		},
	}
}

func start(ctx context.Context) error {
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}

	handler, err := InitializeHttpHandler(
		SourceEventsQueueName,
		PostgreSQLUser, PostgreSQLPassword, PostgreSQLHost, PostgreSQLPort, PostgreSQLDb, PostgreSQLSSLMode,
		SteampipeHost, SteampipePort, SteampipeDb, SteampipeUser, SteampipePassword,
		logger,
		AWSPermissionCheckURL,
		KeyARN,
		InventoryBaseURL,
		DescribeBaseURL,
		MasterAccessKey, MasterSecretKey,
	)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(logger, HttpAddress, handler)
}
