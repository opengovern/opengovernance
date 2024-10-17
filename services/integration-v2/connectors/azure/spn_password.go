package azure

import (
	"github.com/opengovern/opengovernance/services/integration-v2/connectors/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

type AzureSPNPassword struct{}

func NewAzureSPNPassword() *interfaces.CredentialType {
	return nil
}

func (c *AzureSPNPassword) HealthCheck() error {
	return nil
}

func (c *AzureSPNPassword) GetIntegrations() ([]models.Integration, error) {
	return nil, nil
}

func (c *AzureSPNPassword) ToJSON() ([]byte, error) {
	return nil, nil
}

func (c *AzureSPNPassword) ParseJSON([]byte) error {
	return nil
}
