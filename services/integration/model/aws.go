package model

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/kaytu-io/kaytu-util/pkg/fp"
)

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

	if org != nil {
		metadata.OrganizationID = org.Id
		metadata.OrganizationMasterAccountEmail = org.MasterAccountEmail
		metadata.OrganizationMasterAccountId = org.MasterAccountId
		metadata.OrganizationDiscoveredAccountCount = fp.Optional[int](len(accounts))
	}
	return &metadata, nil
}
