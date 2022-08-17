package describe

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	//"go.opentelemetry.io/otel/semconv"
	"net/http"
	"strings"

	trace2 "gitlab.com/keibiengine/keibi-engine/pkg/trace"
	"go.opentelemetry.io/otel/codes"

	"github.com/go-redis/redis/v8"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/hashicorp/vault/api/auth/kubernetes"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/queue"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"

	"go.opentelemetry.io/otel"
)

type Worker struct {
	id             string
	jobQueue       queue.Interface
	jobResultQueue queue.Interface
	kfkProducer    sarama.SyncProducer
	kfkTopic       string
	vault          vault.SourceConfig
	rdb            *redis.Client
	es             keibi.Client
	logger         *zap.Logger
	pusher         *push.Pusher
	tp             *trace.TracerProvider
}

func InitializeWorker(
	id string,
	rabbitMQUsername string,
	rabbitMQPassword string,
	rabbitMQHost string,
	rabbitMQPort int,
	describeJobQueue string,
	describeJobResultQueue string,
	kafkaBrokers []string,
	kafkaTopic string,
	vaultAddress string,
	vaultRoleName string,
	vaultToken string,
	vaultCaPath string,
	vaultUseTLS bool,
	logger *zap.Logger,
	elasticSearchAddress string,
	elasticSearchUsername string,
	elasticSearchPassword string,
	prometheusPushAddress string,
	redisAddress string,
	jaegerAddress string,
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
	qCfg.Queue.Name = describeJobQueue
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = w.id
	describeQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobQueue = describeQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeJobResultQueue
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = w.id
	describeResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobResultQueue = describeResultsQueue

	producer, err := newKafkaProducer(strings.Split(KafkaService, ","))
	if err != nil {
		return nil, err
	}

	w.kfkProducer = producer

	k8sAuth, err := kubernetes.NewKubernetesAuth(
		vaultRoleName,
		kubernetes.WithServiceAccountToken(vaultToken),
	)
	if err != nil {
		return nil, err
	}

	// setup vault
	v, err := vault.NewSourceConfig(vaultAddress, vaultCaPath, k8sAuth, vaultUseTLS)
	if err != nil {
		return nil, err
	}

	w.logger = logger

	w.logger.Info("Connected to vault:", zap.String("vaultAddress", vaultAddress))
	w.vault = v
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

	w.pusher = push.New(prometheusPushAddress, "describe-worker")
	w.pusher.Collector(DoDescribeJobsCount).
		Collector(DoDescribeJobsDuration)

	w.rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	exp, _ := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerAddress)))
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"http://keibi.io/",
			attribute.String("environment", "production"),
		),
	)

	w.tp = trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(r),
	)
	otel.SetTracerProvider(w.tp)

	return w, nil
}

