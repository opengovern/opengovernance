package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	types2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/worker/types"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	client2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"io"
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

var tracer = otel.Tracer("new_compliance_worker")

func (j *Job) Run(jc JobConfig) error {
	hctx := &httpclient.Context{UserRole: api.InternalRole}

	ctx := context.Background()
	_, span1 := tracer.Start(ctx, "new_JobRun")
	span1.SetName("new_JobRun")

	assignment, err := jc.complianceClient.ListAssignmentsByBenchmark(hctx, j.BenchmarkID)
	if err != nil {
		jc.logger.Error("failed to list assignments by benchmark", zap.String("benchmarkID", j.BenchmarkID), zap.Error(err))
		return err
	}

	span1.AddEvent("information", trace.WithAttributes(
		attribute.String("benchmark id", j.BenchmarkID),
	))
	span1.End()

	bs := types2.BenchmarkSummary{
		BenchmarkID:      j.BenchmarkID,
		JobID:            j.ID,
		EvaluatedAtEpoch: j.CreatedAt.Unix(),
		BenchmarkResult: types2.Result{
			QueryResult:    map[types.ComplianceResult]int{},
			SeverityResult: map[types.FindingSeverity]int{},
		},
		Connections:         map[string]types2.Result{},
		ResourceTypes:       map[string]types2.Result{},
		ResourceCollections: map[string]types2.Result{},
		Policies:            map[string]types2.PolicyResult{},
	}

	_, span2 := tracer.Start(ctx, "new_ConnectionsRun")
	span2.SetName("new_ConnectionsRun")

	jc.logger.Info("Running benchmark",
		zap.String("benchmark_id", j.BenchmarkID),
		zap.Int("connections", len(assignment.Connections)),
		zap.Int("resourceCollection", len(assignment.ResourceCollections)),
	)

	for _, connection := range assignment.Connections {
		if !connection.Status {
			continue
		}

		err := j.RunForConnection(ctx, connection.ConnectionID, nil, &bs, jc)
		if err != nil {
			jc.logger.Error("failed to run for connection", zap.String("connectionID", connection.ConnectionID), zap.Error(err))
			return err
		}
	}

	span2.End()
	_, span3 := tracer.Start(ctx, "new_ResourceCollections")
	span3.SetName("new_ResourceCollections")

	for _, resourceCollection := range assignment.ResourceCollections {
		if !resourceCollection.Status {
			continue
		}

		err := j.RunForConnection(ctx, "all", &resourceCollection.ResourceCollectionID, &bs, jc)
		if err != nil {
			jc.logger.Error("failed to run for resource collection", zap.String("resourceCollectionID", resourceCollection.ResourceCollectionID), zap.Error(err))
			return err
		}
	}

	span3.End()
	_, span4 := tracer.Start(ctx, "new_RemoveOldFindings")
	span4.SetName("new_RemoveOldFindings")

	//err = RemoveOldFindings(jc, j.ID, j.BenchmarkID)
	//if err != nil {
	//	return err
	//}

	span4.End()
	_, span5 := tracer.Start(ctx, "new_Summarize")
	span5.SetName("new_Summarize")

	bs.Summarize()

	jc.logger.Info(fmt.Sprintf("bs={%v}", bs))

	span5.End()
	_, span6 := tracer.Start(ctx, "kafka_DoSend")
	span6.SetName("kafka_DoSend")

	err = kafka.DoSend(jc.kafkaProducer, jc.config.Kafka.Topic, -1, []kafka.Doc{bs}, jc.logger, nil)
	if err != nil {
		return err
	}

	span6.End()
	return nil
}

