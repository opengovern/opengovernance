package analytics

import (
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/migrations/shared"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

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
	Title       string              `json:"title" yaml:"Title"`
	Description string              `json:"description" yaml:"Description"`
	Connectors  []source.Type       `json:"connectors" yaml:"Connectors"`
	Query       shared.Query        `json:"query" yaml:"Query"`
	Tags        map[string][]string `json:"tags" yaml:"Tags"`
}
