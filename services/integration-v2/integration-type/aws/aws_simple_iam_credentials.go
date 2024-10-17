package aws

import (
	"encoding/json"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

// AWSSimpleIAMCredentials represents AWS single account credentials.
type AWSSimpleIAMCredentials struct {
	AccessKeyID     string  `json:"access_key_id" binding:"required"`
	AccessKeySecret string  `json:"access_key_secret" binding:"required"`
	RoleToAssume    *string `json:"role_to_assume,omitempty"`
}

func CreateAWSSimpleIAMCredentials(jsonData []byte) (interfaces.CredentialType, map[string]any, error) {
	var credentials AWSSimpleIAMCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, nil, err
	}

	return &credentials, credentials.ConvertToMap(), nil
}

func (c *AWSSimpleIAMCredentials) HealthCheck() error {
	return nil
}

func (c *AWSSimpleIAMCredentials) GetIntegrations() ([]models.Integration, error) {
	return nil, nil
}

func (c *AWSSimpleIAMCredentials) ToJSON() ([]byte, error) {
	return nil, nil
}

func (c *AWSSimpleIAMCredentials) ParseJSON([]byte) error {
	return nil
}

func (c *AWSSimpleIAMCredentials) ConvertToMap() map[string]any {
	result := map[string]any{
		"access_key_id":     c.AccessKeyID,
		"access_key_secret": c.AccessKeySecret,
	}

	// Add RoleToAssume if it's not nil
	if c.RoleToAssume != nil {
		result["role_to_assume"] = *c.RoleToAssume
	}

	return result
}
