package azure_subscription

import (
	"encoding/json"
	azureDescriberLocal "github.com/opengovern/og-describer-azure/provider/configs"
	"github.com/opengovern/opengovernance/services/integration-v2/integration-type/interfaces"
	"github.com/opengovern/opengovernance/services/integration-v2/models"
)

// AzureClientCertificateCredentials represents Azure SPN credentials using a certificate.
type AzureClientCertificateCredentials struct {
	azureDescriberLocal.AccountCredentials
}

func CreateAzureClientCertificateCredentials(jsonData []byte) (interfaces.CredentialType, error) {
	var credentials AzureClientCertificateCredentials
	err := json.Unmarshal(jsonData, &credentials)
	if err != nil {
		return nil, err
	}

	return &credentials, nil
}

func (c *AzureClientCertificateCredentials) HealthCheck() (bool, error) {
	//var password []byte
	//if c.CertificatePassword != "" {
	//	password = []byte(c.CertificatePassword)
	//}
	//
	//// Parse the certificate using azidentity.ParseCertificates
	//certs, key, err := azidentity.ParseCertificates([]byte(c.CertificateData), password)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to parse certificate: %w", err)
	//}
	//
	//// Create the ClientCertificateCredential
	//cred, err := azidentity.NewClientCertificateCredential(
	//	c.AzureTenantID,
	//	c.AzureClientID,
	//	certs,
	//	key,
	//	nil,
	//)
	//if err != nil {
	//	return false, fmt.Errorf("failed to create ClientCertificateCredential: %w", err)
	//}
	return true, nil
}

func (c *AzureClientCertificateCredentials) DiscoverIntegrations() ([]models.Integration, error) {
	//ctx := context.Background()
	//
	//pvkBlock, _ := pem.Decode([]byte(c.AzureSPNPrivateKey))
	//if pvkBlock == nil {
	//	return nil, errors.New("failed to decode PEM block containing the private key")
	//}
	//if pvkBlock.Type != "PRIVATE KEY" {
	//	return nil, fmt.Errorf("PEM block is not of type 'PRIVATE KEY'")
	//}
	//
	//// Parse the EC private key
	//privateKey, err := x509.ParsePKCS8PrivateKey(pvkBlock.Bytes)
	//if err != nil {
	//	return nil, err
	//}
	//
	//// Check if it's an RSA private key
	//rsaKey, ok := privateKey.(*rsa.PrivateKey)
	//if !ok {
	//	return nil, err
	//}
	//
	//// Decode the PEM-encoded certificate
	//block, _ := pem.Decode([]byte(c.AzureSPNCertificate))
	//if block == nil {
	//	return nil, errors.New("failed to decode PEM block containing the certificate")
	//}
	//
	//// Parse the certificate from the PEM block
	//cert, err := x509.ParseCertificate(block.Bytes)
	//if err != nil {
	//	return nil, err
	//}
	//
	//// Create the certificate chain
	//certs := []*x509.Certificate{cert}
	//
	//// Create credential with certificate
	//identity, err := azidentity.NewClientCertificateCredential(
	//	c.AzureTenantID,
	//	c.AzureClientID,
	//	certs,
	//	rsaKey,
	//	&azidentity.ClientCertificateCredentialOptions{},
	//)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to create ClientCertificateCredential: %v", err)
	//}
	//
	//client, err := armsubscription.NewSubscriptionsClient(identity, nil)
	//if err != nil {
	//	return nil, err
	//}
	//
	//it := client.NewListPager(nil)
	//subs := make([]model.AzureSubscription, 0)
	//for it.More() {
	//	page, err := it.NextPage(ctx)
	//	if err != nil {
	//		return nil, err
	//	}
	//	for _, v := range page.Value {
	//		if v == nil || v.State == nil {
	//			continue
	//		}
	//		tagsClient, err := armresources.NewTagsClient(*v.SubscriptionID, identity, nil)
	//		if err != nil {
	//			return nil, err
	//		}
	//		tagIt := tagsClient.NewListPager(nil)
	//		tagList := make([]armresources.TagDetails, 0)
	//		for tagIt.More() {
	//			tagPage, err := tagIt.NextPage(ctx)
	//			if err != nil {
	//				return nil, err
	//			}
	//			for _, tag := range tagPage.Value {
	//				tagList = append(tagList, *tag)
	//			}
	//		}
	//		localV := v
	//		subs = append(subs, model.AzureSubscription{
	//			SubscriptionID: *v.SubscriptionID,
	//			SubModel:       *localV,
	//			SubTags:        tagList,
	//		})
	//	}
	//}

	var integrations []models.Integration
	//for _, sub := range subs {
	//	var name string
	//	if sub.SubModel.DisplayName != nil {
	//		name = *sub.SubModel.DisplayName
	//	}
	//	integrations = append(integrations, models.Integration{
	//		IntegrationID:   uuid.New(),
	//		ProviderID:      sub.SubscriptionID,
	//		Name:            name,
	//		IntegrationType: IntegrationTypeAzureSubscription,
	//	})
	//}
	return integrations, nil
}
