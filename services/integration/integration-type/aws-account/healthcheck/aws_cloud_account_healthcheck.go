package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
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

// AccountResult represents the outcome of validating an AWS account.
type AccountResult struct {
	AccountID string        `json:"account_id"` // The AWS account ID being validated.
	Healthy   bool          `json:"healthy"`    // Indicates if the account is healthy (accessible and has required policies).
	Details   PolicyDetails `json:"details"`    // Detailed information about policies and accessibility.
}

// PolicyDetails provides detailed information about the policies attached to the IAM principal.
type PolicyDetails struct {
	RequiredPolicies []string `json:"required_policies"` // List of required policy ARNs.
	AttachedPolicies []string `json:"attached_policies"` // List of policies attached to the principal.
	MissingPolicies  []string `json:"missing_policies"`  // List of required policies that are missing.
	CredentialType   string   `json:"credential_type"`   // Type of credentials used ("Single Account" or "Multi-Account").
	IsAccessible     bool     `json:"is_accessible"`     // Indicates if the account is accessible.
	IamPrincipal     string   `json:"iam_principal"`     // ARN of the IAM principal.
	HasPolicies      bool     `json:"has_policies"`      // Indicates if required policies are attached.
	Error            string   `json:"error,omitempty"`   // Error message, if any.
}

// List of required policy ARNs. Modify this list as needed.
var requiredPolicies = []string{
	"arn:aws:iam::aws:policy/SecurityAudit",
	// Add more policy ARNs as needed
}

func AWSIntegrationHealthCheck(creds AWSConfigInput, accountID string) (bool, error) {
	// Perform account validation
	result := ValidateIntegrationHealth(accountID, creds)
	if result.Details.Error != "" {
		return false, errors.New(result.Details.Error)
	}

	return result.Healthy, nil
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

// ValidateIntegrationHealth validates the integration health of the specified AWS account.
// It checks if the account is accessible and if the IAM principal has the required policies.
func ValidateIntegrationHealth(accountID string, creds AWSConfigInput) AccountResult {
	var result AccountResult
	result.AccountID = accountID
	result.Details.RequiredPolicies = requiredPolicies
	result.Details.CredentialType = "Single Account" // default, may change to "Multi-Account"

	// Create AWS Config using provided credentials
	awsCfg, err := GenerateAWSConfig(
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.RoleNameInPrimaryAccount,
		creds.CrossAccountRoleARN,
		creds.ExternalID,
		creds.Region,
	)
	if err != nil {
		result.Details.Error = fmt.Sprintf("Failed to generate AWS config: %v", err)
		result.Details.IsAccessible = false
		result.Details.HasPolicies = false
		result.Healthy = false
		return result
	}

	// Initialize STS client
	stsClient := sts.NewFromConfig(*awsCfg)

	// Determine if it's multi-account based on provided inputs
	isMultiAccount := creds.CrossAccountRoleARN != "" || creds.RoleNameInPrimaryAccount != ""

	if isMultiAccount {
		result.Details.CredentialType = "Multi-Account"
		// Assume roles are already handled in GenerateAWSConfig
		// Proceed to get caller identity
	}

	// Get Caller Identity to check access
	identityOutput, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		result.Details.IsAccessible = false
		result.Details.HasPolicies = false
		result.Details.Error = fmt.Sprintf("Failed to get caller identity: %v", err)
		result.Healthy = false
		return result
	}

	// Verify if the account ID matches
	if *identityOutput.Account != accountID {
		result.Details.IsAccessible = false
		result.Details.HasPolicies = false
		result.Details.Error = fmt.Sprintf("Provided credentials do not match the account ID: %s", accountID)
		result.Healthy = false
		return result
	}

	// Record the principal ARN
	result.Details.IsAccessible = true
	result.Details.IamPrincipal = *identityOutput.Arn

	// Check policies attached to the principal using utility functions
	attachedPolicies, missingPolicies, err := GetAttachedPolicies(*awsCfg, result.Details.IamPrincipal, requiredPolicies)
	if err != nil {
		result.Details.HasPolicies = false
		result.Details.Error = fmt.Sprintf("Error checking policies: %v", err)
		result.Healthy = false
		return result
	}

	result.Details.AttachedPolicies = attachedPolicies
	result.Details.MissingPolicies = missingPolicies
	if len(missingPolicies) > 0 {
		result.Details.HasPolicies = false
		result.Details.Error = fmt.Sprintf("Missing policies: %v", missingPolicies)
	} else {
		result.Details.HasPolicies = true
	}

	result.Healthy = result.Details.IsAccessible && result.Details.HasPolicies

	return result
}

