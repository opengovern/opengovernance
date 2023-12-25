package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-azure-describer/azure"
	kaytuAzure "github.com/kaytu-io/kaytu-azure-describer/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	authentication "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"go.uber.org/zap"
	"gorm.io/datatypes"
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

func (h Credential) NewAzureConnection(
	ctx context.Context,
	sub model.AzureSubscription,
	creationMethod source.SourceCreationMethod,
	description string,
	creds model.Credential,
) model.Connection {
	id := uuid.New()

	name := sub.SubscriptionID
	if sub.SubModel.DisplayName != nil {
		name = *sub.SubModel.DisplayName
	}

	metadata := model.NewAzureConnectionMetadata(&sub)
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
		Credential:           creds,
		LifecycleState:       model.ConnectionLifecycleStateInProgress,
		AssetDiscoveryMethod: source.AssetDiscoveryMethodTypeScheduled,
		CreationMethod:       creationMethod,
		Metadata:             datatypes.JSON(jsonMetadata),
	}

	return s
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

func (h Credential) AzureHealthCheck(ctx context.Context, cred *model.Credential) (bool, error) {
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

func (h Credential) AzureOnboard(ctx context.Context, credential model.Credential) ([]entity.Connection, error) {
	connections := make([]entity.Connection, 0)

	cnf, err := h.kms.Decrypt(credential.Secret, h.keyARN)
	if err != nil {
		return nil, err
	}

	azureCnf, err := describe.AzureSubscriptionConfigFromMap(cnf)
	if err != nil {
		return nil, err
	}

	h.logger.Info("discovering azure subscriptions", zap.String("credential-id", credential.ID.String()))

	subs, err := h.AzureDiscoverSubscriptions(ctx, kaytuAzure.AuthConfig{
		TenantID:     azureCnf.TenantID,
		ObjectID:     azureCnf.ObjectID,
		SecretID:     azureCnf.SecretID,
		ClientID:     azureCnf.ClientID,
		ClientSecret: azureCnf.ClientSecret,
	})
	if err != nil {
		h.logger.Error("failed to discover subscriptions", zap.Error(err))

		return nil, err
	}

	h.logger.Info("discovered azure subscriptions", zap.Int("count", len(subs)))

	existingConnections, err := h.connSvc.List(ctx, []source.Type{credential.ConnectorType})
	if err != nil {
		return nil, err
	}

	existingConnectionSubIDs := make([]string, 0, len(existingConnections))
	subsToOnboard := make([]model.AzureSubscription, 0)
	for _, conn := range existingConnections {
		existingConnectionSubIDs = append(existingConnectionSubIDs, conn.SourceId)
	}

	for _, sub := range subs {
		if sub.SubModel.State != nil && *sub.SubModel.State == armsubscription.SubscriptionStateEnabled && !utils.Includes(existingConnectionSubIDs, sub.SubscriptionID) {
			subsToOnboard = append(subsToOnboard, sub)
		} else {
			for _, conn := range existingConnections {
				if conn.SourceId == sub.SubscriptionID {
					name := sub.SubscriptionID
					if sub.SubModel.DisplayName != nil {
						name = *sub.SubModel.DisplayName
					}
					localConn := conn
					if conn.Name != name {
						localConn.Name = name
					}
					if sub.SubModel.State != nil && *sub.SubModel.State != armsubscription.SubscriptionStateEnabled {
						localConn.LifecycleState = model.ConnectionLifecycleStateDisabled
					}
					if conn.Name != name || localConn.LifecycleState != conn.LifecycleState {
						if err := h.connSvc.Update(ctx, localConn); err != nil {
							h.logger.Error("failed to update source", zap.Error(err))
							return nil, err
						}
					}
				}
			}
		}
	}

	h.logger.Info("onboarding subscriptions", zap.Int("count", len(subsToOnboard)))

	for _, sub := range subsToOnboard {
		h.logger.Info("onboarding subscription", zap.String("subscriptionId", sub.SubscriptionID))

		count, err := h.connSvc.Count(ctx, nil)
		if err != nil {
			return nil, err
		}
		if count >= maxConnections {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "maximum number of connections reached")
		}

		isAttached, err := kaytuAzure.CheckRole(kaytuAzure.AuthConfig{
			TenantID:     azureCnf.TenantID,
			ObjectID:     azureCnf.ObjectID,
			SecretID:     azureCnf.SecretID,
			ClientID:     azureCnf.ClientID,
			ClientSecret: azureCnf.ClientSecret,
		}, sub.SubscriptionID, kaytuAzure.DefaultReaderRoleDefinitionIDTemplate)
		if err != nil {
			h.logger.Warn("failed to check role", zap.Error(err))
			continue
		}
		if !isAttached {
			h.logger.Warn("role not attached", zap.String("subscriptionId", sub.SubscriptionID))
			continue
		}

		src := h.NewAzureConnection(
			ctx,
			sub,
			source.SourceCreationMethodAutoOnboard,
			fmt.Sprintf("Auto on-boarded subscription %s", sub.SubscriptionID),
			credential,
		)

		if err := h.connSvc.Create(ctx, src); err != nil {
			return nil, err
		}

		metadata := make(map[string]any)
		if src.Metadata.String() != "" {
			err := json.Unmarshal(src.Metadata, &metadata)
			if err != nil {
				return nil, err
			}
		}

		connections = append(connections, entity.Connection{
			ID:                   src.ID,
			ConnectionID:         src.SourceId,
			ConnectionName:       src.Name,
			Email:                src.Email,
			Connector:            src.Type,
			Description:          src.Description,
			OnboardDate:          src.CreatedAt,
			AssetDiscoveryMethod: src.AssetDiscoveryMethod,
			CredentialID:         src.CredentialID.String(),
			CredentialName:       src.Credential.Name,
			LifecycleState:       entity.ConnectionLifecycleState(src.LifecycleState),
			HealthState:          src.HealthState,
			LastHealthCheckTime:  src.LastHealthCheckTime,
			HealthReason:         src.HealthReason,
			Metadata:             metadata,
		})
	}

	return connections, nil
}

func (h Credential) AzureDiscoverSubscriptions(ctx context.Context, authConfig azure.AuthConfig) ([]model.AzureSubscription, error) {
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

	it := client.NewListPager(nil)
	subs := make([]model.AzureSubscription, 0)
	for it.More() {
		page, err := it.NextPage(ctx)
		if err != nil {
			h.logger.Error("failed to get subscription page", zap.Error(err))
			return nil, err
		}
		for _, v := range page.Value {
			if v == nil || v.State == nil {
				continue
			}
			tagsClient, err := armresources.NewTagsClient(*v.SubscriptionID, identity, nil)
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
			localV := v
			subs = append(subs, model.AzureSubscription{
				SubscriptionID: *v.SubscriptionID,
				SubModel:       *localV,
				SubTags:        tagList,
			})
		}
	}

	return subs, nil
}
