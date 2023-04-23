package describe

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/ProtonMail/gopenpgp/v2/helper"
	"gitlab.com/keibiengine/keibi-engine/pkg/aws"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/producer"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
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
	id                    string
	jobQueue              queue.Interface
	jobResultQueue        queue.Interface
	kfkProducer           sarama.SyncProducer
	kfkTopic              string
	vault                 vault.SourceConfig
	rdb                   *redis.Client
	logger                *zap.Logger
	pusher                *push.Pusher
	tp                    *trace.TracerProvider
	describeIntervalHours time.Duration
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

	result := job.Do(ctx, w.vault, w.rdb, w.kfkProducer, w.kfkTopic, w.logger)
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

type CloudNativeConnectionWorker struct {
	instanceId                         string
	id                                 string
	job                                DescribeConnectionJob
	kfkProducer                        *producer.InMemorySaramaProducer
	kfkTopic                           string
	cloudNativeOutputQueue             *azeventhubs.ProducerClient
	cloudNativeBlobStorageClient       *azblob.Client
	cloudNativeBlobOutputEncryptionKey string
	vault                              vault.SourceConfig
	logger                             *zap.Logger
}

func InitializeCloudNativeConnectionWorker(
	instanceId string,
	id string,
	job DescribeConnectionJob,
	kfkTopic string,
	cloudNativeOutputQueueName string,
	cloudNativeOutputConnectionString string,
	cloudNativeBlobStorageConnectionString string,
	cloudNativeBlobOutputEncryptionKey string,
	secretMap map[string]any,
	logger *zap.Logger,
) (w *CloudNativeConnectionWorker, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	}
	if kfkTopic == "" {
		return nil, fmt.Errorf("'kfkTopic' must be set to a non empty string")
	}

	w = &CloudNativeConnectionWorker{
		instanceId:                         instanceId,
		id:                                 id,
		job:                                job,
		kfkTopic:                           kfkTopic,
		cloudNativeBlobOutputEncryptionKey: cloudNativeBlobOutputEncryptionKey,
	}
	defer func() {
		if err != nil && w != nil {
			w.Stop()
		}
	}()

	w.kfkProducer = producer.NewInMemorySaramaProducer()

	// setup vault
	v := vault.NewInMemoryVaultSourceConfig()
	err = v.Write(job.ConfigReg, secretMap)
	if err != nil {
		return nil, err
	}

	w.logger = logger

	w.vault = v

	producerClient, err := azeventhubs.NewProducerClientFromConnectionString(cloudNativeOutputConnectionString, cloudNativeOutputQueueName, nil)
	if err != nil {
		return nil, err
	}
	w.cloudNativeOutputQueue = producerClient

	blobClient, err := azblob.NewClientFromConnectionString(cloudNativeBlobStorageConnectionString, nil)
	if err != nil {
		return nil, err
	}
	w.cloudNativeBlobStorageClient = blobClient

	return w, nil
}

type CloudNativeConnectionWorkerMessage struct {
	Topic   string
	Key     sarama.StringEncoder
	Headers []sarama.RecordHeader
	Value   sarama.ByteEncoder
}

type CloudNativeConnectionWorkerResult struct {
	JobID         string `json:"jobId" validate:"required"`
	BlobName      string `json:"blobName" validate:"required"`
	ContainerName string `json:"containerName" validate:"required"`
}

type CloudNativeConnectionWorkerData struct {
	JobID     string                                `json:"jobId" validate:"required"`
	JobResult DescribeConnectionJobResult           `json:"jobResult" validate:"required"`
	JobData   []*CloudNativeConnectionWorkerMessage `json:"jobData" validate:"required"`
}

