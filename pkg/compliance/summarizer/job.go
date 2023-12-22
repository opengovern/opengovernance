package summarizer

import (
	"context"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/es"
	types2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer/types"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	es2 "github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/pipeline"
	"go.uber.org/zap"
	"strings"
	"time"
)

type Job struct {
	ID          uint
	BenchmarkID string
	CreatedAt   time.Time
}

func (w *Worker) RunJob(j Job) error {
	ctx := context.Background()

	w.logger.Info("Running summarizer",
		zap.Uint("job_id", j.ID),
		zap.String("benchmark_id", j.BenchmarkID),
	)

	paginator, err := es.NewFindingPaginator(w.esClient, types.FindingsIndex, []kaytu.BoolFilter{
		kaytu.NewTermFilter("parentBenchmarks", j.BenchmarkID),
	}, nil)
	if err != nil {
		return err
	}

	w.logger.Info("FindingsIndex paginator ready")

	bs := types2.BenchmarkSummary{
		BenchmarkID:      j.BenchmarkID,
		JobID:            j.ID,
		EvaluatedAtEpoch: j.CreatedAt.Unix(),
		Connections: types2.BenchmarkSummaryResult{
			BenchmarkResult: types2.ResultGroup{
				Result: types2.Result{
					QueryResult:    map[types.ConformanceStatus]int{},
					SeverityResult: map[types.FindingSeverity]int{},
					SecurityScore:  0,
				},
				ResourceTypes: map[string]types2.Result{},
				Controls:      map[string]types2.ControlResult{},
			},
			Connections: map[string]types2.ResultGroup{},
		},
		ResourceCollections: map[string]types2.BenchmarkSummaryResult{},

		ResourceCollectionCache: map[string]inventoryApi.ResourceCollection{},
		ConnectionCache:         map[string]onboardApi.Connection{},
	}

	resourceCollections, err := w.inventoryClient.ListResourceCollections(&httpclient.Context{UserRole: api.InternalRole})
	if err != nil {
		w.logger.Error("failed to list resource collections", zap.Error(err))
		return err
	}
	for _, rc := range resourceCollections {
		rc := rc
		bs.ResourceCollectionCache[rc.ID] = rc
	}

	connections, err := w.onboardClient.ListSources(&httpclient.Context{UserRole: api.InternalRole}, nil)
	if err != nil {
		w.logger.Error("failed to list connections", zap.Error(err))
		return err
	}
	for _, c := range connections {
		c := c
		// use provider id instead of kaytu id because we need that to check resource collections
		bs.ConnectionCache[strings.ToLower(c.ConnectionID)] = c
	}

	for page := 1; paginator.HasNext(); page++ {
		w.logger.Info("Next page", zap.Int("page", page))
		page, err := paginator.NextPage(ctx)
		if err != nil {
			w.logger.Error("failed to fetch next page", zap.Error(err))
			return err
		}

		resourceIds := make([]string, 0, len(page))
		for _, f := range page {
			resourceIds = append(resourceIds, f.KaytuResourceID)
		}

		lookupResources, err := es.FetchLookupByResourceIDBatch(w.esClient, resourceIds)
		if err != nil {
			w.logger.Error("failed to fetch lookup resources", zap.Error(err))
			return err
		}
		lookupResourcesMap := make(map[string]*es2.LookupResource)
		for _, r := range lookupResources.Hits.Hits {
			r := r
			lookupResourcesMap[r.Source.ResourceID] = &r.Source
		}

		w.logger.Info("page size", zap.Int("pageSize", len(page)))
		for _, f := range page {
			bs.AddFinding(w.logger, f, lookupResourcesMap[f.KaytuResourceID])
		}
	}

	w.logger.Info("Starting to summarizer",
		zap.Uint("job_id", j.ID),
		zap.String("benchmark_id", j.BenchmarkID),
	)

	bs.Summarize()

	w.logger.Info("Summarize done", zap.Any("summary", bs))

	if w.config.ElasticSearch.IsOpenSearch {
		keys, idx := bs.KeysAndIndex()
		bs.EsID = kafka.HashOf(keys...)
		bs.EsIndex = idx

		if err := pipeline.SendToPipeline(w.config.ElasticSearch.IngestionEndpoint, []kafka.Doc{bs}); err != nil {
			return err
		}
	} else {
		err = kafka.DoSend(w.kafkaProducer, w.config.Kafka.Topic, -1, []kafka.Doc{bs}, w.logger, nil)
		if err != nil {
			return err
		}
	}

	w.logger.Info("Finished summarizer",
		zap.Uint("job_id", j.ID),
		zap.String("benchmark_id", j.BenchmarkID),
	)
	return nil
}