// ParsePrincipalArn parses an AWS principal ARN and returns the entity type and entity name.
// This updated function handles assumed roles by extracting the actual role name.
func ParsePrincipalArn(principalArn string) (string, string, error) {
	parts := strings.Split(principalArn, ":")
	if len(parts) < 6 {
		return "", "", fmt.Errorf("invalid ARN format")
	}

	// parts[5] contains the resource part
	resource := parts[5]
	resourceParts := strings.SplitN(resource, "/", 2)
	if len(resourceParts) != 2 {
		return "", "", fmt.Errorf("invalid resource format in ARN")
	}

	entityType := resourceParts[0]
	entityName := resourceParts[1]

	if entityType == "assumed-role" {
		entityType = "role"
		// For assumed roles, entityName is "RoleName/SessionName"
		// We only need the RoleName
		roleParts := strings.SplitN(entityName, "/", 2)
		entityName = roleParts[0]
	}

	return entityType, entityName, nil
}

// GetAttachedPolicies retrieves the attached policies and identifies any missing required policies.
// It uses the ParsePrincipalArn function.
func GetAttachedPolicies(cfg aws.Config, principalArn string, requiredPolicies []string) ([]string, []string, error) {
	var attachedPolicies []string
	var missingPolicies []string

	iamClient := iam.NewFromConfig(cfg)

	entityType, entityName, err := ParsePrincipalArn(principalArn)
	if err != nil {
		return attachedPolicies, missingPolicies, fmt.Errorf("failed to parse principal ARN: %v", err)
	}

	attachedPoliciesMap := make(map[string]bool)

	switch entityType {
	case "user":
		// List policies attached to the user
		input := &iam.ListAttachedUserPoliciesInput{
			UserName: aws.String(entityName),
		}

		paginator := iam.NewListAttachedUserPoliciesPaginator(iamClient, input)

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(context.TODO())
			if err != nil {
				return attachedPolicies, missingPolicies, fmt.Errorf("failed to list attached policies for user %s: %v", entityName, err)
			}

			for _, policy := range page.AttachedPolicies {
				attachedPoliciesMap[*policy.PolicyArn] = true
			}
		}
	case "role":
		// List policies attached to the role
		input := &iam.ListAttachedRolePoliciesInput{
			RoleName: aws.String(entityName),
		}

		paginator := iam.NewListAttachedRolePoliciesPaginator(iamClient, input)

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(context.TODO())
			if err != nil {
				return attachedPolicies, missingPolicies, fmt.Errorf("failed to list attached policies for role %s: %v", entityName, err)
			}

			for _, policy := range page.AttachedPolicies {
				attachedPoliciesMap[*policy.PolicyArn] = true
			}
		}
	default:
		return attachedPolicies, missingPolicies, fmt.Errorf("unsupported entity type: %s", entityType)
	}

	// Convert map to slice
	for arn := range attachedPoliciesMap {
		attachedPolicies = append(attachedPolicies, arn)
	}

	// Check for missing policies
	for _, policyArn := range requiredPolicies {
		if !attachedPoliciesMap[policyArn] {
			missingPolicies = append(missingPolicies, policyArn)
		}
	}

	return attachedPolicies, missingPolicies, nil
}
