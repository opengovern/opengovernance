package doppler_account

import (
	"encoding/json"
	"github.com/jackc/pgtype"
	cohereaiDescriberLocal "github.com/opengovern/opencomply/services/integration/integration-type/cohereai-project/configs"
	"github.com/opengovern/opencomply/services/integration/integration-type/interfaces"
	"github.com/opengovern/opencomply/services/integration/integration-type/cohereai-project/healthcheck"
	"github.com/opengovern/opencomply/services/integration/integration-type/cohereai-project/discovery"

	"github.com/opengovern/opencomply/services/integration/models"
)

type DopplerAccountIntegration struct{}

func (i *DopplerAccountIntegration) GetConfiguration() interfaces.IntegrationConfiguration {
	return interfaces.IntegrationConfiguration{
		NatsScheduledJobsTopic:   cohereaiDescriberLocal.JobQueueTopic,
		NatsManualJobsTopic:      cohereaiDescriberLocal.JobQueueTopicManuals,
		NatsStreamName:           cohereaiDescriberLocal.StreamName,
		NatsConsumerGroup:        cohereaiDescriberLocal.ConsumerGroup,
		NatsConsumerGroupManuals: cohereaiDescriberLocal.ConsumerGroupManuals,

		SteampipePluginName: "doppler",

		UISpecFileName: "doppler-account.json",

		DescriberDeploymentName: cohereaiDescriberLocal.DescriberDeploymentName,
		DescriberRunCommand:     cohereaiDescriberLocal.DescriberRunCommand,
	}
}

func (i *DopplerAccountIntegration) HealthCheck(jsonData []byte, providerId string, labels map[string]string, annotations map[string]string) (bool, error) {
	var credentials cohereaiDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return false, err
	}

	isHealthy, err := healthcheck.CohereAIIntegrationHealthcheck(credentials.APIKey)
	return isHealthy, err
}

func (i *DopplerAccountIntegration) DiscoverIntegrations(jsonData []byte) ([]models.Integration, error) {
	var credentials cohereaiDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, err
	}
	var integrations []models.Integration
	connectors, err1 := discovery.CohereAIIntegrationDiscovery(credentials.APIKey)
	if err1 != nil {
		return nil, err1
	}
	labels := map[string]string{
		"ClientName": credentials.ClientName,
		
	}
	if(len(connectors) > 0){
		labels["OrganizationID"] = connectors[0].OrganizationID
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
	// for in esponse
	for _, connector := range connectors {
integrations = append(integrations, models.Integration{
		ProviderID: connector.ID,
		Name:       connector.Name,
		Labels:     integrationLabelsJsonb,
	})
	}

	

	return integrations, nil
}

func (i *DopplerAccountIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	return cohereaiDescriberLocal.ResourceTypesList, nil
}

func (i *DopplerAccountIntegration) GetResourceTypeFromTableName(tableName string) string {
	if v, ok := cohereaiDescriberLocal.TablesToResourceTypes[tableName]; ok {
		return v
	}

	return ""
}
