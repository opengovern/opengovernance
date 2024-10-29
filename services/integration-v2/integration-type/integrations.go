package integration_type

import (
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/aws_account"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/azure_subscription"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
)

var (
	IntegrationTypeAWSAccount        integration.Type = "AWS_ACCOUNT"
	IntegrationTypeAzureSubscription integration.Type = "AZURE_SUBSCRIPTION"
)

var IntegrationTypes = map[integration.Type]interfaces.IntegrationCreator{
	IntegrationTypeAWSAccount:        aws_account.CreateAWSAccountIntegration,
	IntegrationTypeAzureSubscription: azure_subscription.CreateAzureSubscriptionIntegration,
}
