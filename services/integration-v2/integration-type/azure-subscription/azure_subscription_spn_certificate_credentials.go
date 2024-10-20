package azure_subscription

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/google/uuid"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
	"github.com/opengovern/opengovernance/services/integration/model"
	"time"
)

// AzureSPNCertificateCredentials represents Azure SPN credentials using a certificate.
type AzureSPNCertificateCredentials struct {
	AzureClientID                  string  `json:"azure_client_id" binding:"required"`
	AzureTenantID                  string  `json:"azure_tenant_id" binding:"required"`
	AzureSPNCertificate            string  `json:"azure_spn_certificate" binding:"required"`
	AzureClientCertificatePassword *string `json:"azure_client_certificate_password,omitempty"`
	AzureSPNObjectID               *string `json:"azure_spn_object_id,omitempty"`
}

func CreateAzureSPNCertificateCredentials(jsonData []byte) (interfaces.CredentialType, map[string]any, error) {
	var credentials AzureSPNCertificateCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, nil, err
	}

	return &credentials, credentials.ConvertToMap(), nil
}

func (c *AzureSPNCertificateCredentials) HealthCheck() error {
	// Decode the PEM-encoded certificate
	block, _ := pem.Decode([]byte(c.AzureSPNCertificate))
	if block == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}

	// Parse the certificate from the PEM block
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Create the certificate chain
	certs := []*x509.Certificate{cert}

	// Parse the private key (if provided)
	var privateKey crypto.PrivateKey
	if c.AzureClientCertificatePassword != nil {
		privateKey, err = tls.X509KeyPair([]byte(c.AzureSPNCertificate), []byte(*c.AzureClientCertificatePassword))
	} else {
		privateKey, err = tls.X509KeyPair([]byte(c.AzureSPNCertificate), nil)
	}
	if err != nil {
		return fmt.Errorf("failed to parse certificate or key: %v", err)
	}

	// Create credential with certificate
	cred, err := azidentity.NewClientCertificateCredential(
		c.AzureTenantID,
		c.AzureClientID,
		certs,
		privateKey,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create ClientCertificateCredential: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	scopes := []string{"https://management.azure.com/.default"}
	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: scopes,
	})
	if err != nil {
		return fmt.Errorf("failed to acquire token: %v", err)
	}

	_, err = ExtractObjectID(token.Token)
	if err != nil {
		return fmt.Errorf("failed to extract object ID from token: %v", err)
	}

	return nil
}

func (c *AzureSPNCertificateCredentials) DiscoverIntegrations() ([]models.Integration, error) {
	ctx := context.Background()

	// Decode the PEM-encoded certificate
	block, _ := pem.Decode([]byte(c.AzureSPNCertificate))
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	// Parse the certificate from the PEM block
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Create the certificate chain
	certs := []*x509.Certificate{cert}

	// Parse the private key (if provided)
	var privateKey crypto.PrivateKey
	if c.AzureClientCertificatePassword != nil {
		privateKey, err = tls.X509KeyPair([]byte(c.AzureSPNCertificate), []byte(*c.AzureClientCertificatePassword))
	} else {
		privateKey, err = tls.X509KeyPair([]byte(c.AzureSPNCertificate), nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate or key: %v", err)
	}

	// Create credential with certificate
	identity, err := azidentity.NewClientCertificateCredential(
		c.AzureTenantID,
		c.AzureClientID,
		certs,
		privateKey,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ClientCertificateCredential: %v", err)
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
			IntegrationTracker: uuid.New(),
			IntegrationID:      sub.SubscriptionID,
			IntegrationName:    name,
			Connector:          "Azure",
			Type:               "azure_subscription",
			OnboardDate:        time.Now(),
		})
	}
	return integrations, nil
}

func (c *AzureSPNCertificateCredentials) ConvertToMap() map[string]any {
	result := map[string]any{
		"azure_client_id":       c.AzureClientID,
		"azure_tenant_id":       c.AzureTenantID,
		"azure_spn_certificate": c.AzureSPNCertificate,
	}

	// Add optional fields if they are not nil
	if c.AzureClientCertificatePassword != nil {
		result["azure_client_certificate_password"] = *c.AzureClientCertificatePassword
	}
	if c.AzureSPNObjectID != nil {
		result["azure_spn_object_id"] = *c.AzureSPNObjectID
	}

	return result
}
