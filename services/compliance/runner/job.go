package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	authApi "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/opengovernance/services/compliance/api"
	inventoryApi "github.com/opengovern/opengovernance/services/inventory/api"

	"github.com/opengovern/og-util/pkg/es"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/og-util/pkg/steampipe"
	"github.com/opengovern/opengovernance/pkg/types"
	complianceApi "github.com/opengovern/opengovernance/services/compliance/api"
	es2 "github.com/opengovern/opengovernance/services/compliance/es"
	"go.uber.org/zap"
)

type Caller struct {
	RootBenchmark      string
	TracksDriftEvents  bool
	ParentBenchmarkIDs []string
	ControlID          string
	ControlSeverity    types.ComplianceResultSeverity
}

type ExecutionPlan struct {
	Callers []Caller
	Query   complianceApi.Query

	IntegrationID *string
	ProviderID    *string
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
	esClient      opengovernance.Client
}

func (w *Worker) Initialize(ctx context.Context, j Job) error {
	providerAccountID := "all"
	if j.ExecutionPlan.ProviderID != nil &&
		*j.ExecutionPlan.ProviderID != "" {
		providerAccountID = *j.ExecutionPlan.ProviderID
	}

	err := w.steampipeConn.SetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyAccountID, providerAccountID)
	if err != nil {
		w.logger.Error("failed to set account id", zap.Error(err))
		return err
	}
	err = w.steampipeConn.SetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyClientType, "compliance")
	if err != nil {
		w.logger.Error("failed to set client type", zap.Error(err))
		return err
	}

	return nil
}

