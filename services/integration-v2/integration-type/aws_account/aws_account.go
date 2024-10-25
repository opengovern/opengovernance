package aws_account

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsDescriberLocal "github.com/opengovern/og-aws-describer/local"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

type AWSCredentialType interface {
	interfaces.CredentialType
	CreateAWSSession() (*aws.Config, error)
}

type AWSAccountIntegration struct {
	Credential AWSCredentialType
}

func CreateAWSAccountIntegration(credentialType *string, jsonData []byte) (interfaces.IntegrationType, error) {
	integration := AWSAccountIntegration{}

	if credentialType != nil {
		if _, ok := CredentialTypes[*credentialType]; !ok {
			return nil, fmt.Errorf("invalid credential type: %s", credentialType)
		}
		credentialCreator := CredentialTypes[*credentialType]
		credential, err := credentialCreator(jsonData)
		if err != nil {
			return nil, err
		}
		awsCredential, ok := credential.(AWSCredentialType)
		if !ok {
			return nil, fmt.Errorf("credential is not of type AWSCredentialType")
		}
		integration.Credential = awsCredential
	}
	return &integration, nil
}

var CredentialTypes = map[string]interfaces.CredentialCreator{
	"aws_simple_iam_credentials":              CreateAWSSimpleIAMCredentials,
	"aws_iam_credentials_role_with_role_jump": CreateAWSIAMCredentialsRoleWithRoleJump,
}

func (i *AWSAccountIntegration) GetDescriberConfiguration() interfaces.DescriberConfiguration {
	return interfaces.DescriberConfiguration{
		NatsScheduledJobsTopic: awsDescriberLocal.JobQueueTopic,
		NatsManualJobsTopic:    awsDescriberLocal.JobQueueTopicManuals,
	}
}

func (i *AWSAccountIntegration) GetAnnotations() (map[string]any, error) {
	annotations := make(map[string]any)

	return annotations, nil
}

func (i *AWSAccountIntegration) GetMetadata() (map[string]any, error) {
	annotations := make(map[string]any)

	return annotations, nil
}

func (i *AWSAccountIntegration) GetLabels() (map[string]any, error) {
	labels := make(map[string]any)

	cfg, err := i.Credential.CreateAWSSession()
	if err != nil {
		return nil, err
	}

	// Check if the account is a standalone account (not part of any organization)
	isStandalone, err := CheckStandaloneNonOrganizationAccount(*cfg)
	if err != nil {
		return nil, err
	}
	if isStandalone {
		labels["aws/account-type"] = "standalone"
		return labels, nil
	}

	// Check if the account is a member of an AWS Organization
	isMember, err := CheckOrganizationMemberAccount(*cfg)
	if err != nil {
		return nil, err
	}
	if isMember {
		labels["aws/account-type"] = "organization-member"
		return labels, nil
	}

	// Check if the account is the master account
	isMaster, err := CheckMasterAccount(*cfg)
	if err != nil {
		return nil, err
	}
	if isMaster {
		labels["aws/account-type"] = "organization-master"
		return labels, nil
	}

	return labels, nil
}

func (i *AWSAccountIntegration) HealthCheck() error {
	return i.Credential.HealthCheck()
}

func (i *AWSAccountIntegration) DiscoverIntegrations() ([]models.Integration, error) {
	return i.Credential.DiscoverIntegrations()
}

func (i *AWSAccountIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	return nil, nil
}
