package audit_job

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/opengovern/opencomply/services/describe/db/model"
	integration_type "github.com/opengovern/opencomply/services/integration/integration-type"
	"os"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/opengovern/og-util/pkg/config"
	esSinkClient "github.com/opengovern/og-util/pkg/es/ingest/client"
	"github.com/opengovern/og-util/pkg/jq"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/og-util/pkg/steampipe"
	complianceClient "github.com/opengovern/opencomply/services/compliance/client"
	metadataClient "github.com/opengovern/opencomply/services/metadata/client"
	"go.uber.org/zap"
)

type Config struct {
	ElasticSearch config.ElasticSearch
	NATS          config.NATS
	Compliance    config.OpenGovernanceService
	EsSink        config.OpenGovernanceService
	Steampipe     config.Postgres
}

type Worker struct {
	config           Config
	logger           *zap.Logger
	steampipeConn    *steampipe.Database
	esClient         opengovernance.Client
	jq               *jq.JobQueue
	complianceClient complianceClient.ComplianceServiceClient
	metadataClient   metadataClient.MetadataServiceClient
	sinkClient       esSinkClient.EsSinkServiceClient
}

var (
	ManualTrigger = os.Getenv("MANUAL_TRIGGER")
)

func NewWorker(
	config Config,
	logger *zap.Logger,
	ctx context.Context,
) (*Worker, error) {
	for _, integrationType := range integration_type.IntegrationTypes {
		describerConfig := integrationType.GetConfiguration()
		err := steampipe.PopulateSteampipeConfig(config.ElasticSearch, describerConfig.SteampipePluginName)
		if err != nil {
			return nil, err
		}
	}
	if err := steampipe.PopulateOpenGovernancePluginSteampipeConfig(config.ElasticSearch, config.Steampipe); err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Minute)

	steampipeConn, err := steampipe.StartSteampipeServiceAndGetConnection(logger)
	if err != nil {
		return nil, err
	}

	esClient, err := opengovernance.NewClient(opengovernance.ClientConfig{
		Addresses:     []string{config.ElasticSearch.Address},
		Username:      &config.ElasticSearch.Username,
		Password:      &config.ElasticSearch.Password,
		IsOnAks:       &config.ElasticSearch.IsOnAks,
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

	queueTopic := JobQueueTopic
	if ManualTrigger == "true" {
		queueTopic = JobQueueTopicManuals
	}

	if err := jq.Stream(ctx, StreamName, "compliance runner job queue", []string{queueTopic, ResultQueueTopic}, 1000000); err != nil {
		return nil, err
	}

	return &Worker{
		config:           config,
		logger:           logger,
		steampipeConn:    steampipeConn,
		esClient:         esClient,
		jq:               jq,
		complianceClient: complianceClient.NewComplianceClient(config.Compliance.BaseURL),
		sinkClient:       esSinkClient.NewEsSinkServiceClient(logger, config.EsSink.BaseURL),
	}, nil
}

// Run is a blocking function so you may decide to call it in another goroutine.
// It runs a NATS consumer and it will close it when the given context is closed.
func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("starting to consume")

	queueTopic := JobQueueTopic
	consumer := ConsumerGroup
	if ManualTrigger == "true" {
		queueTopic = JobQueueTopicManuals
		consumer = ConsumerGroupManuals
	}

	consumeCtx, err := w.jq.ConsumeWithConfig(ctx, consumer, StreamName, []string{queueTopic},
		jetstream.ConsumerConfig{
			DeliverPolicy:     jetstream.DeliverAllPolicy,
			AckPolicy:         jetstream.AckExplicitPolicy,
			AckWait:           time.Hour,
			MaxDeliver:        1,
			InactiveThreshold: time.Hour,
			Replicas:          1,
			MemoryStorage:     false,
		}, nil,
		func(msg jetstream.Msg) {
			w.logger.Info("received a new job")
			w.logger.Info("committing")
			if err := msg.InProgress(); err != nil {
				w.logger.Error("failed to send the initial in progress message", zap.Error(err), zap.Any("msg", msg))
			}
			ticker := time.NewTicker(15 * time.Second)
			go func() {
				for range ticker.C {
					if err := msg.InProgress(); err != nil {
						w.logger.Error("failed to send an in progress message", zap.Error(err), zap.Any("msg", msg))
					}
				}
			}()

			err := w.ProcessMessage(ctx, msg)
			if err != nil {
				w.logger.Error("failed to process message", zap.Error(err))
			}
			ticker.Stop()

			if err := msg.Ack(); err != nil {
				w.logger.Error("failed to send the ack message", zap.Error(err), zap.Any("msg", msg))
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

func (w *Worker) ProcessMessage(ctx context.Context, msg jetstream.Msg) (err error) {
	var job AuditJob

	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		return err
	}

	result := JobResult{
		JobID:          job.JobID,
		Status:         model.AuditJobStatusInProgress,
		FailureMessage: "",
	}

	defer func() {
		if err != nil {
			result.FailureMessage = err.Error()
			result.Status = model.AuditJobStatusFailed
		} else {
			result.Status = model.AuditJobStatusSucceeded
		}

		resultJson, err := json.Marshal(result)
		if err != nil {
			w.logger.Error("failed to create job result json", zap.Error(err))
			return
		}

		if _, err := w.jq.Produce(ctx, ResultQueueTopic, resultJson, fmt.Sprintf("audit-job-result-%d", job.JobID)); err != nil {
			w.logger.Error("failed to publish job result", zap.String("jobResult", string(resultJson)), zap.Error(err))
		}
	}()

	resultJson, err := json.Marshal(result)
	if err != nil {
		w.logger.Error("failed to create job in progress json", zap.Error(err))
		return err
	}

	if _, err := w.jq.Produce(ctx, ResultQueueTopic, resultJson, fmt.Sprintf("audit-job-inprogress-%d", job.JobID)); err != nil {
		w.logger.Error("failed to publish job in progress", zap.String("jobInProgress", string(resultJson)), zap.Error(err))
	}

	w.logger.Info("running job", zap.ByteString("job", msg.Data()))

	err = w.RunJob(ctx, &job)
	if err != nil {
		return err
	}

	return nil
}

func (w *Worker) Stop() error {
	w.steampipeConn.Conn().Close()
	steampipe.StopSteampipeService(w.logger)
	return nil
}
