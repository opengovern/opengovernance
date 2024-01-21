package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	complianceApi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/pipeline"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
)

type Caller struct {
	RootBenchmark      string
	ParentBenchmarkIDs []string
	ControlID          string
	ControlSeverity    types.FindingSeverity
}

type ExecutionPlan struct {
	Callers []Caller
	Query   complianceApi.Query

	ConnectionID         *string
	ProviderConnectionID *string
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
}

func (w *Worker) Initialize(ctx context.Context, j Job) error {
	providerAccountID := "all"
	if j.ExecutionPlan.ProviderConnectionID != nil &&
		*j.ExecutionPlan.ProviderConnectionID != "" {
		providerAccountID = *j.ExecutionPlan.ProviderConnectionID
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

	return nil
}

func (w *Worker) RunJob(ctx context.Context, j Job) (int, error) {
	w.logger.Info("Running query",
		zap.Uint("job_id", j.ID),
		zap.String("query_id", j.ExecutionPlan.Query.ID),
		zap.Stringp("query_id", j.ExecutionPlan.ConnectionID),
	)

	if err := w.Initialize(ctx, j); err != nil {
		return 0, err
	}
	defer w.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyAccountID)
	defer w.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyClientType)
	defer w.steampipeConn.UnsetConfigTableValue(ctx, steampipe.KaytuConfigKeyResourceCollectionFilters)

	res, err := w.steampipeConn.QueryAll(ctx, j.ExecutionPlan.Query.QueryToExecute)
	if err != nil {
		return 0, err
	}

	w.logger.Info("Extracting and pushing to nats",
		zap.Uint("job_id", j.ID),
		zap.Int("res_count", len(res.Data)),
		zap.Int("caller_count", len(j.ExecutionPlan.Callers)),
	)
	totalFindingCountMap := make(map[string]int)
	for _, caller := range j.ExecutionPlan.Callers {
		findings, err := j.ExtractFindings(w.logger, caller, res, j.ExecutionPlan.Query)
		if err != nil {
			return 0, err
		}

		var findingsIDs []string
		for _, f := range findings {
			keys, _ := f.KeysAndIndex()
			findingsIDs = append(findingsIDs, es.HashOf(keys...))
		}

		oldFindings, err := w.FetchFindingsNeededHistoryByIDs(ctx, findingsIDs)

		newFindings := make([]types.Finding, 0, len(findings))
		for _, f := range findings {
			if oldFinding, ok := oldFindings[f.EsID]; ok {
				f.History = oldFinding.History
				if len(f.History) == 0 {
					f.History = append(f.History, types.FindingHistory{
						ComplianceJobID:   oldFinding.ComplianceJobID,
						ConformanceStatus: oldFinding.ConformanceStatus,
						EvaluatedAt:       oldFinding.EvaluatedAt,
					})
				}
				if oldFinding.ConformanceStatus != f.ConformanceStatus {
					f.History = append(f.History, types.FindingHistory{
						ComplianceJobID:   f.ComplianceJobID,
						ConformanceStatus: f.ConformanceStatus,
						EvaluatedAt:       f.EvaluatedAt,
					})
				}
			}
			if len(f.History) == 0 {
				f.History = append(f.History, types.FindingHistory{
					ComplianceJobID:   f.ComplianceJobID,
					ConformanceStatus: f.ConformanceStatus,
					EvaluatedAt:       f.EvaluatedAt,
				})
			}
			newFindings = append(newFindings, f)
		}

		mapKey := fmt.Sprintf("%s---___---%s", caller.RootBenchmark, caller.ControlID)
		if _, ok := totalFindingCountMap[mapKey]; !ok {
			totalFindingCountMap[mapKey] = len(newFindings)
		}

		var docs []es.Doc
		for _, f := range newFindings {
			keys, idx := f.KeysAndIndex()
			f.EsID = es.HashOf(keys...)
			f.EsIndex = idx

			docs = append(docs, f)
		}

		if err := pipeline.SendToPipeline(w.config.ElasticSearch.IngestionEndpoint, docs); err != nil {
			w.logger.Error("failed to send findings", zap.Error(err), zap.String("benchmark_id", caller.RootBenchmark), zap.String("control_id", caller.ControlID))
			return 0, err
		}

		if err := w.RemoveOldFindings(j.ID, j.ExecutionPlan.ConnectionID, caller.RootBenchmark, caller.ControlID); err != nil {
			w.logger.Error("failed to remove old findings", zap.Error(err), zap.String("benchmark_id", caller.RootBenchmark), zap.String("control_id", caller.ControlID))
			return 0, err
		}
	}

	totalFindingCount := 0
	for _, v := range totalFindingCountMap {
		totalFindingCount += v
	}

	w.logger.Info("Finished job",
		zap.Uint("job_id", j.ID),
		zap.String("query_id", j.ExecutionPlan.Query.ID),
		zap.Stringp("query_id", j.ExecutionPlan.ConnectionID),
	)
	return totalFindingCount, nil
}

