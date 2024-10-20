package models

import (
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	api "github.com/opengovern/opengovernance/services/integration-v2/api/models"
	"time"
)

type IntegrationLifecycle string

const (
	IntegrationLifecycleActive   IntegrationLifecycle = "ACTIVE"
	IntegrationLifecycleInactive IntegrationLifecycle = "INACTIVE"
	IntegrationLifecycleDisabled IntegrationLifecycle = "ARCHIVED"
)

type Integration struct {
	IntegrationTracker uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"` // Auto-generated UUID
	IntegrationID      string    `gorm:"index:idx_type_integrationid,unique"`
	IntegrationName    string
	Connector          string
	Type               string `gorm:"index:idx_type_integrationid,unique"`
	OnboardDate        time.Time
	Metadata           pgtype.JSONB
	Annotations        pgtype.JSONB
	Labels             pgtype.JSONB

	CredentialID uuid.UUID `gorm:"not null"` // Foreign key to Credential

	Credential Credential `gorm:"constraint:OnDelete:CASCADE;"` // Cascading delete when Integration is deleted

	Lifecycle IntegrationLifecycle
	Reason    string
	LastCheck *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}

func (i Integration) ToApi() (*api.Integration, error) {
	var metadata map[string]string
	if i.Metadata.Status == pgtype.Present {
		if err := json.Unmarshal(i.Metadata.Bytes, &metadata); err != nil {
			return nil, err
		}
	}

	var labels map[string]string
	if i.Labels.Status == pgtype.Present {
		if err := json.Unmarshal(i.Metadata.Bytes, &labels); err != nil {
			return nil, err
		}
	}

	var annotations map[string]string
	if i.Annotations.Status == pgtype.Present {
		if err := json.Unmarshal(i.Metadata.Bytes, &annotations); err != nil {
			return nil, err
		}
	}

	return &api.Integration{
		IntegrationTracker: i.IntegrationTracker.String(),
		IntegrationName:    i.IntegrationName,
		IntegrationID:      i.IntegrationID,
		IntegrationType:    i.Type,
		Connector:          i.Connector,
		OnboardDate:        i.OnboardDate,
		Lifecycle:          string(i.Lifecycle),
		Reason:             i.Reason,
		LastCheck:          i.LastCheck,
		CreatedAt:          i.CreatedAt,
		UpdatedAt:          i.UpdatedAt,
		Metadata:           metadata,
		Labels:             labels,
		Annotations:        annotations,
	}, nil
}
