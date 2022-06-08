package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SourceType string

const (
	SourceCloudAWS   SourceType = "AWS"
	SourceCloudAzure SourceType = "Azure"
)

var SourceTypeValues = []SourceType{
	SourceCloudAWS,
	SourceCloudAzure,
}

func AsSourceType(t string) (SourceType, bool) {
	for _, st := range SourceTypeValues {
		if strings.EqualFold(string(st), t) {
			return st, true
		}
	}

	return "", false
}

type SourceAction string

const (
	SourceCreated SourceAction = "CREATE"
	SourceUpdated SourceAction = "UPDATE"
	SourceDeleted SourceAction = "DELETE"
)

const (
	FREESupportTier string = "FREE"
	PAIDSupportTier string = "PAID"
)

type SourceConfigAWS struct {
	AccountId string   `json:"accountId" validate:"required,len=12"`
	Regions   []string `json:"regions,omitempty"`
	AccessKey string   `json:"accessKey" validate:"required"`
	SecretKey string   `json:"secretKey" validate:"required"`
}

func (s SourceConfigAWS) AsMap() map[string]interface{} {
	in, err := json.Marshal(s)
	if err != nil {
		panic(err) // Don't expect any error
	}

	var out map[string]interface{}
	if err := json.Unmarshal(in, &out); err != nil {
		panic(err) // Don't expect any error
	}

	return out
}

type SourceAwsRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      SourceConfigAWS `json:"config"`
}

type SourceConfigAzure struct {
	SubscriptionId string `json:"subscriptionId" validate:"required,uuid_rfc4122"`
	TenantId       string `json:"tenantId" validate:"required,uuid_rfc4122"`
	ClientId       string `json:"clientId" validate:"required"`
	ClientSecret   string `json:"clientSecret" validate:"required"`
}

func (s SourceConfigAzure) AsMap() map[string]interface{} {
	in, err := json.Marshal(s)
	if err != nil {
		panic(err) // Don't expect any error
	}

	var out map[string]interface{}
	if err := json.Unmarshal(in, &out); err != nil {
		panic(err) // Don't expect any error
	}

	return out
}

type SourceAzureRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Config      SourceConfigAzure `json:"config"`
}

type SourceAzureSPNRequest struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	SubscriptionId string    `json:"subscriptionId" validate:"required,uuid_rfc4122"`
	SPNId          uuid.UUID `json:"spnId"`
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

type CreateSourceResponse struct {
	ID uuid.UUID `json:"id"`
}

type Source struct {
	ID          uuid.UUID  `json:"id"`
	SourceId    string     `json:"sourceId"`
	Name        string     `json:"name"`
	Type        SourceType `json:"type"`
	Description string     `json:"description"`
	OnboardDate time.Time  `json:"onboardDate"`
}

type GetSourcesResponse []Source

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
	TenantId     string `json:"tenantId" validate:"required,uuid_rfc4122"`
	ClientId     string `json:"clientId" validate:"required"`
	ClientSecret string `json:"clientSecret" validate:"required"`
}

type DiscoverAzureSubscriptionsResponse struct {
	ID             string `json:"id"`
	SubscriptionID string `json:"subscriptionId"`
	Name           string `json:"name"`
	Status         string `json:"status"`
}

type SourceEvent struct {
	Action     SourceAction
	SourceID   uuid.UUID
	AccountID  string
	SourceType SourceType
	ConfigRef  string
}

type AWSCredential struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type AzureCredential struct {
	ClientID     string `json:"clientID"`
	TenantID     string `json:"tenantID"`
	ClientSecret string `json:"clientSecret"`
}