func (w *Worker) Run(ctx context.Context) error {
	msgs, err := w.jobQueue.Consume()
	if err != nil {
		return err
	}

	msg := <-msgs

	ctx, span := otel.Tracer(trace2.DescribeWorkerTrace).Start(ctx, "HandleMessage")

	var job DescribeJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		w.logger.Error("Failed to unmarshal task", zap.Error(err))
		err = msg.Nack(false, false)
		if err != nil {
			w.logger.Error("Failed nacking message", zap.Error(err))
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		return err
	}
	result := job.Do(ctx, w.vault, w.rdb, w.es, w.kfkProducer, w.kfkTopic, w.logger)
	if strings.Contains(result.Error, "ThrottlingException") ||
		strings.Contains(result.Error, "Rate exceeded") ||
		strings.Contains(result.Error, "RateExceeded") {
		w.logger.Error("Rate error happened, retrying in a bit")
		time.Sleep(5 * time.Second)

		if err := msg.Nack(false, true); err != nil {
			w.logger.Error("Failed requeueing message", zap.Error(err))
		}
		span.RecordError(errors.New(result.Error))
		span.SetStatus(codes.Error, result.Error)
		span.End()
		return nil
	}

	err = w.jobResultQueue.Publish(result)
	if err != nil {
		w.logger.Error("Failed to send results to queue: %s", zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	if err := msg.Ack(false); err != nil {
		w.logger.Error("Failed acking message", zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	err = w.pusher.Push()
	if err != nil {
		w.logger.Error("Failed to push metrics", zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()

	w.logger.Info(fmt.Sprintf("Job[%d]: Succeeded", job.JobID))
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

	w.tp.Shutdown(context.Background())
}

func newKafkaProducer(brokers []string) (sarama.SyncProducer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Retry.Max = 3
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Successes = true
	cfg.Version = sarama.V2_1_0_0

	producer, err := sarama.NewSyncProducer(strings.Split(KafkaService, ","), cfg)
	if err != nil {
		return nil, err
	}

	return producer, nil
}

type CleanupWorker struct {
	id              string
	cleanupJobQueue queue.Interface
	esClient        *elasticsearch.Client
	logger          *zap.Logger
	pusher          *push.Pusher
}

func InitializeCleanupWorker(
	id string,
	rabbitMQUsername string,
	rabbitMQPassword string,
	rabbitMQHost string,
	rabbitMQPort int,
	cleanupJobQueueName string,
	elasticAddress string,
	elasticUsername string,
	elasticPassword string,
	logger *zap.Logger,
	prometheusPushAddress string,
) (w *CleanupWorker, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	}

	w = &CleanupWorker{id: id}
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
	qCfg.Queue.Name = cleanupJobQueueName
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = w.id
	cleanupJobQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.cleanupJobQueue = cleanupJobQueue

	esClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{elasticAddress},
		Username:  elasticUsername,
		Password:  elasticPassword,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint,gosec
			},
		},
	})
	if err != nil {
		return nil, err
	}

	w.esClient = esClient
	w.logger = logger
	w.pusher = push.New(prometheusPushAddress, "describe-cleanup")
	w.pusher.Collector(DoDescribeJobsCount).
		Collector(DoDescribeJobsDuration)

	return w, nil
}

func (w *CleanupWorker) Run() error {
	msgs, err := w.cleanupJobQueue.Consume()
	if err != nil {
		return err
	}

	w.logger.Error("Waiting indefinitly for messages. To exit press CTRL+C")
	for msg := range msgs {
		var job DescribeCleanupJob
		if err := json.Unmarshal(msg.Body, &job); err != nil {
			w.logger.Error("Failed to unmarshal task: %s", zap.Error(err))
			err = msg.Nack(false, false)
			if err != nil {
				w.logger.Error("Failed nacking message", zap.Error(err))
			}
			continue
		}

		err := job.Do(w.esClient)
		if err != nil {
			w.logger.Error("Failed to cleanup resources", zap.Error(err))
			err = msg.Nack(false, true) // Requeue if there is any failure
			if err != nil {
				w.logger.Error("Failed nacking message", zap.Error(err))
			}
			continue
		}

		if err := msg.Ack(false); err != nil {
			w.logger.Error("Failed acking message", zap.Error(err))
		}
	}

	return fmt.Errorf("descibe jobs channel is closed")
}

func (w *CleanupWorker) Stop() {
	w.pusher.Push()
	if w.cleanupJobQueue != nil {
		w.cleanupJobQueue.Close()
		w.cleanupJobQueue = nil
	}
}

type ConnectionWorker struct {
	id             string
	jobQueue       queue.Interface
	jobResultQueue queue.Interface
	kfkProducer    sarama.SyncProducer
	kfkTopic       string
	vault          vault.SourceConfig
	rdb            *redis.Client
	es             keibi.Client
	logger         *zap.Logger
	pusher         *push.Pusher
	tp             *trace.TracerProvider
}

func InitializeConnectionWorker(
	id string,
	rabbitMQUsername string,
	rabbitMQPassword string,
	rabbitMQHost string,
	rabbitMQPort int,
	describeJobQueue string,
	describeJobResultQueue string,
	kafkaBrokers []string,
	kafkaTopic string,
	vaultAddress string,
	vaultRoleName string,
	vaultToken string,
	vaultCaPath string,
	vaultUseTLS bool,
	logger *zap.Logger,
	elasticSearchAddress string,
	elasticSearchUsername string,
	elasticSearchPassword string,
	prometheusPushAddress string,
	redisAddress string,
	jaegerAddress string,
) (w *ConnectionWorker, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	} else if kafkaTopic == "" {
		return nil, fmt.Errorf("'kfkTopic' must be set to a non empty string")
	}

	w = &ConnectionWorker{id: id, kfkTopic: kafkaTopic}
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
	qCfg.Queue.Name = describeJobQueue
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = w.id
	describeQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobQueue = describeQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = describeJobResultQueue
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = w.id
	describeResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobResultQueue = describeResultsQueue

	producer, err := newKafkaProducer(strings.Split(KafkaService, ","))
	if err != nil {
		return nil, err
	}

	w.kfkProducer = producer

	k8sAuth, err := kubernetes.NewKubernetesAuth(
		vaultRoleName,
		kubernetes.WithServiceAccountToken(vaultToken),
	)
	if err != nil {
		return nil, err
	}

	// setup vault
	v, err := vault.NewSourceConfig(vaultAddress, vaultCaPath, k8sAuth, vaultUseTLS)
	if err != nil {
		return nil, err
	}

	w.logger = logger

	w.logger.Info("Connected to vault:", zap.String("vaultAddress", vaultAddress))
	w.vault = v
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

	w.pusher = push.New(prometheusPushAddress, "describe-connection-worker")
	w.pusher.Collector(DoDescribeJobsCount).
		Collector(DoDescribeJobsDuration)

	w.rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	exp, _ := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerAddress)))
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"http://keibi.io/",
			attribute.String("environment", "production"),
		),
	)

	w.tp = trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(r),
	)
	otel.SetTracerProvider(w.tp)

	return w, nil
}

func (w *ConnectionWorker) Run(ctx context.Context) error {
	msgs, err := w.jobQueue.Consume()
	if err != nil {
		return err
	}

	msg := <-msgs

	ctx, span := otel.Tracer(trace2.DescribeWorkerTrace).Start(ctx, "HandleMessage")

	var job DescribeConnectionJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		w.logger.Error("Failed to unmarshal task", zap.Error(err))
		err = msg.Nack(false, false)
		if err != nil {
			w.logger.Error("Failed nacking message", zap.Error(err))
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		return err
	}
	result := job.Do(ctx, w.vault, w.rdb, w.es, w.kfkProducer, w.kfkTopic, w.logger)

	err = w.jobResultQueue.Publish(result)
	if err != nil {
		w.logger.Error("Failed to send results to queue: %s", zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	if err := msg.Ack(false); err != nil {
		w.logger.Error("Failed acking message", zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	err = w.pusher.Push()
	if err != nil {
		w.logger.Error("Failed to push metrics", zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()

	w.logger.Info(fmt.Sprintf("Job[%d]: Succeeded", job.JobID))
	return nil
}

func (w *ConnectionWorker) Stop() {
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

	w.tp.Shutdown(context.Background())
}
