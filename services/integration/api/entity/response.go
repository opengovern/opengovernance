package entity

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
)

const (
	FREESupportTier string = "FREE"
	PAIDSupportTier string = "PAID"
)

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

type CreateAWSConnectionRequest struct {
	Config AWSCredentialConfig `json:"config,omitempty"`
}

type AzureCredentialConfig struct {
	TenantId     string `json:"tenantId" validate:"required,uuid_rfc4122"`
	ObjectId     string `json:"objectId" validate:"required,uuid_rfc4122"`
	ClientId     string `json:"clientId" validate:"required"`
	ClientSecret string `json:"clientSecret" validate:"required"`
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

func CredentialTypeToAPI(c model.CredentialType) CredentialType {
	return CredentialType(c)
}
