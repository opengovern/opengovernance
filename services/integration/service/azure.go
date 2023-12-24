package service

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	authentication "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

// NewAzureCredential create a credential instance for azure SPN
func (h Credential) NewAzure(
	ctx context.Context,
	credType model.CredentialType,
	config entity.AzureCredentialConfig,
) (*model.Credential, error) {
	azureCnf, err := describe.AzureSubscriptionConfigFromMap(config.AsMap())
	if err != nil {
		return nil, err
	}

	metadata, err := h.AzureMetadata(ctx, azureCnf)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential metadata: %w", err)
	}

	cred, err := model.NewAzureCredential(credType, metadata)
	if err != nil {
		return nil, err
	}

	secretBytes, err := h.kms.Encrypt(config.AsMap(), h.keyARN)
	if err != nil {
		return nil, err
	}
	cred.Secret = string(secretBytes)

	return cred, nil
}

func (h Credential) AzureMetadata(ctx context.Context, config describe.AzureSubscriptionConfig) (*model.AzureCredentialMetadata, error) {
	identity, err := azidentity.NewClientSecretCredential(
		config.TenantID,
		config.ClientID,
		config.ClientSecret,
		nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	tokenProvider, err := authentication.NewAzureIdentityAccessTokenProvider(identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokenProvider: %w", err)
	}

	authProvider := absauth.NewBaseBearerTokenAuthenticationProvider(tokenProvider)
	requestAdaptor, err := msgraphsdk.NewGraphRequestAdapter(authProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create requestAdaptor: %w", err)
	}

	graphClient := msgraphsdk.NewGraphServiceClient(requestAdaptor)

	metadata := model.AzureCredentialMetadata{}
	if config.ObjectID == "" {
		return &metadata, nil
	}

	result, err := graphClient.ApplicationsById(config.ObjectID).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get Applications: %v", err)
	}

	metadata.SpnName = *result.GetDisplayName()
	metadata.ObjectId = *result.GetId()
	metadata.SecretId = config.SecretID
	for _, passwd := range result.GetPasswordCredentials() {
		if passwd.GetKeyId() != nil && *passwd.GetKeyId() == config.SecretID {
			metadata.SecretId = config.SecretID
			metadata.SecretExpirationDate = *passwd.GetEndDateTime()
		}
	}

	return &metadata, nil
}

func (h Credential) HealthCheck(ctx context.Context, cred *model.Credential) (bool, error) {
	config, err := h.kms.Decrypt(cred.Secret, h.keyARN)
	if err != nil {
		return false, err
	}

	var azureConfig describe.AzureSubscriptionConfig
	azureConfig, err = describe.AzureSubscriptionConfigFromMap(config)
	if err != nil {
		return false, err
	}

	if err := kaytuAzure.CheckSPNAccessPermission(kaytuAzure.AuthConfig{
		TenantID:            azureConfig.TenantID,
		ObjectID:            azureConfig.ObjectID,
		SecretID:            azureConfig.SecretID,
		ClientID:            azureConfig.ClientID,
		ClientSecret:        azureConfig.ClientSecret,
		CertificatePath:     azureConfig.CertificatePath,
		CertificatePassword: azureConfig.CertificatePass,
		Username:            azureConfig.Username,
		Password:            azureConfig.Password,
	}); err != nil {
		return false, err
	}

	return true, nil
}

func (h Credential) Create(ctx context.Context, cred *model.Credential) error {
	return h.repo.Create(ctx, cred)
}
