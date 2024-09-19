package query_runner

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	esSinkClient "github.com/kaytu-io/kaytu-util/pkg/es/ingest/client"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/jq"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	complianceApi "github.com/kaytu-io/open-governance/pkg/compliance/api"
	complianceClient "github.com/kaytu-io/open-governance/pkg/compliance/client"
	inventoryClient "github.com/kaytu-io/open-governance/pkg/inventory/client"
	metadataClient "github.com/kaytu-io/open-governance/pkg/metadata/client"
	onboardClient "github.com/kaytu-io/open-governance/pkg/onboard/client"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type Config struct {
	ElasticSearch         config.ElasticSearch
	NATS                  config.NATS
	Compliance            config.KaytuService
	Onboard               config.KaytuService
	Inventory             config.KaytuService
	Metadata              config.KaytuService
	EsSink                config.KaytuService
	Steampipe             config.Postgres
	PennywiseBaseURL      string `yaml:"pennywise_base_url"`
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
	metadataClient   metadataClient.MetadataServiceClient
	sinkClient       esSinkClient.EsSinkServiceClient

	benchmarkCache map[string]complianceApi.Benchmark
}

func NewWorker(
	config Config,
	logger *zap.Logger,
	prometheusPushAddress string,
	ctx context.Context,
) (*Worker, error) {
	err := steampipe.PopulateSteampipeConfig(config.ElasticSearch, source.CloudAWS)
	if err != nil {
		return nil, err
	}
	err = steampipe.PopulateSteampipeConfig(config.ElasticSearch, source.CloudAzure)
	if err != nil {
		return nil, err
	}

	if err := steampipe.PopulateKaytuPluginSteampipeConfig(config.ElasticSearch, config.Steampipe, config.PennywiseBaseURL); err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Minute)

	steampipeConn, err := steampipe.StartSteampipeServiceAndGetConnection(logger)
	if err != nil {
		return nil, err
	}

	esClient, err := kaytu.NewClient(kaytu.ClientConfig{
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

	if err := jq.Stream(ctx, StreamName, "compliance runner job queue", []string{JobQueueTopic, JobResultQueueTopic}, 1000); err != nil {
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
		metadataClient:   metadataClient.NewMetadataServiceClient(config.Metadata.BaseURL),
		sinkClient:       esSinkClient.NewEsSinkServiceClient(logger, config.EsSink.BaseURL),
		benchmarkCache:   make(map[string]complianceApi.Benchmark),
	}
	ctx2 := &httpclient.Context{Ctx: ctx, UserRole: api.InternalRole}
	benchmarks, err := w.complianceClient.ListAllBenchmarks(ctx2, true)
	if err != nil {
		logger.Error("failed to get benchmarks", zap.Error(err))
		return nil, err
	}
	for _, benchmark := range benchmarks {
		w.benchmarkCache[benchmark.ID] = benchmark
	}

	return w, nil
}

// Run is a blocking function so you may decide to call it in another goroutine.
// It runs a NATS consumer and it will close it when the given context is closed.
func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("starting to consume")

	queueTopic := JobQueueTopic
	consumer := ConsumerGroup

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

			_, _, err := w.ProcessMessage(ctx, msg)
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

func (w *Worker) ProcessMessage(ctx context.Context, msg jetstream.Msg) (commit bool, requeue bool, err error) {
	var job Job

	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		return true, false, err
	}

	w.logger.Info("job message delivered", zap.String("jobID", strconv.Itoa(int(job.ID))))

	result := JobResult{
		ID:             job.ID,
		Status:         QueryRunnerInProgress,
		FailureMessage: "",
	}

	defer func() {
		if err != nil {
			result.FailureMessage = err.Error()
			result.Status = QueryRunnerFailed
		} else {
			result.Status = QueryRunnerSucceeded
		}

		w.logger.Info("job is finished with status", zap.String("ID", strconv.Itoa(int(job.ID))), zap.String("status", string(result.Status)))

		resultJson, err := json.Marshal(result)
		if err != nil {
			w.logger.Error("failed to create job result json", zap.Error(err))
			return
		}

		if _, err := w.jq.Produce(ctx, JobResultQueueTopic, resultJson, fmt.Sprintf("query-runner-result-%d-%d", job.ID, job.RetryCount)); err != nil {
			w.logger.Error("failed to publish job result", zap.String("jobResult", string(resultJson)), zap.Error(err))
		}
	}()

	resultJson, err := json.Marshal(result)
	if err != nil {
		w.logger.Error("failed to create job in progress json", zap.Error(err))
		return true, false, err
	}

	if _, err := w.jq.Produce(ctx, JobResultQueueTopic, resultJson, fmt.Sprintf("query-runner-inprogress-%d-%d", job.ID, job.RetryCount)); err != nil {
		w.logger.Error("failed to publish job in progress", zap.String("jobInProgress", string(resultJson)), zap.Error(err))
	}

	w.logger.Info("running job", zap.ByteString("job", msg.Data()))

	err = w.RunJob(ctx, job)
	if err != nil {
		return true, false, err
	}

	return true, false, nil
}

func (w *Worker) Stop() error {
	w.steampipeConn.Conn().Close()
	steampipe.StopSteampipeService(w.logger)
	return nil
}