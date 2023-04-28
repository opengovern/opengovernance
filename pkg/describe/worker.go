package describe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	"strings"

	trace2 "gitlab.com/keibiengine/keibi-engine/pkg/trace"
	"go.opentelemetry.io/otel/codes"

	"github.com/go-redis/redis/v8"

	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/hashicorp/vault/api/auth/kubernetes"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/queue"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/vault"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"

	"go.opentelemetry.io/otel"
)

type Worker struct {
	id                      string
	jobQueue                queue.Interface
	jobResultQueue          queue.Interface
	kfkProducer             sarama.SyncProducer
	kfkTopic                string
	vault                   vault.SourceConfig
	rdb                     *redis.Client
	logger                  *zap.Logger
	describeDeliverEndpoint *string
	pusher                  *push.Pusher
	tp                      *trace.TracerProvider
	describeIntervalHours   time.Duration
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

	describeIntervalHours, err := strconv.ParseInt(DescribeIntervalHours, 10, 64)
	if err != nil {
		return nil, err
	}
	w.describeIntervalHours = time.Duration(describeIntervalHours) * time.Hour
	w.describeDeliverEndpoint = &DescribeDeliverEndpoint
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
	if time.Now().Add(-2 * w.describeIntervalHours).After(time.UnixMilli(job.DescribedAt)) {
		// already failed
		w.logger.Error("Job is already failed due to timeout", zap.Uint("jobId", job.JobID), zap.Error(err))
		err = msg.Ack(false)
		return nil
	}

	result := job.Do(ctx, w.vault, w.rdb, w.logger, w.describeDeliverEndpoint)
	if strings.Contains(result.Error, "ThrottlingException") ||
		strings.Contains(result.Error, "Rate exceeded") ||
		strings.Contains(result.Error, "RateExceeded") {
		w.logger.Error(fmt.Sprintf("Rate error happened, retrying in a bit, %s", result.Error))
		time.Sleep(5 * time.Second)

		if err := msg.Nack(false, false); err != nil {
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

func newKafkaClient(brokers []string) (sarama.Client, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Retry.Max = 3
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Successes = true
	cfg.Version = sarama.V2_1_0_0
	cfg.Producer.MaxMessageBytes = 1024 * 1024 * 100 // 10MiB

	client, err := sarama.NewClient(brokers, cfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}
