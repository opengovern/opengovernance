package dockertest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
	"gopkg.in/Shopify/sarama.v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

func GetDockerHost() string {
	return getEnv("DOCKERTEST_HOST", "localhost")
}

func GetDockerHostPort() string {
	return getEnv("DOCKER_HOST", "tcp://localhost:2375")
}

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
		orm, err = gorm.Open(postgres.Open(fmt.Sprintf("postgres://postgres:postgres@%s:%s/postgres", GetDockerHost(), resource.GetPort("5432/tcp"))), &gorm.Config{})
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

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         "keibi_test_rabbitmq",
		Hostname:     "keibi_test_rabbitmq",
		Repository:   "bitnami/rabbitmq",
		Tag:          "3.8.23-debian-10-r18",
		ExposedPorts: []string{"15672"},
		PortBindings: map[dc.Port][]dc.PortBinding{
			"15672": {{
				HostIP:   "0.0.0.0",
				HostPort: "15672",
			}},
		},
	})
	require.NoError(err, "status rabbitmq")

	t.Cleanup(func() {
		err := pool.Purge(resource)
		require.NoError(err, "purge resource %s", resource)
	})

	port, err := strconv.Atoi(resource.GetPort("5672/tcp"))
	require.NoError(err, "status rabbitmq")

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		for i := 0; i < 3; i++ {
			res, err := http.Get(fmt.Sprintf("http://user:bitnami@%s:15672/api/aliveness-test/", GetDockerHost()) + "%2F")
			if err != nil {
				return err
			}

			if res.StatusCode != 200 {
				return errors.New("status is not 200")
			}

			time.Sleep(time.Second)
		}

		url := fmt.Sprintf("amqp://%s:%s@%s:%d/",
			"user",
			"bitnami",
			GetDockerHost(),
			port,
		)

		conn, err := amqp.Dial(url)
		if err != nil {
			return err
		}

		_, err = conn.Channel()
		if err != nil {
			return err
		}

		return nil
	})
	require.NoError(err, "wait for rabbitmq connection")

	return RabbitMQServer{
		Host:     GetDockerHost(),
		Port:     port,
		Username: "user",
		Password: "bitnami",
	}
}

type KafkaServer struct {
	Address  string
	Producer sarama.SyncProducer
}

func StartupKafka(t *testing.T) KafkaServer {
	t.Helper()

	require := require.New(t)

	pool, err := dockertest.NewPool("")
	require.NoError(err, "connect to docker")

	net, err := pool.CreateNetwork("kafka")
	require.NoError(err, "creating network")
	t.Cleanup(func() {
		err := pool.RemoveNetwork(net)
		require.NoError(err, "purge resource %s", net)
	})

	zookeeperResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:       "keibi_test_zookeeper",
		Repository: "confluentinc/cp-zookeeper",
		Tag:        "latest",
		Env: []string{
			"ZOOKEEPER_CLIENT_PORT=2181",
			"ZOOKEEPER_TICK_TIME=2000",
		},
		ExposedPorts: []string{"2181"},
		Networks:     []*dockertest.Network{net},
	})
	t.Cleanup(func() {
		err := pool.Purge(zookeeperResource)
		require.NoError(err, "purge resource %s", zookeeperResource)
	})
	require.NoError(err, "status zookeeper")

	kafkaResource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:       "keibi_test_kafka",
		Repository: "confluentinc/cp-kafka",
		Tag:        "latest",
		Env: []string{
			"KAFKA_BROKER_ID=1",
			"KAFKA_ZOOKEEPER_CONNECT=keibi_test_zookeeper:2181",
			fmt.Sprintf("KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://keibi_test_kafka:9092,PLAINTEXT_HOST://%s:29092", GetDockerHost()),
			"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT",
			"KAFKA_INTER_BROKER_LISTENER_NAME=PLAINTEXT",
			"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1",
		},
		ExposedPorts: []string{"29092"},
		PortBindings: map[dc.Port][]dc.PortBinding{
			"29092": {{
				HostIP:   "0.0.0.0",
				HostPort: "29092",
			}},
		},
		Networks: []*dockertest.Network{net},
	})
	t.Cleanup(func() {
		err := pool.Purge(kafkaResource)
		require.NoError(err, "purge resource %s", kafkaResource)
	})
	require.NoError(err, "status kafka")

	kafkaUrl := fmt.Sprintf("%s:", GetDockerHost()) + kafkaResource.GetPort("29092/tcp")
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	var producer sarama.SyncProducer
	err = pool.Retry(func() error {
		cfg := sarama.NewConfig()
		cfg.Producer.Retry.Max = 3
		cfg.Producer.RequiredAcks = sarama.WaitForAll
		cfg.Producer.Return.Successes = true
		cfg.Version = sarama.V2_1_0_0

		producer, err = sarama.NewSyncProducer([]string{kafkaUrl}, cfg)
		if err != nil {
			return err
		}
		return nil
	})
	require.NoError(err, "wait for kafka connection")

	return KafkaServer{
		Address:  kafkaUrl,
		Producer: producer,
	}
}

type ElasticSearchServer struct {
	Address string
}

func StartupElasticSearch(t *testing.T) ElasticSearchServer {
	t.Helper()

	require := require.New(t)

	pool, err := dockertest.NewPool("")
	require.NoError(err, "connect to docker")

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:         "keibi_test_es",
		Repository:   "docker.elastic.co/elasticsearch/elasticsearch",
		Tag:          "7.15.1",
		ExposedPorts: []string{"9200"},
		Env: []string{
			"xpack.security.enabled=false",
			"discovery.type=single-node",
		},
	})
	require.NoError(err, "status elasticsearch")
	esUrl := fmt.Sprintf("http://%s:", GetDockerHost()) + resource.GetPort("9200/tcp")
	t.Cleanup(func() {
		err := pool.Purge(resource)
		require.NoError(err, "purge resource %s", resource)
	})

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	err = pool.Retry(func() error {
		res, err := http.Get(esUrl)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		var resp map[string]interface{}
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return err
		}

		return nil
	})
	require.NoError(err, "wait for elastic search connection")

	return ElasticSearchServer{
		Address: esUrl,
	}
}