type FindingsMultiGetResponse struct {
	Docs []struct {
		Source types.Finding `json:"_source"`
	}
}

func (w *Worker) FetchFindingsNeededHistoryByIDs(ctx context.Context, ids []string) (map[string]types.Finding, error) {
	request := map[string]any{
		"ids": ids,
	}
	query, err := json.Marshal(request)
	if err != nil {
		w.logger.Error("failed to create es query", zap.Error(err))
		return nil, err
	}
	es := w.esClient.ES()
	res, err := es.Mget(
		bytes.NewReader(query),
		es.Mget.WithIndex(types.FindingsIndex),
		es.Mget.WithContext(ctx),
		es.Mget.WithFilterPath(
			"docs._source.es_id",
			"docs._source.history",
			"docs._source.complianceJobID",
			"docs._source.conformanceStatus",
			"docs._source.evaluatedAt",
		),
	)
	defer kaytu.CloseSafe(res)
	if err != nil {
		w.logger.Error("failure while querying es", zap.Error(err))
		return nil, err
	} else if err := kaytu.CheckError(res); err != nil {
		if kaytu.IsIndexNotFoundErr(err) {
			return nil, nil
		}
		b, _ := io.ReadAll(res.Body)
		w.logger.Error("failure while querying es", zap.Error(err), zap.String("response", string(b)))
		return nil, err
	}

	var response FindingsMultiGetResponse
	resBytes, err := io.ReadAll(res.Body)
	if err != nil {
		w.logger.Error("failed to read response", zap.Error(err))
		return nil, fmt.Errorf("read response: %w", err)
	}
	if err := json.Unmarshal(resBytes, &response); err != nil {
		w.logger.Error("failed to unmarshal response", zap.Error(err))
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	findings := make(map[string]types.Finding)
	for _, doc := range response.Docs {
		findings[doc.Source.EsID] = doc.Source
	}
	return findings, nil
}

func (w *Worker) RemoveOldFindings(jobID uint,
	connectionId *string,
	benchmarkID,
	controlID string,
) error {
	ctx := context.Background()
	idx := types.FindingsIndex
	var filters []map[string]any
	mustFilters := make([]map[string]any, 0, 4)
	mustFilters = append(mustFilters, map[string]any{
		"term": map[string]any{
			"benchmarkID": benchmarkID,
		},
	})
	mustFilters = append(mustFilters, map[string]any{
		"term": map[string]any{
			"controlID": controlID,
		},
	})
	if connectionId != nil {
		mustFilters = append(mustFilters, map[string]any{
			"term": map[string]any{
				"connectionID": *connectionId,
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
	defer kaytu.CloseSafe(res)
	if err != nil {
		b, _ := io.ReadAll(res.Body)
		w.logger.Error("failure while deleting es", zap.Error(err), zap.String("benchmark_id", benchmarkID), zap.String("control_id", controlID), zap.String("response", string(b)))
		return err
	} else if err := kaytu.CheckError(res); err != nil {
		if kaytu.IsIndexNotFoundErr(err) {
			return nil
		}
		b, _ := io.ReadAll(res.Body)
		w.logger.Error("failure while querying es", zap.Error(err), zap.String("benchmark_id", benchmarkID), zap.String("control_id", controlID), zap.String("response", string(b)))
		return err
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	return nil
}
