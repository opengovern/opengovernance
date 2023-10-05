package alerting

import (
	"context"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
)

var (
	PostgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	PostgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	PostgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	PostgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	PostgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")
	PostgreSQLSSLMode  = os.Getenv("POSTGRESQL_SSLMODE")

	OnboardCliendBaseUrl    = os.Getenv("ONBOARD_BASE_URL")
	ComplianceClientBaseUrl = os.Getenv("COMPLIANCE_BASE_URL")

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
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}

	handler, err := InitializeHttpHandler(
		PostgreSQLHost,
		PostgreSQLPort,
		PostgreSQLDb,
		PostgreSQLUser,
		PostgreSQLPassword,
		PostgreSQLSSLMode,
		ComplianceClientBaseUrl,
		OnboardCliendBaseUrl,
		logger,
	)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	go handler.TriggerRulesJobCycle()

	return httpserver.RegisterAndStart(logger, HttpAddress, handler)
}
