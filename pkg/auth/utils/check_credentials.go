package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
)

// Input structure for JSON input
type Input struct {
	AWS   AWSInput   `json:"aws"`
	Azure AzureInput `json:"azure"`
}

type AWSInput struct {
	AccessKeyID     string   `json:"accessKeyID"`
	SecretAccessKey string   `json:"secretAccessKey"`
	SessionToken    string   `json:"sessionToken"`
	RoleName        string   `json:"roleName"`
	CheckTypes      []string `json:"checkTypes"`
}

type AzureInput struct {
	TenantID       string   `json:"tenantID"`
	ClientID       string   `json:"clientID"`
	ClientSecret   string   `json:"clientSecret"`
	SubscriptionID string   `json:"subscriptionID"`
	CheckTypes     []string `json:"checkTypes"`
}

// Result represents the output structure for AWS and Azure
type Result struct {
	Provider         string   `json:"provider"`
	CheckType        string   `json:"checkType"`
	Status           string   `json:"status"`
	Message          string   `json:"message"`
	IsOrganization   bool     `json:"isOrganization,omitempty"`
	IsMaster         bool     `json:"isMaster,omitempty"`
	AccessedAccounts []string `json:"accessedAccounts,omitempty"`
	AzureSPNValid    bool     `json:"azureSPNValid,omitempty"`
	HasRequiredRole  bool     `json:"hasRequiredRole,omitempty"`
	Subscriptions    []string `json:"subscriptions,omitempty"`
}

// ValidateAWSCredentials validates the provided AWS credentials
func ValidateAWSCredentials(accessKeyID, secretAccessKey, sessionToken string) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, sessionToken),
		)),
	)
	if err != nil {
		return cfg, fmt.Errorf("failed to load AWS configuration: %v", err)
	}

	stsClient := sts.NewFromConfig(cfg)
	_, err = stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return cfg, fmt.Errorf("invalid AWS credentials: %v", err)
	}

	return cfg, nil
}

// CheckAWSOrganization checks if the AWS account is part of an AWS Organization
func CheckAWSOrganization(cfg aws.Config) (bool, error) {
	orgClient := organizations.NewFromConfig(cfg)

	_, err := orgClient.DescribeOrganization(context.TODO(), &organizations.DescribeOrganizationInput{})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "AWSOrganizationsNotInUseException" {
			return false, nil
		}
		return false, fmt.Errorf("error checking organization status: %v", err)
	}

	return true, nil
}

// IsOrganizationMaster checks if the AWS account is the master account of the organization
func IsOrganizationMaster(cfg aws.Config) (bool, error) {
	stsClient := sts.NewFromConfig(cfg)

	output, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return false, fmt.Errorf("error retrieving caller identity: %v", err)
	}

	orgClient := organizations.NewFromConfig(cfg)
	orgDetails, err := orgClient.DescribeOrganization(context.TODO(), &organizations.DescribeOrganizationInput{})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "AccessDeniedException" {
			return false, fmt.Errorf("insufficient permissions to check organization master account: %v", err)
		}
		return false, fmt.Errorf("error retrieving organization details: %v", err)
	}

	if *orgDetails.Organization.MasterAccountId == *output.Account {
		return true, nil
	}

	return false, nil
}

// ValidateAzureSPN validates the Azure SPN credentials
func ValidateAzureSPN(tenantID, clientID, clientSecret string) (azcore.TokenCredential, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Azure: %v", err)
	}
	return cred, nil
}

