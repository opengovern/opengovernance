package cost_estimator

import (
	"fmt"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/client"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
	"strings"
)

type HttpHandler struct {
	client kaytu.Client
	//awsClient   kaytuAws.Client
	azureClient     kaytuAzure.Client
	kafkaProducer   *confluent_kafka.Producer
	kafkaTopic      string
	workspaceClient client.CostEstimatorPricesClient

	logger *zap.Logger
}

func InitializeHttpHandler(
	workspaceClientURL string,
	esConf config.ElasticSearch,
	logger *zap.Logger,
) (h *HttpHandler, err error) {
	h = &HttpHandler{}
	h.logger = logger

	h.logger.Info("Initializing http handler")

	h.client, err = kaytu.NewClient(kaytu.ClientConfig{
		Addresses:     []string{esConf.Address},
		Username:      &esConf.Username,
		Password:      &esConf.Password,
		IsOpenSearch:  &esConf.IsOpenSearch,
		AwsRegion:     &esConf.AwsRegion,
		AssumeRoleArn: &esConf.AssumeRoleArn,
	})
	if err != nil {
		return nil, err
	}

	//h.awsClient = kaytuAws.Client{
	//	Client: h.client,
	//}
	h.azureClient = kaytuAzure.Client{
		Client: h.client,
	}
	h.logger.Info("Initialized elasticSearch", zap.String("client", fmt.Sprintf("%v", h.client)))

	h.workspaceClient = client.NewCostEstimatorClient(workspaceClientURL)
	h.logger.Info("Workspace client initialized")

	kafkaProducer, err := newKafkaProducer(strings.Split(KafkaService, ","))
	if err != nil {
		return nil, err
	}
	h.kafkaProducer = kafkaProducer
	h.kafkaTopic = KafkaTopic

	return h, nil
}

func newKafkaProducer(brokers []string) (*confluent_kafka.Producer, error) {
	return confluent_kafka.NewProducer(&confluent_kafka.ConfigMap{
		"bootstrap.servers":            strings.Join(brokers, ","),
		"linger.ms":                    100,
		"compression.type":             "lz4",
		"message.timeout.ms":           10000,
		"queue.buffering.max.messages": 100000,
		"message.max.bytes":            104857600,
	})
}
