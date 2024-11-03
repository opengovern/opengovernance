package interfaces

import "github.com/opengovern/opengovernance/services/integration-v2/models"

type DescriberConfiguration struct {
	NatsScheduledJobsTopic string
	NatsManualJobsTopic    string
	NatsStreamName         string
}

type IntegrationType interface {
	GetDescriberConfiguration() DescriberConfiguration
	GetAnnotations(credentialType string, jsonData []byte) (map[string]string, error)
	GetLabels(credentialType string, jsonData []byte) (map[string]string, error)
	GetResourceTypesByLabels(map[string]string) ([]string, error)
	HealthCheck(credentialType string, jsonData []byte) (bool, error)
	DiscoverIntegrations(credentialType string, jsonData []byte) ([]models.Integration, error)
	GetResourceTypeFromTableName(tableName string) string
}

// IntegrationCreator IntegrationType interface, credentials, error
type IntegrationCreator func() (IntegrationType, error)