// CheckAzureRole checks if the SPN has access to a specific Azure subscription
func CheckAzureRole(cred azcore.TokenCredential, subscriptionID, requiredRole string) (bool, error) {
	roleClient, err := armauthorization.NewRoleAssignmentsClient(subscriptionID, cred, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create role assignments client: %v", err)
	}

	// Example role assignment list logic
	pager := roleClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return false, fmt.Errorf("failed to list role assignments: %v", err)
		}
		for _, assignment := range page.Value {
			// Ensure RoleDefinitionID is not nil before dereferencing
			if assignment.Properties != nil && assignment.Properties.RoleDefinitionID != nil {
				if strings.Contains(*assignment.Properties.RoleDefinitionID, requiredRole) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// ListAzureSubscriptions lists subscriptions the SPN has access to
func ListAzureSubscriptions(cred azcore.TokenCredential) ([]string, error) {
	subClient, err := armsubscriptions.NewClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriptions client: %v", err)
	}

	pager := subClient.NewListPager(nil)
	subscriptionIDs := []string{}
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to list subscriptions: %v", err)
		}
		for _, sub := range page.Value {
			subscriptionIDs = append(subscriptionIDs, *sub.SubscriptionID)
		}
	}
	return subscriptionIDs, nil
}

// Function to read JSON input file
func readInputFile(filePath string) (Input, error) {
	var input Input
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return input, fmt.Errorf("failed to read input file: %v", err)
	}
	err = json.Unmarshal(data, &input)
	if err != nil {
		return input, fmt.Errorf("failed to parse JSON input: %v", err)
	}
	return input, nil
}

// Function to process AWS checks with default check types if not specified
func processAWSChecks(input AWSInput) ([]Result, error) {
	// Set default check types if not provided
	if len(input.CheckTypes) == 0 {
		input.CheckTypes = []string{"credentials", "organization", "master"}
	}

	var results []Result
	cfg, err := ValidateAWSCredentials(input.AccessKeyID, input.SecretAccessKey, input.SessionToken)
	if err != nil {
		results = append(results, Result{
			Provider:  "AWS",
			CheckType: "credentials",
			Status:    "error",
			Message:   fmt.Sprintf("Error validating AWS credentials: %v", err),
		})
		return results, nil
	}

	for _, checkType := range input.CheckTypes {
		switch checkType {
		case "credentials":
			results = append(results, Result{
				Provider:  "AWS",
				CheckType: "credentials",
				Status:    "success",
				Message:   "AWS credentials are valid.",
			})
		case "organization":
			isOrgAccount, err := CheckAWSOrganization(cfg)
			if err != nil {
				results = append(results, Result{
					Provider:  "AWS",
					CheckType: "organization",
					Status:    "error",
					Message:   fmt.Sprintf("Error checking AWS Organization status: %v", err),
				})
			} else {
				results = append(results, Result{
					Provider:       "AWS",
					CheckType:      "organization",
					Status:         "success",
					Message:        "AWS account organization status checked.",
					IsOrganization: isOrgAccount,
				})
			}
		case "master":
			isMaster, err := IsOrganizationMaster(cfg)
			if err != nil {
				results = append(results, Result{
					Provider:  "AWS",
					CheckType: "master",
					Status:    "error",
					Message:   fmt.Sprintf("Error checking if the AWS account is the master account: %v", err),
				})
			} else {
				results = append(results, Result{
					Provider:  "AWS",
					CheckType: "master",
					Status:    "success",
					Message:   "AWS account master status checked.",
					IsMaster:  isMaster,
				})
			}
		}
	}
	return results, nil
}

