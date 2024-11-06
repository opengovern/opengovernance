package azure_subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	azureDescriberLocal "github.com/opengovern/og-describer-azure/provider/configs"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
	"github.com/opengovern/opengovernance/services/integration/model"
	"time"
)

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

	it := client.NewListPager(nil)
	subs := make([]model.AzureSubscription, 0)
	for it.More() {
		page, err := it.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, v := range page.Value {
			if v == nil || v.State == nil {
				continue
			}
			tagsClient, err := armresources.NewTagsClient(*v.SubscriptionID, identity, nil)
			if err != nil {
				return nil, err
			}
			tagIt := tagsClient.NewListPager(nil)
			tagList := make([]armresources.TagDetails, 0)
			for tagIt.More() {
				tagPage, err := tagIt.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, tag := range tagPage.Value {
					tagList = append(tagList, *tag)
				}
			}
			localV := v
			subs = append(subs, model.AzureSubscription{
				SubscriptionID: *v.SubscriptionID,
				SubModel:       *localV,
				SubTags:        tagList,
			})
		}
	}

	var integrations []models.Integration
	for _, sub := range subs {
		var name string
		if sub.SubModel.DisplayName != nil {
			name = *sub.SubModel.DisplayName
		}
		integrations = append(integrations, models.Integration{
			IntegrationID:   uuid.New(),
			ProviderID:      sub.SubscriptionID,
			Name:            name,
			IntegrationType: IntegrationTypeAzureSubscription,
		})
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
