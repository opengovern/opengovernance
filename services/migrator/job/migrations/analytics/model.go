package analytics

import (
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/opengovernance/services/migrator/job/migrations/shared"
)

type Metric struct {
	IntegrationTypes         []integration.Type  `json:"integrationType" yaml:"integrationType"`
	Name                     string              `json:"name" yaml:"name"`
	Query                    string              `json:"query" yaml:"query"`
	Tables                   []string            `json:"tables" yaml:"tables"`
	FinderQuery              string              `json:"finderQuery" yaml:"finderQuery"`
	FinderPerConnectionQuery string              `json:"finderPerConnectionQuery" yaml:"finderPerConnectionQuery"`
	Tags                     map[string][]string `json:"tags" yaml:"tags"`
	Status                   string              `json:"status" yaml:"status"`
}

type NamedQuery struct {
	Title            string              `json:"title" yaml:"Title"`
	Description      string              `json:"description" yaml:"Description"`
	IntegrationTypes []integration.Type  `json:"integrationType" yaml:"IntegrationType"`
	Query            shared.Query        `json:"query" yaml:"Query"`
	Tags             map[string][]string `json:"tags" yaml:"Tags"`
}
