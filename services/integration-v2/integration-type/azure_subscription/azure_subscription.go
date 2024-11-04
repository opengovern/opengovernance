package azure_subscription

import (
	"fmt"
	"github.com/opengovern/og-azure-describer/azure"
	azureDescriberLocal "github.com/opengovern/og-azure-describer/local"
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

const (
	IntegrationTypeAzureSubscription integration.Type = "AZURE_SUBSCRIPTION"
)

type AzureSubscriptionIntegration struct{}

func CreateAzureSubscriptionIntegration() (interfaces.IntegrationType, error) {
	return &AzureSubscriptionIntegration{}, nil
}

var CredentialTypes = map[string]interfaces.CredentialCreator{
	"azure_spn_password":    CreateAzureSPNPasswordCredentials,
	"azure_spn_certificate": CreateAzureSPNCertificateCredentials,
}

func (i *AzureSubscriptionIntegration) GetDescriberConfiguration() interfaces.DescriberConfiguration {
	return interfaces.DescriberConfiguration{
		NatsScheduledJobsTopic: azureDescriberLocal.JobQueueTopic,
		NatsManualJobsTopic:    azureDescriberLocal.JobQueueTopicManuals,
		NatsStreamName:         azureDescriberLocal.StreamName,
	}
}

func (i *AzureSubscriptionIntegration) GetAnnotations(credentialType string, jsonData []byte) (map[string]string, error) {
	annotations := make(map[string]string)

	return annotations, nil
}

func (i *AzureSubscriptionIntegration) GetLabels(credentialType string, jsonData []byte) (map[string]string, error) {
	annotations := make(map[string]string)

	return annotations, nil
}

func (i *AzureSubscriptionIntegration) HealthCheck(credentialType string, jsonData []byte) (bool, error) {
	azureCredential, err := getCredentials(credentialType, jsonData)
	if err != nil {
		return false, fmt.Errorf("failed to parse Azure credentials of type %s: %s", credentialType, err.Error())
	}

	return azureCredential.HealthCheck()
}

func (i *AzureSubscriptionIntegration) DiscoverIntegrations(credentialType string, jsonData []byte) ([]models.Integration, error) {
	azureCredential, err := getCredentials(credentialType, jsonData)
	if err != nil {
		return nil, err
	}

	return azureCredential.DiscoverIntegrations()
}

func (i *AzureSubscriptionIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	return azure.ListResourceTypes(), nil
}

func getCredentials(credentialType string, jsonData []byte) (interfaces.CredentialType, error) {
	if _, ok := CredentialTypes[credentialType]; !ok {
		return nil, fmt.Errorf("invalid credential type: %s", credentialType)
	}
	credentialCreator := CredentialTypes[credentialType]
	credential, err := credentialCreator(jsonData)
	if err != nil {
		return nil, err
	}

	return credential, nil
}

func (i *AzureSubscriptionIntegration) GetResourceTypeFromTableName(tableName string) string {
	return ""
}
