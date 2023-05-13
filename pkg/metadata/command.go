package metadata

import (
	"context"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"os"
	"strconv"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/connector"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	PostgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	PostgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	PostgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	PostgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	PostgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")
	PostgreSQLSSLMode  = os.Getenv("POSTGRESQL_SSLMODE")

	RedisAddress  = os.Getenv("REDIS_ADDRESS")
	RedisPassword = os.Getenv("REDIS_PASSWORD")
	RedisDB       = os.Getenv("REDIS_DB")
	RedisTTLSec   = os.Getenv("REDIS_TTL_SEC")

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

	redisDBInt := 0
	if RedisDB != "" {
		redisDBInt, err = strconv.Atoi(RedisDB)
		if err != nil {
			return fmt.Errorf("redis db: %w", err)
		}
	}
	redisTTLSecInt := 300
	if RedisTTLSec != "" {
		redisTTLSecInt, err = strconv.Atoi(RedisTTLSec)
		if err != nil {
			return fmt.Errorf("redis ttl: %w", err)
		}
	}

	handler, err := InitializeHttpHandler(
		PostgreSQLUser,
		PostgreSQLPassword,
		PostgreSQLHost,
		PostgreSQLPort,
		PostgreSQLDb,
		PostgreSQLSSLMode,
		RedisAddress,
		RedisPassword,
		redisDBInt,
		time.Duration(redisTTLSecInt)*time.Second,
		logger,
	)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(logger, HttpAddress, handler)
}
