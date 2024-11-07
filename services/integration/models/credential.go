package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/opengovern/opengovernance/services/integration/api/models"
	"time"
)

type Credential struct {
	ID             uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	CredentialType string
	Secret         string
	Metadata       pgtype.JSONB

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}

func (c *Credential) ToApi() (*models.Credential, error) {
	var metadata map[string]string
	if c.Metadata.Status == pgtype.Present {
		if err := json.Unmarshal(c.Metadata.Bytes, &metadata); err != nil {
			fmt.Println("could not unmarshal metadata", err)
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
