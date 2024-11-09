package aws_account

import (
	"encoding/json"
	awsDescriberLocal "github.com/opengovern/opengovernance/services/integration/integration-type/aws-account/configs"
	"github.com/opengovern/opengovernance/services/integration/integration-type/aws-account/discovery"
	"github.com/opengovern/opengovernance/services/integration/integration-type/aws-account/healthcheck"
	"github.com/opengovern/opengovernance/services/integration/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration/models"
)

type AwsCloudAccountIntegration struct{}

func CreateAwsCloudAccountIntegration() (interfaces.IntegrationType, error) {
	return &AwsCloudAccountIntegration{}, nil
}

func (i *AwsCloudAccountIntegration) GetDescriberConfiguration() interfaces.DescriberConfiguration {
	return interfaces.DescriberConfiguration{
		NatsScheduledJobsTopic: awsDescriberLocal.JobQueueTopic,
		NatsManualJobsTopic:    awsDescriberLocal.JobQueueTopicManuals,
		NatsStreamName:         awsDescriberLocal.StreamName,
	}
}

func (i *AwsCloudAccountIntegration) GetAnnotations(jsonData []byte) (map[string]string, error) {
	annotations := make(map[string]string)

	return annotations, nil
}

func (i *AwsCloudAccountIntegration) GetLabels(jsonData []byte) (map[string]string, error) {
	annotations := make(map[string]string)

	return annotations, nil
}

func (i *AwsCloudAccountIntegration) HealthCheck(jsonData []byte, providerId string, labels map[string]string) (bool, error) {
	var credentials awsDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return false, err
	}

	return healthcheck.AWSIntegrationHealthCheck(healthcheck.Config{
		AWSAccessKeyID:            credentials.AwsAccessKeyID,
		AWSSecretAccessKey:        credentials.AwsSecretAccessKey,
		RoleToAssumeInMainAccount: credentials.RoleToAssumeInMainAccount,
		CrossAccountRole:          credentials.CrossAccountRoleName,
		ExternalID:                credentials.ExternalID,
	}, providerId)
}

func (i *AwsCloudAccountIntegration) DiscoverIntegrations(jsonData []byte) ([]models.Integration, error) {
	var credentials awsDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, err
	}

	var integrations []models.Integration
	accounts, err := discovery.AWSIntegrationDiscovery(discovery.Config{
		AWSAccessKeyID:            credentials.AwsAccessKeyID,
		AWSSecretAccessKey:        credentials.AwsSecretAccessKey,
		RoleToAssumeInMainAccount: credentials.RoleToAssumeInMainAccount,
		CrossAccountRole:          credentials.CrossAccountRoleName,
		ExternalID:                credentials.ExternalID,
	})
	if err != nil {
		return nil, err
	}
	for _, a := range accounts {
		integrations = append(integrations, models.Integration{
			ProviderID: a.AccountID,
			Name:       a.AccountName,
		})
	}

	return integrations, nil
}

func (i *AwsCloudAccountIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	return awsDescriberLocal.ResourceTypesList, nil
}

func (i *AwsCloudAccountIntegration) GetResourceTypeFromTableName(tableName string) string {
	return ""
}
