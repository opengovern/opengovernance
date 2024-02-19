package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	complianceApi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	es2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/es"
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
	RetryCount  int
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
		findings, err := j.ExtractFindings(w.logger, w.benchmarkCache, caller, res, j.ExecutionPlan.Query)
		if err != nil {
			return 0, err
		}
		w.logger.Info("Extracted findings", zap.Int("count", len(findings)),
			zap.Uint("job_id", j.ID),
			zap.String("benchmarkID", caller.RootBenchmark))

		findingsMap := make(map[string]types.Finding)
		for i, f := range findings {
			f := f
			keys, idx := f.KeysAndIndex()
			f.EsID = es.HashOf(keys...)
			f.EsIndex = idx
			findings[i] = f
			findingsMap[f.EsID] = f
		}

		filters := make([]kaytu.BoolFilter, 0)
		filters = append(filters, kaytu.NewTermFilter("benchmarkID", caller.RootBenchmark))
		filters = append(filters, kaytu.NewTermFilter("controlID", caller.ControlID))
		for _, parentBenchmarkID := range caller.ParentBenchmarkIDs {
			filters = append(filters, kaytu.NewTermFilter("parentBenchmarks", parentBenchmarkID))
		}
		filters = append(filters, kaytu.NewRangeFilter("complianceJobID", "", "", fmt.Sprintf("%d", j.ID), ""))
		if j.ExecutionPlan.ConnectionID != nil {
			filters = append(filters, kaytu.NewTermFilter("connectionID", *j.ExecutionPlan.ConnectionID))
		} else {
			filters = append(filters, kaytu.NewTermFilter("connectionID", "all"))
		}

		newFindings := make([]types.Finding, 0, len(findings))
		findingsEvents := make([]types.FindingEvent, 0, len(findings))
		paginator, err := es2.NewFindingPaginator(w.esClient, types.FindingsIndex, filters, nil, nil)
		if err != nil {
			w.logger.Error("failed to create paginator", zap.Error(err))
			return 0, err
		}
		closePaginator := func() {
			if err := paginator.Close(context.Background()); err != nil {
				w.logger.Error("failed to close paginator", zap.Error(err))
			}
		}
		for paginator.HasNext() {
			oldFindings, err := paginator.NextPage(ctx)
			if err != nil {
				w.logger.Error("failed to get next page", zap.Error(err))
				closePaginator()
				return 0, err
			}

			for _, f := range oldFindings {
				f := f
				newFinding, ok := findingsMap[f.EsID]
				if !ok {
					if f.StateActive {
						f := f
						f.StateActive = false
						f.LastTransition = j.CreatedAt.UnixMilli()
						f.ComplianceJobID = j.ID
						f.ParentComplianceJobID = j.ParentJobID
						f.EvaluatedAt = j.CreatedAt.UnixMilli()
						reason := fmt.Sprintf("Engine didn't found resource %s in the query result", f.KaytuResourceID)
						f.Reason = reason
						fs := types.FindingEvent{
							FindingEsID:               f.EsID,
							ParentComplianceJobID:     j.ParentJobID,
							ComplianceJobID:           j.ID,
							PreviousConformanceStatus: f.ConformanceStatus,
							ConformanceStatus:         f.ConformanceStatus,
							PreviousStateActive:       true,
							StateActive:               f.StateActive,
							EvaluatedAt:               j.CreatedAt.UnixMilli(),
							Reason:                    reason,

							BenchmarkID:               f.BenchmarkID,
							ControlID:                 f.ControlID,
							ConnectionID:              f.ConnectionID,
							Connector:                 f.Connector,
							Severity:                  f.Severity,
							KaytuResourceID:           f.KaytuResourceID,
							ResourceID:                f.ResourceID,
							ResourceType:              f.ResourceType,
							ParentBenchmarkReferences: f.ParentBenchmarkReferences,
						}
						keys, idx := fs.KeysAndIndex()
						fs.EsID = es.HashOf(keys...)
						fs.EsIndex = idx

						w.logger.Info("Finding is not found in the query result setting it to inactive", zap.Any("finding", f), zap.Any("event", fs))
						findingsEvents = append(findingsEvents, fs)
						newFindings = append(newFindings, f)
					}
					continue
				}

				if (f.ConformanceStatus != newFinding.ConformanceStatus) ||
					(f.StateActive != newFinding.StateActive) {
					newFinding.LastTransition = j.CreatedAt.UnixMilli()
					fs := types.FindingEvent{
						FindingEsID:               f.EsID,
						ParentComplianceJobID:     j.ParentJobID,
						ComplianceJobID:           j.ID,
						PreviousConformanceStatus: f.ConformanceStatus,
						ConformanceStatus:         newFinding.ConformanceStatus,
						PreviousStateActive:       f.StateActive,
						StateActive:               newFinding.StateActive,
						EvaluatedAt:               j.CreatedAt.UnixMilli(),
						Reason:                    newFinding.Reason,

						BenchmarkID:               newFinding.BenchmarkID,
						ControlID:                 newFinding.ControlID,
						ConnectionID:              newFinding.ConnectionID,
						Connector:                 newFinding.Connector,
						Severity:                  newFinding.Severity,
						KaytuResourceID:           newFinding.KaytuResourceID,
						ResourceID:                newFinding.ResourceID,
						ResourceType:              newFinding.ResourceType,
						ParentBenchmarkReferences: newFinding.ParentBenchmarkReferences,
					}
					keys, idx := fs.KeysAndIndex()
					fs.EsID = es.HashOf(keys...)
					fs.EsIndex = idx

					w.logger.Info("Finding status changed", zap.Any("old", f), zap.Any("new", newFinding), zap.Any("event", fs))
					findingsEvents = append(findingsEvents, fs)
				} else {
					newFinding.LastTransition = f.LastTransition
				}

				newFindings = append(newFindings, newFinding)
				delete(findingsMap, f.EsID)
				delete(findingsMap, newFinding.EsID)
			}
		}
		closePaginator()
		for _, newFinding := range findingsMap {
			newFinding.LastTransition = j.CreatedAt.UnixMilli()
			fs := types.FindingEvent{
				FindingEsID:           newFinding.EsID,
				ParentComplianceJobID: j.ParentJobID,
				ComplianceJobID:       j.ID,
				ConformanceStatus:     newFinding.ConformanceStatus,
				StateActive:           newFinding.StateActive,
				EvaluatedAt:           j.CreatedAt.UnixMilli(),
				Reason:                newFinding.Reason,

				BenchmarkID:               newFinding.BenchmarkID,
				ControlID:                 newFinding.ControlID,
				ConnectionID:              newFinding.ConnectionID,
				Connector:                 newFinding.Connector,
				Severity:                  newFinding.Severity,
				KaytuResourceID:           newFinding.KaytuResourceID,
				ResourceID:                newFinding.ResourceID,
				ResourceType:              newFinding.ResourceType,
				ParentBenchmarkReferences: newFinding.ParentBenchmarkReferences,
			}
			keys, idx := fs.KeysAndIndex()
			fs.EsID = es.HashOf(keys...)
			fs.EsIndex = idx

			w.logger.Info("New finding", zap.Any("finding", newFinding), zap.Any("event", fs))
			findingsEvents = append(findingsEvents, fs)
			newFindings = append(newFindings, newFinding)
		}

		var docs []es.Doc
		for _, fs := range findingsEvents {
			keys, idx := fs.KeysAndIndex()
			fs.EsID = es.HashOf(keys...)
			fs.EsIndex = idx

			docs = append(docs, fs)
		}
		for _, f := range newFindings {
			keys, idx := f.KeysAndIndex()
			f.EsID = es.HashOf(keys...)
			f.EsIndex = idx
			docs = append(docs, f)
		}
		mapKey := strings.Builder{}
		mapKey.WriteString(caller.RootBenchmark)
		mapKey.WriteString("$$")
		mapKey.WriteString(caller.ControlID)
		for _, parentBenchmarkID := range caller.ParentBenchmarkIDs {
			mapKey.WriteString("$$")
			mapKey.WriteString(parentBenchmarkID)
		}
		if _, ok := totalFindingCountMap[mapKey.String()]; !ok {
			totalFindingCountMap[mapKey.String()] = len(newFindings)
		}

		if err := pipeline.SendToPipeline(w.config.ElasticSearch.IngestionEndpoint, docs); err != nil {
			w.logger.Error("failed to send findings", zap.Error(err), zap.String("benchmark_id", caller.RootBenchmark), zap.String("control_id", caller.ControlID))
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
	} `json:"docs"`
}

func (w *Worker) setOldFindingsInactive(jobID uint,
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
	mustFilters = append(mustFilters, map[string]any{
		"range": map[string]any{
			"complianceJobID": map[string]any{
				"lt": jobID,
			},
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
	request["doc"] = map[string]any{
		"stateActive": false,
	}

	query, err := json.Marshal(request)
	if err != nil {
		return err
	}

	es := w.esClient.ES()
	res, err := es.UpdateByQuery(
		[]string{idx},
		es.UpdateByQuery.WithContext(ctx),
		es.UpdateByQuery.WithBody(bytes.NewReader(query)),
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
