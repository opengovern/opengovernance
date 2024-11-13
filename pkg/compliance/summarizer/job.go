package summarizer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"strings"

	es2 "github.com/opengovern/og-util/pkg/es"
	"github.com/opengovern/opengovernance/pkg/compliance/es"
	types2 "github.com/opengovern/opengovernance/pkg/compliance/summarizer/types"
	es3 "github.com/opengovern/opengovernance/pkg/describe/es"
	inventoryApi "github.com/opengovern/opengovernance/pkg/inventory/api"
	"github.com/opengovern/opengovernance/pkg/types"
	integrationApi "github.com/opengovern/opengovernance/services/integration/api/models"
	"go.uber.org/zap"
)

func (w *Worker) RunJob(ctx context.Context, j types2.Job) error {
	w.logger.Info("Running summarizer",
		zap.Uint("job_id", j.ID),
		zap.String("benchmark_id", j.BenchmarkID),
	)

	// We have to sort by platformResourceID to be able to optimize memory usage for resourceFinding generations
	// this way as soon as paginator switches to next resource we can send the previous resource to the queue and free up memory
	paginator, err := es.NewComplianceResultPaginator(w.esClient, types.ComplianceResultsIndex, []opengovernance.BoolFilter{
		opengovernance.NewTermFilter("stateActive", "true"),
	}, nil, []map[string]any{
		{"platformResourceID": "asc"},
		{"resourceType": "asc"},
	})
	if err != nil {
		return err
	}
	defer func() {
		if err := paginator.Close(ctx); err != nil {
			w.logger.Error("failed to close paginator", zap.Error(err))
		}
	}()

	w.logger.Info("ComplianceResultsIndex paginator ready")

	jd := types2.JobDocs{
		BenchmarkSummary: types2.BenchmarkSummary{
			BenchmarkID:      j.BenchmarkID,
			JobID:            j.ID,
			EvaluatedAtEpoch: j.CreatedAt.Unix(),
			Integrations: types2.BenchmarkSummaryResult{
				BenchmarkResult: types2.ResultGroup{
					Result: types2.Result{
						QueryResult:    map[types.ComplianceStatus]int{},
						SeverityResult: map[types.ComplianceResultSeverity]int{},
						SecurityScore:  0,
					},
					ResourceTypes: map[string]types2.Result{},
					Controls:      map[string]types2.ControlResult{},
				},
				Integrations: map[string]types2.ResultGroup{},
			},
			ResourceCollections: map[string]types2.BenchmarkSummaryResult{},
		},
		ResourcesFindings:       make(map[string]types.ResourceFinding),
		ResourcesFindingsIsDone: make(map[string]bool),

		ResourceCollectionCache: map[string]inventoryApi.ResourceCollection{},
		IntegrationCache:        map[string]integrationApi.Integration{},
	}

	resourceCollections, err := w.inventoryClient.ListResourceCollections(&httpclient.Context{Ctx: ctx, UserRole: api.AdminRole})
	if err != nil {
		w.logger.Error("failed to list resource collections", zap.Error(err))
		return err
	}
	for _, rc := range resourceCollections {
		rc := rc
		jd.ResourceCollectionCache[rc.ID] = rc
	}

	integrations, err := w.integrationClient.ListIntegrations(&httpclient.Context{Ctx: ctx, UserRole: api.AdminRole}, nil)
	if err != nil {
		w.logger.Error("failed to list integrations", zap.Error(err))
		return err
	}
	for _, c := range integrations.Integrations {
		c := c
		// use provider id instead of opengovernance id because we need that to check resource collections
		jd.IntegrationCache[strings.ToLower(c.ProviderID)] = c
	}

	for page := 1; paginator.HasNext(); page++ {
		w.logger.Info("Next page", zap.Int("page", page))
		page, err := paginator.NextPage(ctx)
		if err != nil {
			w.logger.Error("failed to fetch next page", zap.Error(err))
			return err
		}

		platformResourceIDs := make([]string, 0, len(page))
		for _, f := range page {
			platformResourceIDs = append(platformResourceIDs, f.PlatformResourceID)
		}

		lookupResourcesMap, err := es.FetchLookupByResourceIDBatch(ctx, w.esClient, platformResourceIDs)
		if err != nil {
			w.logger.Error("failed to fetch lookup resources", zap.Error(err))
			return err
		}

		w.logger.Info("resource lookup result", zap.Any("platformResourceIDs", platformResourceIDs),
			zap.Any("lookupResourcesMap", lookupResourcesMap))
		w.logger.Info("page size", zap.Int("pageSize", len(page)))
		for _, f := range page {
			var resource *es2.LookupResource
			potentialResources := lookupResourcesMap[f.PlatformResourceID]
			for _, r := range potentialResources {
				r := r
				w.logger.Info("potential resources", zap.Any("potentialResources", potentialResources),
					zap.String("f.ResourceType", f.ResourceType), zap.String("r.ResourceType", r.ResourceType))
				if strings.ToLower(r.ResourceType) == strings.ToLower(f.ResourceType) {
					resource = &r
					break
				}
			}

			jd.AddComplianceResult(w.logger, j, f, resource)
		}

		var docs []es2.Doc
		for resourceIdType, isReady := range jd.ResourcesFindingsIsDone {
			if !isReady {
				continue
			}
			resourceFinding := jd.SummarizeResourceFinding(w.logger, jd.ResourcesFindings[resourceIdType])
			keys, idx := resourceFinding.KeysAndIndex()
			resourceFinding.EsID = es2.HashOf(keys...)
			resourceFinding.EsIndex = idx
			docs = append(docs, resourceFinding)
			delete(jd.ResourcesFindings, resourceIdType)
			delete(jd.ResourcesFindingsIsDone, resourceIdType)
		}
		w.logger.Info("Sending resource finding docs", zap.Int("docCount", len(docs)))

		if _, err := w.esSinkClient.Ingest(&httpclient.Context{Ctx: ctx, UserRole: api.AdminRole}, docs); err != nil {
			w.logger.Error("failed to send to ingest", zap.Error(err))
			return err
		}
	}

	err = paginator.Close(ctx)
	if err != nil {
		return err
	}

	w.logger.Info("Starting to summarizer",
		zap.Uint("job_id", j.ID),
		zap.String("benchmark_id", j.BenchmarkID),
	)

	jd.Summarize(w.logger)

	w.logger.Info("Summarize done", zap.Any("summary", jd))

	keys, idx := jd.BenchmarkSummary.KeysAndIndex()
	jd.BenchmarkSummary.EsID = es2.HashOf(keys...)
	jd.BenchmarkSummary.EsIndex = idx

	docs := make([]es2.Doc, 0, len(jd.ResourcesFindings)+1)
	docs = append(docs, jd.BenchmarkSummary)
	resourceIds := make([]string, 0, len(jd.ResourcesFindings))
	for resourceId, rf := range jd.ResourcesFindings {
		resourceIds = append(resourceIds, resourceId)
		keys, idx := rf.KeysAndIndex()
		rf.EsID = es2.HashOf(keys...)
		rf.EsIndex = idx
		docs = append(docs, rf)
	}
	if _, err := w.esSinkClient.Ingest(&httpclient.Context{Ctx: ctx, UserRole: api.AdminRole}, docs); err != nil {
		w.logger.Error("failed to send to ingest", zap.Error(err))
		return err
	}

	// Delete old resource findings
	if len(resourceIds) > 0 {
		err = w.deleteOldResourceFindings(ctx, j, resourceIds)
		if err != nil {
			w.logger.Error("failed to delete old resource findings", zap.Error(err))
			return err
		}
	}

	w.logger.Info("Deleting compliance results and resource findings of removed integrations", zap.String("benchmark_id", j.BenchmarkID), zap.Uint("job_id", j.ID))

	currentInregrations, err := w.integrationClient.ListIntegrations(&httpclient.Context{Ctx: ctx, UserRole: api.AdminRole}, nil)
	if err != nil {
		w.logger.Error("failed to list integrations", zap.Error(err), zap.String("benchmark_id", j.BenchmarkID), zap.Uint("job_id", j.ID))
		return err
	}
	currentIntegrationIds := make([]string, 0, len(currentInregrations.Integrations))
	for _, i := range currentInregrations.Integrations {
		currentIntegrationIds = append(currentIntegrationIds, i.IntegrationID)
	}

	err = w.deleteComplianceResultsAndResourceFindingsOfRemovedIntegrations(ctx, j, currentIntegrationIds)
	if err != nil {
		w.logger.Error("failed to delete compliance results and resource findings of removed integrations", zap.Error(err), zap.String("benchmark_id", j.BenchmarkID), zap.Uint("job_id", j.ID))
		return err
	}

	w.logger.Info("Finished summarizer",
		zap.Uint("job_id", j.ID),
		zap.String("benchmark_id", j.BenchmarkID),
		zap.Int("resource_count", len(jd.ResourcesFindings)),
	)
	return nil
}

