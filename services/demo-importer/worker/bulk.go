package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	bulkSize = 1000
)

type IndexConfig struct {
	Settings json.RawMessage `json:"settings"`
	Mappings json.RawMessage `json:"mappings"`
}

func BulkIndexData(osClient *opensearch.Client, requests []map[string]interface{}, indexName string) error {
	var buf bytes.Buffer
	for _, req := range requests {
		meta := []byte(`{ "index" : { "_index" : "` + indexName + `" } }` + "\n")
		data, err := json.Marshal(req)
		if err != nil {
			return err
		}
		buf.Grow(len(meta) + len(data) + 1)
		buf.Write(meta)
		buf.Write(data)
		buf.WriteByte('\n')
	}

	req := opensearchapi.BulkRequest{
		Body: bytes.NewReader(buf.Bytes()),
	}

	res, err := req.Do(context.Background(), osClient)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("Bulk indexing error: %s", body)
	}
	fmt.Println("Bulk indexing succeeded.")
	return nil
}

func ProcessJSONFile(logger *zap.Logger, osClient *opensearch.Client, filePath, indexName string, wg *sync.WaitGroup) {
	defer wg.Done()

	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("Error reading data file", zap.String("filePath", filePath), zap.String("indexName", indexName), zap.Error(err))
		return
	}
	defer file.Close()

	var requests []map[string]interface{}
	decoder := json.NewDecoder(file)

	_, err = decoder.Token()
	if err != nil {
		logger.Error("Error decoding data file", zap.String("filePath", filePath), zap.String("indexName", indexName), zap.Error(err))
		return
	}

	for decoder.More() {
		var doc map[string]interface{}
		err := decoder.Decode(&doc)
		if err != nil {
			logger.Error("Error decoding data file", zap.String("filePath", filePath), zap.String("indexName", indexName), zap.Error(err))
			return
		}

		requests = append(requests, doc)

		if len(requests) >= bulkSize {
			err = BulkIndexData(osClient, requests, indexName)
			if err != nil {
				logger.Error("Error Bulking file", zap.String("indexName", indexName), zap.Error(err))
				return
			}

			requests = nil
		}
	}

	if len(requests) > 0 {
		err = BulkIndexData(osClient, requests, indexName)
		if err != nil {
			logger.Error("Error Bulking file", zap.String("indexName", indexName), zap.Error(err))
			return
		}
	}
}

func ReadIndexConfigs(dir string) (map[string]IndexConfig, error) {
	indexConfigs := make(map[string]IndexConfig)

	mappingFiles, err := filepath.Glob(filepath.Join(dir, "*.mapping.json"))
	if err != nil {
		return nil, err
	}

	for _, mappingFile := range mappingFiles {
		baseName := strings.TrimSuffix(filepath.Base(mappingFile), ".mapping.json")
		settingsFile := filepath.Join(dir, baseName+".settings.json")

		mappingData, err := os.ReadFile(mappingFile)
		if err != nil {
			return nil, err
		}

		settingsData, err := os.ReadFile(settingsFile)
		if err != nil {
			return nil, err
		}

		indexConfigs[baseName] = IndexConfig{
			Mappings: mappingData,
			Settings: settingsData,
		}
	}

	return indexConfigs, nil
}
