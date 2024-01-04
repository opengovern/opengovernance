package summarizer

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer/types"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/jq"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

const (
	JobQueueTopic    = "compliance-summarizer-job-queue"
	ResultQueueTopic = "compliance-summarizer-job-result"
	ConsumerGroup    = "compliance-summarizer"
)

type Config struct {
	ElasticSearch         config.ElasticSearch
	NATS                  config.NATS
	PrometheusPushAddress string
	Inventory             config.KaytuService
	Onboard               config.KaytuService
}

type Worker struct {
	config   Config
	logger   *zap.Logger
	esClient kaytu.Client
	jq       *jq.JobQueue

	inventoryClient inventoryClient.InventoryServiceClient
	onboardClient   onboardClient.OnboardServiceClient
}

func InitializeNewWorker(
	config Config,
	logger *zap.Logger,
	prometheusPushAddress string,
) (*Worker, error) {
	esClient, err := kaytu.NewClient(kaytu.ClientConfig{
		Addresses:     []string{config.ElasticSearch.Address},
		Username:      &config.ElasticSearch.Username,
		Password:      &config.ElasticSearch.Password,
		IsOpenSearch:  &config.ElasticSearch.IsOpenSearch,
		AwsRegion:     &config.ElasticSearch.AwsRegion,
		AssumeRoleArn: &config.ElasticSearch.AssumeRoleArn,
	})
	if err != nil {
		return nil, err
	}

	jq, err := jq.New(config.NATS.URL, logger)
	if err != nil {
		return nil, err
	}

	w := &Worker{
		config:          config,
		logger:          logger,
		esClient:        esClient,
		jq:              jq,
		inventoryClient: inventoryClient.NewInventoryServiceClient(config.Inventory.BaseURL),
		onboardClient:   onboardClient.NewOnboardServiceClient(config.Onboard.BaseURL),
	}

	return w, nil
}

// Run is a blocking function so you may decide to call it in another goroutine.
// It runs a NATS consumer and it will close it when the given context is closed.
func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("starting to consume")

	consumeCtx, err := w.jq.Consume(ctx, "compliance", "", []string{JobQueueTopic}, ConsumerGroup, func(msg jetstream.Msg) {
		w.logger.Info("received a new job")

		if err := w.ProcessMessage(context.Background(), msg); err != nil {
			w.logger.Error("failed to process message", zap.Error(err))
		}

		w.logger.Info("processing a job completed")
	})
	if err != nil {
		return err
	}

	w.logger.Info("consuming")

	for {
		select {
		case <-ctx.Done():
			consumeCtx.Stop()
		default:
		}
	}
}

func (w *Worker) ProcessMessage(ctx context.Context, msg jetstream.Msg) error {
	startTime := time.Now()

	var job types.Job
	err := json.Unmarshal(msg.Data(), &job)
	if err != nil {
		return err
	}

	defer func() {
		result := JobResult{
			Job:       job,
			StartedAt: startTime,
			Status:    ComplianceSummarizerSucceeded,
			Error:     "",
		}

		if err != nil {
			result.Error = err.Error()
			result.Status = ComplianceSummarizerFailed
		}

		resultJson, err := json.Marshal(result)
		if err != nil {
			w.logger.Error("failed to create job result json", zap.Error(err))
			return
		}

		if err := w.jq.Produce(ctx, ResultQueueTopic, resultJson, fmt.Sprintf("job-result-%d", job.ID)); err != nil {
			w.logger.Error("failed to publish job result", zap.String("jobResult", string(resultJson)), zap.Error(err))
		}
	}()

	runtime.GC()

	w.logger.Info("running job", zap.ByteString("job", msg.Data()))

	err = w.RunJob(job)
	if err != nil {
		w.logger.Info("failure while running job", zap.Error(err))
		return err
	}

	return nil
}

func (w *Worker) Stop() error {
	return nil
}
