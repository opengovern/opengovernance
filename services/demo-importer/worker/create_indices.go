package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

func CreateIndex(osClient *opensearch.Client, indexName string, settings, mappings json.RawMessage) error {
	config := map[string]json.RawMessage{
		"settings": settings,
		"mappings": mappings,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	req := opensearchapi.IndicesCreateRequest{
		Index: indexName,
		Body:  bytes.NewReader(configJSON),
	}

	res, err := req.Do(context.Background(), osClient)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Error creating index: %s", res.String())
	}
	fmt.Println("Index created:", indexName)
	return nil
}
