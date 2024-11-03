package integration_type

import (
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/aws_account"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/azure_subscription"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"strings"
)

const (
	IntegrationTypeAWSAccount        = aws_account.IntegrationTypeAWSAccount
	IntegrationTypeAzureSubscription = azure_subscription.IntegrationTypeAzureSubscription
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

func UnparseTypes(types []integration.Type) []string {
	result := make([]string, 0, len(types))
	for _, t := range types {
		result = append(result, t.String())
	}
	return result
}
