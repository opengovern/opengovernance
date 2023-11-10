package runner

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
)

type Caller struct {
	RootBenchmark      string
	ParentBenchmarkIDs []string
	PolicyID           string
	PolicySeverity     types.FindingSeverity
}

type ExecutionPlan struct {
	Callers        []Caller
	QueryID        string
	QueryEngine    string
	QueryConnector source.Type

	ConnectionID         *string
	ResourceCollectionID *string
}

type Job struct {
	ID          uint
	ParentJobID uint
	CreatedAt   time.Time

	ExecutionPlan ExecutionPlan
}

type JobConfig struct {
	config        Config
	logger        *zap.Logger
	steampipeConn *steampipe.Database
	esClient      kaytu.Client
	kafkaProducer *confluent_kafka.Producer
}

func (w *Worker) Initialize(ctx context.Context, j Job) error {
	providerAccountID := "all"
	if j.ExecutionPlan.ConnectionID != nil &&
		*j.ExecutionPlan.ConnectionID != "" &&
		*j.ExecutionPlan.ConnectionID != "all" {
		conn, err := w.onboardClient.GetSource(&httpclient.Context{UserRole: api.InternalRole}, *j.ExecutionPlan.ConnectionID)
		if err != nil {
			w.logger.Error("failed to get source", zap.Error(err), zap.String("connection_id", *j.ExecutionPlan.ConnectionID))
			return err
		}
		providerAccountID = conn.ConnectionID
	}

	err := w.steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyAccountID, providerAccountID)
	if err != nil {
		w.logger.Error("failed to set account id", zap.Error(err))
		return err
	}
	err = w.steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyClientType, "compliance")
	if err != nil {
		w.logger.Error("failed to set client type", zap.Error(err))
		return err
	}

	if j.ExecutionPlan.ResourceCollectionID != nil {
		rc, err := w.inventoryClient.GetResourceCollection(&httpclient.Context{UserRole: api.InternalRole}, *j.ExecutionPlan.ResourceCollectionID)
		if err != nil {
			w.logger.Error("failed to get resource collection", zap.Error(err), zap.String("rc_id", *j.ExecutionPlan.ResourceCollectionID))
			return err
		}
		filtersJson, err := json.Marshal(rc.Filters)
		if err != nil {
			w.logger.Error("failed to marshal resource collection filters", zap.Error(err), zap.String("rc_id", *j.ExecutionPlan.ResourceCollectionID))
			return err
		}
		err = w.steampipeConn.SetConfigTableValue(ctx, steampipe.KaytuConfigKeyResourceCollectionFilters, base64.StdEncoding.EncodeToString(filtersJson))
		if err != nil {
			w.logger.Error("failed to set resource collection filters", zap.Error(err), zap.String("rc_id", *j.ExecutionPlan.ResourceCollectionID))
			return err
		}
	}

	return nil
}

