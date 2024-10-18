package aws_account

import "github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"

func CreateAWSAccountIntegration(credentialType string, jsonData []byte) (interfaces.CredentialType, map[string]any, error) {
	credentialCreator := CredentialTypes[credentialType]
	return credentialCreator(jsonData)
}

var CredentialTypes = map[string]interfaces.CredentialCreator{
	"aws_simple_iam_credentials":              CreateAWSSimpleIAMCredentials,
	"aws_iam_credentials_role_with_role_jump": CreateAWSIAMCredentialsRoleWithRoleJump,
}
