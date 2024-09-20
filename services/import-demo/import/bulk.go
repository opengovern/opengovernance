package _import

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	bulkSize = 1000
)

type IndexConfig struct {
	Settings json.RawMessage `json:"settings"`
	Mappings json.RawMessage `json:"mappings"`
}

func BulkIndexData(es *elasticsearch.Client, requests []map[string]interface{}, indexName string) error {
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

	req := esapi.BulkRequest{
		Body: bytes.NewReader(buf.Bytes()),
	}

	res, err := req.Do(context.Background(), es)
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

func ProcessJSONFile(es *elasticsearch.Client, filePath, indexName string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var requests []map[string]interface{}
	decoder := json.NewDecoder(file)

	_, err = decoder.Token()
	if err != nil {
		return err
	}

	for decoder.More() {
		var doc map[string]interface{}
		err := decoder.Decode(&doc)
		if err != nil {
			return err
		}
		requests = append(requests, doc)

		if len(requests) >= bulkSize {
			err = BulkIndexData(es, requests, indexName)
			if err != nil {
				return err
			}
			requests = nil
		}
	}

	if len(requests) > 0 {
		err = BulkIndexData(es, requests, indexName)
		if err != nil {
			return err
		}
	}

	return nil
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
