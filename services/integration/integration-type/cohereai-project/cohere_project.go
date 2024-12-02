package cohereai_project

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	cohereaiDescriberLocal "github.com/opengovern/opencomply/services/integration/integration-type/cohereai-project/configs"
	"github.com/opengovern/opencomply/services/integration/integration-type/interfaces"
	"github.com/opengovern/opencomply/services/integration/integration-type/openai-project/healthcheck"
	"github.com/opengovern/opencomply/services/integration/models"
)

type CohereAIProjectIntegration struct{}

func (i *CohereAIProjectIntegration) GetConfiguration() interfaces.IntegrationConfiguration {
	return interfaces.IntegrationConfiguration{
		NatsScheduledJobsTopic:   cohereaiDescriberLocal.JobQueueTopic,
		NatsManualJobsTopic:      cohereaiDescriberLocal.JobQueueTopicManuals,
		NatsStreamName:           cohereaiDescriberLocal.StreamName,
		NatsConsumerGroup:        cohereaiDescriberLocal.ConsumerGroup,
		NatsConsumerGroupManuals: cohereaiDescriberLocal.ConsumerGroupManuals,

		SteampipePluginName: "cohereai",

		UISpecFileName: "cohereai-project.json",

		DescriberDeploymentName: cohereaiDescriberLocal.DescriberDeploymentName,
		DescriberRunCommand:     cohereaiDescriberLocal.DescriberRunCommand,
	}
}

func (i *CohereAIProjectIntegration) HealthCheck(jsonData []byte, providerId string, labels map[string]string, annotations map[string]string) (bool, error) {
	var credentials cohereaiDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return false, err
	}

	isHealthy, err := healthcheck.OpenAIIntegrationHealthcheck(credentials.APIKey)
	return isHealthy, err
}

func (i *CohereAIProjectIntegration) DiscoverIntegrations(jsonData []byte) ([]models.Integration, error) {
	var credentials cohereaiDescriberLocal.IntegrationCredentials
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
		"ApiKey":     credentials.APIKey,
		"ClientName": credentials.ClientName,
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
		ProviderID: uuid.New().String(),
		Name:       credentials.ClientName,
		Labels:     integrationLabelsJsonb,
	})

	return integrations, nil
}

func (i *CohereAIProjectIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	return cohereaiDescriberLocal.ResourceTypesList, nil
}

func (i *CohereAIProjectIntegration) GetResourceTypeFromTableName(tableName string) string {
	if v, ok := cohereaiDescriberLocal.TablesToResourceTypes[tableName]; ok {
		return v
	}

	return ""
}
