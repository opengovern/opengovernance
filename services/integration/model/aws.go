package model

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/kaytu-io/kaytu-util/pkg/fp"
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

type AWSCredentialConfig struct {
	AccountID           string   `json:"accountID"`
	AssumeRoleName      string   `json:"assumeRoleName"`
	HealthCheckPolicies []string `json:"healthCheckPolicies"`
	ExternalId          *string  `json:"externalId"`
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

func AWSCredentialConfigFromMap(cnf map[string]any) (*AWSCredentialConfig, error) {
	in, err := json.Marshal(cnf)
	if err != nil {
		return nil, err
	}

	var out AWSCredentialConfig
	if err := json.Unmarshal(in, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
