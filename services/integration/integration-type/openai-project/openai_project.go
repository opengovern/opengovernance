package openai_project

import (
	"encoding/json"
	"github.com/jackc/pgtype"
	"github.com/opengovern/opencomply/services/integration/integration-type/interfaces"
	openaiDescriberLocal "github.com/opengovern/opencomply/services/integration/integration-type/openai-project/configs"
	"github.com/opengovern/opencomply/services/integration/integration-type/openai-project/healthcheck"
	"github.com/opengovern/opencomply/services/integration/models"
)

type OpenAIProjectIntegration struct{}

func (i *OpenAIProjectIntegration) GetConfiguration() interfaces.IntegrationConfiguration {
	return interfaces.IntegrationConfiguration{
		NatsScheduledJobsTopic:   openaiDescriberLocal.JobQueueTopic,
		NatsManualJobsTopic:      openaiDescriberLocal.JobQueueTopicManuals,
		NatsStreamName:           openaiDescriberLocal.StreamName,
		NatsConsumerGroup:        openaiDescriberLocal.ConsumerGroup,
		NatsConsumerGroupManuals: openaiDescriberLocal.ConsumerGroupManuals,

		SteampipePluginName: "openai",

		UISpecFileName: "openai-project.json",

		DescriberDeploymentName: openaiDescriberLocal.DescriberDeploymentName,
		DescriberImageAddress:   openaiDescriberLocal.DescriberImageAddress,
		DescriberImageTagKey:    openaiDescriberLocal.DescriberImageTagKey,
		DescriberRunCommand:     openaiDescriberLocal.DescriberRunCommand,
	}
}

func (i *OpenAIProjectIntegration) HealthCheck(jsonData []byte, providerId string, labels map[string]string, annotations map[string]string) (bool, error) {
	var credentials openaiDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return false, err
	}

	isHealthy, err := healthcheck.OpenAIIntegrationHealthcheck(credentials.APIKey)
	return isHealthy, err
}

func (i *OpenAIProjectIntegration) DiscoverIntegrations(jsonData []byte) ([]models.Integration, error) {
	var credentials openaiDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, err
	}
	var integrations []models.Integration
	_, err = healthcheck.OpenAIIntegrationHealthcheck(credentials.APIKey)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{
		"OrganizationID": credentials.OrganizationID,
	}
	labelsJsonData, err := json.Marshal(labels)
	if err != nil {
		return nil, err
	}
	integrationLabelsJsonb := pgtype.JSONB{}
	err = integrationLabelsJsonb.Set(labelsJsonData)
	if err != nil {
		return nil, err
	}
	integrations = append(integrations, models.Integration{
		ProviderID: credentials.ProjectID,
		Name:       credentials.ProjectName,
		Labels:     integrationLabelsJsonb,
	})

	return integrations, nil
}

func (i *OpenAIProjectIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	return openaiDescriberLocal.ResourceTypesList, nil
}

func (i *OpenAIProjectIntegration) GetResourceTypeFromTableName(tableName string) string {
	if v, ok := openaiDescriberLocal.TablesToResourceTypes[tableName]; ok {
		return v
	}

	return ""
}