package entra_id_directory

import (
	"encoding/json"
	"github.com/opengovern/og-util/pkg/integration"
	azureDescriberLocal "github.com/opengovern/opengovernance/services/integration-v2/integration-type/azure_subscription/configs"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/azure_subscription/discovery"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/azure_subscription/healthcheck"
	entraidDescriberLocal "github.com/opengovern/opengovernance/services/integration-v2/integration-type/entra_id_directory/configs"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

const (
	IntegrationTypeEntraIdDirectory integration.Type = "ENTRA_ID_DIRECTORY"
)

type EntraIdDirectoryIntegration struct{}

func CreateAzureSubscriptionIntegration() (interfaces.IntegrationType, error) {
	return &EntraIdDirectoryIntegration{}, nil
}

func (i *EntraIdDirectoryIntegration) GetDescriberConfiguration() interfaces.DescriberConfiguration {
	return interfaces.DescriberConfiguration{
		NatsScheduledJobsTopic: entraidDescriberLocal.JobQueueTopic,
		NatsManualJobsTopic:    entraidDescriberLocal.JobQueueTopicManuals,
		NatsStreamName:         entraidDescriberLocal.StreamName,
	}
}

func (i *EntraIdDirectoryIntegration) GetAnnotations(credentialType string, jsonData []byte) (map[string]string, error) {
	annotations := make(map[string]string)

	return annotations, nil
}

func (i *EntraIdDirectoryIntegration) GetLabels(credentialType string, jsonData []byte) (map[string]string, error) {
	annotations := make(map[string]string)

	return annotations, nil
}

func (i *EntraIdDirectoryIntegration) HealthCheck(credentialType string, jsonData []byte, providerId string, labels map[string]string) (bool, error) {
	var configs entraidDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &configs)
	if err != nil {
		return false, err
	}

	return healthcheck.AzureIntegrationHealthcheck(healthcheck.Config{
		TenantID:       configs.TenantID,
		ClientID:       configs.ClientID,
		ClientSecret:   configs.ClientSecret,
		CertPath:       configs.CertificatePath,
		CertContent:    configs.CertificatePath,
		CertPassword:   configs.CertificatePass,
		SubscriptionID: providerId,
	})
}

func (i *EntraIdDirectoryIntegration) DiscoverIntegrations(credentialType string, jsonData []byte) ([]models.Integration, error) {
	var configs azureDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &configs)
	if err != nil {
		return nil, err
	}

	var integrations []models.Integration
	subscriptions, err := discovery.AzureIntegrationDiscovery(discovery.Config{
		TenantID:     configs.TenantID,
		ClientID:     configs.ClientID,
		ClientSecret: configs.ClientSecret,
		CertPath:     configs.CertificatePath,
		CertContent:  configs.CertificatePath,
		CertPassword: configs.CertificatePass,
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

func (i *EntraIdDirectoryIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	return entraidDescriberLocal.ResourceTypesList, nil
}

func (i *EntraIdDirectoryIntegration) GetResourceTypeFromTableName(tableName string) string {
	return ""
}
