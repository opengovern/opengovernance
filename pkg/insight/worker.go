package insight

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"strings"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/queue"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/prometheus/client_golang/prometheus/push"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"
	"go.uber.org/zap"
)

type Worker struct {
	id             string
	jobQueue       queue.Interface
	jobResultQueue queue.Interface
	kfkProducer    *confluent_kafka.Producer
	kfkTopic       string
	s3Bucket       string
	logger         *zap.Logger
	steampipeConn  *steampipe.Database
	es             keibi.Client
	pusher         *push.Pusher
	onboardClient  client.OnboardServiceClient
	uploader       *s3manager.Uploader
}

func InitializeWorker(
	id string,
	rabbitMQUsername string, rabbitMQPassword string, rabbitMQHost string, rabbitMQPort int,
	insightJobQueue string, insightJobResultQueue string,
	kafkaBrokers []string, kafkaTopic string,
	logger *zap.Logger,
	prometheusPushAddress string,
	steampipeHost string, steampipePort string, steampipeDb string, steampipeUsername string, steampipePassword string,
	elasticSearchAddress string, elasticSearchUsername string, elasticSearchPassword string,
	onboardBaseURL string,
	s3Endpoint, s3AccessKey, s3AccessSecret, s3Region, s3Bucket string,
) (w *Worker, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	} else if kafkaTopic == "" {
		return nil, fmt.Errorf("'kfkTopic' must be set to a non empty string")
	}

	w = &Worker{id: id, kfkTopic: kafkaTopic}
	defer func() {
		if err != nil && w != nil {
			w.Stop()
		}
	}()

	qCfg := queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = insightJobQueue
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = w.id
	insightQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobQueue = insightQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = insightJobResultQueue
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = w.id
	insightResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobResultQueue = insightResultsQueue

	producer, err := newKafkaProducer(kafkaBrokers)
	if err != nil {
		return nil, err
	}

	w.kfkProducer = producer

	w.logger = logger

	// setup steampipe connection
	steampipeConn, err := steampipe.NewSteampipeDatabase(steampipe.Option{
		Host: steampipeHost,
		Port: steampipePort,
		User: steampipeUsername,
		Pass: steampipePassword,
		Db:   steampipeDb,
	})
	w.steampipeConn = steampipeConn
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized steampipe database: ", steampipeConn)

	w.pusher = push.New(prometheusPushAddress, "insight-worker")
	w.pusher.Collector(DoInsightJobsCount).
		Collector(DoInsightJobsDuration)

	defaultAccountID := "default"
	w.es, err = keibi.NewClient(keibi.ClientConfig{
		Addresses: []string{elasticSearchAddress},
		Username:  &elasticSearchUsername,
		Password:  &elasticSearchPassword,
		AccountID: &defaultAccountID,
	})
	if err != nil {
		return nil, err
	}

	w.onboardClient = client.NewOnboardServiceClient(onboardBaseURL, nil)

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

	w.logger.Error("Waiting indefinitly for messages. To exit press CTRL+C")
	for msg := range msgs {
		var job Job
		if err := json.Unmarshal(msg.Body, &job); err != nil {
			w.logger.Error("Failed to unmarshal task", zap.Error(err))
			err = msg.Nack(false, false)
			if err != nil {
				w.logger.Error("Failed nacking message", zap.Error(err))
			}
			continue
		}
		w.logger.Info("Processing job", zap.Int("jobID", int(job.JobID)))
		result := job.Do(w.es, w.steampipeConn, w.onboardClient, w.kfkProducer, w.uploader, w.s3Bucket, w.kfkTopic, w.logger)
		w.logger.Info("Publishing job result", zap.Int("jobID", int(job.JobID)))
		err := w.jobResultQueue.Publish(result)
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
	}

	return fmt.Errorf("descibe jobs channel is closed")
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
