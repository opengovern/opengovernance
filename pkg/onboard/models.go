package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/connectors"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/google/uuid"
	kaytuAws "github.com/kaytu-io/kaytu-aws-describer/aws"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type AWSAccountType string

const (
	AWSAccountTypeStandalone          AWSAccountType = "standalone"
	AWSAccountTypeOrganizationMember  AWSAccountType = "organization_member"
	AWSAccountTypeOrganizationManager AWSAccountType = "organization_manager"
)

type AWSConnectionMetadata struct {
	AccountID           string              `json:"account_id"`
	AccountName         string              `json:"account_name"`
	AccountType         AWSAccountType      `json:"account_type"`
	Organization        *types.Organization `json:"account_organization,omitempty"`
	OrganizationAccount *types.Account      `json:"organization_account,omitempty"`
	OrganizationTags    map[string]string   `json:"organization_tags,omitempty"`
}

func NewAWSConnectionMetadata(ctx context.Context, logger *zap.Logger, cfg connectors.AWSAccountConfig, connection model.Connection, account awsAccount) (AWSConnectionMetadata, error) {
	metadata := AWSConnectionMetadata{
		AccountID: account.AccountID,
	}

	if connection.Credential.CredentialType == model.CredentialTypeAutoAws {
		metadata.AccountType = AWSAccountTypeStandalone
	} else {
		metadata.AccountType = AWSAccountTypeOrganizationMember
	}

	if account.AccountName != nil {
		metadata.AccountName = *account.AccountName
	}
	metadata.Organization = account.Organization
	metadata.OrganizationAccount = account.Account
	if metadata.Organization != nil && metadata.Organization.MasterAccountId != nil &&
		*metadata.Organization.MasterAccountId == account.AccountID {
		metadata.AccountType = AWSAccountTypeOrganizationManager
	}
	if account.Organization != nil {
		sdkCnf, err := kaytuAws.GetConfig(ctx, cfg.AccessKey, cfg.SecretKey, "", "", nil)
		if err != nil {
			logger.Error("failed to get aws config", zap.Error(err), zap.String("account_id", metadata.AccountID))
			return metadata, err
		}
		organizationClient := organizations.NewFromConfig(sdkCnf)

		tags, err := organizationClient.ListTagsForResource(ctx, &organizations.ListTagsForResourceInput{
			ResourceId: &metadata.AccountID,
		})
		if err != nil {
			logger.Error("failed to get organization tags", zap.Error(err), zap.String("account_id", metadata.AccountID))
			return metadata, err
		}
		metadata.OrganizationTags = make(map[string]string)
		for _, tag := range tags.Tags {
			if tag.Key == nil || tag.Value == nil {
				continue
			}
			metadata.OrganizationTags[*tag.Key] = *tag.Value
		}
		if account.Account == nil {
			orgAccount, err := organizationClient.DescribeAccount(ctx, &organizations.DescribeAccountInput{
				AccountId: &metadata.AccountID,
			})
			if err != nil {
				return metadata, err
			}
			metadata.OrganizationAccount = orgAccount.Account
		}
	}

	return metadata, nil
}

func NewAWSSource(ctx context.Context, logger *zap.Logger, cfg connectors.AWSAccountConfig, account awsAccount, description string) model.Connection {
	id := uuid.New()
	provider := source.CloudAWS

	credName := fmt.Sprintf("%s - %s - default credentials", provider, account.AccountID)
	creds := model.Credential{
		ID:             uuid.New(),
		Name:           &credName,
		ConnectorType:  provider,
		Secret:         "",
		CredentialType: model.CredentialTypeAutoAws,
	}

	accountName := account.AccountID
	if account.AccountName != nil {
		accountName = *account.AccountName
	}
	accountEmail := ""
	if account.Account != nil && account.Account.Email != nil {
		accountEmail = *account.Account.Email
	}

	s := model.Connection{
		ID:                   id,
		SourceId:             account.AccountID,
		Name:                 accountName,
		Email:                accountEmail,
		Type:                 provider,
		Description:          description,
		CredentialID:         creds.ID,
		Credential:           creds,
		LifecycleState:       model.ConnectionLifecycleStateInProgress,
		AssetDiscoveryMethod: source.AssetDiscoveryMethodTypeScheduled,
		LastHealthCheckTime:  time.Now(),
		CreationMethod:       source.SourceCreationMethodManual,
	}

	if len(strings.TrimSpace(s.Name)) == 0 {
		s.Name = s.SourceId
	}

	metadata, err := NewAWSConnectionMetadata(ctx, logger, cfg, s, account)
	if err != nil {
		// TODO: log error
	}

	marshalMetadata, err := json.Marshal(metadata)
	if err != nil {
		marshalMetadata = []byte("{}")
	}
	s.Metadata = marshalMetadata

	return s
}

