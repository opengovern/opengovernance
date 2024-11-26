package models

import (
	"github.com/jackc/pgtype"
	"github.com/opengovern/opencomply/services/integration/api/models"
)

type IntegrationType struct {
	ID               int64  `gorm:"primaryKey"`
	Name             string `gorm:"unique;not null"`
	IntegrationType  string
	Label            string
	Tier             string
	Annotations      pgtype.JSONB
	Labels           pgtype.JSONB
	ShortDescription string
	Description      string
	Logo             string
	Enabled          bool
}

func (it *IntegrationType) ToApi() (*models.IntegrationType, error) {
	return &models.IntegrationType{
		ID:           it.ID,
		Name:         it.Name,
		PlatformName: it.IntegrationType,
		Label:        it.Label,
		Tier:         it.Tier,
		Logo:         it.Logo,
		Enabled:      it.Enabled,
	}, nil
}
