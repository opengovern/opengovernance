package analytics

import "github.com/kaytu-io/kaytu-util/pkg/source"

type Metric struct {
	Connectors               []source.Type       `json:"connectors" yaml:"connectors"`
	Name                     string              `json:"name" yaml:"name"`
	Query                    string              `json:"query" yaml:"query"`
	Tables                   []string            `json:"tables" yaml:"tables"`
	FinderQuery              string              `json:"finderQuery" yaml:"finderQuery"`
	FinderPerConnectionQuery string              `json:"finderPerConnectionQuery" yaml:"finderPerConnectionQuery"`
	Tags                     map[string][]string `json:"tags" yaml:"tags"`
	Status                   string              `json:"status" yaml:"status"`
}

type SmartQuery struct {
	Connectors []source.Type `json:"connectors" yaml:"connectors"`
	Title      string        `json:"title" yaml:"title"`
	Query      string        `json:"query" yaml:"query"`
}
