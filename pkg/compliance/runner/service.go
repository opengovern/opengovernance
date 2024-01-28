package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner/types"
	"time"

	complianceClient "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	schedulerClient "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/jq"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type Config struct {
	ElasticSearch         config.ElasticSearch
	NATS                  config.NATS
	Compliance            config.KaytuService
	Onboard               config.KaytuService
	Inventory             config.KaytuService
	Scheduler             config.KaytuService
	Steampipe             config.Postgres
	PrometheusPushAddress string
}

type Worker struct {
	config           Config
	logger           *zap.Logger
	steampipeConn    *steampipe.Database
	esClient         kaytu.Client
	jq               *jq.JobQueue
	complianceClient complianceClient.ComplianceServiceClient
	onboardClient    onboardClient.OnboardServiceClient
	inventoryClient  inventoryClient.InventoryServiceClient
	schedulerClient  schedulerClient.SchedulerServiceClient
}

func NewWorker(
	config Config,
	logger *zap.Logger,
	prometheusPushAddress string,
) (*Worker, error) {
	err := steampipe.PopulateSteampipeConfig(config.ElasticSearch, source.CloudAWS)
	if err != nil {
		return nil, err
	}
	err = steampipe.PopulateSteampipeConfig(config.ElasticSearch, source.CloudAzure)
	if err != nil {
		return nil, err
	}

	steampipeConn, err := steampipe.StartSteampipeServiceAndGetConnection(logger)
	if err != nil {
		return nil, err
	}

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
		config:           config,
		logger:           logger,
		steampipeConn:    steampipeConn,
		esClient:         esClient,
		jq:               jq,
		complianceClient: complianceClient.NewComplianceClient(config.Compliance.BaseURL),
		onboardClient:    onboardClient.NewOnboardServiceClient(config.Onboard.BaseURL),
		inventoryClient:  inventoryClient.NewInventoryServiceClient(config.Inventory.BaseURL),
		schedulerClient:  schedulerClient.NewSchedulerServiceClient(config.Scheduler.BaseURL),
	}

	return w, nil
}

// Run is a blocking function so you may decide to call it in another goroutine.
// It runs a NATS consumer and it will close it when the given context is closed.
func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("starting to consume")

	consumeCtx, err := w.jq.Consume(ctx, "compliance-runner", StreamName, []string{JobQueueTopic}, ConsumerGroup, func(msg jetstream.Msg) {
		w.logger.Info("received a new job")

		commit, requeue, err := w.ProcessMessage(context.Background(), msg)
		if err != nil {
			w.logger.Error("failed to process message", zap.Error(err))
		}

		if requeue {
			if err := msg.Nak(); err != nil {
				w.logger.Error("failed to send a not ack message", zap.Error(err))
			}
		}

		if commit {
			w.logger.Info("committing")
			if err := msg.Ack(); err != nil {
				w.logger.Error("failed to send an ack message", zap.Error(err))
			}
		}

		w.logger.Info("processing a job completed")
	})
	if err != nil {
		return err
	}

	w.logger.Info("consuming")

	<-ctx.Done()
	consumeCtx.Drain()
	consumeCtx.Stop()

	return nil
}

func (w *Worker) ProcessMessage(ctx context.Context, msg jetstream.Msg) (commit bool, requeue bool, err error) {
	var job types.Job

	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		return true, false, err
	}

	result := types.JobResult{
		Job:               job,
		StartedAt:         time.Now(),
		Status:            ComplianceRunnerInProgress,
		Error:             "",
		TotalFindingCount: nil,
	}

	defer func() {
		if err != nil {
			result.Error = err.Error()
			result.Status = ComplianceRunnerFailed
		} else {
			result.Status = ComplianceRunnerSucceeded
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

	resultJson, err := json.Marshal(result)
	if err != nil {
		w.logger.Error("failed to create job in progress json", zap.Error(err))
		return true, false, err
	}

	if err := w.jq.Produce(ctx, ResultQueueTopic, resultJson, fmt.Sprintf("job-result-%d", job.ID)); err != nil {
		w.logger.Error("failed to publish job in progress", zap.String("jobInProgress", string(resultJson)), zap.Error(err))
	}

	w.logger.Info("running job", zap.ByteString("job", msg.Data()))

	totalFindingCount, err := w.RunJob(ctx, job)
	if err != nil {
		return true, false, err
	}

	result.TotalFindingCount = &totalFindingCount
	return true, false, nil
}

func (w *Worker) Stop() error {
	w.steampipeConn.Conn().Close()
	steampipe.StopSteampipeService(w.logger)
	return nil
}
