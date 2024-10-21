package aws_account

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

// AWSIAMCredentialsRoleWithRoleJump represents AWS cross-account credentials.
type AWSIAMCredentialsRoleWithRoleJump struct {
	AccessKeyID              string  `json:"access_key_id" binding:"required"`
	AccessKeySecret          string  `json:"access_key_secret" binding:"required"`
	CrossAccountRoleToAssume string  `json:"cross_account_role_to_assume" binding:"required"`
	RoleToAssume             *string `json:"role_to_assume,omitempty"`
	ExternalID               *string `json:"external_id,omitempty"`
}

func CreateAWSIAMCredentialsRoleWithRoleJump(jsonData []byte) (interfaces.CredentialType, error) {
	var credentials AWSIAMCredentialsRoleWithRoleJump
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, err
	}

	return &credentials, nil
}

func (c *AWSIAMCredentialsRoleWithRoleJump) HealthCheck() error {
	return nil
}

func (c *AWSIAMCredentialsRoleWithRoleJump) DiscoverIntegrations() ([]models.Integration, error) {
	return nil, nil
}

func (c *AWSIAMCredentialsRoleWithRoleJump) CreateAWSSession() (*aws.Config, error) {
	return nil, nil
}
