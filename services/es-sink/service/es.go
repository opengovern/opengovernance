package service

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/es-sink/metrics"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	essdk "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/opensearch-project/opensearch-go/v2/opensearchutil"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"io"
	"strings"
	"time"
)

type EsSinkModule struct {
	logger *zap.Logger

	elasticsearch essdk.Client
	indexer       opensearchutil.BulkIndexer

	existingIndices map[string]bool

	inputChan chan es.DocBase
	retryChan chan opensearchutil.BulkIndexerItem
}
type debugLogger struct {
	logger *zap.Logger
}

func (d debugLogger) Printf(format string, args ...any) {
	d.logger.Warn(fmt.Sprintf(format, args...))
}

func NewEsSinkModule(ctx context.Context, logger *zap.Logger, elasticSearch essdk.Client) (*EsSinkModule, error) {
	inputChan := make(chan es.DocBase, 1000)
	retryChan := make(chan opensearchutil.BulkIndexerItem, 1000)

	indexer, err := opensearchutil.NewBulkIndexer(opensearchutil.BulkIndexerConfig{
		NumWorkers:  4,
		Client:      elasticSearch.ES(),
		DebugLogger: debugLogger{logger: logger},
		OnError: func(ctx context.Context, err error) {
			logger.Error("bulk indexer error", zap.Error(err))
		},
	})
	if err != nil {
		logger.Error("failed to create bulk indexer", zap.Error(err))
		return nil, err
	}

	indices, err := elasticSearch.ListIndices(ctx, logger)
	if err != nil {
		logger.Error("failed to list indices", zap.Error(err))
		return nil, err
	}
	existingIndices := make(map[string]bool)
	for _, idx := range indices {
		existingIndices[idx] = true
	}

	return &EsSinkModule{
		logger:          logger,
		elasticsearch:   elasticSearch,
		inputChan:       inputChan,
		retryChan:       retryChan,
		indexer:         indexer,
		existingIndices: existingIndices,
	}, nil
}

func (m *EsSinkModule) QueueDoc(doc es.DocBase) {
	m.inputChan <- doc
}

func (m *EsSinkModule) Start(ctx context.Context) {
	utils.EnsureRunGoroutine(func() {
		m.updateStatsCycle()
	})
	for {
		select {
		case resource := <-m.inputChan:
			if resource == nil {
				continue
			}
			id, idx := resource.GetIdAndIndex()
			if _, ok := m.existingIndices[idx]; !ok {
				err := m.elasticsearch.CreateIndexIfNotExist(ctx, m.logger, idx)
				if err != nil {
					m.logger.Error("failed to create index", zap.Error(err))
					continue
				}
				m.existingIndices[idx] = true
			}
			resourceJson, err := json.Marshal(resource)
			if err != nil {
				m.logger.Error("failed to marshal resource", zap.Error(err))
				continue
			}
			m.logger.Info("indexing resource", zap.String("id", id), zap.String("index", idx), zap.String("resource", string(resourceJson)))
			err = m.indexer.Add(ctx, opensearchutil.BulkIndexerItem{
				Index:           idx,
				Action:          "index",
				DocumentID:      id,
				Body:            strings.NewReader(string(resourceJson)),
				RetryOnConflict: utils.GetPointer(5),
				OnFailure:       m.handleFailure,
				OnSuccess: func(ctx context.Context, item opensearchutil.BulkIndexerItem, response opensearchutil.BulkIndexerResponseItem) {
					m.logger.Info("resource indexed", zap.String("id", id), zap.String("index", idx))
				},
			})
			if err != nil {
				m.logger.Error("failed to add resource to bulk indexer", zap.Error(err))
				continue
			}
		case resource := <-m.retryChan:
			err := m.indexer.Add(ctx, resource)
			if err != nil {
				m.logger.Error("failed to retry indexing", zap.Error(err))
				continue
			}
		case <-ctx.Done():
			m.indexer.Close(context.Background())
			return
		}
	}
}

func (m *EsSinkModule) handleFailure(ctx context.Context, item opensearchutil.BulkIndexerItem, response opensearchutil.BulkIndexerResponseItem, err error) {
	// TODO handle too many requests with retry chan

	resourceJson, err2 := io.ReadAll(item.Body)
	if err != nil {
		m.logger.Error("failed to read failed resource", zap.Error(err2), zap.Any("item", item), zap.Any("response", response), zap.Any("originalError", err))
		return
	}
	m.logger.Error("failed to index resource", zap.Error(err), zap.String("resource", string(resourceJson)), zap.Any("item", item), zap.Any("response", response))
	//TODO write to a DLQ
}

func (m *EsSinkModule) updateStatsCycle() {
	statsTicker := time.NewTicker(30 * time.Second)
	defer statsTicker.Stop()
	for range statsTicker.C {
		stats := m.indexer.Stats()
		m.logger.Info("updating metric es sink stats", zap.Any("stats", stats))
		metrics.EsSinkDocsNumAdded.Set(float64(stats.NumAdded))
		metrics.EsSinkDocsNumFlushed.Set(float64(stats.NumFlushed))
		metrics.EsSinkDocsNumFailed.Set(float64(stats.NumFailed))
		metrics.EsSinkDocsNumIndexed.Set(float64(stats.NumIndexed))
		metrics.EsSinkDocsNumCreated.Set(float64(stats.NumCreated))
		metrics.EsSinkDocsNumUpdated.Set(float64(stats.NumUpdated))
		metrics.EsSinkDocsNumDeleted.Set(float64(stats.NumDeleted))
		metrics.EsSinkDocsNumRequests.Set(float64(stats.NumRequests))
	}
}
