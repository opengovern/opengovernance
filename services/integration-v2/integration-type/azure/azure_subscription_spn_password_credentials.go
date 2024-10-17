package azure

import (
	"encoding/json"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

// AzureSPNPasswordCredentials represents Azure SPN credentials using a password.
type AzureSPNPasswordCredentials struct {
	AzureClientID       string  `json:"azure_client_id" binding:"required"`
	AzureTenantID       string  `json:"azure_tenant_id" binding:"required"`
	AzureClientPassword string  `json:"azure_client_password" binding:"required"`
	AzureSPNObjectID    *string `json:"azure_spn_object_id,omitempty"`
}

func CreateAzureSPNPasswordCredentials(jsonData []byte) (interfaces.CredentialType, map[string]any, error) {
	var credentials AzureSPNPasswordCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, nil, err
	}

	return &credentials, credentials.ConvertToMap(), nil
}

func (c *AzureSPNPasswordCredentials) HealthCheck() error {
	return nil
}

func (c *AzureSPNPasswordCredentials) GetIntegrations() ([]models.Integration, error) {
	return nil, nil
}

func (c *AzureSPNPasswordCredentials) ToJSON() ([]byte, error) {
	return nil, nil
}

func (c *AzureSPNPasswordCredentials) ParseJSON([]byte) error {
	return nil
}

func (c *AzureSPNPasswordCredentials) ConvertToMap() map[string]any {
	result := map[string]any{
		"azure_client_id":       c.AzureClientID,
		"azure_tenant_id":       c.AzureTenantID,
		"azure_client_password": c.AzureClientPassword,
	}

	// Add optional field if it is not nil
	if c.AzureSPNObjectID != nil {
		result["azure_spn_object_id"] = *c.AzureSPNObjectID
	}

	return result
}
