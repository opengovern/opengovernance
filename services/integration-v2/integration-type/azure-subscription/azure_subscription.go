package azure_subscription

import (
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
)

func CreateAzureSubscriptionIntegration(credentialType string, jsonData []byte) (interfaces.CredentialType, map[string]any, error) {
	credentialCreator := CredentialTypes[credentialType]
	return credentialCreator(jsonData)
}

var CredentialTypes = map[string]interfaces.CredentialCreator{
	"azure_spn_password":    CreateAzureSPNPasswordCredentials,
	"azure_spn_certificate": CreateAzureSPNCertificateCredentials,
}
