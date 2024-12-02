package labels

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	"golang.org/x/net/context"
)

// AWSConfigInput encapsulates all possible AWS credentials and role information.
type AWSConfigInput struct {
	AccessKeyID              string `json:"aws_access_key_id"`     // Changed from "access_key_id"
	SecretAccessKey          string `json:"aws_secret_access_key"` // Changed from "secret_access_key"
	RoleNameInPrimaryAccount string `json:"role_name_in_primary_account"`
	CrossAccountRoleARN      string `json:"cross_account_role_arn"`
	ExternalID               string `json:"external_id"`
	Region                   string `json:"region"`
}

// IsOrganizationMasterAccount checks if the current AWS account is part of an AWS Organization
// and if it is the management (master) account of that organization.
// Returns true if both conditions are met, otherwise false.
func IsOrganizationMasterAccount(ctx context.Context, creds AWSConfigInput) (bool, error) {
	cfg, err := GenerateAWSConfig(
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.RoleNameInPrimaryAccount,
		creds.CrossAccountRoleARN,
		creds.ExternalID,
		creds.Region,
	)
	if err != nil {
		return false, err
	}

	// Create an STS client
	stsClient := sts.NewFromConfig(*cfg)

	// Get the caller identity
	callerIdentityOutput, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return false, fmt.Errorf("unable to get caller identity: %w", err)
	}

	currentAccountID := aws.ToString(callerIdentityOutput.Account)
	if currentAccountID == "" {
		return false, fmt.Errorf("unable to determine current account ID")
	}

	// Create an Organizations client
	orgClient := organizations.NewFromConfig(*cfg)

	// Describe the organization
	describeOrgOutput, err := orgClient.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
	if err != nil {
		var orgErr smithy.APIError
		if errors.As(err, &orgErr) {
			// If the error code is AWSOrganizationsNotInUseException, the account is not part of an organization
			if orgErr.ErrorCode() == "AWSOrganizationsNotInUseException" {
				return false, nil
			}
		}
		// For other errors, return the error
		return false, fmt.Errorf("unable to describe organization: %w", err)
	}

	if describeOrgOutput.Organization == nil {
		// Organization is not present
		return false, nil
	}

	// Get the management account ID
	managementAccountID := aws.ToString(describeOrgOutput.Organization.MasterAccountId)
	if managementAccountID == "" {
		return false, fmt.Errorf("unable to determine management account ID")
	}

	// Compare the current account ID with the management account ID
	isMaster := currentAccountID == managementAccountID

	return isMaster, nil
}

// GenerateAWSConfig initializes and returns an AWS configuration based on the provided inputs.
// It determines whether to perform single or multi-account validation based on the inputs.
func GenerateAWSConfig(
	accessKeyID string,
	secretAccessKey string,
	roleNameInPrimaryAccount string,
	crossAccountRoleARN string,
	externalID string,
	region string,
) (*aws.Config, error) {
	// Step 1: Set default region if not provided
	if region == "" {
		region = "us-east-2"
	}

	// Step 2: Initialize the base credentials provider
	if accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("AccessKeyID and SecretAccessKey must be provided")
	}
	baseCredentials := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""))

	// Step 3: Create the AWS Config manually
	cfg := aws.Config{
		Region:      region,
		Credentials: baseCredentials,
	}

	// Step 4: Determine the type of validation based on provided inputs
	isMultiAccount := crossAccountRoleARN != "" || roleNameInPrimaryAccount != ""

	// Step 5: Assume roles if needed
	if isMultiAccount {
		// Create an STS client from the existing configuration
		stsClient := sts.NewFromConfig(cfg)

		// Assume Role in Primary Account if RoleNameInPrimaryAccount is provided
		if roleNameInPrimaryAccount != "" {
			primaryRoleARN := roleNameInPrimaryAccount // Expected to be full ARN

			// Configure AssumeRole options
			primaryAssumeRoleOptions := func(o *stscreds.AssumeRoleOptions) {
				o.RoleSessionName = "primary-account-session"
				// Optional: o.DurationSeconds = 3600
			}

			// Create an AssumeRole provider for the primary account role
			primaryRoleProvider := stscreds.NewAssumeRoleProvider(stsClient, primaryRoleARN, primaryAssumeRoleOptions)

			// Cache the credentials
			primaryCredentials := aws.NewCredentialsCache(primaryRoleProvider)

			// Update the AWS configuration to use the assumed primary role credentials
			cfg.Credentials = primaryCredentials

			// Update STS client with new credentials
			stsClient = sts.NewFromConfig(cfg)
		}

		// Assume Role in Cross Account if CrossAccountRoleARN is provided
		if crossAccountRoleARN != "" {
			// Configure AssumeRole options
			crossAccountAssumeRoleOptions := func(o *stscreds.AssumeRoleOptions) {
				o.RoleSessionName = "cross-account-session"
				if externalID != "" {
					o.ExternalID = aws.String(externalID)
				}
				// Optional: o.DurationSeconds = 3600
			}

			// Create an AssumeRole provider for the cross account role
			crossAccountRoleProvider := stscreds.NewAssumeRoleProvider(stsClient, crossAccountRoleARN, crossAccountAssumeRoleOptions)

			// Cache the credentials
			crossAccountCredentials := aws.NewCredentialsCache(crossAccountRoleProvider)

			// Update the AWS configuration to use the assumed cross account role credentials
			cfg.Credentials = crossAccountCredentials
		}
	}

	return &cfg, nil
}
