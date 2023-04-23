package onboard

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/subscription/mgmt/subscription"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	authentication "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure"
	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

func discoverAzureSubscriptions(ctx context.Context, authConfig azure.AuthConfig) ([]subscription.Model, error) {
	authorizer, err := azure.NewAuthorizerFromConfig(authConfig)
	if err != nil {
		return nil, err
	}

	client := subscription.NewSubscriptionsClient()
	client.Authorizer = authorizer

	authorizer.WithAuthorization()

	it, err := client.ListComplete(ctx)
	if err != nil {
		return nil, err
	}
	//
	subs := make([]subscription.Model, 0) // don't convert to var so the returned list won't be nil
	for it.NotDone() {
		v := it.Value()
		if v.State != subscription.Enabled {
			continue
		}
		subs = append(subs, v)

		if it.NotDone() {
			err := it.NextWithContext(ctx)
			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	return subs, nil
}

func getAzureCredentialsMetadata(ctx context.Context, config api.SourceConfigAzure) (*source.AzureCredentialMetadata, error) {
	identity, err := azidentity.NewClientSecretCredential(
		config.TenantId,
		config.ClientId,
		config.ClientSecret,
		nil)
	if err != nil {
		return nil, err
	}

	tokenProvider, err := authentication.NewAzureIdentityAccessTokenProvider(identity)
	if err != nil {
		return nil, err
	}

	authProvider := absauth.NewBaseBearerTokenAuthenticationProvider(tokenProvider)
	requestAdaptor, err := msgraphsdk.NewGraphRequestAdapter(authProvider)
	if err != nil {
		return nil, err
	}

	graphClient := msgraphsdk.NewGraphServiceClient(requestAdaptor)
	result, err := graphClient.ApplicationsById(config.ObjectId).Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	metadata := source.AzureCredentialMetadata{
		SpnName:  *result.GetDisplayName(),
		ObjectId: *result.GetId(),
	}
	for _, passwd := range result.GetPasswordCredentials() {
		if passwd.GetKeyId() != nil && *passwd.GetKeyId() == config.SecretId {
			metadata.SecretId = config.SecretId
			metadata.SecretExpirationDate = *passwd.GetEndDateTime()
		}
	}

	if metadata.SecretId == "" {
		return nil, fmt.Errorf("failed to find the secret in application's credential list")
	}

	return &metadata, nil
}
