package discovery

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsarn "github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	organizationsTypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// DefaultMaxAccounts is the default maximum number of accounts to retrieve.
const DefaultMaxAccounts = 500

// Config represents the configuration loaded from the JSON file.
type Config struct {
	AWSAccessKeyID            string `json:"aws_access_key_id"`
	AWSSecretAccessKey        string `json:"aws_secret_access_key"`
	RoleToAssumeInMainAccount string `json:"role_to_assume_in_main_account,omitempty"`
	CrossAccountRole          string `json:"cross_account_role,omitempty"`
	ExternalID                string `json:"external_id,omitempty"`
	MaxAccounts               int    `json:"max_accounts,omitempty"`
}

// AccountResult holds the result for each account.
type AccountResult struct {
	AccountID   string  `json:"account_id"`
	AccountName string  `json:"account_name"`
	AccountType string  `json:"account_type"` // "Organization Master", "Organization Member", or "Standalone"
	Details     Details `json:"details"`
}

func AWSIntegrationDiscovery(cfg Config) ([]AccountResult, error) {
	credentialType := "Single Account"

	if cfg.CrossAccountRole != "" {
		credentialType = "Multi-Account"
	}

	if credentialType == "Multi-Account" {
		// Set MaxAccounts
		maxAccounts := cfg.MaxAccounts
		if maxAccounts == 0 {
			maxAccounts = DefaultMaxAccounts
		}

		// Perform organization discovery
		results, err := DiscoverOrganizationAccounts(cfg, maxAccounts)
		if err != nil {
			return nil, err
		}
		return results, nil
	} else {
		result, err := DiscoverSingleAccount(cfg)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return []AccountResult{*result}, nil
		} else {
			return []AccountResult{}, nil
		}
	}
}

