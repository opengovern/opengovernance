package azure

import (
	"github.com/opengovern/opengovernance/services/integration-v2/connectors/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

type AzureConnector struct{}

func NewAWSConnector() interfaces.Connector {
	return &AzureConnector{}
}

func (c *AzureConnector) FindIntegrations(credentials interfaces.CredentialType) ([]models.Integration, error) {
	return nil, nil
}

func (c *AzureConnector) IntegrationHealthcheck(integration models.Integration) error {
	return nil
}
