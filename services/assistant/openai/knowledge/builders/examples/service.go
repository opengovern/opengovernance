package examples

import (
	"github.com/goccy/go-yaml"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
)

type Example struct {
	Title       string `yaml:"Title"`
	Description string `yaml:"Description"`
	Query       string `yaml:"Query"`
}

type Def struct {
	Examples []Example `yaml:"Examples"`
}

func ExtractExamples(complianceClient client.ComplianceServiceClient) (map[string]string, error) {
	var examples []Example
	controls, err := complianceClient.ListControl(&httpclient.Context{UserRole: api.InternalRole}, nil)
	if err != nil {
		return nil, err
	}

	for _, c := range controls {
		if c.Query == nil {
			continue
		}
		examples = append(examples, Example{
			Title:       c.Title,
			Description: c.Description,
			Query:       c.Query.QueryToExecute,
		})
	}

	e, err := yaml.Marshal(Def{Examples: examples})
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"examples.yaml": string(e),
	}, nil
}
