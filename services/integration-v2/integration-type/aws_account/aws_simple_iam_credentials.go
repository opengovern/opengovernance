package aws_account

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/uuid"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
	"time"
)

// AWSSimpleIAMCredentials represents AWS single account credentials.
type AWSSimpleIAMCredentials struct {
	AccessKeyID     string  `json:"access_key_id" binding:"required"`
	AccessKeySecret string  `json:"access_key_secret" binding:"required"`
	RoleToAssume    *string `json:"role_to_assume,omitempty"`
}

func CreateAWSSimpleIAMCredentials(jsonData []byte) (interfaces.CredentialType, error) {
	var creds AWSSimpleIAMCredentials
	err := json.Unmarshal(jsonData, &creds)
	if err != nil {
		return nil, err
	}

	return &creds, nil
}

func (c *AWSSimpleIAMCredentials) HealthCheck() (bool, error) {
	creds := credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.AccessKeySecret, "")

	// Load the AWS configuration with the custom credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(creds),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		return false, fmt.Errorf("failed to load AWS config: %v", err)
	}

	stsClient := sts.NewFromConfig(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return false, fmt.Errorf("failed to validate AWS credentials: %v", err)
	}

	return true, nil
}

func (c *AWSSimpleIAMCredentials) DiscoverIntegrations() ([]models.Integration, error) {
	cfg, err := c.CreateAWSSession()
	if err != nil {
		return nil, err
	}

	orgClient := organizations.NewFromConfig(*cfg)

	// List AWS accounts using Organizations service
	accounts := make([]types.Account, 0)
	paginator := organizations.NewListAccountsPaginator(orgClient, &organizations.ListAccountsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to list AWS accounts: %v", err)
		}

		for _, account := range page.Accounts {
			if account.Id != nil {
				accounts = append(accounts, account)
			}
		}
	}

	var integrations []models.Integration
	for _, acc := range accounts {
		var name, id string
		if acc.Name != nil {
			name = *acc.Name
		}
		if acc.Id != nil {
			id = *acc.Id
		}
		integrations = append(integrations, models.Integration{
			IntegrationTracker: uuid.New(),
			IntegrationID:      id,
			IntegrationName:    name,
			Connector:          "AWS",
			Type:               "aws_account",
			OnboardDate:        time.Now(),
		})
	}
	return integrations, nil
}

func (c *AWSSimpleIAMCredentials) CreateAWSSession() (*aws.Config, error) {
	// Create custom credentials provider
	creds := credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.AccessKeySecret, "")

	// Load the AWS configuration with the custom credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(creds),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		return &cfg, fmt.Errorf("unable to load SDK config, %v", err)
	}

	return &cfg, nil
}
