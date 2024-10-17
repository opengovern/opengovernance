package aws

import (
	"github.com/opengovern/opengovernance/services/integration-v2/connectors/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

type AWSConnector struct {
}

func NewAWSConnector() interfaces.Connector {
	return &AWSConnector{}
}

func (c *AWSConnector) FindIntegrations(credentials interfaces.CredentialType) ([]models.Integration, error) {
	return nil, nil
}

func (c *AWSConnector) IntegrationHealthcheck(integration models.Integration) error {
	return nil
}
