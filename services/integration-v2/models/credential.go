package models

import (
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/opengovern/opengovernance/services/integration-v2/api/models"
	"time"
)

type Credential struct {
	ID             uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	CredentialType string    `json:"credentialType"`
	Secret         string    `json:"-"`
	Metadata       pgtype.JSONB

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}

func (c *Credential) ToApi() (*models.Credential, error) {
	var metadata map[string]string
	if c.Metadata.Status == pgtype.Present {
		if err := json.Unmarshal(c.Metadata.Bytes, &metadata); err != nil {
			return nil, err
		}
	}

	return &models.Credential{
		ID:             c.ID.String(),
		CredentialType: c.CredentialType,
		Secret:         c.Secret,
		Metadata:       metadata,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}, nil
}
