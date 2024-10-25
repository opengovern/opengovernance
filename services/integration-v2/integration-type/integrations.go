package integration_type

import (
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/aws_account"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/azure_subscription"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
)

type IntegrationType string

var (
	IntegrationTypeAWSAccount        IntegrationType = "AWS_ACCOUNT"
	IntegrationTypeAzureSubscription IntegrationType = "AZURE_SUBSCRIPTION"
)

var IntegrationTypes = map[IntegrationType]interfaces.IntegrationCreator{
	IntegrationTypeAWSAccount:        aws_account.CreateAWSAccountIntegration,
	IntegrationTypeAzureSubscription: azure_subscription.CreateAzureSubscriptionIntegration,
}