type AzureConnectionMetadata struct {
	TenantID       string                       `json:"tenant_id"`
	SubscriptionID string                       `json:"subscription_id"`
	SubModel       armsubscription.Subscription `json:"subscription_model"`
	SubTags        map[string][]string          `json:"subscription_tags"`
}

func NewAzureConnectionMetadata(sub azureSubscription, tenantID string) AzureConnectionMetadata {
	metadata := AzureConnectionMetadata{
		TenantID:       tenantID,
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

func NewAzureConnectionWithCredentials(sub azureSubscription, creationMethod source.SourceCreationMethod, description string, creds model.Credential, tenantID string) model.Connection {
	id := uuid.New()

	name := sub.SubscriptionID
	if sub.SubModel.DisplayName != nil {
		name = *sub.SubModel.DisplayName
	}

	metadata := NewAzureConnectionMetadata(sub, tenantID)
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

func NewAWSAutoOnboardedConnection(ctx context.Context, logger *zap.Logger, cfg connectors.AWSAccountConfig, account awsAccount, creationMethod source.SourceCreationMethod, description string, creds model.Credential) model.Connection {
	id := uuid.New()

	name := account.AccountID
	if account.AccountName != nil {
		name = *account.AccountName
	}

	lifecycleState := model.ConnectionLifecycleStateDiscovered
	if creds.AutoOnboardEnabled {
		lifecycleState = model.ConnectionLifecycleStateInProgress
	}

	if account.Account.Status != types.AccountStatusActive {
		lifecycleState = model.ConnectionLifecycleStateArchived
	}

	s := model.Connection{
		ID:                   id,
		SourceId:             account.AccountID,
		Name:                 name,
		Description:          description,
		Type:                 source.CloudAWS,
		CredentialID:         creds.ID,
		Credential:           creds,
		LifecycleState:       lifecycleState,
		AssetDiscoveryMethod: source.AssetDiscoveryMethodTypeScheduled,
		LastHealthCheckTime:  time.Now(),
		CreationMethod:       creationMethod,
	}

	metadata, err := NewAWSConnectionMetadata(ctx, logger, cfg, s, account)
	if err != nil {
		// TODO: log error
	}
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		jsonMetadata = []byte("{}")
	}
	s.Metadata = jsonMetadata

	return s
}

func NewAWSAutoOnboardedConnectionV2(ctx context.Context, org *types.Organization, logger *zap.Logger, account types.Account, creationMethod source.SourceCreationMethod, description string, creds model.Credential, awsConfig aws.Config) (*model.Connection, error) {
	id := uuid.New()

	name := *account.Id
	if account.Name != nil {
		name = *account.Name
	}

	lifecycleState := model.ConnectionLifecycleStateDiscovered
	if creds.AutoOnboardEnabled {
		lifecycleState = model.ConnectionLifecycleStateInProgress
	}

	if account.Status != types.AccountStatusActive {
		lifecycleState = model.ConnectionLifecycleStateArchived
	}

	s := model.Connection{
		ID:                   id,
		SourceId:             *account.Id,
		Name:                 name,
		Description:          description,
		Type:                 source.CloudAWS,
		CredentialID:         creds.ID,
		Credential:           creds,
		LifecycleState:       lifecycleState,
		AssetDiscoveryMethod: source.AssetDiscoveryMethodTypeScheduled,
		LastHealthCheckTime:  time.Now(),
		CreationMethod:       creationMethod,
	}
	metadata := AWSConnectionMetadata{
		AccountID:           *account.Id,
		AccountName:         name,
		Organization:        nil,
		OrganizationAccount: &account,
		OrganizationTags:    nil,
	}
	if creds.CredentialType == model.CredentialTypeAutoAws {
		metadata.AccountType = AWSAccountTypeStandalone
	} else {
		metadata.AccountType = AWSAccountTypeOrganizationMember
	}

	metadata.Organization = org
	if org != nil {
		if org.MasterAccountId != nil &&
			*metadata.Organization.MasterAccountId == *account.Id {
			metadata.AccountType = AWSAccountTypeOrganizationManager
		}

		organizationClient := organizations.NewFromConfig(awsConfig)
		tags, err := organizationClient.ListTagsForResource(ctx, &organizations.ListTagsForResourceInput{
			ResourceId: &metadata.AccountID,
		})
		if err != nil {
			logger.Error("failed to get organization tags", zap.Error(err), zap.String("account_id", metadata.AccountID))
			return nil, err
		}
		metadata.OrganizationTags = make(map[string]string)
		for _, tag := range tags.Tags {
			if tag.Key == nil || tag.Value == nil {
				continue
			}
			metadata.OrganizationTags[*tag.Key] = *tag.Value
		}
	}

	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	s.Metadata = jsonMetadata
	return &s, nil
}

//
//func (s Source) ToSourceResponse() *api.CreateSourceResponse {
//	return &api.CreateSourceResponse{
//		ID: s.ID,
//	}
//}

func NewAzureCredential(name string, credentialType model.CredentialType, metadata *AzureCredentialMetadata) (*model.Credential, error) {
	id := uuid.New()
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	crd := &model.Credential{
		ID:             id,
		Name:           &name,
		ConnectorType:  source.CloudAzure,
		Secret:         fmt.Sprintf("sources/%s/%s", strings.ToLower(string(source.CloudAzure)), id),
		CredentialType: credentialType,
		Metadata:       jsonMetadata,
	}
	if credentialType == model.CredentialTypeManualAzureSpn || credentialType == model.CredentialTypeManualAzureEntraId {
		crd.AutoOnboardEnabled = true
	}

	return crd, nil
}

func NewAWSCredential(name string, metadata *AWSCredentialMetadata, credentialType model.CredentialType, version int) (*model.Credential, error) {
	id := uuid.New()
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	crd := &model.Credential{
		ID:             id,
		Name:           &name,
		ConnectorType:  source.CloudAWS,
		Secret:         fmt.Sprintf("sources/%s/%s", strings.ToLower(string(source.CloudAWS)), id),
		CredentialType: credentialType,
		Metadata:       jsonMetadata,
		Version:        version,
	}
	if credentialType == model.CredentialTypeManualAwsOrganization {
		crd.AutoOnboardEnabled = true
	}

	return crd, nil
}

type AWSCredentialMetadata struct {
	AccountID                          string    `json:"account_id"`
	IamUserName                        *string   `json:"iam_user_name"`
	IamApiKeyCreationDate              time.Time `json:"iam_api_key_creation_date"`
	AttachedPolicies                   []string  `json:"attached_policies"`
	OrganizationID                     *string   `json:"organization_id"`
	OrganizationMasterAccountEmail     *string   `json:"organization_master_account_email"`
	OrganizationMasterAccountId        *string   `json:"organization_master_account_id"`
	OrganizationDiscoveredAccountCount *int      `json:"organization_discovered_account_count"`
}

type AzureCredentialMetadata struct {
	DefaultDomain        *string   `json:"default_domain"`
	SpnName              string    `json:"spn_name"`
	ObjectId             string    `json:"object_id"`
	SecretId             string    `json:"secret_id"`
	SecretExpirationDate time.Time `json:"secret_expiration_date"`
}

func (m AzureCredentialMetadata) GetExpirationDate() time.Time {
	return m.SecretExpirationDate
}