func (w *Worker) deleteOldResourceFindings(ctx context.Context, j types2.Job, currentResourceIds []string) error {
	// Delete old resource findings
	filters := make([]opengovernance.BoolFilter, 0, 2)
	filters = append(filters, opengovernance.NewBoolMustNotFilter(opengovernance.NewTermsFilter("platformResourceID", currentResourceIds)))
	filters = append(filters, opengovernance.NewRangeFilter("jobId", "", "", fmt.Sprintf("%d", j.ID), ""))

	root := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		},
	}
	rootJson, err := json.Marshal(root)
	if err != nil {
		w.logger.Error("failed to marshal root", zap.Error(err))
		return err
	}

	task := es3.DeleteTask{
		DiscoveryJobID: j.ID,
		IntegrationID:  j.BenchmarkID,
		ResourceType:   "resource-finding",
		TaskType:       es3.DeleteTaskTypeQuery,
		Query:          string(rootJson),
		QueryIndex:     types.ResourceFindingsIndex,
	}

	keys, idx := task.KeysAndIndex()
	task.EsID = es2.HashOf(keys...)
	task.EsIndex = idx
	if _, err := w.esSinkClient.Ingest(&httpclient.Context{Ctx: ctx, UserRole: api.AdminRole}, []es2.Doc{task}); err != nil {
		w.logger.Error("failed to send delete message to elastic", zap.Error(err))
		return err
	}

	return nil
}

