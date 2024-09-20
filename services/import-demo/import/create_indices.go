package _import

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

func CreateIndex(es *elasticsearch.Client, indexName string, settings, mappings json.RawMessage) error {
	config := map[string]json.RawMessage{
		"settings": settings,
		"mappings": mappings,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  bytes.NewReader(configJSON),
	}

	res, err := req.Do(context.Background(), es)
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
