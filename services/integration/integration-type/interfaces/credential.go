package interfaces

import (
	"github.com/opengovern/opengovernance/services/integration/models"
)

type CredentialType interface {
	HealthCheck() (bool, error)
	DiscoverIntegrations() ([]models.Integration, error)
}

// IntegrationCreator CredentialType interface, credentials, error
type CredentialCreator func(jsonData []byte) (CredentialType, error)
