package worker

import (
	"context"
	"fmt"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
	"time"
)

type Job struct {
	ID          uint
	CreatedAt   time.Time
	BenchmarkID string
	IsStack     bool
}

type JobConfig struct {
	config           Config
	logger           *zap.Logger
	complianceClient client.ComplianceServiceClient
	onboardClient    client2.OnboardServiceClient
	steampipeConn    *steampipe.Database
	esClient         kaytu.Client
	kafkaProducer    *confluent_kafka.Producer
}

func (j *Job) Run(jc JobConfig) error {
	hctx := &httpclient.Context{UserRole: api.InternalRole}

	assignment, err := jc.complianceClient.ListAssignmentsByBenchmark(hctx, j.BenchmarkID)
	if err != nil {
		return err
	}

	bs := BenchmarkSummary{
		BenchmarkID:     j.BenchmarkID,
		JobID:           j.ID,
		BenchmarkResult: Result{},
		Connections:     map[string]Result{},
		ResourceTypes:   map[string]Result{},
		Policies:        map[string]PolicyResult{},
	}

	for _, connection := range assignment.Connections {
		err := j.RunForConnection(connection.ConnectionID, nil, &bs, jc)
		if err != nil {
			return err
		}
	}

	for _, resourceCollection := range assignment.ResourceCollections {
		err := j.RunForConnection("all", &resourceCollection.ResourceCollectionID, &bs, jc)
		if err != nil {
			return err
		}
	}

	bs.Summarize()

	jc.logger.Info(fmt.Sprintf("bs={%v}", bs))

	var docs []kafka.Doc
	docs = append(docs, bs)
	err = kafka.DoSend(jc.kafkaProducer, jc.config.Kafka.Topic, -1, docs, jc.logger, nil)
	if err != nil {
		return err
	}

	return nil
}

func (j *Job) RunForConnection(connectionID string, resourceCollectionID *string, benchmarkSummary *BenchmarkSummary, jc JobConfig) error {
	conn, err := jc.onboardClient.GetSource(&httpclient.Context{UserRole: api.InternalRole}, connectionID)
	if err != nil {
		return err
	}

	err = jc.steampipeConn.SetConfigTableValue(context.Background(), steampipe.KaytuConfigKeyAccountID, conn.ConnectionID)
	if err != nil {
		return err
	}

	plans, err := ListExecutionPlans(connectionID, nil, j.BenchmarkID, jc)
	if err != nil {
		return err
	}

	for _, plan := range plans {
		res, err := jc.steampipeConn.QueryAll(context.Background(), plan.Query.QueryToExecute)
		if err != nil {
			return err
		}

		findings, err := j.ExtractFindings(plan, connectionID, resourceCollectionID, res, jc)
		if err != nil {
			return err
		}

		for _, f := range findings {
			benchmarkSummary.AddFinding(f)
		}

		if !j.IsStack {
			findings, err = j.FilterFindings(plan, findings, jc)
			if err != nil {
				return err
			}
		}

		for idx, finding := range findings {
			finding.ParentBenchmarks = plan.ParentBenchmarkIDs
			findings[idx] = finding
		}

		var docs []kafka.Doc
		for _, f := range findings {
			docs = append(docs, f)
		}

		err = kafka.DoSend(jc.kafkaProducer, jc.config.Kafka.Topic, -1, docs, jc.logger, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
