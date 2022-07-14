package inventory

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"go.uber.org/zap"
)

var (
	RedisAddress = os.Getenv("REDIS_ADDRESS")

	ElasticSearchAddress  = os.Getenv("ES_ADDRESS")
	ElasticSearchUsername = os.Getenv("ES_USERNAME")
	ElasticSearchPassword = os.Getenv("ES_PASSWORD")

	PostgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	PostgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	PostgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	PostgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	PostgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")

	SteampipeHost     = os.Getenv("STEAMPIPE_HOST")
	SteampipePort     = os.Getenv("STEAMPIPE_PORT")
	SteampipeDb       = os.Getenv("STEAMPIPE_DB")
	SteampipeUser     = os.Getenv("STEAMPIPE_USERNAME")
	SteampipePassword = os.Getenv("STEAMPIPE_PASSWORD")

	SchedulerBaseUrl = os.Getenv("SCHEDULER_BASE_URL")
	OnboardBaseUrl = os.Getenv("ONBOARD_BASE_URL")

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
		ElasticSearchAddress,
		ElasticSearchUsername,
		ElasticSearchPassword,
		PostgreSQLHost,
		PostgreSQLPort,
		PostgreSQLDb,
		PostgreSQLUser,
		PostgreSQLPassword,
		SteampipeHost,
		SteampipePort,
		SteampipeDb,
		SteampipeUser,
		SteampipePassword,
		SchedulerBaseUrl,
		OnboardBaseUrl,
		logger,
		RedisAddress,
	)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(logger, HttpAddress, handler)
}
