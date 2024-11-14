package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/opengovernance/services/integration/api/models"
	"time"
)

type Credential struct {
	ID              uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	IntegrationType integration.Type
	CredentialType  string
	Secret          string
	Metadata        pgtype.JSONB

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}

func (c *Credential) ToApi(returnSecret bool) (*models.Credential, error) {
	var metadata map[string]string
	if c.Metadata.Status == pgtype.Present {
		if err := json.Unmarshal(c.Metadata.Bytes, &metadata); err != nil {
			fmt.Println("could not unmarshal metadata", err)
		}
	}

	credential := &models.Credential{
		ID:              c.ID.String(),
		IntegrationType: c.IntegrationType,
		CredentialType:  c.CredentialType,
		Metadata:        metadata,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
	if returnSecret {
		credential.Secret = c.Secret
	}

	return credential, nil
}
