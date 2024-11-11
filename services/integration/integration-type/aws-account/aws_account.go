package aws_account

import (
	"encoding/json"
	"fmt"
	"github.com/jackc/pgtype"
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

func (i *AwsCloudAccountIntegration) HealthCheck(jsonData []byte, providerId string, labels map[string]string, annotations map[string]string) (bool, error) {
	var credentials awsDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return false, err
	}

	return healthcheck.AWSIntegrationHealthCheck(healthcheck.AWSConfigInput{
		AccessKeyID:              credentials.AwsAccessKeyID,
		SecretAccessKey:          credentials.AwsSecretAccessKey,
		RoleNameInPrimaryAccount: credentials.RoleToAssumeInMainAccount,
		CrossAccountRoleARN:      credentials.CrossAccountRoleName,
		ExternalID:               credentials.ExternalID,
	}, providerId)
}

func (i *AwsCloudAccountIntegration) DiscoverIntegrations(jsonData []byte) ([]models.Integration, error) {
	var credentials awsDescriberLocal.IntegrationCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, err
	}

	var integrations []models.Integration
	accounts := discovery.AWSIntegrationDiscovery(discovery.Config{
		AWSAccessKeyID:                credentials.AwsAccessKeyID,
		AWSSecretAccessKey:            credentials.AwsSecretAccessKey,
		RoleNameToAssumeInMainAccount: credentials.RoleToAssumeInMainAccount,
		CrossAccountRoleName:          credentials.CrossAccountRoleName,
		ExternalID:                    credentials.ExternalID,
	})
	for _, a := range accounts {
		if a.Details.Error != "" {
			return nil, fmt.Errorf(a.Details.Error)
		}

		labels := map[string]string{
			"RoleNameInMainAccount": a.Labels.RoleNameInMainAccount,
			"AccountType":           a.Labels.AccountType,
			"CrossAccountRoleARN":   a.Labels.CrossAccountRoleARN,
			"ExternalID":            a.Labels.ExternalID,
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
			ProviderID: a.AccountID,
			Name:       a.AccountName,
			Labels:     integrationLabelsJsonb,
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
