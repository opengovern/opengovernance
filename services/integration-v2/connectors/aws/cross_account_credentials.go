package aws

import (
	"github.com/opengovern/opengovernance/services/integration-v2/connectors/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

type AWSCrossAccountCredentials struct {
}

func NewAWSCrossAccountCredentials() *interfaces.CredentialType {
	return nil
}

func (c *AWSCrossAccountCredentials) HealthCheck() error {
	return nil
}

func (c *AWSCrossAccountCredentials) GetIntegrations() ([]models.Integration, error) {
	return nil, nil
}

func (c *AWSCrossAccountCredentials) ToJSON() ([]byte, error) {
	return nil, nil
}

func (c *AWSCrossAccountCredentials) ParseJSON([]byte) error {
	return nil
}
