package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	authentication "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// NewAzureCredential create a credential instance for azure SPN
func (h Connection) NewAzureCredential(
	ctx context.Context,
	name string,
	credType model.CredentialType,
	config entity.AzureCredentialConfig,
) (*model.Credential, error) {
	azureCnf, err := describe.AzureSubscriptionConfigFromMap(config.AsMap())
	if err != nil {
		return nil, err
	}

	metadata, err := h.AzureCredentialsMetadata(ctx, azureCnf)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential metadata: %w", err)
	}
	if credType == model.CredentialTypeManualAzureSpn {
		name = metadata.SpnName
	}

	cred, err := model.NewAzureCredential(name, credType, metadata)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

func (h Connection) AzureCurrentSubscription(
	ctx context.Context, subId string, authConfig azure.AuthConfig,
) (*model.AzureSubscription, error) {
	identity, err := azidentity.NewClientSecretCredential(
		authConfig.TenantID,
		authConfig.ClientID,
		authConfig.ClientSecret,
		nil)
	if err != nil {
		return nil, err
	}

	client, err := armsubscription.NewSubscriptionsClient(identity, nil)
	if err != nil {
		return nil, err
	}

	sub, err := client.Get(ctx, subId, nil)
	if err != nil {
		return nil, err
	}

	tagsClient, err := armresources.NewTagsClient(*sub.SubscriptionID, identity, nil)
	if err != nil {
		h.logger.Error("failed to create tags client", zap.Error(err))
		return nil, err
	}

	tagIt := tagsClient.NewListPager(nil)
	tagList := make([]armresources.TagDetails, 0)
	for tagIt.More() {
		tagPage, err := tagIt.NextPage(ctx)
		if err != nil {
			h.logger.Error("failed to get tag page", zap.Error(err))

			return nil, err
		}
		for _, tag := range tagPage.Value {
			tagList = append(tagList, *tag)
		}
	}

	return &model.AzureSubscription{
		SubscriptionID: subId,
		SubModel:       sub.Subscription,
		SubTags:        tagList,
	}, nil
}

func (h Connection) AzureCredentialsMetadata(ctx context.Context, config describe.AzureSubscriptionConfig) (*model.AzureCredentialMetadata, error) {
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

func (Connection) NewAzureConnectionMetadata(
	ctx context.Context,
	sub *model.AzureSubscription,
) model.AzureConnectionMetadata {
	metadata := model.AzureConnectionMetadata{
		SubscriptionID: sub.SubscriptionID,
		SubModel:       sub.SubModel,
		SubTags:        make(map[string][]string),
	}
	for _, tag := range sub.SubTags {
		if tag.TagName == nil || tag.Count == nil {
			continue
		}
		metadata.SubTags[*tag.TagName] = make([]string, 0, len(tag.Values))
		for _, value := range tag.Values {
			if value == nil || value.TagValue == nil {
				continue
			}
			metadata.SubTags[*tag.TagName] = append(metadata.SubTags[*tag.TagName], *value.TagValue)
		}
	}

	return metadata
}

// NewAzureConnectionWithCredentials builds a new connection with given credentials,
// also it encrypts the user configuration into it.
func (h Connection) NewAzureConnectionWithCredentials(
	ctx context.Context,
	sub *model.AzureSubscription,
	creationMethod source.SourceCreationMethod,
	description string,
	creds *model.Credential,
	reqConfig map[string]any,
) (model.Connection, error) {
	id := uuid.New()

	name := sub.SubscriptionID
	if sub.SubModel.DisplayName != nil {
		name = *sub.SubModel.DisplayName
	}

	metadata := h.NewAzureConnectionMetadata(ctx, sub)
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		jsonMetadata = []byte("{}")
	}

	s := model.Connection{
		ID:                   id,
		SourceId:             sub.SubscriptionID,
		Name:                 name,
		Description:          description,
		Type:                 source.CloudAzure,
		CredentialID:         creds.ID,
		Credential:           *creds,
		LifecycleState:       model.ConnectionLifecycleStateInProgress,
		AssetDiscoveryMethod: source.AssetDiscoveryMethodTypeScheduled,
		CreationMethod:       creationMethod,
		Metadata:             datatypes.JSON(jsonMetadata),
	}

	secretBytes, err := h.kms.Encrypt(reqConfig, h.keyARN)
	if err != nil {
		return s, err
	}
	s.Credential.Secret = string(secretBytes)

	return s, nil
}
