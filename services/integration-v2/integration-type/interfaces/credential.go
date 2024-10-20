package interfaces

import (
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

type CredentialType interface {
	HealthCheck() error
	DiscoverIntegrations() ([]models.Integration, error)
	ConvertToMap() map[string]any
}

// IntegrationCreator CredentialType interface, credentials, error
type CredentialCreator func(jsonData []byte) (CredentialType, map[string]any, error)
