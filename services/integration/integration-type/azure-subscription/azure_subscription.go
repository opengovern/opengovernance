package azure_subscription

import (
	"encoding/json"
	azureDescriberLocal "github.com/opengovern/opengovernance/services/integration/integration-type/azure-subscription/configs"
	"github.com/opengovern/opengovernance/services/integration/integration-type/azure-subscription/discovery"
	"github.com/opengovern/opengovernance/services/integration/integration-type/azure-subscription/healthcheck"
	"github.com/opengovern/opengovernance/services/integration/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration/models"
)

type AzureSubscriptionIntegration struct{}

func (i *AzureSubscriptionIntegration) GetConfiguration() interfaces.IntegrationConfiguration {
	return interfaces.IntegrationConfiguration{
		NatsScheduledJobsTopic: azureDescriberLocal.JobQueueTopic,
		NatsManualJobsTopic:    azureDescriberLocal.JobQueueTopicManuals,
		NatsStreamName:         azureDescriberLocal.StreamName,

		UISpecFileName: "azure-subscription.json",
	}
}

func (i *AzureSubscriptionIntegration) HealthCheck(jsonData []byte, providerId string, labels map[string]string, annotations map[string]string) (bool, error) {
	var credentials azureDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return false, err
	}

	return healthcheck.AzureIntegrationHealthcheck(healthcheck.Config{
		TenantID:       credentials.TenantID,
		ClientID:       credentials.ClientID,
		ClientSecret:   credentials.ClientPassword,
		CertPath:       "",
		CertContent:    credentials.Certificate,
		CertPassword:   credentials.CertificatePassword,
		SubscriptionID: providerId,
	})
}

func (i *AzureSubscriptionIntegration) DiscoverIntegrations(jsonData []byte) ([]models.Integration, error) {
	var credentials azureDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, err
	}

	var integrations []models.Integration
	subscriptions, err := discovery.AzureIntegrationDiscovery(discovery.Config{
		TenantID:     credentials.TenantID,
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientPassword,
		CertPath:     "",
		CertContent:  credentials.Certificate,
		CertPassword: credentials.CertificatePassword,
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
	return azureDescriberLocal.ResourceTypesList, nil
}

func (i *AzureSubscriptionIntegration) GetResourceTypeFromTableName(tableName string) string {
	if v, ok := azureDescriberLocal.TablesToResourceTypes[tableName]; ok {
		return v
	}

	return ""
}