// Details holds detailed information for each account.
type Details struct {
	Email            string   `json:"email,omitempty"`             // Email associated with the account
	IsAccessible     bool     `json:"isAccessible"`                // Indicates if the account is accessible
	HasPolicies      bool     `json:"hasPolicies"`                 // Indicates if required policies are attached
	AttachedPolicies []string `json:"attached_policies,omitempty"` // List of attached policy ARNs
	RequiredPolicies []string `json:"required_policies,omitempty"` // List of required policy ARNs
	IamPrincipal     string   `json:"iam_principal,omitempty"`     // ARN of the assumed role or IAM principal
	Healthy          bool     `json:"healthy"`                     // Overall health status
	CredentialType   string   `json:"credential_type"`             // "Single Account" or "Multi-Account"
	Error            string   `json:"error,omitempty"`             // Any errors encountered
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
		if len(resourceParts) < 3 {
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

// DiscoverOrganizationAccounts retrieves all active accounts in an AWS Organization
// and checks their accessibility and attached policies.
func DiscoverOrganizationAccounts(cfg Config, maxAccounts int) ([]AccountResult, error) {
	var results []AccountResult

	// Create initial AWS Config using provided credentials
	credsProvider := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		"",
	))

	awsCfg, err := GenerateAWSConfig(credsProvider)
	if err != nil {
		return nil, err
	}

	// Initialize STS client
	stsClient := sts.NewFromConfig(awsCfg)

	// Retrieve Caller Identity to get main account details
	identityOutput, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	mainAccountID := *identityOutput.Account
	mainAccountARN := *identityOutput.Arn

	// Set credential type as Multi-Account by default
	credentialType := "Multi-Account"

	// Initialize Organizations client
	orgClient := organizations.NewFromConfig(awsCfg)

	// Attempt to describe the organization
	// Discard orgOutput since it's not used
	_, err = orgClient.DescribeOrganization(context.TODO(), &organizations.DescribeOrganizationInput{})
	if err != nil {
		return nil, err
	}

	// If RoleToAssumeInMainAccount is provided, assume that role
	if cfg.RoleToAssumeInMainAccount != "" {
		// Construct Role ARN for main account
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", mainAccountID, cfg.RoleToAssumeInMainAccount)
		assumeRoleInput := &sts.AssumeRoleInput{
			RoleArn:         aws.String(roleArn),
			RoleSessionName: aws.String("AssumeMainAccountRoleSession"),
		}

		// Include ExternalID if provided
		if cfg.ExternalID != "" {
			assumeRoleInput.ExternalId = aws.String(cfg.ExternalID)
		}

		// Attempt to assume the role
		assumeRoleOutput, err := stsClient.AssumeRole(context.TODO(), assumeRoleInput)
		if err != nil {
			return nil, err
		}

		// Create new AWS Config with assumed role credentials
		credsProvider = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			*assumeRoleOutput.Credentials.AccessKeyId,
			*assumeRoleOutput.Credentials.SecretAccessKey,
			*assumeRoleOutput.Credentials.SessionToken,
		))

		awsCfg, err = GenerateAWSConfig(credsProvider)
		if err != nil {
			return nil, err
		}

		// Update STS client with new configuration
		stsClient = sts.NewFromConfig(awsCfg)
	}

	// Prepare to list accounts within the organization
	listAccountsInput := &organizations.ListAccountsInput{
		MaxResults: aws.Int32(20), // Default page size
	}

	var accounts []organizationsTypes.Account

	// Initialize paginator for listing accounts
	paginator := organizations.NewListAccountsPaginator(orgClient, listAccountsInput)

	// Iterate through pages of accounts
	for paginator.HasMorePages() && len(accounts) < maxAccounts {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			fmt.Printf("Error listing accounts: %v\n", err)
			break
		}

		// Append active accounts to the list
		for _, acct := range page.Accounts {
			if acct.Status != organizationsTypes.AccountStatusActive {
				continue
			}
			accounts = append(accounts, acct)
			if len(accounts) >= maxAccounts {
				break
			}
		}
	}

	// Iterate through each account to assess accessibility and policies
	for _, acct := range accounts {
		accountResult := AccountResult{
			AccountID:   *acct.Id,
			AccountName: *acct.Name,
			Details: Details{
				RequiredPolicies: requiredPolicies,
				Email:            *acct.Email,
				CredentialType:   credentialType,
			},
		}

		// Determine the account type (Master or Member)
		if *acct.Id == mainAccountID {
			accountResult.AccountType = "Organization Master"
		} else {
			accountResult.AccountType = "Organization Member"
		}

		// Handle the main account separately if no role assumption is needed
		if cfg.RoleToAssumeInMainAccount == "" && *acct.Id == mainAccountID {
			// Mark account as accessible
			accountResult.Details.IsAccessible = true
			accountResult.Details.IamPrincipal = mainAccountARN // The IAM user's ARN

			// Check policies attached to the IAM user
			attachedPolicies, missingPolicies, err := GetAttachedPolicies(awsCfg, mainAccountARN, requiredPolicies)
			if err != nil {
				accountResult.Details.HasPolicies = false
				accountResult.Details.Error = fmt.Sprintf("Error checking policies: %v", err)
				accountResult.Details.Healthy = false
				results = append(results, accountResult)
				continue
			}

			accountResult.Details.AttachedPolicies = attachedPolicies
			if len(missingPolicies) > 0 {
				accountResult.Details.HasPolicies = false
				accountResult.Details.Error = fmt.Sprintf("Missing policies: %v", missingPolicies)
			} else {
				accountResult.Details.HasPolicies = true
			}

			// Set overall health status
			accountResult.Details.Healthy = accountResult.Details.IsAccessible && accountResult.Details.HasPolicies

			results = append(results, accountResult)
			continue
		}

		// For member accounts, attempt to assume the cross-account role
		// Construct Role ARN for the member account
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", *acct.Id, cfg.CrossAccountRole)

		// Prepare AssumeRole input
		assumeRoleInput := &sts.AssumeRoleInput{
			RoleArn:         aws.String(roleArn),
			RoleSessionName: aws.String("DiscoverSession"),
		}

		// Include ExternalID if provided
		if cfg.ExternalID != "" {
			assumeRoleInput.ExternalId = aws.String(cfg.ExternalID)
		}

		// Attempt to assume the role
		assumeRoleOutput, err := stsClient.AssumeRole(context.TODO(), assumeRoleInput)
		if err != nil {
			// Cannot assume role; mark account as inaccessible
			accountResult.Details.IsAccessible = false
			accountResult.Details.HasPolicies = false
			accountResult.Details.Error = fmt.Sprintf("Failed to assume role: %v", err)
			// Healthy remains false by default
			results = append(results, accountResult)
			continue
		}

		// Mark account as accessible and record the assumed role ARN
		accountResult.Details.IsAccessible = true
		accountResult.Details.IamPrincipal = *assumeRoleOutput.AssumedRoleUser.Arn

		// Create AWS Config with assumed role credentials
		assumedCredsProvider := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			*assumeRoleOutput.Credentials.AccessKeyId,
			*assumeRoleOutput.Credentials.SecretAccessKey,
			*assumeRoleOutput.Credentials.SessionToken,
		))

		assumedCfg, err := GenerateAWSConfig(assumedCredsProvider)
		if err != nil {
			accountResult.Details.HasPolicies = false
			accountResult.Details.Error = fmt.Sprintf("Failed to generate AWS config with assumed role: %v", err)
			// Healthy remains false by default
			results = append(results, accountResult)
			continue
		}

		// Retrieve the ARN of the assumed role principal
		principalArn := *assumeRoleOutput.AssumedRoleUser.Arn

		// Check policies attached to the assumed role
		attachedPolicies, missingPolicies, err := GetAttachedPolicies(assumedCfg, principalArn, requiredPolicies)
		if err != nil {
			accountResult.Details.HasPolicies = false
			accountResult.Details.Error = fmt.Sprintf("Error checking policies: %v", err)
			// Healthy remains false by default
			results = append(results, accountResult)
			continue
		}

		accountResult.Details.AttachedPolicies = attachedPolicies
		if len(missingPolicies) > 0 {
			accountResult.Details.HasPolicies = false
			accountResult.Details.Error = fmt.Sprintf("Missing policies: %v", missingPolicies)
		} else {
			accountResult.Details.HasPolicies = true
		}

		// Set overall health status
		accountResult.Details.Healthy = accountResult.Details.IsAccessible && accountResult.Details.HasPolicies

		results = append(results, accountResult)
	}

	return results, nil
}