func (w *Worker) RunJob(ctx context.Context, j Job) (int, error) {
	//cutOff := time.Now().AddDate(0, -3, 0)
	//w.logger.Info("Deleting old complianceResults", zap.Uint("job_id", j.ID), zap.Time("cut_off", cutOff))
	//if err := w.handleOldComplianceResultsStateByTime(ctx, cutOff, false); err != nil {
	//	w.logger.Error("failed to delete old complianceResults", zap.Error(err), zap.Uint("job_id", j.ID), zap.Time("cut_off", cutOff))
	//	return 0, err
	//}

	w.logger.Info("Running query",
		zap.Uint("job_id", j.ID),
		zap.String("query_id", j.ExecutionPlan.Query.ID),
		zap.Stringp("integration_id", j.ExecutionPlan.IntegrationID),
		zap.Stringp("provider_id", j.ExecutionPlan.ProviderID),
	)

	if err := w.Initialize(ctx, j); err != nil {
		return 0, err
	}
	defer w.steampipeConn.UnsetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyAccountID)
	defer w.steampipeConn.UnsetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyClientType)
	defer w.steampipeConn.UnsetConfigTableValue(ctx, steampipe.OpenGovernanceConfigKeyResourceCollectionFilters)

	queryParams, err := w.metadataClient.ListQueryParameters(&httpclient.Context{Ctx: ctx, UserRole: authApi.AdminRole})
	if err != nil {
		w.logger.Error("failed to get query parameters", zap.Error(err))
		return 0, err
	}
	queryParamMap := make(map[string]string)
	for _, qp := range queryParams.QueryParameters {
		queryParamMap[qp.Key] = qp.Value
	}

	for _, param := range j.ExecutionPlan.Query.Parameters {
		if _, ok := queryParamMap[param.Key]; !ok && param.Required {
			w.logger.Error("required query parameter not found",
				zap.String("key", param.Key),
				zap.String("query_id", j.ExecutionPlan.Query.ID),
				zap.Stringp("integration_id", j.ExecutionPlan.IntegrationID),
				zap.Uint("job_id", j.ID),
			)
			return 0, fmt.Errorf("required query parameter not found: %s for query: %s", param.Key, j.ExecutionPlan.Query.ID)
		}
		if _, ok := queryParamMap[param.Key]; !ok && !param.Required {
			w.logger.Info("optional query parameter not found",
				zap.String("key", param.Key),
				zap.String("query_id", j.ExecutionPlan.Query.ID),
				zap.Stringp("integration_id", j.ExecutionPlan.IntegrationID),
				zap.Uint("job_id", j.ID),
			)
			queryParamMap[param.Key] = ""
		}
	}
	var res *steampipe.Result

	if j.ExecutionPlan.Query.Engine == api.QueryengineCloudQL || j.ExecutionPlan.Query.Engine == api.QueryEngine_cloudql {
		res, err = w.runSqlWorkerJob(ctx, j, queryParamMap)
	} else if j.ExecutionPlan.Query.Engine == api.QueryEngine_cloudqlRego {
		res, err = w.runRegoWorkerJob(ctx, j, queryParamMap)
	} else {
		res, err = w.runSqlWorkerJob(ctx, j, queryParamMap)
	}

	if err != nil {
		w.logger.Error("failed to get results", zap.Error(err))
		return 0, err
	}

	w.logger.Info("Extracting and pushing to nats",
		zap.Uint("job_id", j.ID),
		zap.Int("res_count", len(res.Data)),
		zap.Int("caller_count", len(j.ExecutionPlan.Callers)),
		zap.Any("res", *res),
		zap.String("query", j.ExecutionPlan.Query.QueryToExecute),
		zap.String("query_id", j.ExecutionPlan.Query.ID),
	)
	totalComplianceResultCountMap := make(map[string]int)

	complianceResults, err := j.ExtractComplianceResults(w.logger, w.benchmarkCache, j.ExecutionPlan.Callers[0], res, j.ExecutionPlan.Query)
	if err != nil {
		return 0, err
	}
	w.logger.Info("Extracted complianceResults", zap.Int("count", len(complianceResults)),
		zap.Uint("job_id", j.ID),
		zap.String("benchmarkID", j.ExecutionPlan.Callers[0].RootBenchmark))

	complianceResultsMap := make(map[string]types.ComplianceResult)
	for i, f := range complianceResults {
		f := f
		keys, idx := f.KeysAndIndex()
		f.EsID = es.HashOf(keys...)
		f.EsIndex = idx
		complianceResults[i] = f
		complianceResultsMap[f.EsID] = f
	}

	filters := make([]opengovernance.BoolFilter, 0)
	filters = append(filters, opengovernance.NewTermFilter("benchmarkID", j.ExecutionPlan.Callers[0].RootBenchmark))
	filters = append(filters, opengovernance.NewTermFilter("controlID", j.ExecutionPlan.Callers[0].ControlID))
	for _, parentBenchmarkID := range []string{j.ExecutionPlan.Callers[0].RootBenchmark} {
		filters = append(filters, opengovernance.NewTermFilter("parentBenchmarks", parentBenchmarkID))
	}
	filters = append(filters, opengovernance.NewRangeFilter("complianceJobID", "", "", fmt.Sprintf("%d", j.ID), ""))
	if j.ExecutionPlan.IntegrationID != nil {
		filters = append(filters, opengovernance.NewTermFilter("integrationID", *j.ExecutionPlan.IntegrationID))
	}

	newComplianceResults := make([]types.ComplianceResult, 0, len(complianceResults))
	complianceResultDriftEvents := make([]types.ComplianceResultDriftEvent, 0, len(complianceResults))

	trackDrifts := false
	for _, f := range j.ExecutionPlan.Callers {
		if f.TracksDriftEvents {
			trackDrifts = true
			break
		}
	}

	filtersJSON, _ := json.Marshal(filters)
	w.logger.Info("Old complianceResult query", zap.Int("length", len(complianceResults)), zap.String("filters", string(filtersJSON)))
	paginator, err := es2.NewComplianceResultPaginator(w.esClient, types.ComplianceResultsIndex, filters, nil, nil)
	if err != nil {
		w.logger.Error("failed to create paginator", zap.Error(err))
		return 0, err
	}
	closePaginator := func() {
		if err := paginator.Close(ctx); err != nil {
			w.logger.Error("failed to close paginator", zap.Error(err))
		}
	}

	for paginator.HasNext() {
		oldComplianceResults, err := paginator.NextPage(ctx)
		if err != nil {
			w.logger.Error("failed to get next page", zap.Error(err))
			closePaginator()
			return 0, err
		}

		w.logger.Info("Old complianceResult", zap.Int("length", len(oldComplianceResults)))
		for _, f := range oldComplianceResults {
			f := f
			newComplianceResult, ok := complianceResultsMap[f.EsID]
			if !ok {
				if f.StateActive {
					f := f
					f.StateActive = false
					f.LastUpdatedAt = j.CreatedAt.UnixMilli()
					f.RunnerID = j.ID
					f.ComplianceJobID = j.ParentJobID
					f.EvaluatedAt = j.CreatedAt.UnixMilli()
					reason := fmt.Sprintf("Engine didn't found resource %s in the query result", f.PlatformResourceID)
					f.Reason = reason
					fs := types.ComplianceResultDriftEvent{
						ComplianceResultEsID:     f.EsID,
						ParentComplianceJobID:    j.ParentJobID,
						ComplianceJobID:          j.ID,
						PreviousComplianceStatus: f.ComplianceStatus,
						ComplianceStatus:         f.ComplianceStatus,
						PreviousStateActive:      true,
						StateActive:              f.StateActive,
						EvaluatedAt:              j.CreatedAt.UnixMilli(),
						Reason:                   reason,

						BenchmarkID:        f.BenchmarkID,
						ControlID:          f.ControlID,
						IntegrationID:      f.IntegrationID,
						IntegrationType:    f.IntegrationType,
						Severity:           f.Severity,
						PlatformResourceID: f.PlatformResourceID,
						ResourceID:         f.ResourceID,
						ResourceType:       f.ResourceType,
					}
					keys, idx := fs.KeysAndIndex()
					fs.EsID = es.HashOf(keys...)
					fs.EsIndex = idx

					w.logger.Info("ComplianceResult is not found in the query result setting it to inactive", zap.Any("complianceResult", f), zap.Any("event", fs))
					if trackDrifts {
						complianceResultDriftEvents = append(complianceResultDriftEvents, fs)
					}
					newComplianceResults = append(newComplianceResults, f)
				} else {
					w.logger.Info("Old complianceResult found, it's inactive. doing nothing", zap.Any("complianceResult", f))
				}
				continue
			}

			if (f.ComplianceStatus != newComplianceResult.ComplianceStatus) ||
				(f.StateActive != newComplianceResult.StateActive) {
				newComplianceResult.LastUpdatedAt = j.CreatedAt.UnixMilli()
				newComplianceResult.RunnerID = j.ID
				newComplianceResult.ComplianceJobID = j.ParentJobID
				fs := types.ComplianceResultDriftEvent{
					ComplianceResultEsID:     f.EsID,
					ParentComplianceJobID:    j.ParentJobID,
					ComplianceJobID:          j.ID,
					PreviousComplianceStatus: f.ComplianceStatus,
					ComplianceStatus:         newComplianceResult.ComplianceStatus,
					PreviousStateActive:      f.StateActive,
					StateActive:              newComplianceResult.StateActive,
					EvaluatedAt:              j.CreatedAt.UnixMilli(),
					Reason:                   newComplianceResult.Reason,

					BenchmarkID:        newComplianceResult.BenchmarkID,
					ControlID:          newComplianceResult.ControlID,
					IntegrationID:      newComplianceResult.IntegrationID,
					IntegrationType:    newComplianceResult.IntegrationType,
					Severity:           newComplianceResult.Severity,
					PlatformResourceID: newComplianceResult.PlatformResourceID,
					ResourceID:         newComplianceResult.ResourceID,
					ResourceType:       newComplianceResult.ResourceType,
				}
				keys, idx := fs.KeysAndIndex()
				fs.EsID = es.HashOf(keys...)
				fs.EsIndex = idx

				w.logger.Info("ComplianceResult status changed", zap.Any("old", f), zap.Any("new", newComplianceResult), zap.Any("event", fs))
				if trackDrifts {
					complianceResultDriftEvents = append(complianceResultDriftEvents, fs)
				}
			} else {
				w.logger.Info("ComplianceResult status didn't change. doing nothing", zap.Any("complianceResult", newComplianceResult))
				newComplianceResult.LastUpdatedAt = f.LastUpdatedAt
				newComplianceResult.RunnerID = j.ID
				newComplianceResult.ComplianceJobID = j.ParentJobID
			}

			newComplianceResults = append(newComplianceResults, newComplianceResult)
			delete(complianceResultsMap, f.EsID)
			delete(complianceResultsMap, newComplianceResult.EsID)
		}
	}
	closePaginator()
	for _, newComplianceResult := range complianceResultsMap {
		newComplianceResult.LastUpdatedAt = j.CreatedAt.UnixMilli()
		newComplianceResult.RunnerID = j.ID
		newComplianceResult.ComplianceJobID = j.ParentJobID
		fs := types.ComplianceResultDriftEvent{
			ComplianceResultEsID:  newComplianceResult.EsID,
			ParentComplianceJobID: j.ParentJobID,
			ComplianceJobID:       j.ID,
			ComplianceStatus:      newComplianceResult.ComplianceStatus,
			StateActive:           newComplianceResult.StateActive,
			EvaluatedAt:           j.CreatedAt.UnixMilli(),
			Reason:                newComplianceResult.Reason,

			BenchmarkID:        newComplianceResult.BenchmarkID,
			ControlID:          newComplianceResult.ControlID,
			IntegrationID:      newComplianceResult.IntegrationID,
			IntegrationType:    newComplianceResult.IntegrationType,
			Severity:           newComplianceResult.Severity,
			PlatformResourceID: newComplianceResult.PlatformResourceID,
			ResourceID:         newComplianceResult.ResourceID,
			ResourceType:       newComplianceResult.ResourceType,
		}
		keys, idx := fs.KeysAndIndex()
		fs.EsID = es.HashOf(keys...)
		fs.EsIndex = idx

		w.logger.Info("New complianceResult", zap.Any("complianceResult", newComplianceResult), zap.Any("event", fs))
		if trackDrifts {
			complianceResultDriftEvents = append(complianceResultDriftEvents, fs)
		}
		newComplianceResults = append(newComplianceResults, newComplianceResult)
	}

	var docs []es.Doc
	if trackDrifts {
		for _, fs := range complianceResultDriftEvents {
			keys, idx := fs.KeysAndIndex()
			fs.EsID = es.HashOf(keys...)
			fs.EsIndex = idx

			docs = append(docs, fs)
		}
	}
	for _, f := range newComplianceResults {
		keys, idx := f.KeysAndIndex()
		f.EsID = es.HashOf(keys...)
		f.EsIndex = idx
		docs = append(docs, f)
	}
	mapKey := strings.Builder{}
	mapKey.WriteString(j.ExecutionPlan.Callers[0].RootBenchmark)
	mapKey.WriteString("$$")
	mapKey.WriteString(j.ExecutionPlan.Callers[0].ControlID)
	for _, parentBenchmarkID := range []string{j.ExecutionPlan.Callers[0].RootBenchmark} {
		mapKey.WriteString("$$")
		mapKey.WriteString(parentBenchmarkID)
	}
	if _, ok := totalComplianceResultCountMap[mapKey.String()]; !ok {
		totalComplianceResultCountMap[mapKey.String()] = len(newComplianceResults)
	}

	if _, err := w.sinkClient.Ingest(&httpclient.Context{Ctx: ctx, UserRole: authApi.AdminRole}, docs); err != nil {
		w.logger.Error("failed to send complianceResults", zap.Error(err), zap.String("benchmark_id", j.ExecutionPlan.Callers[0].RootBenchmark),
			zap.String("control_id", j.ExecutionPlan.Callers[0].ControlID))
		return 0, err
	}

	totalComplianceResultCount := 0
	for _, v := range totalComplianceResultCountMap {
		totalComplianceResultCount += v
	}

	w.logger.Info("Finished job",
		zap.Uint("job_id", j.ID),
		zap.String("query_id", j.ExecutionPlan.Query.ID),
		zap.Stringp("query_id", j.ExecutionPlan.IntegrationID),
	)
	return totalComplianceResultCount, nil
}

