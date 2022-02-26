package onboard

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/api"
)

func InitializeDb(db *Database) {
	db.orm.AutoMigrate(
		&Source{},
		&AWSMetadata{},
	)
}

type AWSMetadata struct {
	gorm.Model
	ID             uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	SourceID       string
	AccountID      string
	OrganizationID *string // null of not part of an aws organization
	Email          string
	Name           string
	SupportTier    string
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
	gorm.Model
	ID          uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	SourceId    string
	Name        string         `gorm:"not null"`
	Type        api.SourceType `gorm:"not null"`
	Description string
	ConfigRef   string
}

func NewAWSSource(in api.SourceAwsRequest) Source {
	id := uuid.New()
	provider := api.SourceCloudAWS

	s := Source{
		ID:          id,
		SourceId:    in.Config.AccountId,
		Name:        in.Name,
		Description: in.Description,
		Type:        provider,
		ConfigRef:   fmt.Sprintf("sources/%s/%s", strings.ToLower(string(provider)), id),
	}

	if len(strings.TrimSpace(s.Name)) == 0 {
		s.Name = s.SourceId
	}

	return s
}

func NewAzureSource(in api.SourceAzureRequest) Source {
	id := uuid.New()
	provider := api.SourceCloudAzure

	s := Source{
		ID:          id,
		SourceId:    in.Config.SubscriptionId,
		Name:        in.Name,
		Description: in.Description,
		Type:        provider,
		ConfigRef:   fmt.Sprintf("sources/%s/%s", strings.ToLower(string(provider)), id),
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
