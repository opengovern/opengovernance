package onboard

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/subscription/mgmt/subscription"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	authentication "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

type azureSubscription struct {
	SubscriptionID string
	SubModel       subscription.Model
}

func discoverAzureSubscriptions(ctx context.Context, authConfig azure.AuthConfig) ([]azureSubscription, error) {
	authorizer, err := azure.NewAuthorizerFromConfig(authConfig)
	if err != nil {
		return nil, err
	}

	client := subscription.NewSubscriptionsClient()
	client.Authorizer = authorizer
	authorizer.WithAuthorization()

	it, err := client.List(ctx)
	if err != nil {
		return nil, err
	}
	subs := make([]azureSubscription, 0)
	for it.NotDone() {
		for _, v := range it.Values() {
			if v.State != subscription.Enabled {
				continue
			}
			subs = append(subs, azureSubscription{SubscriptionID: *v.SubscriptionID, SubModel: v})
		}
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

func currentAzureSubscription(ctx context.Context, subId string, authConfig azure.AuthConfig) (*azureSubscription, error) {
	authorizer, err := azure.NewAuthorizerFromConfig(authConfig)
	if err != nil {
		return nil, err
	}
	client := subscription.NewSubscriptionsClient()
	client.Authorizer = authorizer
	authorizer.WithAuthorization()

	sub, err := client.Get(ctx, subId)
	if err != nil {
		return nil, err
	}

	return &azureSubscription{
		SubscriptionID: subId,
		SubModel:       sub,
	}, nil
}

func getAzureCredentialsMetadata(ctx context.Context, config describe.AzureSubscriptionConfig) (*AzureCredentialMetadata, error) {
	identity, err := azidentity.NewClientSecretCredential(
		config.TenantID,
		config.ClientID,
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
	result, err := graphClient.ApplicationsById(config.ObjectID).Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	metadata := AzureCredentialMetadata{
		SpnName:  *result.GetDisplayName(),
		ObjectId: *result.GetId(),
	}
	for _, passwd := range result.GetPasswordCredentials() {
		if passwd.GetKeyId() != nil && *passwd.GetKeyId() == config.SecretID {
			metadata.SecretId = config.SecretID
			metadata.SecretExpirationDate = *passwd.GetEndDateTime()
		}
	}

	if metadata.SecretId == "" {
		return nil, fmt.Errorf("failed to find the secret in application's credential list")
	}

	return &metadata, nil
}