func (j *Job) RunForConnection(ctx context.Context, connectionID string, resourceCollectionID *string, benchmarkSummary *types2.BenchmarkSummary, jc JobConfig) error {
	onboardConnectionId := connectionID
	if connectionID != "all" {
		conn, err := jc.onboardClient.GetSource(&httpclient.Context{UserRole: api.InternalRole}, connectionID)
		if err != nil {
			jc.logger.Error("failed to get source", zap.String("connectionID", connectionID), zap.Error(err))
			return err
		}
		onboardConnectionId = conn.ConnectionID
	}

	err := jc.steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyAccountID, onboardConnectionId)
	if err != nil {
		jc.logger.Error("failed to set account id", zap.String("connectionID", connectionID), zap.Error(err))
		return err
	}
	defer jc.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyAccountID)
	err = jc.steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyClientType, "compliance")
	if err != nil {
		jc.logger.Error("failed to set client type", zap.String("connectionID", connectionID), zap.Error(err))
		return err
	}
	defer jc.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyClientType)

	executionPlans, err := ListExecutionPlans(connectionID, nil, j.BenchmarkID, jc)
	if err != nil {
		jc.logger.Error("failed to list execution plans", zap.String("connectionID", connectionID), zap.Error(err))
		return err
	}

	for _, plans := range executionPlans.Plans {
		if len(plans) == 0 {
			continue
		}
		plan := plans[0]

		jc.logger.Info("running query",
			zap.String("query", plan.Query.QueryToExecute),
			zap.String("connectionID", connectionID),
			zap.String("benchmarkID", j.BenchmarkID),
			zap.Int("planCount", len(executionPlans.Plans)),
			zap.Int("duplicateCount", len(plans)),
		)

		res, err := jc.steampipeConn.QueryAll(ctx, plan.Query.QueryToExecute)
		if err != nil {
			jc.logger.Error("failed to run query", zap.String("query", plan.Query.QueryToExecute), zap.Error(err))
			return err
		}

		//TODO-Saleh probably push result into s3 to keep historical data

		//if !j.IsStack {
		//	findings, err = j.FilterFindings(plan, findings, jc)
		//	if err != nil {
		//		return err
		//	}
		//}
		for _, plan := range plans {
			findings, err := j.ExtractFindings(plan, connectionID, resourceCollectionID, res, jc)
			if err != nil {
				jc.logger.Error("failed to extract findings", zap.String("query", plan.Query.QueryToExecute), zap.Error(err))
				return err
			}
			for _, f := range findings {
				benchmarkSummary.AddFinding(f)
			}

			for idx, finding := range findings {
				finding.ParentBenchmarks = plan.ParentBenchmarkIDs
				findings[idx] = finding
			}

			var docs []kafka.Doc
			for _, f := range findings {
				docs = append(docs, f)
			}

			jc.logger.Info("pushing findings into kafka",
				zap.Int("count", len(docs)),
				zap.Int("dataCount", len(res.Data)),
				zap.String("connectionID", connectionID),
				zap.String("benchmarkID", plan.Policy.ID),
			)
			err = kafka.DoSend(jc.kafkaProducer, jc.config.Kafka.Topic, -1, docs, jc.logger, nil)
			if err != nil {
				jc.logger.Error("failed to push findings into kafka", zap.String("connectionID", connectionID), zap.Error(err))
				return err
			}
		}

	}
	return nil
}

func RemoveOldFindings(jc JobConfig, jobID uint, benchmarkID string) error {
	ctx := context.Background()
	es := jc.esClient.ES()

	index := []string{types.FindingsIndex, types.ResourceCollectionsFindingsIndex}

	var filters []map[string]any
	filters = append(filters, map[string]any{
		"bool": map[string]any{
			"should": []map[string]any{
				{
					"bool": map[string]any{
						"must_not": map[string]any{
							"term": map[string]any{
								"complianceJobID": jobID,
							},
						},
					},
				},
				{
					"bool": map[string]any{
						"must": map[string]any{
							"term": map[string]any{
								"parentBenchmarks": benchmarkID,
							},
						},
					},
				},
			},
		},
	})

	request := make(map[string]any)
	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}

	query, err := json.Marshal(request)
	if err != nil {
		return err
	}

	jc.logger.Info("delete by query", zap.String("body", string(query)))

	res, err := es.DeleteByQuery(
		index,
		bytes.NewReader(query),
		es.DeleteByQuery.WithContext(ctx),
	)
	if err != nil {
		return err
	}

	defer kaytu.CloseSafe(res)
	if err != nil {
		b, _ := io.ReadAll(res.Body)
		fmt.Printf("failure while deleting es: %v\n%s\n", err, string(b))
		return err
	} else if err := kaytu.CheckError(res); err != nil {
		if kaytu.IsIndexNotFoundErr(err) {
			return nil
		}
		b, _ := io.ReadAll(res.Body)
		fmt.Printf("failure while querying es: %v\n%s\n", err, string(b))
		return err
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	return nil
}
