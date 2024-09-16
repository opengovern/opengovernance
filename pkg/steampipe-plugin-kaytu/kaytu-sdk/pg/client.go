package pg

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"os"

	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/open-governance/pkg/steampipe-plugin-kaytu/kaytu-sdk/config"
	"github.com/turbot/steampipe-plugin-sdk/v5/connection"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Client struct {
	db *gorm.DB
}

func NewClientCached(c config.ClientConfig, cache *connection.ConnectionCache, ctx context.Context) (Client, error) {
	value, ok := cache.Get(ctx, "kaytu-pg-client")
	if ok {
		return value.(Client), nil
	}

	plugin.Logger(ctx).Warn("pg client is not cached, creating a new one")

	client, err := NewClient(ctx, c)
	if err != nil {
		return Client{}, err
	}

	cache.Set(ctx, "kaytu-pg-client", client)

	return client, nil
}

func NewInventoryClientCached(c config.ClientConfig, cache *connection.ConnectionCache, ctx context.Context) (Client, error) {
	value, ok := cache.Get(ctx, "kaytu-inventory-pg-client")
	if ok {
		return value.(Client), nil
	}

	plugin.Logger(ctx).Warn("pg client is not cached, creating a new one")

	c.PgDatabase = aws.String("inventory")
	client, err := NewClient(ctx, c)
	if err != nil {
		return Client{}, err
	}

	cache.Set(ctx, "kaytu-inventory-pg-client", client)

	return client, nil
}

func NewClient(ctx context.Context, c config.ClientConfig) (Client, error) {
	if c.PgHost == nil || len(*c.PgHost) == 0 {
		host := os.Getenv("PG_HOST")
		c.PgHost = &host
	}

	if c.PgPort == nil || len(*c.PgPort) == 0 {
		port := os.Getenv("PG_PORT")
		c.PgPort = &port
	}

	if c.PgUser == nil || len(*c.PgUser) == 0 {
		user := os.Getenv("PG_USER")
		c.PgUser = &user
	}

	if c.PgPassword == nil || len(*c.PgPassword) == 0 {
		password := os.Getenv("PG_PASSWORD")
		c.PgPassword = &password
	}

	if c.PgDatabase == nil || len(*c.PgDatabase) == 0 {
		database := os.Getenv("PG_DATABASE")
		c.PgDatabase = &database
	}

	if c.PgSslMode == nil || len(*c.PgSslMode) == 0 {
		sslMode := os.Getenv("PG_SSL_MODE")
		c.PgSslMode = &sslMode
	}

	cfg := postgres.Config{
		Host:    *c.PgHost,
		Port:    *c.PgPort,
		User:    *c.PgUser,
		Passwd:  *c.PgPassword,
		DB:      *c.PgDatabase,
		SSLMode: *c.PgSslMode,
	}

	orm, err := postgres.NewClient(&cfg, zap.NewNop())
	if err != nil {
		return Client{}, err
	}
	return Client{db: orm}, nil
}

func (c Client) DB() *gorm.DB {
	return c.db
}
