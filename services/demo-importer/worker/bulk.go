package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"go.uber.org/zap"
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

func BulkIndexData(ctx context.Context, client *opensearchapi.Client, requests []map[string]interface{}) error {
	var documentsBuilder strings.Builder
	for _, request := range requests {
		indexLineJson := map[string]map[string]string{
			"index": {
				"_id":    fmt.Sprintf("%v", request["_id"]),
				"_index": fmt.Sprintf("%v", request["_index"]),
			},
		}
		indexLine, err := json.Marshal(indexLineJson)
		if err != nil {
			return err
		}
		dataLine, err := json.Marshal(request["_source"])
		if err != nil {
			return err
		}
		// Append index line and data line with newlines
		documentsBuilder.WriteString(string(indexLine) + "\n")
		documentsBuilder.WriteString(string(dataLine) + "\n")
	}

	documents := documentsBuilder.String()

	// Execute the bulk request
	req := opensearchapi.BulkReq{
		Body: strings.NewReader(documents),
	}

	bulkResp, err := client.Bulk(ctx, req)
	if err != nil {
		fmt.Println("err", err)
		return err
	}
	_, err = json.MarshalIndent(bulkResp, "", "  ")
	if err != nil {
		fmt.Println("err", err)
		return err
	}
	//fmt.Printf("Bulk Resp:\n%s\n", string(respAsJson))

	return nil
}

func ProcessJSONFile(ctx context.Context, logger *zap.Logger, osClient *opensearchapi.Client, filePath, indexName string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err.Error())
		logger.Error("Error open file", zap.String("filePath", filePath), zap.String("indexName", indexName), zap.Error(err))
		return
	}
	defer file.Close()

	var requests []map[string]interface{}

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')

		if len(line) > 0 {
			var doc map[string]interface{}
			err = json.Unmarshal([]byte(line), &doc)
			if err != nil {
				fmt.Println(err.Error())
				logger.Error("Error decoding data file", zap.String("filePath", filePath), zap.String("indexName", indexName), zap.Error(err))
				return
			}

			requests = append(requests, doc)

			if len(requests) >= bulkSize {
				err = BulkIndexData(ctx, osClient, requests)
				if err != nil {
					logger.Error("Error Bulking file", zap.String("indexName", indexName), zap.Error(err))
					return
				}

				requests = nil
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println(err.Error())
			logger.Error("Error reading data file", zap.String("filePath", filePath), zap.String("indexName", indexName), zap.Error(err))
			return
		}
	}

	if len(requests) > 0 {
		err = BulkIndexData(ctx, osClient, requests)
		if err != nil {
			logger.Error("Error Bulking file", zap.String("indexName", indexName), zap.Error(err))
			return
		}
	}

	fmt.Println("Index data import completed:", indexName)
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
		fixedSettingsData := strings.ReplaceAll(string(settingsData), `\"`, `"`)
		fixedSettingsData = strings.TrimPrefix(fixedSettingsData, "\"")
		fixedSettingsData = strings.TrimSpace(fixedSettingsData)
		fixedSettingsData = strings.TrimSuffix(fixedSettingsData, "\"")
		indexConfigs[baseName] = IndexConfig{
			Mappings: mappingData,
			Settings: []byte(fixedSettingsData),
		}
	}

	return indexConfigs, nil
}


