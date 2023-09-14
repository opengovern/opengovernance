package analytics

import "github.com/kaytu-io/kaytu-util/pkg/source"

type Metric struct {
	Connectors  []source.Type       `json:"connectors"`
	Name        string              `json:"name"`
	Query       string              `json:"query"`
	Tables      []string            `json:"tables"`
	FinderQuery string              `json:"finderQuery"`
	Tags        map[string][]string `json:"tags"`
	Visible     *bool               `json:"visible"`
}

type SmartQuery struct {
	Connector source.Type `json:"connector"`
	Title     string      `json:"title"`
	Query     string      `json:"query"`
}
