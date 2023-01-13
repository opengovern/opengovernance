package metadata

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/internal/cache"
	"gitlab.com/keibiengine/keibi-engine/pkg/metadata/internal/database"
	"go.uber.org/zap"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
)

type HttpHandler struct {
	db    database.Database
	redis *cache.MetadataRedisCache
}

func InitializeHttpHandler(
	postgresUsername string,
	postgresPassword string,
	postgresHost string,
	postgresPort string,
	postgresDb string,
	postgresSSLMode string,
	redisAddress string,
	redisPassword string,
	redisDB int,
	redisTTL time.Duration,
	logger *zap.Logger,
) (*HttpHandler, error) {

	fmt.Println("Initializing http handler")

	cfg := postgres.Config{
		Host:    postgresHost,
		Port:    postgresPort,
		User:    postgresUsername,
		Passwd:  postgresPassword,
		DB:      postgresDb,
		SSLMode: postgresSSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	fmt.Println("Connected to the postgres database: ", postgresDb)

	db := database.NewDatabase(orm)
	err = db.Initialize()
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized postgres database: ", postgresDb)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: redisPassword,
		DB:       redisDB,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}
	redisCache := cache.NewMetadataRedisCache(redisClient, redisTTL)
	fmt.Printf("Connected to the redis database: %d in address %s", redisDB, redisAddress)

	return &HttpHandler{
		db:    db,
		redis: redisCache,
	}, nil
}