func (w *Worker) RunJob(j Job) (int, error) {
	ctx := context.Background()

	w.logger.Info("Running query",
		zap.Uint("job_id", j.ID),
		zap.String("query_id", j.ExecutionPlan.QueryID),
		zap.Stringp("query_id", j.ExecutionPlan.ConnectionID),
		zap.Stringp("rc_id", j.ExecutionPlan.ResourceCollectionID),
	)

	if err := w.Initialize(ctx, j); err != nil {
		return 0, err
	}
	defer w.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyAccountID)
	defer w.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyClientType)
	defer w.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyResourceCollectionFilters)

	query, err := w.complianceClient.GetQuery(&httpclient.Context{UserRole: api.InternalRole}, j.ExecutionPlan.QueryID)
	if err != nil {
		return 0, err
	}

	res, err := w.steampipeConn.QueryAll(ctx, query.QueryToExecute)
	if err != nil {
		return 0, err
	}

	w.logger.Info("Extracting and pushing to kafka",
		zap.Uint("job_id", j.ID),
		zap.Int("res_count", len(res.Data)),
		zap.Int("caller_count", len(j.ExecutionPlan.Callers)),
	)
	totalFindingCountMap := make(map[string]int)
	for _, caller := range j.ExecutionPlan.Callers {
		findings, err := j.ExtractFindings(w.logger, caller, res, *query)
		if err != nil {
			return 0, err
		}

		mapKey := fmt.Sprintf("%s---___---%s", caller.RootBenchmark, caller.PolicyID)
		if _, ok := totalFindingCountMap[mapKey]; !ok {
			totalFindingCountMap[mapKey] = len(findings)
		}

		var docs []kafka.Doc
		for _, f := range findings {
			docs = append(docs, f)
		}

		err = kafka.DoSend(w.kafkaProducer, w.config.Kafka.Topic, -1, docs, w.logger, nil)
		if err != nil {
			w.logger.Error("failed to send findings", zap.Error(err), zap.String("benchmark_id", caller.RootBenchmark), zap.String("policy_id", caller.PolicyID))
			return 0, err
		}

		err = w.RemoveOldFindings(j.ID, j.ExecutionPlan.ConnectionID, j.ExecutionPlan.ResourceCollectionID, caller.RootBenchmark, caller.PolicyID)
		if err != nil {
			w.logger.Error("failed to remove old findings", zap.Error(err), zap.String("benchmark_id", caller.RootBenchmark), zap.String("policy_id", caller.PolicyID))
			return 0, err
		}
	}

	totalFindingCount := 0
	for _, v := range totalFindingCountMap {
		totalFindingCount += v
	}

	w.logger.Info("Finished job",
		zap.Uint("job_id", j.ID),
		zap.String("query_id", j.ExecutionPlan.QueryID),
		zap.Stringp("query_id", j.ExecutionPlan.ConnectionID),
		zap.Stringp("rc_id", j.ExecutionPlan.ResourceCollectionID),
	)
	return totalFindingCount, nil
}

func (w *Worker) RemoveOldFindings(jobID uint,
	connectionId *string,
	resourceCollectionId *string,
	benchmarkID,
	policyID string) error {
	ctx := context.Background()
	idx := types.FindingsIndex
	if resourceCollectionId != nil {
		idx = types.ResourceCollectionsFindingsIndex
	}
	var filters []map[string]any
	mustFilters := make([]map[string]any, 0, 4)
	mustFilters = append(mustFilters, map[string]any{
		"term": map[string]any{
			"benchmarkID": benchmarkID,
		},
	})
	mustFilters = append(mustFilters, map[string]any{
		"term": map[string]any{
			"policyID": policyID,
		},
	})
	if connectionId != nil {
		mustFilters = append(mustFilters, map[string]any{
			"term": map[string]any{
				"connectionID": *connectionId,
			},
		})
	}
	if resourceCollectionId != nil {
		mustFilters = append(mustFilters, map[string]any{
			"term": map[string]any{
				"resourceCollection": *resourceCollectionId,
			},
		})
	}

	filters = append(filters, map[string]any{
		"bool": map[string]any{
			"must": []map[string]any{
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
						"filter": mustFilters,
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

	es := w.esClient.ES()
	res, err := es.DeleteByQuery(
		[]string{idx},
		bytes.NewReader(query),
		es.DeleteByQuery.WithContext(ctx),
	)
	if err != nil {
		w.logger.Error("failed to delete old findings", zap.Error(err), zap.String("benchmark_id", benchmarkID), zap.String("policy_id", policyID))
		return err
	}
	defer kaytu.CloseSafe(res)
	if err != nil {
		b, _ := io.ReadAll(res.Body)
		w.logger.Error("failure while deleting es", zap.Error(err), zap.String("benchmark_id", benchmarkID), zap.String("policy_id", policyID), zap.String("response", string(b)))
		return err
	} else if err := kaytu.CheckError(res); err != nil {
		if kaytu.IsIndexNotFoundErr(err) {
			return nil
		}
		b, _ := io.ReadAll(res.Body)
		w.logger.Error("failure while querying es", zap.Error(err), zap.String("benchmark_id", benchmarkID), zap.String("policy_id", policyID), zap.String("response", string(b)))
		return err
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	return nil
}
