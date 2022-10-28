package onboard

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/google/uuid"

	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
)

func InitializeDb(db *Database) (err error) {
	err = db.orm.AutoMigrate(
		&Source{},
		&AWSMetadata{},
	)
	return
}

type AWSMetadata struct {
	ID             uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	SourceID       string
	AccountID      string
	OrganizationID *string // null of not part of an aws organization
	Email          string
	Name           string
	SupportTier    string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}

func (a AWSMetadata) toAWSMetadataResponse() *api.AWSMetadataResponse {
	return &api.AWSMetadataResponse{
		ID:             a.ID.String(),
		SourceID:       a.SourceID,
		AccountID:      a.AccountID,
		OrganizationID: a.OrganizationID,
		Email:          a.Email,
		Name:           a.Name,
		SupportTier:    a.SupportTier,
	}
}

type Source struct {
	ID          uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	SourceId    string    `gorm:"index:idx_source_id,unique"`
	Name        string    `gorm:"not null"`
	Email       string
	Type        source.Type `gorm:"not null"`
	Description string
	ConfigRef   string
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   sql.NullTime `gorm:"index"`
}

func NewAWSSource(in api.SourceAwsRequest) Source {
	id := uuid.New()
	provider := source.CloudAWS

	s := Source{
		ID:          id,
		SourceId:    in.Config.AccountId,
		Name:        in.Name,
		Email:       in.Email,
		Description: in.Description,
		Type:        provider,
		ConfigRef:   fmt.Sprintf("sources/%s/%s", strings.ToLower(string(provider)), id),
		Enabled:     true,
	}

	if len(strings.TrimSpace(s.Name)) == 0 {
		s.Name = s.SourceId
	}

	return s
}

func NewAzureSource(in api.SourceAzureRequest) Source {
	id := uuid.New()
	provider := source.CloudAzure

	// SPN -> PK: TenantID & ClientID
	s := Source{
		ID:          id,
		SourceId:    in.Config.SubscriptionId,
		Name:        in.Name,
		Description: in.Description,
		Type:        provider,
		ConfigRef:   fmt.Sprintf("sources/%s/%s", strings.ToLower(string(provider)), id),
		Enabled:     true,
	}

	if len(strings.TrimSpace(s.Name)) == 0 {
		s.Name = s.SourceId
	}

	return s
}

func (s Source) toSourceResponse() *api.CreateSourceResponse {
	return &api.CreateSourceResponse{
		ID: s.ID,
	}
}

func NewAzureSourceWithSPN(in api.SourceAzureSPNRequest) Source {
	id := uuid.New()
	provider := source.CloudAzure

	s := Source{
		ID:          id,
		SourceId:    in.SubscriptionId,
		Name:        in.Name,
		Description: in.Description,
		Type:        provider,
		Enabled:     true,
		ConfigRef:   fmt.Sprintf("sources/%s/spn/%s", strings.ToLower(string(provider)), in.SPNId),
	}

	if len(strings.TrimSpace(s.Name)) == 0 {
		s.Name = s.SourceId
	}

	return s
}

type SPN struct {
	ID        uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantId  string    `gorm:"index:idx_tenant_client_id,unique"`
	ClientId  string    `gorm:"index:idx_tenant_client_id,unique"`
	ConfigRef string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}

func (s SPN) toSPNResponse() *api.CreateSPNResponse {
	return &api.CreateSPNResponse{
		ID: s.ID,
	}
}

func NewSPN(in api.CreateSPNRequest) SPN {
	id := uuid.New()
	provider := source.CloudAzure

	return SPN{
		ID:        id,
		ClientId:  in.Config.ClientId,
		TenantId:  in.Config.TenantId,
		ConfigRef: fmt.Sprintf("sources/%s/spn/%s", strings.ToLower(string(provider)), id),
	}
}
