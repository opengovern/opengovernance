package oci_repository

import (
	"context"
	"encoding/json"
	"github.com/opengovern/opencomply/services/integration/integration-type/interfaces"
	"github.com/opengovern/opencomply/services/integration/integration-type/oci-repository/configs"
	"github.com/opengovern/opencomply/services/integration/integration-type/oci-repository/healthcheck"
	"github.com/opengovern/opencomply/services/integration/models"
)

type Integration struct{}

func (i *Integration) GetConfiguration() interfaces.IntegrationConfiguration {
	return interfaces.IntegrationConfiguration{
		NatsScheduledJobsTopic:   configs.JobQueueTopic,
		NatsManualJobsTopic:      configs.JobQueueTopicManuals,
		NatsStreamName:           configs.StreamName,
		NatsConsumerGroup:        configs.ConsumerGroup,
		NatsConsumerGroupManuals: configs.ConsumerGroupManuals,

		SteampipePluginName: "oci",

		UISpecFileName: "oci-integration.json",

		DescriberDeploymentName: configs.DescriberDeploymentName,
		DescriberRunCommand:     configs.DescriberRunCommand,
	}
}

func (i *Integration) HealthCheck(jsonData []byte, _ string, _ map[string]string, _ map[string]string) (bool, error) {
	var credentials configs.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return false, err
	}

	return healthcheck.IntegrationHealthcheck(context.TODO(), healthcheck.Config{})
}

func (i *Integration) DiscoverIntegrations(jsonData []byte) ([]models.Integration, error) {
	return []models.Integration{}, nil
}

func (i *Integration) GetResourceTypesByLabels(_ map[string]string) ([]string, error) {
	return configs.ResourceTypesList, nil
}

func (i *Integration) GetResourceTypeFromTableName(tableName string) string {
	if v, ok := configs.TablesToResourceTypes[tableName]; ok {
		return v
	}

	return ""
}