// DiscoverSingleAccount handles the case when credentials are only for a single account.
func DiscoverSingleAccount(cfg Config) (*AccountResult, error) {
	var result AccountResult

	// Create AWS Config
	credsProvider := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		"",
	))

	awsCfg, err := GenerateAWSConfig(credsProvider)
	if err != nil {
		return nil, err
	}

	// Get STS client
	stsClient := sts.NewFromConfig(awsCfg)

	// Get Caller Identity
	identityOutput, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	accountID := *identityOutput.Account
	accountARN := *identityOutput.Arn

	result.AccountID = accountID
	result.Details.RequiredPolicies = requiredPolicies
	result.Details.CredentialType = "Single Account"

	// Try to get account alias as account name
	iamClient := iam.NewFromConfig(awsCfg)
	aliasOutput, err := iamClient.ListAccountAliases(context.TODO(), &iam.ListAccountAliasesInput{})
	if err == nil && len(aliasOutput.AccountAliases) > 0 {
		result.AccountName = aliasOutput.AccountAliases[0]
	} else {
		// Fallback to using the account ID as the name
		result.AccountName = accountID
	}

	// Check if the account is part of an organization
	orgClient := organizations.NewFromConfig(awsCfg)
	if orgOutput, err := orgClient.DescribeOrganization(context.TODO(), &organizations.DescribeOrganizationInput{}); err == nil {
		// The account is part of an organization
		if orgOutput.Organization != nil && orgOutput.Organization.MasterAccountId != nil {
			if *orgOutput.Organization.MasterAccountId == accountID {
				result.AccountType = "Organization Master"
			} else {
				result.AccountType = "Organization Member"
			}
		}
	} else {
		if strings.Contains(err.Error(), "AWSOrganizationsNotInUseException") {
			result.AccountType = "Standalone"
		} else {
			result.AccountType = "Unknown"
			result.Details.Error = fmt.Sprintf("Error describing organization: %v", err)
		}
	}

	// Handle role assumption if role_to_assume_in_main_account is provided
	if cfg.RoleToAssumeInMainAccount != "" {
		// Assume the role in main account
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, cfg.RoleToAssumeInMainAccount)
		input := &sts.AssumeRoleInput{
			RoleArn:         aws.String(roleArn),
			RoleSessionName: aws.String("AssumeRoleSession"),
		}

		if cfg.ExternalID != "" {
			input.ExternalId = aws.String(cfg.ExternalID)
		}

		assumeRoleOutput, err := stsClient.AssumeRole(context.TODO(), input)
		if err != nil {
			result.Details.IsAccessible = false
			result.Details.HasPolicies = false
			result.Details.Error = fmt.Sprintf("Failed to assume role: %v", err)
			result.Details.Healthy = false
			return &result, nil
		}

		// Update AWS Config with assumed role credentials
		credsProvider = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			*assumeRoleOutput.Credentials.AccessKeyId,
			*assumeRoleOutput.Credentials.SecretAccessKey,
			*assumeRoleOutput.Credentials.SessionToken,
		))

		awsCfg, err = GenerateAWSConfig(credsProvider)
		if err != nil {
			result.Details.Error = fmt.Sprintf("Failed to generate AWS config with assumed role: %v", err)
			result.Details.IsAccessible = false
			result.Details.HasPolicies = false
			result.Details.Healthy = false
			return &result, nil
		}

		result.Details.IamPrincipal = *assumeRoleOutput.AssumedRoleUser.Arn
	} else {
		result.Details.IamPrincipal = accountARN
	}

	// Check policies attached to the IAM user or assumed role
	attachedPolicies, missingPolicies, err := GetAttachedPolicies(awsCfg, result.Details.IamPrincipal, requiredPolicies)
	if err != nil {
		return nil, err
		//result.Details.HasPolicies = false
		//result.Details.Error = fmt.Sprintf("Error checking policies: %v", err)
		//result.Details.IsAccessible = false
		//result.Details.Healthy = false
		//return &result, nil
	}

	result.Details.AttachedPolicies = attachedPolicies
	if len(missingPolicies) > 0 {
		result.Details.HasPolicies = false
		result.Details.Error = fmt.Sprintf("Missing policies: %v", missingPolicies)
	} else {
		result.Details.HasPolicies = true
	}

	result.Details.IsAccessible = true
	result.Details.Healthy = result.Details.IsAccessible && result.Details.HasPolicies

	return &result, nil
}
