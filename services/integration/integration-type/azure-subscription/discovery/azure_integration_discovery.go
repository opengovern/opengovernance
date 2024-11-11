package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/time/rate"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
)

const MAX_SUBSCRIPTIONS = 500

// Limit for read requests per second (80% of 25 as per the new throttling limits)
const READ_REQUESTS_PER_SECOND = 20

// Subscription struct represents a subscription and implements the IsHealthy method.
type Subscription struct {
	ID    string
	Name  string
	State string
}

// IsHealthy checks if the service principal has any role assignments in the subscription
func (s *Subscription) IsHealthy(ctx context.Context, cred azcore.TokenCredential, spnObjectID string) (bool, error) {
	// Create a RoleAssignmentsClient
	roleAssignmentsClient, err := armauthorization.NewRoleAssignmentsClient(s.ID, cred, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create role assignments client: %w", err)
	}

	// List role assignments for the SPN in the subscription
	scope := fmt.Sprintf("/subscriptions/%s", s.ID)
	pager := roleAssignmentsClient.NewListForScopePager(
		scope,
		&armauthorization.RoleAssignmentsClientListForScopeOptions{
			Filter: to.Ptr(fmt.Sprintf("principalId eq '%s'", spnObjectID)),
		},
	)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to get next page of role assignments: %w", err)
		}

		if len(page.Value) > 0 {
			// SPN has role assignments in the subscription
			return true, nil
		}
	}

	// No role assignments found for SPN in the subscription
	return false, nil
}

// getSubscriptionsWithLimit retrieves a limited number of subscriptions the SPN has access to.
// It returns the subscriptions in a slice.
func getSubscriptionsWithLimit(cred azcore.TokenCredential, limit int, includeStates ...armsubscription.SubscriptionState) ([]*Subscription, error) {
	// Create a SubscriptionsClient
	client, err := armsubscription.NewSubscriptionsClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriptions client: %w", err)
	}

	// Create a pager to handle pagination
	pager := client.NewListPager(nil)
	ctx := context.Background()
	var subscriptions []*Subscription

	// Default behavior: Exclude Deleted, Disabled, and Expired subscriptions
	if len(includeStates) == 0 {
		includeStates = []armsubscription.SubscriptionState{
			armsubscription.SubscriptionStateEnabled,
			armsubscription.SubscriptionStatePastDue,
			armsubscription.SubscriptionStateWarned,
		}
	}

	// Create a set of states to include for efficient lookup
	statesToInclude := make(map[armsubscription.SubscriptionState]bool)
	for _, state := range includeStates {
		statesToInclude[state] = true
	}

	// Create a rate limiter
	limiter := rate.NewLimiter(rate.Limit(READ_REQUESTS_PER_SECOND), READ_REQUESTS_PER_SECOND)

	// Define maximum retries
	const MAX_RETRIES = 5
	retries := 0

	for pager.More() && len(subscriptions) < limit {
		// Wait for permission to make the next request
		err := limiter.Wait(ctx)
		if err != nil {
			return nil, fmt.Errorf("rate limiter error: %w", err)
		}

		// Proceed to make the request
		page, err := pager.NextPage(ctx)
		if err != nil {
			// Handle throttling and retry logic
			if shouldRetry(err, retries, MAX_RETRIES) {
				retries++
				continue
			} else {
				return nil, fmt.Errorf("failed to get next page: %w", err)
			}
		}

		// Reset retries after a successful request
		retries = 0

		for _, sub := range page.Value {
			if sub.State != nil {
				state := *sub.State
				if statesToInclude[state] {
					// Safeguard against nil pointers
					if sub.SubscriptionID == nil || sub.DisplayName == nil || sub.State == nil {
						continue
					}

					s := &Subscription{
						ID:    *sub.SubscriptionID,
						Name:  *sub.DisplayName,
						State: string(*sub.State),
					}
					subscriptions = append(subscriptions, s)
					if len(subscriptions) >= limit {
						break
					}
				}
			}
		}
	}

	return subscriptions, nil
}

// shouldRetry handles throttling and retry logic
func shouldRetry(err error, retries, maxRetries int) bool {
	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		if responseError.StatusCode == http.StatusTooManyRequests {
			// Read Retry-After header
			retryAfter := time.Duration(1) * time.Second // Default retry after 1 second
			if responseError.RawResponse != nil {
				if h := responseError.RawResponse.Header.Get("Retry-After"); h != "" {
					// Retry-After can be in seconds or HTTP-date
					if seconds, parseErr := strconv.Atoi(h); parseErr == nil {
						retryAfter = time.Duration(seconds) * time.Second
					} else if date, parseErr := http.ParseTime(h); parseErr == nil {
						retryAfter = time.Until(date)
					}
				}
			}
			// Log the error details
			log.Printf("Throttled: %s. Retrying after %v (Attempt %d/%d)", responseError.Error(), retryAfter, retries+1, maxRetries)
			time.Sleep(retryAfter)
			return retries < maxRetries
		} else {
			// Handle other 429 errors or different status codes
			log.Printf("Received HTTP %d: %s", responseError.StatusCode, responseError.Error())
		}
	} else {
		// Not a ResponseError, return the error
		log.Printf("Failed to get next page: %v", err)
	}
	return false
}

