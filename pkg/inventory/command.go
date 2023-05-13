package inventory

import (
	"context"
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	RedisAddress = os.Getenv("REDIS_ADDRESS")
	CacheAddress = os.Getenv("CACHE_ADDRESS")

	ElasticSearchAddress  = os.Getenv("ES_ADDRESS")
	ElasticSearchUsername = os.Getenv("ES_USERNAME")
	ElasticSearchPassword = os.Getenv("ES_PASSWORD")

	PostgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	PostgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	PostgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	PostgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	PostgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")
	PostgreSQLSSLMode  = os.Getenv("POSTGRESQL_SSLMODE")

	Neo4jHost     = os.Getenv("NEO4J_HOST")
	Neo4jPort     = os.Getenv("NEO4J_PORT")
	Neo4jUser     = os.Getenv("NEO4J_USERNAME")
	Neo4jPassword = os.Getenv("NEO4J_PASSWORD")

	SteampipeHost     = os.Getenv("STEAMPIPE_HOST")
	SteampipePort     = os.Getenv("STEAMPIPE_PORT")
	SteampipeDb       = os.Getenv("STEAMPIPE_DB")
	SteampipeUser     = os.Getenv("STEAMPIPE_USERNAME")
	SteampipePassword = os.Getenv("STEAMPIPE_PASSWORD")

	S3Endpoint     = os.Getenv("S3_ENDPOINT")
	S3AccessKey    = os.Getenv("S3_ACCESS_KEY")
	S3AccessSecret = os.Getenv("S3_ACCESS_SECRET")
	S3Region       = os.Getenv("S3_REGION")
	S3Bucket       = os.Getenv("S3_BUCKET")

	SchedulerBaseUrl  = os.Getenv("SCHEDULER_BASE_URL")
	OnboardBaseUrl    = os.Getenv("ONBOARD_BASE_URL")
	ComplianceBaseUrl = os.Getenv("COMPLIANCE_BASE_URL")

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
		ElasticSearchAddress, ElasticSearchUsername, ElasticSearchPassword,
		PostgreSQLHost, PostgreSQLPort, PostgreSQLDb, PostgreSQLUser, PostgreSQLPassword, PostgreSQLSSLMode,
		Neo4jHost, Neo4jPort, Neo4jUser, Neo4jPassword,
		SteampipeHost, SteampipePort, SteampipeDb, SteampipeUser, SteampipePassword,
		SchedulerBaseUrl, OnboardBaseUrl, ComplianceBaseUrl,
		logger,
		RedisAddress,
		CacheAddress,
		S3Endpoint, S3AccessKey, S3AccessSecret, S3Region, S3Bucket,
	)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(logger, HttpAddress, handler)
}
