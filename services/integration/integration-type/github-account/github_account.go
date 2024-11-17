package github_account

import (
	"encoding/json"
	"github.com/jackc/pgtype"
	githubDescriberLocal "github.com/opengovern/opengovernance/services/integration/integration-type/github-account/configs"
	"github.com/opengovern/opengovernance/services/integration/integration-type/github-account/discovery"
	"github.com/opengovern/opengovernance/services/integration/integration-type/github-account/healthcheck"
	"github.com/opengovern/opengovernance/services/integration/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration/models"
)

type GithubAccountIntegration struct{}

func (i *GithubAccountIntegration) GetConfiguration() interfaces.IntegrationConfiguration {
	return interfaces.IntegrationConfiguration{
		NatsScheduledJobsTopic: githubDescriberLocal.JobQueueTopic,
		NatsManualJobsTopic:    githubDescriberLocal.JobQueueTopicManuals,
		NatsStreamName:         githubDescriberLocal.StreamName,

		SteampipePluginName: "github",

		UISpecFileName: "github-account.json",
	}
}

func (i *GithubAccountIntegration) HealthCheck(jsonData []byte, providerId string, labels map[string]string, annotations map[string]string) (bool, error) {
	var credentials githubDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return false, err
	}

	isHealthy, _, err := healthcheck.GithubIntegrationHealthcheck(healthcheck.Config{
		Token:          credentials.Token,
		BaseURL:        credentials.BaseURL,
		AppId:          credentials.AppId,
		InstallationId: credentials.InstallationId,
		PrivateKeyPath: credentials.PrivateKeyPath,
	})
	return isHealthy, err
}

func (i *GithubAccountIntegration) DiscoverIntegrations(jsonData []byte) ([]models.Integration, error) {
	var credentials githubDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, err
	}
	var integrations []models.Integration
	accounts, err := discovery.GithubIntegrationDiscovery(discovery.Config{
		Token:          credentials.Token,
		BaseURL:        credentials.BaseURL,
		AppId:          credentials.AppId,
		InstallationId: credentials.InstallationId,
		PrivateKeyPath: credentials.PrivateKeyPath,
	})
	for _, a := range accounts {
		labels := map[string]string{
			"AccountType": a.Type,
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
			ProviderID: a.ID,
			Name:       a.Name,
			Labels:     integrationLabelsJsonb,
		})
	}
	return integrations, nil
}

func (i *GithubAccountIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	return githubDescriberLocal.ResourceTypesList, nil
}

func (i *GithubAccountIntegration) GetResourceTypeFromTableName(tableName string) string {
	if v, ok := githubDescriberLocal.TablesToResourceTypes[tableName]; ok {
		return v
	}

	return ""
}
