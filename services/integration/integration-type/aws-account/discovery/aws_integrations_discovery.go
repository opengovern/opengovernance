package discovery

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsarn "github.com/aws/aws-sdk-go-v2/aws/arn"
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
	AWSAccessKeyID                string `json:"aws_access_key_id"`
	AWSSecretAccessKey            string `json:"aws_secret_access_key"`
	RoleNameToAssumeInMainAccount string `json:"role_name_to_assume_in_main_account,omitempty"`
	CrossAccountRoleName          string `json:"cross_account_role_name,omitempty"`
	ExternalID                    string `json:"external_id,omitempty"`
	MaxAccounts                   int    `json:"max_accounts,omitempty"`
}

// AccountLabels holds the labels associated with each account.
type AccountLabels struct {
	AccountType           string `json:"account_type"` // "Organization Master", "Organization Member", or "Standalone"
	CrossAccountRoleARN   string `json:"cross_account_role_arn,omitempty"`
	RoleNameInMainAccount string `json:"role_name_in_main_account,omitempty"`
	ExternalID            string `json:"external_id,omitempty"`
}

// AccountResult holds the result for each account.
type AccountResult struct {
	AccountID   string        `json:"account_id"`
	AccountName string        `json:"account_name"`
	Labels      AccountLabels `json:"labels"`
	Details     Details       `json:"details"`
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

func AWSIntegrationDiscovery(cfg Config) []AccountResult {
	// Determine credential type
	credentialType := "Single Account"

	if cfg.CrossAccountRoleName != "" {
		credentialType = "Multi-Account"
	}

	if credentialType == "Multi-Account" {
		results := DiscoverOrganizationAccounts(cfg)
		return results
	} else {
		result := DiscoverSingleAccount(cfg)
		return []AccountResult{result}
	}
}

// GenerateAWSConfig creates an AWS configuration using the provided credentials.
// It can assume a role if roleNameToAssume is provided.
func GenerateAWSConfig(awsAccessKeyID string, awsSecretAccessKey string, roleNameToAssume string, externalID string, accountID string) (aws.Config, error) {
	// Step 1: Create base credentials provider
	credsProvider := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		awsAccessKeyID,
		awsSecretAccessKey,
		"",
	))

	// Step 2: Manually create the AWS Config struct with explicit credentials and region
	cfg := aws.Config{
		Region:      "us-east-2",
		Credentials: credsProvider,
	}

	// Step 3: If a role is specified to assume, perform the AssumeRole operation
	if roleNameToAssume != "" {
		// Construct Role ARN
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleNameToAssume)

		// Use STS client with the current config
		stsClient := sts.NewFromConfig(cfg)

		// Prepare AssumeRole input
		input := &sts.AssumeRoleInput{
			RoleArn:         aws.String(roleArn),
			RoleSessionName: aws.String("GenerateAWSConfigSession"),
		}
		if externalID != "" {
			input.ExternalId = aws.String(externalID)
		}

		// Perform AssumeRole
		assumeRoleOutput, err := stsClient.AssumeRole(context.TODO(), input)
		if err != nil {
			return aws.Config{}, fmt.Errorf("failed to assume role: %v", err)
		}

		// Update credentials provider with assumed role credentials
		credsProvider = aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			*assumeRoleOutput.Credentials.AccessKeyId,
			*assumeRoleOutput.Credentials.SecretAccessKey,
			*assumeRoleOutput.Credentials.SessionToken,
		))

		// Update the AWS Config with the new credentials
		cfg.Credentials = credsProvider
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
func DiscoverOrganizationAccounts(cfg Config) []AccountResult {
	var results []AccountResult

	// Step 1: Create base credentials provider
	credsProvider := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		"",
	))

	// Step 2: Manually create the AWS Config struct
	initialCfg := aws.Config{
		Region:      "us-east-2",
		Credentials: credsProvider,
	}

	// Step 3: Get primary account ID using initial configuration
	stsClient := sts.NewFromConfig(initialCfg)

	identityOutput, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		fmt.Printf("Failed to get caller identity: %v\n", err)
		return results
	}

	mainAccountID := *identityOutput.Account

	// Step 4: Generate AWS Config possibly with role assumption
	awsCfg, err := GenerateAWSConfig(
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		cfg.RoleNameToAssumeInMainAccount,
		cfg.ExternalID,
		mainAccountID,
	)
	if err != nil {
		fmt.Printf("Failed to generate AWS config: %v\n", err)
		return results
	}

	// Initialize Organizations client
	orgClient := organizations.NewFromConfig(awsCfg)

	// Attempt to describe the organization
	_, err = orgClient.DescribeOrganization(context.TODO(), &organizations.DescribeOrganizationInput{})
	if err != nil {
		fmt.Printf("This account is not an AWS Organizations management account or lacks permissions: %v\n", err)
		return results
	}

	// Prepare to list accounts within the organization
	listAccountsInput := &organizations.ListAccountsInput{
		MaxResults: aws.Int32(20), // Default page size
	}

	maxAccounts := cfg.MaxAccounts
	if maxAccounts == 0 {
		maxAccounts = DefaultMaxAccounts
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
			Labels: AccountLabels{
				ExternalID: cfg.ExternalID, // Include ExternalID if provided
			},
			Details: Details{
				RequiredPolicies: requiredPolicies,
				Email:            *acct.Email,
				CredentialType:   "Multi-Account",
			},
		}

		// Determine the account type (Master or Member)
		if *acct.Id == mainAccountID {
			accountResult.Labels.AccountType = "Organization Master"
		} else {
			accountResult.Labels.AccountType = "Organization Member"
		}

		// Handle the main account
		if *acct.Id == mainAccountID {
			// Mark account as accessible
			accountResult.Details.IsAccessible = true
			accountResult.Details.IamPrincipal = *identityOutput.Arn // The IAM user's ARN

			// Set RoleNameInMainAccount if role was assumed
			if cfg.RoleNameToAssumeInMainAccount != "" {
				accountResult.Labels.RoleNameInMainAccount = cfg.RoleNameToAssumeInMainAccount
			}

			// Check policies attached to the IAM user or assumed role
			attachedPolicies, missingPolicies, err := GetAttachedPolicies(awsCfg, accountResult.Details.IamPrincipal, requiredPolicies)
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
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", *acct.Id, cfg.CrossAccountRoleName)

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
		accountResult.Labels.CrossAccountRoleARN = roleArn

		// Manually create AWS Config with assumed role credentials
		assumedCfg := aws.Config{
			Region: "us-east-2",
			Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
				*assumeRoleOutput.Credentials.AccessKeyId,
				*assumeRoleOutput.Credentials.SecretAccessKey,
				*assumeRoleOutput.Credentials.SessionToken,
			)),
		}

		// Check policies attached to the assumed role
		attachedPolicies, missingPolicies, err := GetAttachedPolicies(assumedCfg, accountResult.Details.IamPrincipal, requiredPolicies)
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

	return results
}

