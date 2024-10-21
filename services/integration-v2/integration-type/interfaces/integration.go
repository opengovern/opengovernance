package interfaces

import "github.com/opengovern/opengovernance/services/integration-v2/models"

type IntegrationType interface {
	GetAnnotations() (map[string]any, error)
	GetMetadata() (map[string]any, error)
	GetLabels() (map[string]any, error)
	HealthCheck() error
	DiscoverIntegrations() ([]models.Integration, error)
}

// IntegrationCreator IntegrationType interface, credentials, error
type IntegrationCreator func(certificateType string, jsonData []byte) (IntegrationType, map[string]any, error)
