package azure_subscription

import (
	"encoding/json"
	azureDescriberLocal "github.com/opengovern/opengovernance/services/integration-v2/integration-type/azure_subscription/configs"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/azure_subscription/discovery"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/azure_subscription/healthcheck"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/entra_id_directory/configs"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

type AzureSubscriptionIntegration struct{}

func CreateAzureSubscriptionIntegration() (interfaces.IntegrationType, error) {
	return &AzureSubscriptionIntegration{}, nil
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

func (i *AzureSubscriptionIntegration) HealthCheck(credentialType string, jsonData []byte, providerId string, labels map[string]string) (bool, error) {
	var credentials configs.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return false, err
	}

	return healthcheck.AzureIntegrationHealthcheck(healthcheck.Config{
		TenantID:       credentials.TenantID,
		ClientID:       credentials.ClientID,
		ClientSecret:   credentials.ClientSecret,
		CertPath:       credentials.CertificatePath,
		CertContent:    credentials.CertificatePath,
		CertPassword:   credentials.CertificatePass,
		SubscriptionID: providerId,
	})
}

func (i *AzureSubscriptionIntegration) DiscoverIntegrations(credentialType string, jsonData []byte) ([]models.Integration, error) {
	var credentials configs.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, err
	}

	var integrations []models.Integration
	subscriptions, err := discovery.AzureIntegrationDiscovery(discovery.Config{
		TenantID:     credentials.TenantID,
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientSecret,
		CertPath:     credentials.CertificatePath,
		CertContent:  credentials.CertificatePath,
		CertPassword: credentials.CertificatePass,
	})
	if err != nil {
		return nil, err
	}
	for _, s := range subscriptions {
		integrations = append(integrations, models.Integration{
			ProviderID: s.SubscriptionID,
			Name:       s.DisplayName,
		})
	}

	return integrations, nil
}

func (i *AzureSubscriptionIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	return configs.ResourceTypesList, nil
}

func (i *AzureSubscriptionIntegration) GetResourceTypeFromTableName(tableName string) string {
	return ""
}
