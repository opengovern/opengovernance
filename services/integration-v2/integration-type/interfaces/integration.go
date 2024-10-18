package interfaces

import (
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/aws"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/azure-subscription"
)

type IntegrationCreator func(certificateType string, jsonData []byte) (CredentialType, map[string]any, error)

var IntegrationTypes = map[string]IntegrationCreator{
	"aws_account":        aws.CreateAWSAccountIntegration,
	"azure_subscription": azure_subscription.CreateAzureSubscriptionIntegration,
}