func (w *Worker) runSqlWorkerJob(ctx context.Context, j Job, queryParamMap map[string]string) (*steampipe.Result, error) {
	queryTemplate, err := template.New(j.ExecutionPlan.Query.ID).Parse(j.ExecutionPlan.Query.QueryToExecute)
	if err != nil {
		w.logger.Error("failed to parse query template", zap.Error(err))
		return nil, err
	}
	var queryOutput bytes.Buffer
	if err := queryTemplate.Execute(&queryOutput, queryParamMap); err != nil {
		w.logger.Error("failed to execute query template",
			zap.Error(err),
			zap.String("query_id", j.ExecutionPlan.Query.ID),
			zap.Stringp("integration_id", j.ExecutionPlan.IntegrationID),
			zap.Uint("job_id", j.ID),
		)
		return nil, fmt.Errorf("failed to execute query template: %w for query: %s", err, j.ExecutionPlan.Query.ID)
	}

	w.logger.Info("runSqlWorkerJob QueryOutput",
		zap.Uint("job_id", j.ID),
		zap.Int("caller_count", len(j.ExecutionPlan.Callers)),
		zap.String("query", j.ExecutionPlan.Query.QueryToExecute),
		zap.String("query_id", j.ExecutionPlan.Query.ID),
		zap.String("query", queryOutput.String()))
	res, err := w.steampipeConn.QueryAll(ctx, queryOutput.String())
	if err != nil {
		w.logger.Error("failed to run query", zap.Error(err), zap.String("query_id", j.ExecutionPlan.Query.ID), zap.Stringp("integration_id", j.ExecutionPlan.IntegrationID))
		return nil, err
	}

	return res, nil
}

func (w *Worker) runRegoWorkerJob(ctx context.Context, j Job, queryParamMap map[string]string) (*steampipe.Result, error) {
	ctx2 := &httpclient.Context{Ctx: ctx, UserRole: authApi.AdminRole}
	ctx2.Ctx = ctx
	var engine inventoryApi.QueryEngine
	engine = inventoryApi.QueryEngine_cloudqlRego
	queryResponse, err := w.inventoryClient.RunQuery(ctx2, inventoryApi.RunQueryRequest{
		Page: inventoryApi.Page{
			No:   1,
			Size: 1000,
		},
		Engine: &engine,
		Query:  &j.ExecutionPlan.Query.QueryToExecute,
		Sorts:  nil,
	})
	if err != nil {
		return nil, err
	}

	results := &steampipe.Result{
		Headers: queryResponse.Headers,
		Data:    queryResponse.Result,
	}
	return results, nil
}

type ComplianceResultsMultiGetResponse struct {
	Docs []struct {
		Source types.ComplianceResult `json:"_source"`
	} `json:"docs"`
}
