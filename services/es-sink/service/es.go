package service

import (
	"bytes"
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/es-sink/metrics"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	essdk "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/opensearch-project/opensearch-go/v2/opensearchutil"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"io"
	"time"
)

type EsSinkModule struct {
	logger *zap.Logger

	elasticsearch essdk.Client
	indexer       opensearchutil.BulkIndexer

	existingIndices map[string]bool

	inputChan chan es.Doc
	retryChan chan opensearchutil.BulkIndexerItem
}

func NewEsSinkModule(ctx context.Context, logger *zap.Logger, elasticSearch essdk.Client) (*EsSinkModule, error) {
	inputChan := make(chan es.Doc, 1000)
	retryChan := make(chan opensearchutil.BulkIndexerItem, 1000)
	indexer, err := opensearchutil.NewBulkIndexer(opensearchutil.BulkIndexerConfig{
		Client: elasticSearch.ES(),
	})
	if err != nil {
		logger.Error("failed to create bulk indexer", zap.Error(err))
		return nil, err
	}

	indices, err := elasticSearch.ListIndices(ctx)
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

func (m *EsSinkModule) QueueDoc(doc es.Doc) {
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
			key, idx := resource.KeysAndIndex()
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
			err = m.indexer.Add(ctx, opensearchutil.BulkIndexerItem{
				Index:           idx,
				Action:          "index",
				DocumentID:      es.HashOf(key...),
				Body:            bytes.NewReader(resourceJson),
				RetryOnConflict: utils.GetPointer(5),
				OnFailure:       m.handleFailure,
			})
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
