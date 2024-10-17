package aws

import (
	"github.com/opengovern/opengovernance/services/integration-v2/connectors/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

type AWSIAMCredentials struct {
	AccountID      string  `json:"accountID"`
	AssumeRoleName string  `json:"assumeRoleName"`
	ExternalId     *string `json:"externalId,omitempty"`

	AccessKey *string `json:"accessKey,omitempty"`
	SecretKey *string `json:"secretKey,omitempty"`
}

func NewAWSIAMCredentials() *interfaces.CredentialType {
	return nil
}

func (c *AWSIAMCredentials) HealthCheck() error {
	return nil
}

func (c *AWSIAMCredentials) GetIntegrations() ([]models.Integration, error) {
	return nil, nil
}

func (c *AWSIAMCredentials) ToJSON() ([]byte, error) {
	return nil, nil
}

func (c *AWSIAMCredentials) ParseJSON([]byte) error {
	return nil
}