func (w *CloudNativeConnectionWorker) Run(ctx context.Context, sendTimeout bool) error {
	var jobResult DescribeConnectionJobResult
	if sendTimeout {
		jobResult = w.job.CloudTimeout()
	} else {
		jobResult = w.job.Do(ctx, w.vault, nil, w.kfkProducer, w.kfkTopic, w.logger)
	}

	saramaMessages := w.kfkProducer.GetMessages()
	messages := make([]*CloudNativeConnectionWorkerMessage, 0, len(saramaMessages))
	for _, saramaMessage := range saramaMessages {
		messages = append(messages, &CloudNativeConnectionWorkerMessage{
			Topic:   saramaMessage.Topic,
			Key:     saramaMessage.Key.(sarama.StringEncoder),
			Headers: saramaMessage.Headers,
			Value:   saramaMessage.Value.(sarama.ByteEncoder),
		})
	}

	resultData := &CloudNativeConnectionWorkerData{
		JobID:     fmt.Sprint(w.instanceId),
		JobResult: jobResult,
		JobData:   messages,
	}

	resultDataJson, err := json.Marshal(resultData)
	if err != nil {
		w.logger.Error("Failed to marshal messages", zap.Error(err))
		return err
	}

	encMessage, err := helper.EncryptMessageArmored(w.cloudNativeBlobOutputEncryptionKey, string(resultDataJson))
	if err != nil {
		w.logger.Error("Failed to encrypt messages", zap.Error(err))
		return err
	}

	containerName := fmt.Sprintf("connection-worker-%s", strings.ToLower(fmt.Sprint(w.instanceId)))

	// get hash of resource types
	hash := sha256.New()
	for _, v := range w.job.ResourceJobs {
		hash.Write([]byte(v))
	}
	hashString := hex.EncodeToString(hash.Sum(nil))

	blobName := fmt.Sprintf("%s---%s.json", w.job.SourceID, hashString)

	retryCount := 30
	for i := 0; i < retryCount; i++ {
		_, err = w.cloudNativeBlobStorageClient.UploadBuffer(ctx, containerName, blobName, []byte(encMessage), nil)
		if err != nil {
			w.logger.Error("Failed to upload blob", zap.Error(err))
			if i == retryCount-1 {
				for k, v := range jobResult.Result {
					v.Error = fmt.Sprintf("%s\nFailed to upload blob: %s", v.Error, err.Error())
					v.Status = api.DescribeResourceJobFailed
					jobResult.Result[k] = v
				}
				break
			}
			time.Sleep(time.Duration(rand.Intn(15)+1) * time.Second)
		} else {
			break
		}
	}

	return nil
}

func (w *CloudNativeConnectionWorker) Stop() {
	return
}

type OldCleanerWorker struct {
	lowerThan uint
	esClient  *elasticsearch.Client
	logger    *zap.Logger
}

func InitializeOldCleanerWorker(
	lowerThan uint,
	elasticAddress string,
	elasticUsername string,
	elasticPassword string,
	logger *zap.Logger,
) (w *OldCleanerWorker, err error) {
	w = &OldCleanerWorker{lowerThan: lowerThan}

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

	return w, nil
}

func (w *OldCleanerWorker) Run() error {
	startTime := time.Now().Unix()
	ctx, cancel := context.WithTimeout(context.Background(), 10*cleanupJobTimeout)
	defer cancel()

	awsResourceTypes := aws.ListResourceTypes()
	azureResourceTypes := azure.ListResourceTypes()

	resourceTypes := append(awsResourceTypes, azureResourceTypes...)

	for _, resourceType := range resourceTypes {
		rIndex := ResourceTypeToESIndex(resourceType)
		fmt.Printf("Cleaning resources with resource_job_id lower than %d from index %s\n", w.lowerThan, rIndex)

		query := map[string]any{
			"query": map[string]any{
				"range": map[string]any{
					"resource_job_id": map[string]any{
						"lt": w.lowerThan,
					},
				},
			},
		}

		// Delete the resources from both inventory_summary and resource specific index
		indices := []string{
			rIndex,
		}

		resp, err := keibi.DeleteByQuery(ctx, w.esClient, indices, query,
			w.esClient.DeleteByQuery.WithRefresh(true),
			w.esClient.DeleteByQuery.WithConflicts("proceed"),
		)
		if err != nil {
			DoDescribeCleanupJobsDuration.WithLabelValues(resourceType, "failure").Observe(float64(time.Now().Unix() - startTime))
			DoDescribeCleanupJobsCount.WithLabelValues(resourceType, "failure").Inc()
			return err
		}

		if len(resp.Failures) != 0 {
			body, err := json.Marshal(resp)
			if err != nil {
				return err
			}

			DoDescribeCleanupJobsDuration.WithLabelValues(resourceType, "failure").Observe(float64(time.Now().Unix() - startTime))
			DoDescribeCleanupJobsCount.WithLabelValues(resourceType, "failure").Inc()
			return fmt.Errorf("elasticsearch: delete by query: %s", string(body))
		}

		fmt.Printf("Successfully delete %d resources of type %s\n", resp.Deleted, resourceType)
		DoDescribeCleanupJobsDuration.WithLabelValues(resourceType, "successful").Observe(float64(time.Now().Unix() - startTime))
		DoDescribeCleanupJobsCount.WithLabelValues(resourceType, "successful").Inc()
	}

	return nil
}
