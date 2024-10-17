package interfaces

import "github.com/opengovern/opengovernance/services/integration-v2/models"

type Connector interface {
	AddCredential(credentials Credential) error
	FindIntegrations(credentials Credential) ([]models.Integration, error)
}

type BaseConnector struct {
	Connector Connector
}

func (b *BaseConnector) AddIntegrationsToDatabase() (*models.Integration, error) {
	return nil, nil
}
