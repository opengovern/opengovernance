package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
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

type AzureCredentialMetadata struct {
	SpnName              string    `json:"spn_name"`
	ObjectId             string    `json:"object_id"`
	SecretId             string    `json:"secret_id"`
	SecretExpirationDate time.Time `json:"secret_expiration_date"`
}

func (m AzureCredentialMetadata) GetExpirationDate() time.Time {
	return m.SecretExpirationDate
}

func NewAzureCredential(name string, credentialType CredentialType, metadata *AzureCredentialMetadata) (*Credential, error) {
	id := uuid.New()
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	crd := &Credential{
		ID:             id,
		Name:           &name,
		ConnectorType:  source.CloudAzure,
		Secret:         fmt.Sprintf("sources/%s/%s", strings.ToLower(string(source.CloudAzure)), id),
		CredentialType: credentialType,
		Metadata:       jsonMetadata,
	}
	if credentialType == CredentialTypeManualAzureSpn {
		crd.AutoOnboardEnabled = true
	}

	return crd, nil
}
