package openai_project

import (
	"encoding/json"
	"github.com/jackc/pgtype"
	"github.com/opengovern/opencomply/services/integration/integration-type/interfaces"
	openaiDescriberLocal "github.com/opengovern/opencomply/services/integration/integration-type/openai-project/configs"
	"github.com/opengovern/opencomply/services/integration/integration-type/openai-project/healthcheck"
	"github.com/opengovern/opencomply/services/integration/integration-type/openai-project/discovery"

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
	orgResponse, err1 := discovery.OpenAIIntegrationDiscovery(credentials.APIKey)
	if err1 != nil {
		return nil, err1
	}
	labels := map[string]string{
		"OrganizationID": orgResponse.OrganizationID,
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
	// for in thr orgResponse.Projects
	for _, project := range orgResponse.Projects {
integrations = append(integrations, models.Integration{
		ProviderID: project.ID,
		Name:       project.Name,
		Labels:     integrationLabelsJsonb,
	})

	}


	

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
