package azure_subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	azureDescriberLocal "github.com/opengovern/og-describer-azure/provider/configs"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
	"golang.org/x/time/rate"
	"net/http"
	"strconv"
	"time"
)

const MAX_SUBSCRIPTIONS = 500

const READ_REQUESTS_PER_SECOND = 20

const LIMIT = 10

// AzureClientSecretCredentials represents Azure SPN credentials using a password.
type AzureClientSecretCredentials struct {
	azureDescriberLocal.AccountCredentials
}

func CreateAzureClientSecretCredentials(jsonData []byte) (interfaces.CredentialType, error) {
	var credentials AzureClientSecretCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, err
	}

	return &credentials, nil
}

func (c *AzureClientSecretCredentials) HealthCheck() (bool, error) {
	cred, err := azidentity.NewClientSecretCredential(c.TenantID, c.ClientID, c.ClientSecret, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create ClientSecretCredential: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	scopes := []string{"https://management.azure.com/.default"}

	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: scopes,
	})
	if err != nil {
		return false, fmt.Errorf("failed to acquire token: %v", err)
	}

	_, err = ExtractObjectID(token.Token)
	if err != nil {
		return false, fmt.Errorf("failed to extract object ID from token: %v", err)
	}

	return true, nil
}

func (c *AzureClientSecretCredentials) DiscoverIntegrations() ([]models.Integration, error) {
	ctx := context.Background()
	identity, err := azidentity.NewClientSecretCredential(
		c.TenantID,
		c.ClientID,
		c.ClientSecret,
		nil)
	if err != nil {
		return nil, err
	}
	client, err := armsubscription.NewSubscriptionsClient(identity, nil)
	if err != nil {
		return nil, err
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

// ExtractObjectID parses the token and extracts the object ID (oid claim).
func ExtractObjectID(tokenString string) (string, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if oid, ok := claims["oid"].(string); ok {
			return oid, nil
		}
		return "", fmt.Errorf("oid claim not found in token")
	}
	return "", fmt.Errorf("failed to parse claims")
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
			time.Sleep(retryAfter)
			return retries < maxRetries
		}
	}
	return false
}
