package worker

import (
	"context"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
	"time"
)

type Job struct {
	ID                   uint
	CreatedAt            time.Time
	ResourceCollectionId *string
	BenchmarkID          string
	IsStack              bool
}

type JobConfig struct {
	config           Config
	logger           *zap.Logger
	complianceClient client.ComplianceServiceClient
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

	for _, connection := range assignment.Connections {
		err := j.RunForConnection(connection, jc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (j *Job) RunForConnection(assignment api2.BenchmarkAssignedConnection, jc JobConfig) error {
	plans, err := ListExecutionPlans(assignment.ConnectionID, nil, j.BenchmarkID, jc)
	if err != nil {
		return err
	}

	for _, plan := range plans {
		res, err := jc.steampipeConn.QueryAll(context.Background(), plan.Query.QueryToExecute)
		if err != nil {
			return err
		}

		findings, err := j.ExtractFindings(plan, assignment, res, jc)
		if err != nil {
			return err
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
