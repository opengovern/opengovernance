package openai_project

import (
	"encoding/json"
	"github.com/jackc/pgtype"
	"github.com/opengovern/opencomply/services/integration/integration-type/interfaces"
	linodeDescriberLocal "github.com/opengovern/opencomply/services/integration/integration-type/linode-account/configs"
	"github.com/opengovern/opencomply/services/integration/integration-type/linode-account/discovery"
	"github.com/opengovern/opencomply/services/integration/integration-type/linode-account/healthcheck"
	"github.com/opengovern/opencomply/services/integration/models"
	"strconv"
)

type OpenAIProjectIntegration struct{}

func (i *OpenAIProjectIntegration) GetConfiguration() interfaces.IntegrationConfiguration {
	return interfaces.IntegrationConfiguration{
		NatsScheduledJobsTopic:   linodeDescriberLocal.JobQueueTopic,
		NatsManualJobsTopic:      linodeDescriberLocal.JobQueueTopicManuals,
		NatsStreamName:           linodeDescriberLocal.StreamName,
		NatsConsumerGroup:        linodeDescriberLocal.ConsumerGroup,
		NatsConsumerGroupManuals: linodeDescriberLocal.ConsumerGroupManuals,

		SteampipePluginName: "linode",

		UISpecFileName: "openai-project.json",

		DescriberDeploymentName: linodeDescriberLocal.DescriberDeploymentName,
		DescriberImageAddress:   linodeDescriberLocal.DescriberImageAddress,
		DescriberImageTagKey:    linodeDescriberLocal.DescriberImageTagKey,
		DescriberRunCommand:     linodeDescriberLocal.DescriberRunCommand,
	}
}

func (i *OpenAIProjectIntegration) HealthCheck(jsonData []byte, providerId string, labels map[string]string, annotations map[string]string) (bool, error) {
	var credentials linodeDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return false, err
	}

	isHealthy, err := healthcheck.LinodeIntegrationHealthcheck(credentials.Token)
	return isHealthy, err
}

func (i *OpenAIProjectIntegration) DiscoverIntegrations(jsonData []byte) ([]models.Integration, error) {
	var credentials linodeDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, err
	}
	var integrations []models.Integration
	account, err := discovery.LinodeIntegrationDiscovery(credentials.Token)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{
		"Email": account.Email,
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
		ProviderID: strconv.Itoa(account.ID),
		Name:       account.Username,
		Labels:     integrationLabelsJsonb,
	})

	return integrations, nil
}

func (i *OpenAIProjectIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	return linodeDescriberLocal.ResourceTypesList, nil
}

func (i *OpenAIProjectIntegration) GetResourceTypeFromTableName(tableName string) string {
	if v, ok := linodeDescriberLocal.TablesToResourceTypes[tableName]; ok {
		return v
	}

	return ""
}
