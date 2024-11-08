package aws_account

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	//awsDescriber "github.com/opengovern/og-aws-describer/aws"
	//awsDescriberLocal "github.com/opengovern/og-aws-describer/local"
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/opengovernance/services/integration/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration/models"
)

const (
	IntegrationTypeAWSAccount integration.Type = "AWS_ACCOUNT"
)

type AWSCredentialType interface {
	interfaces.CredentialType
	CreateAWSSession() (*aws.Config, error)
}

type AWSAccountIntegration struct{}

func CreateAWSAccountIntegration() (interfaces.IntegrationType, error) {
	return &AWSAccountIntegration{}, nil
}

var CredentialTypes = map[string]interfaces.CredentialCreator{
	"aws_simple_iam_credentials":              CreateAWSSimpleIAMCredentials,
	"aws_iam_credentials_role_with_role_jump": CreateAWSIAMCredentialsRoleWithRoleJump,
}

func (i *AWSAccountIntegration) GetDescriberConfiguration() interfaces.DescriberConfiguration {
	return interfaces.DescriberConfiguration{
		NatsScheduledJobsTopic: "og_aws_describer_job_queue",
		NatsManualJobsTopic:    "og_aws_describer_manuals_job_queue",
		NatsStreamName:         "og_aws_describer",
	}
}

func (i *AWSAccountIntegration) GetAnnotations(jsonData []byte) (map[string]string, error) {
	annotations := make(map[string]string)

	return annotations, nil
}

func (i *AWSAccountIntegration) GetLabels(jsonData []byte) (map[string]string, error) {
	labels := make(map[string]string)
	credentialType := "aws_simple_iam_credentials"
	awsCredential, err := getCredentials(credentialType, jsonData)
	if err != nil {
		return nil, err
	}

	cfg, err := awsCredential.CreateAWSSession()
	if err != nil {
		return nil, err
	}

	// Check if the account is a standalone account (not part of any organization)
	isStandalone, err := CheckStandaloneNonOrganizationAccount(*cfg)
	if err != nil {
		return nil, err
	}
	if isStandalone {
		labels["aws_cloud_account/account-type"] = "standalone"
		return labels, nil
	}

	// Check if the account is a member of an AWS Organization
	isMember, err := CheckOrganizationMemberAccount(*cfg)
	if err != nil {
		return nil, err
	}
	if isMember {
		labels["aws_cloud_account/account-type"] = "organization-member"
		return labels, nil
	}

	// Check if the account is the master account
	isMaster, err := CheckMasterAccount(*cfg)
	if err != nil {
		return nil, err
	}
	if isMaster {
		labels["aws_cloud_account/account-type"] = "organization-master"
		return labels, nil
	}

	return labels, nil
}

func (i *AWSAccountIntegration) HealthCheck(jsonData []byte, providerId string, labels map[string]string) (bool, error) {
	credentialType := "aws_simple_iam_credentials"

	awsCredential, err := getCredentials(credentialType, jsonData)
	if err != nil {
		return false, fmt.Errorf("failed to parse AWS credentials of type %s: %s", credentialType, err.Error())
	}

	return awsCredential.HealthCheck()
}

func (i *AWSAccountIntegration) DiscoverIntegrations(jsonData []byte) ([]models.Integration, error) {
	credentialType := "aws_simple_iam_credentials"

	awsCredential, err := getCredentials(credentialType, jsonData)
	if err != nil {
		return nil, err
	}

	return awsCredential.DiscoverIntegrations()
}

func (i *AWSAccountIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	//return awsDescriber.ListResourceTypes(), nil
	return nil, nil
}

func getCredentials(credentialType string, jsonData []byte) (AWSCredentialType, error) {
	if _, ok := CredentialTypes[credentialType]; !ok {
		return nil, fmt.Errorf("invalid credential type: %s", credentialType)
	}
	credentialCreator := CredentialTypes[credentialType]
	credential, err := credentialCreator(jsonData)
	if err != nil {
		return nil, err
	}
	awsCredential, ok := credential.(AWSCredentialType)
	if !ok {
		return nil, fmt.Errorf("credential is not of type AWSCredentialType")
	}

	return awsCredential, nil
}

func (i *AWSAccountIntegration) GetResourceTypeFromTableName(tableName string) string {
	return ""
}
