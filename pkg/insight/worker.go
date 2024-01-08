package insight

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	describeClient "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/jq"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"
)

type Worker struct {
	id string

	config WorkerConfig

	jq *jq.JobQueue

	logger *zap.Logger

	onboardClient   client.OnboardServiceClient
	inventoryClient inventoryClient.InventoryServiceClient
	schedulerClient describeClient.SchedulerServiceClient
	pusher          *push.Pusher

	s3Bucket string
	uploader *s3manager.Uploader
}

type WorkerConfig struct {
	NATS                  config.NATS
	ElasticSearch         config.ElasticSearch
	Onboard               config.KaytuService
	Inventory             config.KaytuService
	Scheduler             config.KaytuService
	SteampipePg           config.Postgres
	PrometheusPushAddress string
}

func NewWorker(
	id string,
	workerConfig WorkerConfig,
	insightJobQueue string, insightJobResultQueue string,
	logger *zap.Logger,
	s3Endpoint, s3AccessKey, s3AccessSecret, s3Region, s3Bucket string,
) (*Worker, error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	}

	w := &Worker{id: id, config: workerConfig}

	w.logger = logger

	w.pusher = push.New(workerConfig.PrometheusPushAddress, "insight-worker")
	w.pusher.Collector(DoInsightJobsCount).
		Collector(DoInsightJobsDuration)

	w.onboardClient = client.NewOnboardServiceClient(workerConfig.Onboard.BaseURL)
	w.inventoryClient = inventoryClient.NewInventoryServiceClient(workerConfig.Inventory.BaseURL)
	w.schedulerClient = describeClient.NewSchedulerServiceClient(workerConfig.Scheduler.BaseURL)

	if s3Region == "" {
		s3Region = "us-west-2"
	}

	var awsConfig *aws.Config
	if s3AccessKey == "" || s3AccessSecret == "" {
		// load default credentials
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
	w.jq.Consume(context.Background(), "insight-service", "insight", []string{InsightJobsQueueName}, "", func(msg jetstream.Msg) {
		var job Job
		if err := json.Unmarshal(msg.Data(), &job); err != nil {
			w.logger.Error("Failed to unmarshal task", zap.Error(err))

			if err := msg.Nak(); err != nil {
				w.logger.Error("Failed not ack message", zap.Error(err))
			}

			return
		}

		w.logger.Info("Processing job", zap.Uint("jobID", job.JobID))

		result := job.Do(w.config.ElasticSearch,
			w.config.SteampipePg,
			w.onboardClient,
			w.inventoryClient,
			w.schedulerClient,
			w.uploader, w.s3Bucket,
			CurrentWorkspaceID, w.logger,
		)

		w.logger.Info("Publishing job result", zap.Uint("jobID", job.JobID))

		bytes, err := json.Marshal(result)
		if err != nil {
			w.logger.Error("failed to marshal result as json", zap.Error(err))
		}

		if err := w.jq.Produce(context.Background(), InsightResultsQueueName, bytes, fmt.Sprintf("job-%d", job.JobID)); err != nil {
			w.logger.Error("Failed to send results to queue", zap.Error(err))
		}

		if err := msg.Ack(); err != nil {
			w.logger.Error("Failed ack message", zap.Error(err))
		}

		if err := w.pusher.Push(); err != nil {
			w.logger.Error("Failed to push metrics", zap.Error(err))
		}
	})

	for {
	}
}

func (w *Worker) Stop() {
	w.pusher.Push()
}
