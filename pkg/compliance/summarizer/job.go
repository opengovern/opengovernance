package summarizer

import (
	"context"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/es"
	types2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer/types"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
	"time"
)

type Job struct {
	ID          uint
	BenchmarkID string
	CreatedAt   time.Time
}

type JobConfig struct {
	config        Config
	logger        *zap.Logger
	esClient      kaytu.Client
	kafkaProducer *confluent_kafka.Producer
}

func (j *Job) Run(jc JobConfig) error {
	ctx := context.Background()

	jc.logger.Info("Running summarizer",
		zap.Uint("job_id", j.ID),
		zap.String("benchmark_id", j.BenchmarkID),
	)

	paginator, err := es.NewFindingPaginator(jc.esClient, types.FindingsIndex, []kaytu.BoolFilter{
		kaytu.NewTermFilter("parentBenchmarks", j.BenchmarkID),
	}, nil)
	if err != nil {
		return err
	}

	bs := types2.BenchmarkSummary{
		BenchmarkID:      j.BenchmarkID,
		JobID:            j.ID,
		EvaluatedAtEpoch: j.CreatedAt.Unix(),
		BenchmarkResult: types2.Result{
			QueryResult:    map[types.ComplianceResult]int{},
			SeverityResult: map[types.FindingSeverity]int{},
			SecurityScore:  0,
		},
		Connections:         map[string]types2.Result{},
		ResourceCollections: map[string]types2.Result{},
		ResourceTypes:       map[string]types2.Result{},
		Policies:            map[string]types2.PolicyResult{},
	}

	for paginator.HasNext() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, f := range page {
			bs.AddFinding(f)
		}
	}

	bs.Summarize()

	err = kafka.DoSend(jc.kafkaProducer, jc.config.Kafka.Topic, -1, []kafka.Doc{bs}, jc.logger, nil)
	if err != nil {
		return err
	}
	return nil
}
