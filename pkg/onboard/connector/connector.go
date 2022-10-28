package connector

import (
	_ "embed"
	"encoding/json"
)

type Category struct {
	ID   string `json:"category_id"`
	Name string `json:"category_name"`
}

type Connector struct {
	ID          string `json:"connector_id"`
	Name        string `json:"connector_name"`
	Icon        string `json:"connector_icon"`
	Status      string `json:"connector_status"`
	Popular     string `json:"connector_popular"`
	Type        string `json:"connector_type"`
	SourceType  string `json:"source_type"`
	Description string `json:"connector_description"`
}

type ConnectorCount struct {
	Connector

	ConnectionCount int64 `json:"connection_count"`
}

type ConnectorCategoryMapping struct {
	CategoryID  string `json:"category_id"`
	ConnectorID string `json:"conn_id"`
}

var (
	//go:embed category_list.json
	categoryListString string
	CategoryList       []Category
	//go:embed connector_categories.json
	connectorsString string
	Connectors       []Connector
	//go:embed catt-conn-mapping.json
	categoryConnectorMappingString string
	CategoryConnectorMapping       []ConnectorCategoryMapping
)

func Init() error {
	if err := json.Unmarshal([]byte(categoryListString), &CategoryList); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(connectorsString), &Connectors); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(categoryConnectorMappingString), &CategoryConnectorMapping); err != nil {
		return err
	}
	return nil
}
