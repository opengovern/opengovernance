package dockertest

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func StartupPostgreSQL(t *testing.T) *gorm.DB {
	t.Helper()

	require := require.New(t)

	pool, err := dockertest.NewPool("")
	require.NoError(err, "connect to docker")

	resource, err := pool.Run("postgres", "14", []string{"POSTGRES_PASSWORD=postgres"})
	require.NoError(err, "status postgres")

	t.Cleanup(func() {
		err := pool.Purge(resource)
		require.NoError(err, "purge resource %s", resource)
	})

	var orm *gorm.DB
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		orm, err = gorm.Open(postgres.Open(fmt.Sprintf("postgres://postgres:postgres@localhost:%s/postgres", resource.GetPort("5432/tcp"))), &gorm.Config{})
		if err != nil {
			return err
		}

		d, err := orm.DB()
		if err != nil {
			return err
		}

		return d.Ping()
	})
	require.NoError(err, "wait for postgres connection")

	tx := orm.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	require.NoError(tx.Error, "enable uuid v4")

	return orm
}

type RabbitMQServer struct {
	Host     string
	Port     int
	Username string
	Password string
}

func StartupRabbitMQ(t *testing.T) RabbitMQServer {
	t.Helper()

	require := require.New(t)

	pool, err := dockertest.NewPool("")
	require.NoError(err, "connect to docker")

	resource, err := pool.Run("bitnami/rabbitmq", "3.8.23-debian-10-r18", nil)
	require.NoError(err, "status rabbitmq")

	t.Cleanup(func() {
		err := pool.Purge(resource)
		require.NoError(err, "purge resource %s", resource)
	})

	port, err := strconv.Atoi(resource.GetPort("5672/tcp"))
	require.NoError(err, "status rabbitmq")

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		url := fmt.Sprintf("amqp://%s:%s@%s:%d/",
			"user",
			"bitnami",
			"localhost",
			port,
		)

		_, err := amqp.Dial(url)
		if err != nil {
			return err
		}

		return nil
	})
	require.NoError(err, "wait for rabbitmq connection")

	return RabbitMQServer{
		Host:     "localhost",
		Port:     port,
		Username: "user",
		Password: "bitnami",
	}
}
