package azure_subscription

import (
	"fmt"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

type AzureSubscriptionIntegration struct {
	Credential interfaces.CredentialType
}

func CreateAzureSubscriptionIntegration(credentialType *string, jsonData []byte) (interfaces.IntegrationType, error) {
	integration := AzureSubscriptionIntegration{}

	if credentialType != nil {
		if _, ok := CredentialTypes[*credentialType]; !ok {
			return nil, fmt.Errorf("invalid credential type: %s", credentialType)
		}
		credentialCreator := CredentialTypes[*credentialType]
		credential, err := credentialCreator(jsonData)
		if err != nil {
			return nil, err
		}
		integration.Credential = credential
	}

	return &integration, nil
}

var CredentialTypes = map[string]interfaces.CredentialCreator{
	"azure_spn_password":    CreateAzureSPNPasswordCredentials,
	"azure_spn_certificate": CreateAzureSPNCertificateCredentials,
}

func (i *AzureSubscriptionIntegration) GetAnnotations() (map[string]any, error) {
	annotations := make(map[string]any)

	return annotations, nil
}

func (i *AzureSubscriptionIntegration) GetMetadata() (map[string]any, error) {
	annotations := make(map[string]any)

	return annotations, nil
}

func (i *AzureSubscriptionIntegration) GetLabels() (map[string]any, error) {
	annotations := make(map[string]any)

	return annotations, nil
}

func (i *AzureSubscriptionIntegration) HealthCheck() error {
	return i.Credential.HealthCheck()
}

func (i *AzureSubscriptionIntegration) DiscoverIntegrations() ([]models.Integration, error) {
	return i.Credential.DiscoverIntegrations()
}

func (i *AzureSubscriptionIntegration) GetResourceTypesByLabels(map[string]any) ([]string, error) {
	return nil, nil
}
