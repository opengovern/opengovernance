package metrics

import (
	"github.com/goccy/go-yaml"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"go.uber.org/zap"
)

type assistantMetric struct {
	ID             string              `json:"id"`
	CloudProviders []string            `json:"cloud_providers"`
	Type           string              `json:"type"`
	Name           string              `json:"name"`
	Tags           map[string][]string `json:"tags"`
}

func ExtractMetrics(logger *zap.Logger, i inventoryClient.InventoryServiceClient) (map[string]string, error) {
	metrics, err := i.ListAnalyticsMetrics(&httpclient.Context{UserRole: authApi.InternalRole}, nil)
	if err != nil {
		logger.Error("failed to list analytics metrics", zap.Error(err))
		return nil, err
	}

	var assistantMetrics []assistantMetric
	for _, m := range metrics {
		cloudProviders := make([]string, 0, len(m.Connectors))
		for _, c := range m.Connectors {
			cloudProviders = append(cloudProviders, c.String())
		}

		assistantMetrics = append(assistantMetrics, assistantMetric{
			ID:             m.ID,
			CloudProviders: cloudProviders,
			Type:           string(m.Type),
			Name:           m.Name,
			Tags:           m.Tags,
		})
	}

	yamlAssistantMetrics, err := yaml.Marshal(assistantMetrics)
	if err != nil {
		logger.Error("failed to marshal assistant metrics", zap.Error(err))
		return nil, err
	}

	return map[string]string{
		"assistant_metrics.yaml": string(yamlAssistantMetrics),
	}, nil
}
