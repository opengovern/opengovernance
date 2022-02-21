package onboard

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type OrganizationRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=50"`
	Description string `json:"description" validate:"required,min=10,max=128"`
	AdminEmail  string `json:"adminEmail" validate:"required,email"`
	KeibiUrl    string `json:"keibiUrl" validate:"required,url"`
}

func (s OrganizationRequest) toOrganization() *Organization {
	return &Organization{
		ID:          uuid.New(),
		Name:        s.Name,
		Description: s.Description,
		AdminEmail:  s.AdminEmail,
		KeibiUrl:    s.KeibiUrl,
		CreatedAt:   time.Now().UTC(),
	}
}

type OrganizationResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	AdminEmail  string    `json:"adminEmail"`
	KeibiUrl    string    `json:"keibiUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (s Organization) toOrganizationResponse() *OrganizationResponse {
	return &OrganizationResponse{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		AdminEmail:  s.AdminEmail,
		KeibiUrl:    s.KeibiUrl,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

type SourceAwsRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	Config struct {
		AccountId string   `json:"accountId" validate:"required,len=12"`
		Regions   []string `json:"regions"` // array
		AccessKey string   `json:"accessKey" validate:"required"`
		SecretKey string   `json:"secretKey" validate:"required"`
	}
}

func (s SourceAwsRequest) toSource(orgId uuid.UUID) *Source {
	o := &Source{
		ID:             uuid.New(),
		SourceId:       s.Config.AccountId,
		OrganizationID: orgId,
		Name:           s.Name,
		Description:    s.Description,
		Type:           SourceCloudAWS,
		CreatedAt:      time.Now().UTC(),
	}

	if len(strings.TrimSpace(s.Name)) == 0 {
		o.Name = o.SourceId
	}

	return o
}

type SourceAzureRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	Config struct {
		SubscriptionId string `json:"subscriptionId" validate:"required,uuid_rfc4122"`
		TenantId       string `json:"tenantId" validate:"required,uuid_rfc4122"`
		ClientId       string `json:"clientId" validate:"required"`
		ClientSecret   string `json:"clientSecret" validate:"required"`
	}
}

func (s SourceAzureRequest) toSource(orgId uuid.UUID) *Source {
	o := &Source{
		ID:             uuid.New(),
		SourceId:       s.Config.SubscriptionId,
		OrganizationID: orgId,
		Name:           s.Name,
		Description:    s.Description,
		Type:           SourceCloudAzure,
		CreatedAt:      time.Now().UTC(),
	}

	if len(strings.TrimSpace(s.Name)) == 0 {
		o.Name = o.SourceId
	}

	return o
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

type SourceResponse struct {
	ID             uuid.UUID            `json:"id"`
	OrganizationID uuid.UUID            `json:"organizationId"`
	Name           string               `json:"name"`
	Description    string               `json:"description"`
	Type           string               `json:"type"`
	AWSMetadata    *AWSMetadataResponse `json:"aws_metadata"`
	CreatedAt      time.Time            `json:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
}

func (s Source) toSourceResponse() *SourceResponse {
	return &SourceResponse{
		ID:             s.ID,
		OrganizationID: s.OrganizationID,
		Name:           s.Name,
		Description:    s.Description,
		Type:           string(s.Type),
		AWSMetadata:    s.AWSMetadata.toAWSMetadataResponse(),
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

func (a AWSMetadata) toAWSMetadataResponse() *AWSMetadataResponse {
	return &AWSMetadataResponse{
		ID:             a.ID.String(),
		SourceID:       a.SourceID,
		AccountID:      a.AccountID,
		OrganizationID: a.OrganizationID,
		Email:          a.Email,
		Name:           a.Name,
		SupportTier:    a.SupportTier,
	}
}

type SourceEvent struct {
	Action     SourceAction
	SourceID   uuid.UUID
	SourceType SourceType
	ConfigRef  string
}
