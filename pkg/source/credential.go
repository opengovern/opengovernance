package source

import "time"

type AWSCredentialMetadata struct {
	AccountID             string    `json:"account_id"`
	IamUserName           *string   `json:"iam_user_name"`
	IamApiKeyCreationDate time.Time `json:"iam_api_key_creation_date"`
	AttachedPolicies      []string  `json:"attached_policies"`
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
