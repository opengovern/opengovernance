package models

import (
	"encoding/json"
	"github.com/jackc/pgtype"
	"github.com/opengovern/opengovernance/services/integration/api/models"
)

type IntegrationType struct {
	ID               int64 `gorm:"primaryKey"`
	Name             string
	IntegrationType  string `gorm:"unique;not null"`
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
	var labels map[string]string
	if it.Labels.Status == pgtype.Present {
		if err := json.Unmarshal(it.Labels.Bytes, &labels); err != nil {
			return nil, err
		}
	}

	var annotations map[string]string
	if it.Annotations.Status == pgtype.Present {
		if err := json.Unmarshal(it.Annotations.Bytes, &annotations); err != nil {
			return nil, err
		}
	}

	return &models.IntegrationType{
		ID:               it.ID,
		Name:             it.Name,
		Label:            it.Label,
		Description:      it.Description,
		ShortDescription: it.ShortDescription,
		Tier:             it.Tier,
		Logo:             it.Logo,
		Enabled:          it.Enabled,
		Labels:           labels,
		Annotations:      annotations,
	}, nil
}
