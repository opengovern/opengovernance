package interfaces

import "github.com/opengovern/opengovernance/services/integration-v2/models"

type DescriberConfiguration struct {
	NatsScheduledJobsTopic string
	NatsManualJobsTopic    string
}

type IntegrationType interface {
	GetDescriberConfiguration() DescriberConfiguration
	GetAnnotations() (map[string]any, error)
	GetMetadata() (map[string]any, error)
	GetLabels() (map[string]any, error)
	GetResourceTypesByLabels(map[string]string) ([]string, error)
	HealthCheck() error
	DiscoverIntegrations() ([]models.Integration, error)
}

// IntegrationCreator IntegrationType interface, credentials, error
type IntegrationCreator func(certificateType *string, jsonData []byte) (IntegrationType, error)
