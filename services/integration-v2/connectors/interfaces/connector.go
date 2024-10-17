package interfaces

import "github.com/opengovern/opengovernance/services/integration-v2/models"

type Connector interface {
	FindIntegrations(credentials CredentialType) ([]models.Integration, error)
	IntegrationHealthcheck(integration models.Integration) error
}
