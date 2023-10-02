package insight

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"strings"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/queue"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"
)

type Worker struct {
	id string

	config WorkerConfig

	jobQueue       queue.Interface
	jobResultQueue queue.Interface

	logger *zap.Logger

	kfkProducer   *confluent_kafka.Producer
	onboardClient client.OnboardServiceClient
	pusher        *push.Pusher

	s3Bucket string
	uploader *s3manager.Uploader
}

type WorkerConfig struct {
	RabbitMQ              config.RabbitMQ
	ElasticSearch         config.ElasticSearch
	Kafka                 config.Kafka
	Onboard               config.KaytuService
	PrometheusPushAddress string
}

func InitializeWorker(
	id string,
	workerConfig WorkerConfig,
	insightJobQueue string, insightJobResultQueue string,
	logger *zap.Logger,
	s3Endpoint, s3AccessKey, s3AccessSecret, s3Region, s3Bucket string,
) (w *Worker, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	} else if workerConfig.Kafka.Topic == "" {
		return nil, fmt.Errorf("'kfkTopic' must be set to a non empty string")
	}

	w = &Worker{id: id}
	defer func() {
		if err != nil && w != nil {
			w.Stop()
		}
	}()

	qCfg := queue.Config{}
	qCfg.Server.Username = workerConfig.RabbitMQ.Username
	qCfg.Server.Password = workerConfig.RabbitMQ.Password
	qCfg.Server.Host = workerConfig.RabbitMQ.Service
	qCfg.Server.Port = 5672
	qCfg.Queue.Name = insightJobQueue
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = w.id
	insightQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobQueue = insightQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = workerConfig.RabbitMQ.Username
	qCfg.Server.Password = workerConfig.RabbitMQ.Password
	qCfg.Server.Host = workerConfig.RabbitMQ.Service
	qCfg.Server.Port = 5672
	qCfg.Queue.Name = insightJobResultQueue
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = w.id
	insightResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobResultQueue = insightResultsQueue

	producer, err := newKafkaProducer(strings.Split(workerConfig.Kafka.Addresses, ","))
	if err != nil {
		return nil, err
	}

	w.kfkProducer = producer

	w.logger = logger

	w.pusher = push.New(workerConfig.PrometheusPushAddress, "insight-worker")
	w.pusher.Collector(DoInsightJobsCount).
		Collector(DoInsightJobsDuration)

	w.onboardClient = client.NewOnboardServiceClient(workerConfig.Onboard.BaseURL, nil)

	if s3Region == "" {
		s3Region = "us-west-2"
	}

	var awsConfig *aws.Config
	if s3AccessKey == "" || s3AccessSecret == "" {
		//load default credentials
		awsConfig = &aws.Config{
			Region: aws.String(s3Region),
		}
	} else {
		awsConfig = &aws.Config{
			Endpoint:    aws.String(s3Endpoint),
			Region:      aws.String(s3Region),
			Credentials: credentials.NewStaticCredentials(s3AccessKey, s3AccessSecret, ""),
		}
	}

	sess := session.Must(session.NewSession(awsConfig))
	w.uploader = s3manager.NewUploader(sess)
	w.s3Bucket = s3Bucket

	return w, nil
}

func (w *Worker) Run() error {
	msgs, err := w.jobQueue.Consume()
	if err != nil {
		return err
	}
	msg := <-msgs
	var job Job
	if err = json.Unmarshal(msg.Body, &job); err != nil {
		w.logger.Error("Failed to unmarshal task", zap.Error(err))
		err2 := msg.Nack(false, false)
		if err2 != nil {
			w.logger.Error("Failed nacking message", zap.Error(err2))
		}
		return err
	}
	w.logger.Info("Processing job", zap.Int("jobID", int(job.JobID)))
	result := job.Do(w.config.ElasticSearch, w.onboardClient,
		w.kfkProducer,
		w.uploader, w.s3Bucket, CurrentWorkspaceID,
		w.config.Kafka.Topic, w.logger)
	w.logger.Info("Publishing job result", zap.Int("jobID", int(job.JobID)))
	err = w.jobResultQueue.Publish(result)
	if err != nil {
		w.logger.Error("Failed to send results to queue: %s", zap.Error(err))
	}

	if err := msg.Ack(false); err != nil {
		w.logger.Error("Failed acking message", zap.Error(err))
	}

	err = w.pusher.Push()
	if err != nil {
		w.logger.Error("Failed to push metrics", zap.Error(err))
	}

	return nil
}

func (w *Worker) Stop() {
	w.pusher.Push()

	if w.jobQueue != nil {
		w.jobQueue.Close() //nolint,gosec
		w.jobQueue = nil
	}

	if w.jobResultQueue != nil {
		w.jobResultQueue.Close() //nolint,gosec
		w.jobResultQueue = nil
	}

	if w.kfkProducer != nil {
		w.kfkProducer.Close() //nolint,gosec
		w.kfkProducer = nil
	}
}

func newKafkaProducer(brokers []string) (*confluent_kafka.Producer, error) {
	return confluent_kafka.NewProducer(&confluent_kafka.ConfigMap{
		"bootstrap.servers":            strings.Join(brokers, ","),
		"linger.ms":                    100,
		"compression.type":             "lz4",
		"message.timeout.ms":           10000,
		"queue.buffering.max.messages": 100000,
	})
}
