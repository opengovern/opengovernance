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

func (j *Job) Run(jc JobConfig) error {
	hctx := &httpclient.Context{UserRole: api.InternalRole}

	assignment, err := jc.complianceClient.ListAssignmentsByBenchmark(hctx, j.BenchmarkID)
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
		},
		Connections:   map[string]types2.Result{},
		ResourceTypes: map[string]types2.Result{},
		Policies:      map[string]types2.PolicyResult{},
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

func (j *Job) RunForConnection(connectionID string, resourceCollectionID *string, benchmarkSummary *types2.BenchmarkSummary, jc JobConfig) error {
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
		jc.logger.Info("running query",
			zap.String("query", plan.Query.QueryToExecute),
			zap.String("connectionID", connectionID),
			zap.String("benchmarkID", plan.Policy.ID),
		)

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

		err = RemoveOldFindings(jc, plan.Policy.ID, connectionID, resourceCollectionID)
		if err != nil {
			return err
		}

		//TODO-Saleh probably push result into s3 to keep historical data

		//if !j.IsStack {
		//	findings, err = j.FilterFindings(plan, findings, jc)
		//	if err != nil {
		//		return err
		//	}
		//}

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
			zap.String("connectionID", connectionID),
			zap.String("benchmarkID", plan.Policy.ID),
		)

		err = kafka.DoSend(jc.kafkaProducer, jc.config.Kafka.Topic, -1, docs, jc.logger, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func RemoveOldFindings(jc JobConfig, policyID string, connectionID string, resourceCollectionID *string) error {
	ctx := context.Background()
	es := jc.esClient.ES()

	index := []string{types.FindingsIndex}

	var filters []map[string]any
	filters = append(filters, map[string]any{
		"term": map[string]any{
			"policyID": policyID,
		},
	})
	filters = append(filters, map[string]any{
		"term": map[string]any{
			"connectionID": connectionID,
		},
	})
	if resourceCollectionID != nil {
		filters = append(filters, map[string]any{
			"term": map[string]any{
				"resourceCollection": resourceCollectionID,
			},
		})
	}

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
