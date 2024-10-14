package model

import (
	"context"
	"encoding/json"
	"github.com/opengovern/opengovernance/pkg/describe/connectors"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/opengovern/og-aws-describer/aws"
	"github.com/opengovern/og-util/pkg/fp"
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

type AWSCredentialMetadata struct {
	AccountID                          string    `json:"account_id"`
	AccountName                        *string   `json:"account_name"`
	IamUserName                        *string   `json:"iam_user_name"`
	IamApiKeyCreationDate              time.Time `json:"iam_api_key_creation_date"`
	AttachedPolicies                   []string  `json:"attached_policies"`
	OrganizationID                     *string   `json:"organization_id"`
	OrganizationMasterAccountEmail     *string   `json:"organization_master_account_email"`
	OrganizationMasterAccountId        *string   `json:"organization_master_account_id"`
	OrganizationDiscoveredAccountCount *int      `json:"organization_discovered_account_count"`
}

type AWSAccount struct {
	AccountID    string
	AccountName  *string
	Organization *types.Organization
	Account      *types.Account
}

func ExtractCredentialMetadata(accountID string, org *types.Organization, accounts []types.Account) (*AWSCredentialMetadata, error) {
	metadata := AWSCredentialMetadata{
		AccountID:             accountID,
		IamUserName:           nil,
		IamApiKeyCreationDate: time.Time{},
		AttachedPolicies:      nil,
	}
	for _, account := range accounts {
		if account.Id != nil && *account.Id == accountID {
			metadata.AccountName = account.Name
			break
		}
	}

	if org != nil {
		metadata.OrganizationID = org.Id
		metadata.OrganizationMasterAccountEmail = org.MasterAccountEmail
		metadata.OrganizationMasterAccountId = org.MasterAccountId
		metadata.OrganizationDiscoveredAccountCount = fp.Optional[int](len(accounts))
	}
	return &metadata, nil
}

type AWSCredentialConfig struct {
	AccountID      string  `json:"accountID"`
	AssumeRoleName string  `json:"assumeRoleName"`
	ExternalId     *string `json:"externalId,omitempty"`
	AccessKey      *string `json:"accessKey,omitempty"`
	SecretKey      *string `json:"secretKey,omitempty"`
}

func (s AWSCredentialConfig) AsMap() map[string]any {
	in, err := json.Marshal(s)
	if err != nil {
		panic(err) // Don't expect any error
	}

	var out map[string]any
	if err := json.Unmarshal(in, &out); err != nil {
		panic(err) // Don't expect any error
	}

	return out
}

func NewAWSConnectionMetadata(ctx context.Context, cfg connectors.AWSAccountConfig, connection Connection, account AWSAccount) (AWSConnectionMetadata, error) {
	metadata := AWSConnectionMetadata{
		AccountID: account.AccountID,
	}

	if connection.Credential.CredentialType == CredentialTypeAutoAws {
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
		sdkCnf, err := aws.GetConfig(ctx, cfg.AccessKey, cfg.SecretKey, "", "", nil)
		if err != nil {
			return metadata, err
		}
		organizationClient := organizations.NewFromConfig(sdkCnf)

		tags, err := organizationClient.ListTagsForResource(ctx, &organizations.ListTagsForResourceInput{
			ResourceId: &metadata.AccountID,
		})
		if err != nil {
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
