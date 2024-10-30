package integration_type

import (
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/aws_account"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/azure_subscription"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"strings"
)

const (
	IntegrationTypeAWSAccount        integration.Type = "AWS_ACCOUNT"
	IntegrationTypeAzureSubscription integration.Type = "AZURE_SUBSCRIPTION"
)

var AllIntegrationTypes = []integration.Type{
	IntegrationTypeAWSAccount,
	IntegrationTypeAzureSubscription,
}

var IntegrationTypes = map[integration.Type]interfaces.IntegrationCreator{
	IntegrationTypeAWSAccount:        aws_account.CreateAWSAccountIntegration,
	IntegrationTypeAzureSubscription: azure_subscription.CreateAzureSubscriptionIntegration,
}

func ParseType(str string) integration.Type {
	str = strings.ToLower(str)
	for _, t := range AllIntegrationTypes {
		if str == t.String() {
			return t
		}
	}
	return ""
}

func ParseTypes(str []string) []integration.Type {
	result := make([]integration.Type, 0, len(str))
	for _, s := range str {
		t := ParseType(s)
		if t == "" {
			continue
		}
		result = append(result, t)
	}
	return result
}
