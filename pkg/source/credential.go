package source

import "time"

type Credential interface {
	GetExpirationDate() time.Time
}

type AWSCredentialMetadata struct {
	AccountID               string    `json:"account_id"`
	IamUserName             *string   `json:"iam_user_name"`
	IamApiKeyId             string    `json:"iam_api_key_id"`
	IamApiKeyExpirationDate time.Time `json:"iam_api_key_expiration_date"`
	AttachedPolicies        []string  `json:"attached_policies"`
}

func (m AWSCredentialMetadata) GetExpirationDate() time.Time {
	return m.IamApiKeyExpirationDate
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
