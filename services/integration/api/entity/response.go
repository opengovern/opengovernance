package entity

import (
	"encoding/json"

	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/google/uuid"
)

const (
	FREESupportTier string = "FREE"
	PAIDSupportTier string = "PAID"
)

type AWSCredentialConfig struct {
	AccountId            string   `json:"accountId"`
	Regions              []string `json:"regions,omitempty"`
	AccessKey            string   `json:"accessKey" validate:"required"`
	SecretKey            string   `json:"secretKey" validate:"required"`
	AssumeRoleName       string   `json:"assumeRoleName,omitempty"`
	AssumeAdminRoleName  string   `json:"assumeAdminRoleName,omitempty"`
	AssumeRolePolicyName string   `json:"assumeRolePolicyName,omitempty"`
	ExternalId           *string  `json:"externalId,omitempty"`
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

type CreateAWSConnectionRequest struct {
	Name      string                `json:"name"`
	AWSConfig AWSCredentialV2Config `json:"awsConfig"`
}

type ConnectionAWSRequest struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Email       string               `json:"email"`
	Config      *AWSCredentialConfig `json:"config,omitempty"`
}

type AzureCredentialConfig struct {
	SubscriptionId string `json:"subscriptionId"`
	TenantId       string `json:"tenantId" validate:"required,uuid_rfc4122"`
	ObjectId       string `json:"objectId" validate:"required,uuid_rfc4122"`
	SecretId       string `json:"secretId" validate:"required,uuid_rfc4122"`
	ClientId       string `json:"clientId" validate:"required"`
	ClientSecret   string `json:"clientSecret" validate:"required"`
}

func (s AzureCredentialConfig) AsMap() map[string]any {
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

type CreateAzureConnectionRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Config      AzureCredentialConfig `json:"config"`
}

type AWSMetadataResponse struct {
	ID             string  `json:"uuid"`
	SourceID       string  `json:"source_id"`
	AccountID      string  `json:"account_id"`
	OrganizationID *string `json:"organization_id"`
	Email          string  `json:"email"`
	Name           string  `json:"name"`
	SupportTier    string  `json:"support_tier"`
}

type GetConnectionsRequest struct {
	SourceIDs []string `json:"source_ids"`
}

type DiscoverAWSAccountsRequest struct {
	AccessKey string `json:"accessKey" validate:"required"`
	SecretKey string `json:"secretKey" validate:"required"`
}

type DiscoverAWSAccountsResponse struct {
	AccountID      string `json:"accountId"`
	Status         string `json:"status"`
	OrganizationID string `json:"organizationId,omitempty"` // Nil if not part of an AWS organization
	Email          string `json:"email"`
	Name           string `json:"name"`
}

type DiscoverAzureSubscriptionsRequest struct {
	Config AzureCredentialConfig `json:"config"`
}

type DiscoverAzureSubscriptionsSPNRequest struct {
	SPNId uuid.UUID `json:"spnId"`
}

type DiscoverAzureSubscriptionsResponse struct {
	ID             string `json:"id"`
	SubscriptionID string `json:"subscriptionId"`
	Name           string `json:"name"`
}

type CreateCredentialV2Request struct {
	Connector source.Type            `json:"connector" example:"Azure"`
	AWSConfig *AWSCredentialV2Config `json:"awsConfig"`
}

type CreateCredentialV2Response struct {
	ID string `json:"id"`
}

func (req CreateCredentialV2Request) GetAWSConfig() (*AWSCredentialV2Config, error) {
	configStr, err := json.Marshal(req.AWSConfig)
	if err != nil {
		return nil, err
	}

	config := AWSCredentialV2Config{}
	err = json.Unmarshal(configStr, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

type AWSCredentialV2Config struct {
	AccountID           string   `json:"accountID"`
	AssumeRoleName      string   `json:"assumeRoleName"`
	HealthCheckPolicies []string `json:"healthCheckPolicies"`
	ExternalId          *string  `json:"externalId"`
}

func (s AWSCredentialV2Config) AsMap() map[string]any {
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

func AWSCredentialV2ConfigFromMap(cnf map[string]any) (*AWSCredentialV2Config, error) {
	in, err := json.Marshal(cnf)
	if err != nil {
		return nil, err
	}

	var out AWSCredentialV2Config
	if err := json.Unmarshal(in, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func CredentialTypeToAPI(c model.CredentialType) CredentialType {
	return CredentialType(c)
}
