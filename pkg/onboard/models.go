package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/opengovern/opengovernance/pkg/describe/connectors"
	"github.com/opengovern/opengovernance/services/integration/model"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/google/uuid"
	"github.com/opengovern/og-util/pkg/source"
	"go.uber.org/zap"
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

func NewAWSConnectionMetadata(ctx context.Context, logger *zap.Logger, cfg connectors.AWSAccountConfig, connection any, account awsAccount) (AWSConnectionMetadata, error) {
	metadata := AWSConnectionMetadata{
		AccountID: account.AccountID,
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
		//sdkCnf, err := opengovernanceAws.GetConfig(ctx, cfg.AccessKey, cfg.SecretKey, "", "", nil)
		//if err != nil {
		//	logger.Error("failed to get aws config", zap.Error(err), zap.String("account_id", metadata.AccountID))
		//	return metadata, err
		//}
		//organizationClient := organizations.NewFromConfig(sdkCnf)

		//tags, err := organizationClient.ListTagsForResource(ctx, &organizations.ListTagsForResourceInput{
		//	ResourceId: &metadata.AccountID,
		//})
		//if err != nil {
		//	logger.Error("failed to get organization tags", zap.Error(err), zap.String("account_id", metadata.AccountID))
		//	return metadata, err
		//}
		//metadata.OrganizationTags = make(map[string]string)
		//for _, tag := range tags.Tags {
		//	if tag.Key == nil || tag.Value == nil {
		//		continue
		//	}
		//	metadata.OrganizationTags[*tag.Key] = *tag.Value
		//}
		//if account.Account == nil {
		//	orgAccount, err := organizationClient.DescribeAccount(ctx, &organizations.DescribeAccountInput{
		//		AccountId: &metadata.AccountID,
		//	})
		//	if err != nil {
		//		return metadata, err
		//	}
		//	metadata.OrganizationAccount = orgAccount.Account
		//}
	}

	return metadata, nil
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

func NewAzureCredential(name string, credentialType any, metadata any) (any, error) {
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
	if credentialType == model.CredentialTypeManualAzureSpn {
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