func (w *Worker) deleteComplianceResultsAndResourceFindingsOfRemovedIntegrations(ctx context.Context, j types2.Job, currentIntegrationIds []string) error {
	// Delete compliance results
	filters := make([]opengovernance.BoolFilter, 0, 2)
	filters = append(filters, opengovernance.NewBoolMustNotFilter(opengovernance.NewTermsFilter("integrationID", currentIntegrationIds)))
	filters = append(filters, opengovernance.NewRangeFilter("jobId", "", "", fmt.Sprintf("%d", j.ID), ""))

	root := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		},
	}
	rootJson, err := json.Marshal(root)
	if err != nil {
		w.logger.Error("failed to marshal root", zap.Error(err))
		return err
	}

	task := es3.DeleteTask{
		DiscoveryJobID: j.ID,
		IntegrationID:  j.BenchmarkID,
		ResourceType:   "compliance-result-old-integrations-removal",
		TaskType:       es3.DeleteTaskTypeQuery,
		Query:          string(rootJson),
		QueryIndex:     types.ComplianceResultsIndex,
	}

	keys, idx := task.KeysAndIndex()
	task.EsID = es2.HashOf(keys...)
	task.EsIndex = idx
	if _, err := w.esSinkClient.Ingest(&httpclient.Context{Ctx: ctx, UserRole: api.AdminRole}, []es2.Doc{task}); err != nil {
		w.logger.Error("failed to send delete message to elastic", zap.Error(err))
		return err
	}

	// Delete resource findings
	filters = make([]opengovernance.BoolFilter, 0, 2)
	filters = append(filters, opengovernance.NewBoolMustNotFilter(opengovernance.NewTermsFilter("integrationID", currentIntegrationIds)))
	filters = append(filters, opengovernance.NewRangeFilter("jobId", "", "", fmt.Sprintf("%d", j.ID), ""))

	root = map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		},
	}
	rootJson, err = json.Marshal(root)
	if err != nil {
		w.logger.Error("failed to marshal root", zap.Error(err))
		return err
	}

	task = es3.DeleteTask{
		DiscoveryJobID: j.ID,
		IntegrationID:  j.BenchmarkID,
		ResourceType:   "resource-finding-old-integrations-removal",
		TaskType:       es3.DeleteTaskTypeQuery,
		Query:          string(rootJson),
		QueryIndex:     types.ResourceFindingsIndex,
	}

	keys, idx = task.KeysAndIndex()
	task.EsID = es2.HashOf(keys...)
	task.EsIndex = idx
	if _, err := w.esSinkClient.Ingest(&httpclient.Context{Ctx: ctx, UserRole: api.AdminRole}, []es2.Doc{task}); err != nil {
		w.logger.Error("failed to send delete message to elastic", zap.Error(err))
		return err
	}

	return nil
}
