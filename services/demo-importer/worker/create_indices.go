package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"strings"
)

func extractNestedField(rawJSON json.RawMessage, index, field string) (json.RawMessage, error) {
	var data map[string]json.RawMessage
	err := json.Unmarshal(rawJSON, &data)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling raw JSON: %w", err)
	}

	var outerData map[string]json.RawMessage
	err = json.Unmarshal(data[index], &outerData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling outer field %s: %w", index, err)
	}

	innerData, ok := outerData[field]
	if !ok {
		return nil, fmt.Errorf("inner field %s not found", field)
	}

	return innerData, nil
}

func CreateIndex(ctx context.Context, client *opensearchapi.Client, indexName string, settings, mappings json.RawMessage) error {
	fixedSettings, err := extractNestedField(settings, indexName, "settings")
	fixedMapping, err := extractNestedField(mappings, indexName, "mappings")
	config := map[string]json.RawMessage{
		"settings": fixedSettings,
		"mappings": fixedMapping,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	req := opensearchapi.IndicesCreateReq{
		Index: indexName,
		Body:  bytes.NewReader(configJSON),
	}

	res, err := client.Indices.Create(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "status: 400, type: resource_already_exists_exception") {
			return nil
		}
		return err
	}
	_, err = json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println("Index created:", indexName)
	return nil
}
