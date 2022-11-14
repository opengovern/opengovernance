package onboard

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/connector"

	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
)

const (
	SourceEventsQueueName = "source-events-queue"
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

	VaultAddress  = os.Getenv("VAULT_ADDRESS")
	VaultToken    = os.Getenv("VAULT_TOKEN")
	VaultRoleName = os.Getenv("VAULT_ROLE")
	VaultCaPath   = os.Getenv("VAULT_TLS_CA_PATH")
	VaultUseTLS   = strings.ToLower(strings.TrimSpace(os.Getenv("VAULT_USE_TLS"))) == "true"

	AWSPermissionCheckURL = os.Getenv("AWS_PERMISSION_CHECK_URL")

	HttpAddress = os.Getenv("HTTP_ADDRESS")
)

func Command() *cobra.Command {
	return &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			return start(cmd.Context())
		},
	}
}

func start(ctx context.Context) error {
	err := connector.Init()
	if err != nil {
		return fmt.Errorf("populating connectors: %w", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}

	handler, err := InitializeHttpHandler(
		RabbitMQUsername,
		RabbitMQPassword,
		RabbitMQService,
		RabbitMQPort,
		SourceEventsQueueName,
		PostgreSQLUser,
		PostgreSQLPassword,
		PostgreSQLHost,
		PostgreSQLPort,
		PostgreSQLDb,
		PostgreSQLSSLMode,
		VaultAddress,
		VaultToken,
		VaultRoleName,
		VaultCaPath,
		VaultUseTLS,
		logger,
		AWSPermissionCheckURL,
	)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(logger, HttpAddress, handler)
}