// Function to process Azure checks with default check types if not specified
func processAzureChecks(input AzureInput) ([]Result, error) {
	// Set default check types if not provided
	if len(input.CheckTypes) == 0 {
		input.CheckTypes = []string{"spn", "subscriptions"}
	}

	var results []Result

	cred, err := ValidateAzureSPN(input.TenantID, input.ClientID, input.ClientSecret)
	if err != nil {
		results = append(results, Result{
			Provider:      "Azure",
			CheckType:     "spn",
			Status:        "error",
			Message:       fmt.Sprintf("Azure SPN validation failed: %v", err),
			AzureSPNValid: false,
		})
		return results, nil
	}

	for _, checkType := range input.CheckTypes {
		switch checkType {
		case "spn":
			results = append(results, Result{
				Provider:      "Azure",
				CheckType:     "spn",
				Status:        "success",
				Message:       "Azure SPN credentials are valid.",
				AzureSPNValid: true,
			})
		case "role":
			if input.SubscriptionID == "" {
				results = append(results, Result{
					Provider:  "Azure",
					CheckType: "role",
					Status:    "error",
					Message:   "Subscription ID must be specified for role checks.",
				})
				continue
			}
			hasRole, err := CheckAzureRole(cred, input.SubscriptionID, "Reader")
			if err != nil {
				results = append(results, Result{
					Provider:  "Azure",
					CheckType: "role",
					Status:    "error",
					Message:   fmt.Sprintf("Error checking role for Azure subscription: %v", err),
				})
			} else {
				results = append(results, Result{
					Provider:        "Azure",
					CheckType:       "role",
					Status:          "success",
					Message:         "Checked Azure role successfully.",
					HasRequiredRole: hasRole,
				})
			}
		case "subscriptions":
			subscriptions, err := ListAzureSubscriptions(cred)
			if err != nil {
				results = append(results, Result{
					Provider:  "Azure",
					CheckType: "subscriptions",
					Status:    "error",
					Message:   fmt.Sprintf("Error listing Azure subscriptions: %v", err),
				})
			} else {
				results = append(results, Result{
					Provider:      "Azure",
					CheckType:     "subscriptions",
					Status:        "success",
					Message:       "Listed Azure subscriptions successfully.",
					Subscriptions: subscriptions,
				})
			}
		}
	}
	return results, nil
}

// Define a structure for grouped results
type GroupedResults struct {
	Provider     string   `json:"provider"`
	Summary      Summary  `json:"summary"`
	CheckDetails []Result `json:"checkDetails"`
}

// Define a structure for summary results
type Summary struct {
	TotalChecks      int `json:"totalChecks"`
	SuccessfulChecks int `json:"successfulChecks"`
	FailedChecks     int `json:"failedChecks"`
}

// Handler for processing the checks
func ProcessChecksHandler(input Input) ([]GroupedResults, error) {
	// Initialize grouped results for AWS and Azure
	var awsResults GroupedResults
	var azureResults GroupedResults

	// Initialize the check counters for summary
	var awsSummary Summary
	var azureSummary Summary

	// Check if AWS details are provided and process AWS checks
	if input.AWS.AccessKeyID != "" && input.AWS.SecretAccessKey != "" {
		awsChecks, err := processAWSChecks(input.AWS)
		if err != nil {
			return nil, fmt.Errorf("error processing AWS checks: %v", err)
		}

		// Populate the AWS result details and summary
		for _, check := range awsChecks {
			awsSummary.TotalChecks++
			if check.Status == "success" {
				awsSummary.SuccessfulChecks++
			} else {
				awsSummary.FailedChecks++
			}
			awsResults.CheckDetails = append(awsResults.CheckDetails, check)
		}

		awsResults.Provider = "AWS"
		awsResults.Summary = awsSummary
	}

	// Check if Azure details are provided and process Azure checks
	if input.Azure.TenantID != "" && input.Azure.ClientID != "" && input.Azure.ClientSecret != "" {
		azureChecks, err := processAzureChecks(input.Azure)
		if err != nil {
			return nil, fmt.Errorf("error processing Azure checks: %v", err)
		}

		// Populate the Azure result details and summary
		for _, check := range azureChecks {
			azureSummary.TotalChecks++
			if check.Status == "success" {
				azureSummary.SuccessfulChecks++
			} else {
				azureSummary.FailedChecks++
			}
			azureResults.CheckDetails = append(azureResults.CheckDetails, check)
		}

		azureResults.Provider = "Azure"
		azureResults.Summary = azureSummary
	}

	var finalOutput []GroupedResults
	if len(awsResults.CheckDetails) > 0 {
		finalOutput = append(finalOutput, awsResults)
	}
	if len(azureResults.CheckDetails) > 0 {
		finalOutput = append(finalOutput, azureResults)
	}

	return finalOutput, nil
}
