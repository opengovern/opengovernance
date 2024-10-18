package azure_subscription

import (
	"fmt"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
)

func CreateAzureSubscriptionIntegration(credentialType string, jsonData []byte) (interfaces.CredentialType, map[string]any, error) {
	if _, ok := CredentialTypes[credentialType]; !ok {
		return nil, nil, fmt.Errorf("invalid credential type")
	}
	credentialCreator := CredentialTypes[credentialType]
	return credentialCreator(jsonData)
}

var CredentialTypes = map[string]interfaces.CredentialCreator{
	"azure_spn_password":    CreateAzureSPNPasswordCredentials,
	"azure_spn_certificate": CreateAzureSPNCertificateCredentials,
}
