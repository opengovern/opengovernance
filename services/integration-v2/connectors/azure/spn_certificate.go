package azure

import (
	"github.com/opengovern/opengovernance/services/integration-v2/connectors/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

type AzureSPNCerificate struct {
	TenantId     string `json:"tenantId"`
	ObjectId     string `json:"objectId"`
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

func NewAzureSPNCerificate() *interfaces.CredentialType {
	return nil
}

func (c *AzureSPNCerificate) HealthCheck() error {
	return nil
}

func (c *AzureSPNCerificate) GetIntegrations() ([]models.Integration, error) {
	return nil, nil
}

func (c *AzureSPNCerificate) ToJSON() ([]byte, error) {
	return nil, nil
}

func (c *AzureSPNCerificate) ParseJSON([]byte) error {
	return nil
}
