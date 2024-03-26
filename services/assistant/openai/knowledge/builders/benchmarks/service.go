package benchmarks

import (
	"github.com/goccy/go-yaml"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	complianceClient "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"go.uber.org/zap"
)

type assistantBenchmark struct {
	ID            string `json:"id" yaml:"id"`
	Title         string `json:"title" yaml:"title"`
	CloudProvider string `json:"cloud_provider" yaml:"cloud_provider"`
	Tags          map[string][]string
}

func ExtractBenchmarks(logger *zap.Logger, complianceClient complianceClient.ComplianceServiceClient, tags map[string][]string) (map[string]string, error) {
	benchmarks, err := complianceClient.ListBenchmarks(&httpclient.Context{UserRole: authApi.InternalRole}, tags)
	if err != nil {
		logger.Error("failed to list benchmarks", zap.Error(err))
		return nil, err
	}

	var assistantBenchmarks []assistantBenchmark
	for _, c := range benchmarks {
		b := assistantBenchmark{
			ID:    c.ID,
			Title: c.Title,
			Tags:  c.Tags,
		}
		if len(c.Connectors) > 0 {
			b.CloudProvider = c.Connectors[0].String()
		}
		assistantBenchmarks = append(assistantBenchmarks, b)
	}

	yamlAssistantBenchmarks, err := yaml.Marshal(assistantBenchmarks)
	if err != nil {
		logger.Error("failed to marshal assistant benchmarks", zap.Error(err))
		return nil, err
	}

	return map[string]string{
		"assistant_benchmarks.yaml": string(yamlAssistantBenchmarks),
	}, nil
}
