package onboard

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SourceType string

const (
	SourceCloudAWS   SourceType = "AWS"
	SourceCloudAzure SourceType = "Azure"
)

func InitializeDb(db *Database) {
	db.orm.AutoMigrate(
		&Organization{},
		&Source{},
		&AWSMetadata{},
	)
}

func IsValidSourceType(t SourceType) bool {
	switch t {
	case SourceCloudAWS, SourceCloudAzure:
		return true
	default:
		return false
	}
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

type AWSMetadata struct {
	gorm.Model
	ID                 uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	SourceID           string
	AccountID          string
	OrganizationID     *string // null of not part of an aws organization
	Email              string
	Name               string
	SupportTier        string
	AlternateContact   string
	SecurityHubEnabled bool
	MacieEnabled       bool
}

type Source struct {
	gorm.Model
	ID             uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID `gorm:"not null"`
	SourceId       string
	Name           string     `gorm:"not null"`
	Type           SourceType `gorm:"not null"`
	Description    string
	ConfigRef      string
	AWSMetadata    AWSMetadata `gorm:"foreignKey:SourceID"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Organization struct {
	gorm.Model
	ID          uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name        string    `gorm:"not null"`
	Description string    `gorm:"not null"`
	AdminEmail  string    `gorm:"not null"`
	KeibiUrl    string    `gorm:"not null"`
	VaultRef    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Sources     []Source `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE;"`
}

type SourceConfigAzure struct {
	SubscriptionId string
	TenantId       string
	ClientId       string
	ClientSecret   string
}

type SourceConfigAWS struct {
	Regions   []string
	AccessKey string
	SecretKey string
}