// DiscoverSingleAccount handles the case when credentials are only for a single account.
func DiscoverSingleAccount(cfg Config) AccountResult {
	var result AccountResult

	// Step 1: Create base credentials provider
	credsProvider := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		"",
	))

	// Step 2: Manually create the AWS Config struct
	initialCfg := aws.Config{
		Region:      "us-east-2",
		Credentials: credsProvider,
	}

	// Step 3: Get primary account ID using initial configuration
	stsClient := sts.NewFromConfig(initialCfg)

	identityOutput, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		fmt.Printf("Failed to get caller identity: %v\n", err)
		result.Details.Error = fmt.Sprintf("Failed to get caller identity: %v", err)
		return result
	}

	accountID := *identityOutput.Account
	accountARN := *identityOutput.Arn

	// Step 4: Generate AWS Config possibly with role assumption
	awsCfg, err := GenerateAWSConfig(
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		cfg.RoleNameToAssumeInMainAccount,
		cfg.ExternalID,
		accountID,
	)
	if err != nil {
		fmt.Printf("Failed to generate AWS config: %v\n", err)
		result.Details.Error = fmt.Sprintf("Failed to generate AWS config: %v", err)
		return result
	}

	result.AccountID = accountID
	result.Details.RequiredPolicies = requiredPolicies
	result.Details.CredentialType = "Single Account"
	result.Labels.ExternalID = cfg.ExternalID // Include ExternalID if provided

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
				result.Labels.AccountType = "Organization Master"
			} else {
				result.Labels.AccountType = "Organization Member"
			}
		}
	} else {
		if strings.Contains(err.Error(), "AWSOrganizationsNotInUseException") {
			result.Labels.AccountType = "Standalone"
		} else {
			result.Labels.AccountType = "Unknown"
			result.Details.Error = fmt.Sprintf("Error describing organization: %v", err)
		}
	}

	// Handle role assumption if RoleNameToAssumeInMainAccount is provided
	if cfg.RoleNameToAssumeInMainAccount != "" {
		// Set RoleNameInMainAccount to the role used
		result.Labels.RoleNameInMainAccount = cfg.RoleNameToAssumeInMainAccount

		// Get the assumed role ARN
		stsClient = sts.NewFromConfig(awsCfg)
		identityOutput, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
		if err != nil {
			fmt.Printf("Failed to get caller identity: %v\n", err)
			result.Details.Error = fmt.Sprintf("Failed to get caller identity: %v", err)
			return result
		}

		result.Details.IamPrincipal = *identityOutput.Arn
	} else {
		result.Details.IamPrincipal = accountARN
	}

	// Check policies attached to the IAM user or assumed role
	attachedPolicies, missingPolicies, err := GetAttachedPolicies(awsCfg, result.Details.IamPrincipal, requiredPolicies)
	if err != nil {
		result.Details.HasPolicies = false
		result.Details.Error = fmt.Sprintf("Error checking policies: %v", err)
		result.Details.IsAccessible = false
		result.Details.Healthy = false
		return result
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

	return result
}
