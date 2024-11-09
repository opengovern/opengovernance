package healthcheck

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsarn "github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Config represents the configuration loaded from the JSON file.
type Config struct {
	AWSAccessKeyID            string `json:"aws_access_key_id"`
	AWSSecretAccessKey        string `json:"aws_secret_access_key"`
	RoleToAssumeInMainAccount string `json:"role_to_assume_in_main_account,omitempty"`
	CrossAccountRole          string `json:"cross_account_role,omitempty"`
	ExternalID                string `json:"external_id,omitempty"`
}

func AWSIntegrationHealthCheck(config Config, accountID string) (bool, error) {
	result := ValidateAccount(accountID, config)

	var err error
	if result.Details.Error != "" {
		err = fmt.Errorf(result.Details.Error)
	}
	return result.Healthy, err
}

// AccountResult holds the result for the account.
type AccountResult struct {
	AccountID string  `json:"account_id"`
	Healthy   bool    `json:"healthy"`
	Details   Details `json:"details"`
}

// Details holds detailed information for the account.
type Details struct {
	IsAccessible     bool     `json:"isAccessible"`
	HasPolicies      bool     `json:"hasPolicies"`
	AttachedPolicies []string `json:"attached_policies,omitempty"`
	RequiredPolicies []string `json:"required_policies,omitempty"`
	IamPrincipal     string   `json:"iam_principal,omitempty"`
	CredentialType   string   `json:"credential_type"`
	Error            string   `json:"error,omitempty"`
}

// List of required policy ARNs. Modify this list as needed.
var requiredPolicies = []string{
	"arn:aws:iam::aws:policy/SecurityAudit",
	// Add more policy ARNs as needed
}

// GenerateAWSConfig creates an AWS configuration using the provided credentials provider.
func GenerateAWSConfig(credsProvider aws.CredentialsProvider) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credsProvider),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load configuration: %v", err)
	}
	return cfg, nil
}

// GetAttachedPolicies retrieves the attached policies and identifies any missing required policies.
func GetAttachedPolicies(cfg aws.Config, principalArn string, requiredPolicies []string) ([]string, []string, error) {
	var attachedPolicies []string
	var missingPolicies []string

	iamClient := iam.NewFromConfig(cfg)

	entityType, entityName, err := parsePrincipalArn(principalArn)
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

// parsePrincipalArn parses the ARN and returns the entity type and name.
func parsePrincipalArn(principalArn string) (entityType string, entityName string, err error) {
	// Parse the ARN
	parsedArn, err := awsarn.Parse(principalArn)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse ARN: %v", err)
	}

	switch parsedArn.Service {
	case "iam":
		// Resource format: {entityType}/{entityName}
		resourceParts := strings.SplitN(parsedArn.Resource, "/", 2)
		if len(resourceParts) < 2 {
			return "", "", fmt.Errorf("invalid resource format in ARN: %s", principalArn)
		}

		entityType = strings.ToLower(resourceParts[0])
		entityName = resourceParts[1]
	case "sts":
		// Resource format: assumed-role/{role-name}/{session-name}
		resourceParts := strings.SplitN(parsedArn.Resource, "/", 3)
		if len(resourceParts) < 2 {
			return "", "", fmt.Errorf("invalid resource format in ARN: %s", principalArn)
		}

		if strings.ToLower(resourceParts[0]) != "assumed-role" {
			return "", "", fmt.Errorf("unsupported resource type in STS ARN: %s", resourceParts[0])
		}

		entityType = "role"
		entityName = resourceParts[1]
	default:
		return "", "", fmt.Errorf("unsupported service in ARN: %s", parsedArn.Service)
	}

	return entityType, entityName, nil
}

// ValidateAccount attempts to access the specified account using the provided credentials and configurations.
// It returns an AccountResult indicating whether the account is accessible and has required policies.
func ValidateAccount(accountID string, cfg Config) AccountResult {
	var result AccountResult
	result.AccountID = accountID
	result.Details.RequiredPolicies = requiredPolicies
	result.Details.CredentialType = "Single Account" // default, may change to "Multi-Account"

	// Create AWS Config using provided credentials
	credsProvider := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		"",
	))

	awsCfg, err := GenerateAWSConfig(credsProvider)
	if err != nil {
		result.Details.Error = fmt.Sprintf("Failed to generate AWS config: %v", err)
		result.Details.IsAccessible = false
		result.Details.HasPolicies = false
		result.Healthy = false
		return result
	}

	// Initialize STS client
	stsClient := sts.NewFromConfig(awsCfg)

	// Check if we need to assume a role
	var roleToAssume string
	if cfg.CrossAccountRole != "" {
		// Use CrossAccountRole to assume role in target account
		roleToAssume = cfg.CrossAccountRole
		result.Details.CredentialType = "Multi-Account"
	} else if cfg.RoleToAssumeInMainAccount != "" {
		// Use RoleToAssumeInMainAccount to assume role in target account
		roleToAssume = cfg.RoleToAssumeInMainAccount
		result.Details.CredentialType = "Multi-Account"
	}

	if roleToAssume != "" {
		// Attempt to assume the specified role in the target account
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleToAssume)
		assumeRoleInput := &sts.AssumeRoleInput{
			RoleArn:         aws.String(roleArn),
			RoleSessionName: aws.String("AssumeRoleSession"),
		}

		// Include ExternalID if provided
		if cfg.ExternalID != "" {
			assumeRoleInput.ExternalId = aws.String(cfg.ExternalID)
		}

		// Attempt to assume the role
		assumeRoleOutput, err := stsClient.AssumeRole(context.TODO(), assumeRoleInput)
		if err != nil {
			result.Details.IsAccessible = false
			result.Details.HasPolicies = false
			result.Details.Error = fmt.Sprintf("Failed to assume role %s in account %s: %v", roleToAssume, accountID, err)
			result.Healthy = false
			return result
		}

		// Create new AWS Config with assumed role credentials
		credsProvider = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			*assumeRoleOutput.Credentials.AccessKeyId,
			*assumeRoleOutput.Credentials.SecretAccessKey,
			*assumeRoleOutput.Credentials.SessionToken,
		))

		awsCfg, err = GenerateAWSConfig(credsProvider)
		if err != nil {
			result.Details.IsAccessible = false
			result.Details.HasPolicies = false
			result.Details.Error = fmt.Sprintf("Failed to generate AWS config with assumed role: %v", err)
			result.Healthy = false
			return result
		}

		// Record the assumed role ARN
		result.Details.IsAccessible = true
		result.Details.IamPrincipal = *assumeRoleOutput.AssumedRoleUser.Arn
	} else {
		// No role assumption, use provided credentials directly

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
	}

	// Check policies attached to the principal
	attachedPolicies, missingPolicies, err := GetAttachedPolicies(awsCfg, result.Details.IamPrincipal, requiredPolicies)
	if err != nil {
		result.Details.HasPolicies = false
		result.Details.Error = fmt.Sprintf("Error checking policies: %v", err)
		result.Healthy = false
		return result
	}

	result.Details.AttachedPolicies = attachedPolicies
	if len(missingPolicies) > 0 {
		result.Details.HasPolicies = false
		result.Details.Error = fmt.Sprintf("Missing policies: %v", missingPolicies)
	} else {
		result.Details.HasPolicies = true
	}

	result.Healthy = result.Details.IsAccessible && result.Details.HasPolicies

	return result
}
