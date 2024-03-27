package controls

import (
	"github.com/goccy/go-yaml"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	complianceClient "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"go.uber.org/zap"
)

type assistantControl struct {
	ID            string              `json:"id" yaml:"id"`
	Title         string              `json:"title" yaml:"title"`
	CloudProvider []string            `json:"cloud_provider" yaml:"cloud_provider"`
	Severity      string              `json:"severity" yaml:"severity"`
	Tags          map[string][]string `json:"tags" yaml:"tags"`
}

func ExtractControls(logger *zap.Logger, complianceClient complianceClient.ComplianceServiceClient, tags map[string][]string) (map[string]string, error) {
	controls, err := complianceClient.ListControl(&httpclient.Context{UserRole: authApi.InternalRole}, nil, tags)
	if err != nil {
		logger.Error("failed to list controls", zap.Error(err))
		return nil, err
	}

	var assistantControls []assistantControl
	for _, c := range controls {
		var connectors []string
		for _, con := range c.Connector {
			connectors = append(connectors, con.String())
		}
		assistantControls = append(assistantControls, assistantControl{
			ID:            c.ID,
			Title:         c.Title,
			CloudProvider: connectors,
			Severity:      c.Severity.String(),
			Tags:          c.Tags,
		})
	}

	yamlAssistantControls, err := yaml.Marshal(assistantControls)
	if err != nil {
		logger.Error("failed to marshal assistant metrics", zap.Error(err))
		return nil, err
	}

	return map[string]string{
		"assistant_controls.yaml": string(yamlAssistantControls),
	}, nil
}
