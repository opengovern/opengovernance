package source

type Credential interface {
	GetExpirationDate() int64
}

type AWSCredentialMetadata struct {
	AccountID               string   `json:"account_id"`
	IamUserName             *string  `json:"iam_user_name"`
	IamApiKeyId             string   `json:"iam_api_key_id"`
	IamApiKeyExpirationDate int64    `json:"iam_api_key_expiration_date"`
	AttachedPolicies        []string `json:"attached_policies"`
}

func (m AWSCredentialMetadata) GetExpirationDate() int64 {
	return m.IamApiKeyExpirationDate
}

type AzureCredentialMetadata struct {
	SpnName              string `json:"spn_name"`
	ObjectId             string `json:"object_id"`
	SecretId             string `json:"secret_id"`
	SecretExpirationDate int64  `json:"secret_expiration_date"`
}

func (m AzureCredentialMetadata) GetExpirationDate() int64 {
	return m.SecretExpirationDate
}
