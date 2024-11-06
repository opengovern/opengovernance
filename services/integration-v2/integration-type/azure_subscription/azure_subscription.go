package azure_subscription

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/google/uuid"
	azure "github.com/opengovern/og-describer-azure/pkg/describer"
	azureDescriberLocal "github.com/opengovern/og-describer-azure/provider/configs"
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	IntegrationTypeAzureSubscription integration.Type = "AZURE_SUBSCRIPTION"
)

type AzureSubscriptionIntegration struct{}

func CreateAzureSubscriptionIntegration() (interfaces.IntegrationType, error) {
	return &AzureSubscriptionIntegration{}, nil
}

var CredentialTypes = map[string]interfaces.CredentialCreator{
	"client_secret":      CreateAzureClientSecretCredentials,
	"client_certificate": CreateAzureClientCertificateCredentials,
}

func (i *AzureSubscriptionIntegration) GetDescriberConfiguration() interfaces.DescriberConfiguration {
	return interfaces.DescriberConfiguration{
		NatsScheduledJobsTopic: azureDescriberLocal.JobQueueTopic,
		NatsManualJobsTopic:    azureDescriberLocal.JobQueueTopicManuals,
		NatsStreamName:         azureDescriberLocal.StreamName,
	}
}

func (i *AzureSubscriptionIntegration) GetAnnotations(credentialType string, jsonData []byte) (map[string]string, error) {
	annotations := make(map[string]string)

	return annotations, nil
}

func (i *AzureSubscriptionIntegration) GetLabels(credentialType string, jsonData []byte) (map[string]string, error) {
	annotations := make(map[string]string)

	return annotations, nil
}

func (i *AzureSubscriptionIntegration) HealthCheck(credentialType string, jsonData []byte, providerId string, labels map[string]string) (bool, error) {
	var configs azureDescriberLocal.AccountCredentials
	err := json.Unmarshal(jsonData, &configs)
	if err != nil {
		return false, err
	}

	ctx := context.Background()

	cred, err := getCredentials(credentialType, jsonData)
	if err != nil {
		return false, err
	}

	roleAssignmentsClient, err := armauthorization.NewRoleAssignmentsClient(providerId, cred, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create role assignments client: %w", err)
	}

	spnObjectID, err := getSPNObjectID(cred, configs.ClientID)

	scope := fmt.Sprintf("/subscriptions/%s", providerId)
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
			return true, nil
		}
	}

	return false, nil
}

func (i *AzureSubscriptionIntegration) DiscoverIntegrations(credentialType string, jsonData []byte) ([]models.Integration, error) {
	ctx := context.Background()

	cred, err := getCredentials(credentialType, jsonData)
	if err != nil {
		return nil, err
	}

	client, err := armsubscription.NewSubscriptionsClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriptions client: %w", err)
	}

	pager := client.NewListPager(nil)

	includeStates := []armsubscription.SubscriptionState{
		armsubscription.SubscriptionStateEnabled,
		armsubscription.SubscriptionStatePastDue,
		armsubscription.SubscriptionStateWarned,
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
	var integrations []models.Integration

	for pager.More() && len(integrations) < LIMIT {
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

					s := models.Integration{
						IntegrationID:   uuid.New(),
						ProviderID:      *sub.SubscriptionID,
						Name:            *sub.DisplayName,
						IntegrationType: IntegrationTypeAzureSubscription,
						//State: string(*sub.State),
					}
					integrations = append(integrations, s)
					if len(integrations) >= LIMIT {
						break
					}
				}
			}
		}
	}

	return integrations, nil
}

func (i *AzureSubscriptionIntegration) GetResourceTypesByLabels(map[string]string) ([]string, error) {
	return azure.ListResourceTypes(), nil
}

func getCredentials(credentialType string, jsonData []byte) (azcore.TokenCredential, error) {
	var configs azureDescriberLocal.AccountCredentials
	err := json.Unmarshal(jsonData, &configs)
	if err != nil {
		return nil, err
	}
	if credentialType == "client_secret" {
		cred, err := azidentity.NewClientSecretCredential(
			configs.TenantID,
			configs.ClientID,
			configs.ClientSecret,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create ClientSecretCredential: %w", err)
		}
		return cred, nil
	} else if credentialType == "client_certificate" {
		certData, err := ioutil.ReadFile(configs.CertificatePass)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate file: %w", err)
		}

		var password []byte
		if configs.CertificatePass != "" {
			password = []byte(configs.CertificatePass)
		}

		// Parse the certificate using azidentity.ParseCertificates
		certs, key, err := azidentity.ParseCertificates(certData, password)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}

		// Create the ClientCertificateCredential
		cred, err := azidentity.NewClientCertificateCredential(
			configs.TenantID,
			configs.ClientID,
			certs,
			key,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create ClientCertificateCredential: %w", err)
		}

		return cred, nil
	}

	return nil, fmt.Errorf("invalid credential type: %s", credentialType)
}

func (i *AzureSubscriptionIntegration) GetResourceTypeFromTableName(tableName string) string {
	return ""
}

func getSPNObjectID(cred azcore.TokenCredential, clientID string) (string, error) {
	ctx := context.Background()

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