// getSubscriptions retrieves the list of subscriptions the SPN has access to,
// supporting optional filters on subscription states.
// If no states are specified, it excludes Deleted, Disabled, and Expired subscriptions.
// It returns the subscriptions in a slice.
func getSubscriptions(cred azcore.TokenCredential, includeStates ...armsubscription.SubscriptionState) ([]*Subscription, error) {
	subscriptions, err := getSubscriptionsWithLimit(cred, MAX_SUBSCRIPTIONS, includeStates...)
	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}

// getSPNObjectID retrieves the object ID of the service principal using Microsoft Graph API
func getSPNObjectID(ctx context.Context, cred azcore.TokenCredential, clientID string) (string, error) {
	// Get a token for Microsoft Graph
	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://graph.microsoft.com/.default"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get token for Microsoft Graph: %w", err)
	}

	// Construct the URL
	baseURL, err := url.Parse("https://graph.microsoft.com/v1.0/servicePrincipals")
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Construct the filter query parameter
	filter := fmt.Sprintf("appId eq '%s'", clientID)
	// Encode the query parameters
	params := url.Values{}
	params.Add("$filter", filter)
	baseURL.RawQuery = params.Encode()

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set the Authorization header
	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("received non-200 response: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var result struct {
		Value []struct {
			ID string `json:"id"`
		} `json:"value"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Value) == 0 {
		return "", fmt.Errorf("service principal not found")
	}

	// Return the first matching service principal's ID
	return result.Value[0].ID, nil
}

// Config represents the JSON input configuration
type Config struct {
	ObjectID     string `json:"object_id,omitempty"`
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
	CertPath     string `json:"cert_path,omitempty"`
	CertContent  string `json:"cert_content,omitempty"` // New Field
	CertPassword string `json:"cert_password,omitempty"`
}

type SubscriptionOutput struct {
	SubscriptionID string `json:"subscriptionId"`
	DisplayName    string `json:"subscriptionDisplayName"`
	State          string `json:"subscriptionState"`
	IsHealthy      bool   `json:"isHealthy"`
}

func AzureIntegrationDiscovery(config Config) ([]SubscriptionOutput, error) {
	ctx := context.Background()

	var cred azcore.TokenCredential
	var err error

	if config.CertPath != "" || config.CertContent != "" {
		var certData []byte
		var err error

		if config.CertContent != "" {
			certData = []byte(config.CertContent)
		} else {
			certData, err = ioutil.ReadFile(config.CertPath)
			if err != nil {
				log.Fatalf("Failed to read certificate file: %v", err)
			}
		}

		var password []byte
		if config.CertPassword != "" {
			password = []byte(config.CertPassword)
		}

		// Parse the certificate using azidentity.ParseCertificates
		certs, key, err := azidentity.ParseCertificates(certData, password)
		if err != nil {
			log.Fatalf("Failed to parse certificate: %v", err)
		}

		// Create the ClientCertificateCredential
		cred, err = azidentity.NewClientCertificateCredential(
			config.TenantID,
			config.ClientID,
			certs,
			key,
			nil, // Additional options can be set here if needed
		)
		if err != nil {
			log.Fatalf("Failed to create certificate credential: %v", err)
		}
	} else if config.ClientSecret != "" {
		// Use client secret authentication
		cred, err = azidentity.NewClientSecretCredential(
			config.TenantID,
			config.ClientID,
			config.ClientSecret,
			nil, // Additional options can be set here if needed
		)
		if err != nil {
			log.Fatalf("Failed to create client secret credential: %v", err)
		}
	} else {
		return nil, fmt.Errorf("no valid authentication method found. Set AZURE_CLIENT_SECRET or AZURE_CLIENT_CERT_PATH.")
	}

	// The credential check logic has been removed.
	// Proceeding without verifying the credential's validity upfront.

	// Get the SPN object ID if not provided
	if config.ObjectID == "" {
		config.ObjectID, err = getSPNObjectID(ctx, cred, config.ClientID)
		if err != nil {
			log.Fatalf("Failed to get SPN object ID: %v", err)
		}
	}

	// Get subscriptions
	subscriptions, err := getSubscriptions(
		cred,
	)
	if err != nil {
		log.Fatalf("Failed to get subscriptions: %v", err)
	}

	var output []SubscriptionOutput

	// Check the health of each subscription
	for _, sub := range subscriptions {
		isHealthy, err := sub.IsHealthy(ctx, cred, config.ObjectID)
		if err != nil {
			log.Printf("Failed to check health for subscription %s: %v", sub.ID, err)
			isHealthy = false
		}

		output = append(output, SubscriptionOutput{
			SubscriptionID: sub.ID,
			DisplayName:    sub.Name,
			State:          sub.State,
			IsHealthy:      isHealthy,
		})
	}

	return output, nil
}
